# WORK-069 — независимый code/security re-review performance

**FINAL APPROVE** исправленного performance source и новых regression tests на
`96f3bfd191c65f8ad8edf622777ef161a5c5c10e`. Оба P1 R063-01/02 закрыты;
новых блокирующих замечаний нет. Это итог code/security review всего текущего
performance-среза, включая неизменённые механизмы baseline, а не только
согласие с текстом WORK066. Process/runtime-приёмка, выпуск и полный V1 этим
вердиктом не подтверждаются.

## Версия и независимость

Checkout `/Users/shagi/Desktop/Project/agent-board/.local/performance-fixes-003`;
base `b9fdc155a1d0a02bf6000c3654e006a778be23cd`. Независимо проверены до и после
проб 51 source hash и 6 delivery hashes из `.local/work066/source-manifest.json`:
оба раза mismatch 0, HEAD неизменен. Source aggregate51
`f3037b91790333b2df6bc1bc6fbd4c36feccb1be14cdc874649c40ca00a01370`;
delivery aggregate6
`3c217eccf6fe302560ff91c643b468835cd7e7a103cf123c1f023425aa6eea0b`.
Определение aggregate — SHA256 compact sorted JSON path→SHA256 map.
Доказательства: `.local/work069/source-before.json`, `source-after.json`.

Проверяющий `/root/rereview069`, отдельный от автора066. Прочитаны AGENTS Accord,
процесс/parallel-work и профили reviewer/security Давинчи, performance053,
config-contract, runtime, WORK066, исходные final review063/security064 и их
probes. Named Claude bindings недоступны; автор и reviewer используют одну
наследуемую модель. Tandem OFF по прямому указанию пользователя. Дополнительные
агенты не запускались. Production source/tests, чужие WORK/TASK, Git state,
Давинчи и live не изменялись; свои unit probes подключены только ignored overlay.

Исходный b9 имеет исторические REQUEST CHANGES063/064, aggregate48
`42d32f7826481346d8fc16fede29bb04c674708560b36edbe8e517c61272c2d6`.
Docs-only HEAD старой копии не является source базой исправления. Текущий diff
от b9 — ровно 2 production Go, 2 test Go и 2 документа. Он не меняет migration4,
dependencies, transport, limiter, config/authority policy или resource targets.

## Закрытие R063-01 · P1

`internal/store/tasks.go:70,72` вызывает единый `reopenEpic`, чей актуальный lookup
и `editTask` находятся в `internal/store/transitions.go:114`. Parent из
`readTaskFor` используется только для чтения priority/areas/type/state. После
verifier reopen того же epic повторный lookup видит уже `open`: версия и
`epic_reopened` увеличиваются ровно один раз. Для разных parent/target оба epic
проходят собственный barrier. Dirty persistence сохраняет обе entity вместе
с child, audit, replay и projection; shared committed pointer не записывается.

Независимо повторены новые проверки `reopen_performance_fix_test.go:73,105,148`
и старые063 probes. Developer, verifier same/cross epic: cache и raw SQL имеют
open/version+1 и один reopen audit; actual Close/OpenExisting сохраняет результат,
Next выдаёт child после restart. Replay до/после restart сохраняет DTO и полный
SQL dump без session renewal, повторного audit/sequence/replay. Caller poisoning
ответа не меняет replay. Invalid areas и metadata после reopen не меняют RAM,
SQL, audit/projection, sequence и replay, включая restart; request_id доступен
для исправленного retry. При now==expires_at rejected Create коммитит только
предусмотренный unrelated lease_expired, без reopen parent или renewal session.
Это соответствует отдельной expiry/candidate границе `mutation.go`, не обходу
доменной валидации. **CLOSED**.

## Закрытие R063-02 · P1 и границы полномочий

`internal/store/handoff.go:42` возвращает Task через generic
`transaction(..., false, ...)`, `taskFor` делает entity COW. Полный result graph
проходит JSON detach в `transaction_snapshot.go:102` до saveBoard и Commit
(106/109). Ошибка копирования/SQL/commit возвращает zero result, откатывает и
инвалидирует поколение; fallible обработки результата после commit нет.
Режим false сохраняет admin no-expiry, общий writer gate/current SQL config и
проверки handoff (open epic, existing owner, CAS, live child lease) сохранены.

Авторские natural API и synthetic persisted-history fixtures проверены как
разные виды доказательства. Они покрывают timestamps, attempts/EndedAt/
TokensSpent, evidence map и RawMessage, Areas, Links, Comments, nested warning
areas, similarity, review и EvidenceComplete. Sequential poisoning проверяет
видимый Get и SQL до concurrent poison/read; race13 PASS. Synthetic history
не выдаётся за штатное создание epic attempts через публичный API. Lease graph
и live-child rejection дополнительно покрывают неизменные tests060 и HandoffCAS.
Audit INSERT fault возвращает ошибку с пустым Task и оставляет owner/version,
audit/projection и следующий Get неизменными; retry успешен, stale CAS без writes.

Собственный probe069 с настоящей API-created dependency Link подтверждает,
что изменение caller Links не меняет endpoint/condition в cache/SQL и будущую
успешность completed gate после выполнения source. Второй probe на expiry
равенстве подтверждает неизменность child lease/session и ровно один handoff
audit с local actor; admin не материализует и не продлевает expiry. Bearer API
не получает этот entrypoint: `transport/operations.go` содержит только
аутентифицированный Transition; owner handoff в `transitions.go:130` требует
coordinator. Local CLI `cmd/agent-board/epic.go:29` использует отдельный Store API.
Это integrity boundary для trusted local admin, а не новое remote право. **CLOSED**.

## Полнота соседних механизмов и качество тестов

Повторно прочитаны transaction_snapshot/snapshot_copy/mutation/persistence,
reopen/handoff/dependencies и соседние admin wrappers. CreateToken возвращает
собственные scalars; RevokeAllTokens и RecoverState — числа; SetTokenScopes —
SQL DTO с новым roles slice; ApplyConfig получает копирующий CanonicalJSON.
Config/history/local token/session lists читают собственные SQL DTO. Другого
захваченного board entity result вокруг generic struct{} boundary не найдено.
ReadTaskFor в Create/Link/Note не даёт write через shared entity: Link target
остаётся taskFor/editTask. Новых дублей и ненужных абстракций нет; общий reopen
переиспользован. Все 4 изменённых Go файла ≤248 строк. API/dependency обновлений
нет; существующие JSON/SQL/generic API не заменялись.

Новые regression tests проверяют наблюдаемое поведение API+SQL+restart, а не
calls к editTask/reopenEpic или внутренний dirty flag. Проверен helper полного
SQL dump: включает все business tables и стабильную сортировку, не только tasks.
Негативный overlay создан reviewer из точных git show b9 tasks.go/handoff.go;
bytes совпали с авторским baseline, остальные source/tests остаются final.
Не используется whole-board подмена или заранее запрограммированные failures.
На b9 5 новых tests + 3 исходных063 probes дают EXPECTED FAIL8 с конкретными
assertions: SQL/cache divergence, double audit, rejected cache pollution,
DTO alias, ошибочный result при audit fault. Дополнительный real-link probe069
даёт такой же alias FAIL. Незатронутый no-expiry probe069 остаётся PASS на b9,
что служит контролем сохранённого поведения. Ни compile failure, ни случайный
concurrent map panic не используются как доказательство negative control.
Новые тесты также **APPROVE**.

Сохранены исходные границы performance review: retained immediate writer и
context gate; data_version на том же connection generation; SQL token/scopes и
current canonical config до replay; error/commit ambiguity cache invalidation;
full outward DTO detach; dirty entity/event/replay/projection atomicity. Новый
reopen входит в affected epic/review/dependency closure и dirty index delta;
local handoff больше не обходит общий outward boundary. Read-only/no-op,
replay scope correction и typed BUSY/BUSY_RECOVERY startup-only retry056
не переписаны; business/Commit blind retry не добавлен. Schema4 exact decimal
stats, sec/ns bounds/fallback, indexed bounded events и migration rollback
не затронуты; их 8 unit tests дополнительно повторены. Нет основания повторять
полный annual diagnostic из-за этой дельты. Admission100rps/burst200/max64,
p95≤50ms, RSS≤128MiB/start≤1s и target1vCPU/512MiB контракт остаются прежними.

## Фактические проверки

Pinned Go1.27.2 darwin/arm64, GOTOOLCHAIN=local; shared toolchains build/mod/tmp.
Все exact commands, timestamps, exits, counts и SHA256 логов —
`.local/work069/results.json`; overlays/probes/source checks — там же.

| Проверка reviewer069 | Фактический результат |
| --- | --- |
| Affected units, исходные063/security064 probes и собственные2 | PASS48 top-level,39 subcases |
| Race final новых5 + old0633 + security0643 + own0692 | PASS13 top-level,11 subcases |
| Exact b9 negative overlay final регрессий + own2 | EXPECTED FAIL9, PASS1; исходный пакет8/8 FAIL |
| Existing events/projection/migration atomicity/rollback | PASS8 top-level,2 subcases |
| go vet ./internal/store | PASS |
| git diff --check | PASS |
| Final 51 source + 6 delivery hashes, HEAD | PASS, mismatch0 |

Все15 author066 log hashes независимо совпали с results.json. Его affected43
race и build PASS прочитаны, не приписываются reviewer; собственный whole-suite
race/build/annual200k **NOT-RUN**. Независимый baseline review063/064 сохранён
как evidence неизменённых областей, здесь дополнен exact delta review,
повтором affected authority/fault/COW/gates и bounded schema4 tests.

## Остаточные ограничения и передача

WORK049 target VPS latency **FAIL** и D process056 **FAIL** остаются OPEN до
отдельной process/runtime приёмки. Unit/race PASS этого не закрывает. Required
process QA, target VPS p95/RSS/start/migration под reviewed workload,
TLS/live deployment, physical2PC и реальные providers здесь **NOT-RUN**.
Новые product process/E2E не создавались; теперь после данного source approve
их отдельный tester может готовить в согласованных root границах. Никакого
полного V1/production/runtime PASS или разрешения выпуска этот review не даёт.

Изменены только этот report и собственная WORK069; evidence в ignored
`.local/work069`. Обязательной следующей implementation delta и отложенных
рекомендаций по этому fix нет. Root получает APPROVE exact source/tests и
открытые runtime ограничения; запись после финальной сверки остановлена.
