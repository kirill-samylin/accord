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
