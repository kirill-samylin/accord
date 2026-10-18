# API пилота — v1

Дата: 18.10.2026. Контракт TASK-001. Типы Go: `internal/core/types.go`;
правила — [requirements.md](requirements.md). Новые поля не отменяют инварианты.
На wire используется `state` и единое `outcome`; это нормализация вариантов
`to`, `verdict`, `result` исходного ТЗ. Разработчики сверяют реализацию с этим
контрактом; наличие документа не является доказательством реализации.

## Авторизация и ответы

Все `/api/v1/*` и `/mcp` требуют `Authorization: Bearer <token>`.
После открытия board-сессии REST требует `X-Session: <session_id>`, а каждый
MCP tool input — `session_id`. MCP transport id не заменяет board session id.
Сервер связывает сессию с token principal; machine при открытии совпадает с
machine токена. Без сессии доступны только describe/session и служебный health.

Успех пилота: bare JSON business-результата. Общие `server_time`/`high_water`
во все ответы не добавляются; `high_water` обязателен только для EventsPage.
Мутации требуют `request_id` (1..128 символов), scoped токеном+сессией;
describe scoped токеном. Канонический payload включает operation, целевой ключ,
поля тела и поколение; `session_id` — часть scope. Replay возвращает исходный
сохранённый результат и исходные lease timestamps, не продлевая ничего.

Ошибки REST: `{"error":{"code":"LEASE_LOST","message":"…"}}`.
400 VALIDATION/INVALID_TRANSITION/EVIDENCE_MISSING/CYCLE;
401 UNAUTHORIZED; 403 FORBIDDEN_ROLE; 404 NOT_FOUND/NO_SESSION;
409 LEASE_HELD/ROLE_HELD/LEASE_LOST/IDEMPOTENCY_CONFLICT/RULE_VIOLATION.
MCP domain error — tool result `isError=true` с тем же JSON; malformed
JSON-RPC/transport errors обрабатывает SDK. Токен/секрет не отражается в ошибке.

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
coordinator. Ограничения по авторизации применяет core, не только HTTP handler.

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

Link types: blocks,contract_of,spawned_by,related. `unlink:true` удаляет указанную
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
since_cursor≥0, limit=1..200; новый фильтр начинает новый проход с сохранённой
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
11 имён tools в стабильном порядке: comment, create, describe, events, get,
heartbeat, link, next, release, transition, update. Локально авторизованным
principal список одинаков; role authorization проверяется на вызове.
`describe` не вызывает tools/list_changed и не меняет схемы списка.
Читающие tools помечаются readOnly; у мутаций нет обещания безопасности лишь
потому, что для повтора нужен request_id.

Описания краткие; проверка ≤12 обязательна, оценка ≤3000 токенов не выдаётся
за результат конкретного токенизатора без измерения. Resources/prompts, OAuth,
note/statistics/import и config apply — последующие этапы. Весь transport
делегирует правила core; отдельного упрощённого MCP пути без проверок нет.

## Изменения полной V1

В V1 появятся отдельные /epics, /notes, /stats, /config, /tokens,
/import,/export,/webhooks/github и веб-cookie auth. Добавление этих маршрутов
не разрешает менять существующие fencing/idempotency/cursor semantics.
Подробные пагинация, лимиты и compatibility matrices фиксируются до реализации
соответствующего этапа, а не симулируются пустыми успешными ответами пилота.
