# WORK136 — независимый контрактный и security review TASK009

20.10.2026. Reviewer/архитектурный критик, не автор реализации. Владение:
эта WORK и ignored `.local/task009-review/`. Source, TASK, Git, SSH/VPS и
реальные данные reviewer не изменяет. Прочитаны AGENTS, requirements/API,
architecture, control-web-architecture, TASK008 и действующие Go handlers,
board/web stores, CLI, MCP resources и backup operators.

## Исходные ограничения по actual source

- `internal/store/auth.go`: principal = ID конкретного machine-token в board.
  Owner/machine — подписи, их совпадение не доказывает одну identity. Нельзя
  объединять principals по имени или переносить право из похожей подписи.
- `internal/store/mutation.go`: native replay привязан к token/session/request,
  проверяет revoke до replay; session ownership/role/SHA/generation rules
  остаются в общем store. Новый project gateway не должен обходить этот слой.
- `internal/web/server.go:69` и `:100`: cookie имеет одно имя, Path `/`, запись
  session_hash и TokenID проверяется в единственной web.db. Несколько Server
  без отдельной привязки cookie/session к project создадут смешение контекста.
- `internal/web/actions.go`: durable action ID связан с session_hash и operation,
  но не project. Проектная изоляция требует отдельной web.db/подписанной границы
  перед dispatch, а не выбранного browser dropdown после загрузки capabilities.
- `internal/store/pagination.go`: cursor содержит filter/high/after, project не
  включён. Межпроектная подстановка cursor должна отвергаться, даже если ключи
  и high-water случайно подходят и пользователь допущен в оба проекта.
- `internal/transport/http.go`, resources/design_resources: legacy REST/MCP
  аутентифицируются через один Service, resource URIs содержат board session.
  Нельзя менять их board вслед за последним browser switch. Новые explicit
  scoped routes должны одинаково охватывать tools, resources, designs, blobs.
- `infra/accord/web/backup.py`: snapshot/restore знает только board.db и web/.
  Registry + все project boards/web/blobs потребуют согласованного inventory и
  snapshot после остановки sole writer, проверки полного manifest и recovery.

## Обязательные условия контракта до реализации

1. Stable opaque project ID, не repository URL/name/path. Существующая board
   становится default без перемещения/пересоздания задач, sessions, leases,
   evidence, token hashes, actions и blobs. Legacy endpoints закреплены за default.
2. Чёткое разделение global identity, project membership и board principal.
   Legacy `legacy:<TokenID>` допустим с прежним default credential authority.
   Старые токены автоматически сохраняют только прежний default доступ;
   создание проектов/управление доступом требуют явного trusted CLI bootstrap.
   Все старые machine tokens нельзя молча повышать до registry administrators.
3. Каждое чтение и мутация проверяет актуальные credential expiry/revocation,
   project membership/status/capability до board/file read и до action replay.
   Global-admin и project-admin — разные полномочия. Удаление/понижение membership
   действует на уже открытые native/browser sessions и ранее выданные action IDs.
4. Viewer/member/admin semantics фиксируются по каждой операции. Describe/login
   могут создавать authentication/session metadata, но viewer не получает claim,
   task/file write или mutation только потому, что core role=human/coordinator.
   Нельзя оставить прямой unguarded project-local AuthenticateID обход gateway.
5. Request project явен и неизменен на всём пути. Cross-tab switch не перенаправляет
   старую форму, загрузку, pending replay или heartbeat в другой board. Query keys,
   drafts, dialogs, aborts и cursor passes клиента включают project context.
6. Repository attachment — безопасная canonical reference, не секрет в URL и не
   произвольный DB/filesystem path. Если доступ не проверяется сетью/worker,
   UI не утверждает clone/credentials verified. Если проверяется — отдельные
   SSRF/redirect/DNS/credential/timeout/output limits и shell-free invocation.
7. Между registry/legacy credential DB/board нельзя обещать общую WAL atomicity.
   Publish-last project lifecycle с durable staging/recovery делает orphan files
   недоступными. ACL grant не виден до готовности inert board principal; orphan
   principal без secret и опубликованного membership не даёт публичного доступа.
8. Race revoke/downgrade vs mutation линеаризуется. Для DB guard подхода фиксировать
   lock order registry → credential authority → selected board. Default board
   нельзя оборачивать вложенной транзакцией того же max-one-connection Store.
   Глобальная сериализация guard — явное ограничение, не доказательство масштабирования.
9. Backup содержит registry и все зарегистрированные stores/web/blobs как единый
   offline snapshot. Restore требует auth reset по всем boards/browser sessions.
   Binary rollback не должен возвращать старые пользовательские DB поверх новых.
10. Старый Go без registry понимает только default tokens и проигнорирует новый
    ACL/tombstone. После активации multiproject stable startup guard вне
    откатываемого release обязан запрещать такую бинарную версию. Rollback не
    возвращает прежний default-only backup operator: новые projects иначе
    останутся вне резервирования. Реальный rehearsal должен включать этот отказ.
11. Global identity cookie может сохранить прежнее имя/значение при строгом
    globalHash→identity→project-principal binding в registry и отдельной web.db
    каждого project. B-only identity входит/видит каталог без Describe/default
    membership. Legacy cookie fallback разрешён только при отсутствии global
    записи/tombstone и после проверки token authority; logout/revoke tombstone
    не позволяет старой default browser row заново зарегистрировать этот hash.
12. Нельзя удерживать глобальный SQLite guard во время120s multipart body или
   50MiB download. Native guards охватывают короткую board transaction; upload
    повторно проверяет именно WRITE capability после stream/inspect перед
    публикацией. `designActor` через read ListTasks не доказывает право editor
    после downgrade→viewer. Linearizable commit требует короткого commit guard,
    иначе контракт обязан честно определить допустимые уже начавшиеся операции.
13. Existing SSR/forms/redirects/download metadata содержат `/web/...`. Scoped B
    route не может отдавать их без project-aware URL generation: shared A+B user
    отправит B-форму в default A. Допустим узкий контракт new-project HTML redirect
    на соответствующий scoped Next UI, сохраняя старый default SSR; JSON redirects
    и binary URLs должны быть структурно scoped, не text-replaced в произвольном
    HTML/JSON body. Native MCP design metadata получает тот же project boundary.

SQLite официально ограничивает atomicity WAL транзакций одним database file:
[ATTACH](https://www.sqlite.org/lang_attach.html),
[WAL](https://www.sqlite.org/wal.html). Cookie `__Host-` сохраняет Secure/Path `/`
без Domain; разные проекты нельзя отделить сменой Path с сохранением этого prefix:
[Set-Cookie](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie).

## Необходимая проверка изоляции

Два проекта с одинаковыми T-/E-/N-keys и сходными metadata; principals с одинаковым
owner/machine, разными IDs; пользователь A-only, B-only, shared, viewer и admin.
Проверить UI/REST/MCP/tools/resources/downloads: чужие key/session/lease/action/
CSRF/cursor/assets, revoked/expired credentials, revoke/downgrade старой сессии,
duplicate/ambiguous project selectors, archived project, mixed tab/draft/retry,
request cancellation при switch, unknown outcome и restart. Неавторизованный
ответ не раскрывает существование чужих задач/названия/путь БД/секреты.

Сохранность default: exact logical state до/после миграции, старый token и browser
cookie/native client, live generation/attempt/evidence, durable pending action,
old key/cursor behavior, restart/crash migration recovery, полный backup/restore.
Новые process/E2E пишет отдельный tester после independent source approve;
автор пишет unit/integration, новые tests проходят отдельный review.

## Статус

Source inspection и ранний security feedback root/WORK135 — PASS.
Выше — contract gates, не утверждение наличия реализации и не финальный APPROVE.
Ожидаются точная proposal WORK135 и canonical TASK009, затем их independent review.
Implementation/unit/process/browser/migration/production checks — NOT-RUN.

## WORK135/137 — bounded architectural APPROVE

Прочитана конкретная proposal WORK135 целиком и frontend boundary WORK137.
Направление **APPROVE для реализации после фиксации root canonical TASK009**:
изолированные board/web DB, registry identity/ACL, один identity cookie с
globalHash/project-principal bindings, B-only login без default Describe,
legacy aliases всегда под ACL, short guards, structural scoped URLs,
publish-last recovery, all-project backup и независимый capability startup guard.

Root переданы четыре уточнения для freeze, не скрытые новые product choices:

1. WORK135 предлагает всем authenticated identities создавать собственные projects.
   Это не global administrator, но новая resource capability: root явно выбирает
   policy и фиксирует hard project/provision/list limits; нельзя обещать unbounded
   scale по SQLite registry/1ГиБ VPS или доверять только client-side limits.
2. On-demand import нового legacy token может материализовать прежний default
   grant на GET. Это противоречит буквальному «GET не меняет membership» в той же
   proposal. Canonical contract должен назвать безопасное compatibility exception
   (только уже существовавшее default право, tombstone сохраняется), либо CLI
   запишет grant раньше. Ни одна ветвь не даёт catalog-token доступ к default.
3. При compatibility import глобальный cookie/session expiry не больше старого
   browser expiry и credential expiry; GET не продлевает срок или board session.
   Logout сначала durable tombstone, затем удаление fallback binding/cookie.
4. Scoped direct GET без project board binding возвращает authenticated
   continuation gate без Describe. Только explicit switch/login/continue POST
   создаёт binding/session. Перед ensureSession в целевом проекте восстанавливаются
   durable pending actions с их исходными session IDs; нельзя открыть новую
   сессию прежде unknown-result recovery. Другие project sessions не трогаются.

Это ограниченное одобрение архитектуры, не source/unit/runtime verdict. Автор
и root продолжают реализацию в согласованных ownership boundaries; независимый
review кода, новых tests и evidence остаётся обязательным. Зависимости не меняются.

## Принятое root уточнение: revoke/regrant epoch

Root выбрал стабильный local principal и отдельный `access_epoch`: revoke
повышает epoch, regrant не уменьшает его, Describe отмечает session→epoch,
каждый native actor/browser binding проверяет текущий epoch до cached replay.
Это сохраняет Requester/provenance и закрывает старые session capabilities.
Downgrade сохраняет чтение, но запрещает write/replay текущими permissions;
не обещается постоянное уничтожение read session при временном viewer downgrade.
Направление **APPROVE** с тремя обязательными implementation checks:

- Untagged session fallback допускается только legacy identity в default при
  исходном epoch0; никогда в новом проекте или для catalog identity.
- Проверяется epoch выданного action и его исходного `a.SessionID`, не только
  свежего `b.Session`. В `runAction` cached `a.Result` должен идти после этой
  проверки. Старый unbound action тоже нельзя привязать к новой epoch1 session
  с прежним cookie hash: ему требуется issued epoch. Legacy unmapped action
  compatibility ограничена default initial epoch0.
- Регистрация session epoch не перезаписывает конфликтующий старый epoch.
  Unkeyed hash нового Describe request ID сам по себе не предотвращает
  prestaging: клиент до revoke мог использовать будущий effective ID как raw
  legacy request ID. Для pre-mode untagged session cache необходимо заранее
  отметить все старые session IDs epoch0, либо использовать server-secret HMAC
  для нового Describe namespace (project/identity/epoch/client request). Retry
  после commit/tagging failure в той же epoch остаётся recoverable и не создаёт
  вторую board session.

Это уточнение передано root/WORK135 до implementation review. Source и tests
будут проверены независимо; runtime approval по этому контракту не выдавался.

Canonical `docs/multi-project.md` независимо прочитан после root freeze:
policy create-own и hard limits32/8/16/128/provision2, legacy materialization
exception, expiry bounds, HMAC namespace, direct-link continuation, archive
owner metadata, all-project snapshot и capability guard зафиксированы.
Прежние architecture clarifications получили явные dispositions; **APPROVE
контракта для реализации подтверждён**. Отдельно root/автору передано, что epoch
выдачи охватывает также unbound web actions, не только catalog actions. Source
в момент чтения ещё WIP; нет ни source APPROVE, ни заявления о runtime PASS.

## Incremental source review: infra и registry (WIP)

Проверены `infra/accord/projects/{guard,layout,validate,snapshot,backup,render}.py`,
оба systemd drop-in и текущие Go registry/auth/access/runtime/catalog/browser
guards. Безусловный `require_capable` и постоянный ExecStartPre не зависят от
наличия activation marker: исчезнувший marker не разрешает старый executable.
Snapshot format2 включает default, registry, managed projects и raw unpublished
provisioning; private modes, manifest exact file set и self-contained SQLite
проверяются до публикации. Это source observations, не runtime evidence.

Переданы root два обязательных infra finding: immutable read не должен
игнорировать nonempty rollback journal; active board/web inventory не должен
молча пропускать неизвестные исходные файлы. Root сообщил об исправлениях и
author regression units; независимое повторное чтение и запуск ещё впереди.

Переданы backend135 следующие implementation findings:

- P1: `bindCatalog` и бизнес-мутация разделены transactions; concurrent identical
  `repository:add` action может создать две записи. Fresh auth/epoch и result
  replay должны повторно проверяться внутри business transaction до записи.
- P1: final upload guard проверяет текущую membership без исходного session ID;
  revoke+regrant между `designActor` и `finish` позволяет старой epoch опубликовать
  upload. Исходный actor/session проверяется под тем же коротким guard, что
  публикация, без удержания registry lock во время чтения тела.
- P2: regrant inactive membership требует скрытой tombstone version, но members
  выдаёт только active rows и generic grant форма версии не знает. Нужен
  доступный владельцу regrant flow. Regrant также должен соблюдать MaxMembers,
  проверяемый сейчас только для отсутствующей membership.

Root отдельно получил integration check: restore auth reset не должен оставлять
  provisioning без исходного recoverable action mapping, если actions/key
  сбрасываются. Нужна явная offline finish/reject policy; pending private файлы
  нельзя терять. Backend и infra всё ещё изменяются; **source verdict NOT-RUN**,
  product APPROVE не выдан. Новые process/E2E тесты reviewer не создавал.

Повторное чтение infra fixes и author unit source завершено. Независимый запуск
`python3 -m unittest discover -s tests/infra -p 'test_projects_*.py' -v`:
**18/18 PASS**. Unknown source storage fail-closed, unresolved WAL/journal,
missing-marker capability rejection, pending raw storage, checksum/private modes
и bounded ingress transformation проверены. Это unit/synthetic evidence;
реальные systemd/nginx/VPS и restore rehearsal **NOT-RUN**.

Дополнительные findings переданы авторам до freeze:

- Global `/web/logout` ошибочно зависел от default ACL для B-only identity.
- Scoped HTML POST отдавал legacy unscoped redirects/forms; согласовано
  отклонение до business path, HTML GET остаётся Next redirect.
- Scoped definitive 4xx без project metadata ошибочно превращался клиентом в
  uncertain result и постоянный lock. Frontend137 согласовал удаление
  непомеченного rejected page перед merge; настоящий foreign ID по-прежнему
  отвергается. Backend добавляет current Project для разрешённых validation4xx.
- Offline recover-owner вызывал web.New без Renderer/Origin. Backend135 сообщил
  об отделении board-only offline runtime.
- Provision должен fsync новые ancestor directories перед ACTIVE registry
  commit. Backend135 сообщил о добавленном bounded sync до публикации.
- Restore auth reset не может терять единственный resume mapping pending
  provisioning при смене signing key/удалении actions; partial unpublished DB
  также нельзя безусловно обрабатывать как готовую board. Policy с root/135
  ещё согласуется.
- Infra apply durable `activation-started` после SIGKILL/power loss не принимается
  текущим resume (только `blocked-roll-forward`). Root получил требование
  recoverable idempotent roll-forward без выбора старого Go.

Сообщения авторов о fixes зафиксированы отдельно от независимой проверки:
повторный source/unit review product fixes ещё впереди. Для React/Next review
прочитан `vercel-react-best-practices/SKILL.md`; приоритет здесь scope/auth и
correctness, без несвязанных dependency/performance изменений.

Независимый повторный запуск полной infra suite: **111/111 PASS**; новые
`test_projects_operations.py` и `test_projects_restore.py` прочитаны. Они
mock host/reset boundaries и не доказывают systemd/Linux/native reset runtime.
Source `operate.py` теперь принимает `activation-started` для resume, старый Go
не выбирается при failure; quarantine allowlist добавлен в format2. Source
review выявил remaining directory durability: новый WORK/revision journal и
restore target требуют parent fsync после mkdir; root уведомлён до infra freeze.

Frontend137 frozen source независимо перечитан: QueryClient remount по project
scope, scoped keys/URLs/assets, immutable mutation body/path, scope-aware late401,
rejected envelope без foreign page merge. Отдельный запуск reviewer на bundled
Node: **53/53 unit PASS**, format/typecheck **PASS**. Первый запуск через системный
старый Node был **NOT-RUN** (`--experimental-strip-types` не поддерживается),
зависимости не менялись. UI/browser/runtime **NOT-RUN**. Единственный UX check
отправлен137: default archive должен быть disabled/hidden по server capability,
вместо заведомо отклоняемой обычной кнопки.

## Infra-only source/unit verdict

После повторного чтения итоговых `projects/{guard,layout,validate,snapshot,
backup,restore,host_ops,staging,operate,ingress}.py`, drop-ins, legacy guards,
release packaging и новых infra units: **APPROVE infra source/unit**. Parent
directory fsync для WORK/revision и top-level offline restore исправлен;
maintenance-reset требует явного JSON `restored_state_reset:true`. Owner выбран
оператором по стабильному public token ID в stage journal; bootstrap выполняется
при остановленном Go, same-owner retry идемпотентен, иной owner не выбирается.
Resume допускает power-loss `activation-started`; старый Go не возвращается.
Unpublished partial storage после restore получает private quarantine, отдельно
от routable projects, и входит в следующий полный allowlisted snapshot.

Независимый повтор **28/28 project-specific infra unit PASS** после последней
поправки; до неё полная suite **111/111 PASS** (последний added unit делает112
по author run). Source APPROVE этого слоя допускает отдельный synthetic process
rehearsal, но не заменяет фактический Linux/systemd/nginx/reset/HTTPS/VPS PASS.

Независимый Go1.27.2 run затронутых пакетов: projects/web/transport/client/bridge/
cmd/agent-board **PASS**. Это ещё промежуточный WIP run до author final freeze;
общего product/backend APPROVE пока нет, migration/preservation tests читаются.

## Final frontend disposition и remaining backend closure

**APPROVE frontend source/unit** после `can_archive`: owner/default больше не
получает гарантированно отклоняемую archive кнопку; archived owner restore
сохраняется. Browser/visual/process остаётся NOT-RUN. Последний infra owner-ID
parser допускает только public64lowerhex, не bearer; reviewer **29/29 новых
infra unit PASS**, infra-only APPROVE сохранён. Независимый `make test vet`
после основных Go fixes **PASS** по всем пакетам/существующим acceptance tests.

Дополнительно source review нашёл и135 исправил scoped REST asset item metadata
DownloadURL, который оставался default alias. Добавлен regression metadata URL
follow при одинаковых asset IDs. New `web_action_epochs` требует удаления child
при prune/logout/expiry; добавлен atomic delete trigger и авторские units.
Registry expiry cleanup сохраняет native replay mappings/live tombstones;
его trigger также должен охватывать login-only flow, а не только catalog GET.

Остаётся normal-operation pending recovery: original browser/action после crash
может истечь/выйти и потерять единственный resume mapping, навсегда заняв slot2.
Root/135 согласуют trusted offline reject-pending с сохранением raw private
bytes в quarantine, возвратом capacity и запретом старому create action оживить
тот же project ID. Global restore auth reset для этого слишком широк.
Backend source пока WIP; общего APPROVE нет. WORK превышает300 строк, поскольку
сохраняет независимый multi-stage audit trail и dispositions без потери evidence.

## Итоговый независимый source + author-test review

**APPROVE** TASK009 source и авторских unit/integration tests для перехода к
отдельному QA139. Все mandatory findings выше имеют проверенные dispositions:
catalog same-action concurrency; current ACL/credential/global-cookie/epoch перед
replay/commit; original-session upload guard; B-only logout; scoped HTML и file
metadata; rejected error envelope; default archive UI; durable ancestors и
операционный journal; complete snapshots/quarantine; explicit offline owner;
expired metadata cleanup; ordinary provisioning recovery и immutable503 retry.

Final Go freeze независимо проверен:211 файлов `cmd/**/*.go`,
`internal/**/*.go`, aggregate filename+NUL+bytes+NUL SHA256
`defe01033eb21ce219f7e16cb90736567bfa935c289b257089f01854e983aeeb`.
Авторский `backend-check.log` SHA256
`4cb707a2c2d7a861b6ddbf5b591efa3af918dd2f29123a2be60ceee9f38a3c71`:
fmt-check/vet/test/race/build **PASS**, log и WORK135 hash независимо совпали.
Содержимое новых авторских regression tests прочитано, это не тесты reviewer.

Независимый финальный `go test -race -count=1` на frozen
projects/web/transport/cmd/agent-board: **PASS**, без cached execution.
Ранее независимый `make test vet` всех пакетов **PASS**; frontend53/53 и
format/typecheck **PASS**;29 новых infra unit **PASS**. Последний root full
make-check/production build ещё выполняется отдельно и не присваивается reviewer.

Ordinary `reject-provisioning` требует существующий protected registry/marker,
не создаёт состояние по ошибочному пути, сохраняет raw bytes в private quarantine
и durable rejected ID. Rename-before-SQL crash также запрещает recreate через
проверку quarantine до provision/publication; explicit retry завершает rejection.
503 сохраняет исходные catalog action/body, transient create/switch regressions
подтверждают один project/PendingDescribe/core session.

Это **не runtime/production APPROVE**. Новые process/E2E и их source review,
synthetic all-project migration/restore/reject rehearsal, браузер/visual/negative
ACL/HTTPS, Linux/systemd/nginx/VPS preservation/deployment evidence остаются
обязательными. До их PASS этап TASK009 не завершён. Reviewer process/E2E не писал,
SSH/реальные данные/Git/source не изменял; для новых QA tests требуется отдельное
независимое test review после предоставления автором.

## Post-approval startup authority audit

Root сообщил новый P1, independently confirmed на freeze defe01033 / checkpoint
42f02c7: existing activation marker + missing/empty registry/meta принимаются
как fresh initialization. `Open`/`openRegistry` создают новый default ID и
authority, что теряет ACL/tombstones и разрешает legacy default auto-import.
Для этого bounded bug backend source disposition временно **REQUEST CHANGES**
до fail-closed fix + regression + нового freeze. Frontend/infra approvals
сохраняются. Первая часть root audit про root-entry fsync уже исправлена до
freeze: `storage.go:20-58` синхронизирует created directories и их parents,
включая `/var/lib/accord`; повторная правка этого пути не нужна.

QA139 получил read-only WIP feedback по новым helpers: occupied-port readiness
не доказывает owned process; private root должен отвергать symlink ancestry;
Native.http не должен терять auth headers при custom headers; HTTP/spawn/MCP
ошибки должны иметь deadlines и проходить через owned-PID cleanup. Final new-test
review пока NOT-RUN. Тесты139 reviewer не изменяет, runtime final evidence ждёт
исправленного binary/source approval, sourcefreeze не расширяется без bugs.

Related root audit подтвердил фактический destructive path: missing published
web.db создаётся заново, `initDesigns` создаёт empty design_assets, затем
files_recovery `keep={}` удаляет сохранённые blobs/staging как orphan. Default
web.New вызывается раньше Registry.Open, поэтому registry-only guard опаздывает.
Bounded mp preflight требуется до default Store/web constructors и для published
runtime; fresh/unpublished initialization остаётся разрешённой. Missing existing
authority tables (особенно memberships) также нельзя молча CREATE IF NOT EXISTS:
это восстанавливает lazy legacy editor grant и теряет tombstone. Root/135 получили
точные source traces и regression expectations. Singleboard не переписывается;
это часть текущего REQUEST CHANGES по missing storage/authority, не новый feature.

Root добавил bounded infra readiness ownership: до stage/cutover starts порт
должен быть свободен, после HTTP readiness `ss` listener обязан иметь exact
127.0.0.1 endpoint и множество всех PID ровно `{systemd MainPID}` выбранного
unit. Multiple rows, wildcard адрес, чужой/дополнительный PID отвергаются;
SO_REUSEADDR разрешает только закрытые TIME_WAIT. Изменения host_ops/staging/
operate и новые negative units прочитаны, reviewer **31/31 новых infra unit
PASS**. **Infra source/unit APPROVE подтверждён**; actual Linux listener/systemd
proof по-прежнему NOT-RUN. Go preflight delta пока WIP/re-review отдельно.

## Startup authority/storage amendment — final review

Bounded amendment source/author-test **APPROVE**. Reopened P1 закрыт:
activated marker требует existing initialized registry schema1 с critical auth
tables/meta/key/default binding; missing/empty/corrupt authority не создаётся
заново. Fresh initialization разрешена только на empty root; interrupted initial
activation требует inspection. Registry opener mode=rw не создаёт пропавший DB.
Preflight вызывается до default Store/web constructors, проверяет все published
active/archived board2/web1|2 DB и design metadata; published runtime повторяет
проверку до web orphan recovery. Потеря metadata не удаляет сохранённые blobs.

Авторские missing/empty/corrupt/meta/membership/marker regressions и preservation
прочитаны; destructive-blob fixtures исправлены на допустимые48hex имена, поэтому
они покрывают реальный old orphan-recovery path. Frozen215 Go aggregate SHA256:
`20b896df6eaee22e39410b2ca54eec35a5e9ffe1f720afb38145ee0904662cf5`.
Независимо совпали author log SHA256
`3eb6ece0df1c3a173d5bdaefcddc9779bac18b9fa1404d4c95bb4f046fd329f5`
и WORK135 SHA256
`5c9e598f0c70d111fcccf48d2b76e322c321332bf938d2f8c1187523afa4748b`.
Author fmt/vet/test/race **PASS**. Reviewer uncached `go test -race -count=1`
projects + cmd **PASS** (8.870s/2.417s), private log
`.local/task009-review/authority-final-race.log`. Build/bin/dist не обновлялись
этим review: для QA необходим новый binary от владельца build.

Overall product source/author-test **APPROVE восстановлен** на этом Go freeze
с ранее одобренными frontend/infra. Новые process/browser tests проходят
отдельное review, actual runtime/production/VPS по-прежнему **NOT-RUN**.
Reviewer не менял product/test source, Git, SSH или реальные данные.

## New QA test source — initial independent review

Прочитаны WIP real-process harness/run/catalog/data/stream/startup/lifecycle,
Python capture/pending/restore helpers, browser/settings/tabs и generic
production-readonly. Test source reviewer не меняет. Финальный test/evidence
вердикт **NOT-RUN** до авторского frozen source + actual run results.

Production readonly inspection ограничен собственными login/open/logout, GET
и allowlisted MCP initialize/list/resource reads; Describe/tools-call отсутствуют.
Native Origin-denial requests не путаются с browser asset errors. Private report
содержит aggregate checks/hashes, не IDs/body/token; screenshots остаются private.
Actual запуск этого сценария принадлежит root после поставки, не tester/reviewer.

Переданы139 bounded review findings: late-A test должен дождаться intercepted
response completion/cancellation до assertion и reload (иначе reload маскирует
загрязнение); actual old-binary migration нужен old default blob + byte equality
(новые A/B uploads этого не доказывают); pending raw board/journal hashes нужны
до/после restore/reject; MCP stdin EPIPE должен reject pending и пройти owned
cleanup. Общий test APPROVE до исправления/evidence не выдавался.
