# TASK-004 / WORK-085 — контракт веба Accord

Дата проверки: 19.10.2026. База исходников: `f06338089bec3b8c7f17f541c463131f5c98f96c`.
Статус: архитектурное решение для реализации; не runtime approve. Родительский TASK
прочитан в основной копии: в свежем worktree его ещё нет. Live C2/schema2 — вход
координатора; самостоятельно сервер не проверял. D4/schema4/E не включаются.

## ADR-W1. Один процесс, два хранилища

Выбор: Go SSR `html/template`, embedded CSS и небольшой JS; без новых зависимостей.
Один существующий `serve`, один loopback listener, отдельный `internal/web`.
Companion-процесс потребовал бы отдельного внутреннего auth/RPC и второй службы;
для этого объёма это лишняя эксплуатационная граница.

`board.db` остаётся schema2, authoritative для задач, human leases, ответов и событий.
`<web-data>/web.db` schema1 хранит browser sessions, операции upload, версии дизайна,
web audit; `<web-data>/blobs/` — immutable файлы. Каталоги0700, DB/sidecars/файлы0600,
regular, без symlink, безопасные parents по существующему файловому контракту.
Не добавлять web migration в board migrations и не переиспользовать номера D/E.
Нет бизнес-транзакции через обе БД: answer целиком boardDB; upload целиком webData.
Связь design с существующим task/epic — immutable key-reference с проверкой Get;
задачи физически не удаляются. Состояние/доказательства task из webDB не выводятся.

## ADR-W2. Граница HTTP и авторизация

Расширить `transport.HandlerOptions` полем `Extra http.Handler`.
Существующий transport сохраняет общий recovery/no-store/nosniff/admission/Host guard.
После native Origin reject и health/ready, ДО native bearer guard:
если `Extra != nil` и путь НЕ `/mcp`, `/metrics`, `/healthz`, `/readyz`, и
либо не nativePath, либо `/api/v1/designs`/`/api/v1/designs/...`, передать в Extra.
Иные `/api/*` продолжают существующий путь. Unknown web route отдаёт404.
Таким образом bearer asset routes тоже проходят запрет любого Origin; browser UI
использует только `/web/*` с cookie. Cookie никогда не авторизует native REST/MCP.
Административный `/metrics` и приватные маршруты наружу не открывать.

Новый обязательный при web `--web-origin=https://<canonical-authority>` задаёт
единственный browser origin, без path/query/credentials; URL authority допускает
проверенный public IP. Нельзя выводить его из Forwarded/X-Forwarded-*/Host.
Backend proxy Host остаётся фиксированным loopback, разрешённым текущим Host guard;
публичный authority проверяет nginx. Не ослаблять native external-host/DNS guard.
Каждый POST `/web/*`, включая login, требует ровно один Origin == web-origin;
`null`, отсутствующий, список, иной origin →403. Authed POST дополнительно требует
случайный synchronizer CSRF token (hidden field либо X-CSRF-Token); constant-time.
GET/HEAD не делают claim/heartbeat/logout; нет CORS. Sec-Fetch-Site cross-site →403.

GET login выдаёт preauth cookie/CSRF (hash в webDB, TTL10min); POST проверяет их до token/file IO.
Login принимает один typed password token либо малый token-file (≤4KiB), без echo;
`Authenticate(secret)` затем новый `AuthenticateID(id)` возвращает current principal
и expiry из tokens WHERE id AND revoked=0 AND expires>clock. Секрет только в памяти
обработки запроса; без URL, HTML ответа, cookie, browser storage, лога или webDB.
Browser cookie `__Host-accord_session`: random≥256bit, Secure, HttpOnly, SameSite=Strict,
Path=/, без Domain; SHA256 cookie в webDB; absolute TTL8h, не позже token expiry.
На каждом auth request проверять cookie hash/expiry, затем `AuthenticateID` текущей
boardDB; Principal никогда не строить из присланных owner/machine/tokenID.
К board mutation применится и существующая transactional authenticatePrincipal.
Session fixation: успешный login меняет cookie и CSRF. Logout POST удаляет только
эту browser session, очищает cookie, пишет web audit без сырого cookie/hash/CSRF.
Logout не снимает чужие/native leases; собственная human lease может истечь.

## ADR-W3. Human board-session, retry и история

На login открыть `Describe(role=human, machine=verifiedPrincipal.Machine)` со fresh
random runID/requestID; webDB хранит boardSessionID, expiry, runID, CSRF и pending
action. Один browser-session — одна board-session; вкладки сериализуют мутации
этой session. Human создаёт эпики/задачи всех существующих ролей через `Create`.
Review/tester form требует target developer key и commit согласно C2.
Кнопка «Взять вопрос» вызывает `Next(TaskKey=exact-key)` только human question;
nil Task →409 «Вопрос уже занят или недоступен». Не claim на GET и не fallback
на следующий вопрос. Сначала claim, затем answer `Transition(done, generation,
Evidence.Answer)`. Для агентских ролей кнопки захвата и завершения не показывать.

Истёкшую board-session восстанавливать fresh Describe только POST «Продолжить»
или перед новой create/claim после проверки старой операции. Не менять cookie
TTL. Истёкший editor/answer не reacquire автоматически:409, текст оставить видимым,
предложить проверить состояние и явно взять заново. Нет silent retry с новой generation.
Если предыдущая mutation response неизвестна — сначала exact replay тем же native
session/requestID/payload; C2 replay допустим после session expiry, но не revoke.
Fresh action IDs формируются сервером и привязаны в webDB к cookie hash, operation,
payload; сохранять pending ДО native call, result ПОСЛЕ. Повтор отправки использует
тот же ID; crash между board commit и web result восстанавливается native replay.
Pending describe тоже сохраняется ДО вызова, иначе retry способен плодить sessions.
Heartbeat POST только явно включён пользователем у открытого принадлежащего editor; interval из SessionView,
явные taskKey/generation; после hidden/offline/error/lease_lost остановить, показать
потерю владения. Read-list/history/poll никогда не продлевает lease.
GET при board-session expiry показывает состояние «Нужно продолжить сессию»;
это не logout browser-session. Нет невидимого describe на каждом чтении.

Существенная находка source: `Attempt` НЕ содержит Answer; текущее Evidence заменяется.
Добавить в `core.EventData` nullable `Answer *string`, `Generation int64`,
`AuthorPrincipal string`, `QuestionTitle string`; emit `human_answered` при успешном
human done в той же board transaction ПОСЛЕ обычного task_done/tokens event enrichment. Data.Kind=`decision` (enrichEvents пропустит
готовые facts); Data.Owner/EpicKey/Role задаются из проверенной session/task.
Не менять статистическое событие task_done и не считать human_answered completion.
Append-only raw event rows переживают old binary; старый binary не переписывает их.
History требует новый Board.ListDecisions(ctx,actor,core.DecisionsInput)(core.DecisionsPage,error):
BeforeCursor/HighWater int64, EpicKey string, Limit int; Events []core.Event, NextCursor/HighWater int64, HasMore bool;
bounded SQL DESC cursor, type human_answered, optional event-time epic_key, ≤100, auth read без renewal.
Native Events(types=human_answered) остаётся ascending-compatible для агентов;
текст, автор, title, generation и дата — event-time facts. До web-версии доступен
только текущий Evidence.Answer с явной пометкой legacy; не выдумывать прошлые тексты.
Нужны тест повторного ответа после reopen и rollback-old-reader без потери event rows.

## ADR-W4. Дизайн и файлы

Все authenticated participants пилота читают project designs; роли не являются ACL.
Upload доступен человеку через cookie+CSRF. Agents получают metadata и authenticated
download через bearer; новые MCP tools не добавлять (остаётся12).
DesignID — случайный ID группы; AssetID — случайный immutable ID версии.
Scope enum project/epic/task, scope_key обязателен для последних двух; metadata также
содержит epic_key (для task берётся из Get; присланный иной epic отклоняется); проверять
существование и тип core.Get под текущим actor, project key — configured Project.
Версии группы неизменяемы, номер выделяется webDB transaction UNIQUE(design_id,version).
Новый upload к тому же DesignID — новая версия; не overwrite, не delete history.
Metadata: IDs, version, scope/key, title≤200runes, display filename≤255bytes,
server media type, size, SHA256, uploader principalID/owner, createdAt, downloadURL.
В имени для Content-Disposition использовать стандартный mime.FormatMediaType;
никакие filename/title из клиента не участвуют в filesystem path.

Принимаемые типы только PNG/JPEG/WEBP/PDF/ZIP/HTML/SVG по safe sniff; corrupted raster→422.
Файл≤50MiB, весь multipart≤50MiB+64KiB; только один file part, bounded fields,
без ParseMultipartForm/ReadAll файла. Общая квота1GiB включая reserved/pending;
reserve50MiB атомарно ДО чтения, одновременно≤2 uploads, disk reserve512MiB
проверяется до и во время stream порциями; при невозможности определить free disk
отказать upload, без отключения чтения board. Quota/disk-full →413/507 без публикации.
Deadline upload/download120s, остальной native server30s в прочитанной базе;
не поднимать общий server timeout. До чтения body поставить ResponseController
SetReadDeadline/SetWriteDeadline(now+120s), обработать unsupported до создания файла.
Сохранить обычный ReadHeaderTimeout5s/MaxHeaderBytes16KiB.

Сначала pending webDB row с requestID и reservation. Случайный basename, O_EXCL0600
и NOFOLLOW в private staging; parent не writable чужим UID. Stream+hash, fsync file;
атомарно publish в blobs без overwrite (random basename и exclusive final reservation),
fsync parent; затем metadata status=ready commit. Ответ только после commit.
Crash может оставить pending/orphan, но не ready без durable file. На startup до
ready сверить pending с file/size/hash: завершить publication либо удалить partial
и освободить reservation с audit. Повтор requestID+metadata+digest возвращает ту же
версию; иной payload →409. Для lost response upload retry body надо дочитать/hash
bounded, иначе нельзя различить прежний файл и иной под тем же ID.
Cleanup каждые15min удаляет только expired browser sessions и подтверждённые
abandoned staging older24h, не ready blobs; pending active исключить под lock.
Счётчики reserved/used, last cleanup success/error видимы оператору без secrets.

ZIP opaque, без unpack. HTML/SVG/PDF — attachment application/octet-stream
nosniff; никакого execution/inline preview активного формата. Только PNG/JPEG/WEBP
могут иметь preview после sniff+DecodeConfig и width/height≤8192, pixels≤16M, concurrent preview≤1.
Если для WEBP нет доступного decoder без новой зависимости — attachment, не fake
preview. Preview отдельный authenticated endpoint, CSP sandbox/default-src none,
image content-type, не доверять extension/client MIME; повреждённый image →attachment.
Downloads разрешать только ready metadata → server-generated basename → regular
file, NOFOLLOW, verified size; никакого HTTP FileServer для web-data.

## Контракты для трёх исполнителей и интегратора

1. WEB-BACKEND: `internal/web/**` КРОМЕ `design*`/`files*`, `internal/store/web_auth.go` и tests.
   Один package web; backend владеет contracts.go, Server, web.db open/schema1/session
   migrations, auth, actions, task HTTP. Imports core, не store/transport/webui.
   Board: Authenticate; AuthenticateID(ctx,id)(core.Principal,time.Time,error); Describe,
   Next, Get, Create, Transition, Release, Heartbeat, Events, ListTasks, ListEpics
   с точными existing Service сигнатурами; дополнительно ListDecisions, описанный выше.
   Options{Origin,DataDir string; Renderer Renderer; Static http.Handler};
   New(Board,Options)(*Server,error), Close()error, Ready(ctx)error; Server=http.Handler.
2. WEB-FILES: ТОЛЬКО `internal/web/design*`, `internal/web/files*` и соответствующие tests.
   Тот же Server имеет `db *sql.DB`, `board Board`, `dataDir string`; backend вызывает
   `s.initDesigns(ctx)error`, затем `s.recoverDesigns(ctx)error` ДО readiness/schema1 publish.
   Files владеет CREATE TABLE IF NOT EXISTS design/assets/upload_reservations; backend —
   session/action/audit tables. Никаких конкурентных startup migration и отдельных DBhandles.
   Routing hook `s.serveDesigns(w,r) bool` вызывается backend перед его404; files использует
   backend `s.browserIdentity(w,r)(Identity,bool)` либо `s.bearerIdentity(w,r)(Identity,bool)`.
   Identity{Actor core.Actor; BrowserKey string}; helpers включают current auth, live board
   session и Origin/CSRF для unsafe browser requests ДО любой file IO; BrowserKey — hash.
   Exported Server.ListDesigns(ctx,actor,DesignFilter)(DesignPage,error),
   GetDesign(ctx,actor,assetID)(DesignAsset,error) — MCP metadata bridge, auth read повторяется.
3. WEB-UI: только `internal/webui/**`; imports web/core/standard lib; New()(*Renderer,error),
   Static()http.Handler; implements Renderer.Render(w,status int,name string,page Page)error.
   Root отдельно: cmd/transport/core/catalog.go + core/decisions.go + store/web_history.go +
   store/transitions.go + deploy/docs/integration tests. Source авторы не трогают эти файлы.
   UI/backend/files согласуют frozen contracts.go до реализации; не дублируют DTO.

Фактически согласованный DTO закреплён в `internal/web/contracts.go` (WORK087/088/089).
`Page.Draft` — `map[string]string`, `Actions` — `map[string]string`, FieldErrors
отдельно; это уточнение интеграции вместо первоначального typed Draft.
DesignAsset использует AssetID, UploaderPrincipal, UploaderOwner, PreviewURL;
DesignPage — Assets/NextCursor; Options — StaticHandler/TrustedProxy.
Backend-owned contracts.go — единственный источник точных полей для UI/files/MCP.
Template names/ViewIDs: login, board, task, designs, history, error; вопросы
`/web/?view=questions`, задачи `?view=tasks`. Actions keys create,logout,continue,
upload,claim:T-N,answer:T-N,release:T-N,heartbeat:T-N; разные формы имеют разные
серверные operation-bound IDs. Heartbeat JSON session,next_action_id.
Token/password не входят в Page/Draft. Только escaped html/template.

## Routes/forms (замороженный интеграционный контракт)

Browser GET `/`→303 `/web/login` для гостя, `/web/` для действительной browser-session; `/web/login`, `/web/?role=human` (вопросы), `/web/`, `/web/tasks/{key}`,
`/web/designs?scope=&scope_key=&design_id=&cursor=`, `/web/history?cursor=`.
Static GET `/web/static/{asset}`; renderer.Static монтирует интегратор через Server.
POST `/web/login` token либо token_file; `/web/logout`; `/web/session/continue`;
`/web/tasks` поля type,epic_key,title,description,role,priority,owner,areas,target_key,target_commit; `/web/tasks/{key}/claim`;
`/web/tasks/{key}/answer` answer,generation; `/web/tasks/{key}/release` reason,generation;
`/web/heartbeat` task_key,generation; `/web/designs` multipart file,title,scope,
scope_key,epic_key,design_id(optional). Все authed POST имеют csrf и action_id; никаких
client principal/sessionID. HTML action response PRG303; validation/conflict
renders escaped form state с4xx, JSON heartbeat с200/4xx; raw answer не терять.
GET `/web/designs/{assetID}/download` и `/preview` — cookie, без bearer в URL.
Bearer GET `/api/v1/designs?scope=&scope_key=&design_id=&limit=&cursor=`;
GET `/api/v1/designs/{assetID}` metadata; `.../download` bytes.
Bearer требует текущий токен и live `X-Session`, проверяемые board read; Origin denied.
MCP resources `accord://designs{?session_id,scope,scope_key,design_id,limit,cursor}`
и `accord://designs/{asset_id}{?session_id}` возвращают bounded JSON metadata+URL,
не base64 blobs. Реализация optional designReader interface, injected into MCP;
`Service`/tools count не расширять; malformed/repeated/unknown params rejected.

## Backup, проверка, live и откат

Отдельный reviewed helper web-data backup, не менять существующий WORK168 board backup.
Остановить единственного writer на время согласованного board+web snapshot; trap
восстанавливает службу при ошибке. Только после остановки checkpoint/закрытия DB
копировать web.db+blobs в private tmp dir; проверить hashes/refs, manifest SHA256,
fsync, atomic rename. Self-contained web archive≤1GiB+DB; учитывать минимум512MiB
свободного пространства ПОСЛЕ создания временной копии, иначе отказ с audit.
Retention7 успешных daily snapshots, никогда не удалять последний хороший до нового;
владельцу сообщить disk capacity boundary, backup на том же VPS не offsite.
Restore в отдельный loopback fixture: integrity_check, hashes, versions, scoping,
история, exact native replay; web sessions удалить перед восстановлением сети.
Board recovery сохраняет существующее требование revoke-all+invalidate sessions.

Unit/integration авторов: current revoke/expiry, fixation/logout, Origin duplicate/
missing/null/forwarded spoof, CSRF, XSS, separate cookie/bearer domains; same-question
race, two tabs, stale generation, session expiry, crash/replay ответов; pagination;
quota concurrent reservation, chunked>50MiB, disk full/cancel, symlink/path traversal,
MIME polyglot/invalid image, opaque ZIP, retry duplicate version, every crash boundary,
restart cleanup и восстановление backup. Native REST/MCP12 compatibility обязательно.
Slow real net/http stream >30s и timeout120s проверить real listener, не Recorder.
После independent approve: отдельный tester browser journey people→agent question→
human claim/answer→agent read; human task→native agent claim; upload→version→
authorized download→restart; mobile/keyboard/error states + независимый UI review.
Live: private backup+checksums; validate nginx syntax/systemd; switch binary only;
nginx root routes browser UI, upload location body50MiB+64KiB и120s, остальные native
body64KiB unchanged; private backend/Tandem untouched. Verify HTTPS/auth/revoke,
cookie flags, native Origin, MCP12/resources, streamed design roundtrip; no load test VPS.
Rollback: прежний binary + прежний nginx config, restart health/ready/native smoke;
board schema2 неизменна, новые web файлы остаются закрытыми и не удаляются. Не
restore board backup ради rollback: это потеряло бы новые ответы/задачи.

Источники API проверены19.10: [ResponseController](https://pkg.go.dev/net/http#ResponseController.SetReadDeadline)
(deadline поставить до истечения); [os.Root](https://pkg.go.dev/os#Root) (root containment
не заменяет NOFOLLOW, symlink внутри root надо запрещать отдельно).
Нагрузка и SLA не обещаются; D4 measurements, E runtime и физический второй ПК — отдельно.
