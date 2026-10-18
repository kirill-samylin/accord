# WORK072 — combined D/schema4 process acceptance

**PASS локальной D/schema4 функциональной приёмки; QA056-01 закрыт для этой
версии.** Новый набор ещё требует независимого review. Annual/VPS остаются
OPEN; deploy не разрешён. Проверен approved source069 на integrated checkout
`c1ef9ae2e8176064e1e652a9b1d480bf7b038943`. Это отдельная, привязанная к версии
проверка. Исторический full056 FAIL/busy261 и VPS latency049 FAIL сохранены;
никакого исторического отчёта или runtime результата072 не переписывает.

Tester `/root/perfqa072` независим от автора и code/security reviewer069.
Именованные Claude model bindings недоступны; inherited same model.
Tandem OFF explicit user; Git mutation/push/SSH/live/Davinci не выполнялись.
Source/cmd/internal/infra/schema/templates/build/dependencies readonly;
общую TASK меняет только root. Новые тесты требуют свежего review.

## Версия, окружение и fixture

macOS27.0.1/darwin-arm64 M1Pro; pinned Go1.27.2; настоящий ordinary Go child
и loopback HTTP/REST/stdio/MCP SDK. Все DB/session/token/task данные synthetic,
DB только собственных temporary directories, чужой сервис не используется.
TestMain строит child без `-race`, включая запуск из race harness. `make race`
проверяет Go source/harness; subprocesses остаются ordinary, race-child NOT-RUN.

Root source111 manifest file SHA256
`a6033126032d3e4a94d2c23ce9158dd5bd9115e195ef9ade3f130660e5a9c307`
проверен независимо (compact aggregate111 отдельно:
`96d3d544696524cd27d1bc0ed1d64763578d22156983ab2439bdd4c6a5c5e0fa`). Расширенный immutable source182 включает все tracked
cmd/internal/infra/schema/templates + Makefile/go.mod/go.sum, в том числе
авторские unit tests; aggregate
`75de79a54889210e0739111de35f660284385d684f0159f32b52c5607a19501b`.
Aggregate = SHA256 compact sorted JSON path→SHA256. Freeze/evidence `.local/work072/`.

Ordinary current child SHA256
`140be012c7d46adf762786f5a347a905554a4526ded79b083d6677145207b764`.
Исторический C2 binary SHA256
`59c633e286eacfb5c713cfd540857aff2152508c0801c78ec6f58a5a4029e55c`.
Native D3 pinned source d5a3a12bcf5a87d854171288139788bff0e2ab9a,
read-only build из `.local/config-fixes-003` в ignored output072, SHA256
`3075235e87ff260b41d7e44bec7cf5af927fc0ffacb018d553a3fc4a934373ac`.
Название executable не используется как доказательство schema/version:
fixtures проверяют фактический PRAGMA и stdout/error/поведение.

## Набор072v1 и normative сценарии

Прежние33 top-level C/D process cases056 сохранены с ожиданием schema4
через явную `expectedCurrentSchema=4`. Ожидание не читает store.SchemaVersion.
Exact legacy JSON/replay, roleTTL guards, семь historical scopes, lease
fencing, proxy/bridge/security/int64/notes assertions не ослаблены.

| ID | Шаги и нормативный результат | Targeted |
| --- | --- | --- |
| QA056-01/D056-02 | Два реальных CLI config apply same expected2, разные quotas. Ровно1 winner, loser CONFIG_CONFLICT, active3/history3, audit delta1/hash winner. Busy261 — FAIL. Отдельные audit INSERT failure/invalid-config rollback сохраняются. | PASS |
| C-04 | Canonical schema1 SQL fixture → offline stale apply0 → schema4/compat config1. Exact task/event/replay bytes проверяются до serve/Sweep, verified prebackup. Затем incomplete legacy provenance/no invented times/notes/restart. | PASS |
| D056-11/12 | Native исторический C binary создаёт schema2.2→4 backup hash/modes0700/0600, exact seven scopes/tokens, restart/hash. Старый C precise refusal+unchanged DB. DDL conflict rollback без partial config/history/scopes/audit. | PASS |
| Existing backup/reset | Реальный live WAL → CLI consistent schema4 snapshot/0600/checksum → byte copy private DB. Exact task/event/note/config/history/replay. Offline reset revokes2 tokens/SID, leases1, config/history/notes retained; fresh generation+1, historical attempts/audit. | PASS |
| P072-01 | Native D3 server+CLI создают task/note/config/scopes/replay. Offline stale apply0 мигрирует3→4 до Sweep. Exact8 table rows; built_cursor=max, dirty0/compatibility0,3 event indexes/3 invalidation triggers/stats backfill. Restart без SQL changes, verified schema3 prebackup, oldD3 precise refusal+unchanged DB bytes. | PASS |
| P072-02 | В nativeD3 offline создать conflicting stats_prefix. Current apply fails migration; complete logical SQL dump включая schema/tables до/после равен, user_version3, нет partial indexes/projection/groups/blocked tables, verified prebackup. | PASS |
| P072-03 | Real REST identical describe replay before/after natural1s session expiry не renew-ит session. Другой CLI removes token human scope / config removes human allowlist owner. Same live PID exact old request→403 FORBIDDEN_ROLE, complete SQL unchanged; то же после restart. | PASS обе restriction |
| P072-04 | Developer/verifier same epic/verifier cross epic: done parents; invalid areas/metadata Create→VALIDATION без SQL/cache/restart changes/request reservation. Corrected same request succeeds: parent open/version+1, ровно1 reopen audit/parent; exact replay+restart без session/audit writes; child claim доступен. | PASS все3 |
| InfraP3 | Deterministic close вокруг прежнего SQLite transaction context. Historical2→3/unchanged3 preserved;3→4 failure does not downgrade, unchanged4 allows safe previous executable recovery. | PASS19 Python cases |

P072 fixtures не подставляют schema3 вручную через PRAGMA для success path:
предыдущая схема создаётся историческим D3 executable. Единственная synthetic
DDL write — явно negative конфликт. Полный SQL dump сохраняет raw string cells,
не только текущие DTO. Runtime stable data проверяется отдельно от offline
migration byte preservation; обычный Sweep может нормативно сериализовать defaults.

Первоначальные red targeted протоколы сохранены:2 FAIL из-за nonmigrating
config get trigger;1 FAIL из-за нормативного no-op config_applied audit;
1 FAIL из-за неправильного exact C2 stderr для D3. Это исправления новых
fixtures против actual API contract. Final strict assertions сохраняют exact
rows/bytes и precise per-binary error; production source не менялся.

## Фактические проверки

**Единственный combined `make check`: PASS/exit0**, 2026-10-18
17:37:35–17:39:36UTC,121.056s. Final test freeze43 aggregate
`1313e0f5c212efe13aabe57af60a5c65bc2c359c7a0d351b1a02cd187d6312e2`.
После gate source182/test43 сверены без изменений. Команды/окружение/времена/
exit/SHA лога — `full-check-command.json`, `full-check-result.json`.
Лог SHA256 `5a8a33838dddaf598a11dbb22f13a3caf3ef5f3cc95641a697f2399ad932913f`.

| Gate | Фактический результат |
| --- | --- |
| fmt-check / vet ./... | PASS |
| full ordinary Go | PASS8 packages,204 top-level tests+FuzzParse seed suite;1 explicit developer annual diagnostic skip |
| ordinary acceptance | PASS37 top-level (33 прежних+4 новых), все native binary fixtures реально запущены |
| Python infra | PASS106 cases;3 ignored destructor ResourceWarning в полном логе — это не clean warnings run |
| full race Go source/harness | PASS8 packages,204 top-level tests+FuzzParse seed suite; тот же1 annual skip; data race не зарегистрирован |
| race harness acceptance | PASS37 top-level; ordinary child, не race-child |
| ordinary build | PASS, SHA256 current child совпал с указанным выше |
| strict CAS в full обычном и race harness | PASS оба: winner1/active3/history3/audit delta1, CONFIG_CONFLICT exact guard сохранён |
| source182 / tests43 / git diff --check | PASS, unchanged source и final test freeze |

Commandline Make variables явно задают absolute GO/GOFMT/GOCACHE/GOMODCACHE/
GOTMPDIR, поскольку Makefile переопределяет cache environment. GOTOOLCHAIN=local;
GOFLAGS=-v; ResourceWarning as error. Python выводит3 `Exception ignored while
finalizing database connection`, не превращая destructor warning в FAIL unittest.
Они появились после unchanged test_activation_atomic, в котором остаются3
`with sqlite3.connect` без close; traceback не указывает creation site,
поэтому это вероятный источник, а не доказанная трассировка каждого объекта.
Изменённый recovery module19 targeted clean. Соседний cleanup не выполнен
после freeze, нужен отдельный малый follow-up; full gate не повторялся.

CAS private targeted evidence: candidates/process stdout/stderr, poststate JSON,
verified snapshot metadata в `cas-targeted/`. Для combined gate diagnostic
snapshot env не задаётся: один и тот же test проходит ordinary и race harness,
путь single-snapshot не должен переиспользоваться/перезаписываться.
Strict test assertions и verbose poststate остаются включены в обоих прогонах.

## Annual, target и дальнейшая приёмка

По приоритизации root работа ограничена process/fixtures и одним full gate.
Большой annual process200000 events **NOT-RUN**; переносимая annual методика
**НЕ ПОСТАВЛЕНА/НЕ ПРИНЯТА**. Уже начатый tiny draft2sessions/2tasks/64events
окончен **FAIL** на неверном harness update route404; все4 модуля и raw evidence
убраны в ignored `annual-draft/`, не входят delivery и не запускаются повторно.
Seed/migration/waves tiny не выдаются за целевую нагрузку или готовую методику.
Следующая scoped WORK должна исправить/проверить/review методику и отдельно
измерить20sessions/2000tasks/200000events на target1CPU/512MiB с actual400 samples,
p95≤50ms/RSS≤128MiB/schema4 restart≤1s, migration timed separately.
Старый049 unpaced baseline FAIL остаётся; approved055v2 method неизменён.

Отдельный backup_process*.py method068 reserved073 и в072 не запускался/не
менялся; его REQUEST CHANGES070 не закрывается этим отчётом. CLI backup/reset
Go process test здесь отдельный существующий drill, его schema4 ожидание обновлено.
E5/pureconfig067 отсутствует в source и не сливалась в QA.

**NOT-RUN:** Linux/Windows/VPS runtime/resource target, D4 deployment/live apply,
TLS, actual Claude Code/Codex clients/providers/OAuth/UI, второй физический ПК,
power-loss/OS crash, произвольный kill during commit/fsync/migration.
Approved Store diagnostic200k автора не является process acceptance072.
QA072 не закрывает TASK-003/V1 и не разрешает D4 deploy/release.

## Delivery и остановка

Изменены ровно12 файлов:

- `tests/acceptance/backup_process_test.go` — ожидание schema4 metadata, прежний drill.
- `tests/acceptance/config_migration_process_test.go` — native C2→4/restart/refusal expectations.
- `tests/acceptance/migration_process_test.go` — explicit expected4, прежний exact offline legacy guard.
- `tests/acceptance/process_test.go` — optional fixture executable для native D3; default child прежний.
- `tests/acceptance/performance_helpers_test.go` — complete logical SQL/raw-table checks/native D3/projection helpers.
- `tests/acceptance/performance_migration_process_test.go` — P072-01/02.
- `tests/acceptance/performance_authority_process_test.go` — P072-03 real REST current authority/replay.
- `tests/acceptance/performance_reopen_process_test.go` — P072-04 developer/verifier/rejected/restart.
- `tests/infra/test_recovery.py` — deterministic close и schema4 guards.
- `docs/work/WORK-056-config-qa.md` — только лишняя пустая строка EOF P3-061.
- `docs/work/WORK-072-performance-qa.md` — собственный журнал/ограничения/handoff.
- `docs/acceptance-072-performance.md` — этот протокол.

Новые Go-файлы≤133 строк, production dependencies/API/guards/admission
не менялись. Exact file hashes в ignored `delivery-manifest.json`;
source/test before+after и `manifest-check.json` фиксируют конечную версию.
Старые failed fixture/draft logs сохранены рядом с final PASS, а не удалены.
Запись после конечной сверки остановлена. Следующий владелец — root:
независимый reviewer для этой test delta, затем отдельная scoped annual
method/target работа по приоритету пользователя. Никаких новых задач/чатов,
Git mutation или deployment tester не выполнял.
