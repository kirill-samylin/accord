# WORK-256 — HTTPS220V3: request lifecycle diagnosis

Статус: DIAGNOSIS COMPLETE / prospective contract review PENDING.
Автор gui_connection_qa. Только сохранённые артефакты и исходники; без нового
browser/HTTP/native/provider/CLI, чтения токена или изменений production/source V3.

## Непосредственно подтверждено

V3 source `2f84be95e541eba1788fbaa29477d64bc52340edfa9d07cea0bb811eecb331ac`.
Runtime result `763be6a59964b2562bbb0e9e0bcb92f318882a7eeeb86f7c6fd8b75675e43e56`.
FAIL остаётся неизменным. Stage CATALOG_SWITCH, Error, собственный frame scope.cjs4:
это deadline tracker.settle ДО open.click. Switch POST ещё не выполнен. Ни UI
selector, ни реальный OpenProject POST в этом запуске не проверены.

Сохранены четыре GET/fetch ERR_ABORTED: anonymous protected automation;
authenticated login, OTHER_PATH и projects. Это наблюдавшийся код отказа, а не
доказанная причина отмены. Started stage, response status для того же request ID
и active tracker snapshot НЕ сохранены. Поэтому неизвестно, какой request остался
active, была ли потеря terminal event либо незавершённая сеть. Реальный путь
OTHER_PATH нельзя восстановить из безопасной категории.

Контексты/server закрыты; ChildProcess close подтверждает code0/signalnull/stdio.
Main operation joined; sources/inputs unchanged. all_requests_join FAIL, поэтому
полный cleanup/manual flag старого отчёта НЕ исправляется. Положительное завершение
ОС-процесса отдельно подтверждено; это не доказательство полноты сетевого журнала.
Нет blocked, projection failure, pageerror, unexpected HTTP/console errors.

Private finite proof + SHA всех исходных артефактов:
`.local/task014/https220/diagnosis256/diagnosis.private.json`, SHA
`d8f02913c8e8a2f4c16aeca287491d5e747532f29ee86b16f6922e7860eeaceb`.

## Точные source механизмы (не причинный диагноз каждого request)

- api.ts readPage вызывает unauthorized до await response.json. В ControlContent
  unauthorized выполняет client.clear + router.replace. QueryBoundary unmount
  делает cancelQueries + clear. Поэтому штатный anonymous401 может отменить
  исходный fetch при переходе на login даже после получения headers401.
- Login success также clear + router.replace. V3 сразу после waitForURL ещё раз
  делает полный goto('/app/projects'), хотя SPA уже перешла к каталогу. Это лишний
  teardown с возможной отменой старых login/route/catalog reads.
- OpenProject запускает void invalidateQueries и сразу router.replace; смена
  catalog→project QueryBoundary может отменить новый catalog refetch. В V3 до
  этого места не дошли; это перспективный контрактный случай, не actual V3 факт.
- Pinned Playwright coreBundle BrowserContext _onRequestFinished/_onRequestFailed
  всегда emit context event, а page event — только если params.page присутствует.
  Context subscription полнее page subscription. Отсутствие page в конкретном
  старом событии не наблюдалось; потерю event как точную причину не утверждаем.

## Минимальный prospective V4 контракт — на review202

1. Никаких повторных goto каталога: login с поддержанным next=/app/projects,
   ждать точную catalog DOM/уникальную project-card. Перед login submit дождаться
   текущих login reads. Anonymous отдельный context после проверок закрывается
   явно; его tracker отделён от authenticated. Native/models по-прежнему0.
2. Request/response/finished/failed слушать на context до создания page. При старте
   присвоить bounded local ID и finite context/startedstage/path/method/type;
   response status связывать с тем же объектом. При deadline/final сохранять
   finite active snapshot и lifecycle totals. Caps500/128, без raw URL/query/
   body/headers/cookies/messages. Terminal event — только finished/failed, не
   просто response или route.continue. Не исчезать записям без доказательства.
3. Обычные authenticated/asset/page/network/unknown ошибки остаются FAIL. Любое
   особое EXPECTED abort требует отдельного явного approval202 контракта, точного
   request binding и положительного исхода ожидаемого UI перехода. Возможный
   узкий anonymous случай: собственный protected automation GET/fetch, literal
   ERR_ABORTED, same-request response401 и подтверждённая login DOM/redirect;
   explicit JSON401 probe, произвольный путь/status/asset сюда не входят.
4. Возможный catalog-scope abort после собственного successful switchPOST200 и
   точного scoped route+JSON200 — отдельный prospective случай. До независимого
   approval не исключать его из FAIL и не считать любой ERR_ABORTED нормальным.
   В старом V3 вообще не было этого POST, поэтому к нему допуск неприменим.
5. Прежние180+30s/hash/version/audit/strictTLS/positive Chrome0/joins/readonly
   guards остаются. Никакого automatic retry, business mutation, global allowlist,
   ретро-PASS старого результата или принятия UNKNOWN.

Root/reviewer получили диагноз до правок. Полный V4 source ещё не написан;
следующий шаг — точный contract disposition202 и отдельное root SOURCE GO.

## Reviewer202 prospective disposition

Context-level до page, per-context finite ID/startstage/status/terminal/active-map,
убрать повторный catalog goto только после exact redirect+DOM assertion, login
settlement и отдельный anonymous closure — APPROVE prospective. Все caps прежние.
**Оба abort exception НЕ approved**: same-request401/DOM либо switch200/target200
и исходный код сами по себе не доказывают намеренную отмену конкретного request.
Предпочтение202: первая V4 без новых исключений, ALL operational ERR_ABORTED FAIL.
Иная граница потребует точного наблюдаемого контракта и отдельного root выбора.
Ни один source/runtime FAIL не пересмотрен.

## V4 SOURCE READY (root выбрал prospective A, reviewer202 CONTRACT APPROVE)

После уточнения root/202 выбран ТОЛЬКО A: exact anonymous protected GET/fetch/noquery,
same-request401/ERR_ABORTED/context/startstage/terminalstage ANON_PROTECTED_APP,
реальные login route+DOM и отдельный fully consumed denied JSON401/UNAUTHORIZED.
Это qualified expected protocol outcome, не утверждение AbortController intent.
B/authenticated superseded read exceptions НЕ приняты и НЕ реализованы.

Заморожено `.local/task014/https220/v4/source-pins.json`, SHA
`c6aebfdfa4f00a6d2da8fb9aba8e8249f28353ea616d6d4369002d3827025744`,26members.
Context события подписаны до newPage, per-context/request IDs/status/start/terminal/
active snapshots сохранены до cleanup и после; route/tracker/scope joins отдельны.
Устранён повторный goto каталога; supported login next+exact catalog DOM обязательны;
anonymous context явно settled/closed до authenticated. OS closure отдельно строгий0.

A квалифицируется только после положительных проверок: explicit probe захватывает
свой фактический context request, fully consumed body, FINISHED401 и отличающийсяID.
Proof frozen, FAILED facts frozen и сохраняются без переписывания в FINISHED;
final everyfailure predicate допускает только полное A, остальные ошибки FATAL.
Нет нового business/native разрешения; preflight/private/scope/pins/libraries/
reviewed ui-controls byteexactV3.180+30/TLS/assets/hash/audit/identity guards прежние.

Static13CJS syntax PASS, max229lines,26pins,0700/0600. Новые pure EventEmitter/context
и A-boundary fixtures + прежний projection test написаны, NOT-RUN автором;202
разбужен followup_task для whole source review и независимого выполнения трёх pure
файлов `test-projections.cjs`, `test-context-events.cjs`, `test-anonymous-abort.cjs`.
Браузер/HTTP/native/provider/токен/runtime автором не запускались. Старые FAIL неизменны.
Будущий root: pinned node24.19 `v4/main.cjs <new-private-input0600>`, прежняя input
schema, новые exact source202 receipt/fresh approved212 audit≤30min/rootGO.

## V4 actual FAIL и prospective V5 (отдельный root/202 контракт)

V4 whole SOURCE APPROVE c6aebfdf; independent pure A1+51negative и две3group
проверки PASS. Реальный V4 results SHA
`ca53291abdb1ceb8be01677ae05e3d1489e7629782a2f8bc33846c9bb03cbe65` остаётся FAIL.
Все шесть JSON200/DOM/screenshots, формы, assets/TLS/identity прошли; final assertion
отклонил7 authenticated GET/fetch ERR_ABORTED. Все terminal FAILED/active0;
Chrome0/context/server/drains/route+tracker+scope joins положительны/manualfalse.

Семь конечных tuple: login app/projects200/querypresent и web/login/noheaders;
switch app-scoped-root200/querypresent и web/projects/noheaders; overview web-scoped-
root200; nav app-scoped-profiles200/querypresent и app-scoped-automation200/querypresent.
OTHER_PATH failures отсутствуют. Querypresent НЕ доказывает RSC в старом запуске:
markers/headers там не сохранялись; новый контракт не применяется ретроактивно.

Next16.4.2 fetch-server-response.js в pinned release manifest непосредственно hash-
проверен; он содержит отмену неиспользуемых stream/cache clone. Локальный locked
ReactQuery5.104.5 query.ts подтверждает removeObserver/cancelRefetch → abortController;
ControlApp clear/cancel и OpenProject invalidate-before-router поддерживают обычную
SPA cancellation. Это объясняет допустимость нового функционального контракта,
но не доказывает точный AbortController cause каждого старого события.

Root и202 prospectively APPROVE конечную DATA/RSC матрицу с immutable собственным
witness LOGIN/SWITCH/NAV: OPEN до actual click; monotonic requestStart/FAILED до SEAL;
SEAL после targetURL/uniqueDOM/fully-consumed JSON200/rightproject, для login ещё
originalidentity+membership; actual login/switch POST200/FINISHED обязательны.
DATA: только from/to web endpoints текущего witness/queryempty. RSC: только target
app route/ровно один _rsc ключ/shape-bounded непустое значение/RSC1 boolean. Значения
headers/query не сохраняются. Status literal200|null, GET/fetch/ERR_ABORTED, точные
context/stages/sequence bounds. Все прочие ошибки fatal, rawFAILED сохраняется.
Функциональная квалификация не является заявлением о причинности cancel.

## V5 SOURCE READY

`.local/task014/https220/v5/source-pins.json` SHA
`cdb4364d0f7f1861d671f1edabb157820ca787aa27af6eae2720fb0e16c6d8a0`,31members.
Новый navigation-witness62lines; каждый sealed witness и raw prequalification
snapshot сохраняются до qualification. Убран лишний overview self-click, сохранены
sidebarhref/uniqueheading/fullJSON; initial overview sealsSWITCH, пять navs отдельны.
A helper/tests byteexactV4; all preflight/private/scope/pins/libraries/controls
byteexact, TLS/assets/version/audit/identity/zero-native/business/caps/3joins/Chrome0
сохраняются.15CJS syntax PASS,max252. Pure navigation matrix+старые3fixtures написаны,
NOT-RUN автором;202 разбужен followup_task для wholeSOURCE и4purefiles. Никакого
нового live/browser/HTTP/native/provider/токен-чтения/production действия автором.
Старые V3/V4 source и реальные FAIL неизменны; runtime/rootGO только после review.

## V5 actual FAIL и V6 A2 SOURCE READY

V5 source APPROVE cdb4364d; independent navigation6positive+94negative/A1+51/
projection+context3groups PASS. Реальный V5 results
`469b8cc5ca50ae71c02d799047c1785e9ada010eec08332cfc1e1ee3022b5be6` остаётся FAIL:
единственный fatal — anonymous /app/login GET/fetch/RSC1/200/ERR_ABORTED, все stages
ANON_PROTECTED_APP. Query keys/next там не сохранены; точная форма UNKNOWN.

Root+202 prospectively APPROVE только A2: future sameanonymous/context/3stages,
FAILED/GETfetch/typed200/ERR_ABORTED/RSC1/query_emptyfalse; exact query keys next+
_rsc once each/noextras, decodednext строго ownprotectedautomation, ASCII_rsc1..128.
Pinned Next cache-bust source сохраняет existing query и добавляет_rsc; ControlApp
создаёт login?next через safeLoginReturn. Это source основания, не retro-runtimeproof.
Строгий прежний A proof обязателен целиком: loginroute/uniqueDOM + distinct normal
FINISHED explicit protected401/okfalse/UNAUTHORIZED. RawFAILED и исходные FAIL не
переписываются; никакого exact cancel intent claim.

V6 `.local/task014/https220/v6/source-pins.json` SHA
`679e2be33c29257964fcf921e96c605ad24dd7d6d74b47a44aa6d0a448f5123a`,36members.
Из старых files изменены только README/browser/request-tracker (два finite flags).
Новый anonymous-login-rsc helper и pure boundary fixture; main/smoke/A/navigation/
прочие guards byteexactV5.17syntax PASS,max254,0700/0600; pure написаны NOT-RUN
автором.202 разбужен followup_task для wholeSOURCE и пяти purefiles. Никаких новых
browser/HTTP/native/provider/token/runtime запусков; rootGO только после review.
