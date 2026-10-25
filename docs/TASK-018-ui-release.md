# TASK018 — публикация нового интерфейса и обновление стенда

Поручение пользователя 25.10.2026: отправить новые изменения в исходный
`https://github.com/kirill-samylin/accord.git` и обновить рабочий стенд.
Координатор — root. Source ветка `codex/native-preflight`; исходный remote
восстановлен как `origin`, прежний DenisShagi remote сохранён как `denis-origin`.
История и MIT LICENSE сохраняются, force push не используется.

В поставку входит уже закоммиченный redesign3062 и новые улучшения первого входа,
создания проекта/эпика/задачи, справки и восстановления черновиков. Отдельный
review выявил утечку черновика между последовательными пользователями вкладки;
исправление очищает только собственные draft keys при подтверждённой смене входа.
Go/infra/dependency delta от принятого d800 отсутствует. Форматирование одного
прежнего web файла исправлено без изменения поведения.

Старые untracked `deploy-task012.command` и `qa/first-users/start-local.command`
сохранены локально, исключены из новой публикации: первый непригоден schema6,
второй завершает процессы по портам без ownership. Они не исполнялись.
Исторический QA REPORT относится к отдельному локальному прогону и не является
проверкой production candidate.

Проверки до commit: TypeScript PASS,109 web tests PASS, prettier PASS,
`git diff --check` PASS. Source review — WORK286; QA/fix — WORK288;
capable all-project release — WORK287. Предварительный Next build PASS,
release build ещё должен быть выполнен из exact clean committed SHA.

Production baseline d800 подтверждается исходным accepted release evidence;
новый live readback, stage/apply/HTTPS/finalize пока NOT-RUN. Только canonical
`infra/accord/projects/upgrade.py`, coherent Go+Next same SHA, format2 backup всех
проектов, прежние ACL/floors/history/logical rows и roll-forward сохраняются.
Busy/unknown блокирует обновление; пользовательские работы не останавливаются
для получения окна. Native Windows/Claude execution/второй физическийПК и общий
fullV1 этим UI выпуском не проверяются.
