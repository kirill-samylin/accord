# WORK284 — подготовка узкой поставки TASK017

Исполнитель release_prepare. **WIP SOURCE PREPARED / runtime NOT-RUN**.
Область записи: эта карточка и новая ignored
`.local/task017-taskless-launch/ops/`. Общую TASK ведёт root; чужие изменения,
существующие ops268/receipts и production не изменяются.

Прочитаны canonical `infra/accord/projects/upgrade.py`, операционный README,
WORK268 и exact executed ops268 source/pins/manifest/final268 receipt. Исходный
manifest `e8acec650ce394f96627680921cdf7f8996b06e61709bdcf500295ca6b71bb7b`:
все12members совпали. Принятый baseline4ae — документированный input; нового
live readback автор не выполнял и не выдаёт его за проверку текущего VPS.

Подготовлены7 own copies reviewed/executed268 с bounded delta: baseline
`4ae529272e1542e8473902de26d68349af266d1d`, схема6, TASK017 GO label,
original final268 evidence и отдельные pending exact candidate source/build/QA
approvals. Исторические gates/sample261 не retag. Exact4projects/9runtimeDB/
9historicalDB, UID999 WAL-aware close, durable UNKNOWN, owned paths/process,
canonical capable all-project backup/history/ACL/floors и roll-forward сохранены.
Нового updater/framework нет. Existing268 origin hashes и полный diff сохранены.

`PLAN.md` задаёт минимальный путь: approved clean committed candidate → actual
Go+Next build одного SHA → independent build/QA → pinned archives/source freeze
и independent ops review → new inactive upload → preflight/stage/fresh preflight/
canonical apply → audit → independent external HTTPS QA → finalize → final-audit.
Даже Next-only исправление идёт как coherent Go+Next capable release. Схема6→6,
без DDL и новых historical copies. Busy/unknown fail closed; принудительная
остановка пользовательских работ для окна поставки не поручена.

Сейчас **candidate/build/QA pins отсутствуют**, `build_ready=false`; null поля и
отсутствующий executable `manifest.json` блокируют driver до транспорта. Archive
не создавался. Следующий freeze допустим только после actual input от root,
независимого source review и отдельного per-phase GO. Положительных квитанций
автор не создавал. Подробные prerequisites/командные templates и fail-closed
границы — private PLAN, без токенов/пользовательских данных в Git.

Фактические проверки: source268 exact hash readback; Python AST-only и local
reference hashing (результат ниже). Никаких SSH/HTTP/liveDB/import/build/archive/
upload/stage/apply/finalize/native действий. Это подготовка поставки, не PASS
production, не fullV1/Windows/Claude/второй физическийПК.

WIP validation: **7 Python AST PASS**, max205lines/script; все5 pinned local
reference hashes совпали. WIP manifest SHA256
`f4b81ca1e5c89dbd1317d93603744947533f1aaaad6f77c8c1d3021893d96af6`.
Это `source-wip-manifest.json`, не executable manifest/approval. Новый source
нуждается в независимом review после поступления actual candidate pins.

## Actual candidate inputs — approval freeze pending

Root передал committed clean candidate
`d80040cf546ff92320eccac0f84ca586a3b4b6f6`. Read-only сверка actual build.private,
Go/Next manifests и всех members выполнена; actual Linux Go SHA
`f46e5365231c29a3843003810701d53f9a25add7f6023fd9e57d931f98e26e95`,
Go manifest `e42f92c0d172473a3fdf8ab371169369a3d2c6e32ae948b1bed3f7348e42ccb4`,
Next manifest `9cb3472434620b810aef19622717d70a085659539df2e1574ec7583f3a140019`,
BUILD_ID `JhofWwPcTj3Bzr4jsJ4Hh`; fixed Node hash прежний. Buildinfo hash и
source independent-task017-source APPROVE с source manifest
`18ac77960e7bec49efe0d82658238174fb79719fb05273d41bcda58c7b9ea856` сверены.
Отчёт и source manifest остаются собственностью reviewer, автор их не менял.

По отдельному root поручению созданы локальные regular-only USTAR archives:
Go55payload+manifest и Next1256payload+manifest. Каждый исходный member/path и
каждый tar member readback/hash проверены. Go archive SHA
`7bf610b48e95550c144acfe9e433e84b5c8dd4b4f7be2be4d970bc49bd13a6d4`,
Next archive SHA
`093513f7a674f98360542e548305299c010cc6e3411eb47d769c72b180a8c085`.
Это локальная упаковка actual inputs, не новая build/runtime проверка.

PLAN уточнён: candidate QA перед поставкой ограничена фактической regression/
static проверкой; новая browser UI проверяется после apply независимым external
HTTPS gate. Прежняя source preparation этого browser PASS не заявляет.
Independent build/QA receipts ещё pending: **build_ready=false**, executable
manifest отсутствует. Remote actions по-прежнему NOT-RUN.

## Final source freeze — independent ops review pending

Получены original independent build APPROVE SHA
`1deec12be571f82b201f4e0fb30407115307bc15d22a3797ffa4b54b215dbc09`
и limited predeploy QA PASS/APPROVE reviewer `qa_launch`, SHA
`6bf61ad5fe067b8d2715d8625403b388c18f3687badb1ac244556512db0b645e`.
QA report scope сохранён: source/targeted tests/typecheck; production browser и
candidate native NOT-RUN. Проверены exact revision/runtime_go/verdict/status,
report hashes, build producer/source manifest и actual Go/Next/Node поля. Все16
local refs совпали; full regular-only archive member/hash readback повторён.
Original final268 остаётся independent evidence baseline4ae, без retagging.

Первый unreviewed freeze был сделан до поступившего HOLD на QA reviewer metadata;
сохранён как `manifest-pre-reviewer-field.json`, никогда не исполнялся. После
actual QA metadata update подготовлен final executable source manifest SHA
`a577b0b141187a3dc5262215b00422b9e986b69a2b72be5116978ef1e4080d52`.
12source/metadata members;7AST PASS, max208lines/script. Local-only вызовы
verify_source/delta_check PASS; main/transport не вызывались. `build_ready=true`
означает actual approved input готовность; independent ops approval и per-phase
GO ещё обязательны. Никаких SSH/HTTP/upload/stage/apply/native действий.

На момент source freeze c1 workflow ещё доходил до финального tester. Это был
root-reported статус, не local live readback автора. Последующее выполнение ниже
сделано координатором после actual terminal и нулевых all-project counters.

## Actual root execution

Координатор выполнил exact reviewed a577b0… цепочку для clean d80040c.
Upload-go1792852821705889000 и upload-next1792852857113168000 подтвердили NEW
inactive regular-only releases и отсутствие переключения active4ae.
Preflight1792852929169221000/stage1792852970313862000/fresh preflight
1792853014127197000/apply1792853048093068000/audit1792853107381514000 — exit0.
Все четыре board busy counters0, UID999 connections closed, registry/ACL/floors,
ingress/markers/helpers сохранены. Canonical capable updater сохранил logical
rows9DB/history и создал validated format2 pre-backup, без DDL/старого rollback.

Независимый postapply APPROVE92dda578… и actual HTTPS/Safari QA PASS51267242…,
дополнительный independent evidence review e9601fb8… приняты в их исходных
областях проверки. Browser gate относится к actual d800 UI; native c1 case
отдельно original4ae/dfcef, не retagged как новый native d800 запуск.

После нового нулевого all-project busy check canonical finalize
1792853563261450000 и final-audit1792853621919609000 — exit0/journal complete.
Новый validated format2 backup manifest
`93f29fd5bb33f7197d7b5f4174a7d45ed8ba43c81d0eaf5f887ac6c44004fb81`,
9runtimeDB/9historicalDB, inventory/history и logical rows сохранены; Go/Next
owned. Независимая final release acceptance — PASS/APPROVE49eb418e…:
immutable review/final-approval.json/report333baa5e… и37input manifest приняты.
Production release d800 закрыт, полный V1/новый native Windows/Claude не заявлены.
Драйверы/receipts/archives
после source freeze не менялись. Автор adapter новых runtime действий не выполнял.
