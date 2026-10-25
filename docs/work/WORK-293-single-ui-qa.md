# WORK293 — независимая QA единственного UI

Исполнитель `validate_candidate`, независимый от backend автора
`release_prepare`. Область: эта карточка, после root source APPROVE —
`tests/web/upload-process.cjs`, узкое ожидание в `next-security.cjs` и собственные
ignored `.local/task019-remove-legacy-ui/qa/`. Продукт/WORK291/общая TASK/Git и
production этим исполнителем не меняются.

## План до source APPROVE

Прочитаны существующие real-process `harness.cjs`, `upload-process.cjs`,
`next-fixture.cjs`, `next-upload.cjs`, `next-security.cjs`, `next-acceptance.cjs`
и прежний WORK128. Новые process/E2E пока **НЕ НАПИСАНЫ**, runtime **NOT-RUN**.
Ожидается независимый backend source APPROVE и отдельный root signal.

После него login/формовые capabilities в upload-process переводятся с HTML
парсинга на Accept JSON: csrf/action_id из настоящего DTO, JSON login200 и
Set-Cookie, те же multipart fields/Origin. Exact50MiB ожидает JSON200 и
50MiB+1byte —413; успешный JSON upload дополнительно сверяется по server asset
size/SHA, если текущий DTO это позволяет без расширения продукта.
Настоящий незавершённый upload должен завершиться сервером через120s в прежней
118–128s tolerance, без клиентского timeout вместо проверки. DB pending/reserved
и staging должны остаться0; readiness после deadline сохраняется. Cleanup только
собственных процессов, все failed attempts сохраняются.

Existing `next-upload.cjs` уже работает через JSON и содержит реальный31s
streaming200. По уточнению root этот сценарий добавляется в узкий direct-Go
upload-process наряду с exact50MiB/+1byte, actual120s incomplete и
reservation/staging cleanup; прежний Next runner не запускается и не чинится
целиком. Нельзя заменять сроки коротким stub/timer или выдавать старый binary
за actual candidate. В `next-security.cjs` stale heading «Добро пожаловать» будет
заменён на актуальное «Войти» без изменения продукта и guards. Strict JSON
контракт дополнительно проверяется text/html POST406 без успешной мутации.

После source APPROVE новые/изменённые tests проходят отдельное independent review
`review_changes` до actual запуска. Сохраняются source/binary/Next buildID/test
hashes и Node/Chrome versions; fixtures используют только synthetic private data
в ignored QA_DIR. Зависимости не добавляются, реальные credentials не нужны.

Browser acceptance: actual keyboard-accessible choose/replace/cancel/remove,
отдельное empty/name состояние и empty submit, возврат фокуса; содержимое token
не выводится. Отдельно отсутствие fallback links, маршрутизация старых UI URLs
в единственный Next UI, JSON/auth/CSRF/Origin и download protections. Exact
unknown retry/locked форму проверяет existing suite; новый proof заявляется
только по фактической проверке. Этот tester ранее автор узкого picker UI; его
source независимо принят `review_changes`, а backend пишет другой исполнитель.

## Известные ограничения прежнего runner

Старый `next-acceptance.cjs` security/remaining scopes намеренно добавляют прежние
presentation FAIL findings, поэтому их нельзя выдавать за пригодный новый общий
PASS. `next-fixture.create` также использует прежние `[name=type]` и кнопку
«Создать»; текущий диалог имеет отдельные имена «Создать задачу/эпик». По уточнению
root этот общий runner не запускается, исторические findings сохраняются.
Предложение нового общего runner отменено; продукт под старые assumptions не
меняется.

Existing Next fixture привязан к Go3101 и production Next build rewrite3101;
release bundle7892 нельзя бесшумно использовать вместо него. Преддеплойный
process gate использует реальный Go с динамическим loopback port и новым private
state. Picker/UI links/redirects проверяются отдельно postapply actual HTTPS
gate без создания tasks/projects/workflows, не через старый Next fixture.
Чужие процессы не останавливаются.

## Test source после root APPROVE signal — review pending

Root передал independent product static APPROVE `review_changes`; после этого
изменены только два порученных test файла. `upload-process.cjs` получает real
JSON login/DTO capabilities, проверяет strict text/html POST406 без asset
мутации, actual50MiB200 с server size/SHA и +1byte413. Добавлен actual31s stream
с прежним ZIP payload и проверкой size/SHA. Сохранён actual120s incomplete,
118–128s server tolerance, DB pending/reserved и staging0, readiness/own cleanup.
В `next-security.cjs` изменён ровно stale login heading на «Войти».

Syntax и diff check — **PASS**. Test source передан `review_changes`:
upload SHA256 `d207581fd633f7fa54d7c370e6ce53ba686b1f5afb24fa4d5ef5c19467a44e74`,
security SHA256 `b33d7f4a65e7c4bf052965a9c51542d66c225398055534a5004cb52b7b8c75ed`.
Actual process execution пока **NOT-RUN**: требуется independent test APPROVE
и fresh actual binary от root. Общий прежний Next runner не запускается.

## Actual candidate/process/build — PASS

Исторические planning/PENDING/NOT-RUN checkpoints выше сохраняются. После них
получен independent test source APPROVE `review_changes` с exact upload/security/
harness members; receipt `.local/task019-remove-legacy-ui/review/test-source-approval.json`,
SHA256 `64d117e7f88b261daead855fababc31915b40735cee5c38fb60807ddbdf61313`.
До запуска сверены source/test/report hashes и clean candidate
`fa5f5cca9cf72edab6d81affe5d51d0bdc411848`. Dedicated actual Mac binary SHA256
`5577afe6a6097d7bca3b77a3416222b11577930025814e9327e688691bdbc0d6`,
actual buildinfo подтверждает darwin/arm64 и `vcs.modified=false`.

Fresh ignored run `qa/process-1792933820975146000/` под TASK019 дал **FULL4/4 PASS**:

- Non-JSON valid-Origin POST406/JSON_REQUIRED без asset mutation.
- Exact52428800bytes200 с server size/SHA;52428801bytes413.
- Actual slow stream **31025ms200**, server size22/SHA совпали.
- Actual incomplete upload **120011msECONNRESET** внутри прежней118–128s
  tolerance; readiness200, DB pending/reserved0 и staging0, own server exit0.

Это direct-Go loopback real-process проверка; Next rewrite deadline этим run
**NOT-RUN**. Общий старый Next E2E не запускался. Report `upload-results.json`
SHA256 `5e5ce7cabed6138c112138661b1b2187d690aa146e34758a1fd8c2c90cd6ce2a`.
Source/test hashes после runtime совпали; tracked source чистый, отдельно был
зафиксирован новый untracked WORK294 другого исполнителя, без product drift.

Независимая проверка root-produced artifacts сверила720source members,
51deletion tombstones,55Go/1257Next и Mac binary, все manifests/member/source/
producer/log hashes, обе actual Go buildinfo clean; Next upstream7892 и BUILD_ID
`ipLB8vqDOb2bl6SWl7Oep`. Сверены фактические root logs full Go unit, infra174,
web109/type/format, fmt/vet/race/build, Linux/MacGo и production Next/package.
Тестер не создавал эту сборку и не подменял root log собственным повтором.

Exact candidate approvals:

- `review/build-approval.json` под TASK019, SHA256
  `74e0d588623babfbfc5fe33fad09f51725f69496c267c3233dfacdfd6fe20571`.
- `qa/candidate-approval.json`, SHA256
  `fd540888388be8a9b3c789eb12e5f1bcb2d7ce7e254e376882adb1cfea27c82a`.

Receipt `runtime_go=false` явно означает отсутствие production operational GO
authority. В bound report отдельно записан `local_go_process_executed=true`:
реальный локальный Go действительно выполнялся. Backend author и tester разные;
своё авторство narrow picker UI tester не скрывает, его source independently
принят другим reviewer.

## Actual postapply HTTPS/native Chrome UI — PASS

После root APPLIED/AUDIT exact candidate signal выполнен предварительно
независимо принятый HTTPS script SHA256
`5f7f63b6f75a0342bacdf3e4e7b963b44950d837c4d6210fece35574421018d7`.
Его independent receipt `review/https-test-source-approval.json` под TASK019
имеет SHA256
`6a86dd39e1eab402eeeaf1d146254e7bab68c0af65acc5a344bc1d906721dd2d`.

Strict TLS/no automatic redirects: **PASS**12served asset hashes exactBUILD_ID,
4scopedpages плюс designs, fresh owner JSON login/catalog switch,
6stateless/cookie-free legacy303 canonical/scoped/catalog/shared-login/host-scope
query redirects, old JS/CSS404, valid-Origin JSONless POST406 и foreign-Origin403,
unauthenticated binary401 без redirect. Production tasks/projects/workflows/
managed runs или policies/settings/grants не создавались и не менялись.

CUA IAB и Chrome browser-provider оказались unavailable, inventory browsers[].
Root явно разрешил безопасный fallback. Через documented native CUA getApp
запущен ранее закрытый Chrome; создано собственное incognito окно. Safari и
существующие пользовательские окна не использовались. Сначала native URL typing
дало ошибочный404; после наблюдения pasted exact known URL фактически перешёл
из `/web/login` в `/app/login` с актуальным UI. Это диагностическое исправление
не выдаётся за initial navigation PASS без ошибки.

Actual browser **PASS**: отдельная кнопка/статус/hint, отсутствие fallback ссылки
на login, локальная русская empty-file validation; Shift+Tab фокусирует chooser,
Enter открывает настоящий OS picker; synthetic file chosen/name отдельно;
замена на длинное synthetic имя с переносом без overflow; remove очищает статус
и возвращает фокус кнопке выбора. В собственном native Chrome picker также
нажата настоящая **«Отменить»** после выбранного synthetic файла — прежнее имя
сохранилось. Следовательно первоначальный accepted native-cancel NOT-RUN был
позднее закрыт actual PASS, а не `setFiles([])` или предположением.

Синтетические файлы не отправлялись на вход; production authentication отдельно
проверена настоящим HTTPS JSON owner login. Native-browser successful login,
readonly participant browser и successful production binary download —
**NOT-RUN**; no new production assets созданы. Local Go file integration/security
и unauthenticated binary routing проверены в указанных областях.

Оба собственных окна (incognito и созданная при запуске blank вкладка/окно)
закрыты. Содержимое токенов не читалось UI-кодом и не выводилось. Screenshot
`qa/https/01-picker-empty.png` под TASK019 содержит пустую форму для пользователя;
`03-picker-replaced.png` с synthetic long filename фактически осмотрен и принят.
Screenshots/AX/HTTP reports и approvals остаются private ignored и immutable.

Fresh `independent-https-smoke` PASS/APPROVE:
`qa/https/approval.json` под TASK019, SHA256
`ab5b20ac0d0837b240ea31a55f0873712d3dc1e909e05245a332edaff4d12820`.
Bound actual report `qa/https/report.json`, SHA256
`4b7bc61e9baa431d21f774c608f595c9e165691e7c742de79820757f19a19367`.
Root уведомлён о завершении browser checks до finalize. Deploy/finalize/SSH этот
тестер не выполнял. Windows/Claude execution/второй физический ПК — **NOT-RUN**;
общий full V1 этим срезом не объявляется.
