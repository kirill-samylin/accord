package worker

import (
	"slices"
	"strings"
)

// askModes asks the write mode of each configured provider and explains the
// consequence in plain words. The defaults are the current values or the
// most restrictive mode; the dialogue never silently widens an existing one.
func (d *setupDialogue) askModes(config *Config) error {
	d.p.say("\nШаг 4. Права агентов в рабочих папках.")
	if config.Executables["codex"] != "" {
		options := []string{"read-only — Codex только читает файлы; правки и команды, меняющие папку, запрещены", "workspace-write — Codex может менять файлы выбранной папки (команды в ней выполняются от вашего имени)"}
		fallback := 0
		if config.CodexSandbox == "workspace-write" {
			fallback = 1
		}
		choice, err := d.p.choose("Режим Codex (codex_sandbox):", options, fallback)
		if err != nil {
			return err
		}
		config.CodexSandbox = []string{"read-only", "workspace-write"}[choice]
	} else {
		config.CodexSandbox = ""
	}
	if config.Executables["claude"] != "" {
		options := []string{"default — Claude Code спрашивает подтверждение на правки; без ответа правка отклоняется", "acceptEdits — правки файлов в папке принимаются автоматически; остальные подтверждения остаются"}
		fallback := 0
		if config.ClaudePermissionMode == "acceptEdits" {
			fallback = 1
		}
		choice, err := d.p.choose("Режим Claude Code (claude_permission_mode):", options, fallback)
		if err != nil {
			return err
		}
		config.ClaudePermissionMode = []string{"default", "acceptEdits"}[choice]
	} else {
		config.ClaudePermissionMode = ""
	}
	if config.Executables["gennadiy"] != "" {
		options := []string{"read-only — Gennadiy только читает; edit и bash запрещены", "workspace-write — Gennadiy может править файлы и выполнять shell от вашего имени (у OpenCode нет песочницы ОС)"}
		fallback := 0
		if config.GennadiyPermissionMode == "workspace-write" {
			fallback = 1
		}
		choice, err := d.p.choose("Режим Gennadiy (gennadiy_permission_mode):", options, fallback)
		if err != nil {
			return err
		}
		config.GennadiyPermissionMode = []string{"read-only", "workspace-write"}[choice]
	} else {
		config.GennadiyPermissionMode = ""
	}
	return nil
}

// askTools picks the Accord bridge tools each provider may call without a
// native prompt. Unselected tools stay under the native policy (Codex/Claude)
// or are denied (Gennadiy, whose headless run rejects every prompt).
func (d *setupDialogue) askTools(config *Config) error {
	d.p.say("\nШаг 5. Какие действия на доске Accord агент может выполнять без подтверждения.")
	d.p.say("«Только чтение»: describe, next, get, events. «Работа по задаче» добавляет update, comment, note, transition, release, link.")
	tools := map[string][]string{}
	for _, provider := range sortedProviders(config.Executables) {
		current := config.NativeMCPApprovedTools[provider]
		options := []string{"только чтение (" + strings.Join(setupReadOnlyTools, ", ") + ")", "работа по задаче (" + strings.Join(setupTaskTools, ", ") + ")", "ничего не разрешать заранее"}
		fallback := 0
		if len(current) > 0 {
			options = append(options, "оставить текущий набор: "+strings.Join(current, ", "))
			fallback = 3
		}
		if provider == "gennadiy" {
			d.p.say("Gennadiy без разрешённых инструментов не сможет взять задачу: его headless-режим отклоняет любые запросы подтверждения.")
		}
		choice, err := d.p.choose("Инструменты Accord для "+provider+":", options, fallback)
		if err != nil {
			return err
		}
		switch choice {
		case 0:
			tools[provider] = slices.Clone(setupReadOnlyTools)
		case 1:
			tools[provider] = slices.Clone(setupTaskTools)
		case 2:
		default:
			tools[provider] = slices.Clone(current)
		}
	}
	if len(tools) == 0 {
		config.NativeMCPApprovedTools = nil
	} else {
		config.NativeMCPApprovedTools = tools
	}
	return nil
}

// askModels records the exact model names a profile may request; empty keeps
// the CLI default. The worker never substitutes another model silently.
func (d *setupDialogue) askModels(config *Config) error {
	d.p.say("\nШаг 6. Модели. Пусто — модель по умолчанию установленного CLI.")
	d.p.say("Если профиль агента в панели указывает конкретную модель, перечислите её точное имя здесь.")
	models := map[string][]string{}
	for _, provider := range sortedProviders(config.Executables) {
		hint := ""
		if provider == "gennadiy" {
			hint = " (формат provider/model, например gennadiy/…)"
		}
		answer, err := d.p.ask("Разрешённые модели для "+provider+hint+", через запятую", strings.Join(config.Models[provider], ","))
		if err != nil {
			return err
		}
		if list := splitList(answer); len(list) > 0 {
			models[provider] = list
		}
	}
	if len(models) == 0 {
		config.Models = nil
	} else {
		config.Models = models
	}
	return nil
}
