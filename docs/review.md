# Независимое ревью TASK-001

Дата: 18.10.2026. Вердикт первой итерации: **CHANGES REQUESTED**.
Ревьюер — отдельный агент, не автор реализации. Авторские области WORK-003 и
WORK-004 прочитаны фактически, включая untracked source; пустой Git diff не
использовался как доказательство отсутствия изменений. Модель Claude-профиля
не закреплена в этой среде; проверяющий использует модель текущей среды.

## Проверенная версия

Ветка `codex/pilot`, начальный `origin/main` — `9537cb7`. Исходники ещё не
закоммичены. Снимок 37 файлов (все Go-файлы, go.mod/sum, Makefile, README,
AGENTS, требования/архитектура/API) зафиксирован ниже. SHA-256 текста manifest:
`6f40906bfca781fcc3aced55fc211541728f3eb42f2fa9563eb58cc593ad94de`.
Это идентичность первой проверенной версии, не approve последующих правок.

## Блокирующие замечания

### R1 — P1: новая зависимость оставляет эпик ложноположительно завершённым

Место: `internal/store/dependencies.go:124-128`.

Воспроизведение: завершить developer и его эпик; в другом открытом эпике
создать незавершённый analyst; связать analyst `blocks` завершённый developer.
Link успешно добавляет gate, `Get(developer).blocked=true`, но `Get(epic).state`
остаётся `done`. Закрытие эпика уже не соответствует актуальным gates
REQ-16/17. Отдельный probe также воспроизвёл уже существующую связь между эпиками:
после принятия consumer и закрытия обоих эпиков reopen source делает consumer
blocked, но его эпик остаётся done. Место второго пути —
`internal/store/transitions.go:73-74` (reopen только собственного эпика).

Требуемый результат: изменение blocking graph/валидности source не оставляет
эпик `done` при неудовлетворённом обязательном gate. Допустимо отвергнуть новую
blocking связь terminal consumer или атомарно переоткрыть затронутый эпик;
выбранную семантику закрепить в контракте и regression test. Для изменения
source после уже принятой связи проверить также зависимые эпики.

### R2 — P1: назначенная проверка другого эпика обходится при закрытии target

Места: `internal/store/tasks.go:46-59`,
`internal/store/transitions.go:102-110`.

Воспроизведение: завершить developer эпика A; создать reviewer эпика B с
`target_key` этого developer и SHA результата; не выполнять reviewer.
Закрытие A успешно, потому что transitionEpic ищет review obligations только
среди дочерних записей A, тогда как Create разрешает target из B.
Обязанность текущей положительной проверки по REQ-17 не обеспечена.

Требуемый результат: либо явно ограничить review/tester target тем же эпиком
и отклонять cross-epic входы транзакционно, либо учитывать все назначенные
обязательства каждого target независимо от epic review и переоткрывать ранее
завершённый target epic при назначении новой обязанности. Нужен regression.

## Синхронизация контракта перед approve

- ADR-002 обещает отдельные таблицы epics/leases/attempts/links и FK,
  но schema в `internal/store/store.go:73-83` хранит epics/lease/attempt/link
  внутри task JSON и не включает foreign_keys. WORK-003 честно описывает
  фактическую per-row JSON модель. Нужно устранить противоречие архитектуры с
  source, зафиксировать валидацию ссылок внутри транзакционного домена и
  ограничения схемы. Производственную нормализацию/оптимизацию для пилота
  саму по себе ревью не требует.
- REQ-12 разрешает unlink coordinator/requester consumer; source допускает
  также live holder (`dependencies.go:89`), а API допускает live-holder
  изменение связи. Синхронизировать выбранную матрицу полномочий.
- Уточнить продление аренды успешной mutation: Create продлевает session,
  но не task этой сессии. REQ-20/ADR-003 можно прочитать как продление всех
  живых аренд. Это не объявлено отдельным дефектом до однозначного контракта;
  обязательный heartbeat уже описан. Временный третий probe фиксирует такое
  прочтение, его FAIL не является третьим blocker.

## Фактические проверки

PASS: существующие targeted core unit/integration tests:

```sh
go test ./internal/store -run 'Test(ConcurrentClaim|LeaseExpiry|Singleton|LostResponses|RejectedMutation|EventsCursor|FailedReview|ApprovalBecomes)' -count=1 -v
```

8 верхнеуровневых сценариев, включая 5 отрицательных review/tester subcases:
SQLite connection race, TTL equality/expiry/restart/fencing, singleton и
heartbeat atomicity, no duplicate replay/events, rejected completion rollback,
filtered event cursor/concurrent append, negative и stale same-SHA gates.

PASS: 3 существующих targeted transport/CLI integration сценария:

```sh
go test ./internal/transport ./cmd/agent-board -run 'Test(SDKStatelessToolsShareRESTState|SDKLeaseWorkflowAndRevocation|CallDescribeCreateGetAndEvents)$' -count=1 -v
```

Official SDK initialization/list stability, REST/MCP shared SQLite state,
lease workflow/revocation и CLI describe/create/get/events проверены локально.

FAIL с воспроизводимыми ожиданиями инвариантов R1/R2: независимые временные
unit-probes через Go overlay в `.local/evidence/review/`. Production source
и авторские тесты не изменялись. Артефакты `probe_test.go`, `overlay.json`,
`probes-initial.txt`, `probe-source-invalidation.txt`, `core-subsets.txt`,
`transport-subsets.txt` локальные,
игнорируемые, не содержат токенов. Это не новые process/E2E тесты.

Все production/test Go-файлы <300 строк; нет обнаруженных необоснованных
дублей бизнес-правил между REST/MCP. Generic `any` используется для Go
serialization/type parameters, не для обхода типизации TypeScript. UI N/A.
API official SDK сверяется компилируемым локальным закреплённым модулем;
профильный security review выполняется отдельно и этим отчётом не заменён.

NOT-RUN: полный make check/race зачётного результата (владелец — координатор),
process/E2E (тестировщик только после актуального approve), реальные
Claude Code/Codex, две физические машины, Linux/Windows, remote TLS,
нагрузка/production acceptance. Успех SDK не объявляется успехом реальных
клиентов. Нагрузка per-row task/session loading — известная граница пилота.

## Следующий шаг

Координатор принял R1/R2: поддержать cross-epic semantics, глобальные
обязательства по developer target и reconciliation ранее done эпиков после
изменения gates. Не вводить полный запрет cross-epic. Отдельно принято
уточнить docs: Create продлевает session/singleton, task продлевается только
lease-bound mutation с generation или heartbeat. REQ-12 синхронизируется
с holder unlink. Это решения для следующей авторской версии, не её approve.

WORK-003/назначенный исполнитель и координатор исправляют R1/R2 и контракт.
После frozen версии — повторное независимое ревью затронутой дельты;
только актуальный APPROVE разрешает новый этап process/E2E WORK-006.
Push/PR/merge/выпуск не выполнялись.

## Manifest первой итерации

```text
fd022e37a8d2937d16923285dd125c7531800db7cff2c7e0035448f0e4cec4aa  AGENTS.md
73be413242bf4e1fcfc4f762efabdb165e72a28b0f60ab2bb0aa5db4c462d14a  Makefile
6ec24b3e92ccfa7dbce587d96f74fc79d0ebd08b683c9173d01ecc7b9d59b4dc  README.md
ae9702f6beea7a112b510b4bc6f52e6da52f42ba82bbbfdea6f5c6b8dafa63ca  cmd/agent-board/client.go
54fd07eb51bc1c2c669f6ab5a22dd9fdac80bfa0d100b6735d6b4eccc4c6eaa6  cmd/agent-board/client_test.go
f13e4a9d4668ee7559a254795d028ba50d26bc3c3c3552e7227fdd3c686e23cb  cmd/agent-board/main.go
5555bf7e57e26017072f5c967384d09e212a066c01fd604ba4c744928bf31026  cmd/agent-board/token.go
c1c641d84084bcff7c256a767ce73b30d0d2b4c693521f17c94466c82be3c315  docs/api.md
471a65b502670383e81af7f69596d275d2b892f8bfc80a2575f52ecc299997d0  docs/architecture.md
8f1c776b90225f57ca14a61812f12602cb70bbed60ebfc1ed1bea3e67fec96da  docs/requirements.md
f1ee024ee7e93d6c54dabaf7a60e523c733a5e9a4855ed4b10ad7f9541fcc6c0  go.mod
83418331dd831bcf5abbd9646375e96e00bdfb8427450da8fc089c8ae6c41222  go.sum
8ffe8ba1ba6898ae6b3e2934014f04065e7590ee906de3b213d0f47d261baed6  internal/core/types.go
a8ee5a490f93d9219a0f3b304a1b2de04c94e41a00e16313be4ff7c30f8b8b86  internal/store/attempts.go
31e991211961509490208f2abbe4a20a46631df3ebdf1fa491d44f9e546859a3  internal/store/auth.go
1339af28bdb7ee196aafc2435d8385c077303a8c424f6bebc6d25b3033b52147  internal/store/dependencies.go
c6811d2995f58376acdfec6da29de1d70fe3b6973f6acc7ea5753752e16fbc8d  internal/store/events.go
1af2c93771d4c2a22b4fe2a763a366b48800cd4628c728cb348d4ed6834ad45e  internal/store/events_test.go
0410e12d548766bfbfd34152184706c5fed81b7e26c91c4d926e4b086280121b  internal/store/evidence.go
cf4e09bd4089e9fa46906b336c3624c8e66d1878d39e445a62e80b0e1ce7713f  internal/store/helpers_test.go
673ca0177999f3f416607ba5b667cca0f50de77e01499f074a2f76b183a03b99  internal/store/idempotency_test.go
303811e0b51d5a7fec1d8d12c4ab7857daf71cb75034de5d9627adb24dce0f1b  internal/store/leases_test.go
ae4938eb08957469a5a37c2685144a547618f9a46e25c852947ef67c7e8b2554  internal/store/mutation.go
7f0a16ea505dc274bceb67775c016fb791307ca8f41e6ab2385405d0feb2f937  internal/store/persistence.go
3ce94f5dc415040248e044558b83ece5d42363c14c89187fb797175fbea644c2  internal/store/reviews_test.go
87c29d4493bafcd3fe70a5d91f7b845dc4a643bc93a091ec265d2cf98772d360  internal/store/sessions.go
97f4dc1ca3d25e42c124385e223534cfc1c5f97db3a39d30eb965053e502a173  internal/store/store.go
27582bd443bbc48278bd02489493451dda51d4516de17a3641fe107c1a6d8962  internal/store/tasks.go
76a098adbb31f43f73d4625dfd42bc73910d3f1b5af20c19140f5c53d6771006  internal/store/transitions.go
f1bf15239fd8bb6474b131e6852ef87459a28e97ea28115360379da518923ea0  internal/store/workflow_test.go
6a3689502c5df63fe1b397b37e35fea228ffeeecf868d2c6c92523a507bc6ba2  internal/transport/errors.go
c688b58349cf3ee03086cbd9aee07a2cc6c67599380feb7806c1dae13893dde1  internal/transport/http.go
380ce13112322172caeaf239d99e56eb24cafcce042f87fb3db3c171525f829b  internal/transport/http_test.go
d84b622842e31f7086ec16ab53168436b12f41e6e175695c706da2b4b79202b6  internal/transport/mcp.go
a00adb7a12e70a1743e5bd4ac4bcb5ac034167ec40e8459a61b050c0990dc10a  internal/transport/mcp_test.go
bbfe56b23c14e694829adb80e28cda5c4d6f7ea8e9d5719acc29b8bdbecf209c  internal/transport/operations.go
999c73691e8055d3fb94426b7bdac25c3686d6a7998d8b15a59cd59b2773bb93  internal/transport/workflow_test.go
```
