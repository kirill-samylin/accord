# API пилота — v1

> TASK014 добавляет ограниченный автоматический цикл, контракт —
> [automation.md](automation.md), текущая проверка — [TASK014](TASK-014-automation.md).
> Эти endpoints реализованы в source; установка и native приёмка отдельно.
> В registry-режиме MCP дополнен четырьмя tools: `workflow_submit`,
> `workflow_command`, `workflow_plan`, `workflow_question`. Исторический лимит
>12 относится к прежним этапам и standalone board без registry authority.

Scoped REST `/api/v1/p/{projectID}/automation/` принимает POST JSON с обычным
`X-Session`: `submit`, `command`, `plan`, `question`, `answer`, `list`, `get`,
`report`. Generic CLI `call workflow_<operation>` использует тот же контракт.
Требование без policy сохраняется как `awaiting_authorization`, без запуска.
Такой черновик (без `grant`) запускается только `command` `resume` с явными
`refresh_policy_id/refresh_policy_version`; `resume`/`replan` без них —
`VALIDATION` «Черновик ещё не запущен…», а не `FORBIDDEN_ROLE`. `WorkflowPage`
содержит `can_resume`: подсказку, что простой `resume` без новой policy сейчас
пройдёт структурные проверки (для черновика всегда `false`); авторизацию
по-прежнему решает сам `command`.
`command` использует текущую version, поддерживает pause/resume/stop/clarify
и явный replan остановленного цикла. Доставка уточнений — на следующем turn;
статусы очереди не подтверждают применение или остановку процесса.

Owner CLI REST `policies/create|update|revoke` принимает отдельное поле
`host_proof` (revoke также `host_consent_version`). Сервер проверяет актуальный
локальный proof владельца и consent, создаёт typed approval внутри authority
guard; proof не сохраняется в policy/receipt. `policies/get` читает ревизии.
`cursor/limit` ограничивают историю; `next_cursor` продолжает её при неизменной
текущей версии. `policy` всегда содержит действующее состояние политики.
Generic CLI: `call automation_policy_create|update|revoke|get`. Это не MCP tools.
Веб подтверждает policy через отдельные CSRF/action проверки.

Worker-only POST `capabilities/register` и `attest` требуют текущую human owner
session, host proof и точную dispatched попытку. Child получает отдельный
execution credential, не owner bearer. Child не имеет catalog/browser/admin,
intake/control/host доступа. Read resources —
`accord://workflows`, `accord://workflow`, `accord://workflow-report` с query
`session_id,workflow_id,state,cursor,limit`; child читает только свой workflow.
Локальный bridge предоставляет `workflow_run_check {request_id,preset_id}` и
status-only `accord://workflow-check{?request_id,preset_id,session_id}`.
Пути, команды, аргументы и чужая task/session/generation моделью не передаются.
В ACK `attest` поле `attestation.control_epoch` сохраняет исходную попытку;
верхнее `control_epoch` — текущие сведения workflow. Более новый epoch в
terminal ACK после pause не предоставляет права продолжить исполнение.
`observations.check_aborted=true` исключает успешную проверку даже при exit0.

Архив ротации истории (read-only, только owner проекта или global admin с
membership; editor/viewer/child — `403`): POST `/api/v1/p/{projectID}/automation/archive`
с `{kind:"workflows"|"managed_runs",cursor?,limit?≤32}` и обычным `X-Session`
(query-параметры запрещены), MCP resource
`accord://automation-archive{?session_id,kind,cursor,limit}`, generic CLI
`call automation_archive`. Ответ `{kind,workflows,runs,next_cursor?}`; проекция
как у live list (без prompt/output/instructions). Это не MCP tool.


> TASK013 расширяет scoped REST/browser API управлением добровольно
> подключёнными workers. Host proof и native controller ACL не заменяют
> project/board session auth. MCP остаётся12tools. DTO, monotonic command
> phases, idempotency и native provider границы — [managed-agents.md](managed-agents.md);
> фактическая приёмка schema5/managed_agents1 — [TASK013](TASK-013-managed-agents.md).

Managed bridge добавляет в `OpenSession`/`SessionView` необязательную пару
`managed_run_id`, `managed_generation`. Пара закрепляется приватной конфигурацией
попытки и подставляется bridge, не выбирается моделью. Только текущий владелец
active host под действующей worker lease и подтверждённой execution-командой
может связать точную сохранённую версию профиля run после изменения/архивации
каталога. Обычный `describe` по-прежнему требует актуальный active профиль;
gates назначения задач и независимого review остаются отдельными.

> TASK012 добавляет scoped REST `POST tasks/similar`, `POST tasks/{key}/duplicate`,
> `GET team`; CLI `similar|duplicate|team`, MCP resources similar/team и
> существующий link с `duplicate_of` без13-го tool. Create/Claim могут возвращать
> advisory warnings; strict create требует явные force/force_reason.
> DTO, права, replay, browser и ограничения — [TASK012](TASK-012-dedup-team.md).
> Durable semantics защищены `deduplication:1` floor при прежней schema4.

> TASK011: назначения задач pinned профилям, объяснимые gates и ручной цикл
> review → QA описаны в [coordination.md](coordination.md). Schema4/coordination1
> floor защищает новые task semantics; фактическая готовность — TASK011.

> TASK010 добавляет постоянные [профили агентов](agent-profiles.md):
> GET `profiles?state=active|archived|all`, `profile?profile_id=ap_...&version=N`
> (version необязательна), POST `profile_create/profile_update/profile_archive`.
> Все scoped aliases/ACL сохраняются. MCP12tools: чтение через resources
> `accord://profiles{?session_id,state}` и
> `accord://profile{?session_id,profile_id,version}`, выбор через describe
> с обязательной парой profile_id/profile_version и совпадающей role.
> Browser `/web/p/{id}/profiles[/{profileID}]`; полный DTO и mutation/retry
> контракт — agent-profiles.md. Профиль не token/session/worker execution.

> TASK009 добавляет явные scoped endpoints `/api/v1/p/{projectID}/...`,
> `/mcp/p/{projectID}`, `/web/p/{projectID}/...` и browser catalog. Legacy paths
> сохраняют default scope и проверку его ACL. Точный identity/membership,
> cookie/CSRF/action/epoch и миграционный контракт —
> [multi-project.md](multi-project.md); фактические проверки — TASK009.

Дата: 18.10.2026. Контракт TASK-001. Типы Go: `internal/core/types.go`;
правила — [requirements.md](requirements.md). Новые поля не отменяют инварианты.
На wire используется `state` и единое `outcome`; это нормализация вариантов
`to`, `verdict`, `result` исходного ТЗ. Разработчики сверяют реализацию с этим
контрактом; наличие документа не является доказательством реализации.

## Авторизация и ответы

Все `/api/v1/*` и `/mcp` требуют `Authorization: Bearer <token>`.
После открытия board-сессии REST требует `X-Session: <session_id>`, а каждый
MCP tool input — `session_id`. MCP transport id не заменяет board session id.
Сервер связывает сессию с token principal; `machine` в describe опционален:
пустое значение означает machine токена (WORK299), а явное значение обязано
с ним совпадать. Без сессии доступны только describe/session и служебный health.

Успех пилота: bare JSON business-результата. Общие `server_time`/`high_water`
во все ответы не добавляются; `high_water` обязателен только для EventsPage.
Мутации требуют `request_id` (1..128 символов — считаются Unicode-символы, не
байты; валидный UTF-8 без управляющих символов), scoped токеном+сессией;
describe scoped токеном. Канонический payload включает operation, целевой ключ,
поля тела и поколение; `session_id` — часть scope. Replay возвращает исходный
сохранённый результат и исходные lease timestamps, не продлевая ничего.

Ошибки REST: `{"error":{"code":"LEASE_LOST","message":"…"}}`.
400 VALIDATION/INVALID_TRANSITION/EVIDENCE_MISSING/CYCLE;
401 UNAUTHORIZED; 403 FORBIDDEN_ROLE; 404 NOT_FOUND/NO_SESSION;
409 LEASE_HELD/ROLE_HELD/LEASE_LOST/IDEMPOTENCY_CONFLICT/RULE_VIOLATION.
LEASE_LOST получает только бывший держатель (его сессия имела попытку этой
задачи) либо вызов без действующей аренды; сессия, которая никогда не держала
задачу, занятую другой сессией, получает LEASE_HELD.
Managed worker (`managed/report`, `managed/poll`) получает явные коды, не текст:
409 STALE_GENERATION (устаревшая generation), STALE_DISPATCH (не текущий dispatch),
COMMAND_TERMINAL (команда уже terminal), MANAGED_UNCERTAIN (судьба прежнего
dispatch неизвестна — нужна локальная сверка), LEASE_HELD (прежний worker ещё
держит live lease — временно, повторить); 404 RUN_ARCHIVED (run перенесён в
архив истории) или NOT_FOUND. Коды на проводе в верхнем регистре.
MCP domain error — tool result `isError=true` с тем же JSON; malformed
JSON-RPC/transport errors обрабатывает SDK. Ошибка чтения resource приходит
JSON-RPC ошибкой с сообщением `CODE: message`; stdio bridge сохраняет код
домена в `error.data.code` (NOT_FOUND → -32602 по SEP-2164, VALIDATION →
-32602, прочие → -32000). Токен/секрет не отражается в ошибке.

Текст: title, request_id, run_id, owner, areas и имена проектов — валидный
UTF-8 без управляющих символов (C0, DEL, C1); description и comment text
дополнительно допускают `\n`, `\t` и CRLF. Нарушение — VALIDATION.

## Пилотные маршруты

| REST | MCP tool | Операция |
| --- | --- | --- |
| POST /api/v1/session | describe | Открытие board-сессии |
| POST /api/v1/tasks | create | Создание задачи или эпика по type |
| GET /api/v1/tasks/{key} | get | Явное чтение задачи/эпика |
| POST /api/v1/tasks/next | next | Атомарный захват задачи своей роли |
| PATCH /api/v1/tasks/{key} | update | Разрешённые метаданные |
| POST /api/v1/tasks/{key}/transition | transition | Переход с evidence |
| POST /api/v1/tasks/{key}/release | release | Освобождение своей аренды |
| POST /api/v1/tasks/{key}/comments | comment | Краткий отчёт своей попытки |
| POST /api/v1/links | link | Направленная зависимость |
| POST /api/v1/heartbeat | heartbeat | Явное продление текущих поколений |
| GET /api/v1/events | events | Страница журнала |
| GET /healthz, GET /readyz | — | Жив процесс / доступно хранилище |

REST path `{key}` — canonical `T-N`/`E-N`; транспорт переносит его в TaskKey.
Нельзя одновременно прислать другой task_key в теле. Статические маршруты
`next` регистрируются отдельно от чтения по ключу.
Эпик в пилоте может возвращаться унифицированным WorkItem/Task DTO:
key=E-N, role пуст, state=open|done|cancelled; бизнес-правила эпика отдельны.

## DTO

`Principal{ id,owner,machine }` получается только через Authenticate(token).
`Actor{Principal,SessionID}` строит транспорт, а не декодирует из тела клиента.

```json
{"request_id":"open-dev-1","role":"developer","run_id":"dev-pc-a-1","machine":"pc-a"}
```

Describe → `{session_id,role,role_generation,expires_at,project}`.
`role_generation=0` у несинглтонных ролей; architect получает
положительное поколение. Board-сессия не меняет роль; новый role — новая сессия.
TTL сессии/ролей и heartbeat interval возвращаются сервером либо документируются
CLI конфигурацией; текущий пилотный default lease TTL=15м, heartbeat=5м.

```json
{"request_id":"create-1","type":"epic","title":"Документы","description":"Цель"}
```

```json
{"request_id":"create-2","type":"task","epic_key":"E-1",
 "title":"Реализация","description":"Входы и ожидаемый результат",
 "role":"developer","priority":1}
```

`priority` опционален, целое 0..100, меньше — срочнее: отсутствующее поле
означает 2, явный `0` сохраняется как P0 (прежняя замена 0 на 2 удалена;
клиенты, отправлявшие 0 вместо «по умолчанию», должны опустить поле). Проверка
reviewer/tester наследует priority target. `description` (как и comment) —
валидный UTF-8 без управляющих символов, кроме `\n`, `\t` и CRLF.

Create задачи в пилоте сразу создаёт ready; стадия черновика created
зарезервирована полной V1 и не включается неописанным параметром.
Для reviewer/tester обязательны `target_key` и `target_commit`; target —
developer-задача в том же экземпляре. `target_generation` фиксирует сервер:
для done target при создании, иначе при первом claim проверки после target done.
Проверка создаётся для конкретного SHA;
после нового SHA создаётся новый цикл проверки, история старого сохраняется.
Для текущих SHA/generation последняя назначенная проверка той же роли
заменяет предыдущую: старый approve не обходит новый pending/negative.
Исторический approve больше не открывает и явные зависимости. После нового
положительного цикла потребитель явно заменяет старую связь через unlink/link.
Поля areas и owner опциональны в пилоте; не выдавать их отсутствие за
проверенную блокировку пересечения областей или epic-owner policy.
areas: ≤32 различных значений, каждое 1..128 символов, без сегментов `.`/`..`.
owner принимается только для epic (1..128 символов); owner в create задачи —
VALIDATION, а не молча отброшенное поле: участник задачи — держатель аренды.
SHA (`target_commit`, evidence `commit`/`verified_commit`) принимаются в любом
регистре, сохраняются в нижнем и сравниваются без учёта регистра.

```json
{"request_id":"claim-1","task_key":"T-1"}
```

Next: task_key опционален; без него сортированная очередь. epic_key — фильтр.
Возвращает `{"task":{…},"lease":{"session_id":"…","generation":1,
"expires_at":"…"}}`; при пустой очереди task=null.
Claim нельзя использовать для чужой роли или передать другому actor.

Task DTO: `key,epic_key,title,description,role,state,priority,version,
generation,lease?,evidence,review?,author_session?,attempts,comments,links,blocked`.
Review: `target_key,target_commit,target_generation,outcome,author_session`; author_session —
provenance целевого результата, не произвольная присланная строка.
Attempt: `session_id,generation,started_at,ended_at?,reason?,branch?,commit?`.
Lease: `session_id,generation,expires_at`. Version относится к изменению записи,
generation — исключительно к fencing захватов; одно не заменяет другое.

Get input MCP: `{session_id,task_key}`; REST path key. GET не продлевает аренду.
Update: `{request_id,generation,title?,description?}`; только разрешённые
метаданные. Evidence/role/state/lease/attempts нельзя свободно подменять patch.
Активную задачу изменяет её держатель с generation; незахваченную — requester/
coordinator. Update без полей либо с теми же значениями — успешный no-op:
возвращает текущую задачу, не меняет version, не создаёт событие и не продлевает
аренду (для продления служит heartbeat). Ограничения по авторизации применяет core, не только HTTP handler.
Ненулевое generation требует действующую аренду и после expiry/release даёт
LEASE_LOST; административный запрос к незахваченной задаче опускает generation
либо передаёт0. Такое же правило применяется к link/unlink.

```json
{"request_id":"done-1","generation":1,"state":"done",
 "evidence":{"commit":"1111111111111111111111111111111111111111",
 "run_summary":"go test ./...: PASS"}}
```

Transition получает task_key из REST path/MCP body. Для reviewer:
`outcome=approve|changes_requested|blocked`, evidence `verified_commit,findings`.
Для tester: `outcome=passed|failed|not_run|not_applicable`, evidence
`verified_commit,run_summary`. Эти результаты допустимо хранить в Review,
но verified_commit всегда проверяется с target_commit и текущим commit target.
Для analyst/architect: evidence `artifact:["docs/decision.md"],summary:"…"`;
human: `answer`; coordinator: `summary`.
Переоткрытие: `state=ready,reason,evidence_ref`, generation не даёт права
снять чужую аренду; матрица переоткрытия проверяется сервером.
Done возвращает актуальную задачу; снятие gates и события — в одной транзакции.

Release: `{request_id,task_key,generation,reason,branch?}`. Reason обязателен.
Comment: `{request_id,task_key,generation,text}` (text ≤4 КиБ), только своя живая
попытка в пилоте; общий комментарий любого читателя — расширение V1.
Истёкшая аренда даёт LEASE_LOST даже до фонового sweep.

```json
{"request_id":"dependency-1","from":"T-2","to":"T-1","generation":1,
 "type":"blocks","condition":"completed"}
```

Link types: blocks,contract_of,spawned_by,related. Связь элемента с самим
собой (from = to) отклоняется VALIDATION для любого типа, включая duplicate_of. `unlink:true` удаляет указанную
связь с аудитом по тем же правам на consumer, что добавление. Это не снимает
историческую обязательность роли проверки target для закрытия эпика.
Condition можно опустить:
для reviewer default approved, tester passed, прочих completed. Явное completed
для проверки не обходит требование успешного результата роли. approved/passed
требуют подходящей роли и актуального commit. Цикл проверяется транзакционно.
Для живого consumer `to` обязателен его `generation`: менять связь может
только держатель этой аренды с актуальным поколением и сроком. Для незахваченного
consumer — requester/coordinator. Это позволяет запросить архитектора во время
работы, не освобождая задачу в гонке между release и добавлением блокировки.

Heartbeat: `{request_id,task_key?,generation?,role_generation?}`.
Если задача у сессии есть, task_key+generation обязательны; для architect
обязательно role_generation. Нельзя одним heartbeat частично продлить роль,
но получить отказ по устаревшей задаче. Ответ — актуальная SessionView с
арендами либо ошибка целиком. Replay heartbeat не продлевает повторно.

Events MCP: `{since_cursor:0,limit:100,types:[],task_key?:"T-1"}`;
REST: `?since_cursor=0&limit=100&types=claimed,task_done&task_key=T-1`.
Возвращает `{events:[{cursor,type,task_key?,session_id?,at}],next_cursor,
high_water,has_more}`. `high_water` фиксируется на начало страницы.
При фильтре next_cursor продвигается до последней просмотренной записи,
но не перескакивает непоказанную подходящую запись. Пустая страница может
продвинуть next_cursor до high_water, если на этом диапазоне совпадений нет.
since_cursor≥0, limit=1..200 (числа вне диапазона, например 1e309, —
VALIDATION). since_cursor больше high_water не отклоняется, а ограничивается:
next_cursor всегда ≤ high_water. Новый фильтр начинает новый проход с сохранённой
нужной позицией. Высокий high_water из другого ответа читать как прогресс нельзя.

## Store API и границы

Core DTO находятся в `internal/core`. Store предоставляет:
`Open(path,Config,Clock)`, `Close`, `CreateToken`, `RevokeToken`, `Authenticate`, `Describe`,
`Next`, `Get`, `Create`, `Update`, `Transition`, `Release`, `Comment`, `Link`,
`Heartbeat`, `Events`. Контекст первым аргументом; операции возвращают тип и error.
Clock внедряется для точных TTL-тестов; сетевой клиент не задаёт серверное now.
Токены создаются/отзываются локально CLI bootstrap (`token create`,
`token revoke --id …`), не публичным анонимным REST endpoint.

## MCP bootstrap и ограничения

Streamable HTTP `/mcp`, stateless transport, бизнес-сессии в SQLite.
12 имён board tools в стабильном порядке: comment, create, describe, events,
get, heartbeat, link, next, note, release, transition, update. Если у проекта
есть registry automation authority, добавляются ещё четыре: workflow_submit,
workflow_command, workflow_plan, workflow_question (всего 16). Локально
авторизованным principal список одинаков; role authorization проверяется на вызове.
`describe` не вызывает tools/list_changed и не меняет схемы списка.
Читающие tools помечаются readOnly; у мутаций нет обещания безопасности лишь
потому, что для повтора нужен request_id.

Описания краткие; проверка ≤12 board tools (≤16 с automation) обязательна, оценка ≤3000 токенов не выдаётся
за результат конкретного токенизатора без измерения. Prompts, OAuth,
import и config apply — последующие этапы. Весь transport
делегирует правила core; отдельного упрощённого MCP пути без проверок нет.

## Изменения полной V1

В V1 появятся отдельные /config, /tokens,
/import,/export,/webhooks/github и веб-cookie auth. Добавление этих маршрутов
не разрешает менять существующие fencing/idempotency/cursor semantics.
Подробные пагинация, лимиты и compatibility matrices фиксируются до реализации
соответствующего этапа, а не симулируются пустыми успешными ответами пилота.

## Срез C: списки, заметки и статистика

Эта секция описывает текущие source contracts. Проверки и установленную
версию сверяйте с TASK-003: новый source не устанавливается автоматически.

Все запросы требуют bearer и собственный `X-Session`. GET не продлевает lease.
`GET /api/v1/tasks`, `/epics`, `/sessions`: limit1..200, cursor, state, role,
epic_key, owner, include_closed=true|false. Ответ tasks либо sessions,
next_cursor/high_water/has_more; курсор непрозрачен и привязан к фильтрам.
High_water ограничивает создание новых объектов, но не замораживает состояния
существующих. Для нового прохода не передавайте cursor. История done/cancelled
включается явно state либо include_closed. Session state — active|expired.

`POST /api/v1/notes` использует тот же DTO, что MCP `note`:

```json
{"request_id":"note-example-1","action":"create","scope_type":"role",
 "scope_id":"developer","source_task_key":"T-2","text":"Краткий вывод"}
```

Для удаления: action=delete, key=N-N, reason. Удаление мягкое, provenance
сохраняется. Собственная роль либо эпик доступной source task; curator может
удалять чужие заметки. Текст≤2048UTF8bytes, quotas проверяются транзакционно.
`GET /api/v1/notes`: scope_type, scope_id, limit, cursor, include_deleted.
MCP resource template:
`accord://notes{?session_id,scope_type,scope_id,limit,cursor,include_deleted}`.
Прямой HTTP требует session_id в URI; stdio worker подставляет только свой.
Наличие session_id не заменяет bearer. Resources не увеличивают число tools.

`GET /api/v1/stats?from=RFC3339&to=RFC3339&group_by=role|owner|epic`:
полуоткрытый UTC интервал, totals/groups, incomplete_data/incomplete_events.
Старые события без provenance не дополняются предположениями. Текущая очередь
отдельна от исторических completion/tokens/duration. Это не рейтинг моделей.
CLI: `call tasks|epics|sessions|notes|stats` получает query-поля через JSON stdin;
`call note` выполняет мутацию с request_id. Не передавайте bearer через JSON.

Статистика считает tokens_spent/attempts/missing_tokens по попыткам, закрытым
в интервале [from,to); активные delta доступны в attempts задачи. Это итоги
по времени завершения попыток. Final total заменяет delta total. Audit хранит
tokens_spent_delta, tokens_spent_total и correction_delta отдельно; их нельзя
суммировать как независимые расходы. tokens_spent_delta и tokens_spent
ограничены 10^12 на попытку (больше — VALIDATION). Если старые строки журнала
содержат большие значения, сумма насыщается до int64 max и отчёт помечается
incomplete_data=true вместо ошибки. current_queue относится ко времени
запроса. Неизвестные first timestamps старых задач остаются null/отсутствуют;
legacy timing/attribution помечается incomplete.

group_by=owner означает участника фактической попытки/серверной сессии
(token.owner), а не владельца эпика. Для текущей очереди owner известен только
при живой аренде; незахваченная работа относится к группе с пустым ключом.

## Browser read API TASK-006

`GET|HEAD /web/data/dashboard` — browser cookie и действующая сессия доски,
не bearer/native API. Параметры URL не принимаются. Ответ содержит `overview`,
`agents` (до 6 активных agent sessions), `activity` (до 20 последних событий).
DTO и отдельные времена чтения — [контракт TASK-006](TASK-006-control-refinement.md).
Overview дополнительно содержит `tasks_available`: ready задачи без аренды, не
заблокированные незавершёнными зависимостями, в открытом эпике, с доступным
назначением и не human-вопросы — тот же предикат, что `unclaimed_total` экрана
«Команда». `tasks_ready` остаётся счётчиком состояния (колонки состояний
по-прежнему суммируются в `tasks_total`). `done_recent` участника «Команды»
считает задачи, завершённые его попыткой (включая ответы человека) за окно.
События browser-ленты активности содержат вычисляемое `actor` — владельца
сессии события, если событие не записало собственного `data.owner`; в журнал
событий и native `events` это поле не попадает.
Ошибка любой коллекции возвращает ошибку всего запроса; GET не продлевает сессию
или аренду. Прежний `/web/data/overview` сохраняет контракт TASK-005.

## Представление страниц для Next.js — TASK-007

Новый клиент `/app` получает страницы и отправляет формы прежним `/web/**`
с `Accept: application/json`. Это browser API с cookie/Origin/CSRF, не native
Bearer API. Обычный HTML остаётся доступен без этого Accept. Статика,
`/web/data/**` и binary download/preview сохраняют прежние представления.

GET возвращает `{ok:true,page:{...}}`; успешная форма —
`{ok:true,redirect:"/web/..."}`, heartbeat — также `session`/`next_action_id`.
Ошибки сохраняют HTTP status и возвращают `{ok:false,error:{code,message},
page,draft,field_errors,action_id,actions,retry_same_action}`. Набор полей зависит
от страницы/операции. Durable action ID и capability берутся из ответа сервера;
при неизвестном исходе повторяют прежнее действие, а не создают новое.
Ошибки каталога (`/web/catalog/**`, администрирование) дополнительно содержат
`editable`: `true` для VALIDATION и конфликтов, решённых самими полями формы
(дубль участника или репозитория, предел, последний владелец/админ). Тогда
форма остаётся редактируемой и следующая отправка берёт свежий action ID;
конфликт устаревшей версии (`… changed`) требует перечитать страницу.

`POST /web/tasks/{key}/update` принимает form fields `csrf`, `action_id`,
`generation`, `title`, `description`. Разрешения проверяет тот же `Store.Update`:
собственные незахваченные задачи/эпики human-requester либо собственная живая
аренда с актуальным generation. Чужой lease не обходится. Заголовок обязателен
и ограничен200символами; описание допускает пустую строку, максимум16КиБ.
Роль, приоритет и состояние этим действием не меняются.

`error.message` browser-ответов всегда по-русски: native REST/MCP сохраняют
стабильные английские сообщения store, а веб-слой переводит известные
(например self-link — «Задача не может зависеть от самой себя») и иначе
объясняет код (`CYCLE`, `CONFLICT`, `LIMIT_EXCEEDED`, `RULE_VIOLATION`,
`EVIDENCE_MISSING`, `IDEMPOTENCY_CONFLICT`, `ROLE_HELD`…). `error.code` не
меняется. Формы создания/правки задачи заранее проверяют текст тем же правилом
управляющих символов и отдают `field_errors` нужного поля; `priority` формы:
пустое поле — 2, `0` — P0, иначе `field_errors.priority`. В каталоге проектов
повторное подключение того же репозитория (URL без учёта регистра хоста,
схемы https/ssh, хвостового `/` и `.git`, с той же веткой) — `CONFLICT`
«Этот репозиторий уже подключён»; повторный grant активного участника —
`CONFLICT` «Этот участник уже есть в проекте; измените его роль в списке».
Повторное открытие формы входа во второй вкладке переиспользует живую
preauth-cookie (тот же CSRF), не делая недействительной первую форму.

Полная проекция полей и проверки — [WORK-124](work/WORK-124-next-browser-bridge.md).
Топология Next/Go, ограничения rewrite/upload и локальный запуск —
[next-web.md](next-web.md). Native REST/MCP wire contract выше сохраняется.
