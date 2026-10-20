# WORK-144 — TASK010 capable update и schema3 backup

20.10.2026. Автор root/координатор. Область: `infra/accord/projects/**`,
`tests/infra/**`, release/VPS интеграция; канонические TASK/docs отдельно у root.
Исходный checkpoint867f905, текущий Go/Next VPS5454 по передаче TASK009;
фактическая инвентаризация перед поставкой обязательна. Пользователь поручил
безопасную работающую поставку следующего этапа. Tandem не используется.

## Реализация

- Отдельный `upgrade.py` stage/apply/resume/finalize для уже включённого registry.
  Не использует initial apply, bootstrap-owner или reset/restore productionDB.
- Stage проверяет immutable candidate+installed environment/units, фиксирует
  исходные hashes helpers и неизменённый ingress; Linux Next hardened stage
  переиспользует проверенные primitives TASK009 без смены active pointers.
- Apply ставит owned fsynced inspection marker до stop, под maintenance lock
  делает полный format2 snapshot всех проектов, фиксирует immutable manifest.
  Сначала новый permanent guard floor schema3/profiles, затем capable pointers,
  offline additive migration всех published boards и exact logical comparison
  всех прежних таблиц registry/board/web и immutable private content.
- Снятие только своего marker перед owned startup; actual /proc PID/argv/ELF
  SHA проверяются. Неуспех сохраняет marker/stops/journal blocked-roll-forward.
  Resume того же журнала/релиза не восстанавливает старые БД или прежний Go.
- Finalize после независимого HTTPS smoke требует новый all-project backup и
  повторную проверку effective launch/guard/backup и actual owned PID/SHA.
- Format2 принимает board2/3, проверяет profile columns в3; исторические
  schema1/2 migration snapshots+checksum private/strict allowlist сохраняются
  вместе с проектами и default при copy/validate/restore. Unknown/incomplete
  entries не игнорируются. Registry1/web1/2 не подменяются пустыми хранилищами.

## Проверки и статус

Предварительные125 прежних infra units + capability floor regression: PASS.
Добавлены реальные isolated SQLite snapshot/history/preservation units и
update ordering/failure/resume units. Полный финальный count после review.
Unit/mock не является Linux migration/production доказательством.

Независимый review, process/browser QA, Linux synthetic rehearsal, production
update/full backup/HTTPS — пока NOT-RUN. Physical second PC — NOT-RUN.
