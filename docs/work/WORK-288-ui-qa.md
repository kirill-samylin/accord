# WORK288 — независимая проверка UI candidate TASK018

Исполнитель `validate_candidate`. Область записи: эта карточка,
`.local/task018-ui-release/qa/`, только форматирование
`web/src/lib/automation-evidence.ts` и порученное узкое исправление изоляции
черновиков. Общую TASK ведёт координатор.

Прочитаны AGENTS.md и `vercel-react-best-practices`. Исправлено только
форматирование прежнего `automation-evidence.ts`: prettier перенёс строковое
значение `checkSummary` на следующую строку, семантика не менялась. Это устраняет фактический
FAIL общей проверки форматирования, сообщённый координатором.

Начальный checkpoint до committed candidate: exact build linkage и runtime
были **NOT-RUN**. Их последующие фактические результаты приведены ниже.

По поручению координатора исправлен найденный independent reviewer P2:
черновики пользователя A оставались после смены identity на B в той же вкладке.
`lib/session-drafts.ts` удаляет только `accord:create-project-name` и ключи
`accord:create-draft:*`/`accord:pending-task:*`, сохраняя чужие записи storage.
`lib/api.ts` вызывает cleanup при подтверждённых успешных login/logout,
unauthorized event и успешной загрузке login fallback. Неизвестный исход logout
не выдаётся за подтверждённый; login401 прежнюю форму не сбрасывает.

Добавлены пять unit regression проверок этих переходов, отказа storage и
сохранения посторонних ключей. **5 PASS**, новые process/E2E не добавлялись.
Исходники и новые тесты независимо приняты `review_changes`: APPROVE/PASS;
пять новых regression tests reviewer также повторил самостоятельно.

## Exact candidate и артефакты — PASS

Проверен clean commit `bdcabbd9e97f3aa20ae32aeed1853529be1130dd` после
independent source APPROVE. Самостоятельно повторены typecheck, полный набор
**109 unit tests / 0 fail**, форматирование и `git diff --check`; рабочее дерево
было чистым. Лог — `.local/task018-ui-release/qa/candidate-check.log`.

Build review независимо сверил 202 source members, 55 Go members и 1256 Next
members, manifest/producer/source/log hashes, actual `go version -m` для Linux
amd64 с `vcs.modified=false`. Backend/infra/dependencies не изменились относительно
`d80040c`. Next настроен на Go7892; BUILD_ID `B1sXF5AaCW_jt_RCH8Orq`, Go SHA
`c3f544812bf47aba17192df66b802ae2afebe74a2aab7dfcec907fe41601739d`,
Next manifest SHA
`f12a22d93d4fcac3684d609d9eee3ae96868abb771962fffe282aeffb598f03b`.
Проверяющий не создавал эту сборку. Его авторство узкого auth cleanup и
форматирования явно отражено в QA report; их независимый source review выполнен
другим исполнителем. Это не утверждение независимого авторства собственных fixes.

Квитанции APPROVE/PASS относительно exact SHA, `runtime_go=false`:

- `.local/task018-ui-release/review/build-approval.json`, SHA256
  `396a6ec1bfa8ed708a6b266d6537eda3066aeac45ffb7043e7237533b9a9fbea`.
- `.local/task018-ui-release/qa/candidate-approval.json`, SHA256
  `e24b9f1ab36ed50ac486f09e244d09109907f6ed21b9684597fce4b73ea99b75`.

## Actual postapply HTTPS и браузер — PASS

После root APPLIED/AUDIT signal проверен внешний production HTTPS с обычным
owner login и catalog switch. Все четыре project pages относятся к ожидаемому
Test_internet; 12 served assets, включая обязательный BUILD_ID manifest, побайтно
совпали с actual candidate. Неавторизованный scoped tasks GET получил 401;
synthetic login POST с чужим Origin — 403. Проверка TLS оставалась включённой,
redirects автоматически не следовались, секреты не выводились.

Первый HTTPS attempt остановился fail-closed до входа: `/` имеет согласованный
nginx 302 на exact same-origin `/app`. Initial failure log сохранён. Narrow v2
проверяет точный 302 Location и делает отдельный GET `/app`200. Исправленный source
независимо принят до повторного исполнения: SHA256
`807bfc9f0e6708c54311de1abd468647de37c558f51e20f77bd0fa413726cc3f`,
receipt `review/https-test-source-v2-approval.json` под TASK018, SHA256
`5d41587499c7439d5d0842998539ed5dde7155020bc2d47cefe28b568cd44b34`.
Предыдущие source/receipt/failure не переписывались.

В отдельной фоновой IAB вкладке actual новый интерфейс проверен через UI:
overview/tasks, переключение проекта, обязательные поля и русская локальная
валидация новой задачи, точное восстановление title/description после закрытия
и открытия, справка, валидация пустого названия проекта и восстановление названия.
Создание задачи/проекта не отправлялось. После подтверждённого logout новая форма
входа имеет актуальные подсказки; повторный вход **тем же owner identity** показал
пустые черновики проекта и задачи. Это actual same-identity logout/relogin proof;
distinct identity B в браузере **NOT-RUN**, cross-identity invariant дополнительно
проверен unit regression и независимым source review.

Initial file login попал в uncertain во время окна поставки. Выполнен явный retry
того же original запроса; после успеха новая сборка перезагружена. Это не выдаётся
за безошибочный initial login. Own IAB tab закрыта, пользовательские вкладки не
изменялись. Screenshot/DOM и ignored evidence сохранены неизменными.

Новых tasks/projects/workflows/managed runs, policy/config changes — **0**.
Browser back отдельным действием, successful project/epic creation и readonly
participant browser — **NOT-RUN**; закрытие/повторное открытие и unauth guards
проверены в указанной области. Новый native execution, Windows, Claude execution
и второй физический ПК — **NOT-RUN**. `runtime_go=false` означает отсутствие
прямого native Go execution этим проверяющим; HTTPS использовал работающий backend.

Fresh `independent-https-smoke` PASS/APPROVE для finalize:
`.local/task018-ui-release/qa/https/approval.json`, SHA256
`cd89503ee181694d3ce6bcd1a6ed01b20fd4b53a241a7bb55990f463614a2b29`.
Связанный actual report `qa/https/report.json` под TASK018, SHA256
`e2bfd1c1263994fe6f0b1d686ef352c043448d49891d00db993d458ea7fa7a2e`.
Root уведомлён, browser work завершена до finalize; сам проверяющий deploy,
finalize и SSH не выполнял. Эта карточка не объявляет общий full V1 PASS.
