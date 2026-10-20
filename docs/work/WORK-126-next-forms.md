# WORK-126 — формы действий Next.js

20.10.2026. Исполнитель отдельного среза TASK-007. Область записи:
`web/src/features/mutations/**` и этот WORK. Общую TASK ведёт координатор.

Прочитаны AGENTS, требования, архитектура, control-web architecture,
WORK-122/123 и реальные Go login/mutations/actions/pages/design upload contracts.
Использован vercel-react-best-practices; операции выполняются в event handlers,
без effect-мутаций, без токенов в URL, query cache или browser storage.

## Реализовано

- `LoginForm.tsx`: ровно один источник токена (поле либо файл до4096байт),
  preauth CSRF, multipart, очистка input/File после подтверждённого входа.
- `CreateTaskDialog.tsx`: настоящий create task/epic, название/описание,
  приоритет, эпик, роль, ответственный, области; review/test target и commit.
  Поля draft фильтруются; прежний action_id не подменяет свежую capability.
- `EditTaskDialog.tsx`: по server `can_update`, только title/description и
  fenced generation; никаких неподдерживаемых изменений роли/статуса/приоритета.
- `QuestionActions.tsx`: явный claim, answer, release, ручной heartbeat,
  countdown без фоновых записей, generation, правильные task-bound action IDs.
  При утрате владения answer остаётся доступным для копирования; явное
  обновление читает состояние, не уничтожая текст. Новый claim не автоматический.
- `DesignUpload.tsx`: метаданные/привязка/версия, единственный file последним
  после csrf/action_id и метаданных; клиентский предел50МиБ плюс серверная
  проверка. Сохраняет выбранный File и метаданные после ошибки/неизвестного исхода.
- `SessionControls.tsx`: настоящий logout и отдельное явное продолжение сессии.
- `useAction.ts`, `actionState.ts`, `contracts.ts`, `ActionNotice.tsx`:
  единая обработка мутаций. Первый payload снимком фиксирует ID, поколение,
  текст и File. Неизвестный сетевой/серверный исход блокирует редактирование
  и другие действия этой формы; явный retry использует ровно прежний payload.
  Подтверждённая validation принимает новые ID, сохраняя пользовательский draft.
  Lease/auth loss блокирует новые записи. Нет optimistic domain success.
  Успешный redirect блокирует старую форму до новых server capabilities.

Диалоги используют общие Radix primitives frontend125: focus trap, Escape,
возврат focus и обработка overlay принадлежат общей Dialog реализации.
Styling классов форм выполняет frontend125. API helpers/types также его область.
Root интегрирует формы в десять экранов. Экспорт каждого компонента именованный,
из отдельного одноимённого файла; props согласованы с125 и root.

Все продуктовые файлы этого среза меньше300строк. Зависимости не добавлялись.
Реальные данные и токены для тестов не использовались.

## Проверки

- **PASS**:15 unit/integration tests `contracts.test.ts`/`actionState.test.ts`/`loginState.test.ts`, bundled
  Node24.19.0. Покрыты task-bound capabilities, неизменный retry generation/text,
  multipart ordering + original File, защита свежих csrf/action_id от draft,
  отбрасывание новых capabilities при unknown outcome, новые ID после validation,
  server ownership + blocked writes после lease loss, heartbeat ID/lease update,
  settled redirect.
- **PASS**: общий TypeScript typecheck после интеграции root-клиента,20.10.2026.
- **NOT-RUN**: независимый review форм, browser/process/E2E и реальные
  boundary upload проверки. По AGENTS их выполняют после approve отдельные роли.
- **NOT-RUN**: deployment, remote, push, PR, merge, проверка двух физических ПК.

Проверенная версия: текущий незакоммиченный TASK-007 worktree поверх исходного
HEAD, точный общий SHA фиксирует координатор после завершения интеграции.
Этот WORK не объявляет TASK-007 или V1 завершёнными.


## Исправления после независимого review127

- P1 login: общий stoppedBy сохраняется для доменных действий; только LoginForm
  использует login-specific policy. Подтверждённый неправильный токен401
  освобождает отклонённый payload и оставляет форму editable с действующим
  preauth csrf. Подтверждённый403 очищает непригодный csrf и блокирует повтор,
  предлагая отдельную кнопку обновления формы. Новый bootstrap читает только
  `/web/login` без token/file; native input/File остаётся смонтированным.
  API helper125 больше не публикует global auth-loss event для login401;
  остальные401 сохраняют прежнее очищение персонального cache и переход на вход.
- Create: `createDraft.ts`, вызываемый фактическим CreateTaskDialog, принимает
  только create metadata, строго task|epic, известную роль и допустимый priority.
  `type=claimed` из фильтра Activity не попадает в POST create.
- DesignUpload/QuestionActions получили `onLockChange`, вызываемый синхронно
  перед POST. При unknown он остаётся true; при подтверждённом ответе false.
 125 блокирует закрытие upload Dialog через X/Escape/overlay, а TaskPeek делает
  read-only. Full detail сохраняет composer смонтированным при смене tabs.
- Форматирование всех файлов этого среза pinned Prettier3.9.10; максимальный
  файл247строк.15/15 unit/integration PASS, общий typecheck PASS после fixes.
  Новые проверки используют mocked fetch для реальных readPage/submitAction,
  проверяют отсутствие token в bootstrap body и отсутствие global auth event
  при wrong login. Новые process/browser/E2E самостоятельно не запускались.

Source freeze126 передан координатору/reviewer127; source review и browser QA
остаются отдельными этапами, а не частью авторского PASS.

## Исправление upload после повторного review127

`uploadDraft.ts` используется настоящим DesignUpload: пустой/неизвестный scope
из list-query больше не перекрывает project. Нормализатор возвращает только
пять разрешённых metadata полей; project очищает scope_key/epic_key, epic
не несёт лишний epic_key. Query csrf/action_id/cursor/owner не переносятся.
При ссылке новой версии существующий материал из page.designs даёт scope,
scope_key и название, если они не переданы явно в редактируемом draft.
Привязку версий по-прежнему окончательно проверяет Go; серверные проверки
не ослаблялись. File и порядок streaming multipart не изменялись.

**PASS:**19/19 unit/integration после добавления четырёх upload regressions:
пустой/неизвестный scope; task/epic version default binding; сохранение явно
исправляемых полей failed draft; исключение query capabilities и текущие
защитные ID/file-last в multipart. Общий TypeScript PASS; новые три файла
отформатированы Prettier3.9.10, лимит300строк соблюдён. Source freeze126
повторно передан root/reviewer127. Browser/process проверки не запускал.
