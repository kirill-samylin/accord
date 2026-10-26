package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// SetupOptions drive `worker setup`. Only ConfigPath, In and Out are public;
// the remaining seams exist for tests and default to the real implementations.
type SetupOptions struct {
	ConfigPath string
	In         io.Reader
	Out        io.Writer

	probe   *setupProbe
	service *workerService
	// connect enrolls a new host or re-sends consent for an enrolled one.
	connect func(ctx context.Context, config Config, out io.Writer) error
	// preflight checks the chosen CLIs before anything is written.
	preflight  func(Config) error
	executable string
}

// Setup walks the owner through connecting this computer: discovers CLIs,
// runs the dialogue, validates and writes worker.json atomically, then
// stops the background service, enrolls or re-consents, and starts it again.
func Setup(ctx context.Context, o SetupOptions) error {
	if o.ConfigPath == "" || o.In == nil || o.Out == nil {
		return errors.New("worker setup requires --config PRIVATE_JSON and an interactive terminal")
	}
	if err := o.defaults(); err != nil {
		return err
	}
	p := newSetupPrompter(o.In, o.Out)
	current, existing, err := readSetupConfig(o.ConfigPath)
	if err != nil {
		return err
	}
	p.say("Подключение компьютера к Accord. Секреты не запрашиваются и не печатаются; токен остаётся в своём файле.")
	p.say("Ищу установленные CLI (codex, claude, gennadiy)…")
	config, err := runSetupDialogue(p, *o.probe, o.ConfigPath, current, existing)
	if err != nil {
		return err
	}
	if err = config.Validate(); err != nil {
		return fmt.Errorf("configuration rejected, nothing written: %w", err)
	}
	p.say("\nПроверяю выбранные CLI (--version/--help)…")
	if err = o.preflight(config); err != nil {
		return fmt.Errorf("native CLI preflight failed, nothing written: %w", err)
	}
	if err = writeSetupConfig(o.ConfigPath, config); err != nil {
		return err
	}
	p.say("Конфигурация записана: %s (права 0600).", o.ConfigPath)
	return o.connectWithService(ctx, p, config)
}

func (o *SetupOptions) defaults() error {
	if o.probe == nil {
		probe := defaultSetupProbe()
		o.probe = &probe
	}
	if o.preflight == nil {
		o.preflight = probeProviders
	}
	if o.connect == nil {
		o.connect = connectHost
	}
	if o.executable == "" {
		self, err := os.Executable()
		if err != nil {
			return err
		}
		if o.executable, err = filepath.EvalSymlinks(self); err != nil {
			return err
		}
	}
	if o.service == nil {
		home, _ := os.UserHomeDir()
		service, err := findWorkerService(home)
		if err != nil {
			return err
		}
		o.service = service
	}
	return nil
}

// connectWithService stops the LaunchAgent (it holds the state lock),
// optionally refreshes its binary, enrolls or re-consents, and restarts it.
// A connect failure never leaves the service stopped.
func (o *SetupOptions) connectWithService(ctx context.Context, p *setupPrompter, config Config) error {
	stopped := false
	if o.service != nil {
		p.say("\nНайдена фоновая служба %s.", o.service.label)
		ok, err := p.confirm("Остановить её на время подключения и затем запустить снова?", true)
		if err != nil {
			return err
		}
		if !ok {
			p.say("Служба не остановлена; enroll/consent невозможны, пока она держит состояние. Остановите её и выполните `worker consent` вручную.")
			return nil
		}
		if err = o.service.stop(ctx); err != nil {
			p.say("Не удалось остановить службу (%v); продолжаю — возможно, она не была запущена.", err)
		}
		stopped = true
		if !o.service.sameExecutable(o.executable) {
			ok, err = p.confirm("Обновить бинарь службы "+o.service.binary+" копией текущего agent-board ("+o.executable+")?", false)
			if err != nil {
				return o.restart(ctx, p, stopped, err)
			}
			if ok {
				if err = replaceExecutable(o.executable, o.service.binary); err != nil {
					return o.restart(ctx, p, stopped, err)
				}
				p.say("Бинарь службы обновлён.")
			}
		}
	}
	err := o.connect(ctx, config, p.out)
	if err != nil {
		p.say("Подключение не завершено: %v", err)
		p.say("Конфигурация сохранена; после устранения причины выполните `agent-board worker consent --config %s` (или enroll для нового компьютера).", o.ConfigPath)
	}
	return o.restart(ctx, p, stopped, err)
}

func (o *SetupOptions) restart(ctx context.Context, p *setupPrompter, stopped bool, err error) error {
	if stopped {
		if startErr := o.service.start(ctx); startErr != nil {
			p.say("Служба не запустилась: %v. Запустите её вручную: launchctl bootstrap %s %s", startErr, o.service.domain, o.service.plist)
			if err == nil {
				err = startErr
			}
		} else {
			p.say("Служба %s запущена снова.", o.service.label)
		}
	} else if err == nil {
		p.say("Фоновой службы нет: запустите `agent-board worker run --config %s` и оставьте его работать.", o.ConfigPath)
	}
	return err
}

// connectHost opens the private state and either enrolls (first time) or
// re-sends local consent, which also re-pins the new executables/folders.
func connectHost(ctx context.Context, config Config, out io.Writer) error {
	engine, err := openWhenReleased(ctx, config)
	if err != nil {
		return err
	}
	defer engine.Close()
	if engine.Enrolled() {
		host, err := engine.Consent(ctx)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, "Local host consent updated:", host.ID, "— providers:", len(config.Executables), "— controllers:", len(host.Controllers))
		return err
	}
	host, err := engine.Enroll(ctx)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, "Local host enrolled:", host.ID, "— owner controller ID:", host.OwnerPrincipalID)
	return err
}

// openWhenReleased waits briefly for a just-stopped service to release the
// private state lock (launchctl bootout returns before the worker exits).
func openWhenReleased(ctx context.Context, config Config) (*Engine, error) {
	deadline := time.Now().Add(30 * time.Second)
	for {
		engine, err := Open(config)
		if err == nil || err.Error() != "worker state is already in use" || time.Now().After(deadline) {
			return engine, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// Enrolled reports whether this private state already belongs to a host.
func (e *Engine) Enrolled() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.journal.HostID != ""
}
