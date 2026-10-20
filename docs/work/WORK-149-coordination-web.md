# WORK-149 — Next UI координации

20.10.2026. Отдельный frontend автор TASK011. Область записи: `web/src/**` и
этот WORK; Go/infra/канонические документы не изменяются. Требования —
`docs/coordination.md`; независимый архитектурный APPROVE — WORK147. Root
подтвердил ответ пользователя «Да, этот контур».

Статус: SOURCE READY. Авторские client67/67/typecheck/build/format PASS;
независимый source review и дальнейшая process/browser QA ведутся отдельно.
Process/E2E и production — отдельный tester после source APPROVE, не этот WORK.

## Реализация

План: task assignment pinned profile revision/общая очередь, read-only причины и
проверки, browser dependency/lifecycle/explicit review+QA forms. Существующие
project boundaries/useAction/durable same-action retry сохраняются. Human не
исполняет задачи developer/reviewer/tester. Provider preferences не worker.
Применён skill vercel-react-best-practices, без новых зависимостей.

## Реализовано

- `web/src/features/coordination/`: read-only панель task detail с раздельными
  routing/dependency/closure reasons, ссылками в исходный проект, pinned profile
  revision, текущей lease-session, зависимостями и историей точных SHA/generation
  проверок. QA доступна только по authoritative `check.successful`, не по одному
  `outcome=approve`; backend включает latest current checks в bounded200.
- Диалог assignment/role queue, dependencies add/unlink, cancel/restore/reopen
  (reason + defect evidence) и epic lifecycle, explicit review/QA create. Диалог
  фиксирует исходную Task.Version и profile revision; known409 сохраняет черновик,
  refresh требует явного rebase, uncertain outcome повторяет тот же frozen action.
- `CreateTaskDialog`: optional exact profile pair атомарно с task create; профиль
  выбирается из active metadata нужной роли. Edit/archive/role change не заменяют
  выбранную версию незаметно. Ответственный текст остаётся только у epic;
  task assignment показывается как профиль/общая очередь, отдельно от live lease.
- `types`, `api`, `format`, task screens/common и coordination CSS: scoped mutation
  allowlist новых четырёх действий, read-only flag suppression, русские outcomes/
  audit labels. Все links prefetch=false; новых dependencies/SDK нет.
- При несовпадении `page.task.version` и `coordination.task_version` (разные чтения
  при конкурентном изменении) controls отключаются и показано требование обновить
  страницу. Это не выдуманная атомарность page response.

Новые files меньше300строк. Существующий `screens/tasks.tsx` стал309 (был306):
добавлен один переиспользуемый assignment renderer; переработка всего прежнего
списка не входит в этот этап. `CreateTaskDialog`292, detail288, остальные новые
компоненты≤249. Конфигурация/lockfile/Go/infra/production не изменялись.

## Авторские проверки

На Node24.19.0 выполнены `make web-test web-typecheck web-format-check web-build`:
PASS, client67/67 (60 прежних +7 новых), types/Prettier/Next production build PASS.
Приватный локальный журнал `.local/task011-frontend-check.log`.
Новые regressions: exact pin/changed+archive+role mismatch, malformed pair;
known conflict не rebases draft; role queue clear не отправляет profile pair;
human не получает agent completion/foreign lease actions; QA rejects recursive
blocked/superseded/negative checks; viewer/expired/mixed-version read guards;
новые scoped actions сохраняют original body/action/version и uncertain retry;
check create передаёт target_version, не самостоятельно выбранный SHA.

Существующее Node предупреждение MODULE_TYPELESS_PACKAGE_JSON сохраняется;
package metadata/dependencies не менялись ради подавления предупреждения.
Go/source independent review, process/E2E, physical second PC и production:
NOT-RUN этим автором. Реализация передана на независимый source review.
