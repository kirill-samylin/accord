package worker

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// connectionHints are non-secret defaults read from the owner's local
// connection.json beside the token file (the GUI client writes it). They
// are only proposed as defaults; the owner confirms every value.
type connectionHints struct {
	Origin    string `json:"origin"`
	ProjectID string `json:"project_id"`
	Machine   string `json:"machine"`
}

func readConnectionHints(tokenFile string) connectionHints {
	var hints connectionHints
	file, err := os.Open(filepath.Join(filepath.Dir(tokenFile), "connection.json"))
	if err != nil {
		return hints
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 65536))
	if err != nil || json.Unmarshal(data, &hints) != nil {
		return connectionHints{}
	}
	return hints
}

// defaultHostName proposes a worker name from the hostname, reduced to the
// identifier alphabet Config.Validate accepts.
func defaultHostName() string {
	host, _ := os.Hostname()
	host, _, _ = strings.Cut(host, ".")
	host = sanitizeIdentifier(host)
	if host == "" {
		return "pc"
	}
	return host
}

func sanitizeIdentifier(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			if b.Len() > 0 {
				b.WriteRune(r)
			}
		case r == ' ':
			if b.Len() > 0 {
				b.WriteRune('-')
			}
		}
	}
	out := strings.TrimRight(b.String(), "-_.")
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}

// askConnection fills server/project/token_file/state_dir/name/machine for a
// configuration that does not exist yet. An existing file keeps these fields
// untouched (see runSetupDialogue).
func (d *setupDialogue) askConnection(config *Config) error {
	d.p.say("\nШаг 1. Подключение к Accord. Нужны ваш личный токен участника (файл),")
	d.p.say("HTTPS-адрес сервера и ID проекта из вашего connection.json.")
	for {
		token, err := d.p.ask("Путь к приватному файлу токена", config.TokenFile)
		if err != nil {
			return err
		}
		if !filepath.IsAbs(token) {
			d.p.say("Нужен абсолютный путь.")
			continue
		}
		if _, err = ReadToken(token); err != nil {
			d.p.say("Файл токена не принят: %v. Файл должен быть обычным, с правами 0600.", err)
			continue
		}
		config.TokenFile = token
		break
	}
	hints := readConnectionHints(config.TokenFile)
	for {
		server, err := d.p.ask("HTTPS-адрес сервера Accord", firstNonEmpty(config.Server, hints.Origin))
		if err != nil {
			return err
		}
		if !strings.HasPrefix(server, "https://") && !strings.HasPrefix(server, "http://127.0.0.1") && !strings.HasPrefix(server, "http://localhost") {
			d.p.say("Адрес должен начинаться с https:// (http только для локального сервера).")
			continue
		}
		config.Server = strings.TrimRight(server, "/")
		break
	}
	for {
		project, err := d.p.ask("ID проекта (p_…)", firstNonEmpty(config.Project, hints.ProjectID))
		if err != nil {
			return err
		}
		if project == "" {
			d.p.say("ID проекта обязателен.")
			continue
		}
		config.Project = project
		break
	}
	for {
		machine, err := d.p.ask("Имя машины из connection.json (machine)", firstNonEmpty(config.Machine, hints.Machine))
		if err != nil {
			return err
		}
		if machine == "" {
			d.p.say("Имя машины обязательно: оно должно совпадать с токеном.")
			continue
		}
		config.Machine = machine
		break
	}
	for {
		name, err := d.p.ask("Имя этого компьютера в панели", firstNonEmpty(config.Name, defaultHostName()))
		if err != nil {
			return err
		}
		if !identifier.MatchString(name) {
			d.p.say("Допустимы латинские буквы, цифры, «-», «_», «.»; до 128 символов.")
			continue
		}
		config.Name = name
		break
	}
	for {
		state, err := d.p.ask("Приватный каталог состояния worker", firstNonEmpty(config.StateDir, filepath.Join(filepath.Dir(d.configPath), "state")))
		if err != nil {
			return err
		}
		if !filepath.IsAbs(state) {
			d.p.say("Нужен абсолютный путь.")
			continue
		}
		config.StateDir = state
		break
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

var errSetupAborted = errors.New("setup cancelled by the owner; nothing was written")
