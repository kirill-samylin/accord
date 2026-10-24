# TASK016 — новое поручение из экрана управления

24.10.2026. Пользователь повторно сообщил о неисполненном поручении создать
`c1.js` в корне Test_internet с `console.log(1)`. Run
`mr_93545b960e7c112be4e7a6e87c37cd22` completed: native процесс успешно
завершился, но отказался писать без task/lease. Это не повтор pin failure
[WORK278](work/WORK-278-test-internet-pin-repair.md) и не выполнение задания.

Пользователь явно выбрал: **новое поручение проходит весь цикл автоматически —
задача, разработка, независимые review и QA**. Используется существующий
[workflow TASK014](automation.md), его разрешения и ограничения сохраняются.

## Изменение поведения

- Основной вход из «Управления» отправляет новое поручение в существующую
  форму полного цикла с явным выбором действующей команды/версии разрешения.
  Одна отправка запускает дальнейший цикл; политика не выбирается и не
  продлевается скрыто. Черновик не выдаётся за запущенную работу.
- Прежний запуск отдельного агента остаётся ручным: для изменения файлов
  нужна задача и отдельно полученная самим агентом актуальная аренда.
  Без задачи допускается только явно выбранное поручение без изменений.
  Указание «только чтение» не является обещанием дополнительной OS-изоляции.
- Исходный текст taskless run можно перенести в форму полного цикла в памяти
  текущего project scope; без текста в URL/глобальном хранилище, автоподтверждения
  либо отправки при переключении проектов. Mutation uncertainty сохраняется.
- Completed — завершение шага процесса. Выполнение требования подтверждается
  задачами, фактическим SHA, review и QA, а не exit0 или анализом слов модели.

## Владельцы и проверки

Реализация/UI unit/integration — [WORK281](work/WORK-281-taskless-launch-author.md).
Независимые architecture/security/source review —
[WORK282](work/WORK-282-taskless-launch-review.md). Отдельная process/UI/native QA
после source approve — WORK283. Capable production release — WORK284.
Root ведёт этот checkpoint, точечную подготовку локального разрешения и операции.

Для точного `c1.js` прежний local source scope недостаточен. Reviewable draft
добавляет только `c1.js`, повышает host/workspace/check versions1→2 и
пересчитывает preset digest. Остальные пути, tools, права, роли, controllers,
сроки, бюджеты и действующая policy сохраняются; policy обновляется как новая
immutable revision, без дубликата разрешения. До независимого approve draft
не применяется. Токены/host proof остаются private runtime вне Git.

Сценарий приёмки: настоящее исходное поручение → analyst plan → один developer
commit с единственным новым `c1.js` → отдельные exact-SHA review/tester и реальные
project checks. Проверка вывода Node допускается отдельно на exact reviewed
файле установленным runtime и явно отличается от workflow preset/JUnit.
Координатор не создаёт файл вместо native разработчика. Старая история сохранена.

Private evidence: игнорируемая `.local/task017-taskless-launch/`.
Production поставка только capable all-project upgrade/roll-forward с полным
backup/историей/ACL, отдельным review и HTTPS QA; старый binary/DB rollback запрещён.
## Промежуточный checkpoint

Concrete draft независимо APPROVE в WORK282 и применён штатно с private backup,
idle/отсутствием child processes, bootout, atomic config, consent exit0 и bootstrap.
Automation/workspace/project-tests versions2; source roots добавлен только
`c1.js`. Existing policy `apolicy_ca6a1de12e24aefd97a96f42d85960fd` получила
revision2/hostConsent2, сохранив прежний expiry, budget, роли и participants.
Три прежних completed workflows не обновлялись и не запускались снова.

UI candidate `d80040cf546ff92320eccac0f84ca586a3b4b6f6` независимо SOURCE APPROVE,
clean Go/Next production build одного SHA: full104web tests/typecheck PASS;
bounded predeploy independent regression APPROVE. Native server/worker для
исходного поручения пока прежние4ae/dfcef, это отдельно от новой UI revision.
Workflow `wf_80c5b653534825ea69f11a5a2bbcd715` completed/version25, независимые
WORK283 и WORK282 native final audit — PASS/APPROVE. Один local commit
`489c123250237254608e8a16bdd492b60c8d6ee5`, parent
`bf77b6db0fadfc5a18da3045f0f4fd5bf510a314`: добавлен только `c1.js` с
`console.log(1)` и LF. Все восемь прежних tracked файлов byte-exact.
Четыре роли — отдельные настоящие Codex UUID/server sessions; reviewer approve,
tester passed относятся к current target/generation1. Developer и tester
выполнили два настоящих project-tests/JUnit, каждый8passed/0failures/errors/skipped.
Все terminal ACK flags true; pending/leases/reservations/active runs/owned
processes0. Без replan/fix/refresh/добавления времени или бюджета.
Отдельная независимая Node QA exact reviewed файла: stdout `1\n`, exit0;
это дополнительная проверка, не workflow preset. Push примера не выполнялся.

Production capable apply и postapply audit выполнены для d80040c: новый Go/Next
active, schema6/floors сохранены, validated pre-snapshot format2/4projects/
9runtimeDB/9historicalDB, canonical logical-row preservation PASS. Независимый
postapply audit принят. Actual HTTPS/Safari UI PASS: основной scoped переход,
ручная задача/read-only, process-only status, exact64char перенос старого текста,
default draft и явный policy2, без повторного submit. Served BUILD_ID и asset
hashes соответствуют d800. Исходный выполненный workflow виден в интерфейсе.

Canonical finalize1792853563261450000 и final-audit1792853621919609000 exit0,
journal complete. Новый validated format2 backup manifest
`93f29fd5bb33f7197d7b5f4174a7d45ed8ba43c81d0eaf5f887ac6c44004fb81`:
9runtimeDB/9historicalDB, inventory/history и logical rows сохранены,
owned Go/Next подтверждены. Ни один старый binary/DB не применялся.
Independent release final acceptance — PASS/APPROVE:
`review/final-approval.json` SHA
`49eb418e6256cfa7189dd331addd159ff0c427b794f75dbc75cf6d90cd3397d3`.
После финализации Test_internet online/epoch6/fresh lease; pins по-прежнему
совпадают, pending0 и активных runs0. Старые пользовательские runs сохранены.

Ограничения QA: browser readonly participant, error/uncertain paths и actual
submit нового диалога — NOT-RUN. Его source/tests и существующий submit путь
проверены; повторный workflow на уже созданный файл намеренно не создавался.
Native case относится к original4ae/dfcef, новая UI отдельно к d800.

Текущий статус: **CLOSED / PASS ограниченного Mac/production сценария**.
Новые Windows/Claude execution/второй ПК и полный V1 не заявляются.

Пользовательский результат: `/Users/shagi/Developer/Test_internet/c1.js`;
готовый цикл — `/app/p/p_bc866cc2ce2d052ab93e438c544560a5/automation/workflows/wf_80c5b653534825ea69f11a5a2bbcd715`.
Для следующих поручений: «Управление» → «Выполнить требование» → выбрать команду
→ «Начать выполнение». Для исходного `c1.js` повторная отправка не нужна.
Source и документация сохранены локальными commits; push/merge этого исправления
отдельно не выполнялись. Чужой `deploy-task012.command` сохранён.
