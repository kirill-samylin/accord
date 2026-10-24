# WORK281 — новый ввод через полный цикл, ручной запуск по задаче

Статус: implementation candidate; independent final review и browser QA ожидаются.
Дата: 24.10.2026. База: `d2af9c882683da7067056351f72218b03b141659`,
ветка `codex/native-preflight`. Карточка автора; общие TASK не редактировались.

## Причина и согласованный объём

Ручная форма позволяла отправить изменение файлов без задачи. Процесс мог
корректно закончить turn, отказавшись писать без lease, а пользователь ожидал
готовый файл. Запрет записи без аренды сохраняется. После выбора пользователем
полного цикла и независимого architecture approve WORK282 новый ввод направлен
в существующий TASK014 workflow с analyst/developer/reviewer/tester.

Прочитаны AGENTS, requirements, architecture, managed-agents, automation;
для React/Next применён vercel-react-best-practices. Новых API, зависимостей,
схем, policy или разрешений нет. Policy и локальное consent остаются явными.

## Изменения

- `web/src/features/managed/managed-screen.tsx`: основной переход «Выполнить
  требование» в текущем проекте; отдельная вторичная кнопка ручного запуска.
- `web/src/features/managed/start-dialog.tsx`: ручной запуск для изменений
  требует существующий task key. Taskless диагностика возможна лишь после
  отдельного checkbox «только чтение…». Ссылка на задачу по-прежнему не lease.
- `web/src/lib/managed-agents.ts`: UI validation обязательной задачи;
  read-only instruction добавляется к исходному тексту и учитывается в квоте
  16 KiB. Она также запрещает запрос и захват задач/аренд Accord. При выборе
  диагностики task key не отправляется. Это инструкция
  агенту, не OS sandbox; UI прямо сообщает ограничение. Backend optional task
  contract, ACL, profile snapshots, lease и workspace exclusion сохранены.
  `completed` обозначен как «Шаг процесса завершён», без вывода о результате.
- `web/src/features/managed/managed-detail.tsx` и новый
  `web/src/features/managed/requirement-dialog.tsx`: для taskless работы
  исходное поручение переносится в форму existing workflow только в памяти
  открытого диалога. Fresh project-scoped read получает capabilities/policy;
  проверка project identity и AbortSignal остаются в existing readPage.
  Project QueryBoundary и key scope размонтируют форму при смене проекта.
  Текст не помещается в URL, local/sessionStorage или query cache. Policy не
  выбирается автоматически; запускается только явной отправкой пользователя.
- `web/src/features/automation/requirement-form.tsx`: optional initialText и
  onLockChange позволяют reuse существующей формы без нового механизма submit.
  Sending/uncertain удерживает global project lock и запрещает закрыть диалог.
- `web/src/lib/managed-agents.test.ts`: regression для обязательной задачи,
  explicit read-only, restrictive prefix, сохранения исходного текста, omit
  task key, byte quota и сохранённых control/workspace guards.

Все затронутые production source файлы меньше 300 строк. Посторонние untracked
карточки WORK278–280 и deploy-task012.command сохранены.

## Проверки автора

- PASS — `npm run typecheck`.
- PASS — Node 24.19.0, targeted managed-agents, managed-actions,
  managed-observation, automation, mutation actionState/contracts: 35 tests.
- PASS — targeted Prettier, `git diff --check`.
- NOT-RUN — browser/process/E2E QA: по правилам новый process test выполняет
  отдельный тестировщик после независимого source approve.
- NOT-RUN — production deploy, worker install/config, native launch; автор не
  менял runtime и не создавал workflow. Новый полный цикл пользовательского
  поручения — отдельная работа координатора.

Первый вызов тестов системным Node 20.20.0 не поддержал strip-types; повтор с
уже установленным Node 24.19.0 прошёл. Зависимости и global environment не менялись.
