# Постоянные профили агентов — TASK010

20.10.2026. Контракт следующего этапа после TASK009. Полномочие — прямая
очередь пользователя «постоянные профили → координация → управляемый запуск».
Архитектурная проверка — WORK140; фактические результаты — TASK010.

## Сущности и принятые границы

Профиль принадлежит одному проекту и хранится в его board.db. Это шаблон
разрешённых настроек и инструкций. Он не credential, identity, native session,
присутствие, аренда или execution run. Совпадение имён ничего не связывает.
Проектные права остаются в registry; профиль не даёт доступа к другому проекту,
не повышает ACL и не доказывает независимость reviewer.

Настройки provider/model — декларативные подписи. Credentials, произвольные
адреса, команды, env, shell, tool grants, расписания, бюджет и запуск в этом
этапе не принимаются. Сервис не обращается к provider и не исполняет инструкции.
Клиент явно получает snapshot инструкций при describe; соблюдение инструкций
моделью сервер не подтверждает. Координация и выполнение — следующие этапы.

CRUD означает создание, чтение, полное редактирование и архив/восстановление.
Физическое удаление не вводится: архив сохраняет историю и ссылки сессий.
Профили не создаются автоматически из прежних сессий/токенов/имён.

## Данные, версии, ограничения

`AgentProfile`: `id:ap_<32 lowercase hex>`, `name` (1–120 Unicode символов),
`description` (до4КиБ UTF-8), `role` (coordinator/analyst/architect/developer/
reviewer/tester), `instructions` (до16КиБ), `provider` (unconfigured/openai/
anthropic/local/custom), `model` (до200 Unicode символов, без control chars),
`state:active|archived`, `version` (целое≥1), `created_at`, `updated_at`,
`created_by`, `updated_by` (server-generated session ID, не credential ID). Имя не уникально;
stable ID — единственный ключ. Provider/model не означают поддержку интеграции.
Свободные тексты считаются пользовательским контентом, не secret storage.

Не более128профилей, включая архив. Список ограничен этой квотой, отсортирован
детерминированно; фильтр state active/archived/all. Лимиты тела прежние64КиБ.
Нельзя передавать неизвестные/повторные поля, actor, project/DB/upstream paths.
Create создаёт active/version1; update передаёт ожидаемую version и все
редактируемые поля. Archive/restore передают version и только новое state.
Каждое изменение увеличивает version и записывает неизменяемую ревизию.
Не более1024ревизий на профиль; переполнение даёт явный409 без удаления истории.
История возвращает максимум1024записи метаданных; полная ревизия читается явно.
Конфликт версии —409, а не успех. Revisions и audit фиксируются в той же
транзакции, что mutation и durable native response. Events содержат ID/version/
операцию, но не полный текст инструкций.

## Права, session binding и повтор

Owner/editor активного проекта создаёт/редактирует/архивирует/восстанавливает
профили через собственную живую native session. Viewer читает, но не пишет;
архивный проект читается без profile writes и новых profile-bound sessions.
В single-board режиме действуют прежние credential/session правила.
Auth/ACL/access_epoch/session ownership проверяются до чтения/replay по тем же
guards, что tasks. Profile ID другого проекта даёт not_found. Все mutation
request IDs и browser action capabilities scoped к исходному проекту/эпохе.

`describe` обратно совместим: без profile fields прежний контракт. Для привязки
обязательны вместе `profile_id` и `profile_version`; `role` обязана совпасть с
профилем. Profile должен быть active и version актуальна в момент новой сессии.
Exact replay возвращает прежнюю сессию/snapshot после редактирования/архива,
если current auth/ACL допускают replay. Другой payload с тем же ID — conflict.
Session получает неизменный `profile` snapshot (id, version, name, role,
description, instructions, provider, model). Уже открытые сессии и их heartbeat
используют snapshot; profile edits/archive не меняют role/leases/результаты и
не закрывают чужую работу. Для отзыва доступа применяется registry, не архив.
Старые sessions остаются без profile. Independent review по session IDs/SHA/
generation сохраняется, в том числе при одном общем profile у автора и reviewer.

## REST/MCP/CLI и браузер

Native операции `profiles` (state?), `profile` (profile_id, version?; без version
текущая), `profile_create`, `profile_update`, `profile_archive` используют общий
store. Mutations содержат request_id; update/archive — profile_id/version.
REST `/api/v1/p/{id}/<operation>` и CLI `--project` сохраняют прежние
scope/Origin/size/auth границы. В MCP сохраняются12tools: profile reads
доступны resources (list/current/revision), использование — существующий describe;
CRUD — через browser/REST/CLI. Legacy aliases остаются default.

Browser JSON: GET `/web/p/{id}/profiles` и `/profiles/{profileID}`; PageData
`profiles`, `profile`, `profile_revisions` (метаданные версий без дублирования
инструкций), `can_manage_profiles`. POST `/profiles`, `/profiles/{id}/update`,
`/profiles/{id}/archive` принимают CSRF и issued action_id. Keys actions:
`profile:create`, `profile:update`, `profile:archive`. Archive поле state;
update содержит полную форму. Validation400 выдаёт новую capability при
исправимом payload; неизвестный результат сохраняет исходную отправку.
Recovery pending actions включает новые операции. Redirect — исходный scope.

Next показывает «Агенты» с понятными отдельными входами «Профили» и «Сессии».
Список/детали/форма/архив реально работают с backend; сессии отображают
привязанный ID/version/name только если связь сохранена сервером. Нет fake
online/process/model/cost и кнопки запуска. Читатель видит настройки без
доступных mutation controls. Query keys, drafts, cancel/late response и retries
сохраняют TASK009 project boundaries. Стиль и компоненты Paperclip/Next прежние.

## Миграция и эксплуатация

Board schema2→3 добавляет profile/revision tables без изменения старых строк;
schema1 продолжает прежнюю последовательную migration. До DDL — private durable
snapshot; version3 с отсутствующими tables отвергается, не чинится CREATE IF
NOT EXISTS. Published storage preflight принимает строго полную2или3; неизвестные
версии failclosed до web orphan cleanup. Registry1/web2 не меняются.

Capabilities объявляют `agent_profiles:1,board_schema:[3]`; permanent boot guard
перед первой migration требует новый floor. Оператор TASK009 initial apply
не переиспользуется. Новый capable update/roll-forward сохраняет registry/ACL,
default и все projects, recovery markers, эффективный argv с projects-data,
полный schema2/3 backup/restore, immutable bundles и owned PID/ELF SHA.
Не выполнять downgrade/board-only restore поверх новых данных. В случае
неуспеха durable startup hold и остановка служб сохраняются для roll-forward.

Проверки: авторские unit/integration → независимый source APPROVE → отдельный
tester process/браузер и review новых тестов → Linux synthetic migration/update/
recovery/full restore → production backup/update/HTTPS/preservation evidence.
Физический второй ПК и полный V1 остаются NOT-RUN.
