# Accord — agent-board pilot

A project coordination service for human participants and AI agents. The working
binary and Go module are named `agent-board`; the repository is
[Accord](https://github.com/kirill-samylin/accord).

The pilot implements server-owned sessions, an epic/task queue, dependencies,
lease generations, idempotent mutations, evidence gates and a durable event log.
REST, Streamable HTTP MCP and the JSON CLI share the same Go/SQLite service.
It coordinates work; clients remain responsible for stopping local edits when
connectivity or ownership becomes uncertain.

This is an initial local pilot, **not a production release or completed V1**.
See [requirements 1.1](docs/requirements.md), [architecture](docs/architecture.md),
[wire API](docs/api.md), [task status](docs/TASK-001-pilot.md) and
[remaining V1 scope](docs/roadmap.md). The original specification is preserved in
[docs/source](docs/source/TZ-agent-board-service-v1.0.md).

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
`transition`, `release`, `link`, `comment`, `heartbeat`, `events`. The CLI
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
reconnect and verify ownership. An expired lease cannot be revived with a
heartbeat. Server fencing protects board mutations; it cannot stop a detached
process writing files on another machine. Areas are labels in this pilot,
not distributed file locks.

## MCP and HTTP boundaries

MCP endpoint: `http://127.0.0.1:8080/mcp`, Streamable HTTP, stateless transport,
protocol **2026-07-28** with **2025-11-25** compatibility through official
Go SDK **v1.8.0**. Board sessions persist in
SQLite independently of transport connections. The 11 tools stay fixed after
`describe`; each later call includes `session_id`. Authentication is required
for initialization, listing and calls. REST uses the same DTOs and rules.

Both transports enforce body limits and reject unexpected business fields.
Cross-origin unsafe requests and SDK localhost DNS rebinding requests are
rejected. No tokens or incoming request bodies are logged. Token creation and
revocation are local administration commands, not anonymous HTTP endpoints.

`serve` accepts only a literal loopback IP. It has no built-in TLS. Remote use
requires a separately configured and reviewed TLS reverse proxy on the host,
with bearer headers preserved, bounded requests and appropriate network access.
The pilot does not ship a public proxy deployment or claim that it was tested.
Do not expose unencrypted HTTP using forwarding or a public interface.

Actual client checks and a second-PC setup are tracked in
[TASK-002](docs/TASK-002-publication-readiness.md) and
[second-pc.md](docs/second-pc.md). Two physical machines, Linux/Windows runtime,
production security and load remain separate acceptance checks. SDK integration
tests do not prove compatibility with those applications. OAuth discovery and
GitHub sign-in/webhooks, configurable process rules, full statistics,
import/export, UI, release packages and production hardening remain in the
[roadmap](docs/roadmap.md). There is no placeholder web application.

## Checks and license

```sh
make check
```

This runs formatting validation, vet, unit/integration tests, race detection
and a binary build. Process/E2E acceptance is maintained separately after an
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
