package worker

import (
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Ready-made bridge tool consents the dialogue offers. The owner may still
// edit native_mcp_approved_tools by hand; the presets only avoid typos.
var (
	setupReadOnlyTools = []string{"describe", "next", "get", "events"}
	setupTaskTools     = []string{"describe", "next", "get", "events", "update", "comment", "note", "transition", "release", "link"}
)

type setupDialogue struct {
	p          *setupPrompter
	probe      setupProbe
	configPath string
	found      map[string][]discoveredCLI
	existing   bool
}

// runSetupDialogue builds the new local allowlist. An existing file is edited:
// server/project/token_file/state_dir/controllers/automation are kept as they
// are and only executables, workspaces, modes, tools and models are revisited.
func runSetupDialogue(p *setupPrompter, probe setupProbe, configPath string, current Config, existing bool) (Config, error) {
	d := &setupDialogue{p: p, probe: probe, configPath: configPath, found: probe.discover(), existing: existing}
	for _, provider := range nativeProviders {
		if len(d.found[provider]) == 0 {
			p.say("  %s: не найден", provider)
		}
		for _, cli := range d.found[provider] {
			p.say("  %s: %s", provider, cli.describe())
		}
	}
	config := current
	if existing {
		p.say("\nШаг 1. Подключение сохранено: сервер %s, проект %s, компьютер %s.", config.Server, config.Project, config.Name)
		p.say("Эти поля, controllers и automation не меняются этой командой.")
	} else if err := d.askConnection(&config); err != nil {
		return config, err
	}
	if err := d.askProviders(&config); err != nil {
		return config, err
	}
	if err := d.askWorkspaces(&config); err != nil {
		return config, err
	}
	if err := d.askModes(&config); err != nil {
		return config, err
	}
	if err := d.askTools(&config); err != nil {
		return config, err
	}
	if err := d.askModels(&config); err != nil {
		return config, err
	}
	d.summary(config)
	ok, err := p.confirm("Записать эту конфигурацию в "+configPath+"?", true)
	if err != nil {
		return config, err
	}
	if !ok {
		return config, errSetupAborted
	}
	return config, nil
}

// askProviders selects one acceptable executable per provider or skips it.
func (d *setupDialogue) askProviders(config *Config) error {
	d.p.say("\nШаг 2. Агенты (CLI) на этом компьютере. Выберите установленные программы.")
	d.p.say("Принимаются только настоящие исполняемые файлы по абсолютному пути; скрипты-шимы не принимаются.")
	executables := map[string]string{}
	for _, provider := range nativeProviders {
		options := []string{"не подключать " + provider}
		var paths []string
		fallback := 0
		for _, cli := range d.found[provider] {
			options = append(options, cli.describe())
			paths = append(paths, cli.Path)
			if config.Executables[provider] == cli.Path {
				fallback = len(paths)
			}
		}
		if current := config.Executables[provider]; current != "" && fallback == 0 {
			options = append(options, "оставить текущий: "+current)
			paths = append(paths, current)
			fallback = len(paths)
		}
		options = append(options, "указать другой абсолютный путь")
		choice, err := d.p.choose("Провайдер "+provider+":", options, fallback)
		if err != nil {
			return err
		}
		switch {
		case choice == 0:
			continue
		case choice <= len(paths):
			executables[provider] = paths[choice-1]
		default:
			path, err := d.askExecutablePath(provider)
			if err != nil {
				return err
			}
			if path != "" {
				executables[provider] = path
			}
		}
	}
	config.Executables = executables
	return nil
}

func (d *setupDialogue) askExecutablePath(provider string) (string, error) {
	for {
		path, err := d.p.ask("Абсолютный путь к исполняемому файлу "+provider+" (пусто — не подключать)", "")
		if err != nil || path == "" {
			return "", err
		}
		resolved, _ := d.probe.resolve(provider, path)
		if resolved == "" {
			d.p.say("Файл не принят: нужен существующий обычный файл без символических ссылок и не скрипт.")
			continue
		}
		return resolved, nil
	}
}

// askWorkspaces keeps existing folders (re-asking their providers) and adds
// new ones. Every workspace needs at least one configured provider.
func (d *setupDialogue) askWorkspaces(config *Config) error {
	d.p.say("\nШаг 3. Рабочие папки. Агент работает только внутри перечисленных папок.")
	configured := sortedProviders(config.Executables)
	if len(configured) == 0 {
		d.p.say("Не выбран ни один провайдер; без него подключение невозможно.")
		return errSetupAborted
	}
	workspaces := map[string]Workspace{}
	for _, alias := range sortedAliases(config.Workspaces) {
		workspace := config.Workspaces[alias]
		providers, err := d.askWorkspaceProviders(alias, workspace.Path, configured, workspace.Providers)
		if err != nil {
			return err
		}
		workspaces[alias] = Workspace{Path: workspace.Path, Providers: providers}
	}
	for len(workspaces) < 32 {
		prompt := "Добавить рабочую папку (абсолютный путь; пусто — закончить)"
		if len(workspaces) == 0 {
			prompt = "Абсолютный путь первой рабочей папки"
		}
		path, err := d.p.ask(prompt, "")
		if err != nil {
			return err
		}
		if path == "" {
			if len(workspaces) == 0 {
				d.p.say("Нужна хотя бы одна папка.")
				continue
			}
			break
		}
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil || !filepath.IsAbs(canonical) || validateLocalPath(canonical) != nil {
			d.p.say("Папка не принята: нужен существующий абсолютный путь без символических ссылок и не сетевой диск.")
			continue
		}
		alias, err := d.p.ask("Короткое имя папки в панели", sanitizeIdentifier(filepath.Base(canonical)))
		if err != nil {
			return err
		}
		if _, taken := workspaces[alias]; taken || !identifier.MatchString(alias) {
			d.p.say("Имя занято или недопустимо (латиница, цифры, «-», «_», «.»).")
			continue
		}
		providers, err := d.askWorkspaceProviders(alias, canonical, configured, configured)
		if err != nil {
			return err
		}
		workspaces[alias] = Workspace{Path: canonical, Providers: providers}
	}
	config.Workspaces = workspaces
	return nil
}

func (d *setupDialogue) askWorkspaceProviders(alias, path string, configured, current []string) ([]string, error) {
	var fallback []string
	for _, provider := range current {
		if slices.Contains(configured, provider) {
			fallback = append(fallback, provider)
		}
	}
	if len(fallback) == 0 {
		fallback = configured
	}
	for {
		answer, err := d.p.ask("Провайдеры для папки "+alias+" ("+path+"), через запятую из "+strings.Join(configured, ", "), strings.Join(fallback, ","))
		if err != nil {
			return nil, err
		}
		providers := splitList(answer)
		valid := len(providers) > 0
		for _, provider := range providers {
			valid = valid && slices.Contains(configured, provider)
		}
		if valid {
			sort.Strings(providers)
			return providers, nil
		}
		d.p.say("Укажите хотя бы один из настроенных провайдеров.")
	}
}

func sortedAliases(workspaces map[string]Workspace) []string {
	var aliases []string
	for alias := range workspaces {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

func (d *setupDialogue) summary(config Config) {
	d.p.say("\nИтог:")
	for _, provider := range sortedProviders(config.Executables) {
		d.p.say("  %s: %s", provider, config.Executables[provider])
	}
	for _, alias := range sortedAliases(config.Workspaces) {
		d.p.say("  папка %s: %s — %s", alias, config.Workspaces[alias].Path, strings.Join(config.Workspaces[alias].Providers, ", "))
	}
	d.p.say("  права: codex=%s claude=%s gennadiy=%s", firstNonEmpty(config.CodexSandbox, "по умолчанию"), firstNonEmpty(config.ClaudePermissionMode, "по умолчанию"), firstNonEmpty(config.GennadiyPermissionMode, "read-only"))
	for _, provider := range sortedProviders(config.Executables) {
		d.p.say("  инструменты Accord для %s: %s; модели: %s", provider, firstNonEmpty(strings.Join(config.NativeMCPApprovedTools[provider], ","), "нет"), firstNonEmpty(strings.Join(config.Models[provider], ","), "по умолчанию CLI"))
	}
}
