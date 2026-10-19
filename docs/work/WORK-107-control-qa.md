# WORK-107 — независимая process/browser QA control UI

- Родитель TASK-005 / REQ-005; роль tester, исполнитель `/root/control_qa107`.
- Статус: **выполнено — функциональная QA24checks и независимый APPROVE115 финального E2E source**. Полный legacy14PASS один раз и новые control data/layout/poll10PASS зафиксированы ниже с точными scopes/failed attempts. Reviewer115 отдельно повторил data5 и poll4PASS. Product code/security112 и design114 APPROVE получены до final runs; общая TASK/выпуск этим не закрываются.
- Область записи после GO: только `tests/web/**`, эта WORK и private `/private/tmp/accord-control-099/qa/`. Продукт, Go, infra, Git, SSH, live, Tandem и Давинчи не изменяются.
- Прочитаны tester profile, общий процесс/parallel-work/testing Давинчи read-only, самостоятельный AGENTS Accord, control-web-contract §8/9 и поздние уточнения, WORK100/103, прежние QA095/review097 и existing browser/process harness. Tandem OFF по AGENTS Accord. Именованные Claude bindings недоступны; модель наследована, отдельность tester от авторов сохраняется.

## План приёмки

Все fixture данные synthetic. Реальные Go server/native stdio MCP worker/Chrome, literal loopback, private state/token files. Подменять HTTP/domain assertions успешными результатами нельзя.

| Группа | Действия и наблюдаемый результат |
| --- | --- |
| Существующая регрессия | Сохранить 14 browser journeys и domain assertions; адаптировать только login/nav/selectors/routes новой UI семантики. Полный browser core один раз после approve. Не повторять неизменённый upload-process 120s и HTTP 50MiB, уже PASS095/097; сверить source SHA. |
| Overview/search | Создать больше100 задач через native; получить точный total независимо от страницы. Найти поздний hit по title/description/key, mixed-case кириллицу, буквальные HTML/XSS; DOM безопасен. |
| Sessions | Две native сессии разных synthetic owners с distinct run IDs и human browser session. Same-owner concurrency этим runtime не проверяется. Проверить kind/role/owner/state/paging/detail, текущее владение после claim/release. Отозвать synthetic token при будущем TTL: effective count/status/task badge меняются, UI не называет session OS процессом. |
| Tasks/activity | List/board показывают одни отфильтрованные задачи, paging сохраняет layout/q/filters; смена q/filter начинает новый запрос. Activity strict DESC, task/session/type filters, новый insert не повторяет старую границу. |
| Stats/projects/settings | Native completion без tokens и с tokens даёт реальную missing coverage; browser период/group. Один реальный read-only project, effective TTL/rules/settings, отсутствующие process/model/revision данные честно неизвестны. |
| Freshness | Только overview GET poll: visible/online; max1 in-flight, stale retains last good counters/time; offline pause/reconnect; 401/409 terminal stop; GET не renews lease/session и не пишет draft/form values. Failures остаются FAIL, transport probes отмечаются отдельно. |
| Responsive/accessibility | 1440 desktop и390 mobile, нет document overflow; drawer keyboard/Tab/Escape/focus return, theme persists, self-hosted font loads; скриншоты и private connection metadata для независимого design reviewer. |

## Окружение и доказательства

2026-10-19T19:31:04Z: проверены существование bundled Node/Playwright, Chrome, локального Go toolchain и old C2 binary; Node `v24.19.0`. Server/browser не запускались. Новый binary ещё не получен.

Результаты и source/runtime SHA manifests будут в private QA_DIR; токены/HTTP auth/cookies не копируются в отчёты. Фактический runtime scope macOS/loopback, не production HTTPS, не второй физический ПК и не полный developer delivery cycle. Full makecheck/race выполняет root отдельно; здесь NOT-RUN.

## Журнал

- 2026-10-19T19:31:04Z — planning: read-only contract/harness/environment inspection; GO dependency остаётся открытой.

## Начало исполнения после разрешения

- 2026-10-19 — root передал GO после APPROVE106/109. QA binary SHA256 `0fb2f2a78623f659a2be46e68390fc3cd50296b9bb5ece5e1204e11f1d6a3aff`, source-freeze-fixed manifest `7afa42654df715f1df3ddb7b33cf18facf43f86bf3d0589261b779639c0ad4ad`, source `a30746c06ae9379c0fca09ac1757330f0f69ac42`; Go1.27.2 darwin/arm64 buildvcs=false. Runtime источник идентифицирован, это не production binary.
- Написан bounded control suite; прежний browser меняет только login label/nav count10/desktop1440. Harness допускает необязательные fixture TTL с прежними default; upload/lifecycle/state-hash не меняются.
- Bootstrap failures сохранены в `qa/control-initial`, `control-data2`, `control-data3`: tester ошибочно пытался создать epic из developer и прочитать catalog через отсутствующий MCP tool/resource. Core корректно отказал; fixture исправлена на human form + native REST catalog, domain rules не ослаблены.
- `qa/control-data4`: PASS overview112точных totals/search кириллица+HTML/задача после100/paging до следующей проверки. **Настоящий P2:** `/web/tasks?role=developer&state=ready&epic_key=E-2&layout=list` дал400 вместо200; web query использовал T-only task validator для epic_key. Screenshot/failed JSON сохранены, root подтвердил; исправление выполняет другой автор113, затем review112. Assertion сохраняется.
- `qa/control-poll1`: реальные Go/native/Chrome PASS offline→online, одна in-flight, SIGSTOP timeout сохраняет counters/time, recovery, lease и draft неизменны, actual401 revoke terminal stop, actual409 после GoTTL2s terminal stop+explicitcontinue. Hidden acceptance там NOT-RUN (headless Chrome остаётся visible); capability measurement не является hidden PASS.
- `qa/control-poll2`, `control-visibility3`, `control-visibility4`: exploratory hidden probes FAIL; обычный Playwright принудительно включает focus emulation. Это ограничение стенда, не подтверждённый app defect; evidence сохранены.
- `qa/control-visibility5`: **PASS actual hidden/visible** в отдельном headed Chrome private profile, через официальный CDP `noDefaults:true` в новом default context. Реальные background/foreground tabs дали hidden/visible;45s JS timers0hidden requests, после возврата1настоящий Go200. Document visibility JS override не используется.
- Полный прежний14 и final control ещё NOT-RUN: root ожидает accepted CSS111+epic113 batch/review112 и передаст новый final binary. Full makecheck/race выполняет root, здесь не запускаются.

## Финальная версия и фактический результат

- Final GO root: source `ac6c965f3391822b803114efde2a7dcb757ed38c`, независимые code112/design114 APPROVE. Новый binary `/private/tmp/accord-control-099/accord-batch-final` SHA256 `9efa45893de16e369d3a12bd21add9ca9793f891993f229a5af0539250f25a27`; app source manifest `source-freeze-batch.json` SHA256 `451e6888353900b1ba8016bdf2830c185b961b892d2749bb6fc83d10bae50a1c`. Старый0fb для final серверов не использовался. Все свои fixture processes закрыты.
- **PASS14/14, один полный старый browser run:** `qa/final-browser/results.json`. LoginJS/basic, legacy sections desktop/mobile, human/native12/XSS, explicit heartbeat/read-no-renew/correction/answer/history, claim race/stale draft, release400/correction,3scope asset versions/SHA/headers, auth/Origin/CSRF, oversized browser preflight,31s upload, restart/cookie/history/files, offline snapshot/restore auth reset, настоящий oldC2 rollback и new restart/durable hash, fresh login после restore. Full14 не повторялся.
- **PASS5/5 data groups на final binary:** `qa/final-control-paced/control-results.json`, первые5checks. Точный total112 и состояния/blocked/twoepics, поиск позднего hit/кириллица/HTML/key и list/board/filter/cursor,2agent+human/currentclaim/release/futureTTLrevocation/effective counts, DESC event filters/newinsert,2completions/17tokens/2missing из3attempts (66.7%)/singleproject/read-only settings. Этот JSON имеет overall FAIL из-за последующей ошибки селектора самого layout теста; он не подменён успешным.
- **PASS5/5 affected layout/poll groups:** `qa/final-layout-poll/control-results.json`, scope `layout-poll-only`.12маршрутов на1440/390, CSS InterVariable реально применяется, document overflow отсутствует; drawer keyboard focus trap/Escape/focusreturn, theme persistence, WOFF2; visible/offline/oneinflight/SIGSTOPtimeout/stale-retains/recovery/noGETrenew/nodraftrewrite; real hidden/visible tab; реальные401/409 terminal/no-further-polls, explicit protected continue.
- Исторические final failures сохранены: `qa/final-control` (layout bootstrap без применённых CSS при слишком быстром workload; точная transport причина не измерена,429 остаётся предположением), `qa/final-control-paced` (test locator `[data-drawer-open]` после opening совпал также с body). Test исправлен на `button[data-drawer-open]`; workload paced ниже real100rps/defaultburst200, дополнены actualCSS/staticerror assertions. Продуктовые assertions не сняты. В успешном targeted повторе static failures отсутствуют.

Сводное доказательство24checks: `qa/acceptance-evidence.json`, SHA256 `7c526154a63492e9ab86537146a286109653f4fd2aa6460f0c1d0ecfde08482e`.
Frozen final test source manifest: `qa/test-source-final.json`, SHA256 `f02f0e3965e1b2793f115c920d90ae2b808a24eb9ad0aec1efd5ff5b488824c4`.
Каждый JSON сохраняет собственный test SHA и scope; all-files manifests в ранних
runs отличаются только последующей эволюцией других/затронутых tests. Exercised
legacy browser/harness/lifecycle/state-hash не менялись во время его полного run.
После успешных runs перед test freeze добавлены guards cleanup частично открытой
fixture/Chrome при startup failure; успешный путь не меняется. Reviewer115
повторит final exact source двумя bounded scopes data-only и poll-only.

## Передача независимому reviewer115

Статус QA: функциональное покрытие24checks выполнено на final runtime; **новый
E2E source ещё ожидает независимого approve115**, общая TASK/выпуск не закрыты.
Tests только в `tests/web/**` плюс эта WORK; все новые files<300строк.
Финальные screenshots/layout-measurements — `qa/final-layout-poll/`.
Design114 имеет собственную независимую actual fixture; временная моя fixture
уже закрыта и её screenshots не выдаются за его независимое воспроизведение.

Full makecheck/race, production SSH/HTTPS/upgrade, Linux runtime/second physical PC
**NOT-RUN tester**; root выполняет отдельный этап. `upload-process.cjs` SHA
`02c08138e587a87594abb896d43272c6367eb009f8230cef2c5337eedab8c99f`,
`lifecycle.py` SHA `bfcd6851d30939b7d74424c473597c766008df70e29c7d295a6e2e25ae09e363`,
`state-hash.py` SHA `9bb16317b4a91de6618bc6b720a0716dd8aac2c54b06b91dd494e2f73bd632b5`
совпадают с reviewed097 manifest. Отдельный120s/HTTP50MiB run не повторялся;
исторический PASS095/097 сохранён как отдельное доказательство, не текущий run.

- 2026-10-19T19:55:59Z — handed_off115: frozen tests/manifests и24actual checks переданы reviewer/root. Product/Go/infra/Git/SSH/Tandem/Давинчи tester не менял.

## Приоритетное уточнение финального test freeze: Clock race

- 2026-10-19T19:58:14Z — reviewer115 самостоятельно выполнил data-only5PASS, затем poll-only обнаружил flakiness теста: `pauseAt(new Date(await page.evaluate(Date.now)))` иногда получает время в прошлом из-за промежутка между CDP вызовами. Доказательства `review115/poll/` сохранены. Это дефект стенда, не product defect; его failed run не заменён PASS.
- Исправлены только4clock setup места: общий `pausedClock` задаёт неподвижный Node anchor, `install(anchor−1hour)` и `pauseAt(anchor)` **до navigation нового timed UI**. Для401/409/hidden после login создаётся свежая blank page. BrowserDate через RPC больше не читается. В409anchor остаётся60sпозади, GoTTL2s и actual401/409/terminal assertions не меняются.
- **PASS affected poll-only4/4** на final9efa: `qa/final-poll-clock-fix/control-results.json`. Legacy14, неизменённые data и layout повторно не запускались. Reviewer115 повторяет только affectedpoll на exact новом freeze.
- **Актуальный test source manifest:** `qa/test-source-clock-fixed.json`, SHA256 `3f9e0e2835fb68626328cd0425cc3cdfb0906483e35c4ce6a9e75dde97f55f8a`. Прежний f02manifest сохранён как предыдущий freeze.
- **Актуальная сводка24checks:** `qa/acceptance-evidence-clock-fixed.json`, SHA256 `f85892406334575cf418c9d88e81a94c505583fe3dab7a245381b6d3355bb217`. Она объединяет14legacy +5data +1layout +4актуальных polling checks и сохраняет ссылки на failed whole attempts/точные scopes. Executed code projection неизменённых data/layout/legacy не изменилась; новый helper там не вызывается. Не заявляется один монолитный24check run.
- Full make/production trustedHTTPS proofs сообщены root, лично tester их не запускал; ранее указанный NOT-RUN tester остаётся точным.

## Завершение QA

- 2026-10-19T19:59:43.847236+00:00 — completed: root передал итоговый **APPROVE115** final test freeze3f9e после независимых data-only5PASS и affected poll-only4PASS. Функциональное покрытие24checks принято в точных scopes выше; ранние whole/scope FAIL сохранены и за успех не засчитаны.
- Tests source после freeze3f9e не меняется. Legacy14 и data повторно tester не запускал. Review115 — отдельный от tester агент; его результаты не названы собственным tester run.
- Root ведёт guarded apply sourceac6 отдельно после111/112/114/115approve. Production SSH/HTTPS/apply/rollback и второй физический ПК **NOT-RUN tester**; собственные loopback tests не подменяют эти доказательства.
