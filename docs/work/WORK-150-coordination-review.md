# WORK-150 — независимое review координации

20.10.2026. TASK011, reviewer отдельно от backend148/frontend149/root operations.
Область записи: только этот WORK. Product source, тесты, runtime и production
не изменяю. Исходный checkpoint `343b93c71297c58c972f73fed4d1d4ca3a349740`.

## Контрактный разбор до реализации

Прочитаны AGENTS, coordination, TASK011/010, WORK147, requirements/architecture
и реальные store mutation/tasks/dependencies/gates/sessions. Принятый ручной
контур не требует второго scheduler: assignment влияет на новый claim, прежние
leases/fencing/author provenance и SHA/generation gates остаются authoritative.
Pinned revision и current-profile guard не меняют уже выданную аренду; schema4
нужна даже для JSON-only поля, иначе schema3 writer потеряет assignment.

Atomic `check_create` с expected target version и server-captured SHA/generation
закрывает race между browser create tester и link reviewer. Tester должен
проверять `successful(latest reviewer)`, а не только outcome; новый pending
review должен инвалидировать старый approved edge существующим evaluator.
MCP12 и REST/CLI-only mutations совместимы с действующим transport контрактом.

Обязательных архитектурных блокеров по текущему тексту не выявлено. Root переданы
два уточнения для проверяемой реализации:

1. Различать актуальность target provenance и положительный результат проверки.
   `changes_requested` на текущем SHA/generation не должен отображаться как
   исторически устаревший лишь из-за отрицательного outcome. Значения `current`,
   `latest`, `outcome` должны иметь однозначные wire/UI определения.
2. История назначения должна сохранять прежний/новый profile ID+version либо
   явную общую очередь роли и причину. После reassign одного текущего
   Task.assignment недостаточно для объяснимого audit.

Это contract critique, не source APPROVE. Реализация ещё выполняется авторами.

## План независимой проверки

Проверить полный source diff и author tests: current auth/ACL/epoch до replay;
назначение и optional Create pair; atomic next/session-specific queue; pin edits/
archive vs existing lease; expected version и no partial mutation; check_create
и latest/recursive gates; bounded explain и shared predicates; browser issued
capabilities/recovery/strict form parsing; frontend drafts/unknown response retry/
project scope/viewer; additive migration4/preflight/full backup/updater guards.

После source APPROVE отдельный tester пишет process/E2E; новые тесты также
проходят review. Фактическая production evidence проверяется отдельно после
safe update. Пока unit/process/browser/Linux/VPS review: NOT-RUN.

## Operations slice — независимая проверка

Проверен root diff: guard coordination1/schema4 и прежние registry/ACL floor;
layout/validate полные2/3/4 с обязательной ровно одной metadata(1,1); strict
история schema1/2/3; upgrade_storage проверяет migrated4 и прежние logical rows/
immutable bytes. Startup hold/install guard до DDL, owned runtime и roll-forward
пути не ослаблены. Новые7 infra unit покрывают actual SQLite snapshot/history/
mixed3+4 и offline byte copy; auth reset в последнем тесте явно mocked.

Самостоятельно повторён полный infra suite: **PASS155/155** (1.452s), прежние
ResourceWarning незакрытых SQLite fixture connections остаются диагностикой.
Текущий operations source slice: **APPROVE**. Общий product source verdict
ожидает backend/frontend. Process/browser/Linux/production: NOT-RUN этим review.

## Next source slice — независимая проверка

Просмотрены весь tracked web diff и новые coordination panel/checks/dialog/fields/
profile-choice/assignment-label, типы/helpers, CSS и7 author client regressions.
Новых dependencies нет. Task assignment отделён от session lease; selected
profile revision и expected task version фиксируются в draft; refresh не
переписывает их без явного rebase. Unknown result сохраняет исходный payload,
action и project path через прежний useAction; viewer/expired session отключают
mutation controls. Ссылки scoped через ProjectLink, prefetch=false. Browser human
не получает agent completion/release. QA availability требует server successful,
не одного outcome/current. Историческое и отрицательное состояние различаются.

Самостоятельно: новые coordination client tests **PASS7/7**, TypeScript
`--noEmit` **PASS**, diff whitespace **PASS**. Node сообщает прежний
MODULE_TYPELESS_PACKAGE_JSON warning; dependencies/package не менялись.
Next source/author tests slice: **APPROVE**. Actual browser/keyboard/mobile/
two-tabs/late-response QA впереди после общего source APPROVE.

## Backend/source review и исправления до APPROVE

Проверены core DTO, assignment/CheckCreate/Coordination, общий task constructor,
Next/session queue, event enrichment, human administrative gates, migrations/
existing-storage preflight, project actorCall wrappers, REST/CLI/MCP/bridge,
browser actions/projection/recovery, новые author store/project/transport/browser
unit/integration и адаптация прежних schema expectations. Scope managed execution
не добавлен. Транзакции сохраняют auth до replay; независимость и SHA/generation
gates не заменяются profile ID. Новый floor4 не ослабляет registry/ACL.

До окончательного source verdict выявлены и переданы авторам:

- Explain должен ограничивать накопление при обходе DAG, а не только ответ после
  полного рекурсивного обхода. Исправлены visited и unique-reason201 budget;
  отдельная regression покрывает duplicate reasons через два blocking link types.
- Nonzero Transition.Generation не должен после release/expiry переключаться
  на административные права. Добавлен owned precheck, regression подтверждает
  LEASE_LOST для stale cancel и разрешённый явный generation0 cancel.
- Новые native assign/checks POST должны отклонять query-параметры, включая
  malformed. Узкий transport guard и negative HTTP regression добавлены.
- P2: capped ClosureReasons выбирались по random map iteration до сортировки.
  Добавлены deterministic top201 и forward/reverse300 regression. Проверяется
  также dedup двух role-specific messages для одного target, чтобы одно required
  reviewer/tester условие не скрывалось случайным порядком обхода.

Самостоятельный narrow Go повтор assignment/check/coordination/admin tests:
**PASS** store, projects, transport, web. Окончательный повтор после последнего
determinism fix и общий source verdict фиксируются ниже; tester ещё не запускался.

## Итоговый source verdict

**SOURCE APPROVE** после исправления перечисленных замечаний. Required-role
reasons теперь различаются по Code+TaskKey+Message и сортируются с полным tie
break; обе reviewer/tester причины сохраняются. Forward/reverse300 и same-target
different-role regressions проверены. Metadata4 exact singleton также проверяется
Store и published preflight, повреждённая таблица не восстанавливается молча.

Проверенный source/test slice:74изменённых tracked/untracked файлов под cmd,
internal, infra, web/src, tests/infra, tests/acceptance. SHA256 manifest строк
`relative_path + space + sha256(content)`, sorted paths, final newline:
`5cd77400d904fe72783b3beabda63b1fa3a4137ddfc914e651a3347548cc672a`.
WORK/canonical docs в этом content digest не включены; они не product source.

Самостоятельный финальный narrow Go повтор assignment/check/coordination/admin
regressions: **PASS** store/projects/transport/web (0.725/1.120/0.675/0.944s).
Включены concurrency2connections, race create/reopen, negative-current vs success,
assignment audit/replay/live lease, schema3→4/damaged4, ACL/epoch/archive,
strict REST/MCP12/resource и committed browser lost-response recovery. Bridge
SID injection regression также просмотрена; общий author Go/race/vet PASS
повторён автором полностью; после последних reader-only corrections автор
дополнительно подтвердил store unit/race, CLI routing unit и targeted vet PASS.
Full root check остаётся отдельным интеграционным барьером.

Реализация и author tests пригодны для отдельного process/browser tester.
Это **не** QA или production closure: process/E2E тесты ещё должны быть написаны
отдельным tester, review тестов и фактическая local/Linux/production evidence
обязательны. Физический второй ПК/provider execution/full V1 NOT-RUN.

## Review диагностического cache incident

Root сообщил о10известных pyc, созданных первым remote inventory без -B. Проверены
private initial cache-audit, remediation summary и detailed archive proof. Сам
сопоставил exact10before/archived path sets, SHA256/mtime/rootUID/0644/regular и
root0700 archive; независимо вычислил SHA исходников через `git show cbf2020`
для всех10файлов — совпали с installed_source/completed-helper digests.
Detailed proof фиксирует per-file CPython magic, timestamp header и compiled
code equivalence, отсутствие canonical caches и повтор helpers/runtime PASS.

Evidence APPROVE этой узкой remediation: known bytes архивированы, неизвестные
artifacts не удалялись. Detailed proof SHA256
`f649652a1e6c490d7e2c20d80b6baf7c1f80346040dfbde50a7ededc275a1fac`.
Это проверка сохранённых root-run данных; reviewer сам production не изменял.
Production update/HTTPS/финальный backup остаются отдельными барьерами.

## Независимый review tester152 и local QA

**TEST SOURCE + LOCAL QA APPROVE** для описанного ниже23-check snapshot.
Root дополнительно запросил narrow keyboard/focus test нового coordination
dialog; новый test hash/final freeze и его full repeat ещё ожидаются. Это не
отмена проверенного product source APPROVE и не production closure.
Прочитаны все18новых code/README files в
process011/process011-update/browser011, включая production-readonly script.
Assertions проверяют настоящий Go/SQLite/CLI/MCP/Chrome и независимые SQL rows,
потерянный после реального commit browser response, exact retry/audit,409draft/
explicit rebase, dependency/lifecycle, review→QA/SHA/generation, ACL/epochs,
две вкладки и реальный задержанный A-response после B. Тесты не ослабляют gates.
По reviewer замечаниям MCP assigned create теперь действительно MCP call,
release проверяется; restore4 проверяет все board tokens/browser sessions reset.

Последний полный local run1792528339624: **PASS23/23**,16screens,
source_unchanged=true, Go candidatef2edac9/SHA
`daf342c2bb7d714af7d4aceea6910de31eb9e6ddc0751b926ac667284e0bf8ac`.
NextBuildID `p8Oy9g77thoUV1qX6mfsx` сверён с действительным local build.
Report SHA `8b1bc9bdbc96418ffc9e3ac73c5d06e3227730fdea7ddad7b8a723948d828dbb`.
Сам перепроверил все recorded source hashes против текущих файлов и отсутствие
product diff кf2edac9. Репрезентативные assignment/blocker/checks/epic screenshots
1440/390/light/dark просмотрены: layout/тексты/состояния читаемы. Chrome report:
284responses, page/static errors0 и prefetch0; local CSP console не измерен этим
collector, отдельный production-readonly script проверяет CSP явно.

Tester updater **PASS8/8** повторён мной независимо в новой fixture:
`.local/task011-update-qa/run-1792528248434193000/results.json`, source_unchanged,
SHA `aee43664f9b2abe113881260e7a33151262f448b28abc295ebd5ea6c5eaabce1`.
Реальны Go/SQLite3→4/7DB/backup/crash/resume; systemd/Next/UID и prior-helper
scaffold simulated. Actual oldguard3→newguard4 replacement/runtime/VPS не
выводится из этого PASS, нужен отдельный actual stage/apply/HTTPS.

Final18test/docs JSON manifest SHA
`c46a1abb24aa6302c3a89da32e192907bb62dca763eb2bdfeb353284b92e34df`
сам сопоставлен с текущими bytes. Production readonly011 reviewed/syntax PASS:
SHA `5cb2b9b29a8ca6ac2bae87b6a45ff8b71497f7208dfb2a25de5c5206b0828395`.
Разрешены только собственные login/open/logout; native/MCP reads, conditional
existing coordination detail, N/A для пустой production коллекции и digest
existing task rows до/после. Reviewer его на production не запускал.

История FAIL сохранена. Ошибки URL/label locator/ожидания navigation исправлены
только в тестах. Label association независимо проверена actual Chrome:
combobox accessible name «Действие» корректен; getByLabel exact был ошибкой
Playwright locator. Run1792528150784 показал настоящий handled catalog-read error;
HTTP diagnostics того запуска нет, причина **не установлена** и не объявляется
test-only. После diagnostics два последовательных полных PASS8244009/8339624
включили тот же late-A/switch-B сценарий без пропуска. Это историческое
необъяснённое transient-наблюдение; если повторится, требуется product diagnosis.

Local/test bar закрыт; final delivery verdict пока впереди: immutable Linux
release/product/updater, actual production stage/apply/fullbackup/runtime и HTTPS
evidence обязательны. Physical second PC/provider execution/full V1 NOT-RUN.

## Итоговый QA freeze после keyboard дополнения

**FINAL TEST SOURCE + LOCAL QA APPROVE**. Узкий новый keyboard test просмотрен:
настоящие Enter/autofocus/полный Tab+ShiftTab trap/Escape exact trigger return,
business POST0 и неизменные Task.Version/assignment. Дополнительный native
macOS headless picker Arrow selection не входит в PASS: оба FAIL сохранены,
минимальный Chrome reproduction без React/Accord подтверждён reviewer; root
принял этому дополнительному subcase **N/A**. Functional profile selection
проверена настоящими browser forms. Полный accessibility audit NOT-RUN.

Финальный actual process+Chrome run1792528643872: **PASS24/24**,16screens,
source_unchanged=true, тот же clean productf2edac9/Go SHA daf342… и local Next
BuildID p8Oy9g77thoUV1qX6mfsx. Report SHA256
`c29a27320c1c6b49c066aa3a4682cd26b3df4bda8a07110fbe5bbd0b3e46387a`.
Все recorded code hashes и новый19file JSON manifest самостоятельно сверены
с текущими bytes; product diff кf2edac9 отсутствует. Final19manifest SHA256
`983d757a7668080929652e167f985a9a874b4e15da7fefed25a7ff033ee4bdae`.

Production readonly script остаётся прежним reviewed SHA5cb2b9…; независимый
updater8 repeat выше остаётся действителен, повтор не требуется. Можно сохранить
QA checkpoint и готовить immutable final Linux release. Доставка и итоговый
production evidence verdict этим APPROVE не подменяются.

## Immutable Linux release и actual stage review

**EXACT LINUX + STAGE EVIDENCE APPROVE** перед применением. Проверен clean HEAD
`e53c18c9643b9ee086141d0f227ab485f4f88cac`; product diff к approved f2edac9
отсутствует. Сам пересчитал все53 Go manifest entries,1256 Next manifest entries,
Linux ELF/Node/BUILD_ID/package-lock hashes. Go SHA256
`a03f64d5ff008b7f63023117fb7b1c15c33e5617ce7ac08ffc6440610315269d`,
NextBuildID `HHViYVPwDuQx7uj4Uk1YC`, Next manifest SHA256
`cb97c3dd40df2aa8340dd0e3ca2573af4d44a3f956c8498734efadb978b212c9`.
Transfer proofs53/1256 фиксируют active_changed=false.

Root actual Linux product13/13 и updater8/8 PASS: оба exit0/source_unchanged;
сам сверил463 fixture hashes каждого отчёта,421 product source/test hashes и43
updater/operator hashes с текущими файлами. Wrapper source проверяет exact new
ELF hash и old cbf2020 hash до запуска, единственный raw report, успешный exit;
product wrapper дополнительно проверяет свободные fixture ports до/после.
Report raw SHA256: product
`2791bfa98c0f5259ee439e9ca738d16d5da59a745cf37add68df19d1461dde8d`,
updater `e4de5c238968de8f9c941e4ea33ceb7b8ea70f0138957c0b0ea3abd907c6f5c7`.
Это проверка сохранённых root-run доказательств; reviewer remote не запускал.
Linux updater systemd/Next/UID остаются simulated, физический второй ПК NOT-RUN.

Actual hardened stage выполнялся source upgrade.py: bundle/ELF/capability/runtime
checks, Linux Node/sharp от service UID, owned stage listener и nonce login,
finally stop/remove temporary unit. Stage runtime proof повторяет check(before),
helpers/runtime PASS, phase staged, оба active pointers cbf2020, markers0,7894free,
owned PID264891, NRestarts0/MemoryPeak46825472. Inactive-only stage барьер закрыт.

Можно продолжить authorized capable apply со full pre-update snapshot под
startup hold. **FINAL DELIVERY REVIEW PENDING**: actual migration/preservation,
новые active owned runtime, HTTPS readonly smoke, final all-project backup и
complete journal должны быть представлены отдельно. Schema/data rollback или
production business writes этим verdict не разрешаются.

## FINAL independent delivery review

**FINAL APPROVE — TASK011 manual coordination delivery** для неизменённого
release `e53c18c9643b9ee086141d0f227ab485f4f88cac`. Ранее source, tester source,
local24/24, independent updater8/8, exact Linux13/13+8/8 и actual stage APPROVE
сохраняются. Новых обязательных findings нет; product/tests diff к release пуст.
Reviewer сам remote/production не изменял; этот verdict основан на независимом
чтении root-run доказательств и локальной проверке их bindings.

Apply/finalize outputs и journal complete согласованы с release/runtime. До DDL
reviewed updater под startup hold создал полный validated schema3/web2 snapshot
`/var/backups/accord-web/projects-20261020T204410Z-98c7caa68b0e4312a5db97aaa09301cc`,
manifest SHA256 `a4e06648b561c5667071fc9ca5d64a201326951a1a9e69b845a13fe6dd56d013`.
Preservation proof:3DB прежних таблиц logical_rows_preserved=true, boards schema4.
Сам сверил old installed guard SHA с git cbf2020 и new installed guard SHA с
current source: floor действительно заменён. Final collector повторяет
check(after), installed helpers/runtime/capability checks и service-UID guard.

Final Go SHA a03f64…269d и NextBuildID HHViYVPwDuQx7uj4Uk1YC совпали с exact
release; оба pointers новые. Owned GoPID265527/process bytes и NextPID265268,
capabilities coordination1/board_schema[4], services/timers active, restart0,
markers0,7894free. Authority marker regular/UID999/mode0600, SHA равен pre-snapshot.
Full final format2 backup validated на VPS, default schema4/web2, registry и все
наличные projects учтены: current inventory default-only, project_count0,12files.
Путь `/var/backups/accord-web/projects-20261020T204623Z-d90daa53b0614bfab1cdcbf041a9abca`,
manifest SHA256 `28b9886fd12e53d0559091d94b7771346a86b016e500ac46b1d96c31fc600a39`.
Реальные БД reviewer не скачивал; validator проверяет file inventory/hashes,
SQLite integrity/schema, registry identity, historical migrations и blobs.

Approved readonly HTTPS script прежнего SHA5cb2b9…8395 выполнился без изменения
script/support: **PASS7/7**,56screens,858responses200, page/console/asseterrors0,
prefetch0, собственный logout и browser_closed. Existing2tasks coordination
native/MCP/tab действительно прочитаны; task digest не изменён. Production
business mutations NOT-RUN по readonly дизайну, mutations проверены synthetic
local/Linux. Сам сверил report SHA256
`06aa3b53658adf869192d83f6a0c19dd06c92bfa3233cd8e2154d39fc59603a2`,
все56 screenshot hashes, script/support hashes и revision/runtime bindings;
coordination detail1440dark/390light дополнительно просмотрены, overflow нет.

Ограничения сохранены: физический второй ПК и provider/native Claude-Codex/worker
execution/full V1 NOT-RUN;390 — viewport одного Chrome. Полный accessibility
audit NOT-RUN, дополнительный native picker popup N/A. Прежний единичный handled
catalog-read transient остаётся с неизвестной причиной и сохранённой FAIL-history;
последующие полные PASS и HTTPS не доказывают его исправление. При повторении
нужна диагностика продукта. Managed launch остаётся следующей отдельной задачей.
Main/remote/push/merge не выполнялись и этим review не поручаются.
