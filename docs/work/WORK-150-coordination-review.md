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
