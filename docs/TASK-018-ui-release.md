# TASK018 — публикация нового интерфейса и обновление стенда

**CLOSED / PASS: server UI production release bdcabbd.** Полный V1 не заявлен.

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

## Фактическая публикация и применение

Clean candidate `bdcabbd9e97f3aa20ae32aeed1853529be1130dd` отправлен обычным
push в исходный origin, ветка `codex/native-preflight`; readback remote SHA совпал.
`main` не переписывалась. Actual Linux Go и production Next собраны из exact SHA;
Go buildinfo vcs.modified=false. Independent source/build/candidate QA и
operational source APPROVE получены на actual inputs, прежние native proofs
не переназначались новому выпуску.

Ops manifest `a3bd6d026cbdbbb62bc97730f3ea471406c5f2d499245d95578b0034511e0248`.
Upload-go1792931545209925000 и upload-next1792931557612622000 — PASS,
active_changed=false. Preflight1792931588046221000, stage1792931598973579000,
fresh preflight1792931618280539000, apply1792931635573161000,
audit1792931698166309000 — exit0. Live baseline d800 подтверждён; все4boards
busy counters0,9runtimeDB/9historicalDB, UID999 WAL-aware connections closed.
Новый journal pending-https-smoke; оба active процесса exact bdcabbd.

Validated prebackup format2 manifest
`dc884050e4ee8e8d36ee5074078e4d6ec97a32230d087e9f23c597287e9d0989`.
Canonical preservation: databases_preserved9/logical_rows_preserved=true,
board_schema6, historical hashes/ACL/guard/floors/nginx сохранены.
Independent postapply approval — `3aba4f2e43e4d905e5ab8738b3ad10c7acac432816fe31a951915eb14f2613fd`.
External HTTPS/browser и finalize/final-audit ожидаются; одно применение ещё
не закрывает поставку.

## External HTTPS и UI

Actual external QA exact bdcabbd — PASS/APPROVE validate_candidate;
original receipt `cd89503ee181694d3ce6bcd1a6ed01b20fd4b53a241a7bb55990f463614a2b29`,
report `e2bfd1c1263994fe6f0b1d686ef352c043448d49891d00db993d458ea7fa7a2e`.
Независимый evidence review — APPROVE
`0408dc1a43ae37863438659e49965243bbad940791dd5409b090c8667a847af9`.
Проверены12static assets против exact build manifest/BUILD_ID,4authenticated
scoped pages, unauth401/foreignOrigin403. Через собственную IAB вкладку:
задача/local validation, title+description draft restore, справка, project-name
restore и фактические logout/relogin с пустыми draft полями — PASS.
Задачи/проекты/workflows/runs не создавались; собственная вкладка закрыта.

Первый HTTPS тест остановился до credentials на штатном root302; ожидание
исправлено узко и независимо reviewedv2, прежний FAIL сохранён. Native Windows,
Claude execution, второй физическийПК, distinct-user B browser flow,
viewer-specific/back-specific/successful-create сценарии нового UI NOT-RUN.
Первоначальный uncertain browser login во время переключения не выдан за PASS;
точный прежний retry затем завершился успешно, новый UI загружен отдельно.

## Canonical finalize и final-audit

После fresh external QA и independent evidence approve выполнены canonical
finalize1792932124963460000 и final-audit1792932192189420000 — exit0.
Finalize заново проверил все all-project busy counters0 перед полным backup;
пользовательские работы для окна поставки не останавливали.
Journal complete; owned Go/Next exact bdcabbd, backup service success/status0.
Новый validated format2 backup manifest
`ed2caaf57bf114d8e6d977f0b20fa83944768239cee056ea5ba6406b174bf1bb`:
4projects/9runtimeDB/9historicalDB, inventory_preserved=true,
history_preserved=true, logical_rows_preserved=true, schema6.
Scheduled timer tick отдельно NOT-RUN. Локальный Mac worker/consent/settings и
native host pins эта серверная поставка не меняла.

Итоговая независимая приёмка review_changes — **PASS/APPROVE**, original receipt
`e1184a9e4a71367b9102ef2cb5d9b210dfe267c45619250906f1cb03ce208047`,
report `6a0dd437856efee26563e1a6e9342801a6f203645bdf7a9e98299bdff18e930f`,
75input evidence manifest сохранён вместе с private approval.
Frozen ops/source/build/QA reports сохранены без изменения. Source code и
release артефакты принадлежат bdcabbd; последующий documentation-only commit
фиксирует результаты без замены production binary. TASK018 закрыт как
публикация нового серверного UI, а не как новый fullV1/native acceptance.

После finalize выполнен отдельный HTTPS GET `/app`: HTTP200, проверка TLS включена.
