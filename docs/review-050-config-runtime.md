# WORK-050 — independent code review D/config/schema3

**REQUEST CHANGES** на `932aa2a3017da0b4fc06032e5134d4ed8fdf2d5b`.
Автор проверяемого D — WORK-043; reviewer — отдельный агент reviewd050.
C approve не переносился на D. UI/E/deploy/live не входят в review.

## Версия и границы

- Source checkout: `/Users/shagi/Desktop/Project/agent-board/.local/config-runtime-003`, `codex/config-runtime`; diff `1e9025c..932aa2a` — 85 файлов.
- Manifest `.local/work043/source-manifest.json`: SHA256 файла `8de46f25fbb6908828ae7eba1061f545235c38d845045bdc26e22996fbe497e8`.
- Внутренний source aggregate manifest: `414a24ff535517f14d74e92195122b8775922d9a31386b5e2459a05079aff481`. Проверены все 85 записей: совпадение 85/85, drift нет. Этот aggregate относится к freeze до commit/merge; точный review HEAD указан отдельно.
- Source clean до/после проверок. Общая TASK, source/tests и Davinci не менялись. Запись только в этот отчёт, WORK-050 и игнорируемую `.local/evidence/review-d-050/`.
- Прочитаны AGENTS, профиль reviewer, общий процесс и parallel-work, requirements/service-contract/config-contract, configuration-runtime, WORK-043; применён действующий Tandem OFF override. Именованные модели Claude здесь не закреплены; у автора/reviewer наследуемая доступная модель.

## Блокирующая дельта

### REV050-01 — P1: изменение singleton оживляет уже потерянную сессию

Место: `internal/store/runtime_roles.go:146-147`; связанная проверка apply — `internal/store/config_history.go:113-116`.

Repro: сконфигурировать role TTL1s, session/task TTL1h; дать старым setup-сессиям истечь, открыть architect SID, подождать2s. Apply `architect.singleton=false` проходит, поскольку прежняя role lease истекла. Затем Heartbeat старого SID с новым request_id успешно продлевает session. Воспроизведено без вмешательства в DB, через публичные store методы и инъекционный test clock.

Причина: `sessionLive` проверяет role deadline/generation только когда **текущая** роль singleton. После смены flag историческое условие теряется. Это нарушает обязательное fencing и требование «после потери роли прежняя сессия не возобновляется».

Исправление: сохранить необратимую недействительность/тип и generation аренды самой сессии либо консервативно запрещать такую смену до конца оставшейся session TTL. Старый SID не должен оживать после singleton false, удаления/повторного добавления роли или смены поколения. Сохранить поддержку legacy singleton sessions без нового role_expires_at. Добавить regression с разными role/session TTL и apply между expiry и heartbeat.

Доказательство: `TestReview050SingletonFlagChangeResurrection`, `singleton-repro.log`, `repro-final.log`.

### REV050-02 — P1: single_editor_role обходится изменением areas после claim

Место: `internal/store/tasks.go:203-212`, обязательное правило находится только в `internal/store/claim_rules.go:24-31`.

Сценарий: настроить area `restricted` с single_editor_role=architect, выдать developer task на обычную area; developer успешно берёт её и затем Update с текущим generation заменяет areas на restricted. Update проверяет canonical paths и overlap, но не single editor. При overlap off/warn/отсутствии других leases ничто не запрещает теперь держать task с запрещённой областью и закончить её.

Исправление: общий предикат авторизации записи area для claim и изменения областей **у leased task**, проверяющий роль фактического holder; не запрещать координатору готовить unclaimed задачу для другой роли. Отказ не меняет areas, version, lease и audit. Regression: claim safe → update forbidden → reject, при любом overlap mode, включая parent/child и alias.

Это собственный code-tracing вывод. Собственный runtime repro этого пункта **NOT-RUN**: координатор сообщил, что Security051 независимо воспроизводит тот же путь; его результат/ID не подменяется здесь своим PASS.

### REV050-03 — P1: can_cancel=false не применяется к human/coordinator epic cancellation

Место: `internal/store/transitions.go:122-124,142-150`.

Repro: настроить human.can_cancel=false, создать epic токеном координатора, выдать другому токену explicit human scope, открыть human session и отменить чужой epic с reason. Transition успешно даёт cancelled. Другой токен нужен, чтобы исключить легальное право requester; именно такой сценарий воспроизведён.

Причина: human/coordinator проходят начальную special-case ветку, а case cancelled больше не вызывает `canCancel`. Для task этот предикат проверяется. Это оставляет права отмены эпиков вне применённой D конфигурации.

Исправление: в ветке отмены эпика безусловно проверять единый cancel grant/requester baseline. Legacy bootstrap human.can_cancel=true уже сохраняет старое право явно. Добавить отрицательные тесты для human/coordinator с false и чужого requester, положительные для явного grant и собственного requester.

Доказательство: `TestReview050HumanCancelPermission`, `repro-final.log`: can_cancel=false human cancelled foreign epic, state=cancelled.

### REV050-04 — P2: incomplete child evidence не мешает закрыть epic

Место: `internal/store/gates.go:128-138`, сравнить с `gateSuccessful:28-30` и implicit child gate в `gateBlocked:94-98`.

Repro: developer требует extra pr, evidence_required=warn. Завершить developer с built-in commit/run_summary без pr: evidence_complete=false (правильное поведение). Закрыть родительский epic: операция проходит, ответ имеет одновременно state=done и blocked=true. Reconcile использует тот же epicClosable и также оставляет его done.

Причина: epicClosable проверяет terminal state и `blocked(child)`, но последний не учитывает evidence_complete самого child. Implicit child→epic gate и explicit closure расходятся. Эпик публикует completion/event/first_done и попадает в статистику завершений при неудовлетворённом обязательном результате.

Исправление: согласовать closure/reconciliation с successful predicate каждого актуального неотменённого child; отдельно сохранить существующее правило игнорирования superseded/stale исторических verifier cycles. Regression для warn и off, explicit closure и reconcile, без перепроверки исторических completion новой schema.

Доказательство: `TestReview050IncompleteEvidenceEpic`, `repro-final.log`: state=done blocked=true.

### REV050-05 — P2: area_overlap:block останавливает всю автоматическую очередь на первом кандидате

Место: `internal/store/tasks.go:125-128,160-163`.

Repro: developer1 держит task в busy; в очереди developer task priority2 на busy и priority3 на free. Developer2 вызывает Next без task_key. Получает AREA_OVERLAP для первого кандидата; свободная задача не выбирается. Повторяет тот же результат с новым request_id, пока busy не освободится.

Причина: eligibleClaim не включает блокирующий overlap; проверка выполняется лишь после выбора candidates[0], без перебора следующих допустимых задач. Контракт инструмента — «claim one eligible task», а не отказ на первой временно недоступной.

Исправление: учитывать blocking overlap в чистом eligibility predicate либо последовательно искать подходящего кандидата до любых эффектов claim. Для explicit task_key сохранять конкретный AREA_OVERLAP; warn продолжает claim и выдаёт предупреждение. Добавить regression с двумя ready кандидатами и отдельной непересекающейся свободной областью.

Доказательство: `TestReview050AreaBlockedQueueProgress`, `repro-final.log`: AREA_OVERLAP вместо T-4/free.

### REV050-06 — P2: omitted areas и explicit [] имеют одинаковый idempotency hash

Место: `internal/core/types.go:150` (CreateInput.Areas omitempty), `internal/store/mutation.go:30-35`, `internal/store/tasks.go:61-63`.

Repro: epic areas=[src]; create request X с omitted areas наследует src. Повторить X с areas:[] (другая команда — намеренно без наследования). Получается успешный replay с areas=[src], вместо IDEMPOTENCY_CONFLICT. Оба DTO после json.Marshal(in) теряют поле areas из-за omitempty.

Исправление: canonical request hashing обязан сохранять присутствие поля, которое меняет семантику. Например, не опускать nil/empty при hashing либо ввести presence-aware input с последовательной настройкой транспортной schema; повтор той же explicit [] остаётся идемпотентным. Regression на REST/MCP или декодированных DTO с omitted/[] и одинаковым request_id.

Доказательство: `TestReview050ExplicitEmptyAreasConflict`, `areas-idempotency-repro.log`, `repro-final.log`: omitted=[src], explicit-empty=[], returned=[src].

## Что проверено и что выполнено хорошо

- Active immutable config загружается после SQLite writer lock для каждой domain/admin транзакции. Store.cfg больше не участвует в обычной authority. Snapshot байты копируются, Decode даёт независимые карты; общей изменяемой config map не найдено.
- CAS до semantic safety, audited no-op, stale hash/expected conflict, config_history+active+event rollback проверяются осмысленными интеграционными тестами. Read API не выдаёт raw_yaml и не создаёт write/admin endpoint.
- Dynamic roles/tools/scopes, explicit human scopes, category/members/complexity, note limits, reopen и grants просмотрены по цепочке CLI/transport/store/config. Found permission/fencing gaps перечислены выше.
- Raw evidence и typed getters сохраняют integer >2^53. Runtime validator утверждает типы, bounds и built-in floor; unknown/nested duplicate evidence не теряются молча. Historical custom verifier kind сохранён в Review.OutcomeField; current exact SHA/generation, author session и global latest assignment проверены целевыми тестами.
- Миграции1→3/2→3: prebackup, единый writer, atomic DDL/backfill/user_version; failure/newer-schema paths просмотрены и targeted migration/backup tests PASS. Новые schema3 tuples в Python restore/recovery проверены как минимальная дельта. Source inspection не заменяет crash/process drill.
- Draft/ready, inherited priority/areas, alias canonicalization, Unicode casefold/Dice, duplicate_of, handoff CAS, общий DAG и roleTTL→bridge safety horizon просмотрены. Именно обходные/комбинированные сценарии выявили перечисленные дефекты.
- Изменено79 Go-файлов: все ≤300 строк, исключений нет (`decomposition.json`). Существенных новых самостоятельных дубликатов business logic, требующих дополнительной блокирующей декомпозиции, не найдено. При устранении REV050-02/04/05 важно переиспользовать единый предикат, а не копировать проверки.
- Старые Evidence assertions в основном механически адаптированы к map/getters, schema2→3 ожидания перенесены осознанно. Новые unit/integration проверяют поведение; обнаруженные комбинированные сценарии ими пока не покрыты.

## Фактические проверки reviewer

Runtime: Go `go1.27.2 darwin/arm64`, абсолютный MAIN `.local/toolchains/go/bin/go`, GOTOOLCHAIN=local; build/module caches и GOTMPDIR — MAIN .local/toolchains.

| Проверка | Результат | Доказательство |
| --- | --- | --- |
| HEAD+85 manifest entries | PASS,85/85,clean | manifest-check.json |
| Targeted31 top-level developer tests в6 пакетах | PASS | targeted-tests.log |
| Пять private overlay regressions REV050-01/03/04/05/06 | FAIL на reviewed source; воспроизводят дефекты | repro-final.log, review050_test.go, overlay.json |
| Single editor update authorization | Source finding; reviewer runtime NOT-RUN, отдельная Security051 | REV050-02 |
| Diff whitespace | PASS `git diff --check 1e9025c..HEAD` | результат команды reviewer |
| Changed Go decomposition | PASS79 файлов,0>300 | decomposition.json |

Команда targeted tests (без overlay):

```sh
go test ./cmd/agent-board ./internal/config ./internal/core ./internal/store ./internal/bridge ./internal/transport -run 'Test(Config|DefaultsCanonical|RejectHostile|SemanticValidation|DistributableAssets|NoSourceValues|ToolBudget|RefRejects|EvidenceTyped|Schema2To3|LiveConfig|ScopesDefault|StartupExplicit|SecurityDesign|WarnExtra|HandoffCAS|RemovingInactive|ConfiguredHuman|DraftAreas|DuplicateCancellation|DynamicRole|RoleTTL|RoleLease|BackupOnline|MigrationSchema|MigrationBackup|MigrationDDL)' -count=1 -v
```

Private repro:

```sh
go test -overlay=/Users/shagi/Desktop/Project/agent-board/.local/evidence/review-d-050/overlay.json ./internal/store -run '^TestReview050' -count=1 -v
```

Полный cmd/internal, targeted race, vet/build и compile-only всех пакетов заявлены автором и связаны с source manifest; здесь повторён только указанной targeted набор. Собственный повтор полного набора/race/build **NOT-RUN**. Новый D process/E2E, реальные два процесса apply/reload, crash/old-binary/backup restore drill, VPS, второй физический ПК, provider/OAuth/UI **NOT-RUN**. Они не выдаются за PASS. Старые process fixtures имеют известный конфликт persisted TTL/startup/bootstrap/schema3, их runtime suite не запускался. Исправление fixtures не должно ослабить production checks.

## Отложенные P3 и передача

Отдельных P3, требующих дополнительной дельты или повторного review, нет. Не требуется механический переход на другие библиотеки, изменение UI/архитектуры или масштабная очистка.

Автору вернуть REV050-01…06 одним пакетом. Нужны regression unit/integration на каждый исправленный сценарий и новый точный frozen HEAD; затем независимая проверка дельты. Новые process/E2E создаёт отдельный tester после актуального code/security approve. Этот отчёт не разрешает live apply, deploy или выпуск и не закрывает TASK-003.
