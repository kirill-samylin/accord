# TASK019 — единственный интерфейс и понятный выбор файла токена

**CLOSED / PASS: server/Next release fa5f5cc.** Полный V1 не заявлен.

Поручение25.10.2026: полностью удалить из кода упрощённый HTML/no-JS интерфейс,
включая вход `/web/login`, и исправить непонятный нативный выбор файла токена.
Публикация и стенд продолжают ранее порученный цикл в исходном origin
`https://github.com/kirill-samylin/accord.git`, ветка `codex/native-preflight`.
Root — координатор; WORK290 backend removal, WORK291 token UI, WORK292 review.

Единственный UI — Next `/app`. JSON `/web` endpoints, cookie login, CSRF/action-ID,
leases/epochs/ACL/capabilities и scoped downloads/preview сохраняются. Известные
legacy GET/HEAD pages перенаправляются на corresponding fixed relative `/app`
адрес без выдачи page action nonce; неизвестные страницы/static404. NonJSON POST
отклоняется406 до body/мутации, foreign Origin по-прежнему403. Native REST/MCP
и файлы не становятся HTML или публичными redirects.

Полностью удаляются internal/webui templates/assets/JS/CSS/fonts и Renderer/
Static wiring. Next fallback links/menu/noscript/error fallback удалены. Старые
SSR-only browser/control/refinement тесты удаляются, shared Next/lifecycle/upload
coverage сохраняется; meaningful backend/security assertions мигрируются на JSON.
Исходный TZ и исторические evidence документы не переписываются.

Выбор токена: custom accessible кнопка, отдельное имя/статус, hint, замена/очистка;
содержимое файла не читается JS, не сохраняется и не выводится. Multipart4096byte
граница, CSRF и точный повтор uncertain запроса остаются. Independent token UI
source review APPROVE; typecheck/109tests/format PASS на промежуточном дереве,
actual browser/candidate/build/deployment пока NOT-RUN.

Baseline production — bdcabbd, schema6. Только coherent capable all-project
Go+Next update с format2backup, preservation/history/ACL/floors и roll-forward.
Новый кандидат не устанавливается до independent source/tests/ops approvals.

## Source готов к local candidate commit

Независимое source review — APPROVE/PASS review_changes;720members и51deletion
зафиксированы отдельно от evolving docs. Новые Go regressions проверяют actual
single/registry HTTP routes, no-cookie/no-nonce stateless redirects, canonical
scope/unknown404, unread body nonJSON406/foreign403, no logout mutation и binary
anonymous/revoked401. Прежние JSON auth/CSRF/replay/ACL/download byte assertions
сохранены. Full go test ./... PASS12testpackages, web109/type/format PASS,
infra174 PASS. Полный race/vet/build и actual process/HTTPS приёмка ещё идут.

При полной infra проверке обнаружены3 прежних stale fixture ожидания: unknown
schema6 после появления6 и fake native ACK без реальной миграции/reset в4/5
copy tests. Исправлены только tests: future7/history6 и проверка actual rejection
с неизменными copied bytes/boot guard/no RESET_COMPLETE. Production infra не
изменялась; successful native restore6 этим тестом не заявлен. Первоначальные
FAIL логи сохранены, independent review тестовых исправлений APPROVE.

Независимый tester после product approve мигрировал upload-process.cjs на
JSON/Accept и406, сохранив50MiB/+1byte/31s stream/120s timeout cleanup. Изменено
только stale heading «Войти» в существующем next-security. New test-source
review APPROVE до actual runtime; общая починка старого Next E2E harness не
входит в этот выпуск. Actual native Windows/Claude execution/второйПК NOT-RUN.

## Exact candidate, сборки и process QA

Candidate `fa5f5cca9cf72edab6d81affe5d51d0bdc411848` clean committed/pushed в
исходный origin/codex/native-preflight. Source approval bound к720Git blobs и
51deletion tombstone; product/Go/unit/infra delta frozen. Изменённые основные
файлы: internal/web/navigation.go/http.go/contracts.go/project_bindings.go,
internal/projects/browser_router.go/runtime.go/types.go, cmd/agent-board/main.go;
полностью удалён internal/webui (40файлов) и11SSR-only process suites. Next:
LoginForm.tsx/LoginForm.module.css, shell.tsx/control-app.tsx/page.tsx/error.tsx,
удалён fallback-only auth-footer CSS. Makefile/актуальные guides обновлены.

Full go test ./... PASS; fmt/vet/race/build PASS (store race105.858s), web109/
type/format PASS, infra174 PASS. Go Linux+Mac buildinfo exactfa5f/vcs.modified=false,
Next production upstream7892; Go55/Next1257 manifests и members independently
verified. Source/build/predeploy QA/test-source/ops независимые APPROVE.
Mac test binary SHA5577afe6a6097d7bca3b77a3416222b11577930025814e9327e688691bdbc0d6.
Это native Go HTTP process fixture, не native Codex/Claude выполнение требования.

Новый real process full4/4 PASS: nonJSON406 без body/мутации;50MiB200 с exact
size/SHA,+1byte413;31s stream200 с size/SHA; incomplete upload120011ms с
ECONNRESET, ready200, pending/reserved/staging0 и owned process cleanup.
QA receipt `fd540888388be8a9b3c789eb12e5f1bcb2d7ce7e254e376882adb1cfea27c82a`.
Original failed/superseded preparation evidence не переписывалась.

Исполняемый reviewed ops manifest:
`b2044a9a1dd42a2d86f0257c6f3cc2f5db3fd9bf44dde095aefa9e565dc334aa`.
12members/21actualrefs/Go55+Next1257 archives; previous unexecuted manifests
сохранены без runtime. Independent ops approval
`2b8cd28f1f35b70b55fbb9a8979a58b6007e11edbda18239616a1c5986885604`.
Actual upload начат; production apply/HTTPS/finalize ещё не заявляются.

## Actual application

Upload-go1792934378604423000/upload-next1792934392284807000 — exit0,
active_changed=false. Preflight1792934446075220000, stage1792934488315234000,
fresh preflight1792934524904885000, apply1792934538958054000,
audit1792934600279965000 — exit0, exact reviewed M b2044a9a…/candidatefa5f.
Live baselinebdcabbd/4boards/9runtimeDB/9historicalDB подтверждён; busy counters0
на всех4boards, UID999 WAL-aware connections закрыты, guards/helpers/ACL/floors
сохранены. Canonical updater создал validated prebackup перед переключением,
9DB logical rows/history сохранены, оба Go/Next actualfa5f. Journal пока
pending-https-smoke; apply ещё не означает завершённую поставку. Независимая
внешняя HTTPS/browser приёмка выполняется отдельно перед finalize.

## Actual external HTTPS и browser

Fresh external QA exactfa5f — PASS/APPROVE:
receipt `ab5b20ac0d0837b240ea31a55f0873712d3dc1e909e05245a332edaff4d12820`,
report `4b7bc61e9baa431d21f774c608f595c9e165691e7c742de79820757f19a19367`.
12asset hashes exact candidate/BUILD_ID,4scopedpages+designs,
6known legacy redirects303/no cookies, legacy static404, nonJSONPOST406,
foreignOrigin403, anonymous binary401/noLocation — PASS. Только owner JSON
login/switch; tasks/projects/workflows/runs не создавались.

IAB browser provider оказался unavailable; новый browser panel был queued.
Тестировщик через documented native CUA открыл свои Chrome окна (до этого
Chrome был закрыт) и отдельное incognito QA окно. Safari/пользовательские окна
не трогал. Actual oldlogin→/app/login, отсутствие fallback, keyboard ShiftTab/
Enter chooser, select synthetic file1, real native cancel с сохранённым file1,
replace на длинное synthetic filename/wrap, remove→blank+focus return,
empty submit validation — PASS. Это реальный native Cancel, а не setFiles([]).
Собственные окна закрыты, больше browser/runtime действий не требуется.
Root отдельно просмотрел actual empty/replaced screenshots: readable/no overflow
на наблюдавшемся desktop viewport. Mobile viewport этим прогоном NOT-RUN.

## Canonical finalize/final-audit

Независимый external evidence approve
`07563fa4244c0c6bc55a6a236f2cbbd7d5fd3ffee279b78b8cbe81bb0f3d297b`
принят перед finalize. Canonical finalize1792935200908096000 и
final-audit1792935273194347000 — exit0; fresh all-project busy guard выполнен
до backup, journalcomplete, owned Go+Next exactfa5f. Принудительных остановок
пользовательских работ для получения окна не было.

Новый validated format2 backup manifest
`ac4fe2ff07cc215338be36c02472b2bbd2b62fa6a3e40ba2429b93b68ed92700`:
4boards/9runtimeDB/9historicalDB, inventory/history preserved, logical_rows9DB
preserved, schema6/ACL/floors сохранены; backup service success/status0.
После finalize отдельный HTTPS GET `/app/login` с проверкой TLS — HTTP200.
Fresh scheduled timer tick отдельно NOT-RUN. Native browser valid-token submit,
viewer-specific production flow, successful production download, mobile/new
Windows/Claude execution/второй физическийПК и общий fullV1 не заявляются;
positive multipart login/download уже покрыты соответствующими Go проверками,
external JSON owner auth также фактически PASS.

Итоговая независимая приёмка — **APPROVE/PASS** review_changes, exactfa5f:
receipt `0977769d1030634c08e550a95b4d57289a8f4a533d500843f536f2b7f84f3f92`,
immutable report `8ecf959aa3734d6bc3ea0302bbeb2b5a18c2800c8504c2bc96e670316c0b4547`,
70input evidence manifest сохранён с private approval.720source/51deletions,
12ops/21refs/9results+9attempts проверены; source/build/test/QA provenance не
переписывалась. Source code/artifacts принадлежат fa5f; следующий docs-only
commit фиксирует acceptance без новой замены binary. TASK019 закрыт в рамках
удаления старого UI и выбора файла; полная V1/расширение native scope не заявлены.
