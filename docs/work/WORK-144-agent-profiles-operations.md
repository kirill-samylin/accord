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

## Staging ownership finding

Clean e923273 прошёл exact Linux product12/12 и updater8/8 с actual/proc proof;
Go clean build,1256Next members и источники независимо проверены143. Первый
actual staging дал **FAIL до создания work/journal**: `require_runtime` применял
root-owned operator-file проверку к service-created authority marker. На VPS
маркер regular0600/UIDaccord999; обе systemd конфигурации root0/0644. Остальные
10preflight checks PASS, старые Go/Next5454 active, recovery markers отсутствуют.
Никаких chown/chmod/удалений authority не выполнялось.

Fix: root-only проверки остаются для конфигурации/helper code; marker следует
существующей guard/GoPreflight policy через protected_root+LEGACY.private с
обязательной regular0600 existence. Missing/symlink/public mode отвергаются.
Independent143 fix APPROVE,8/8helpersunits и actual updater8/8 повтор PASS;
полные infra148/148 PASS. Реальный readonly marker-policy recheck на VPS PASS.
E923 не активируется; следующий immutable релиз включает исправление и опять
проходит Linux/stage перед apply. Исходный код Go/Next не менялся.

## Фактическая финальная поставка cbf

Clean cbf202086431335d4657c722a0f75328a09dc4ea: ELF SHA66f949aa6022ade9cf4881c1a8cd65201b35d94c6635db942fcd6f73d42546aa,
Next manifest94186245b3d7ff02546f592343ef32ec44cf9b48a3e00f99ce07d6bec1651d06,
BuildIDhI1Cu6Lg4v0sMm34KPKnO,1256members. Exact Linux product12/12 и updater8/8
PASS; source hashes и actual/proc binding сохранены privateproof. Independent143
APPROVE actualstage/apply после hardened stage (0restarts/7894free).

Actual capable apply PASS: полный pre-snapshot доDDL, owned startup hold,
новые floor/helpers, offline schema3 migration и exact logical preservation
3productionDB. На VPS только default;2дополнительных проекта/7DB проверены
только synthetic, не выдаются за production. Старые pointers/Go не выбирались.

Readonly HTTPS-cbf-01 FAIL был ошибкой тестового empty browser array assertion;
tester145 исправил absent-only и отдельные stage labels, independent143 APPROVE.
Production source не менялся. Повтор cbf-02 PASS6/6:52screens,805browser200,
errors0, prefetch0, только own login/project open/logout. Root visually проверил
profile1440light/390light; hashes всех52screens совпали с report.

Finalize PASS, actual journal complete; full final format2 backup validated на
VPS. Finalproof подтверждает GoPID260083/argv/ELF66f9, NextPID259513/BuildIDhI1,
helper/effective launch/backup PASS, service-UID guard PASS, services/timers active,
NRestarts0, recovery markers0. Pre-snapshot validated и checksum preserved.
Пути/полные hashes и точные evidence refs — в итоговой TASK010; private SQLite
не скачивался. Итоговая независимая evidence приёмка WORK143: FINAL APPROVE.
