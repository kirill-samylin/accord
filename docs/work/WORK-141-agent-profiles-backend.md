# WORK-141 — backend постоянных профилей

20.10.2026. Автор TASK010; выделенная область `internal/**` кроме webui,
`cmd/**` и этот WORK. Координатор владеет контрактом/TASK/infra; frontend отдельный.
Основание: agent-profiles.md и архитектурный APPROVE WORK140. Git commit автор
не создавал. Исполнение providers, координация и запуск не добавлены.

## Реализация

- Project-local profiles и immutable revisions в board schema3; случайный ap ID,
  одинаковые имена допустимы. Проверяемые поля, caps128/1024, версии, архив/
  восстановление, audit ID/version/session без инструкций в events. Author IDs
  публичного DTO — server session IDs, credential IDs не добавляются.
- Мутации сохраняют profile, revision, event и replay в общей Store transaction.
  Новые ревизии сначала находятся в candidate board; rejected domain mutation
  не пишет SQL напрямую и не оставляет revision при существующем commit-expiry
  поведении transact. Read не renew leases. Profile mutation renew session как
  прежние мутации, но не удерживаемую task lease.
- Describe принимает explicit ID/version pair и strict role match. Уже открытая
  session хранит immutable snapshot, edit/archive не меняют role/leases. Старые
  sessions unbound. Exact replay проходит current auth/ACL; profile state/version
  проверяются только при создании новой session. HMAC epochs сохраняются.
- Native REST/CLI profile CRUD и reads. MCP остаётся12tools: только новые read
  resources и поля существующего describe. Stdio bridge зеркалирует ресурсы и
  подставляет только собственный SID. Duplicate JSON keys отвергаются до maps
  в native/MCP/CLI/bridge; unknown/required/null profile fields проверяются.
- Browser profile list/detail/history-version read, same Origin/CSRF/action,
  полная edit form и durable original body/session. RecoverActions включает
  profile operations. Definitive validation/conflict выдаёт новые capabilities,
  unavailable сохраняет исходную. Viewer/archived writes/replay блокирует ACL.
  Profile list и общий ControlSessions list не включают instructions; session
  detail сохраняет фактический snapshot. Неверные URL query не игнорируются.

## Schema и offline migration

Schema2→3 проверяет существующую полную схему до DDL, пишет private/fsynced
`board.db.migrations/schema2-64hex.db` и exact SHA256 sidecar, добавляет только
profile tables/user_version. Schema1 сохраняет прежний durable snapshot и
получает notes+profiles. Fresh создаёт3; уже3 с отсутствующей таблицей отклоняется
без repair. Published preflight принимает полные2/3 до web orphan recovery.

`projects migrate-storage --projects-data ROOT --db DEFAULT --web-data WEB`
требует existing activation marker, выполняет полный read-only preflight всех
published boards перед первой DDL, затем existing-only mode=rw миграцию каждого
active/archived board. Pending/quarantine не открывает, registry не пишет,
sessions/expiry/events/credentials не меняет, listeners не запускает. Выход:
`{"storage_migrated":true,"boards":N}`. При сбое частичной последовательности
только roll-forward; operator обязан держать maintenance lock/stop/guard/backup.
`Store.MigrateExisting` отвергает missing/empty/schema0 без bootstrap.

Offline restore/reset валидирует существующие board/web, мигрирует старую2
перед reset, сохраняет profiles/revisions3 и отзывает credentials/session как
прежде. Legacy OpenExisting admin поддерживает1/2/3: load profiles условный,
не заставляет old schema2 token commands читать несуществующие profile tables.
Capabilities: registry1/ACL1 плюс agent_profiles1/board_schema[3]. Новый guard,
full format2 backup/restore и updater принадлежат root.

## Авторские unit/integration

Добавлены Store regressions для same-name nonmerge, immutable revisions,
edit/archive/restore+exact replay/restart, pair/role/human mismatch, missing ID,
quota/invalid mutation atomicity, revoked create/describe replay, unchanged live
lease/generation/role, migration2 preservation и incomplete3 failclosed.

Project handler tests проверяют A/B isolation, viewer read/denied writes/replay,
revoke→regrant старый SID denied и fresh session, archived project reads/writes;
all-board migration сравнивает каждую строку девяти legacy board tables и
registry authority/session/action/audit до/после default+active+archived и retry;
missing/future/incomplete3 обнаруживаются до первой DDL; offline reset сохраняет
profile/revisions и мигрирует2. Browser handler tests: actual CRUD, validation
saved draft/new capability, committed native edit с потерянным browser result,
exact retry после expiry и recoverActions при continuation, historical read,
unknown form/query и credential-ID projection. Transport/bridge tests: strict
payloads, resources actual content, own SID boundary и неизменный12tool budget.

## Проверки и ограничения

- `go test ./cmd/... ./internal/...` — PASS на Go1.27.2 Darwin/arm64;
  финальный повтор после дополнительных тестов фиксируется ниже.
- `go vet ./...` — PASS.
- `git diff --check` для авторской области — PASS.
- Новые product/test files <300строк; расширены прежние contracts/serializers,
  без новых runtime dependencies.
- Полный `make test` выявил прежние process assertions schema2 в
  tests/acceptance/backup_process_test.go и migration_process_test.go. Их
  адаптация принадлежит отдельному tester после source APPROVE; исходники
  process тестов автором не менялись. Это FAIL/ещё не полный make check PASS.
- Независимый review, новый process/browser QA, Linux/VPS update/backup и
  physical second PC — NOT-RUN этим автором; не заявляются по unit evidence.
