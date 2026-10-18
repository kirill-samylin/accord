# WORK-072 — independent combined D/schema4 process QA

Роль: tester `/root/perfqa072`; исполнитель независим от автора source и review069.
Статус: на проверке. Результат роли: PASS локального process scope; independent test review pending. Начало: 2026-10-18T17:30:13.663251+00:00. Вход: clean c1ef9ae2e8176064e1e652a9b1d480bf7b038943,
approved performance source96f3bfd/review069, D tests056/review061,
load method055v2/review061, backup source062/review065. Backup method068
REQUEST CHANGES070 исправляется отдельным073; backup_process*.py reserved.

Прочитаны AGENTS Accord, общий процесс/parallel-work и tester profile Давинчи,
performance053/runtime/config-contract, acceptance056/reviews061/069,
load055v2 и backup-retention. Tandem OFF explicit user; Davinci/live/SSH/Git
mutations запрещены. Named model bindings недоступны; inherited same model.
Общая TASK — root only. Scope: tests/acceptance, targeted tests/infra,
этот WORK, docs/acceptance-072-performance.md и P3 EOF WORK056.
Production/cmd/internal/infra/schema/templates/Makefile/dependencies readonly.
Исходные111 hashes совпали с root manifest; расширенный snapshot включает
все tracked cmd/internal/infra/schema/templates + build/dependencies.

## План и критерии

- schema4 fixtures: явное ожидание4, сохранённые exact legacy bytes/scopes/TTL/
  reset/auth/fencing, native C2→D4 и native D3→D4/restart/old refusal/rollback.
- Strict QA056-01: две CLI same expected, один winner/CONFIG_CONFLICT,
  exact version/hash/history/audit. Исторический056 FAIL не переписывается.
- Current SQL scope replay после expiry/config restriction; process эпик
  regression069; isolated annual load method с готовым binary и paced waves055v2.
- Targeted → freeze → один combined make check; FAIL останавливает full gate.
  Ordinary children отдельно от race harness. Linux/VPS/physical2PC/providers
  UI/OAuth/TLS NOT-RUN. После final manifest запись остановить.

## Журнал

- 2026-10-18T17:30:13.663251+00:00 started: source freeze подтверждён; schema3 fixtures адаптируются к4;
  P3-061 deterministic SQLite close сохраняет transaction boundaries.

## Уточнение root и factual gates

Root приоритизировал реальный доступ второго ПК к текущему stable C2/live.
Annual method/200k не входят в принимаемую072 delta: четыре уже созданных
черновых Python модуля перемещены в ignored `.local/work072/annual-draft/`.
Они НЕ приняты и требуют следующей отдельной WORK/review. Уже начатый tiny
2 sessions/2 tasks/64 events дал FAIL на неверном draft route update404;
raw JSON/log сохранены. Повторов и большого annual запуска не было.
Этот FAIL относится к черновику harness, не к product route contract.
Annual target050ms/RSS128MiB/start1s на1CPU/512MiB остаётся OPEN,
старый049 FAIL не переписывается;072 не разрешает D4 deploy.

Targeted исходный process run: PASS7/FAIL2 top-level. Два красных кейса —
ошибка нового fixture trigger: config get намеренно existing read, не migration.
После смены на offline apply same-hash обнаружен нормативный no-op audit,
поэтому raw events строго отличались одной новой строкой; этот test FAIL
сохранён. Исправленный fixture применяет stale expected0: миграция происходит,
CAS затем отвергается без audit/changes, как approved legacy1 guard.
Следующий красный test был неверным exact stderr ожиданием C2 для D3:
C2 generic `cannot open board database`; native D3 source возвращает
`database schema is newer than this server`. Исправлено точное ожидание
для pinned historical D3; byte equality не ослаблена. Final affected migration
PASS1, DDL rollback PASS1 в предыдущем targeted. Product sourcebug не найден.

Strict QA056-01 targeted PASS: realCLI winner1/CONFIG_CONFLICT, active3,
history3/config_applied delta1/hash winner. Tokens не mint-ятся в этом fixture;
private candidates/raw CLI/poststate/snapshot сохранены в cas-targeted.
Legacy C2→D4 exact seven scopes; legacy1 offline exact raw bytes-before-Sweep;
schema4 live-WAL backup/reset/replay/oldSID/auth/fencing; current SQL token
scope и config-human-allowlist after-expiry exact replay; three epic reopen/
rejected-create SQL/cache/restart/replay варианты — targeted PASS.
Python recovery19 PASS с `-W error::ResourceWarning`: explicit closing вокруг
прежних SQLite transaction contexts, schema3→4 fail-closed + unchanged4 recovery.
P3-061 EOF WORK056 очищен, исторический отчёт056 FAIL сохранён.

- 2026-10-18T17:38:54.731826+00:00 source182 unchanged; tests freeze перед единственным combined make check. Annual draft исключён из tracked delivery.

## Финальные результаты / передача

- 2026-10-18T17:41:03.176874+00:00 completed: единственный full make check exit0 за121.056s.
  fmt/vet/fullGo8/infra106/race8/build PASS. Go204 top-level tests плюс
  FuzzParse seed suite в каждой ordinary/race стадии;1 explicit opt-in
  author annual skip. Acceptance37 top-level PASS в каждой стадии.
  Ordinary children безrace; source/harness race PASS — не race-child.
- QA056-01 versionbound CLOSED наc1ef9ae/ordinary SHA140be012…b764:
  targeted и обе full стадии строго1 winner/CONFIG_CONFLICT/hash/history/audit.
  Исторический056 full FAIL остаётся фактом прошлого run, не редактировался.
- Infra106 green с3 ignored destructor ResourceWarning. Targeted modified
  recovery19 clean; unchanged activation_atomic содержит3 старых unclosed
  context fixtures (вероятный источник, creation traceback отсутствует).
  После freeze соседний cleanup не выполнялся; full gate не повторялся.
- Source182 compact75de79…19501b/test43 compact1313e0…6312e2 freeze unchanged.
  Root manifest111 file SHAa603312…c307; его compact aggregate96d3d5…e0fa,
  разные формы hash явно различены в acceptance072.
- Delivery12 файлов перечислены в acceptance072 и delivery-manifest.json;
  production/AGENTS/Davinci/Git/live unchanged. Final evidence log SHA
  5a8a33838dddaf598a11dbb22f13a3caf3ef5f3cc95641a697f2399ad932913f.
- Следующий владелецroot/свежийreviewer: review нового test delta; annual
  method200k/VPS target OPEN и отложены по explicit root, draft не принят.
  Physical2PC/providers/actual clients/UI/OAuth/TLS/Linux/Windows NOT-RUN072.
  Работа072 не является разрешениемdeploy/release/V1. После final hashes STOP.
