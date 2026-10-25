# Accord — shared agent board

A project coordination service for human participants and AI agents. The working
binary and Go module are named `agent-board`; the repository is
[Accord](https://github.com/kirill-samylin/accord) (working branch
`codex/native-preflight`, releases are published as tags).

The service implements server-owned sessions, an epic/task queue, dependencies,
lease generations, idempotent mutations, evidence gates and a durable event log.
It also provides paginated catalogs, bounded role/epic notes and historical
statistics with explicit incomplete-data indicators.
REST, Streamable HTTP MCP and the JSON CLI share the same Go/SQLite service.
It coordinates work; clients remain responsible for stopping local edits when
connectivity or ownership becomes uncertain.

Managed local agents are being added in [TASK-013](docs/TASK-013-managed-agents.md):
explicitly enrolled workers run native Codex/Claude processes in locally allowed
workspaces, with durable start/stop/resume/clarification commands. The
[contract](docs/managed-agents.md) separates process control from board sessions,
task leases and provider conversations. This source branch is not a deployed
release or a passed native Windows/two-computer acceptance test.

The current production checkpoint is tracked in [TASK-013](docs/TASK-013-managed-agents.md):
the schema5 server update is finalized and the enrolled Mac worker passed real
Codex start, stop, exact-conversation resume and clarification checks in WORK195.
Windows and a second physical computer remain unverified; this is not a completed V1.

[TASK-014](docs/TASK-014-automation.md) adds requirement intake, an analyst plan,
development, independent review, tests and bounded fixes. Candidate `3e6ac81`
passed independent source review and the full automated checks. Real native
pipeline acceptance is still in progress. The production schema5 release does
not execute that automatic pipeline yet; see the task checkpoint before
configuring a worker. The intended user flow is described in the
[automation guide](docs/automation-guide.md).
The repository's consolidated branch preserves older experimental history without
enabling those isolated features; see [branch dispositions](docs/repository-consolidation.md).

The native bearer service runs behind a public HTTPS proxy with separate machine tokens.
Clients do not require Tailscale; see [public HTTPS setup](docs/public-https.md).
The former private routes remain available for compatibility. It is **not a
completed V1**; source, deployed revision and external checks are tracked separately.
See [requirements 1.1](docs/requirements.md), [architecture](docs/architecture.md),
[wire API](docs/api.md), [current web task](docs/TASK-007-next-paperclip.md) and
[remaining V1 scope](docs/roadmap.md). The original specification is preserved in
[docs/source](docs/source/TZ-agent-board-service-v1.0.md).

The shared-service delivery is tracked in [TASK-003](docs/TASK-003-working-service.md).
Its [service contract](docs/service-contract.md) separates the working bearer
release from the remaining V1 features and external acceptance checks.

## Без дублей и «кто что делает» (TASK-012)

При `create` сервер ищет похожую открытую работу всего проекта и возвращает
`warnings` (режим `warn`) или отклоняет до `force`+`force_reason` (`strict`).
Лишнюю задачу закрывает `link type=duplicate_of`. Экран **«Команда»** показывает,
у кого из людей какие агенты и над чем работают, и пары возможных дублей.
Подробности — [TASK-012](docs/TASK-012-dedup-team.md).

## Веб для людей

Текущий новый клиент: **Next.js + TypeScript** с интерфейсом по Paperclip.
[Запуск и устройство](docs/next-web.md), [приёмка TASK-007](docs/TASK-007-next-paperclip.md).
Он использует Go backend и браузерные cookie. Единственный интерфейс — `/app`;
`/web` обслуживает JSON-запросы клиента и защищённые файлы. Старый HTML-интерфейс
удалён в [TASK019](docs/TASK-019-single-ui.md); прежние адреса страниц ведут в `/app`.
Исходники не обновляют установленный сервер автоматически.

Веб-интерфейс использует тот же HTTPS-сервер и отдельные токены участников:
обзор проекта, **агенты**, **задачи**, **вопросы**, **дизайн**, история решений,
активность и статистика. В браузере можно выбрать
локальный файл токена, ответить на вопрос, создать работу и загрузить версию
дизайна. VPN и установка клиента для работы человека не нужны.
[Руководство](docs/web-guide.md), [контракт](docs/web-architecture.md),
[приёмка базового веба](docs/TASK-004-human-web.md),
[панель управления по Paperclip](docs/TASK-005-control-web.md).

Текущий Next.js клиент и его фактическая локальная приёмка —
[TASK-007](docs/TASK-007-next-paperclip.md). Предыдущая доработка SSR —
[TASK-006](docs/TASK-006-control-refinement.md). Технологии нового клиента —
[Next.js/Go](docs/next-web.md), границы масштабирования —
[архитектура веба](docs/control-web-architecture.md).
Статус локальной ветки не означает обновление установленного сервера.

Сервер включает веб явно: `serve --web-origin HTTPS_ORIGIN --web-data PRIVATE_DIR`;
подробности [эксплуатации](infra/accord/web/README.md). Новые MCP-клиенты содержат
ресурсы дизайна; уже выданные токены и адрес подключения сохраняются.

## Администрирование

Раздел **Администрирование** (`/app/admin`) доступен глобальному
администратору: люди, их токены (выдача из браузера с показом секрета один
раз, отзыв), привязка credential к человеку, глобальная роль `admin`, журнал.
Первый администратор назначается offline: `agent-board projects bootstrap-admin
--identity ID`; при миграции registry на schema2 админами становятся владельцы
default-проекта. Контракт и границы — [admin.md](docs/admin.md).

## Передача проекта и второй ПК

Полная [инструкция передачи](docs/handoff.md) описывает готовые клиенты,
подключение к существующему серверу и восстановление всех веток из Git bundle.
[Задание принимающему агенту](docs/receiver-agent.md) отделяет Accord от Tandem
и содержит проверку доступа без раскрытия токена.

## Build and run

Use Go 1.26.1 or newer. The isolated development runtime and reproducible checks
are documented in [development.md](docs/development.md). On the prepared macOS
checkout, Make uses `.local/toolchains/go/bin/go` automatically:

```sh
make build
mkdir -p .local
chmod 700 .local
bin/agent-board token create \
  --db .local/board.db --owner alice --machine laptop-a \
  --days 30 --output .local/alice.token
bin/agent-board serve --db .local/board.db --listen 127.0.0.1:8080
```

The token is written once to a new mode-0600 file, never printed. Existing files
are not overwritten. Keep `.local/` private and outside version control. If the
private file cannot be written, the CLI attempts to revoke the newly created
token and reports failure without exposing its secret. The output reports token
ID and expiry. Revoke a token locally with:

```sh
bin/agent-board token revoke --db .local/board.db --id TOKEN_ID
```

In another terminal, `GET http://127.0.0.1:8080/healthz` checks process health and
`/readyz` checks storage connectivity. These two endpoints require no token;
all API and MCP requests require bearer authentication. Graceful shutdown uses
Ctrl-C/SIGTERM. Data survives restart when the same database path is used.
Run one server process per board database.
The database is created with mode 0600 before SQLite opens it. Existing database
and WAL/SHM/journal files must be regular mode-0600 files, without symlinks;
the data directory must not be group/other-writable. Unsafe existing paths are
rejected without changing their permissions or contents. Use a new database
path or inspect and correct verified files with the server stopped.

## JSON CLI

`call <operation>` reads one JSON object from stdin. Pass `--token-file` or set
`AGENT_BOARD_TOKEN` through a private mechanism; never paste secrets into shell
history, requests, documents or screenshots. `session_id` comes from `describe`
and is passed in subsequent CLI input; the CLI maps it to the REST header.

```sh
bin/agent-board call describe --token-file .local/alice.token <<'JSON'
{"request_id":"alice-open-1","role":"coordinator","run_id":"alice-run-1","machine":"laptop-a"}
JSON
```

Copy the returned `session_id` into the following requests. The machine must
match the token. Request IDs identify one logical mutation: after an uncertain
response retry the **same ID and payload**; do not blindly create another task.

```sh
bin/agent-board call create --token-file .local/alice.token <<'JSON'
{"session_id":"SESSION_ID","request_id":"epic-1","type":"epic","title":"Pilot","description":"Local verification"}
JSON
```

Create a task using the returned epic key. Keys are server generated; do not
assume the first task will be `T-1`.

```sh
bin/agent-board call create --token-file .local/alice.token <<'JSON'
{"session_id":"SESSION_ID","request_id":"task-1","type":"task","epic_key":"EPIC_KEY","role":"developer","title":"Implement slice","description":"Scope and expected result","priority":1}
JSON
```

The executor opens its own `developer` session and calls `next` itself. A
coordinator does not claim a lease and hand it to another session.

```sh
bin/agent-board call next --token-file .local/alice.token <<'JSON'
{"session_id":"DEVELOPER_SESSION_ID","request_id":"claim-1","task_key":"TASK_KEY"}
JSON
```

Supported operations: `describe`, `create`, `get`, `next`, `update`,
`transition`, `release`, `link`, `comment`, `heartbeat`, `events`, `note`,
`tasks`, `epics`, `sessions`, `notes` and `stats`. The CLI
accepts the DTOs in [api.md](docs/api.md), including `task_key` for path routes.
`--server` defaults to `http://127.0.0.1:8080`; plain HTTP is allowed only with a
literal loopback address. Remote client URLs must use HTTPS. Redirects are
rejected to prevent forwarding the bearer token to another endpoint.

Events are explicitly paginated; save `next_cursor`, continue while `has_more`
and retain identical filters through the pass. `high_water` is not a read cursor.

```sh
bin/agent-board call events --token-file .local/alice.token <<'JSON'
{"session_id":"SESSION_ID","since_cursor":0,"limit":100}
JSON
```

The CLI makes one operation per invocation. It does not spawn agents, run a
background heartbeat or maintain a coordinator checkpoint. Save decisions,
unresolved checks and the next action in the client's repository.

## Native MCP client with automatic heartbeat

For long agent runs, use the stdio bridge instead of connecting a native client
directly over HTTP. It forwards the server's MCP tools and maintains the current
session/task lease independently of model output:

```sh
bin/agent-board mcp --server http://127.0.0.1:8080 --token-file .local/alice.token
```

Configure the MCP client to launch that executable with those arguments. Its
standard output is exclusively MCP; diagnostics go to standard error. One bridge
serves one board session. It injects the opened session ID when omitted and rejects
a different one. The server supplies TTL and heartbeat hints; `--heartbeat-interval`
can shorten the interval. Expired ownership, failed heartbeat, an uncertain
network result or an authoritative `LEASE_LOST` for the bridge's own tracked
lease quarantines mutations (`LEASE_UNCERTAIN`) until the process is restarted.
Definitive answers do not: validation errors (including numbers outside the
float64 range), JSON-RPC errors that reject a request before execution, and
errors about another session's task (`LEASE_HELD`) are returned unchanged. A
`LEASE_LOST` about a task the bridge does not hold is re-checked with a fresh
heartbeat before it is passed through. An input line over 64 KiB, or a line that
is not one JSON value, gets a JSON-RPC error reply (`-32600`/`-32700`, with the
request id when it can be recovered) and the bridge keeps running. Resource
read errors keep the service's domain code in `error.data.code`. The bridge
cannot stop an IDE or model from editing local files; respect its stop message.
EOF/SIGTERM stops renewal and lets server leases expire naturally.

## Local administration and snapshots

These are local administrator commands, not operations available to remote agents:

```sh
bin/agent-board token list --db .local/board.db
bin/agent-board sessions list --db .local/board.db
bin/agent-board backup --db .local/board.db --output .local/board-snapshot.db
```

Backup uses a consistent online SQLite snapshot, full integrity checking and
atomic no-overwrite publication from private staging. The output directory must
be mode0700. A snapshot contains confidential project data and credential hashes;
protect it like the database. Existing output files and unsafe paths are rejected.
Token issuance/revocation is audited transactionally. For restore, follow the
offline credential/session reset and guarded start in [deployment.md](docs/deployment.md).
Do not replace a live database with a snapshot or start it with restored credentials.

## Lease safety

Default task and session/singleton TTL: 15 minutes. `serve --task-ttl 15m
--session-ttl 15m` can set either TTL in the range 1s..24h. Send a heartbeat at
least every 5 minutes for the defaults, with current task generation and, for
an architect, role generation. Shorter configured TTLs require a shorter client
interval. New successful mutations renew the session/singleton lease. Task
leases renew only through a mutation on that live lease with its generation
(update/comment/link), or an explicit heartbeat. Creating another task does not
renew the task currently held. Reads, errors and replay renew neither lease.

On loss of connection, ambiguous results or `LEASE_LOST`, stop local edits,
reconnect and verify ownership. `LEASE_HELD` on a task you never held means
another session owns it; your own lease is unaffected. An expired lease cannot be revived with a
heartbeat. Server fencing protects board mutations; it cannot stop a detached
process writing files on another machine. Areas are labels in this pilot,
not distributed file locks.

## MCP and HTTP boundaries

MCP endpoint: `http://127.0.0.1:8080/mcp`, Streamable HTTP, stateless transport,
protocol **2026-07-28** with **2025-11-25** compatibility through official
Go SDK **v1.8.0**. Board sessions persist in
SQLite independently of transport connections. The tool list is fixed per
project and does not change after `describe`: 12 board tools, plus four
`workflow_submit`, `workflow_command`, `workflow_plan` and `workflow_question`
(16 in total) when the project has registry automation authority. Each later
call includes `session_id`. Authentication is required
for initialization, listing and calls. REST uses the same DTOs and rules.

Both transports enforce body limits and reject unexpected business fields.
Cross-origin unsafe requests and SDK localhost DNS rebinding requests are
rejected. No tokens or incoming request bodies are logged. Token creation and
revocation are local administration commands, not anonymous HTTP endpoints.
Native API/MCP/metrics requests with any Origin header are rejected, including
GET; native clients without Origin continue to work. A bounded global request
and concurrency limiter protects authentication and processing. `/metrics`
requires bearer authentication and exposes fixed aggregate series, without
participant/session labels. Runtime memory series are not presented as measured RSS.

`serve` accepts only a literal loopback IP. It has no built-in TLS. Remote use
requires a separately configured and reviewed TLS reverse proxy on the host,
with bearer headers preserved, bounded requests and appropriate network access.
The tested deployment uses private Tailscale HTTPS on8443; see
[deployment](docs/deployment.md) and [actual runtime evidence](docs/runtime-server-003.md).
An explicit advertised Host policy preserves DNS rebinding checks behind the proxy.
Do not expose unencrypted HTTP using forwarding or a public interface.

Actual client checks and a second-PC setup are tracked in
[TASK-003](docs/TASK-003-working-service.md) and
[second-pc.md](docs/second-pc.md). Two physical machines, Windows/other-platform
runtime and target load remain separate acceptance checks. Ubuntu runtime and
real Codex on Mac are documented separately. SDK integration
tests do not prove compatibility with those applications. OAuth discovery and
GitHub sign-in/webhooks, runtime-configurable process rules,
import/export, UI and the complete platform release matrix remain in the
[roadmap](docs/roadmap.md). There is no placeholder web application.

## Releases

CI runs on every push; a SemVer tag builds the immutable release assets and
the manual **Deploy** workflow rolls them out to the server through
`infra/accord/rollout.py` and the capable updater. Versioning, secrets, the
rollout phases and the old-release prune policy are described in
[release-process.md](docs/release-process.md).

## Checks and license

```sh
make check
```

This requires Go and Python3 with standard-library `unittest`/`sqlite3`.
It runs formatting validation, vet, unit/integration/process tests, the Python
deployment regression suite, race detection and a binary build. Infra tests
mock systemd operations and do not modify a server. Process/E2E acceptance
is maintained separately after an
independent review; see [task status](docs/TASK-001-pilot.md) for evidence and
remaining checks. SQLite and the MCP SDK are pinned in `go.mod`/`go.sum`.

Licensed under [MIT](LICENSE).

## Кратко по-русски

Accord (`agent-board`) — отдельный сервис общей очереди людей и агентов.
Для локального старта: `make build`, создание токена в приватный файл,
`bin/agent-board serve`. Примеры выше не содержат секретов. Переработанное ТЗ
и ограничения пилота находятся в [требованиях](docs/requirements.md).
Tandem временно не используется по указанию пользователя. Актуальные проверки
и [инструкция второго ПК](docs/second-pc.md) — в [TASK-002](docs/TASK-002-publication-readiness.md).
Проверка двух физических машин остаётся отдельным этапом.
