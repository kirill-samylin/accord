# Локальные разрешения Codex, Claude и Gennadiy для Accord

Это отдельное разрешение владельца компьютера. Project editor/controller grant
даёт управление процессами через worker, но не отменяет native CLI permissions.
Отказ native CLI нельзя исправлять ложным `readOnlyHint` для изменяющих операций,
глобальным autoapprove или отключением sandbox. `describe` создаёт сессию,
`next` получает аренду задачи; обе операции изменяют состояние доски.

## Private worker config

Опциональное поле `native_mcp_approved_tools` содержит map provider → точные
имена инструментов Accord. Допустимы только configured `codex`/`claude`/`gennadiy`
и уникальные имена из следующего фиксированного каталога:

```text
comment create describe events get heartbeat link next note release transition update
```

Unset/null/пустой список не добавляет native permission overrides для Codex и
Claude. Для Gennadiy см. границу ниже: native default там allow, поэтому
невыбранные инструменты моста всегда явно deny. Worker не
выдаёт разрешения сам, не принимает их из браузера/managed command/profile и
не меняет глобальную конфигурацию CLI. Поле хранится только локально у владельца;
snapshot для подготовки запуска копирует также этот map и вложенные списки.

Пример **отдельного согласия только для тестового чтения**:

```json
{
  "native_mcp_approved_tools": {
    "codex": ["describe", "next"]
  }
}
```

Этот фрагмент не применяется автоматически. В текущей приёмке native Codex
сначала отказал с `MCP tool call requires approval, but approval policy is never`.
Повтор после отказа с таким разрешением допустим только после явного ответа
человека. Тестовый scope — новая synthetic доска и read-only рабочая копия;
production и GUI checkout этим согласием не затрагиваются.

Для обычной работы владелец отдельно выбирает необходимые операции, осознавая
их эффект. Например, согласие на `describe/next` не предоставляет модели вызов
остальных MCP tools, включая `update/transition`. Внутренний heartbeat bridge
для уже полученной аренды сохраняется. Server ACL, assignment, evidence и lease gates
действуют при любом native разрешении. Native builtin tools и права файловой
системы этой опцией не меняются.

## Точная граница адаптеров

Codex: при непустом списке только временная запись `accord_managed` получает
`enabled_tools` с этим списком, default approval mode `prompt`, явные политики
для всех12 известных tools: выбранные `approve`, остальные `prompt`. Чужие MCP
connectors остаются disabled. Command/args bridge по-прежнему проверяются точным
вторым config-only inspection. CLI `mcp list` не раскрывает tool policy; её
runtime introspection не заявляется. Installed CLI parser/get и generator unit
проверяют поддержанную форму и выбранный каталог отдельно.

Claude: только точные имена `mcp__accord_managed__TOOL` добавляются к
`--allowedTools`; оставшиеся известные инструменты этого bridge — к
`--disallowedTools`. `--strict-mcp-config` и per-attempt binding сохраняются.
`--dangerously-skip-permissions` не используется. Cross-build/unit не является
проверкой настоящего Claude на Windows.

Gennadiy (OpenCode fork): у OpenCode нет флага MCP-конфигурации, worker передаёт
`OPENCODE_CONFIG_CONTENT` только дочернему процессу этой попытки. В нём ключи
`accord_managed_TOOL` для всех15 известных tools: выбранные `allow`, остальные
`deny` — и на уровне `permission`, и в `agent.build.permission`, потому что
agent-level policy перекрывает глобальную. Headless `run` отвечает `reject` на
каждый native `ask`, так что prompt-режим недоступен и preset должен быть явным.
`edit`/`bash` следуют локальному `gennadiy_permission_mode` (по умолчанию
`read-only` = deny), `external_directory` всегда deny. Чужие connectors из
`debug config` получают `enabled=false`; URL и headers не копируются.
`--dangerously-skip-permissions` не используется; второе чтение `debug config`
с инъекцией обязано вернуть ровно это binding и policy. Проверено unit-тестами
на synthetic config и config-only тестом `gennadiy_local_test.go` на
установленном gennadiy1.18.10 (WORK295, без вызова модели); реальный managed
запуск модели этим не заявляется.

Локальный владелец меняет config только при остановленном worker, затем
выполняет обычные `worker consent` и `worker run`. Уже работающий процесс не
получает новые разрешения из изменения файла или команды другого участника.

## Источники и приёмка

Поддержанные настройки проверяются по официальным
[Codex MCP options](https://learn.chatgpt.com/docs/extend/mcp),
[Claude Code CLI](https://code.claude.com/docs/en/cli-reference) и
[OpenCode config/permissions](https://opencode.ai/docs/config/); форк gennadiy
сверен локально по `gennadiy run --help` и `gennadiy debug config`.
Installed Codex `0.163.0-alpha.11.1` config-only parser принял disabled synthetic
server с per-tool `approve` и отклонил неправильное значение enum; `mcp get`
вернул ровно выбранные `enabled_tools`. Модель и MCP actions этим probe не
запускались. Само наличие опции не закрывает native acceptance: её результат
и разрешение человека учитываются отдельно в TASK013/WORK168.
