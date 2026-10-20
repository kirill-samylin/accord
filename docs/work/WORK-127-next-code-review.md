# WORK-127 — независимое code/security review TASK-007

20.10.2026. Reviewer127. Автор WORK-122 (архитектура); продуктовые файлы и
авторские тесты TASK-007 не писал. Область записи только этот WORK; исходники
проверяются чтением. Общую TASK ведёт координатор. Tandem OFF.

Итог **APPROVE** code/tests/local evidence на product source
`5cc0a88f74a0040540dcb111466696314810ca8f` после исправлений и финального QA.
Все обязательные найденные code/visual/security пункты закрыты.
Это локальный TASK007 Next-client verdict в пределах записанного контракта;
полная Paperclip parity/production/физический второй ПК этим не объявляются.
Исключение длины WORK объясняется сохранением независимого audit trail
четырнадцати исправлений и нескольких immutable source/test anchors.

## Прочитано

Текущая TASK-007, WORK-122/124/126; Go JSON writer/projection и подключения
к http/design/mutation paths, capability Update, native Store.Update,
новые Go regression suites; forms useAction/actionState/contracts и все шесть
форм; Next API/type/upstream/proxy/layout и появляющиеся screen modules.
Дизайн и независимый visual review не подменяются source review.

## Найдено и исправлено

Все пункты1–12 закрыты по source/unit review; итоговые решения — в таблицах.

1. **P1 — неверный токен навсегда блокирует форму входа.**
   `LoginForm.tsx` использует общий `useAction`, а `stoppedBy(UNAUTHORIZED)`
   переводит ответ401 в `blocked`. Login fieldset становится disabled;
   ввода исправленного токена/обновления preauth в форме нет. Требуется
   login-specific обработка подтверждённого отказа и явное восстановление
   bootstrap после истёкшего/израсходованного preauth, без ослабления auth.
   Отправлено координатору для126.
2. **P1 — открытые эпики исчезают из списка.**
   `TasksScreen` вызывает общий `list(page.epics)`, но группирует по task
   states без `open`. При наличии open эпиков список пустой, хотя счётчик
   ненулевой. Нужны отдельные epic states и пагинация epic_cursor.
   Отправлено125.
3. **P2 — фильтры активности используют несуществующий vocabulary.**
   `organization.tsx`/`format.ts` содержат `task_claimed`, `task_transitioned`,
   `task_released`, `comment_added`; настоящий allowlist
   `internal/core/control_validation.go` содержит `claimed`, `released`,
   `commented` и отдельные `task_ready/task_in_progress/task_done/...`.
   Сейчас выбор предлагаемых фильтров вызывает400. Отправлено125.
4. **P1 — вкладка стирает черновик и неизвестный исход.**
   `TaskDetail` помещает `QuestionActions` в Radix `TabsContent description`
   без forceMount. При переходе на другую вкладку её children unmount;
   локальные answer/reason и frozen pending payload теряются. Нужен стабильный
   владелец состояния либо forceMount+hidden для неактивного content.
   Отправлено125 и координатору.
5. **P2 — query другого экрана подменяет тип новой задачи.**
   `CreateTaskDialog` распространял весь `page.draft` поверх `type:task`;
   фильтр `activity?type=claimed` становится create type вместо task/epic.
   Нужен отдельный whitelist draft и проверка enum при initialization.
   Передано координатору и126.
6. **P1 — пустой фильтр scope ломает обычную загрузку и версии.**
   Go design page всегда содержит `draft.scope`, по умолчанию пустую строку.
   `DesignUpload` init `{scope:project,...page.draft}` заменяет project на пустое
   значение и отправляет upload со scope пустым: сервер400. Для новой версии
   task/epic материала необходимо также восстановить фактические scope/scope_key
   из загруженных assets: Store запрещает менять область существующего design_id.
   Нужны normalized upload draft и регрессия. Отправлено126 и координатору.
7. **P1 — Next external rewrite обрезает upload после10МиБ.**
   Координатор выявил, reviewer подтвердил по установленному Next16.4.2:
   `resolve-routes.js` создаёт clone body вне зависимости от nonce matcher,
   `router-server.js` передаёт cloneBodyStream в external proxy,
   `body-streams.js` обрезает default10МиБ. Для локальной схемы нужен
   `experimental.proxyClientMaxBodySize=51МиБ`, Go лимиты неизменны.
   Production `/web` должен идти прямо Go; Node путь имеет memory copy.
   WORK-122 исправлен, прежняя интерпретация matcher явно отозвана.
8. **P2 — у dashboard polling утрачен deadline/backoff/eligibility abort.**
   Координатор обнаружил, reviewer независимо подтвердил по исходникам:
   прежний `internal/webui/assets/control.js` имел8секундный abort,
   backoff30/60/120секунд, ограниченный Retry-After429/503 и abort при
   hidden/offline/pagehide. Новый `OverviewScreen` использует query signal,
   но не устанавливает fetch deadline, не увеличивает fixed15sec interval
   при сбоях и не отменяет уже начатое чтение при скрытии вкладки.
   Go WriteTimeout30sec и Next upstream130sec не заменяют browser deadline
   и не защищают от зависшего соединения до браузера. Нужен bounded read
   с совмещённым query/lifecycle/deadline signal и прежней политикой retry,
   без изменения upload/POST контракта. Исправлено в reliability delta.
9. **P2 — зависшая mutation оставляет dialog в sending без пути replay.**
   Координатор обнаружил, reviewer подтвердил: прежний fetch heartbeat
   `app.js` имел15секундный abort. Новый `submitAction` не имеет client
   deadline; fetch/body может зависнуть, а `useAction` продолжает блокировать
   закрытие/редактирование без показа immutable retry. Нужен client deadline
   обычных POST15sec; timeout переводит в unknown, сохраняя точный payload
   и action_id. Deadline должен охватывать fetch и чтение response body.
   Upload требуется отдельный бюджет130sec поверх Go120sec, а не15sec.
   **RESOLVED scoped:** root добавил deadline.ts, обёртку useAction и4 unit
   tests. Reviewer проверил optional signal до fetch и deadline вокруг полного
   submitAction, включая JSON body; самостоятельно4/4 PASS Node24.
   200headers+stalled body становится UNKNOWN_RESULT/retry_same_action;
   следующий POST возникает только при явном повторе с тем же объектом body.
   Автоматический повтор мутации не разрешается. Общий freeze ещё ожидается.
10. **P1 — failed refetch уничтожает human draft после heartbeat.**
    `ControlContent` возвращает fatal error page при `result.isError`, даже
    если Query сохраняет `result.data` от успешного чтения. Human heartbeat
    success инвалидирует queries; следующий task GET503/network fail размонтирует
    `TaskDetail/QuestionActions` и стирает несохранённые answer/reason.
    Fatal fallback должен применяться только без last-good data (при401
    сохраняется обязательное удаление identity cache). При failed refetch
    следует оставить стабильную страницу и показать notice. Отправлено
    root/125; исправлено. Browser regression после исправления ожидается.
11. **P2 — новый dashboard validator допускает сломанный snapshot.**
    В первой reliability delta `validDashboard` проверял только counts и
    наличие arrays. `sessions:[{}]`, invalid observed_at, oversize6/20 и
    повреждённые events принимаются и заменяют last-good; render может упасть
    на owner.trim. Прежний TASK-006 validator имел entry validation и limits.
    Проверка и negative unit cases восстановлены125; reviewer принял.
12. **P2 — skipTrailingSlashRedirect оставляет разрешённые URL нерабочими.**
    После отключения Next normalization разрешённый `screen` regex путь
    `/app/login/` мапится в отсутствующий Go `/web/login/`; аналогично detail.
    Нужно нормализовать завершающий slash вне root в appToWeb, сохраняя
    `/app/` → `/web/`. Исправлено125 вместе с canonical regression cases.
13. **P1 — настоящий Next rewrite теряет slash корневого Go пути.**
    QA128 artifact2full01 остановился на реальном входе: при пустом wildcard
    destination `/web/:path*` становится `/web`, но Go overview имеет `/web/`.
    Root и125 исследуют explicit rewrite/canonical `/web/overview`.
    Unit appToWeb проверял строку до proxy, поэтому40PASS не закрыли этот
    дефект. Требуются новый source delta review, настоящий root/login smoke
    через production Next и новый artifact перед full acceptance128.
    Legacy root `view=tasks|questions` должен продолжить работать без
    молчаливого принятия duplicate/invalid query. Исправлено literal root rule.
14. **P1 — mobile navigation скрыта поздней CSS декларацией.**
    Actual artifact3 PNG390 показал отсутствие burger/search/bottom nav.
    Reviewer самостоятельно посмотрел `.local/task007-qa/artifact3-journeys-01/failure.png`
    и подтвердил source chain: `layout.css` mobile media включает элементы,
    но импортированный позже `overlays.css` снова задаёт их display:none.
    Root поручил125 исправить cascade. Reviewer принял узкую дельту:
    две hidden base декларации перенесены в начало layout.css перед его
    же media overrides; других перекрывающих global declarations нет.
    Desktop768+ и safe-area reserve не менялись. Нужны artifact4 и реальный
    mobile повтор; source APPROVE не завершает общий этап.

## Уже проверено

- **PASS:** собственный `.local/toolchains/go/bin/go test ./internal/web`
  после сдачи backend124; завершился успешно (cached).
- **PASS:** собственный Go race suite для `internal/web` (cached).
- **PASS:** собственный bundled Node24.19.0 запуск двух pure mutation suites:
 10/10. Это не browser evidence и не проверяет блокировку login DOM.
- **PASS:** после первой серии исправлений собственные frontend22/22 unit
  и полный TypeScript typecheck. Промежуточный15-suite запуск имел14PASS/1FAIL
  до применения автором125 login401 event exemption; после исправления15/15.
- Go JSON использует allowlist DTO без private browserSession/token data;
  shared handlers сохраняют auth, Origin/CSRF, durable action payload/session,
  update native ownership/fencing и recovery. Binary design paths исключены
  из negotiation, writer Unwrap сохраняет response deadlines.
- Next фиксирует loopback upstream и ограничивает nonce proxy `/app/**`.
  Последнее не устраняет external rewrite buffering (см.7);
  production hydration фактический браузерный PASS пока NOT-RUN.

## Проверка исправлений и финальной версии

| Замечание | Принятое исправление | Повторная проверка |
| --- | --- | --- |
| 1 login | Отдельная login policy401 editable;403 explicit preauth bootstrap; login401 не публикует global unauthorized event | Source + реальные API helpers с mock fetch; регрессии PASS |
| 2 epic open | groupTasks сохраняет open и неизвестные состояния; epic pagination использует отдельный cursor | Source + unit PASS |
| 3 vocabulary | Используются claimed/released/commented и реальные task transition types | Source сопоставлен с core.ValidControlEventType |
| 4 drafts/unknown | description forceMount + CSS inactive display:none; uploadDialog lock callback блокирует X/Escape/outside; quick-look read-only | Source Radix implementation и call chain проверены; browser подтверждение передано128 |
| 5 create query | createDraft с allowlist полей и enum/defaults; create options читаются с tasks page | Unit PASS, protection fields не подменяются |
| 6 upload scope | uploadDraft нормализует пустой scope в project и наследует task/epic scope/key/title существующей версии | Четыре новые unit regression PASS; native guards неизменны |
| 7 body clone | proxyClientMaxBodySize ровно51*1024*1024 +130000ms; Go50МиБ/120sec неизменны | Проверены config, installed Next implementation и типы; wire boundary переданы128 |

`tsBuildInfoFile` направлен в `.local/tsconfig.tsbuildinfo`, прежний ignored
cache вне `.local` удалён автором; собственный полный tsc после этой дельты PASS.
В Go остаётся один mutation journal/runAction с исходной session и payload,
native Update продолжает проверять requester/lease/generation; optimistic
claim/transition или новая browser bearer storage не появились.

**PASS:** собственный финальный frontend unit run27/27 (Node24.19.0), включая
8 API/grouping/revocation и19 form/state tests; полный TypeScript check PASS.
Авторские новые Go и TS тесты прочитаны и приняты; специально проверены
неизвестный исход, changed replay, чужая lease, expiry и source DTO boundary.
Собственные Go web/race checks PASS, cached, как указано выше.

Frontend manifest `.local/task007-frontend/source-manifest.txt`:
SHA256 `d8e0018597e0ecb1734a1c50ce08b19309b5cafe5349676d644925f040f8e008`.
Reviewer самостоятельно пересчитал72/72 файлов: несовпадений0.
Дополнительная scoped fingerprint117 regular tracked+nonignored untracked
файлов `internal/web` и `web`:
`acf64b4bc276c1b03e6acb5a5e06c067c6319c468e2dac4a1c8ae0ea7acd7010`.
Алгоритм: sorted unique Git paths, для каждого path + NUL + lowercase SHA256
содержимого + LF, затем SHA256 объединённого manifest. Generated dependencies,
runtime outputs и документы не входят. Root фиксирует полный application manifest.

Production build автора125 PASS до узкой дельты пути TS cache; reviewer увидел
готовый BUILD_ID, но не объявляет этот build независимой browser проверкой.
Root финальный общий build/check и runtime artifact hashes относятся к TASK-007.

## Повторный source review reliability delta

Проверены `read-deadline.ts`, `dashboard-polling.ts`, `hooks/use-dashboard.ts`,
обновлённые `api.ts`/`OverviewScreen`/`ControlContent`, `next.config.ts`,
root mutation deadline/useAction и все добавленные авторские unit tests.

| Пункт | Итоговое исправление | Повторная проверка |
| --- | --- | --- |
| 8 | Whole-read8sec timeout; combined parent cancellation; failure backoff30/60/120sec и bounded Retry-After; cancel hidden/offline/pagehide, resume eligible; manual refetch cancelRefetch:false, terminal401/409 | Source callback/Query lifecycle и5 unit cases PASS |
| 9 | Whole mutation deadline15sec, upload130sec; timeout остаётся unknown и сохраняет exact pending payload | Root4 unit cases самостоятельно PASS, включая stalled JSON после200headers |
| 10 | Fatal page только если last-good data отсутствуют; ошибка refetch с data выводит notice поверх стабильной страницы | Source проверен, отдельная browser draft регрессия128 ещё ожидается |
| 11 | Старые6/20 limits, даты, agent/task/event entry validation восстановлены; nil collections безопасно нормализуются |2 tests PASS, включая настоящий QueryClient, где malformed response не заменяет prior data |
| 12 | skipTrailingSlashRedirect:true сохраняет Go `/web/`; appToWeb нормализует nonroot trailing slash | Canonical mapping/root GET unit cases PASS |

**PASS:** собственные финальные frontend40/40 unit tests и полный tsc под
Node24.19.0. Новые tests проверены отдельно от их авторов; тестовые global fetch
восстанавливаются, deadline timers очищаются. Ранее source findings больше
не открыты. Авторский format PASS подтверждён root; HTTP/browser evidence
не подменяется этими mocked unit checks.

Reviewed scoped source124 regular tracked+nonignored untracked файлов
`internal/web` и `web`, тот же path+NUL+file-SHA+LF алгоритм:
`1b78986b25b44fd3395aa1ded0e3021fc27570a6b658bfe809698784c4e9ee7b`.
Прежний282-file artifact1 manifest `b3f06d07d9a631520cf25b4dc1e5cb43804315030ed264667832fef478810620`
reviewer самостоятельно проверил282/282 source и3/3 artifact hash без расхождений,
но он **устарел для reliability delta**. Root обязан выпустить artifact2 anchor;
acceptance artifact1 нельзя назвать финальной проверкой нового source.

Artifact2 закреплён root на полном commit
`560b10b60408d85ffd5346cfdff40a046ac656ab`; manifest
`.local/task007-freeze/artifact2.json`, SHA256
`a21c9f6bcba801235f10077f3bc969ad047e403b34c474963f2a8f602171a850`.
Reviewer пересчитал289/289 source и3/3 artifact hashes, расхождений0.
Go binary SHA256 `6e39732486ceadb6cf48f634d69e4bd7baca2fb47b6eeea23f79de5c5d04ee06`;
Next BUILD_ID `wS1GmOKvwDegkZ5IsL_zT`. Новый browser acceptance ожидается.

Пункт13 source fix принят: `browserRewrites` выдаёт literal `/web/` →
upstream `/web/` перед wildcard. `appToWeb` сохраняет прежнюю root/query
семантику, validation остаётся Go. Самостоятельный actual installed
`prepareDestination` regression2/2 PASS: literal сохраняет slash и все
query (включая unknown/duplicate view), wildcard пустого пути действительно
терял slash, nonroot сегменты неизменны. Source fingerprint125 файлов:
`1bb730c99b7d8db9a19a82d87859ce3a88282cc48db9839c2356da3bb2a5e652`.
Это разрешение root на build/smoke нового artifact, не runtime PASS.

Текущий artifact4: source `eb01113d7732ec30001b94eae810127ab7183443`,
manifest `.local/task007-freeze/artifact4.json`, SHA256
`e86c6285852281a03fe707430052ab83026801f3230d7f4b5dad17b894efce1f`.
Reviewer самостоятельно проверил290/290 source и3/3 artifact hashes:
расхождений0; Next BUILD_ID `7GI1JdDqdsi51130kgYwY`.
Artifact2/3 FAIL evidence сохраняется; новый128 acceptance ещё выполняется.

## Первоначально остававшиеся проверки (исторический checkpoint)

Реальный browser/CSP/hydration, upload50МиБ/+1byte,31sec и120sec boundaries,
полный human workflow/drafts и visual QA — отдельный tester после этого APPROVE.
Production, push/merge/release и физический второй ПК — NOT-RUN.

## Ход review новых process/E2E128 (история)

Прочитаны11 `tests/web/next*.cjs` модулей до окончательного test freeze.
Свои runtime процессы reviewer не запускал:3101 закреплён за tester128.
Сильные стороны: настоящий Go/Next/native MCP, строгий source/binary/BUILD_ID
anchor, subprocess faults вместо fake HTTP, separate CSRF/Origin негативы,
no-event отрицательная проверка, real byte/time upload boundaries и actual
headed visibility. Экспериментальные FAIL сохраняются.

Передано128 до freeze:

- `pushState(?layout=list)` в task detail менял queryKey и использовал
  неподдерживаемый query; это false FAIL для retained same-page refetch draft.
  Автор заменяет на barrier реального GET после успешного heartbeat200;
  route.continue сохраняет настоящий Go, подменённого ответа нет.
- Foreign SID через Worker resource отклоняется в bridge до HTTP; требуется
  отдельный запрос native API с действующим bearer и чужим действующим SID.
- Upload helper вручную задавал scope/key и обходил default/new-version
  initialization; нужны реальные browser asserts без этих ручных подстановок.
- Вручную созданная security auth page должна получить observe до навигации;
  CSP script-src надо проверять отдельно от разрешённого inline style.
- Planned answer/edit/upload visual states и keyboard assertions должны иметь
  фактическое evidence либо явный NOT-RUN в финальном отчёте.

Окончательный test/evidence verdict ещё не выдан; до готовых фактических PNG
reviewer не объявляет визуальное APPROVE.

Freeze01 `.local/task007-qa/test-freeze-01.json`, SHA256
`65b1abdd06b6f049bcaf70fb8009840d5286f1aabb7f887b3b1774391ce80667`:
reviewer проверил11/11 hashes и syntax каждого файла; PASS.
Ранние test findings исправлены: настоящий same-key GET barrier после
heartbeat200; native HTTP foreign SID; default/version scope assertions;
CSP script-src без unsafe-inline и observe security page; новые screenshots.
Отдельный `artifact4-journeys-02` PASS5/5 и clean Next exit143 проверены по
JSON evidence; это частичный scope, не полный монолитный прогон.

Cleanup test finding закрыт: terminal status проверяется также при уже
завершённом Go/Next; exit1/SIGKILL больше не проходят early return.
Текущий11-module test source **APPROVE**, freeze05
`.local/task007-qa/test-freeze-05.json`, SHA256
`3b0569b005897baaa6a713d82604cf7e80fa2a6918e330947c043dd82a9c63ae`:
reviewer самостоятельно11/11 hash и Node24 syntax PASS.
Known focus/sidebar findings собираются отдельно и оставляют finalFAIL;
это позволяет проверить остальные реальные deadlines/security без falsePASS.

Artifact4 full04:9 checks PASS, затем реальный command focus FAIL.
Отдельный focus probe подтвердил BODY через1.5sec после Escape.
Full05 выполняется на том же frozen продукте; окончательного runtime и
visual verdict нет. Следующий presentation delta требует повторного source
review/build и реального keyboard/affected-screen retake. Ранее пройденные
upload/API проверки можно сохранять только при доказанно неизменной цепочке.

Presentation proposal125 предварительно **APPROVE к применению**:
ignored `.local/task007-localization.patch`, SHA256
`06acb1bb3be12a698d931a1c797a666edfc66285f96e7cc9f2a4d7ac3af453c0`.
27 labels/filter options совпадают с core allowlist; sidebar height100%
сохраняет scrollable nav; shared focus handler сохраняет callbacks и
defaultPrevented, проверяет live/visible opener, отдельные Peek/Drawer охвачены.
Продукт во время full05 не меняется. Это pre-review patch, не утверждение
реализованного fix: после apply нужны обычные checks/build и настоящий
Escape/focus return всех dialog, включая переход Drawer→Command.

Reviewer осмотрел full05 overview1440dark, tasks390dark, task768light,
edit1440 и upload1440 PNG. Читаемость/перенос длинного literal title и
непрозрачные settled dialogs подтверждены; overview подтверждает два P2.
Новых blockers на этих пяти PNG нет, полной visual матрицей это не является.

Applied presentation8-file delta **APPROVE к final build**: совпадает с
pre-reviewed patch. Artifact4 manifest comparison показывает0 изменений
backend/native/auth, browser api, hooks/deadlines, upstream/config/lock.
Поэтому `artifact4-remaining-01` сохраняет самостоятельное доказательство
upload50МиБ=52428800/SHA, +1byte413,31sec31024ms и120sec120015ms→500;
readies/pending/reservation/staging cleanup assertions PASS.
Overall remaining01 остаётся FAIL: ожидаемый чужому SID status в тесте был
ошибочно403. Reviewer тоже ранее предложил403; исправлено чтением
`store.sessionFor`→unauthorized и `transport.publicError`→401/UNAUTHORIZED.
Это test-only ошибка, auth не ослабляется; окончательный security rerun ждём.

Artifact5 source `45b787ab4b4099adc6832f7cb85e01e99620cf5c` независимо
подтверждён291/291 source и3/3artifact hashes, manifest SHA
`b55ad751d1e583a27c6443c5e815d2f06e572fb0c09a012ba86db3742327129f`.
Test freeze08 SHA `8c611fb56f8922087aa82c0ca3c44a6ba689bbf739bab028af4e8ac3d04b4dcb`
проверен12/12hash и syntax; новый focus/sidebar/27filters модуль APPROVE.
Final01 получил9PASS, затем нашёл реальный mobile accessibility P2:
Tasks и Designs прячут `.desktop-label`, оставляя header plus button без
доступного имени. Semantic selectors сохранены. Два явных aria-label
(`Новая задача`, `Загрузить`) independently source **APPROVE**; следующие
build/retake ожидаются, final01 FAIL не переименовывается.
Reviewer artifact5 PNG подтвердил полную высоту sidebar, русскую activity,
читаемые320command/390drawer, mobile answer CTA и light desktop rows.

Artifact6 source `f995364e1096dc1a26417d8e8c6a75095eb6abcb` проверен291/291
source и3/3artifact hashes; manifest SHA
`028afee5cb1a087ef751cee39ecfabcfe95223e0e9b11bdc9e5cd0e595bdc796`.
Final01 дал10PASS (включая91PNG), затем route focus immediate FAIL;
final02 после bounded1.2sec также BODY — настоящий P2. Причина в source:
cross-path read Loading убирает Shell; восстановленный старый opener исчезает.
Новый ready-main focus должен работать лишь при BODY, не трогая живые поля,
dialog или same-screen background draft. Source fix125 независимо APPROVE:
16-line `use-route-heading-focus` вызывается до conditional returns, только
successful nonplaceholder nonlogin page, once-per-path marking до BODY guard,
ready h1 tabIndex=-1/focuspreventScroll. Same-path поздний refetch не забирает
живой focus. Следующий build7/runtime ещё ожидается.
Freeze10 passive settled probe reviewed; freeze11 SHA
`b25b780e0cbdb12f1ed5aeff96574ef1a17ca898af7c35b06b7ae0cca51960e1`
сохраняет route failures в findings и не выдаёт groupPASS; overallFAIL.
12/12hash совпали. Продолжение final03 нужно для ещё не пройденных security,
а не для переименования известного FAIL. Reviewer осмотрел дополнительные
artifact6 agents1440dark, questions390light, history1440dark, settings390dark:
новых visual blockers нет, pending route focus этим не закрывается.

Build7 source `5cc0a88f74a0040540dcb111466696314810ca8f` независимо292/292
source и3/3artifact hashes PASS; manifest SHA
`e19d708794c944062014a798c4b0d7e6ecda0986ce7fa359106cc5a383bf0231`,
Next BUILD_ID `zSw1a3XnegMZCIwt6Wlk_`. Protected backend/API/deadline/upstream/
config/package source неизменны с actual upload boundary artifact4.
Final test freeze12 **APPROVE**, SHA
`cc9ffa2822e92482c7151b099ae9d37f26475ab1279635d3e338c8d85c25129d`:
12/12hash и syntax PASS. New transport assertion проверяет active answer после
real same-key8s GET abort. Security bootstrap ждёт настоящий GET200 и
enabled existing input, исправляя преждевременный probe. Route focus остаётся
строгим usable assertion; никаких forcefocus вместо результата нет.
Final artifact7 runtime завершён; окончательный verdict ниже.

## Окончательный independent verdict

**APPROVE** product code,12 E2E modules и локальное evidence TASK007.
Reviewer не писал product/test implementation и не запускал дублирующий runtime;
самостоятельно проверены hashes, source/test contracts, JSON results и actual PNG.

- `artifact7-final-01/next-results.json`:21/21 PASS, scope `no-deadlines`, SHA
  `20bccb1dd526451eba8d7e9a772b694d1b0ea66cc8b05aea99ce2679ab9e6974`.
  Source292-files manifest и12 test hashes совпадают с reviewed freeze12.
  Findings/page errors/CSP-hydration console/static assets failures пусты;
  cleanup failures отсутствуют, Next exit143 допустим и Go stops проверены.
- Настоящие claim race200/409, stale draft409, session expiry409→explicit
  continuation и token revoke401→очисткаprivate UI PASS. Same-key GET8s
  сохраняет active answer/draft, mutation15s→same-body retry даёт один event.
- Route focus после Command→detail теперь H1 во всех767/390/320; dialog
  Escape/input/create/edit/peek/upload/preview и Drawer→Command PASS.
  Sidebar900/500px/768px geometry+scroll PASS;27 real event filters PASS.
-106 actual PNG сохранены. Reviewer осмотрел representative10screens,
  light/dark/mobile/desktop и dialogs по ходу review; финальные low-height
  sidebar768×500, route320 и localized activity PNG также просмотрены.
  Известных visual blockers больше нет. Наличие остальных PNG само по себе
  не называется сплошным106-image pixel audit.
-50МиБ/SHA, +1byte413,31s31024ms и real120s120015ms cleanup — отдельные PASS
  assertions `artifact4-remaining-01`, overall того старого запуска остаётся
  FAIL. Их применимость к финалу подтверждена source identity backend/API/
  deadline/upstream/config/lock4→7; это не новый120s runtime на artifact7.

Исторические FAIL не удалялись и не переименовывались. Deployment HTTPS/VPS,
push/merge/release, физический второй ПК и общий full Paperclip parity остаются
за пределами этого локального APPROVE.

Closure docs TASK007, next-web, tests README и WORK128 прочитаны после
обновления результатов: **APPROVE**. Application/test versions,21 checks,
106 PNG, отдельный old-build physical upload и NOT-RUN boundaries изложены
точно. Root может закрыть TASK и локальный Git checkpoint; дополнительных
product/tests/runtime работ по обязательным findings127 не осталось.
