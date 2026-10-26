# WORK295 — третий managed provider: gennadiy (OpenCode fork)

Исполнитель developer, ветка `codex/gennadiy-provider` от
`codex/native-preflight` (46642af). **UNIT PASS / стенд обновлён до 41681b8 /
real native gennadiy run NOT-RUN**.
Поручение пользователя 26.10.2026: внедрить локальный корпоративный агент
`gennadiy` (fork OpenCode 1.18.10 с Keycloak, корпоративным LLM gateway и
корпоративными MCP) как третий провайдер управляемых агентов Accord наряду с
Codex и Claude Code: worker, сервер, веб, документация, тесты.

## Что проверено у gennadiy до реализации (локально, без запуска модели)

- `gennadiy run --help`: `--format json`, `--session`, `--agent`,
  `--dangerously-skip-permissions` (не используется), `--continue`/`--fork`
  (не используются). Prompt читается из stdin, когда stdin не TTY.
- `run --format json` пишет JSON-строки `{type,timestamp,sessionID,part…}`:
  `step_start`, `text`, `tool_use`, `step_finish`, `error`. Явного success
  record нет; процесс выходит с0 после idle, с1 при ошибке.
- Headless `run` на `permission.asked` отвечает `reject`; `question`,
  `plan_enter`, `plan_exit` запрещены. Default agent корпоративного config —
  `plan` (edit/bash deny), `build` — edit/bash `ask`.
- `gennadiy debug config` печатает resolved JSON в stdout (0.7с, stderr пуст);
  `OPENCODE_CONFIG_CONTENT` глубоко сливается с глобальным config (проверено
  инъекцией `mcp`/`permission`).
- Session ID: `ses_` + 26 base62 символов, не UUID.
- Executable: `~/.gennadiy/bin/gennadiy` — bash shim; настоящий Mach-O —
  `node_modules/gennadiy-darwin-arm64/bin/gennadiy` в npm-пакете.
- `run --help` форк печатает в **stderr** (yargs), `--version` и
  `debug config` — в stdout. Worker на Unix stderr отбрасывает, поэтому для
  preflight добавлен bounded config-only режим `launchSpec.mergeStderr`
  (только `run --help`; Windows и так объединяет оба потока).

## Реализованный объём

Worker (`internal/worker`):

- `provider.go`: `providerArgs("gennadiy")` → `run --format json --agent build`
  [+ `--session ses_…`]; `parseProviderEvent` для `step_start`/`step_finish`/
  `text`/`error` и camelCase `sessionID`; `validSessionID` по провайдеру;
  общие `nativeProviders`/`knownProvider`/`profileProviderFor` из `core`;
  `providerEnv` дополнительно не наследует ambient `OPENCODE_CONFIG*`.
- `config.go`: `gennadiy_permission_mode` (`read-only`|`workspace-write`),
  regexp `gennadiySessionID`, до3 executables.
- `gennadiy_config.go` (новый): discovery чужих connectors через
  `debug config`; `gennadiyPermissions` (edit/bash по режиму,
  `external_directory` deny, `accord_managed_TOOL` allow/deny для всех15
  tools); `gennadiyMode` по роли automation как `automationSandbox`;
  `gennadiyConfigContent` (`mcp` с `enabled=false` для чужих без копирования
  URL/headers, `accord_managed` этой попытки, `permission` и
  `agent.build.permission`); `verifyGennadiyConfig`/`checkGennadiyConfig` —
  второе config-only чтение с инъекцией до spawn.
- `bridge_config.go`: `bridgeArgs` возвращает `bridgeLaunch{Args, Env}`;
  gennadiy получает только `Env` (`OPENCODE_DISABLE_AUTOUPDATE`,
  `OPENCODE_DISABLE_MODELS_FETCH`, `OPENCODE_CONFIG_CONTENT`).
  `preparation.go`, `execution.go`, `automation_launch.go` пробрасывают env в
  `launchSpec`; `engine.go` — seams `discoverGennadiy`/`verifyGennadiy`;
  `discovery.go` — `readNativeWithEnv` с явным env и таймаутом (20с для
  `debug config`).
- `probe.go`: preflight `run --help` (через `readGennadiyHelp`, stderr
  merged) на `--format`, `--session`, `--agent`, `json`; `process.go`/
  `process_unix.go` — флаг `mergeStderr` для config-only чтений. `recover_scan.go`, `native_consent.go`, `automation_config.go` —
  третий провайдер в фиксированных списках.

Сервер: `internal/core/managed.go` — `NativeProviders`, `KnownNativeProvider`,
`ProfileProviderFor`; `store/managed_hosts.go` — providers до3 из списка;
`store/managed_runs.go` — соответствие профиля `gennadiy`;
`store/profiles.go` — provider `gennadiy`; `store/managed_report.go` —
`managedSessionID` принимает `ses_[A-Za-z0-9]{16,64}`.

Веб: `providerName`/`profileSupportsProvider` (`lib/managed-agents.ts`),
`providerLabels` (`lib/agent-profiles.ts`, select профиля), опция в
`automation/team-setup.tsx`, тексты `managed/host-card.tsx` и
`lib/worker-errors.ts`.

Документация: `docs/managed-agents.md` (адаптер), `docs/managed-worker-guide.md`
(пример `worker.json`, границы), `docs/native-agent-permissions.md`
(граница policy), README, AGENTS.md.

## Границы и допущения

- Preset прав явный: headless `run` отвергает любой native «ask», prompt-режим
  Codex/Claude здесь невозможен. Невыбранные bridge tools всегда `deny`, так
  как native default для MCP — allow. Без `native_mcp_approved_tools.gennadiy`
  агент не вызовет `describe`/`next`.
- У OpenCode нет OS sandbox: `workspace-write` = обычный shell владельца.
  `external_directory` deny ограничивает только native edit/bash вне cwd.
- Чужие connectors (включая корпоративные Jira/Confluence/Exchange) в
  managed-запуске выключены, как у Codex. Langfuse-трассировка самого
  gennadiy не отключается — текст поручений Accord уйдёт в корпоративный trace.
- `--agent build` фиксирован; `agent.build.permission` перекрывается инъекцией,
  потому что agent-level policy сильнее глобальной.
- Завершение = `step_finish` без последующего `error` и exit0; без
  `step_finish` run failed. Explicit result record у OpenCode нет.
- Git metadata writes/permission inspection (`git_permissions.go`) остаются
  только для Codex; developer-роль gennadiy коммитит обычным `bash` при
  `workspace-write`.
- `gennadiy upgrade` меняет бинарь → `provider_pin_changed`, нужен
  `worker consent` (штатно).

## Фактические проверки

- `make fmt-check vet test build` — PASS; `make race infra-test` — PASS
  (полный прогон до правки `mergeStderr`, затем `go test -race` повторно для
  worker/store/core); `make NEXT_NODE=node24 web-check` (prettier, tsc, 131 web tests,
  `next build`) — PASS.
- Локальный config-only интеграционный тест `gennadiy_local_test.go`
  (`ACCORD_TEST_GENNADIY=<Mach-O>`; NOT-RUN в CI, skip без переменной) на этом
  Mac с установленным gennadiy1.18.10 — **PASS**: `--version`, `run --help`
  (stderr), discovery3 чужих connectors, инъекция, resolved config содержит
  ровно наше binding и policy (включая `agent.build.permission`), подмена
  executable моста отклоняется. Модель и gateway не вызывались.
- `gennadiy_config_test.go` — 317 строк (>300): объединяет аргументы, события,
  validation, policy, content, verify и env-only binding одного адаптера;
  разделение дало бы дубли helper `testGennadiyEngine`.
- Новые unit: `internal/worker/gennadiy_config_test.go` (аргументы, события,
  session format, validation, policy по режиму/роли, content без секретов,
  verify negative cases, env-only binding, discovery/verify failure →
  no launch), `internal/store/managed_gennadiy_test.go` (enroll/profile enum,
  profile/provider match, exact `ses_` session on started),
  `web/src/lib/managed-agents.test.ts` (labels, профиль↔провайдер, startErrors).
- NOT-RUN: реальный `agent-board worker run` с executables.gennadiy, реальный
  `run --format json` через корпоративный gateway, resume того же `ses_`,
  automation roles на gennadiy, Windows. Production VPS/Mac worker не обновлены.

## Выкатка на стенд (26.10.2026)

По прямому разрешению пользователя (SSH из этой сессии, после зелёного CI):
коммит `41681b8041d44312f8ff7e2f5c35dfcdf874350e` запушен в
`origin/codex/gennadiy-provider`; GitHub CI run 40296905801 — Go/Infra/Web
все `success`. Локальный `make release` (Go1.27.2, Node24.19.0) собрал
`.local/releases/41681b8…` и `…-next` (SHA256SUMS 56/56 OK, SOURCE совпадает).
`infra/accord/rollout.py all --prune --keep 1` на `root@157.22.207.48`:
upload (Go57 файлов, Next1345) → stage `staged` → apply `pending-https-smoke`
(full pre-snapshot verified) → smoke `/readyz`200, `/app/login`200 →
finalize `complete` (all-project backup verified) → prune ничего не удалил
(предыдущий `46642af` сохранён как `--keep 1`). Независимый `rollout.py status`
после: `go_current`/`next_current` = `releases/41681b8…`, журнал `complete`,
nginx SHA прежний. Квитанции — ignored `.local/task021/rollout/`.
Схема данных не менялась (schema6, прежние floors); Mac worker и его
`worker.json` не обновлялись — provider `gennadiy` на стенде доступен в
enroll/профилях/UI, но реальный управляемый запуск gennadiy через worker
по-прежнему NOT-RUN. PR в origin не создан из этой сессии (нет `gh`, браузер
не авторизован); тег релиза не ставился.
