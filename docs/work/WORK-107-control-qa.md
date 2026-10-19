# WORK-107 — независимая process/browser QA control UI

- Родитель TASK-005 / REQ-005; роль tester, исполнитель `/root/control_qa107`.
- Статус: план подготовлен; **ожидание актуального code/security APPROVE и frozen QA binary от координатора**. Новые process/browser E2E до сигнала не пишутся и не запускаются. QA PASS не выдан.
- Область записи после GO: только `tests/web/**`, эта WORK и private `/private/tmp/accord-control-099/qa/`. Продукт, Go, infra, Git, SSH, live, Tandem и Давинчи не изменяются.
- Прочитаны tester profile, общий процесс/parallel-work/testing Давинчи read-only, самостоятельный AGENTS Accord, control-web-contract §8/9 и поздние уточнения, WORK100/103, прежние QA095/review097 и existing browser/process harness. Tandem OFF по AGENTS Accord. Именованные Claude bindings недоступны; модель наследована, отдельность tester от авторов сохраняется.

## План приёмки

Все fixture данные synthetic. Реальные Go server/native stdio MCP worker/Chrome, literal loopback, private state/token files. Подменять HTTP/domain assertions успешными результатами нельзя.

| Группа | Действия и наблюдаемый результат |
| --- | --- |
| Существующая регрессия | Сохранить 14 browser journeys и domain assertions; адаптировать только login/nav/selectors/routes новой UI семантики. Полный browser core один раз после approve. Не повторять неизменённый upload-process 120s и HTTP 50MiB, уже PASS095/097; сверить source SHA. |
| Overview/search | Создать больше100 задач через native; получить точный total независимо от страницы. Найти поздний hit по title/description/key, mixed-case кириллицу, буквальные HTML/XSS; DOM безопасен. |
| Sessions | Две native сессии одного agent owner/run distinction и human browser session. Проверить kind/role/owner/state/paging/detail, текущее владение после claim/release. Отозвать synthetic token при будущем TTL: effective count/status/task badge меняются, UI не называет session OS процессом. |
| Tasks/activity | List/board показывают одни отфильтрованные задачи, paging сохраняет layout/q/filters; смена q/filter начинает новый запрос. Activity strict DESC, task/session/type filters, новый insert не повторяет старую границу. |
| Stats/projects/settings | Native completion без tokens и с tokens даёт реальную missing coverage; browser период/group. Один реальный read-only project, effective TTL/rules/settings, отсутствующие process/model/revision данные честно неизвестны. |
| Freshness | Только overview GET poll: visible/online; max1 in-flight, stale retains last good counters/time; offline pause/reconnect; 401/409 terminal stop; GET не renews lease/session и не пишет draft/form values. Failures остаются FAIL, transport probes отмечаются отдельно. |
| Responsive/accessibility | 1440 desktop и390 mobile, нет document overflow; drawer keyboard/Tab/Escape/focus return, theme persists, self-hosted font loads; скриншоты и private connection metadata для независимого design reviewer. |

## Окружение и доказательства

2026-10-19T19:31:04Z: проверены существование bundled Node/Playwright, Chrome, локального Go toolchain и old C2 binary; Node `v24.19.0`. Server/browser не запускались. Новый binary ещё не получен.

Результаты и source/runtime SHA manifests будут в private QA_DIR; токены/HTTP auth/cookies не копируются в отчёты. Фактический runtime scope macOS/loopback, не production HTTPS, не второй физический ПК и не полный developer delivery cycle. Full makecheck/race выполняет root отдельно; здесь NOT-RUN.

## Журнал

- 2026-10-19T19:31:04Z — planning: read-only contract/harness/environment inspection; GO dependency остаётся открытой.
