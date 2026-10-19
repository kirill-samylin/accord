# Control UI: минимальный контракт REQ-005

Дата: 2026-10-19. Архитектор `/root/control_architecture101`, база a6501d8.
Это предложение координатору для принятия в общем контракте, не runtime approve.
Работающий контур: board schema2 + web schema1; D4/schema4/E не включаются.

## 1. Подтверждённая база и границы

- `internal/store/store.go`, `sessions.go`, `catalog.go`: один `core.Config.Project`
  на Store; Sessions имеют роль, run_id, machine, owner, expiry и server session ID.
  Нет стабильного настраиваемого AgentProfile, provider/model, process/run lifecycle
  или точного last_seen. Существующий `RunID` — клиентский идентификатор участия,
  не доказательство выполнения OS-процесса.
- `internal/store/migrations.go`: SchemaVersion=2. Tokens, generations, tasks,
  idempotency и события принадлежат одной board DB. Epic — группировка задач,
  с workflow/gates; он не граница проекта или авторизации.
- `internal/web/storage.go`: web schema1 хранит browser sessions/actions/audit;
  design assets имеют отдельный metadata/blob lifecycle в том же приватном контуре.
- `internal/web/server.go`: browser cookie связан с TokenID; каждый запрос заново
  проверяет отзыв/expiry через AuthenticateID. BrowserSession содержит human
  board session. GET не возобновляет её; продолжение явно через protected POST.
- `internal/transport/http.go`, `http_security.go`: admission, exact Host, запрет
  любого Origin на native `/api`, `/mcp`, `/metrics`. Нельзя подключать браузерный
  клиент к этим native endpoints и снимать запрет ради dashboard.
- `store.Stats` доступен, но сканирует весь журнал, даже при узком периоде.
  `store.Events` — ASC с high_water; это не готовый запрос «последние 20 событий».
  Встроенные catalog reads используют transact и могут материализовать expiry;
  они не продлевают аренды/сессии, но конкурируют за SQLite write lock.

Reference Paperclip: `ui/src/pages/Dashboard.tsx`, `Agents.production.tsx`,
`Projects.tsx`, `packages/db/src/schema/{agents,projects,heartbeat_runs}.ts`.
Оттуда применимы shell/sidebar, обзор, агентные списки, task activity и доска.
Paperclip хранит отдельные agents/adapters/runs/budgets; копирование этих статусов
в Accord без источника данных дало бы ложную наблюдаемость. Полную инвентаризацию
возможностей ведёт аналитик WORK-100; здесь только технические границы.

## 2. ADR-CONTROL-01: сохранить Go SSR

Предложение: Go SSR + ограниченный JS для автообновления читаемых блоков,
фильтров/вида доски, меню и мобильной навигации. Один binary/process, embedded
assets, существующие login/forms/design upload/human answer/release продолжают
использовать действующие handlers, CSRF/action IDs/idempotency и fencing.

React допустим как последующий самостоятельный web frontend через тот же browser
BFF, но сейчас требует сборки/поставки, нового client routing и переноса error/
draft/retry/accessibility поведения. Объём запроса не требует этого переноса.
Визуальный язык Paperclip не зависит от React. Не копируем его backend/DB/router
и не подключаем Node runtime к live сервису. Нет новой зависимости для этого этапа.

SSR HTML — основной результат, пригодный без JS; polling улучшает актуальность.
`html/template` продолжает экранировать пользовательские данные, JS обновляет
текст через textContent; не создавать template.HTML из данных, inline script,
eval, CDN и вставку серверного текста через innerHTML. Официальные API проверены:
[html/template](https://pkg.go.dev/html/template), [net/http](https://pkg.go.dev/net/http).

## 3. Минимальные backend readers

Существующий `web.Board` дополнить `Sessions(ctx, actor, core.ListInput)` и
`Stats(ctx, actor, core.StatsInput)`. Их native контракты сохраняются.
Добавить два transport-independent read метода; naming ниже проектный.

| Метод | Вход | Выход/инварианты |
| --- | --- | --- |
| `ControlOverview` | actor | observed_at, project label, task counts по реальным state, blocked-ready, open epics, active agent sessions, active human sessions, leased tasks, latest event cursor. Полные текущие counts в одной store transaction; не сумма первой страницы. active = живой principal + session expiry > now; leased требует этого же principal, живой task lease и matching generation |
| `RecentEvents` | actor, before_cursor>=0, limit 1..100 (default20), optional task_key/session_id/type | immutable Event[], next_cursor, has_more, observed_at; DESC cursor keyset, LIMIT+1; before_cursor=0 означает начало с последних. Фильтры применяются SQL до LIMIT; каждое следующее чтение cursor<before. Не менять старый ASC Events |

Оба reader используют существующий readActor внутри transact; token/session
ownership и expiry проверяются без обхода. Не создают leases, Describe,
Heartbeat или новых telemetry events. No schema change: counts из текущих Tasks/
Sessions, latest/recent — из append-only events (включая materialized expiry
текущей транзакции: persistNewEvents перед SQL). Ни partial list, ни failed query
не превращаются в «0»/успех. Недоступное поле передаётся отсутствующим/null.

`RecentEvents` без фильтра нужен для homepage/activity; с task_key — карточка,
с session_id — журнал агента. Данные EventData показываются как зафиксированные
факты; пропущенная историческая provenance остаётся неизвестной. Не дополнять
старые события современным owner/title так, будто это event-time данные.

## 4. Browser routes и page models

Все новые browser data endpoints под `/web/data/…`, а не `/api/v1/…`.
GET/HEAD проверяет текущую browser cookie и живую board session; возвращает
no-store JSON с Content-Type application/json либо привычный SSR HTML.
Принятый существующий error code/status contract сохраняется; 401 требует вход,
session_expired — явное продолжение. Не возвращать в JSON HTML login с кодом200.
Данные PrincipalID/token metadata не нужны UI; DTO выбирает только необходимые
session_id, role, run_id, machine, owner, expiry. Никаких bearer/hash/CSRF в URL.

| SSR маршрут | Данные/поведение |
| --- | --- |
| `/web/` | «Обзор»: Overview + latest20 events; навигация к вопросам, задачам, агентам, макетам, истории. Существующие `?view=tasks/questions` продолжают работать; отдельные ссылки могут быть `/web/tasks` и `/web/questions` |
| `/web/agents` | Sessions с фильтрами role/owner/active-or-expired, cursor, limit<=100. Одна строка = одна сессия запуска; human показывается отдельно/явным фильтром. Переход на session activity. Нельзя сливать несколько run в одного вечного агента |
| `/web/agents/{sessionID}` | Session facts из авторизованного каталога и RecentEvents(sessionID); unknown/недоступная сессия —404. Текущая task связь берётся из актуальных задач с lease этой session, а не из сохранённого SessionView.MyTasks |
| `/web/tasks` | Существующий ListTasks/Get; list/board представления одних данных и фильтров. `created`, `ready`, `in_progress`, `done`, `cancelled` сохраняют свой смысл; blocked — признак, не выдуманный переход. Отображается пагинация/полнота колонок; нет drag-drop мутаций, обходящих core |
| `/web/projects` | Один реальный подключённый board из server/session Project. Project и epic раздельны. Без работающего NewProject/switcher; коротко обозначить управляемое подключение будущих проектов |
| `/web/settings` | Read-only текущий проект, роль human/browser session expiry, доступные реальные роли/rules/TTL из authoritative session/config. Без токенов, filesystem DB path, фиктивного выбора model/provider. Выход/продолжение — действующие forms |
| `/web/activity` | RecentEvents, DESC pagination и task/session фильтры |
| `/web/stats` | Существующий Stats по явно заданным from/to/group_by, отображение периода и incomplete_data/missing_tokens. Не обязательный polling источник homepage |

Read JSON: `/web/data/overview`, `/web/data/agents`, `/web/data/activity`;
параметры соответствуют readers, с полной validation неизвестных значений,
лимитов и cursor. SSR и JSON используют один page/query builder. Точный набор
маршрутов принимает root; таблица не требует реализации лишних пустых страниц.

AgentView статус: «Есть аренда задачи» только при живом principal и task/session
арендах; «Сессия активна» при живом principal и session expiry>observed_at без
найденной аренды; «Сессия истекла» — по server truth. Сессия с отозванным/истёкшим
credential не active, даже если её TTL ещё жив; label «Доступ недействителен».
Native Sessions(State=active) фильтрует только TTL, поэтому web использует новый
`ControlSessions` с batch principal validation внутри одной транзакции, без N+1
AuthenticateID и без token IDs/hashes в DTO. Для полного join текущих задач нужен один
store projection либо отдельный точный query, нельзя объявлять absence по первой
странице ListTasks. Если join не реализован, показывать только подтверждённый
session status. Execution state всегда «Нет данных о процессе» без adapter.
`last_recorded_event_at` можно показывать из последнего события; `last_seen_at`
не вычислять как expiry-TTL: мутации/heartbeat имеют разные пути, TTL мог меняться.

Не отображать расходы как0; token totals имеют missing share, wall duration —
календарная длительность, не вычислительное время. Нет fake salary/cost/model,
live logs/process running, pause/resume/start без execution adapter.

## 5. Freshness и ошибки

- Первичный SSR показывает fetched/observed time. Независимые Sessions/Stats
  запросы не выдаются за одну atomic snapshot с Overview; полные counts и их
  timestamp принадлежат Overview. У lists не обещать snapshot текущих mutable
  values на все страницы: high_water фиксирует границу создания, не MVCC историю.
- Опциональный polling только читаемых блоков видимой страницы, default15s;
  максимум один in-flight запрос, AbortController при уходе, backoff на429/503,
  остановка при hidden/offline/401/session_expired. На возвращении один fresh
  request. Не обновлять task forms/DOM с несохранённым ответом.
- GET polling не вызывает продолжение board session и не запускает heartbeat.
  Явный heartbeat human answer editor сохраняет прежнюю semantics.
- Ошибка обновления сохраняет прошлые данные с временем и отметкой «Не обновлено»;
  незавершённый ответ нельзя показывать как пустую очередь. SSR reload доступен.
- Полный Stats не вызывается каждым tick; overview без сканирования всего journal.
  Нагрузочный риск: catalog transact загружает Tasks/Sessions и берёт write lock.
  Не добавлять отдельный poller на каждую карточку. Доказательств новых нагрузочных
  целей нет; необходимые замеры проводятся до выпуска, без утверждения V1 capacity.

## 6. ADR-CONTROL-02: будущие проекты и агенты

Сейчас один board/process; web/board schemas2/1 сохраняются. DTO/shell могут иметь
server-generated opaque project ID и display label; текущий Project string label
не является tenant ACL. Нельзя менять только заголовок или фильтр epic и объявить
изоляцию. Не принимать browser endpoint URL, DB path, MCP URL или общий token.

Наименьшая будущая безопасная поставка: независимый board + DB/webDB/blob root,
project-scoped credentials и отдельный HTTPS origin на каждый проект. Оператор
регистрирует ID/label/canonical origin в контролируемом manifest; UI показывает
разрешённые navigation links без server-side probe/fetch и без переноса токенов
через query/localStorage. Каждый проект требует собственный вход; __Host cookies
не разделяются между hosts. Валидация HTTPS/exact host/отсутствие userinfo и
неожиданного redirect нужна даже для operator-managed links.

Если потребуется единая общая панель/один вход: отдельный allowlisted registry и
ACL {human principal, project ID, capabilities}, project resolution до Store/auth;
выбор браузера только по ID из разрешённого списка. Server owns endpoint +
credential secretRef, per-project board credentials/session/actions/cursors/cache/
assets namespace. Authorize project на каждом read/mutation/download/replay,
включая объекты с совпадающими task keys/session IDs. Кэш/refetch всегда с project
ID; смена проекта отменяет предыдущие запросы и очищает client state. Общий human
token для всех board недопустим. Server-side remote adapter должен явно разрешать
destination network/ports, запрещать произвольные redirects и DNS rebinding,
metadata/loopback/link-local destinations; private destinations — только явно
назначенные оператором. Не строить универсальный URL proxy из browser input.

Multi-tenant одна DB возможна лишь отдельной миграцией, добавляющей project IDs
во все keys/FKs/idempotency/tokens/events/generations/design metadata, а также
negative isolation tests. Сейчас это более дорогой и рискованный вариант, чем
отдельные board. Ни manifest federation, ни shared-login ACL не реализуются в005.

Agent profile и execution adapter — разные ресурсы. Будущий профиль задаёт
имя/роль/provider/model/permissions/secret refs; выполнение отдельно подтверждает
queued/running/exited, run ID, observed timestamp, result/log provenance и cost
completeness. Server-side allowlisted adapters, запрет browser arbitrary command,
рабочей директории/секретов, independent execution authority. Сохранение настроек
не означает запуск. Не включать неподтверждённые D4/E config runtime ради формы.

## 7. Разделение реализации и AC

| Владелец | Файлы/область |
| --- | --- |
| Backend author | новые `internal/core/control.go`, `internal/store/control_overview.go`, `control_activity.go`, targeted store tests; новые `internal/web/control_pages.go`, `control_data.go`, `control_queries.go`, targeted web tests |
| UI author | `internal/webui/templates/*.html`, `assets/app.css`, `assets/control.js`, render helpers/tests, reference-derived shell/pages; существующий app.js изменять лишь согласованно с владельцем forms |
| Root integrator | общие `internal/web/contracts.go`, `http.go`, `pages.go`, `internal/webui/render.go`/Static allowlist и transport interface wiring, общий REQ/контракт; их не редактируют параллельные authors |
| Reviewer/security/design/QA | собственные WORK/evidence; source fixes возвращаются автору; новые process/browser E2E после code approve |

Перед UI author root публикует точные Page DTO/template names/data attributes;
backend и UI можно делать параллельно на этом контракте. Один владелец конкретного
файла; split по ~300 строк без общего control monolith. Migration не нужна;
backup board/web/design перед выпуском и rollback binary проверяются отдельно.

AC/проверки для реализации:
1. Overview counts точны при >page limit задач/сессий, empty board, blocked gates,
   human/nonhuman, expiry, cancelled/done и generation; latest activity DESC,
   no skip/duplicate при пагинации и новых вставках; фильтры/невалидные cursors.
2. Frozen clock: повторные read/poll не меняют ExpiresAt/generation/lease;
   expiry обрабатывается core. Stale browser session не оживает автоматически.
3. Состояние agents не опирается на SessionView.MyTasks, неполную страницу tasks
   или вычисленный last_seen. Два runs одной роли видны отдельно. Token/provenance
   unknown не превращаются в0/подтверждённый факт.
4. Login, logout, current token revocation, native Origin rejection, Host/CSRF,
   fenced answer/release/retry и design upload/download прежние regression PASS.
   XSS labels/events безопасно экранированы; JSON/HTML errors не теряют draft.
5. Browser: desktop/mobile shell, keyboard, list/board/filter/pagination,
   hidden/offline/slow/401/429 polling, редактирование ответа при обновлении,
   реальная активность двух sessions. Independent design review по Paperclip.
6. `go test` targeted packages + полный принятый suite один раз при приёмке;
   build и live authenticated smoke после отдельного deployment stage. Отдельно
   фиксировать source/unit/process/browser/live evidence; этот документ их не доказывает.

## 8. Author DTO freeze (уточнение после сверки с root)

Эта секция уточняет таблицы выше и имеет приоритет для реализации. Existing
Sessions остаётся native reader без изменения semantics; effective web вывод
даёт ControlSessions. Названия JSON snake_case. Core readers:

```go
ControlOverview(context.Context, core.Actor) (core.ControlOverview, error)
ControlSessions(context.Context, core.Actor, core.ControlSessionsInput) (core.ControlSessionsPage, error)
ControlTasks(context.Context, core.Actor, core.ControlTasksInput) (core.TasksPage, error)
RecentEvents(context.Context, core.Actor, core.RecentEventsInput) (core.RecentEventsPage, error)
// Existing Stats(context.Context, core.Actor, core.StatsInput) (core.Stats,error).
```

Минимальные поля DTO (точные Go поля для UI; стандартные json tags обязательны):

```go
type ControlOverview struct {
  ObservedAt time.Time
  Project string
  TasksTotal, TasksCreated, TasksReady, TasksInProgress int
  TasksDone, TasksCancelled, TasksBlocked, OpenEpics int
  ActiveAgentSessions, ActiveHumanSessions, LeasedTasks int
  LatestEventCursor int64
}
type ControlSession struct {
  SessionID, Role, RunID, Machine, Owner string
  RoleGeneration int64
  ExpiresAt time.Time
  Status string // active, expired, credential_invalid
  Task *Task // только effective valid session + live owned lease/current generation
  LastEventAt *time.Time // последнее записанное событие, не last_seen
}
type ControlSessionsInput struct {
  Cursor string
  Limit int
  SessionID, Role, Owner, State string // State: empty, active, expired, credential_invalid
  IncludeClosed bool
}
type ControlSessionsPage struct {
  PageInfo
  ObservedAt time.Time
  Sessions []ControlSession
}
type ControlTasksInput struct {
  ListInput
  Query string
}
type RecentEventsInput struct {
  BeforeCursor int64
  Limit int
  TaskKey, SessionID, Type string
}
type RecentEventsPage struct {
  Events []Event
  NextCursor int64
  HasMore bool
  ObservedAt time.Time
}
```

ControlSessions queries token validity in batch within same transaction; invalid
principal means missing/revoked/expired credential, precedence over TTL status.
Task join uses all current tasks, not first100. Default active sessions, explicit
IncludeClosed/State allows historic; keyset creation order same as existing
Sessions, filter-bound cursor. No capability to reuse another session as actor.
SessionID — optional exact filter for agent detail lookup before pagination;
detail не ищет произвольную сессию только в первой странице списка.
ControlOverview applies exactly the same effective predicate for active counts;
TasksInProgress is workflow fact, LeasedTasks is authorized live lease fact, они
могут различаться при отзыве token. TasksBlocked = nonterminal tasks с core
blocked predicate; subset indicator, не отдельная mutually exclusive колонка.

ControlTasks: whole-project search over title/description/key before pagination
with existing state/role/epic/owner/closed filters. Query trimmed, <=200 runes и
<=1024 bytes, literal case-insensitive substring с единым правилом для кириллицы
и латиницы (Go strings.ToLower по loaded tasks достаточно на C2; не SQL LIKE
wildcard injection). Пустой q эквивалентен ListTasks. Cursor binds normalized q
и остальные фильтры; при смене q/filter paging начинается заново. Не вызывать
native ListTasks и затем искать только в возвращённых100 строках. PageInfo не
обещает immutable values между страницами. Нового native tool/API не добавляется.

`Page` интегратор дополняет `Overview *core.ControlOverview`,
`Agents []core.ControlSession`, `Agent *core.ControlSession`, `Activity []core.Event`,
`Stats *core.Stats`, `ObservedAt time.Time`, `Query string`, `AgentState string`;
существующие Tasks/Epics/NextCursor/View/Draft/Actions/Session переиспользуются.
Для projects/settings используются actual `Session.Project/Rules/TTL`;
не вводить каталог фиктивных проектов. `NextCursor` activity — decimal int64.
Backend author передаёт root builder для этих полей, root wire routes/Board/Page.
UI author читает только frozen поля; details/unknown state показывает честно.

Backend ownership расширяется `internal/core/control_*.go`,
`internal/store/control_{sessions,tasks,overview,activity}.go` и их targeted tests;
не редактирует native catalog.go/transport. При нужном переиспользовании list
helper сначала согласовать file ownership с root; копия business rules запрещена.
UI author добавляет templates `overview`, `agents`, `agent`, `activity`, `settings`,
`projects`; root расширяет renderer whitelist/funcmap/static handler один раз.
SSR existing `board` template получает real search q и board/list selection.
Обязательный negative AC: отозванный token при будущем session TTL исчезает
из effective active counts, status credential_invalid, live lease badge отсутствует;
native Sessions TTL semantics остаётся прежним. Search hit за пределами первой
страницы, кириллический mixed case и escaped HTML также обязательны.

Открытый продуктовый вопрос PC-01 (root уже передал пользователю): нужен ли запуск
внешних агентов прямо из панели. До ответа независимый control UI можно делать;
process execution и model config не считать реализованными. Дальнейшее multi-project
onboarding/ACL/provisioning требует отдельного принятого продуктового контракта.

## 9. Принятые интеграционные уточнения координатора (приоритет над ownership §7/8)

Прямой запрос разрешает реализацию control UI по Paperclip; данные проекта сейчас
одного реального board. Общий REQ остаётся roadmap по всем категориям Paperclip,
TASK-005 поставляет текущую рабочую веб-волну. Открытый PC-01 не блокирует её.

- Backend103 владеет `internal/core/control*.go`, `internal/store/control*.go`,
  при необходимости refactor helpers `catalog.go` без изменения native semantics,
  **всеми изменениями internal/web/** (Board/Page/routes/builders/tests). Root
  не редактирует эти файлы одновременно. Сохраняются старые формы/IDs/POST пути.
- UI104 владеет **internal/webui/** целиком, включая render.go/helpers/unit tests,
  templates/assets/Static whitelist. Root ведёт docs/license/release. Обе стороны
  используют Go DTO §8; добавление поля согласуют сообщением до изменения.
- `ControlSessionsInput` также получает `SessionID, Kind string`. SessionID —
  точный фильтр для agent detail, включён в cursor binding. Kind: empty/all,
  agent (Role!=human), human (Role==human). Не совместимый role/kind —400.
  Контроль длины/допустимого SID до чтения. Нет поиска detail по первым100rows.
- Маршруты SSR: `/web/` overview по умолчанию, старые `?view=questions|tasks`
  работают; `/web/tasks`, `/web/questions`, `/web/agents`, `/web/agents/{id}`,
  `/web/projects`, `/web/activity`, `/web/stats`, `/web/settings` плюс прежние
  designs/history/taskdetail. View названия: overview/tasks/questions/agents/agent/
  projects/activity/stats/settings/designs/history/task. Board list/kanban через
  `layout=list|board` в Draft; выбранный вид сохраняется в pagination links.
- Template names overview/agents/agent/activity/projects/stats/settings — новые,
  прежние board/task/designs/history/login/error сохраняются. Overview agents:
  первая ограниченная страница Kind=agent/State=active, показанная как выборка,
  общий count только Overview. Agent Task указатель содержит полный актуальный
  owned task только для effective lease. Human sessions отдельным фильтром.
- Дополнительные Page поля: `Overview *core.ControlOverview`,
  `Agents []core.ControlSession`, `Agent *core.ControlSession`,
  `Activity []core.Event`, `Stats *core.Stats`, `ObservedAt time.Time`,
  `Query, AgentState string`. Остальные фильтры в существующем Draft (kind,owner,
  session_id,type,layout,days,group_by). Backend указывает эти Draft даже при ошибке.
- Activity filters task_key/session_id/type, NextCursor decimal; Sessions paging
  opaque. Все неизвестные query параметры и неверные значения новых readers
  дают400; cursor привязан к фильтрам. Поиск q применяется только задачам,
  ищет по всему проекту до pagination, не по первой странице.
- Browser GET `/web/data/overview` возвращает JSON `core.ControlOverview` после
  loadBrowser и проверки board session; не вызывает page()/issueAction/Describe/
  Heartbeat. Только этот read poll15s; остальное SSR с ручным обновлением.
  Ошибки — application/json с существующим code/status, без HTML200. Никаких
  native cookie auth, нет CORS. Browser polling только visible+online, одна
  in-flight, timeout/abort/backoff, last good snapshot сохраняется при сбое.
- Stats: readonly отдельный маршрут, query days=1|7|30|90 (default7),
  group_by=role|owner|epic (defaultrole), now-relative interval указанный на странице.
  Не polling. Токены и missing coverage/IncompleteData настоящие; деньги/model
  не выдумываются. Отсутствующий config revision/last_seen/process data —нетданных.
- Root/main редирект гостя в login сохраняется. Прежние POST redirect к вопросам
  можно оставить `?view=questions`, а успешный login —overview. Нельзя терять
  ongoing drafts/explicit heartbeat при новых background readers.
- Overview/tasks counts являются доменными фактами; alive session/lease count
  проверяет token revoked+expiry. Бейдж не утверждает running OS process.
  Search/overview не загружают дорогостоящий полный Stats на каждый refresh.
- Шрифт InterVariable self-host из reference разрешён после переноса официального
  SIL OFL notice/text; UI104 копирует только font asset, root —license/docs.
  Никакого CDN, logo или fixturedata. Paperclip MIT notice уже сохранён.

### Уточнение read cursors/error envelope

Opaque cursors ControlTasks/ControlSessions привязаны к нормализованным фильтрам.
RecentEvents использует числовой `before_cursor` из §8 (как границу DESC, неopaque
filter-bound token); при изменении фильтра форма начинает чтение с начала.
Query parsing/неотрицательная граница валидируются, но криптографическая привязка
этого числа к фильтру не заявляется. Это уточняет общее выражение про cursors выше.
Browser JSON errors: `{code,error}`, строки, HTTPstatus из действующего core
mapping; expired board session —409/LEASE_LOST. Это не новая native error схема.
Поиск q только задачам; questions сохраняет фактическую priority queue.

### Принятое пользовательское уточнение PC-01

19.10.2026 пользователь выбрал «Подключение и контроль уже запущенных агентов».
Это закрывает открытый вопрос выше. Текущая поставка показывает реальные сессии
и владение задачами; управляемый запуск процессов/модели/provider config остаются
будущим слоем. Существующие UI/backend контракты не меняются.
