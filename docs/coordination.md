# Координация — TASK011

20.10.2026. Контракт очередного прямо порученного этапа после TASK010 и перед
управляемым запуском. Независимый разбор: WORK147. Фактические проверки — TASK011.
Контракт расширяет requirements/API, сохраняя leases/fencing, ACL/epochs и SHA gates.

Пользователь явно подтвердил этот ручной контур20.10.2026: «Да, этот контур».

## Границы и решения

Сервис использует существующий pull-контур `describe → next → result`.
Добавляются назначение профилю, объяснение готовности/проверок, browser управление
зависимостями и разрешёнными состояниями, явное создание review и QA. Процессов,
фонового dispatch, AI-планирования, clone, provider connections или credentials
нет. Human browser session не становится developer/reviewer/tester и не завершает
их задачи. Реальную работу получает только самостоятельная native session.

Принято по WORK147: pinned profile revision, optional назначение при create,
отдельный assign, отдельный bounded detail reader, schema4 и coordination floor.
Это явный выбор версии инструкций: edit/archive приостанавливают новые claims
до переназначения, не меняют живые аренды. Stable-ID-only и session-ID assignment
отложены. Автогенерация pipeline и обязательное review для всех legacy задач не
вводятся. Области остаются метками, не файловыми locks.

Human и coordinator вправе административно готовить чужую незахваченную работу
(assignment/link/cancel/restore). Это узкое расширение прежнего requester gate для
рабочей панели owner/editor. Права живого holder, generation и completion не
расширяются; human не может снять чужую аренду. Viewer/архивный проект только read.

## Назначение

`Task.assignment?={profile_id,profile_version,profile_name,role}` хранит выбранные
ID/version/name/role в task JSON. Инструкции остаются в immutable profile revision.
Старые tasks остаются без назначения. `owner` не profile ID и не lease-holder.

При Create optional `profile_id`+`profile_version` принимаются только вместе;
profile active/current в том же проекте, role точно совпадает с Task.Role.
Human/epic не назначаются профилю. Проверка и создание в одной транзакции.

`AssignInput={request_id,task_key,version,mode,profile_id?,profile_version?,reason}`.
Mode `profile` требует полную пару; `role_queue` требует отсутствующую пару.
Version — ожидаемая Task.Version≥1, reason непустая≤4КиБ. Только ready/created,
без живой аренды. Права human/coordinator или прежний requester. Mismatch409;
операция увеличивает Task.Version, audit+response атомарны. Audit сохраняет
прежнее и новое assignment (либо явную role_queue) и reason, история переназначений
не заменяется текущим полем. Generation/attempts/
evidence не сбрасываются. Done/cancelled сначала возвращаются обычным transition.

Без assignment прежняя общая очередь роли доступна bound/unbound сессиям.
С assignment новый claim требует точный session.profile ID/version/role, а
current profile всё ещё active и той же version/role. Два одинаково подходящих
клиента конкурируют прежним atomic next; lease получает один. Новые profile
версии/архив не меняют heartbeat/release/completion существующего holder.
QueueSummary в SessionView/Claim считает доступность этому session snapshot;
общепроектная статистика остаётся общей и не объявляется персональной очередью.

## Объяснимость

`CoordinationInput={task_key}` → `TaskCoordination`:
`task_key,task_version,observed_at,assignment?,routing_reasons[],blockers[],
dependencies[],checks[],epic_closable?,closure_reasons[]`.
Причина имеет `code,message,task_key?`; dependency — `link,satisfied`; check —
`task_key,role,state,outcome,target_commit,target_generation,current,latest,successful`.
Current означает target done+совпадение SHA/generation независимо от outcome;
negative текущего target не называется stale. Latest — последнее назначение,
successful — полный положительный gate с учётом транзитивных зависимостей.
Данные вычисляются в одной authoritative transaction, без claim/renew.
Причины/коллекции сортируются детерминированно; максимум200 элементов в каждой,
`truncated:true` при усечении. Усечение не означает отсутствие других gates.

Routing: profile missing/archived/changed, task not ready/live lease/epic not open.
Actor-specific role/profile mismatch не выдаётся за общую блокировку проекта.
`blocked` прежней Task по-прежнему dependency predicate. Blockers используют
тот же blocked/successful/latestVerification evaluator, включая target state/SHA/
generation, negative/pending/superseded review и транзитивные gates. Explain не
вычисляет независимую ослабленную копию правил. Epic closure вызывает тот же
epicClosable; human не может закрыть эпик в обход обязательств даже после unlink.

## Явный цикл review → QA

`CheckCreateInput={request_id,target_key,target_version,role,profile_id?,
profile_version?}`: только human/coordinator, role reviewer|tester. Target —
done developer с актуальным полным SHA, ожидаемая Task.Version обязательна.
Сервер захватывает текущие SHA/generation в той же транзакции, создаёт новую
проверку в эпике target с понятными сгенерированными title/description.
Reviewer создаётся явно. Tester требует latest current positive reviewer и
атомарно получает зависимость от него `blocks/approved`. Нет промежутка между
созданием QA и постановкой review gate. Assignment валидируется как при Create.

Исторические проверки сохраняются. Новый check не наследует прежний verdict;
latest/pending/negative/stale обрабатывает существующий глобальный gate.
Generic Create остаётся совместимым с прежними explicit target_key/target_commit;
он не заявляется новой атомарной кнопкой «Создать QA». Независимость по server
session ID/author attempts остаётся прежней, profile ID её не заменяет.

## Wire и browser

Native REST scoped и legacy aliases:

| Метод/path | Operation |
| --- | --- |
| GET tasks/{key}/coordination | coordination |
| POST tasks/{key}/assign | assign |
| POST tasks/{key}/checks | check_create |

CLI call `coordination|assign|check_create` использует прежний --project и stdin.
MCP остаётся12tools: create принимает optional assignment; next использует routing.
Чтение через resource `accord://coordination{?session_id,task_key}`; assignment и
check_create через REST/CLI/browser. Параметры/поля unknown/duplicate/null reject,
body≤64КиБ, bearer/native X-Session и project scope сохраняются.

Browser task detail PageData добавляет `coordination`, `assignment_profiles`
(metadata-only active profiles максимум128), `can_coordinate`. Human/coordinator
с живой session и write ACL получают capability. Read не открывает agent session.
POST `/web/tasks/{key}/assign|link|transition|checks`: same-origin/CSRF/action_id,
issued action names `coord:assign|coord:link|coord:transition|coord:checks`.
Link принимает from/type/condition/unlink; to фиксирован path, generation текущего
consumer либо0. Transition — state/reason/evidence_ref/generation, без completion
agent evidence. Native permissions всегда проверяются повторно.

Browser action recovery включает все новые операции. Unknown outcome повторяет
исходный payload/action/session/project; known validation выдаёт новую capability.
409 не перезаписывает draft или expected version фоновым чтением. Все redirects,
cache keys, отмена запросов и blocker/profile links сохраняют project scope.

Next task detail показывает отдельную координацию: профиль/версию, назначение и
возврат в очередь роли, dependencies и причины, текущие/исторические проверки,
действия состояния/эпика и кнопки явного review/QA. Viewer читает без write controls.
Нет fake online/worker и кнопки запуска. Список задач показывает assignment
отдельно от роли и live session. Текущий стиль Paperclip/Next сохраняется.

## Storage, поставка и приёмка

Board3→4 additive: обязательная `coordination_metadata(id=1,version=1)` фиксирует
полноту схемы и поддержку task routing. Старые task/session JSON/ACL/rows не
перезаписываются миграцией. Schema1/2 идут последовательно; до DDL private durable
snapshot. Повреждённая заявленная4 не чинится CREATE IF NOT EXISTS. Registry1/web2
прежние. Published storage preflight принимает полные2/3/4; fresh boards создают4.

Capabilities `coordination:1,agent_profiles:1,board_schema:[4]` и registry/ACL.
Guard floor ставится под owned startup hold до DDL. Capable updater проверяет
полный all-project snapshot/preservation, schema4 backup/restore и исторические
schema1/2/3 migration bytes/checksums. Marker service UID/0600 сохраняется.
Initial TASK009 apply, downgrade, DB rollback и reset production запрещены.

Авторские unit/integration → независимый source APPROVE → отдельные process/
browser тесты и их review → exact Linux/synthetic updater/recovery/full restore →
production full backup/update/HTTPS/owned runtime/preservation → final review.
Физический второй ПК, provider execution и полный V1 остаются NOT-RUN.

Проверены первичные API20.10.2026: [SQLite transactions](https://www.sqlite.org/lang_transaction.html),
[application user_version](https://www.sqlite.org/pragma.html#pragma_user_version),
[Next Link/prefetch](https://nextjs.org/docs/app/api-reference/components/link).
Новые зависимости не требуются.
