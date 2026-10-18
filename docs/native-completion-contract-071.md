# Native completion — контракт WORK-071

18.10.2026. Родитель: [TASK-003](TASK-003-working-service.md). Техническая
декомпозиция существующих §11.3/11.6/15 исходного ТЗ и V1-08; не новый продуктовый
дизайн. Статус: документационный freeze, реализация и независимое review впереди.
Контракт E [WORK-040](integration-contract.md) сохраняется без переписывания.

## 1. Версии и смысл статусов

Все пути ниже относительно соответствующего checkout, не установленного VPS.
**M** = основной `/Users/shagi/Desktop/Project/agent-board`, HEAD
`5a5af88348966eed7e4737948a81611593ba0fe4`, production source C/schema2.
**P** = `.local/performance-fixes-003`, production source
`96f3bfd191c65f8ad8edf622777ef161a5c5c10e`, D+performance/schema4+066.
Во время чтения P HEAD стал `73554ca8d1fac543a2282dcd14a923eb29976aec`:
добавлены только два документа069; source не изменён. Финальный APPROVE069
относится к96f3bfd, не к runtime. **E** = `.local/integration-config-003`, HEAD
`d1d3fb87d4c6d55895a9b0701dd6ae1112121e75`: pure libraries/config, не runtime.
VPS по TASK-003: `1e9025c`, schema2; этот аудит не подключался к серверу.
Снимок входных хешей: M `.local/work071/inputs.json`.

`implemented` означает найденную реализацию в названном source; `source-only` —
она не подтверждена здесь на установленной версии. `required` — незакрытое
требование, `deferred-by-user` — явное внешнее условие, `NOT-RUN` — проверки нет.
Это разные оси: source-only implementation не становится runtime PASS.

| Требование | Найдено / точные пути | Остаток / статус |
| --- | --- | --- |
| MCP ≤12 tools | M/P `internal/transport/operations.go`, `mcp.go`: фиксированные12 | implemented C; ресурсы/промпты не добавляют13-й tool |
| §11.3 notes | M/P `internal/transport/resources.go`; `internal/bridge/resources.go`: только `accord://notes{?...}` | implemented; board aliases required |
| rules, epic summary, report template, 2 prompts | M/P те же файлы, `bridge/bridge.go`: не зарегистрированы | required, не полный §11.3 PASS |
| read не renew | P `store/catalog.go`, `transaction_snapshot.go`; bridge `resources.go` вызывает heartbeat при due | server implemented; read-driven bridge HB исправить по §2 |
| §11.6 task aliases/events follow/stats | M/P `cmd/agent-board/{main,client,client_reads}.go`: `call OP` JSON stdin; stats уже через call | aliases/follow required; stats data не отсутствует |
| config init/validate/apply | P `cmd/agent-board/config.go`, `store/config_history.go`, `internal/config/**` | implemented/source-only; local admin apply, не remote apply из старого ТЗ |
| E/import/export/notifications | E `internal/{interchange,notifications}/**`, `internal/integrations/github/**`, `config/integrations.go` | source-only pure; SQL/runtime/schema5/REST/CLI required по040 |
| project.repository | P/E `internal/config/types.go`, `agent-board-v1.json`; M `docs/integration-contract.md:53` | поле уже есть; не webhook allowlist; github.repository_allowlist отдельно в E |
| systemd/private TLS/backup/update/rollback | M `infra/accord/**`, `docs/deployment.md`, `runtime-server-003.md`; backup062/068 отдельно | implemented; будущая combined schema4/5 приёмка NOT-RUN |
| пять binary targets | M/P `infra/accord/build-release.sh`: только linux/amd64, SOURCE/SHA256SUMS | matrix archives required; прежний cross-build не runtime proof |
| container/Compose | M/P нет Dockerfile/Compose в tracked inventory | required; Linux host-network вариант §5 |
| JSON logs/info/debug | M/P `cmd/agent-board/main.go`: plain listening line; bridge SDK logs discarded | required; stdout stdio остаётся только MCP |
| metrics/native limits | M/P `transport/http_metrics.go`, `http_security.go`: bearer metrics, global100/s burst200 max64 | implemented; точные session/IP требования отдельно §6 |
| payload secret scan | P `config/types.go: Rules`, schema, store mutations: secret_scan отсутствует | required; D strict config не принимает это поле, §6 |
| EN+RU docs/template | M `README.md`, `docs/{api,client-setup,second-pc,deployment}.md`; P `config/davinci.yaml` | README EN уже есть; полные config/template-authoring/platform docs required |
| serve --config/server.yaml, --dev | P `cmd/agent-board/main.go`: db/listen flags и persisted project config | source §15.1 не полностью покрыт; технический выбор §7, без подмены workflow YAML |
| OAuth/browser identity/UI | `requirements.md` V1-01/07, TASK-003 | OAuth deferred-by-user (App нет); UI требования required, макеты/SSR решение открыто |
| второй физический ПК/клиенты/платформы | M `docs/second-pc.md`, TASK-003 | 2PC deferred-by-user; Claude/Windows/new matrix runtime NOT-RUN |
| performance/backup target acceptance | TASK-003, `performance-contract-053.md`, method055/059/061, backup062/065/068/070 | старый049 latency FAIL; новый combined/annual/VPS NOT-RUN |

## 2. MCP identity, каталог и URI

HTTP остаётся stateless: один shared `mcp.Server`, bearer principal только из
request context. Никакого last-describe/last-session на shared handler.
Каждое personalized resources/read и prompts/get передаёт явный `session_id`:
URI query для resource, строковый argument для prompt. Транспорт собирает Actor;
store внутри canonical `transact` повторно проверяет SQL credential/revocation/
expiry, `sessionFor`, `roleAuthorized` и текущую config под writer gate+SQL lock.
Подставленная чужая SID не меняет principal. Role в path не меняет роль Actor.

Один bridge владеет своей SID, подставляет её при отсутствии, явно чужую или
повторную SID отвергает; до describe — NO_SESSION. Поддержать все перечисленные
ресурсы и prompts allowlist, не произвольную upstream URI/имя. Catalog списки
ограничены (сейчас resources≤32; prompts≤2), копии DTO независимы от upstream.
При upstream без capability не рекламировать отсутствующее. Catalog статичен,
аутентифицирован, без персональных данных и не меняется после describe.

Canonical URI templates, MIME `application/json`:

| Template | Service projection |
| --- | --- |
| `board://rules{?session_id}` | RulesView своей текущей роли |
| `board://notes/role/{role}{?session_id,limit,cursor,include_deleted}` | NotesPage, scope_type=role, scope_id=role |
| `board://notes/epic/{key}{?session_id,limit,cursor,include_deleted}` | NotesPage, scope_type=epic, scope_id=key |
| `board://epic/{key}{?session_id,limit,cursor}` | EpicSummaryPage, key=E-N, одна страница открытых tasks |
| `board://templates/report/{role}{?session_id}` | ReportTemplate своей роли |
| `accord://notes{?session_id,scope_type,scope_id,limit,cursor,include_deleted}` | прежний URI/NotesPage, additive compatibility alias |

Сохраняются логические URI исходника; personalized reads требуют SID query.
Все5 board-форм публикуются `resources/templates/list`: даже rules имеет query
SID, поэтому не рекламировать bare `resources/list` item как готовый персональный
ответ. Bridge возвращает content.URI в точности как запросил локальный клиент,
не раскрывая injected SID. Каталог и фактические handlers тестируются вместе.
Прямой `resources/read board://rules` безSID → NO_SESSION, bridge inject допустим.

Отклонять userinfo, fragment, opaque URL, неизвестный host/path/query,
повторные query keys, пустую явную SID, лишние segment/encoded slash/traversal,
неканонический E-N, неизвестную роль, limit вне1..200 и курсор другого фильтра.
Path-role notes сохраняет нынешний ListNotes доступ к выбранному scope для
авторизованного Actor; новое ACL на заметки здесь не вводится. Report-role
совпадает с ролью сессии; ни URI, ни prompts не расширяют token scopes.
При ошибке service — существующий publicError/redaction, без SQL/path/URI/token
в raw error. На resources/prompts ошибки SDK RPC, не tool `isError`.

Чтение не вызывает Describe/Next/Create/Heartbeat/Transition, не создаёт
request_id/replay/claim и не продлевает session/task/role. **Допустимо canonical
expiry housekeeping** `transact`: lease_expired материализуется как в GET;
нового transaction(false)/виртуального lease-состояния не вводить (решение root).
В bridge убрать вызов heartbeat из resource read handler; существующий pulse
живёт отдельно, под тем же lock/quarantine протоколом. Read в quarantine может
только читать свою ещё действующую сессию, не снимает quarantine/не восстанавливает
владение; expiry/401 не маскируются. Новые prompt/resource calls не продлевают lease.

## 3. Service/store DTO и prompts

Предлагаемые additive typed методы в `transport.Service`, типы в новом
`internal/core/resources.go`: `ReadRules(ctx,Actor) (RulesView,error)`,
`ReadEpicSummary(ctx,Actor,EpicSummaryInput) (EpicSummaryPage,error)`,
`ReadReportTemplate(ctx,Actor,ReportTemplateInput) (ReportTemplate,error)`,
`ReadPromptContext(ctx,Actor,PromptInput) (PromptContext,error)`.
Notes используют существующий ListNotes; чистые page/projection helpers общие.
Никакой transport→config file read и вызова local-admin Config вместо Actor API.

`RulesView{config_version int64,config_sha256,project_name,role string,
allowed_tools []string,can_create []string,can_cancel,can_curate_notes bool,
singleton bool,independent_from,reopen []string,rules map[string]string,
leases LeaseHints,notes NoteLimits,evidence_required,recommended []string}`.
LeaseHints повторяет float64 TTL seconds и heartbeat interval из SessionView;
NoteLimits — текущие четыре int quota. FieldDescriptor ниже содержит name/type
string, required bool, placeholder json.RawMessage (только null/описание).
Условия human/area предоставлять лишь релевантной роли безопасным summary;
не raw YAML/config, owners/users/members других ролей, env names/values,
endpoint URLs или webhook secrets. Реальные rule values из active D, не cached
describe. Правила не интерпретируют тексты заметок как инструкции сервера.

`EpicSummaryInput{EpicKey string,Limit int,Cursor string}`;
`EpicSummaryPage{config_version int64,epic EpicSummary,counts map[string]int64,
open_tasks []TaskSummary,next_cursor string,high_water int64,has_more bool}`.
`EpicSummary` = key/title/description/state/owner/priority/version;
`TaskSummary` = key/title/role/state/priority/version/generation/blocked/warnings
и краткий lease{session_id,generation,expires_at}, без attempts/comments/evidence
и истории. Open = created|ready|in_progress; terminal учитываются в counts.
Counts и page строятся из одного snapshot; cursor привязан к epic+open filter и
numeric creation boundary, как существующий list. Это не frozen-state snapshot:
изменения состояний между страницами допустимы. Счётчики всех task, не claim
eligibility. Данные эпика ограничены нынешними field limits, ответ≤1MiB client cap.

`ReportTemplateInput{Role string}`;
`ReportTemplate{config_version,role,evidence_schema json.RawMessage,
required,recommended []string,outcome_field string,outcome_values []string,
fields []FieldDescriptor}`. Объединить D EvidenceOnDone и schema.required;
передать ограничения schema, unknown fields и outcome enum без выдуманных значений.
FieldDescriptor{name,type,required,placeholder}; placeholder только null/текстовое
описание, **не готовый валидный отчёт**. Не генерировать SHA, approved=true,
checks PASS, run_id или подпись проверки. Wire transition использует верхний
`outcome` как нынешний API; schema `verdict/result` — evidence requirement,
не новый wire dialect. Новый report_template config field не нужен (root принял).

Каждый новый store aggregate исполняет **одну** `transact` + readActor;
внутри только общие чистые helpers, без вложенных Store/Service calls и без
network. Полученные DTO полностью detached от COW snapshot (включая pointers,
maps/raw schema); внешний mutate не портит cache/replay. Без schema migration.

`start-as-role` arguments: session_id(required direct), role(optional, default
current), epic_key(optional E-N). Явный role совпадает с ролью Actor; prompt выдаёт
user TextContent с URI правил/своих notes/report, явными действиями next→работа→
transition, request_id/generation и stop-on-uncertain. Он сам ничего не вызывает.
`coordinate`: session_id и since_cursor(required decimal int64≥0), optional
epic_key. Доступен любому live Actor с current read authorization (включая custom
roles), показывает текущую роль. Текст направляет читать events/next_cursor,
а создание/связи/запуск исполнителей выполнять только при имеющихся полномочиях.
Сам prompt их не даёт и роль не меняет. Guidance координатору: не забирать
executor next и не передавать свою аренду; отдельные сессии у исполнителей.
Оба personalized prompts после явного describe; не открывать сессию при get.
Unknown arguments/пустые значения/неправильные role/key/cursor отвергаются;
клиентский текст не вставлять как system instructions. No sampling/agent launch.

Pinned official SDK1.8.0 уже в go.mod: `mcp.Server.AddResourceTemplate`,
`AddPrompt(*mcp.Prompt,mcp.PromptHandler)`; `GetPromptParams.Arguments` —
map[string]string, `GetPromptResult.Messages` — []*PromptMessage с TextContent.
SDK `server.go:getPrompt` сам не проверяет required/unknown arguments — это handler.
Bridge использует `ClientSession.ResourceTemplates/Prompts/GetPrompt/ReadResource`,
capabilities, те же10s deadlines; `MaxRetries:-1`/no SSE retry сохраняются.
API сверены по локальному официальному module cache M
`.local/toolchains/cache/mod/github.com/modelcontextprotocol/go-sdk@v1.8.0/mcp/`
(`server.go`, `protocol.go`, `client.go`); сеть/новая версия SDK не нужны.

## 4. Additive CLI

`call OP --server ORIGIN --token-file PRIVATE < one.json` сохраняется без изменения.
`task next|get|transition|release|comment|link|create [client flags]` — тонкий
alias того же call/DTO/route, JSON stdin остаётся единственным business input.
Без позиционного KEY/генерации request_id/автоdescribe/claim/retry; unexpected
args, неизвестные JSON fields и path/body ambiguity отвергаются как прежде.
`stats --from RFC3339 --to RFC3339 --group role|owner|epic --session-id SID`
использует текущий StatsInput/route; stdin не читается, лишние позиционные args
ошибка. Canonical group `owner`, не старое `participant`; существующий call stats
остаётся. `config init --profile davinci` уже пишет YAML в stdout; validate читает
ровно один file, apply локальный admin с CAS. Это не task JSON stdin команда.
Import/export CLI wiring строго040 и только после E runtime/schema5, не через
generic JSON `call` (точные NDJSON bytes, checksum, private no-overwrite output).

`events --since N --session-id SID [--limit 1..200] [--types CSV] [--task-key T-N|E-N]
[--follow] [client flags]`: stdin не читается. Без follow — одна EventsPage JSON;
follow — одна EventsPage JSON на строку (NDJSON, `events`+`next_cursor`+
`high_water`+`has_more` сохраняются даже для пустой filtered page). Не отдельные
голые Event: иначе нельзя сообщить продвижение пустого фильтра без потери cursor.
Сохранить page после успешной записи+flush, следующий запрос since=next_cursor,
никогда high_water. has_more→следующая bounded page; иначе cancellable timer
1s (технический default) перед новым poll. Новый high_water не становится cursor;
фильтры immutable весь запуск, int64 не проходят float64. Memory≤одна≤1MiB page.
Нельзя буферизовать весь журнал, busy loop, reconnect/reclaim или скрытый retry.
Сеть/429/401/invalid response/output failure→nonzero, redacted stderr и последний
успешно выведенный cursor; неопределённый mutation outcome не скрывается.
SIGINT/SIGTERM отменяет request/timer, закрывает body, не heartbeat/release.
Истёкшая сессия follow завершается, фонового renewal у CLI нет; длительный worker
пользуется bridge. Token/Origin/redirect/body bounds общие internal/client.
Windows: существующий private ACL file→AGENT_BOARD_TOKEN; POSIX mode проверки
`--token-file` не объявлять проверкой Windows ACL. Секретных argv/JSON/URL нет.

## 5. Упаковка и документация

Расширить release builder на linux/{amd64,arm64}, darwin/{amd64,arm64},
windows/amd64 с CGO_ENABLED=0, pinned Go1.27.2, trimpath, source SHA/version,
no absolute build paths; Unix tar.gz, Windows zip. Каждому архиву SOURCE,
SHA256SUMS и LICENSE; общий manifest target/toolchain/SHA/checksum.
Только чистый committed reviewed source; private `.local/releases/<SHA>`.
Static/no-CGO build проверить, runtime статус каждого target отдельный.
macOS signing/notarization/Windows signing не обещаются. Linux/amd64 текущий
installer/rollback контракт сохранить; cross-архивы не становятся Ubuntu units.
Публикация проверенной ветки разрешена; public release/tag/main merge не поручены.

Контейнер: минимальный nonroot image, read-only binary/rootfs, private writable
DB volume (dir0700, DB/WAL0600), safe stop, /readyz без токенов, no secret layers.
Приложение слушает literal127.0.0.1: обычный publish port не достигает container
loopback. Принятый root минимум — **Linux-only Compose network_mode: host**,
без ports, с host127.0.0.1 и уже существующим private host TLS proxy.
Не подменять это bind0.0.0.0/общедоступным listener. Docker Desktop macOS/Windows
runtime и portable networking не поддержаны этим примером; на текущем VPS
systemd остаётся основной проверенной поставкой, контейнер не переключает live.

README EN уже существует; дописать фактические команды/матрицу и отдельные
EN config reference+template authoring guide (все поля/режимы/границы), MCP
resources/prompts/CLI примеры, ключевые RU разделы, checkpoint template.
P `davinci.yaml` существует, но почти без комментариев: добавить комментарии,
не менять defaults/role evidence ради демонстрации. Client guide различает
direct HTTP explicit SID/heartbeat и один-session bridge. Не писать fullV1 READY.

## 6. Наблюдаемость, admission и секреты

JSON server logs через стандартный slog JSON handler: time/level/event,
source_sha, listen authority при старте; на запрос route-template/method/status/
duration/error_code. Не raw URL/query/headers/body/SID/token/principal/owner,
config/evidence/SQL error. info lifecycle/failure, debug технические агрегаты;
bounded strings и фиксированные labels. CLI data stdout и MCP protocol stdout
не логировать: bridge diagnostics только stderr, с прежней redaction.
Compatibility: `serve --log-format text|json`, default text до адаптации
существующих tests/оператора; JSON mode не содержит plain listening line.
Unit выбирает json явно, tests/acceptance startup readers поддерживают оба формата.
Поставка включает JSON mode явно после проверки обновлённых startup readers.
После migration defaultjson можно менять отдельным проверенным шагом, не ломая
`tests/acceptance/{process_test.go,proxy_process_test.go,catalog_runtime.py}`.

Metrics уже authenticated aggregate-only; Runtime Sys не RSS. Сохранять Origin/
Host policy, global100/s+burst200+max64 до auth, no attacker-bearer map.
Точное исходное §15.3:20 вызовов/с/session,5 failed logins/min/IP. Поздний
`security-contract.md:64–70` считает общий limiter минимально достаточным для
native этапа; REQ V1-08 сохраняет rate limits без отмены исходной детализации.
Поэтому current native policy допустима, full exact-policy PASS отсутствует.
До session/IP реализации нужен отдельный security контракт: verified live SID
ключ с проверенным tokenID, bounded buckets/expiry/eviction/fail-closed capacity;
invalid SID не создаёт неограниченные keys, global limiter остаётся первым.
Proxy backend видит loopback, arbitrary XFF не доказывает IP. Варианты решения:
trusted proxy peer identity с проверенным overwriting edge либо enforcement на
доверенном TLS edge; отсутствие реального clientIP нельзя выдавать за5/IP PASS.
Bearer failure и OAuth login failure различаются; OAuth App пока отсутствует.

§15.3 secret scan — предупреждать в **description/comment/note**, block при
`rules.secret_scan.mode:block`; тексты evidence/произвольные файлы не заявлены.
P strict schema вообще не имеет secret_scan. Минимальный отдельный security
срез: добавить существующую требуемую Rule `mode:off|warn|block`, обе поставляемые
schema/типы/defaults/config-init/reference; отсутствующее поле→off для старых
конфигов с явным migration notice, default шаблона согласовать с root.
Не произвольные regex из YAML: короткий versioned встроенный набор только
высокоуверенных token/private-key patterns, bounded linear scan уже ограниченных
UTF8 inputs; false positives/negatives документированы. Реальные secrets не нужны:
synthetic nonusable fixtures. block отклоняет до изменения поля/audit/idempotent
result; warn сохраняет только field+rule ID, **не match/fragment/сырой текст**
в warning/error/log. Не сканировать входящий Authorization или private tokenfile
как board content. Это не универсальная DLP/защита от всех секретов.

## 7. Очередь готовых работ и открытые решения

Названия ниже — предложенные bounded задания, не созданная локальная очередь.
Координатор назначает WORK/ветки/ownership и единолично редактирует общую TASK.
Взаимные файлы входят только в одного владельца; интеграция последовательно.

| Срез / готовность | Область одного автора | Зависимость и обязательная приёмка |
| --- | --- | --- |
| N1 read core/store — готов после root freeze | новые `core/resources.go`, `store/resources*.go`; extraction чистых catalog helpers у этого же автора | P069 APPROVE; one transaction/auth/currentconfig, no-renew/expiry, bounded pages/COW isolation unit+race, независимый code/security |
| N2 MCP adapters/bridge — после N1 typed freeze | `transport/resources*.go`, новые `transport/prompts*.go`, `transport/{mcp,operations}.go`; `bridge/{resources,bridge}.go`, новые prompts files | N1;12tools, URI/catalog SDK/directHTTP/bridge/mismatch/currentroles/quarantine/oldalias tests; независимый reviewer отдельный от автора |
| N3 CLI aliases/follow/stats — готов по контракту | новые `cmd/agent-board/{aliases,events,stats}*.go`, `main.go`,`client*.go`; `internal/client` при нужном extraction | main.go временно только N3; N2 его не пишет. N3 не E import/export. Cancellation/bounds/empty filtered cursor/no renew/no hidden retry tests+review |
| N4 release/container/docs — готов, live не входит | `infra/accord/build-release.sh`, новые Dockerfile/Compose, packaging guide, EN reference/templates comments/README/second-pc | immutable combined source после integration; параллельно N1–3 безопасно кроме финальных SHA. Build5 targets, archive checks, isolated Linux container/runtime, permission/health/stop/security review |
| N5 JSON logging — после N3 main freeze | `main.go`, новые `internal/observability/**`, transport log middleware и свои unit tests | route redaction/stdio compatibility+native host/origin regression; отдельный tester меняет startup readers **после approve** |
| N6 secret scan/admission decision — анализ готов | отдельный security WORK, затем назначенный config/schema+store scanner owner | root выбирает defaults+proxy identity; не параллельно E config/schema владельцу. Unit rollback/replay/redaction/bounded-key tests, independent security/code, потом process |
| N7 E runtime — существующий040, здесь не новый контракт | schema5/store/runtime/REST/CLI import/export под текущим E owner | combined schema4+E067/070 approvals; pure libs не proof; safe apply/outbox/webhook/import/export process после review |
| N8 combined QA/deploy/branch — после approvals | отдельные tester/DevOps WORK, только их tests/evidence+разрешённая поставка | D CAS056 повтор; new APIs/aliases/stdio; annual20/2000/200000 метод061;062/068/070 Linux backup drill; Mac/VPS/Tandem443 preservation; exact-SHA review и verified branch push |

Для N1–N5 новые process/E2E пишет отдельный tester после approve, затем review
новых тестов. Существующие unit/process regressions запускаются по затронутым
областям; final combined make check/race/build/OS evidence обязательны.
N1–N4 можно готовить на изолированных ветках с указанными непересечениями;
`operations.go` Service редактирует N2 после согласованного DTO N1.

Открыто технически: (а) default secret_scan шаблона warn либо off с явным opt-in;
(б) session/IP enforcement и trusted edge; (в) source --dev и server.yaml.
Для (в) минимально отдельный server-only parser/config, не workflow D YAML;
--dev только literal-loopback+private temporary DB+local explicit bootstrap,
без печати credentials, без изменения текущего persistent default. Альтернатива —
root явно переносит эти convenience entrypoints за native delivery, сохраняя
их required в полной V1. Здесь ни вариант, ни новое поле молча не реализованы.
OAuth/provider credentials, физический ПК и визуальное UI решение остаются у
пользователя; контракты native не назначают их готовыми и не блокируют независимые
N1–N5. Этот документ не разрешает public ingress, GitHub API calls или релиз main.
