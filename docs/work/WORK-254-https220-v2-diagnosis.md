# WORK-254 — HTTPS220V2: диагностика и контракт узкой V3

Статус: DIAGNOSIS COMPLETE; prospective SOURCE CONTRACT REVIEW PENDING.
Автор: gui_connection_qa. Только чтение сохранённых данных и исходников; новых
HTTP/browser/native/CLI/provider запусков, чтения токена и изменений production нет.
Исходный V2 FAIL и все его артефакты сохранены неизменными.

## Проверенные факты

- V2 source manifest: `96a3e6665190bfbf7672c8b8169ee22ca5a407676b941802e274e7e58471ee5e`.
- V2 results: `5166c850dfa0531ff7af2475bb0dd24a8b1d7a6d5298ba62db9df3625e7db1a9`.
- Strict TLS, вход, исходная Mac identity/owner membership и шесть scoped страниц
  имеют сохранённые успешные проверки/снимки. Последний снимок — automation;
  `readonly-form-states.private.json` ещё не создан.
- Сохранён только `failure.phase=readonly_https`, `type=Error`: точный упавший
  вызов и stack НЕ сохранены. Их нельзя объявить непосредственно наблюдавшимися.
- 11 requestfailed: 2 anonymous + 9 authenticated; причины/пути/методы НЕ
  сохранены, следовательно причина каждого UNKNOWN. Ни одного blocked/pageerror.
- Две anonymous console 401 соответствуют ожидаемой проверке отказа; authenticated
  console содержит один 409 на первоначальном scoped root. Его response body не
  сохранён. Cleanup complete, Chrome code0/signalnull/drains/join положительны,
  inputs/sources/token unchanged.

Сводка без пользовательских данных и полные SHA входов:
`.local/task014/https220/diagnosis254/diagnosis.private.json`, SHA
`85a7699b7ff483537fb006c57b88059b85bd22119cbe517b40af2228bf9b1cd3`.

## Статические основания и границы вывода

`smoke.cjs` после automation screenshot ожидает exact label «Команда и разрешение».
`web/src/features/automation/requirement-form.tsx` оборачивает SELECT с OPTION в
LABEL. Pinned Playwright getByLabel берёт полный descendant text: короткая подпись
не совпадает с label вместе с вариантами. Это конкретный дефект QA locator;
непосредственная позиция старого runtime throw остаётся UNPROVEN. Такие же
латентные locator имеются у SELECT «Компьютер»/«Рабочая папка». TEXTAREA тоже
следует выбирать через существующий reviewed control helper.

Исходный сценарий сначала переходит напрямую на scoped root и лишь затем нажимает
«Открыть проект». `internal/projects/browser_router.go` возвращает 409
PROJECT_SESSION_REQUIRED при отсутствии project binding. Наблюдённый 409 согласуется
с этой ветвью; его точный error code не наблюдался. Нормальный catalog flow
`catalog-workspace.tsx` → `OpenProject` → `RegistryForm` отправляет существующий
CSRF/action switch и только после успеха переходит в scoped проект.

`use-dashboard.ts` отменяет свои запросы при unmount/pagehide. Это возможный механизм
requestfailed, а НЕ доказанная причина старых 11 отказов. Допуска для них нет.

## Предлагаемый узкий SOURCE контракт V3 (ещё не реализация)

1. В новом каталоге сохранить V2; использовать byteexact reviewed `ui-controls.cjs`
   из 203V13, SELECT/TEXTAREA через exact immediate-label/direct-child selector,
   с count===1 и прежними assertions. Ничего не вводить/не отправлять в формы.
2. После проверки catalog membership нажать единственную кнопку «Открыть проект»
   внутри единственной project-card с точным заголовком Test_internet; наблюдать
   обычный switch POST200 и scoped route/current project. Не заходить в unbound
   scoped root заранее. Никаких общих допусков 409.
3. Перед намеренным переходом можно bounded ждать завершения собственных текущих
   запросов (не более 15s внутри прежних 180s/cap500). Ни отмены, ни retry, ни
   исключения ошибок: requestfailed остаётся FAIL. Новые фоновые запросы не дают
   гарантии отсутствия будущей отмены; новый отказ должен сохранять причину.
4. Перед await сохранять конечный stage ID. При ошибке сохранить только конечные
   stage/type и собственные filename/line/column; без raw message/stack, значений
   форм, query, headers/body/cookies/CSRF/token.
5. requestfailed дополняется context/stage, finite failure-code enum либо UNKNOWN,
   method/resourceType и allowlisted pathname/category без query. Caps сохраняются;
   неизвестное значение не превращается в успех. Console status/location также
   сохранять конечными фактами, не raw text. Все реальные authenticated status,
   page/asset/console/requestfailed ошибки остаются FAIL; anonymous 401 узко прежние.
6. Не менять business mutation запрет, ограничения HTTPS/assets/streaming hashes,
   version/audit/source/token guards,180s+30s, positive owned closure и zero native.

Следующий шаг: независимый review202 этого контракта, затем отдельный root SOURCE
GO на полный минимальный V3 patch/freeze. Это не runtime approval и не PASS220.

## V3 SOURCE READY — 24.10.2026

После prospective CONTRACT APPROVE202 и отдельного root SOURCE GO подготовлен
`.local/task014/https220/v3/source-pins.json`, SHA
`2f84be95e541eba1788fbaa29477d64bc52340edfa9d07cea0bb811eecb331ac`,20members.
Из прежних members изменены только README/browser/main/smoke; preflight/private/
scope/pins/library-pins byteexactV2. Новый ui-controls byteexact approved203V13.
Новые diagnostics/request-tracker/test-projections и lineage/static/diff заморожены.

Реализованы unique catalog-card → обычный CSRF OpenProject POST200, count1 для
четырёх reviewed SELECT/TEXTAREA controls, finite stage/sourceframe projections,
finite method/resource/error-code/path categories без raw message/stack/query.
Actual requestfinished/requestfailed завершают tracker; route/response не считаются
завершением body. Навигация ждёт≤15s внутри180s, без cancel/retry. Все operational
requestfailed, включая ERR_ABORTED и пришедшие после fence для начатого до fence
запроса, остаются FAIL. Projection/overflow/неожиданный HTTP status fail closed.
Только прежний exact anonymous protected-automation401 допускается как expected.
Tracker/route/operation joins обязательны до eventsClosed; positive Chrome0/drains
и прежний cleanup30s сохранены. No new business mutation permission.

Static:10 CJS syntax PASS, max188lines, private dir0700/files0600;20pins match.
Synthetic pure projection/tracker/actual visibleControl uniqueness guards написаны,
NOT-RUN до независимого source review. Никаких HTTP/browser/native/provider/CLI
кандидата/токен-чтения/runtime не было. Старый V2 manifest и FAIL неизменны.

Reviewer command (только после SOURCE APPROVE):
`/Users/shagi/.nvm/versions/node/v24.19.0/bin/node .local/task014/https220/v3/test-projections.cjs`.
Будущий root launch: тот же node, `v3/main.cjs <fresh-private-input0600>`; input schema
и exact `GO WORK220 HTTPS <revision> <newsourceSHA> <freshAuditSHA>` byteexactV2,
нужны новые source202 receipt и свежий independently approved212 audit≤30min.
Эта карточка не создаёт root GO и не является HTTPS/runtime PASS.
