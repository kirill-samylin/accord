# WORK-135 — backend архитектура TASK-009

20.10.2026. Роль: архитектор и backend developer135 после GO root/critique136.
Статус: Go source и author unit/integration реализованы; независимое source review
ещё не завершено. Владение: эта WORK, cmd/internal Go и author tests.
SSH/VPS/Git/Tandem и реальные данные не читаются/не изменяются.
Ниже сохранено архитектурное решение; итоговый контракт — docs/multi-project.md.

## Прочитанный source и обязательные сохранения

AGENTS, requirements/architecture/api, control-web-architecture, TASK008;
`cmd/agent-board/{main,token,client,mcp}.go`, `internal/store/{store,auth,
web_auth,pagination,catalog,backup,migrations}.go`, `internal/transport/{http,
http_security,mcp,resources,design_resources}.go`, `internal/web/{server,
storage,http,login,browser_json,browser_page,contracts,design_read}.go`.
Исходный board schema2, web schema1; итоговый web schema2 сохраняет старые строки. Store auth повторяется внутри board
write transaction; role/session/lease/SHA gates принадлежат Store. Browser
cookie `__Host-accord_session` Path `/`, HttpOnly/Secure/Strict; token ID в
browser_session — board-local identity, owner/machine не аккаунт.

Предлагается **Go registry + изолированные board/web DB на проект**. Существующие
board.db/web directory остаются на прежних путях и привязываются к default
project. Их задачи, токены, sessions/leases/generation, replay, blobs не
копируются/не пересоздаются. Новые boards лежат только в operator-managed
private projects root, пути никогда не принимаются из browser payload.
Board schema2 сохраняется; web migration1→2 добавляет issued-action epochs и
атомарную cleanup trigger, не пересоздавая browser bindings/sessions/actions.

## Сущности и identity

- `Project{id:"p_"+32hex,name,state:active|archived,created_at}`; ID opaque,
  неизменен и никогда не переиспользуется; default ID фиксируется один раз
  в registry. Имя не path и не источник авторизации.
- `Repository{id,project_id,url,branch?}`: несколько записей, create/update/remove,
  version/concurrency check. Допускается credential-free HTTPS Git URL либо
  явный SSH Git ref; нет query/fragment/password, local/file/exec protocol.
  Это сохранённая привязка, **не clone/access verification**. Никакого сетевого
  fetch, исполнения Git/hook, обхода локальных репозиториев сервера.
- `Identity{id,source:legacy|catalog,source_token_id,owner,machine}`. Legacy ID
  выводится из существующего default token ID с namespace; совпадение owner
  или machine не объединяет identities. Catalog credentials хранят только
  hash/expiry/revoked в registry и не создают default board token.
- `Membership{identity_id,project_id,role:owner|editor|viewer,local_principal_id,
  active,version}`. New catalog identity не имеет доступа к default без
  явного grant. Даже admin-доступ к catalog не равен чтению всех boards.
- Каждый authenticated identity вправе создать собственный проект; получает
  owner только созданного проекта. Это исключает необходимость скрытого
  повышения старых токенов до globaladmin. Owner добавляет/меняет/удаляет
  участников, репозитории, name/state. Последнего owner нельзя удалить или
  понизить; self-revoke разрешён только при другом owner.
- Существующие legacy identities сохраняют default editor membership. Первый
  import создаёт durable grants; explicit revoke/downgrade оставляет tombstone
  и никогда не переписывается автоимпортом. CLI-created legacy tokens после
  миграции могут auto-import только default при первой auth, если записи ещё
  нет. Default owner назначается явной локальной admin-командой выбранному ID,
  не всем старым токенам и не первому вошедшему автоматически.
- New catalog token создаётся локальной CLI в новый private file (как нынешний
  token create), identity ID выдаётся без secret для приглашения/назначения
  членства через UI. OAuth/новая система паролей не вводятся.

Project-local principal создаётся server-side через существующий Store token
bootstrap, его случайный secret немедленно уничтожается и никому не отдаётся.
В membership сохраняется только local ID. Legacy identity в default использует
свой прежний ID. Внешний запрос всегда сначала проверяет account credential,
затем membership, затем локального actor; локальный ID не является credential.
Невозможен обход ACL через native API, MCP resource или старый `/web/`.

## Роли и линейная авторизация

Owner/editor могут выполнять разрешённые core операцией действия; project роль
не отменяет session role, lease/generation и independent evidence. Viewer может
читать, входить/выходить и явно создавать human read session. Viewer не может
выдавать singleton architect, claim/create/update/transition/release/heartbeat,
upload/delete designs или менять project metadata/ACL. Прежняя editor-сессия
после downgrade не сохраняет права записи. CanWrite/CanManage приходят с сервера.

`WithAccess(ctx,account,project,write,callback(localPrincipal))` обязателен вокруг
каждого реального Store operation, design reserve/final SQL publication и
registry mutation. Проверка выполняется **до cached action/replay response**.
Порядок блокировок: registry short write transaction → (для legacy credential
в nondefault операции) default Store credential guard transaction → выбранный
Store/domain transaction. Default операции не входят в default Store повторно:
существующий core transaction сам перепроверяет credential. Read-only serving
тоже перепроверяет current credential/membership, но не renew сессию.

Запрещено держать эти SQL locks вокруг HTTP handlers, чтения body, HTML render,
120s uploads/download streams. Upload резервирует отдельно; final публикация
повторно проверяет ACL/credential и не публикует файл после revoke/downgrade.
При отказе освобождаются reserved/staging bytes. До открытия file stream нужна
актуальная авторизация; начатый до revoke stream не обещается отозвать побайтно.

Это **не распределённая cross-DB транзакция**. Grant сначала создаёт inert local
principal, затем последним публикует membership; crash может оставить только
недоступного orphan principal, не неавторизованный доступ. Revoke/change роли
линеаризуются под тем же registry guard, не расходятся с in-flight business commit.
Legacy token revoke через прежнюю CLI сериализуется default credential guard.
Short global guard сериализует protected commits начального экземпляра; это
не доказательство большой нагрузки/горизонтального scale. Нужен test contention.

## Browser и маршрутизация — одно identity cookie

Root выбрал single identity cookie с сохранением текущих name/value. Registry
хранит global browser hash→identity/expiry/CSRF; отдельная project web DB хранит
тот же hash→project-local principal, собственные CSRF/actions и board session.
Нет current-project cookie. URL проекта определяет scope; две вкладки независимы.

- Canonical Next `/app/p/{id}/...`, Go `/web/p/{id}/...`.
- Native `/api/v1/p/{id}/...`, MCP `/mcp/p/{id}`. CLI/stdio bridge получают
  `--project ID`; endpoint определяется оператором, не model tool payload.
- Старые `/app/**`, `/web/**`, `/api/v1/**`, `/mcp` pinned default. Любые запросы
  к legacy aliases при registry-enabled режиме всё равно проходят default ACL.
  New catalog B-only token не получает default по отсутствующему project ID.
- `/web/login` обслуживает global broker. Legacy или catalog credential создаёт
  global browser auth без обязательного обращения к default. B-only identity
  открывает первый доступный B; identity без memberships открывает catalog.
- Только explicit login/switch POST может создать отсутствующий project binding
  и human Describe. GET не продолжает session/lease и не меняет membership.
  Существующий default browser binding/session/hash/CSRF используется как есть.
- Legacy cookie compatibility: только отсутствующий global record и проверенный
  raw default browser binding могут импортироваться, сохраняя hash. Некорректный,
  дублированный, revoked/tombstoned global cookie не fallback-ится в default.
- Любой logout (`/web/logout` тоже) глобально tombstone-ит cookie hash, удаляет
  global session и default fallback binding, сбрасывает cookie. Оставшиеся
  project bindings не дают доступа без global auth. Повтор украденного старого
  cookie после logout не восстанавливает global session. Project membership
  revoke не разлогинивает разрешённые другие проекты.

New scoped HTML GET redirects на эквивалентный `/app/p/{id}` (кроме бинарных
uploads/download/preview и shared static). Нельзя отдавать прежний SSR с его
hardcoded `/web/` forms/links для nondefault проекта: это написало бы в default.
Default legacy SSR сохраняется. JSON redirects и DesignAsset download/preview
URL строятся структурно с project prefix; текст/HTML не переписывается regexp.
MCP metadata URL также project-qualified.

## Предложение JSON contract для согласования root/frontend137

`GET /web/projects` — global browser catalog; не открывает default board session.
Envelope `{ok:true,page:{name:"projects",registry:{identity:{id,owner,machine},
csrf,actions,projects:[ProjectView]}}}`. Здесь нет фиктивного `core.SessionView`.
`ProjectView={id,name,state,role,can_write,can_manage,repositories:[...]}`.
Каждый scoped board page добавляет `project:ProjectView`; каталог можно читать
одним отдельным request, не дублировать полный registry во все polling ответы.

Catalog forms (`Accept:application/json`, same-origin, CSRF, durable action ID):

- `POST /web/catalog/projects`: name, optional repository URL/branch; create own.
- `POST /web/catalog/switch`: project_id, optional safe section; checked redirect
  `/web/p/{id}/...`, materialize read-session only explicitly if missing/expired.
- `POST /web/catalog/projects/{id}/update`: name/state/version.
- `POST /web/catalog/projects/{id}/repositories`: add URL/branch.
- `POST /web/catalog/projects/{id}/repositories/{repoID}`: update/remove/version.
- `GET /web/catalog/projects/{id}`: settings project + registry CSRF/actions;
  owner-only members с identity ID/label/role/version, repos видны участникам.
- `POST /web/catalog/projects/{id}/members`: identity_id/role/version, grant/update/revoke.

Мутации пишут audit и idempotent response в registry transaction. Action key
identity+global browser hash+operation+target project, canonical payload включает
ID и version. Repeated same request returns same result only after current auth;
unknown outcome сохраняет request/action ID. Hidden/unknown project/member IDs
отвечают одинаковым404 без проверки existence через чужие данные. Body unknown/
duplicate fields reject; bounded lengths/list limits и no secret outputs.

## Cursor, URL и локальные scope

Существующие list cursors filter-hash не включают проект (`store/pagination.go`),
поэтому их нельзя переносить напрямую. Scoped Store получает неизменный project
cursor namespace, включаемый в filter hash во всех list/control/notes passes.
Design cursor hash также включает ID; default legacy namespace остаётся прежним
для старых cursors. Numeric event cursor остаётся локальным DB-offset endpoint,
не credential или cross-project snapshot. Actions/replays/notes/leases/events/
designs/audit естественно разделены DB + явным immutable routed project ID.
Ошибочный source project cursor в B —400, не «успешный другой pass».

## Provisioning, migration, archive и backup

Registry file новый schema1/private0600, managed root0700 без symlink ancestors.
Проекты сначала `provisioning`, пути server-derived staging, board/web готовы и
проверены; последняя registry transaction публикует active + creator owner.
Crash/retry одного durable create action завершает тот же ID, не второй проект.
Orphan staged dirs не доступны router; recovery cleanup только managed имена.
Default binding импортируется идемпотентно один раз, не копирует DB/attachments.
Архивирование не удаляет данные: чтение владельцам/участникам сохраняется,
новые writes/claims запрещены; живые leases должны не потерять history. Предлагаю
отклонять archive при живых leases, чтобы не оставлять их без release/heartbeat.
Восстановление active разрешено owner. Default archive запрещён ради legacy aliases.

Backup должен остановить sole Go writer под прежним maintenance lock и включить
registry DB + default board/web + все published project boards/web/blobs/staging
в один проверенный snapshot manifest. Dynamic paths — исключительно allowlist из
registry IDs, не произвольные filesystem пути. Проверить schemas/integrity,
manifest exact files, ready-design refs, fsync; оставить512MiB reserve, private
modes и прежние timers. Restore/drill выполняются на synthetic fixture; реальный
rollback binary не восстанавливает старые DB поверх новых writes. Legacy binary rollback сам по себе **небезопасен**: старый код проигнорировал бы
новые default ACL и global logout tombstones. До активации registry root устанавливает
независимый systemd ExecStartPre capability guard и durable
`MULTIPROJECT_REQUIRES_CAPABILITY` marker. Guard проверяет выбранный binary через
новую `agent-board capabilities --json`; старый binary не проходит и не выходит
в публичный listener. Guard не отменяет RESTORE/START_REQUIRES_INSPECTION. После
активации допустим только rollback на registry/ACL-capable binary; при отсутствии
такого релиза fail-closed остановка/roll-forward, не возвращение legacy auth.
Аналогично backup unit проверяет capability/format нового all-project snapshot:
возврат старого default-only backup недопустим. Real DB restore не является
исправлением downgrade. OS root/manual bypass остаётся вне модели доверия.

## Принятое ownership после APPROVE/GO

Backend135: новый `internal/projects/**`, project Store guard + cursor namespace,
web binding/identity/guard helpers + structural URLs, native/project router,
CLI/bridge --project и registry CLI, авторские unit/integration tests.
Root: TASK009/canonical API/architecture/Git, backup/infra отдельным явным owner
или расширением135; frontend137: только web/src и клиентские unit.
Reviewer136 независим от backend author. Новые process/E2E — отдельный tester
после source APPROVE, также test review. Production/VPS actions только root.

Обязательные negative checks: B-only login безdefaultDescribe/read; unscoped
native denied для B-only; same task key/actions/cursor/file IDs/session fromA
cannotread/writeB; denied/revoked/downgraded membership before replay; viewer
writes denied incl upload finalization; project switch concurrent tabs/pendingA
submission; old cookies/tokens/data/leases survive; logouttombstone no fallback;
ACL/revoke vs corecommit races; crash grant/create/retry; snapshot restoreallboards.

Первичные источники проверены20.10.2026: SQLite transactions
https://www.sqlite.org/lang_transaction.html (read snapshot не блокирует WAL writer;
нужен write guard для линейной revocation), Go net/http Cookie
https://pkg.go.dev/net/http#Cookie (Path/Secure/HttpOnly/SameSite).
Зависимости/toolchain не обновляются. Production/process/browser QA — NOT-RUN
автором; Go developer unit/integration проверки перечислены ниже.


## Фактическая реализация и review checkpoint

`internal/projects/`: registry schema1, identities/credential authority, explicit
membership/ACL, project runtime/router, catalog/repo/member CRUD, global browser
broker, durable actions и offline recovery. Default DB/web paths bind once;
legacy aliases pinned default. Новый catalog bearer не получает default grant.
`core.ProjectView` содержит role/can_write/can_manage/can_archive; последний false
для default/nonowner/archived, restore owner разрешён отдельно.

Revoke увеличивает membership access_epoch; stable local principal сохраняет
requester/history. session_epochs проверяются перед каждым core/cached replay;
untagged допускается только legacy/default/epoch0. Epoch>0 Describe request ID —
HMAC secret registry key(project,identity,epoch,clientRID). Same-epoch lost response
восстанавливает регистрацию; старый tag никогда не перезаписывается. Issued web
и project catalog action IDs также epoch-bound, в том числе ещё unbound actions.

Short registry write gate сериализует authorization и selected board transaction;
legacy source дополнительно guarded raw default credential authority. Default
web-file guard также rechecks source credential. Global cookie hash/identity
проверяется в каждой последующей короткой transaction: in-flight browser request
не начинает commit после logout. Streaming/rendering/network lock не удерживают.
Upload reserve/replay/finalize проверяют WRITE + исходный SessionID/epoch в том
же gate. Viewer не получает core writes/architect Describe, включая cached replay.

Catalog payload/result bind и business mutation повторно checked в одной registry
transaction: concurrent same action не создаёт два repository. Transient503 явно
retry_same_action:true; pending create/Describe повторяется с прежним action/body.
URL credentials отклоняются до durable bind. Scoped HTML POST без JSON Accept
отклоняется406; GET HTML ведёт в Next. JSON redirects, browser/native/MCP design
URLs структурно scoped. Разрешённые validation pages сохраняют verified Project.

Web schema2 включает web_action_epochs и delete trigger без orphan rows при
prune/logout/expiry. Catalog browser metadata чистится на login/catalog issuance;
живые tombstones сохраняются до expiry, native session_epochs не очищаются.
32 projects/8 created per identity/16 repos/128 members/2 pending provisions;
64KiB catalog body,2048 catalog actions/session,128 quarantine recovery sets.

Операционные CLI: capabilities --json; projects token-create/token-revoke,
bootstrap-owner (raw legacy token ID, same-owner retry idempotent), recover-owner
(trusted offline single project/identity), maintenance-reset --reason restore.
Reset rebinds staged paths, revokes restored credentials/browser/core sessions,
rotates HMAC/increments epochs, сохраняет tasks/history/design assets.
projects reject-provisioning --reason provisioning-recovery сохраняет raw pending
bytes в quarantine/{id}, fsync parents, удаляет pending rows и пишет durable
rejected_provisions tombstone. Missing folder оставляет empty durable recovery set.
Create также fail-closed на quarantine entry до SQL tombstone: crash после rename
не resurrect старый live action. Published/default reject запрещён; повтор
rejection идемпотентен. Никакой usable board из quarantine не маршрутизируется.

Авторские tests: B-only native/browser/catalog/logout; default ACL tombstones;
revoke/regrant sessions/actions; viewer cached writes; upload stale epoch/downgrade;
concurrent catalog exactly-once; transient create/switch recovery; safe repo URL;
same task/asset IDs, foreign session/task/design cursors; schema1 cookie/hash/CSRF/
session import без Describe + tasks/live lease/generation/evidence/native replay;
offline reset/quarantine/recover-owner/bootstrap idempotency; pending rejection
capacity/bytes/missing-folder/rename-before-SQL crash; metadata expiry lifecycle.

Developer validation logs: `.local/task009/backend-check.log` и последующий
`backend-authority-check.log` (fmt/vet/test/race, без bin rebuild по root).
Новые process/E2E пишет отдельный
QA после независимого source APPROVE. До этого TASK009 не объявляется завершённой.
Все новые Go files ≤300 строк; большой существующий design_http сохранён ниже300
выносом неизменного deadline adapter в files_deadline.go.

Post-APPROVE bounded amendment: marker/existing authority не инициализируется
заново при missing/empty/corrupt DB/meta/critical tables. Fresh init — только
пустой root; read-only preflight проверяет authority и published board/web metadata
до default constructors и published runtime, сохраняя blobs при потере web.db/
design_assets. Regression проверяет startup/runtime отказ без recreate/delete,
revoked default membership, incomplete activation; singleboard path не меняется.
