# WORK-004 — REST, MCP, CLI

- Роль: разработчик; отдельный субагент текущего координатора.
- Статус: на проверке, 18.10.2026.
- Входы: AGENTS.md, TASK-001, requirements 1.1, architecture.md/api.md,
  профиль разработчика и общий процесс Давинчи (переданы путями).
- Область записи: cmd/agent-board, internal/transport, README.md, эта WORK;
  go.mod/go.sum переданы владельцем WORK-003 после добавления SDK.
- Tandem отключён по указанию пользователя. Remote не создавался/не пушился.
- Модели Claude профиля здесь не закреплены; независимый approve не выдавался.

## Реализовано

REST по canonical таблице api.md и 11 фиксированных MCP tools делегируют
одному Store API. Principal создаётся только Authenticate(bearer), session
приходит из X-Session REST или обязательного session_id MCP. Инициализация,
список MCP и replay проходят повторную проверку токена. Машину и роль проверяет
core. Бизнес JSON декодируется строго, неизвестные поля/хвостовые значения/null
отвергаются. Ограничение body 64 КиБ. Ошибки внутренних компонентов скрываются.

MCP — официальный github.com/modelcontextprotocol/go-sdk v1.8.0, stateless,
только protocol 2026-07-28, JSON responses. Сохранена SDK DNS rebinding защита,
снаружи http.CrossOriginProtection без allowlist обходов. GET/DELETE session
semantics не подменяют бизнес-сессии SQLite. Общие metadata по согласованию
с архитектором исключены: high_water находится только в EventsPage; replay
не создаёт новое server_time.

CLI: serve --db/--listen/--task-ttl/--session-ttl; token create с O_EXCL0600
файлом без секрета в stdout и revoke при ошибке записи; token revoke --id;
call всех 11 операций с JSON stdin, token file/env и явной страницей events.
Удалённые CLI URL требуют HTTPS, redirect запрещён. Сервер bind — только
literal loopback. /healthz и /readyz не создают сессию. Ctrl-C/SIGTERM вызывает
ограниченный graceful shutdown. READY проверяет SQLite PingContext.

README EN с кратким RU: Accord/MIT, запуск, приватные токены, JSON CLI,
генерации/heartbeat/потеря связи, ограничения MCP/двух машин и deferred V1.
Указанный remote принят от координатора после поручения пользователя;
переименования binary/module/каталога в эту работу не входят.

## Проверки

PASS: `go test ./internal/transport ./cmd/agent-board` на изолированном
Go 1.27.2 macOS arm64. На момент передачи оба пакета проходят (~0.9с/~0.4с).

- REST: реальная SQLite, describe/create/get/replay, строгий JSON, body cap,
  session/машина, неизвестный Principal, метод, query, cross-origin, loopback.
- MCP official SDK: initialize 2026-07-28, 11 схем с session_id после describe,
  стабильный список после describe, общее состояние REST/MCP, unknown fields.
- SDK lease workflow: create/get/next/update/comment/heartbeat/link/release/
  повторный next с новым generation/transition(done)/events; отозванный токен
  больше не может выполнить tools/list. Это in-process integration, не E2E.
- CLI unit/integration: private0600/O_EXCL output без секрета в stdout,
  запрет перезаписи, revoke, describe/create/get/events через HTTP и SQLite.
- `go mod tidy` выполнен; dependencies pinned/checksummed. Компилируемый
  production SDK API проверен по локальному коду официального модуля и
  https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/streamable.go.

При интеграции replay create выявил Scan string → json.RawMessage HTTP500;
передано WORK-003, исправлено владельцем, regression теперь PASS.
Первоначальный тест link epic→task исправлен: pilot links имеют task endpoints.

NOT-RUN в этой WORK: полный check/race/vet/build (владелец координатор),
process/E2E (тестировщик после approve), реальные Claude Code/Codex, две
физические машины, remote TLS, Linux/Windows, нагрузка/production. Это не
подтверждение совместимости реальных клиентов или готовности выпуска.

## Передача

Изменённые файлы: README.md; cmd/agent-board/{main,token,client}.go и
client_test.go; internal/transport/{operations,errors,http,mcp}.go,
{http,mcp,workflow}_test.go; go.mod/go.sum; эта WORK. Все исходники <300 строк.
Проверенная версия фиксируется SHA-256 файлов и общей версией координатора
перед независимым review. После передачи файлы заморожены до возврата review.
Следующий исполнитель — независимый reviewer, затем tester после approve.
