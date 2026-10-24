# WORK283 — QA TASK016 полного цикла для c1.js

24.10.2026. Отдельный QA после independent SOURCE APPROVE WORK282 и approved
узкого consent/policy update координатором. Область записи: эта карточка и
private `.local/task017-taskless-launch/qa/`. Runtime config/policy не менять.

## План до submit

Проверить online/idle/pending0, exact policy2 и host automation/workspace/check2,
чистый Test_internet и отсутствие c1.js. Ровно один automation/submit со стабильным
request_id: создать корневой c1.js с console.log(1), один task и один local commit,
только c1.js; прежние src/tests/tools/README/AGENTS/.gitignore сохраняются byte exact.
Без dependencies/install/postоронней сети/push/merge/deploy. Никакой файл результата
QA не создаёт: это делает только настоящий native developer под task/lease.

Наблюдать все steps/native роли и checks в пределах existing deadlines без refresh
и hidden retry. Вопросы/timeout передавать координатору, не отвечать допущениями.
После completed: exact commit/parent/diff/blob, независимые reviewer/tester,
existing project-tests JUnit8passed как regression, terminal ACK/pending0,
нет leases/reservations/owned процессов. Отдельно разрешена проверка exact reviewed
c1.js существующим Node, stdout1/exit0; это отдельная QA, не workflow preset.
Underlying native workflow использует прежний approved production 4ae; новый UI
candidate будет проверен отдельно после deploy координатором.

## Статус

Native полный цикл — PASS. Predeploy candidate source/validation — APPROVE/PASS.
Новая production UI QA — PASS после отдельного postapply gate, см. ниже.

## Отдельная predeploy candidate QA — PASS в ограниченном объёме

Candidate `d80040cf546ff92320eccac0f84ca586a3b4b6f6`, clean isolated checkout.
Независимо прочитан diff UI и новых regression tests с применением
vercel-react-best-practices. Editing требует task; taskless explicit choice,
исключение retained task, запрет task/lease acquisition и byte quota prefix
проверены. RequirementDialog держит prompt локально, делает scoped cancellable
read, требует явный выбор exact policy version и не отправляет автоматически;
uncertain mutation остаётся frozen/locked существующим useAction. Проверены
project switch cleanup и expected project response guard. Completed copy — process-only.

Независимые targeted managed/automation/mutation tests: 23passed/0failed;
независимый TypeScript typecheck и diff check — PASS. Root build evidence
прочитан: 104tests, production Next/Linux Go build; hash root test log,
Next manifest и bundled Node независимо совпали. Go/infra/dependency source delta нет.
New tests прошли отдельное QA чтение; новых process suites QA не добавляла.
Private candidate-report/approval фиксируют APPROVE только source/validation
regression scope, runtime_go=false. Actual production browser и native runtime
нового candidate — NOT-RUN; postapply production UI gate обязателен до finalize.
Текущий c1 workflow использует underlying approved4ae и не выдаётся за проверку candidate.

## Реальный native полный цикл — PASS

Underlying production `4ae529272e1542e8473902de26d68349af266d1d`;
Mac worker epoch6, exact applied policy
`apolicy_ca6a1de12e24aefd97a96f42d85960fd` version2.
Один submit с request_id `task016_c1_full_cycle_20261024_v2_once` дал
`wf_80c5b653534825ea69f11a5a2bbcd715`, completed/version25 в 14:39:06 UTC.
Ни replan, ни refresh, ни повтор execution не потребовались; вопросов не было.

Preflight: online/lease live, active runs0/all прежние workflows completed,
pending0, current policy exact approved2 и current config exact approved draft,
публичный host automation2, clean checkout/c1.js отсутствовал.
Host scope в общем GET list по контракту truncated. Explicit scoped POST list
с exact workspace/check2 сохранён сразу ПОСЛЕ submit; pre-submit exact full server
scope не заявляется. Эта диагностическая корректировка не меняла server/config
и не повторяла submit. Первоначальный helper остановился до submit на KeyError
при обращении к truncated scopes; мутации не произошло.

| Роль | Задача | Native UUID |
| --- | --- | --- |
| Аналитик | T-20 | 01a16e51-e8f7-72f2-8eb2-ac82d14be43b |
| Разработчик | T-21 | 01a16e54-6eb3-7e20-aaed-55fdbeb2cff4 |
| Reviewer | T-22 | 01a16e56-56f0-71f2-811d-789fafc8fb79 |
| Tester | T-23 | 01a16e58-1353-7892-8151-711f8fd47554 |

Все четыре native steps completed; транскрипты фактических CLI сохранены private.
Создана одна задача разработки и отдельные служебные analyst/review/test задачи.
Результат — ровно один local commit
`489c123250237254608e8a16bdd492b60c8d6ee5`, parent
`bf77b6db0fadfc5a18da3045f0f4fd5bf510a314`. Diff строго `A c1.js`, blob и
рабочий файл byte exact `console.log(1)\n`. Все 8 прежних tracked files совпали
по SHA256; status clean. QA файл c1.js не создавала и не редактировала.

Actual coordination: T-22 approve и T-23 passed, обе проверки current/latest/
successful, exact targetSHA и generation1. Project-tests preset2 выполнен
developer и tester; реальные JUnit независимо распарсены — 8 tests/0fail/0error
в каждом. IDs `check_f280b0b8ae990d3f3eb79b34fd4b4593` и
`check_a2fe40cc676dece8e51dcf7ead543daa`. Это regression Python, не JS output proof.

После completed отдельная независимая QA запустила exact reviewed `c1.js`
уже установленным Node24.19.0: stdout `1\n`, stderr empty, exit0; HEAD/status/blob
после совпали. Это отдельная Node QA, не workflow preset и не новый native run.

У каждого из 4 attempts native_exited/exit_durable/managed_ack/exit_ack/
lock_released=true, pending0. Все task done без lease; workflow reserved_runs0,
managed active0. После финала прямых worker children0; дополнительный OS audit
всех periodic process snapshots подтвердил отсутствие прежних owned PID/PGID.
Private final report: `.local/task017-taskless-launch/qa/native-final-report.json`;
полные workflow/run/task/coordination snapshots, JUnit, transcripts и OS audit
расположены рядом, mode0600; каталог mode0700. Токены/proof не публиковались.

Native цикл выше сам по себе не подтверждает установку UI candidate;
новая production UI отдельно проверена ниже. Windows/Claude execution/
второй физический ПК этим кейсом не проверялись.

## Actual postapply HTTPS / Safari UI — PASS

Координатор добавил этот итог из immutable external QA receipt/report отдельного
QA после его сообщения о завершении. Active candidate
`d80040cf546ff92320eccac0f84ca586a3b4b6f6`, served BUILD_ID
`JhofWwPcTj3Bzr4jsJ4Hh`; изменённый UI chunk имеет exact manifest hash.
Обычный TLS, owner login/cookie/CSRF, catalog switch и четыре scoped страницы — PASS.

В настоящем Safari проверены основной переход в scoped полный цикл, обязательная
задача ручного режима и его validation banner, явный read-only выбор с честным
ограничением. Старый taskless run показывает завершение процесса, а не выполнение
задания. Кнопка переноса открывает форму с исходным текстом: exact64char AX/API
equality. Policy изначально draft; после явного выбора version2 доступна отправка.
Форма закрыта без отправки. Новый completed workflow отображает четыре этапа,
точный489c123 и две реальные проверки8passed. Собственная QA вкладка закрыта,
исходная вкладка восстановлена.

No-new-run/workflow IDs, pending/reservations/active0, тот же HEAD/clean checkout.
Readonly participant, browser error/uncertain paths и actual submit нового диалога
— NOT-RUN; соответствующие source/tests проверены, дублирующий intake не создавался.
Это отдельная новая UI приёмка, не retag native case на другую runtime revision.

Private `qa/https/approval.json`: kind independent-https-smoke,
reviewer qa_launch, PASS/APPROVE/runtime_go=false, created1792853438,
SHA51267242943bfcc1b6e271195668698890cf4ee38f5c9bbdbc27e2735d851950.
Immutable `qa/https/report.json` SHA
830c329969f7badef822eee891f8e5786bd5b183423bb8cb3ff003c07b8d760e.
Окончательная серверная поставка требует guarded finalize/final-audit и отдельной
независимой acceptance; QA их не выполняла.
