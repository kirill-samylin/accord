# WORK291 — понятный выбор файла токена

Исполнитель `validate_candidate`. Область: `LoginForm.tsx`, собственный scoped
`LoginForm.module.css`, эта карточка и ignored
`.local/task019-remove-legacy-ui/token-ui/`. Общую TASK ведёт root; ссылки и
server SSR меняются другими исполнителями.

Исходный пользовательский screenshot просмотрен: нативная кнопка и строка
«Файл не выбран» сливались в один плохо читаемый элемент. Прочитаны AGENTS.md и
`vercel-react-best-practices`. Добавлена обычная клавиатурно доступная кнопка
«Выбрать файл токена»; статус вынесен отдельно с `role=status`, имя файла
переносится внутри карточки. Есть краткая подсказка, замена и удаление выбранного
файла. После удаления фокус возвращается на кнопку выбора.

Исходный input остаётся `type=file`/`name=token_file` и участвует в прежнем
`FormData`; он скрыт, открывается кнопкой. Native required снят только с hidden
input: прежняя проверка отсутствующего/пустого файла до отправки сохранена,
чтобы браузер не пытался фокусировать скрытый input. Лимит4096, multipart login,
CSRF, exact immutable retry и общая блокировка fieldset сохраняются. Имя файла
очищается вместе с input после успеха/смены способа входа. Содержимое файла в JS
не читается, в UI выводится только стандартное имя, без сохранения в storage.
Dependencies не изменены. Scoped CSS исключает влияние на остальные формы.

Фактические проверки на Node24.19.0: typecheck **PASS**, прежний полный набор
**109 unit tests / 0 fail**, форматирование **PASS**, `git diff --check` **PASS**.
Лог — `.local/task019-remove-legacy-ui/token-ui/check.log`. Dependencies не
обновлялись; новые process/E2E тесты этим автором не добавлялись.

Независимый source review `review_changes` — **APPROVE**: проверены доступность
кнопки и status, перенос длинного имени, возврат фокуса, прежние file guards,
FormData/name и блокировка uncertain. Browser/runtime новой формы — **NOT-RUN**;
существующий production этим исполнителем не менялся. Для отдельного actual QA
рекомендованы выбор/замена/отмена picker, удаление, keyboard focus и empty submit.
