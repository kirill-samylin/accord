# WORK056 — независимая функциональная приёмка D

**FAIL: QA056-01 — intermittent SQLite busy вместо CAS conflict.** Полная
приёмка D не завершена. Ранее targeted PASS не снимает последующий FAIL полного
прогона. PerformanceC latency FAIL остаётся отдельным blocker architect053;
здесь нагрузка и показатели latency не проверялись.

## Версия и метод

Исполнитель qad056 — независимый tester, не автор043/052 или reviewer054.
HEAD `d5a3a12bcf5a87d854171288139788bff0e2ab9a`, ветка `codex/config-qa`,
копия `.local/config-qa-003`. Новые process/E2E написаны после подтверждённого
APPROVE code/security054. Именованные Claude модели недоступны; наследуемая
модель, как у автора. Tandem OFF. Davinci, общая TASK/main, Git commits/push,
production source и infra implementation не изменялись.

Среда: macOS darwin/arm64, Go1.27.2, настоящие дочерние ordinary CLI/server/stdio
процессы, loopback HTTP sockets, MCP SDK, SQLite WAL. Синтетические пользователи
и данные; внешних providers или действующих клиентов нет. `-race` инструментирует
Go harness/source tests; TestMain собирает обычный child binary без `-race`.
Никаких измеренных свойств race-child или ОС Linux/Windows этот запуск не даёт.

Private evidence: `.local/evidence/config-qa-056/`. Ordinary binary SHA256
`e544283622d6075bf323a5262531eabf68624200dfa829241b2bff23360c5dba`.
Исторический C1e9025c binary SHA256
`59c633e286eacfb5c713cfd540857aff2152508c0801c78ec6f58a5a4029e55c`.

## QA056-01 — воспроизведение и влияние

На native schema3 config_version2 два child CLI одновременно выполняют:

```text
agent-board config apply --db DB --file candidate-0.json --expected-version 2 --actor qa056
agent-board config apply --db DB --file candidate-1.json --expected-version 2 --actor qa056
```

Кандидаты отличаются только notes.max_per_role:3 и4; исходное значение2.
Ожидается один успешный apply/version3 и один `CONFIG_CONFLICT`, одна новая
revision и config_applied. Фактически один результат полного `make check`:

```text
agent-board: database is locked (261)
```

Код завершения ненулевой, нормативный CAS conflict отсутствует.
Evidence: `full-check.log`, TestConfigConcurrentApplyProcessAndAuditRollback,
`config_runtime_process_test.go`. Блокирует функциональную CAS приёмку;
исходники не исправлялись. Этот FAIL-run остановился до счётчика победителей;
его poststate не сохранён из t.TempDir. Двойной winner/partial write в FAIL-run
не доказаны и не заявляются.

По разрешению координатора выполнен **один** небольшой targeted повтор, без
цикла retry: PASS, один winner, active3/history3, config_applied delta1,
active hash равен winner hash. Это подтверждает непостоянный характер сбоя,
но не закрывает его. В `cas-single-repro/` сохранены оба raw stdout/stderr,
оба кандидата, exact CLI, poststate JSON и проверенный private snapshot+hash.
В этом fixture не создаются токены. Итоговый тест дополнен диагностикой:
проверяет winner/hash/history/audit и сохраняет их **до** exact-code отказа.
Guard CONFIG_CONFLICT сохранён. Новая дельта diagnostics требует review;
после source approve WORK057 CAS test обязателен к повтору.

## Кейсы и результаты

Каждый кейс создаёт private изолированную БД и собственные процессы; шаги
ниже — человекочитаемые сценарии, реализация повторения — указанный test.
Отдельный протокол: `acceptance-final.jsonl` (33 top-level PASS), затем
`full-check.log` (общий FAIL). Идентификатор D056-02 остаётся FAIL в приёмке.

| ID / test suffix после TestConfig | Шаги и нормативный результат | Статус |
| --- | --- | --- |
| D056-01 OfflineCASAuditAndLiveReloadProcess | CLI validate/apply, same-hash no-op и stale expected; только два успешных audit. Живой сервер: notes quota2→3 другим CLI, новая note без смены PID. TTL candidate возвращает CONFIG_IN_USE с session/task keys; task bytes/lease неизменны и heartbeat работает. Bearer canonical config/history, limit1, no raw YAML; без bearer401, admin POST отсутствует. | PASS |
| D056-02 ConcurrentApplyProcessAndAuditRollback | Два subprocess apply с одним expected: один winner/CONFIG_CONFLICT; отдельно trigger abort config_applied и invalid candidate не меняют active/history/audit. | **FAIL QA056-01**, отдельные rollback ветки targeted PASS |
| D056-03 StartupBootstrapAndPersistedInputsProcess | Пустая DB без bootstrap → CONFIG_BOOTSTRAP_REQUIRED. Явный project/TTL conflict → CONFIG_CONFLICT. Неявные defaults/start/restart сохраняют project/TTLs/version. | PASS |
| D056-04 ScopesGrantsAndHumanProcess | Новый default token только agent. Human mint только explicit+allowlist. scopes set CAS1→2, stale CONFIG_CONFLICT, removal live role CONFIG_IN_USE. Remote human/role/tool denied FORBIDDEN_ROLE; list остаётся12. Human high forbidden, normal claim+answer проходит. | PASS |
| D056-05 CustomRawEvidenceStdioProcess | Custom role из config, SDK→stdio→HTTP→store done с integer9007199254740993. Exact текст и SQLite совпадают. Wrong type/null/unknown evidence и unknown DTO → VALIDATION. | PASS |
| D056-06 LiveAreaQueueAndCancellationProcess | Leased Update alias secret и restricted/child → FORBIDDEN_ROLE, bytes task/lease/version/events неизменны. Explicit busy next→AREA_OVERLAP; auto next берёт free без изменения skipped. can_cancel=false foreign epic→FORBIDDEN_ROLE атомарно. | PASS |
| D056-07 IncompleteEvidenceCannotCloseEpicProcess | Warn/off допускают extra-field incomplete child done с evidence_complete=false; epic done→RULE_VIOLATION, epic/audit без изменений. | PASS оба mode |
| D056-08 OmittedEmptyAreasDTORESTAndMCPProcess | Typed DTO marshal→REST и SDK→stdio MCP: omitted наследует epic areas, [] остаётся пустым; exact replay идентичен; противоположный payload с тем же request_id→IDEMPOTENCY_CONFLICT. | PASS |
| D056-09 LostSingletonSIDNeverRevivesProcess | RoleTTL1s, task/session1h; expiry materializes через отказ heartbeat, toggle singleton→false, cancel expired task отдельным coordinator, delete/readd role. Старые heartbeat/comment/create→LEASE_LOST; новая role generation+1. | PASS |
| D056-10 RulesLiveReloadAfterSessionExpiryProcess | Session1s естественно истекает, другой CLI применяет duplicate block; тот же server PID/new describe видит rules/config_version, одинаковая task→DUPLICATE. | PASS |
| D056-11 NativeSchema2MigrationOldBinaryRefusalProcess | Исторический C binary создаёт native schema2. D serve мигрирует2→3, проверяет prebackup integrity/hash/modes0700/0600 и точные семь legacy scopes. Restart сохраняет hash. Старый C binary возвращает точный generic open error, не слушает и не меняет DB bytes. | PASS |
| D056-12 Schema2MigrationDDLFailureAtomicProcess | В изолированной native schema2 создан конфликт config_active. D serve отвергает DDL: schema2/tokens/events сохранены; нет частичных config_history/scopes. Проверенный prebackup есть. | PASS |
| D056-13 InitialExplicitApplyYAMLAndHistoryProcess | CLI init davinci→OWNER replace→YAML validate→initial expected0 с явными bootstrap inputs; version1/12roles. Raw YAML/comment только private SQLite0600, HTTP history их не раскрывает. | PASS |
| D056-14 AcknowledgedWritesSurviveSIGKILLProcess | После подтверждённых writes реальный SIGKILL, затем same DB restart: config/hash/history/note и exact replay сохранены, note одна. | PASS |
| D056-15 StdioRenewsMinimumRoleTTLProcess | RoleTTL1s/task-session1h; idle stdio child удерживает singleton >2roleTTL. EOF завершает pulse, role истекает, old SID heartbeat→LEASE_LOST и часовая task lease материализуется expired. | PASS |

Прежние18 top-level process проверки C сохранены: claim/handoff/expiry,
replay/pagination/restart/revocation, stdio heartbeat/fault/quarantine,
TLS proxy/Host security, notes/cursor/resources, metrics int64 и backup/recovery.
Новый config/runtime набор не заменяет author unit/integration combinatorics.

## Миграция, backup и адаптация fixtures

Исправлены именно operational inputs. newClient перед mint сначала явно
bootstrap-ит целевые короткие TTL через serve+stop; token create больше не
успевает незаметно persist15m. Пустые serve fixtures получают bootstrap-owner.
Schema metadata backup/migration теперь ожидает3. SQL schema1 fixture сохранён
без изменений; никаких production config UPDATE через SQL ради обхода guards.

C-04 сохраняет exact task/event/replay bytes: **offline** native CLI
`config apply` мигрирует schema1→3 в compatibility config_version1, затем
expected0 возвращает CONFIG_CONFLICT без применённого кандидата и без saveBoard.
Проверяются exact historical bytes и verified prebackup. **Затем** запускается
serve с его немедленным Sweep и проверяются прежние state/evidence/no-invented-time,
legacy incomplete stats, notes и restart. Обычная сериализация Sweep добавляет
default fields/меняет порядок JSON; первый offline assert не означает отсутствия
любых последующих runtime writes. Старый точный guard не удалён и не ослаблен.

CLI live-WAL backup→byte-copy restore в private loopback fixture→offline reset
проверяет schema3 checksum0600, task/comments/events/attempts/notes/provenance,
config/hash/history и exact idempotency response. Reset отзывает оба токена,
старые SID не работают даже с новым token; fresh claim повышает generation.
Это локальный restore drill, не production Linux restore/deploy.

Python infra41 — synthetic filesystem/SQLite, service commands mocked;
добавлены schema2→3 failure guard и unchanged schema3 recovery. Они не являются
Linux systemd/VPS evidence.

## Фактические прогоны

| Прогон | Результат / предел |
| --- | --- |
| Targeted обычный acceptance-final | PASS33 top-level; JSON protocol, 15 D+18 C, до обнаружения flaky CAS |
| Единый полный make check | **FAIL** на CAS; fmt/vet и7 source Go packages PASS, acceptance32 PASS/1 FAIL. Make остановлен до infra/race/build |
| Первая попытка запуска полного make | LAUNCH FAIL до vet: запрошенный bin/go1.27.2 отсутствует; bin/go реально Go1.27.2. Путь исправлен, логи сохранены; это не второй зачётный PASS |
| Independent Python/build после FAIL | PASS41 Python и ordinary build |
| Independent race remainder | PASS174 top-level /8 packages; CAS исключён только из независимого остатка, его общая приёмка остаётся FAIL |
| Один разрешённый CAS targeted повтор + diagnostic delta | PASS с сохранёнными winner/version/hash/history/audit; QA056-01 остаётся OPEN |
| Source155 tracked files manifest и Git diff-check | PASS; cmd/internal/infra/go.mod/go.sum unchanged |

Точные команды/окружение/времена/exit codes/SHA логов — `full-check-command.json`,
`full-check-result.json`, `independent-results.json`, `results.json`.
Команды используют absolute MAIN toolchain и shared caches, GOTOOLCHAIN=local.
В копии собственных toolchains/dependencies не устанавливалось.

Для review достаточно воспроизвести выбранные2–3 теста через `go test
./tests/acceptance -run '^(TestConfigOmittedEmptyAreasDTORESTAndMCPProcess|TestConfigCustomRawEvidenceStdioProcess|TestConfigConcurrentApplyProcessAndAuditRollback)$'
-count=1 -v` с тем же окружением. Для native2 migration задаётся
ACCORD_LEGACY_BINARY; отсутствие этого binary честно даёт NOT-RUN/skip.
Это команда subset review, не просьба многократно гонять CAS до зелёного.

## Manifest и окончательная test delta

Source155 files: source-before/source-after совпадают; manifest SHA256
`d8bc768d9f9c4dee837106eacab78b1e1d5504b5df2efef1d117fc8c48da7782`.
Full-check test freeze27 files SHA256
`8122cad55d67aabee8689a3cca9f7cb47da0026e9baea4c9886000a8d808ca00`.
Окончательный test manifest28 files SHA256
`344114d04e20f86819df4ce837fd1c99d6552e2fa67eeebde2621a0faacbd4f2`.
После FAIL по разрешению координатора добавлены только diagnostics в
config_runtime_process_test.go и config_cas_evidence_test.go; именно эта дельта
пройдена одним targeted CAS повтором. Повторного полного make check на неё
нет: исходный blocker открыт. Остальные тесты не менялись; independent race
проверил исходный freeze с явным skip только этого CAS process test.
Все новые Go-файлы меньше300 строк. Полный SHA каждого evidence —
`evidence-manifest.json`; source/test сверка — `manifest-check.json`.

## Не проверялось и передача

**NOT-RUN:** VPS/live D apply/deploy, real Claude Code/Codex clients, второй
физический ПК, Windows/Linux runtime, UI, OAuth/providers, performance/load,
power-loss/OS crash, произвольный kill во время migration/commit/fsync.
SIGTERM tests — graceful restart; SIGKILL test — abrupt process death после ack.
Ни один из них не моделирует потерю питания. Atomic rollback проверен SQLite
abort и DDL conflict, без вывода о всех crash points файловой системы.

Следующие владельцы: root/WORK057 — source correction QA056-01;
независимый reviewer — новая test delta и source approve; tester после approve —
обязательный CAS retest. Эта работа не завершает TASK-003 и не снимает load FAIL.
