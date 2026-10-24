# WORK-261 — приёмка одного production Python workflow

Владелец QA: gui_connection_qa. Статус: SOURCE PLAN READY / native NOT-RUN этим QA.
Root исполняет отдельно; итоговые evidence принимает независимый reviewer202,
а не root-executor. Новый harness/tests/installer не создаётся.

## Зафиксированный scope

Один уже сохранённый Test_internet workflow из root live261, без нового submit.
Пользователь отдельно разрешил permanent Test trust и этот Python native запуск.
Это не QA219 исключение. Root применяет только reviewed WORK260 и затем настоящий
UI resume после свежей authority проверки; QA ничего не запускает и не управляет.

Private baseline `.local/task014/qa261/plan.private.json`, SHA
`fb66e31c785207df14bad6eb1751b6eb8c0bbc2b00fdc96e24ad9999e88aab15`:
точные workflow/policy/host IDs, digest/version project-tests и SHA трёх root
`live261/*-before.private.json`. Непосредственно прочитаны только эти scoped API
снимки: все status200, workflow awaiting_authorization/version1/epoch1,
steps/plans/questions/attestations/runs/invocations/reservation0. Это исторический
baseline; root обязан перепроверить lease/grant/consent перед действием.

Server a32 / worker be95 (Mac SHA9755ed0f...) различаются; runtime identity берётся
из actual WORK251 install/provenance, не из QA fixture. Policy version1:
1 plan task,2fix/task и всего2,12native, continuation1,300s/step,1800s/workflow.
Preset project-tests/version1/test/required/trusted_owner_process. Четыре обычных
Codex role profiles; реальный provider/model фиксируются по actual, не выдумываются.

## Минимальный evidence пакет root после actual цикла

1. Сохранить безопасные scoped report/task/run ответы ДО assertions; ошибочный
   HTTP/DTO тоже сохранить как status/code, без headers/bearer/cookies. Exact
   workflow ID берётся из baseline, не «последний» workflow. Supported read-only
   domain POST automation/report (limit100), managed/run и GET tasks/{key};
   это запросы чтения, не command/claim/release. Если нужен automation/list,
   limit<=50. Cursor completeness, уникальные IDs, отсутствие truncation и
   стабильная workflow version обязательны; не скрывать страницы истории.
2. Для каждой materialized analysis/developer/reviewer/tester стадии связать
   workflow/step/run/generation/task/task_generation/control_epoch/command/dispatch/
   native UUID/board session/profile_snapshot ID+version. После resume epoch НЕ
   обязательно1: использовать реальный grant и attempt binding. Новые reviewer
   и tester — отдельные native UUID и board sessions; developer contributors не
   становятся reviewer/tester. Все реальные fix/continuation generations остаются
   в истории; не подгонять число запусков к QA fixture4 и не превышать policy12.
3. Сверить plan/hash/materialized tasks/acceptance с исходным Python требованием.
   Current developer evidence содержит полный40hex Git commit. Сохранить точные
   baseline/final SHA и diff: ожидается обычный local commit реализации
   src/math_ops.py, без push/merge/reset и без замены fixed tests/runner.
   Source manifest/clean proof true; reviewer approve и tester passed относятся
   к ТОЧНОМУ current developer SHA/task_generation. При fix новые SHA требуют
   нового независимого review и QA; старые positive gates не переносятся.
   Developer permission_proof: digest и pinned executable SHA, actual обычный
   checkout/.git dev+inode совпадают с разрешённым Test root; QA identity/digest
   сюда не подходят. Trust260 receipt и профильная локальная consent отдельно.
4. Actual project-tests check от developer и tester: правильный preset/version/
   digest из принятого consent; полный tuple run/step/task/generation/session/SHA;
   typed check_exit_code===0 (отсутствие/null не успех), !check_aborted,
   source_clean===true, outcome_bytes_truncated===false, report_kind=junit,
   SHA256 настоящего report. Сохранить exact report.junit из соответствующего
   check attempt и сверить его SHA с attestation. Fixed2tests реально исполнены,
   failed/errors/skipped0, необходимые positive/negative/zero assertions сохранены.
   Не создавать новый JUnit «для доказательства» и не перезапускать tests из QA.
5. Для всех относящихся к workflow local run records проверить пять literal true:
   native_exited,exit_durable,managed_ack,exit_ack,lock_released; per-attempt pending0,
   scoped journal.pending0, exit_unreportable не true. Соответствующий public run
   completed, actual UUID совпадает; terminal command receipt/dispatch exact и
   start applied/no error/no forced для простого happy path. Receipt.phase сам
   по себе НЕ server ACK; Receipt.state transient, не требовать несуществующий
   persisted value. У Receipt нет numeric native exit_code — не выдумывать его.
6. Workflow completed; current required tasks+epic done; нет workflow leases,
   open questions, active managed runs, reservations; полный отчёт не truncated.
   Genuine UI показывает именно этот результат. Если возникает вопрос/пауза,
   сообщить root и ждать настоящего human action; QA не отвечает за человека.
   needs_recovery/unknown/error/недостающие tuples — FAIL или INCOMPLETE,
   не заменить слово на PASS из-за task.done, CLI0 или видимой надписи «готово».

## Узкие локальные источники и защита данных

Разрешённый собственный root:
`/Users/shagi/.config/accord-test-internet/managed`.
Из WORK251 и shallow metadata проверены state/state.json, state/worker.lock и
logs/worker.stdout.log, logs/worker.stderr.log. Config/token содержимое и raw
journal НЕ читались этим QA. У journal есть host_secret; нельзя копировать его
целиком, как это допустимо было только внутри синтетического QA fixture.

Будущий readback state.json — bounded<=8MiB/no-follow regular/owner0600, только
whitelist runIDs из report. Сохранять проекцию tuple/state/5flags/exit_observations/
checks/связанных receipt fields и scoped pending count. Не сериализовать целиком
journal/automation/check.request: там IPC nonce/capability/token paths/secrets.
Compacted=true допустим после5ACK; ID/generation/observed hashes сохранены, пустые
compacted prompt/instructions не означают отсутствие исходного profile snapshot.

Только известные связанные пути, без обхода всего HOME/FS и broad ps:
state/runs/<runID>/attempt-<generation>/checks/<checkID>/report.junit — bounded4MiB,
точные IDs из report/state; checkout `.artifacts/junit.xml` — лишь дополнительный
latest copy, не замена immutable attempt report. Attempt bridge.json/mcp.json/
execution.token/IPC metadata не читать и не копировать. Логи — только известные
worker stdout/stderr, bounded selected finite diagnostic fields; не полный provider
transcript/reasoning/environment. Git read-only evidence собирает root отдельно
на точном checkout; этот PLAN не разрешает author запуск Git/CLI.

## Bounds, приёмка и честные ограничения

После сообщения root о terminal состоянии — один readback пакет с конечной
pagination<=20pages/limit100 и общим<=120s. Если только ACK ещё доставляются,
допустим отдельный bounded passive scoped readback<=60s/20observations; отсутствие
ACK после него INCOMPLETE, без restart/retry models/контролирующих мутаций.
Работающий постоянный daemon не обязан завершаться; проверяется конец каждого
его native/check attempt, не остановка LaunchAgent ради отчёта.

Root сохраняет private0600 receipts/index с SHA; независимый202 получает именно
этот immutable пакет и выдаёт свой actual evidence verdict. До него native Test
цикл не объявляется принятым. Windows/Claude/физический второй ПК, отдельный MCP
client runtime и remote push этим одним Mac Codex workflow не подтверждаются.
Этим WORK: только план/источники/metadata и baseline scoped API; runtime0.

## Первый actual attempt — bounded manual terminal projection

Root genuine UI resume запустил существующий workflow/controlEpoch2, analyst T2.
Он создал настоящий human question T3 и остановился; root не отвечал кодом и не
делал workaround. Причина/tool availability исследуется отдельно WORK263.
Это не завершённый Python цикл и не PASS.

По отдельному root разрешению выполнен только NOFOLLOW/owner600/bounded16MiB
in-memory parse собственного state, scalar whitelist ровно actual run/step.
Raw journal/config/token/IPC/host_secret не извлекались и не сохранялись.
`.local/task014/qa261/first-attempt-terminal.private.json`, SHA
`43947b52b1ec771b9d6f78f80ec94a8623de0fd1cd8b6e4495dddb7556c413c0`:
exact run generation1 STOPPED; все5ACK literaltrue; scoped pending0/checks0,
receipts started+stopped; source fd/inode/hash/size stable до/после. Это доказывает
конец первого native attempt/checkout release, не отсутствие вопроса и не fullPASS.
Никаких control/restart/resume/model/Git/test/network операций этим QA.

## Минимальный collector SOURCE READY

Прежний approved public PLAN сохранён byteexact в private plan-approved261.md;
плановый review202 receipt83f92... не выдаётся за approval нового helper.
`.local/task014/qa261/collector-v1/source-pins.json`, SHA
`fbf38498c361fbc81c4e4eb2de83888d914c10425f6a5350297032a92bc0aabc`,4members.
Три stdlib файла142/84/63lines, AST PASS, import/helper execution NOT-RUN.

Он принимает только exact own workflow API snapshot и утверждённый plan hash;
bootstrap требует independent202 source receipt kind source-qa261-collector,
verdictAPPROVE/source_manifest_sha256 и фиксированный root-go receipt path ДО
загрузки sibling кода/чтения state. Новый input0600 — только qa261/collector-input
с timestamp по README. Никаких source receipts/GO автор не создаёт.

Только whitelist local tuples/5ACK/pending/observations/check report.junit и
finite API projection; no rawjournal/IPC/check.request/config/token readout.
state16MiB/API4MiB/JUnit4MiB, NOFOLLOW/stability proof, exact IDs/attempt paths;
XML запрещает DTD/entities. Нет network/subprocess/Git/provider/test/control.
Output EVIDENCE_CAPTURED_NOT_FINAL_VERDICT, не PASS; separate independent202
actual verdict по immutable полному пакету обязателен. Root отдельно даёт
read-only Git и terminal task/run/pagination факты. Helper до202 review не запускать.

## Первый receipt recheck и collector V2

Проверка сохранённого43947... подтвердила tuple/epoch2/nativeUUID/5ACK/pending0;
при этом stopped receipt имеет forced:true. Поэтому первая дополнительная
happy-path гипотеза !forced была отвергнута, а не объявлена пройденной.
Доказано STOPPED с положительным durable/ACK/lockrelease; graceful0/no-force
НЕ доказаны, полного workflowPASS нет. Recheck не новая native observation.
Это значение сохранено в first-attempt-review.private.json; source receipt не
переписан. При последующем sameUUID resume latest managedgeneration может смениться:
первый immutable proof надо сохранить как историю, не потерять его в latest state.

Новый collector V2 по узкому ROOT SOURCE REOPEN исправляет только полный batch202:
exit_unreportable strictbool, check.id/status boundedstring<=512 перед сериализацией;
JUnit strictUTF8 decode + запретNUL (включая BOMlessUTF16/32), затем DTD/entity guard.
V1 REQUESTCHANGES/sourcefbf... остаются неизменны. collect261/io261 byteexactV1.

`.local/task014/qa261/collector-v2/source-pins.json` SHA
`db30c4efd6bf3e80ae2525f01b1f5ae99e45a0b9cdb1f333c10091843f020998`,5members.
AST3 PASS; local74lines; no imports/helper/actualexecution. Current-step-only и
compacted-hash omission явно ограничены: полный report/предыдущие attempts/Git/
task/run facts root передаёт отдельно. Helper до SOURCE APPROVE202 не запускать;
дальнейшие native/control/upgrade остаются только root, не автору QA.

## Root actual replanning — revised acceptance, 24.10.2026

Прежние PLAN/source/receipts immutable; этот addendum не объявляет старый запуск
возобновлённым. Root сообщил: настоящий UI answer сохранён после истечения
первого step deadline, поэтому новый native resume не исполнялся/бюджет не
расходовался; затем scheduler queued/CONFLICT version storm. Отдельный WORK264
исследует expiry handling, новый product rollout этим QA не выполняется.
Непосредственно сохранённый workflow-1792821325226531000.private.json подтверждает
queued/version15/epoch4/invocations1/started1 и original analyst interrupted.

Далее root сообщил fresh native human pause200/version82/epoch5 и genuine UI
replan с неизменёнными policy12/global workflow deadline: новый analyst T5,
новый managed run; original analyst/task остались в истории interrupted/cancelled.
Этот более поздний running state пока root-reported, его итоговый saved API пакет
обязателен. Никакого same-UUID native resume proof здесь нет.

Приёмка actual результата: отдельно исторический interrupted analyst + новый
текущий полный analyst→developer→reviewer→tester цикл. При отсутствии дальнейших
fix/continuation ожидаются5 actual native turns/5distinct UUIDs/5steps, а не прежние
QA fixture4 и не «5turns/4UUID при resume». Это ожидаемый сценарий, не выдуманные
actual counters: финальные цифры и все попытки сверяются с полным report/receipt
пакетом; согласованные дополнительные fix не прячутся и policy12 не увеличивается.
Старые5ACK/forced:true/pending0 сохранены отдельным43947... proof. Historical
cancelled analysis не подменяет done нынешних обязательных задач и exactSHA review/QA.

Состав worker версий также не переписывается: первый attempt был be95/9755;
root установил reviewed WORK263 worker dfcef4b/f629... и сообщил тот же host/
consent1 online epoch4 перед replan. Полные artifact/provenance pins берутся из
actual root install receipt; не выдавать сокращённые префиксы за полные SHA.
Server/GUI не менялись этим QA. CollectorV2/source db30c4ef и independent SOURCE
APPROVE192dbf4c остаются замороженными; helper НЕ выполнялся, ждёт отдельный root
30sec GO после terminal. Текущая работа — только план/сохранённые scoped snapshots.


## Независимая terminal evidence QA, 24.10.2026

Результат: **PASS текущего завершённого analyst→developer→reviewer→tester
цикла с явно сохранённой исторической interrupted попыткой**. Это независимая
проверка фактических root snapshots/approved collector capture, не новый запуск.
Отдельный итоговый evidence verdict202 ещё обязателен.

Сверены completed/version107/epoch6 до исходного deadline; 5 native invocations,
5 различных Codex UUID, 4 различных current board authors, profiles id/version
совпадают с grant. Current четыре роли done, новый epic done; старые epic/task
cancelled, human question answered. Reservations0, scoped pending0, task leases
отсутствуют, все пять native exit/durable/managedACK/exitACK/lockrelease flags true.
Четыре текущих public start команды applied без last_error/forced; все command
страницы полные (`commands_truncated:false`), tuple/dispatch совпадают с receipts.

Developer commit `fc5cc89925c071fac17907c9676d6e5a7ef02451`, parent
`6ea8676ad93d2c274a41a7d8765114af71a2cafb`. Собственные read-only Git HEAD/parent,
porcelain и diff-tree подтвердили один commit/clean/единственный src/math_ops.py.
Пять protected файлов byte-exact; фактический inode .git совпал с developer
permission proof. Review approve и tester passed относятся к тому же полному SHA,
T-6 generation1; reviewer/tester авторы отличаются от developer.

Оба настоящих сохранённых attempt JUnit прочитаны независимо: по2 testcase,
failures/errors/skipped0; XML SHA совпал с local check и public attestation.
project-tests/version1/digest совпадают с grant; typed exit0, literal false
truncation, source_clean, manifest/SHA/target/generation/epoch6 проверены.
Developer JUnit `117a6f34…`, tester `420e81e0…`; это две реальные проверки,
тесты повторно этим QA не выполнялись.

Историческая первая попытка НЕ переименована в happy PASS: interrupted/stopped,
forced:true, исходный start delivery_unknown и public last_error сохранены;
отдельный stop applied и все5ACK положительны. Ответ был после step deadline,
потом новый replan/analyst UUID; same-UUID resume и graceful/no-force здесь
**не доказаны**. Первый worker be95/9755, дальнейшие роли dfcef4b/f629;
server остаётся a32. Старые source/FAIL/43947… receipts не изменены.

Private review `.local/task014/qa261/independent-terminal-review/review.json`
SHA `afade37d1ed5efbdbe16fe2c0e9b1eb69398056b4bc9cbded7bd23c589fe3a14`;
41-file evidence index SHA
`65cf0735f66f8abdb52a73d751775ea777eeebc58b3d84b92f1a9702e8db2474`.
Collector root выполнил отдельно один раз; QA его не повторял, raw state/secret/IPC
не читал и не копировал. Final UI completed — root observation, QA новый browser
не открывал. Windows/Claude/физический второй ПК/remote push NOT-RUN.
