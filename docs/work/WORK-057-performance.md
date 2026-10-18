# WORK-057 — performance runtime

Статус: на проверке; REQUEST CHANGES по R063-01/02, не завершена. Исполнитель `/root/perfdev057`, разработчик; именованные
Claude модели в среде не закреплены. Parent: WORK053 contract, TASK003.
Изолированная копия `.local/performance-003`, ветка `codex/performance`,
вход HEAD `30abe1674e972ea28272dacdb18392070840e0ed`.
Tandem OFF по прямому поручению; commit/push/live запрещены.

## Область и последовательность

Единственный migration4 owner. Запись: internal/store, необходимые core/CLI/
bootstrap/infra schema4 compatibility, author unit/integration,
docs/performance-runtime.md и эта WORK. Общая TASK, main, D056, E/UI/OAuth,
limiter и чужие worktree не входят. Технический contract053 принят root.

F1: immediate retained writer + context gate, data_version immutable committed
snapshot, bounded config parsing, detached DTOs. F2: entity COW/dirty journal,
incremental prepared persistence, affected graph/indexes. F3: schema4 bounded
events, persisted exact statistics, streaming atomic migration. Затем author
regressions/race/vet/build и diagnostic profiling; independent review/security,
process QA и target VPS acceptance назначает root отдельно.

## Текущее состояние

F1/F2 WIP: добавлены transaction_snapshot/snapshot_copy, writer DSN `_txlock`
проверен в pinned modernc1.60.1 sqlite.go395–400. Переносятся все admin/expiry/
mutation write barriers. Первый targeted compile/test запущен, результат ещё
не подтверждён. Migration/schema остаётся3 до F3. Никакой performance PASS нет.

## Журнал

- started: 2026-10-18, `/root/perfdev057`; точное время следующего события берётся
  из системного date. Вход frozen root contracts + D052/APPROVE054.

- 2026-10-18T19:50:40+03:00 — F1/F2/F3 реализованы, source ещё не frozen.
  Dedicated immediate writer, committed cache lifecycle, COW/dirty persistence,
  affected graph/verification/queue/title/area indexes; schema4 exact prefixes,
  bounded events, streaming migration и compatibility reducer. Все cmd/internal
  PASS до последней узкой дельты; targeted differential/migration/gates PASS.
  QA056-01 existing process overlay PASS20 repeats; SQL storage-full rollback
  PASS. Source060 co-author пишет только три новые *_performance_extra_test.go;
  его результаты и manifest будут включены после freeze. Независимого approve нет.
- 2026-10-18T19:50:40+03:00 — первый synthetic annual diagnostic PASS38.36s:
  migration29.108s, schema4 Open4.445ms, warmget0.441ms, stats20owners2.749ms,
  обычные мутации0.751–2.334ms. Это не p95/runtime acceptance. Seed memory
  исправлена на streaming для финального diagnostic; первоначальный RSS не
  засчитывается. Root принял migration readiness cap300s, override1..900s,
  обычный schema4 restart cap15s и fail-fast systemd; resource limits прежние.
  Backup retention/byte budget — отдельная DevOps область root, не WORK057.

## Финальный freeze и открытые замечания

По указанию root после независимых findings063 здесь исправлен только узкий
index-plan defect. R063-01 (Create done-epic direct shared write, success/reject)
и R063-02 (local HandoffEpic nested alias) **не исправлялись**: следующий свежий
fixer получит единый пакет review. Source НЕ готов к approve/приёмке. Подробности
и собственные reviewer FAIL probes — [review063](../review-063-performance.md).
Прежние author PASS не покрывали эти две ветки и findings не снимают.

Отдельная D scope-replay correction: исходный mutate возвращал durable replay
до roleAuthorized. Теперь SQL current role scopes проверяются и для replay;
Describe берёт role из exact input, другие методы — из persisted session.
Живая lease для replay не требуется, renewal/audit не добавляются.060 regression
покрывает expired Describe+Create exact replay до удаления scope и отказ обоих
после удаления. Это требование053, не новое право или protocol.

QA056-01: два одновременных CLI apply раньше иногда возвращали SQLITE_BUSY_RECOVERY
261 вместо CAS conflict. DSN busy_timeout применяется на каждом connection;
typed SQLITE_BUSY startup/Begin admission retry bounded и context-aware.
Бизнесоперации/Commit не повторяются. Existing frozen056 process testcase
через private read-only overlay PASS20 repeats, включая durable CAS poststate
и zero partial writes при audit/invalid-candidate fault. Новых process tests
автор не создавал. Зачётный повтор после approve принадлежит tester.

Events final index delta: SQLite выбирал type-only для task+type, затем читал
JSON всех matching types. Explicit INDEXED BY закрепляет schema4 index;
query-plan regression ожидает exact index во всех4combination. Без этой одной
строки private counterfactual **EXPECTED FAIL**, с ней unit/race PASS.
MAX cursor VDBE probe подтвердил оптимизированный Last/SeekLE; MAX не менялся.

### Фактические проверки

- Полный cmd/internal: PASS163 top-level,7пакетов; opt-in annual1 SKIP в обычном
  наборе, отдельно actual annual PASS. Лог unit-integration.log.
- Targeted store+bridge race: PASS30 top-level (annualSKIP), включает060,
  claim/CAS/expiry/role/global/epic и uncertain commit. Этот broad run был до
  последней index-only delta; для неё отдельно final events-race PASS.
- Vet cmd/internal, native Go build, diff whitespace: PASS.
- Python infra unit: PASS44, включая5новых migration readiness/guards.
- Schema4 migration DDL/cancel/storageFULL, raw/replay bytes/private backup,
  randomized differential150ranges/3groups, exact int64 prefixes/overflow,
  zero/legacy/nanoseconds/ties/negative wall/backward-time fallback: PASS.
- Real SQLite Commit fault: durable commit+syntheticerror и rollback+sameerror
  PASS; generation discarded, retry uses durable replay без дубликатов.
- Reviewer063 probes: **FAIL3 /2blockers**, независимые evidence владельца063.
- Full combined makecheck, independent security и final QA: NOT-RUN автором.
  VPS50ms/RSS/restart, TLS, physical2PC, реальные providers — NOT-RUN.049FAIL
  остаётся. UI/design-review — N/A, UI не менялся.

Final annual GOMAXPROCS1/darwin-arm64, native test binary: streaming200k/2000tasks,
prebackup+backfill+validation29.270s, schema4 Open4.248ms; cold/warm get21.889ms/
0.390ms, events20 0.487ms, stats20owners2.487ms, ordinarymutations0.811–2.215ms.
Process peakRSS79,904,768bytes/76.20MiB включая syntheticseed/migration/API,
DB173051904bytes. Это diagnostic single calls, не HTTP/p95 target acceptance.
Raw fixture hash4ef88b11725e7a53c3a8b522666568afd7b4c43e52b57d46be14534cdc3e096b.

Полный author file list (включая все новые untracked), exact hashes/commands/
binary/log hashes: `.local/work057/author-files.txt`, `source-manifest.json`,
`results.json`. Reviewer063 docs/WORK/evidence исключены из author commit list.
Все Go≤254строки, исключений>300 нет. Dependency/limiter/service resource limits
не менялись. Backup retention — отдельная062. Git commit/push/live не выполнялись.

- 2026-10-18T20:05:25+03:00 — handed_off: code source aggregate `42d32f7826481346d8fc16fede29bb04c674708560b36edbe8e517c61272c2d6`, status на проверке / REQUEST CHANGES; адресат root для fresh fixer после полного review/security.
