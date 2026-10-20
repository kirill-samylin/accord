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

## Последующие source и synthetic проверки

WORK143 source APPROVE первоначального snapshot; полный root `make check`
PASS, включая144infra и60clientunits. Source checkpoint9599962 собран в
отдельном clean managed checkout: Linux Go/Next загружены только в inactive
release dirs; production pointers остались5454. Все17установленных helpers
совпали с completed journal, alias совпал; loaded projects/next/web cache dirs
отсутствуют. Старые rootadmin caches не входят в import chain и сохранены.

Отдельный tester146 нашёл **P1 FAIL** actual updater rehearsal: snapshot-specific
DELETE header check применялся к законной закрытой WAL board после DDL.
9599962 не активируется. Root исправил явный `live=True` validation path:
закрытый WAL header допускается только с `readonly` nonempty journal rejection,
полным integrity/schema/metadata check. Snapshot default по-прежнему требует
самодостаточный DELETE header. Два новых real SQLite regressions PASS;
независимый143 повтор7/7 и fix source APPROVE. Финальные infra146/146 PASS.
Нужны повтор actual updater146 и новая immutable сборка; one-off amendment
TASK009 не переиспользуется, production БД/guards не изменялись.

Linux product-only запуск проверенного9599 Go:12/12 actual cases PASS на
synthetic fixture. Первичная external cleanup probe без SO_REUSEADDR вернула
EADDRINUSE из-за TIME_WAIT после успешного отчёта; это не было скрыто как PASS
самого wrapper. Независимая проверка ss показала отсутствие listeners,
повтор bind с SO_REUSEADDR прошёл; private launcher исправлен. Финальный новый
release будет прогнан с исправленной probe, exact hashes сохраняются отдельно.
