# Мультипроектный контракт Accord — TASK009

20.10.2026. Принятый контракт после независимых WORK135/136/137; реализация и
фактические проверки фиксируются отдельно в TASK009. Существующие инварианты
core/API сохраняются. Это расширение прежнего single-board режима, не OAuth
и не выполнение кода из подключённых репозиториев.

## Данные, идентификация и права

`serve --projects-data PRIVATE_ROOT`, либо `ACCORD_PROJECTS_DATA`, включает
registry. Без этой настройки прежний single-board режим сохраняется. Registry
`PRIVATE_ROOT/registry.sqlite`, schema1; board schema2 сохраняется. Web schema2
добавляет epoch действий миграцией с schema1 без сброса прежних строк и сессий.
Fresh registry создаётся только в действительно новом пустом private root.
Существующий marker/DB/sidecars не являются разрешением инициализировать новую
identity authority: missing/empty/corrupt schema/meta/key/memberships дают отказ.
До default Store/web constructors в multi-project режиме проверяются прежние
bound DB, а published managed DB — до открытия runtime. Missing/empty board/web
или обязательные таблицы не заменяются пустыми хранилищами; orphan cleanup не
должен удалять прежние blobs из-за потери web metadata. Новое unpublished
provisioning создаётся отдельно; single-board bootstrap этим не подменяется.
`registry_meta(id=1,default_id,default_db,default_web)` фиксирует прежние пути.
Несовпадение default binding при повторном старте отклоняется, а не создаёт
пустую замену. Default DB, web/blobs и их содержимое не перемещаются.

`Project{id:p_<32hex>,name,state,version}`: state provisioning/active/archived.
ID неизменен; новые board/web находятся в `PRIVATE_ROOT/projects/{id}/`.
Браузер не задаёт DB path, upstream или credential другого проекта.
Новые проекты публикуются active только после готовности storage и owner grant.
Долговечный pending create/retry восстанавливает тот же ID после сбоя.

Identity — отдельный стабильный идентификатор конкретного credential principal.
Legacy identity связана с ID прежнего default token; catalog identity имеет
свой hash/expiry/revoked в registry. Owner/machine — подписи, не ключ объединения.
Catalog token выдаётся локальной административной CLI в новый private file;
участник сообщает только несекретный identity ID для приглашения. Токен нового
B-only участника не записывается в default token store и не даёт default access.

Membership связывает identity, project, local principal и роль owner/editor/viewer.
Project-local secret не выдаётся клиентам. Legacy default использует прежний
principal ID. Один credential может иметь разные права в разных проектах.

| Роль | Возможности |
| --- | --- |
| owner | Чтение/действия core по его session/lease правилам; сведения проекта, репозитории и участники |
| editor | Чтение и действия core; без управления проектом и ACL |
| viewer | Чтение; вход/выход и явная human read-session; без claim/task/file write и singleton architect |

Project role не заменяет native role/session/fencing/SHA gates. Архив читается
участниками, business writes запрещены; owner может управлять метаданными и
восстановить проект. Архив с живыми leases не создаётся. Default архивировать
нельзя, чтобы сохранять прежние адреса и клиентов. Последний owner защищён.

Явная policy TASK009: любая действительная identity может создавать собственные
проекты и становится owner только там; это не global administrator. Начальные
границы:32проекта всего,8созданных на identity,16репозиториев/128участников на
проект,2одновременных provisioning, catalog request64КиБ,2048action на browser
session с ограниченным retention. Эти границы не являются нагрузочным benchmark.

Legacy tokens сохраняют прежний default editor доступ. Начальный импорт может
материализоваться при первом authenticated request: это оформление уже бывшего
права, не новый grant. Explicit revoke/tombstone никогда не заменяется автоимпортом.
Default owner назначается выбранному ID явной локальной admin-командой; все
старые токены не повышаются до управляющих автоматически.

## Проверки доступа, отзыв и транзакции

Каждая Store operation, design reserve/final publication и cached replay сначала
проверяет current credential, membership, role и project state. Порядок коротких
guards: registry → legacy credential authority при необходимости → selected store.
Для default нет вложенной транзакции того же one-connection Store. Guard не
держится вокруг HTML/body parsing,120s upload или file streaming. Перед final
publish права проверяются заново; отказ освобождает staging/reservation.
Побайтный отзыв уже начатого разрешённого download не обещается.

Registry guard сериализует защищённые commits начального экземпляра; отдельные
SQLite WAL файлы не дают общей распределённой atomicity. Недоступные staged
dirs/orphan principals не превращаются в доступные проекты. Audit и durable
registry response фиксируются с мутацией. Idempotency проверяется после auth.

Revoke увеличивает membership access_epoch. Он никогда не уменьшается при
повторном grant. Native session_epochs и project browser binding проверяются
до actor operation/replay: старые session IDs не оживают после regrant.
Principal остаётся прежним для сохранения requester/provenance прав. Только
начальный migrated default epoch0 допускает старые untaged sessions и исходный
Describe request ID; при epoch>0 Describe получает HMAC namespace с приватным
registry key, project/identity/epoch/operation/client ID. Клиент не может заранее
создать legacy request ID, совпадающий с будущим scoped Describe.
Повтор в том же epoch восстанавливает потерянный ответ/регистрацию session tag.
Project-targeted catalog action IDs также привязаны к эпохе выдачи.
Downgrade viewer может сохранить чтение human session, но запрещает writes и
их cached replay; отдельного обещания постоянной инвалидации после repromotion нет.

## Browser identity и несколько вкладок

Сохраняется одно identity cookie прежнего имени/значения:
`__Host-accord_session` (development `accord_dev_session`), Path `/`, HttpOnly,
Secure для HTTPS, SameSite Strict. Registry хранит global hash/identity/CSRF/expiry,
каждая web.db — отдельный binding того же hash к своему principal/CSRF/actions/session.
Cookie выбранного проекта нет; scope определяет URL.

Старая default cookie может быть импортирована без замены board session и без
Describe, только после проверки raw default binding и отсутствия global tombstone.
Global expiry не превышает прежнюю browser expiry и credential expiry. Login
принимает legacy/catalog credential; B-only login не читает default данные и
не создаёт default session. Identity без memberships открывает каталог.

Любой logout, включая прежний `/web/logout`, сначала фиксирует global tombstone,
затем удаляет fallback binding/сбрасывает cookie. Старое cookie не импортируется
снова. Отзыв членства проекта A не разлогинивает разрешённый проект B.

Только явный login/switch POST открывает отсутствующую read-session. GET не
делает Describe/heartbeat. При deep link без project binding JSON409
`PROJECT_SESSION_REQUIRED` содержит разрешённые project/target для явного входа.
Switch сначала разрешает pending actions по исходной сессии, затем при необходимости
создаёт новую; неизвестный результат не становится повтором с новым ID/проектом.

## URL и API

| Интерфейс | Явный project scope |
| --- | --- |
| Next страницы | `/app/p/{id}/...` |
| Browser JSON/actions/files | `/web/p/{id}/...` |
| Native REST | `/api/v1/p/{id}/...` |
| MCP | `/mcp/p/{id}` |
| CLI/stdio bridge | `--project ID`, закреплённый клиентом |

Legacy `/app`, `/web`, `/api/v1`, `/mcp` остаются default aliases и в multi-project
режиме также проверяют default ACL. Неявный выбор проекта не меняет их смысл.
Host/Origin/admission/auth perimeter действует и на scoped native/MCP/resources.
Scoped HTML pages направляются в Next: прежние hardcoded SSR forms не обслуживают
nondefault board. Binary preview/download остаются авторизованными scoped URL.
Scoped redirects/asset URLs строятся структурно; не regex заменой HTML/текста.

`GET /web/projects`: `{ok:true,page:{name:"projects",registry:{identity:{id,owner,
machine},csrf,actions,projects:[ProjectView]}}}`. Здесь нет поддельной core session.
Каждая scoped board page добавляет `project:ProjectView`.

`GET /web/catalog/projects/{id}`: page name `project-settings`, registry identity/
csrf/actions, project и owner-only members. Members для nonowner отсутствуют.

`ProjectView={id,name,state,role,can_write,can_manage,can_archive,version,repositories}`.
`can_archive` разрешён только owner активного nondefault проекта; live leases
дополнительно проверяются перед фактической сменой состояния. Default не
показывает заведомо неисполняемое архивирование.
`Repository={id,url,branch,version}`; `MemberView={identity_id,owner,machine,role,version}`.

Все catalog POST требуют same-origin, global CSRF и server-issued durable action_id:

| POST path | Поля сверх csrf/action_id |
| --- | --- |
| `/web/catalog/projects` | name, repository_url?, branch? |
| `/web/catalog/switch` | project_id, target? в allowlisted local scope выбранного проекта |
| `/web/catalog/projects/{id}/update` | name/state/version |
| `/web/catalog/projects/{id}/repositories` | url, branch? |
| `/web/catalog/projects/{id}/repositories/{repoID}` | action:update/remove, version, url?/branch? |
| `/web/catalog/projects/{id}/members` | action:grant/set/revoke, identity_id, role?, version? |
| `/web/logout` | global logout capability; прежний project-CSRF путь также сохраняется |

Ключи registry.actions: create, logout, switch:{projectID}; detail: update,
repository:add, repository:{id}:update/remove, member:grant,
member:{identityID}:set/revoke. Нельзя отправлять client-chosen actor/DB/upstream.
Неизвестные и повторные поля отклоняются. Hidden/unknown чужие ID одинаково404;
response/DOM/log не содержат secrets. Version конфликт не маскируется success.

Repository CRUD сохраняет безопасную credential-free HTTPS/SSH Git reference и
необязательную branch; local/file/exec protocols, пароли/query/fragment отклоняются.
Это связь, не claim удалённой доступности/clone. Репозиториев может быть несколько.

Opaque cursors включают project namespace, в том числе control/notes/design
lists. Legacy default namespace сохраняет старые cursors; cursor из A не
принимается в B. Numeric event offsets остаются локальными offsets своего DB.
Cache/query keys, placeholders, forms, commands, polling и previews включают
scope. При switch старые reads отменяются, данные/drafts A не появляются в B.
Captured mutation path/body не изменяется после отправки. Pending/uncertain
mutation не переносится переключателем; остаётся retry исходного действия.

## Эксплуатация и восстановление

Перед production migration: согласованный snapshot прежнего state, проверка
восстановления на synthetic fixture, registry binding и preservation evidence.
Новый snapshot включает registry, default board/web и все managed project dirs,
включая recovery state provisioning. Writer остановлен; SQLite integrity/schema,
manifest exact files, design references, fsync/private modes проверяются.
Неизвестные private entries и неразрешённые WAL/rollback journals опубликованных
БД отклоняются до публикации копии. Unpublished provisioning сохраняется raw.
При offline restore после удаления actions/смены HMAC такие pending проекты
отклоняются административно: storage перемещается в private `quarantine/{id}`,
registry rows удаляются, directory entries fsync. Quarantine не маршрутизируется
и не выдаётся за usable board. Все последующие полные snapshots/restore также
сохраняют эти raw recovery sets (строгие ID/file allowlists, максимум128 до
ручного обслуживания). Опубликованные проекты и пользовательские записи не
удаляются при этой операции.
Для обычного crash recovery без полного restore также нужен доверенный offline
отказ от выбранного provisioning: private bytes→quarantine, permanent ID
tombstone, освобождение quota. Старый ещё живой create action не возрождает
отклонённый ID. Эта операция не отзывает credentials/leases опубликованных
проектов и не разрешает удаление active/default проекта. Повтор после потери
питания завершает тот же отказ, не стирает private storage.

В projects root durable `MULTIPROJECT_REQUIRES_CAPABILITY` и независимый
systemd ExecStartPre проверяют выбранный binary через `capabilities --json`:
`project_registry:1,project_acl:1,registry_schema:[1]`. Прежние RESTORE/INSPECTION
guards сохраняются. Старый binary не запускается поверх новых ACL. Rollback
возможен только на capable version; иначе fail-closed/roll-forward, не обход ACL.
Backup с registry нельзя заменить прежним default-only snapshot/script.
Offline restored state отзывает credentials/sessions/actions во всех проектах
до публичного старта. Snapshot не накладывают поверх новых production writes.
Оператор `infra/accord/projects/restore.py` пишет только NEW offline staging,
проверяет неизменность копии и вызывает `projects maintenance-reset`, оставляя
boot guard до нового credential/bootstrap и изолированной проверки. После начала
production cutover только capable roll-forward; durable `activation-started`
после SIGKILL допускает явный resume того же проверенного релиза.

Ни этот контракт, ни unit/mock проверки не являются доказательством миграции,
изоляции или production готовности. Фактические результаты — TASK009 и WORK.
