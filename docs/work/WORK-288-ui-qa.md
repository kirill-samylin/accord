# WORK288 — независимая проверка UI candidate TASK018

Исполнитель `validate_candidate`. Область записи: эта карточка,
`.local/task018-ui-release/qa/`, только форматирование
`web/src/lib/automation-evidence.ts` и порученное узкое исправление изоляции
черновиков. Общую TASK ведёт координатор.

Прочитаны AGENTS.md и `vercel-react-best-practices`. Исправлено только
форматирование прежнего `automation-evidence.ts`: prettier перенёс аргументы
вызова на несколько строк, семантика не менялась. Это устраняет фактический
FAIL общей проверки форматирования, сообщённый координатором.

Проверка exact clean candidate, build linkage и итоговые approvals пока
**NOT-RUN**: ожидаются independent source approve, committed SHA и actual build
артефакты. Новые process/E2E тесты и remote действия не выполнялись.

По поручению координатора исправлен найденный independent reviewer P2:
черновики пользователя A оставались после смены identity на B в той же вкладке.
`lib/session-drafts.ts` удаляет только `accord:create-project-name` и ключи
`accord:create-draft:*`/`accord:pending-task:*`, сохраняя чужие записи storage.
`lib/api.ts` вызывает cleanup при подтверждённых успешных login/logout,
unauthorized event и успешной загрузке login fallback. Неизвестный исход logout
не выдаётся за подтверждённый; login401 прежнюю форму не сбрасывает.

Добавлены пять unit regression проверок этих переходов, отказа storage и
сохранения посторонних ключей. **5 PASS**, новые process/E2E не добавлялись.
Исходники и новые тесты переданы `review_changes` для independent review.
