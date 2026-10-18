# WORK-021 — stdio MCP bridge и автоматическая аренда

- TASK-003, разработчик; 18.10.2026. Статус: реализация завершена, передано
  независимому reviewer. Самопроверка не является approve.
- Входы: AGENTS.md, TASK-003, service-contract slice A, security-contract,
  API/архитектура; профиль developer и общий процесс соседнего Davinci прочитаны.
- Границы: internal/bridge/**, shared internal/client/**, cmd/mcp.go и его tests,
  узкий client.go/shared policy, main dispatch/help, SessionView TTL и Event
  TokenID/Reason, store/sessions.go TTL. Остальные изменения параллельных
  исполнителей сохранены; VPS/реальные MCP-конфигурации не изменялись.

## Реализация

`agent-board mcp --server ORIGIN --token-file PRIVATE
[--heartbeat-interval DURATION]` предоставляет stdio MCP через официальный
Go SDK1.8.0. Каталог инструментов читается у upstream при старте (до200 tools),
поэтому новые серверные note/list tools не требуют фиксированного local switch.
Изменение каталога уже работающего сервера требует перезапуска bridge.
Session ID локально необязателен, автоматически подставляется единственный
канонический ID; явно чужой ID отклоняется без HTTP-вызова. Новое describe в
живом процессе отклонено; точный успешный replay передаётся upstream для
авторизации/идемпотентности, не меняя текущую локальную аренду.

SessionView расширен дробными секундами TTL/heartbeat. Каденс ≤minTTL/3,
пользовательский interval способен только уменьшить его. Сетевой deadline
отдельно ограничен minTTL/3, чтобы маленький interval не превращался в
непрактичный микросекундный сетевой timeout. Фоновый pulse, calls и ownership
state сериализованы; длинная очередь calls дополнительно обслуживает уже
наступивший fresh heartbeat, предотвращая starvation таймера. Каждый tick
имеет криптографически случайный новый request_id. Describe/next проверяются
свежим heartbeat перед успешным возвратом, не принимая исторический replay
за доказательство живой аренды. Release/terminal response очищает текущую
задачу. Исторические replay next/release не меняют это состояние.

Первая ошибка сети/timeout/невалидного результата/authoritative lease/auth loss
необратимо включает quarantine до перезапуска. Никакого reacquire/new generation
или implicit describe нет. Get/events продолжают текущие серверные чтения без
восстановления владения; остальные вызовы блокируются локально. Stderr содержит
постоянное сообщение LEASE_UNCERTAIN с требованием прекратить локальные записи;
MCP logging посылается клиентам, включившим logging level по SDK-контракту.
Bridge не способен физически остановить отдельные shell/file tools агента.
EOF/cancel останавливает pulse без раннего release: прежняя аренда истекает
естественным образом. Журнал успешных внешних request IDs ограничен32768;
достижение предела также требует безопасного перезапуска.

Общая client policy переиспользована CLI: только HTTPS remote/literal-loopback
HTTP, origin-bound bearer, отказ от redirect, private regular tokenfile без
symlink и с ограничением размера; stdout не получает token/diagnostics.
Все response body, включая chunked JSON, ограничены1MiB: overflow возвращает
ошибку, а не успешный обрезанный JSON. SDK localhost/DNS-rebinding защита не
отключалась. CLI `--help`/`-h`/`help` и dispatch перечисляют mcp/backup/token
list/revoke-all/sessions list/maintenance reset.

## Проверки разработчика

На этом Mac, Go1.27.2 darwin/arm64, локальные isolated fixtures без реальных
данных/токенов:

- `go test -race -timeout 60s ./internal/bridge ./internal/client
  ./cmd/agent-board` — PASS, соответственно12.931s/1.248s/1.558s.
- 10 integration tests bridge: >2TTL с живым bridge; natural expiry после EOF
  и следующее generation; replay next/release/describe; canonical session;
  timeout quarantine и запрет mutations/reacquire; current read после fault;
  revoke; authoritative lease loss; concurrent release/tick/read; fractional
  TTL и уменьшение interval; stdout только JSONRPC без credentials.
- 3 shared client tests: origin/redirect/off-origin isolation; private-file/env,
  symlink/public/oversized token rejection; chunked overflow rejection.
- CLI help и invalid flag stdout tests плюс прежние CLI tests — PASS.
- Ранний run обнаружил зависание только synthetic HTTP fault fixture при
  httptest shutdown; fixture теперь ждёт не дольше500ms. Счётчик сетевых
  инструментов исключает SDK cancellation notifications. Остановлен только
  принадлежащий этому WORK тестовый PID, чужие процессы не затронуты.

- `go vet ./internal/bridge ./internal/client ./cmd/agent-board` — PASS.
- `go test ./internal/store` — PASS,1.136s с additive TTL и параллельным admin delta.

## Первичные API-источники

Проверены закреплённые исходники официального
[Go SDK v1.8.0](https://github.com/modelcontextprotocol/go-sdk/tree/v1.8.0/mcp):
client.Connect/Tools/CallTool, StreamableClientTransport (MaxRetries,
DisableStandaloneSSE, detached connect context), IOTransport/Server.Run,
Server.AddTool и ServerSession.Log. Зависимости не обновлялись. Файлы <300 строк.

## Границы результата

Реальный удалённый Codex/Claude, процессный kill/restart и VPS deployment здесь
NOT-RUN: это последующая независимая проверка и эксплуатация TASK-003. Два
логических клиента на одном Mac не названы двумя физическими ПК. GitHub OAuth,
scopes/RBAC и остальные срезы V1 не приписываются bridge.

Manifest: 18 source/test files, SHA256 `dce36125989552836946238e9d3dc4c2e5ee35db400cee6a189a8c0df0ecc79a`; raw manifest `.local/evidence/task-003/WORK-021.sha256` (ignored).
