# WORK-146 — отдельная process QA capable update TASK010

20.10.2026. Tester146, отдельно от product authors141/142, operations144 и
reviewer143. Начато только после source APPROVE143. Область записи:
`tests/process/task010-update/**` и этот WORK. Общая TASK/source/infra/VPS не
редактируются. Production SSH/credentials и Linux execution — координатор.

## Границы проверки

Новый stdlib Python harness создаёт NEW private fixture в ignored
`.local/task010-update-qa/`. Настоящий прежний5454 executable создаёт registry,
default и два проекта, repository associations, native epics/tasks/notes и
идемпотентные записи. Каждый case копирует только synthetic bytes, явно
перепривязывает synthetic default paths и использует actual new executable.

Исполняются реальные `upgrade.apply`/`fail_closed`/`resume`/`finalize`, fsynced
inspection marker, managed pointer operations, helper hash guards, format2
snapshot/validation, `projects migrate-storage`, SQLite preservation и owned Go
HTTP processes. Отдельный worker действительно завершается `os._exit(73)` после
Go pointer и до Next pointer; новый worker читает persisted journal и продолжает.
Linux дополнительно проверяет реальные `/proc` cmdline/ELF SHA собственного PID.

**Явные симуляции:** systemd lifecycle/introspection, Next bundle и Next runtime.
Это проверка operator flow с actual Go/SQLite, не доказательство настоящего
systemd/Next deployment. UID adapter применяет владельца текущего fixture UID;
реальные production root ownership/Next/systemd проверяет координатор отдельно.
Ни один системный service или порт7892/7893/7894/3100/3101 не используется:
канонические аргументы только маршрутизируются в ограниченные fixture hooks,
Go слушает выбранный свободный ephemeral loopback port. Любая иная команда
host.run отклоняется. Timeout worker завершает собственную process group.

Cases: normal success; startup failure после migration и resume; abrupt exit
после первого pointer и отдельный process resume; manual marker; installed
helper drift; complete expected atomic temporary; unknown partial temporary.
Неизвестные marker/helper/temp bytes остаются без удаления. Snapshot2 сохраняется,
finalize создаёт отдельный snapshot3. Запрещён auth reset/restore/legacy rollback
в operator command hook. Source/test hashes фиксируются до и после прогона.

## Обнаруженный product defect

**P1 — live WAL header ошибочно проверялся как standalone snapshot.**
Actual new `projects migrate-storage` успешно мигрировал прежний schema2 WAL
state в3. Затем `upgrade_storage.migrate_storage` вызывал `validate.board` на
live closed board. `validate.database` требует SQLite header18:20 `01 01`,
правильно для self-contained snapshot, но корректная WAL-mode board после
закрытия сохраняет `02 02`. Поэтому каждый реальный update такого состояния
останавливался `snapshot database is not self-contained` до нового listener.

Evidence: private `.local/task010-update-qa/run-1792505198656803000/success/worker.log`.
Root и независимый reviewer143 уведомлены. Исправление должно отделять закрытое
live storage schema/integrity от snapshot header checks, сохраняя отказ на
unresolved journals и полную schema validation. Snapshot validation ослаблять
нельзя. До исправления и нового review/run operator QA — FAIL.

Первые сбои note fixture без обязательного source_task и constructor/rebind/
TIME_WAIT настройки были ошибками самого harness; product source не менялся.
Final clean run/hash provenance и независимое test APPROVE — впереди.

## Исправление и финальный локальный прогон

Root исправил P1: `validate.database/board` имеют explicit `live=False` default;
snapshot по-прежнему требует self-contained header01/01. Только offline-live
проверка принимает допустимый WAL02/02 и сначала вызывает `layout.readonly`,
отвергающий unresolved WAL/rollback journal. `upgrade_storage` выбирает live=True.
Source fix и новые unit regression проверяет reviewer143 отдельно.

После исправления новый clean fixture:
`.local/task010-update-qa/run-1792505477106052000/results.json` — **8/8 PASS**.
Это1actual old5454 seed плюс7 перечисленных operator cases. Все три проекта
имеют реальные задачи/заметки/репозиторий; независимо от operator flag сравниваются
старые строки всех таблиц семи registry/board/web DB. Отдельный abrupt worker
действительно exit73 оставил activation-started/journal/snapshot/owned marker и
разные Go/Next pointers до миграции; новый process resume завершил roll-forward.
Snapshot2 сохранён, новый full snapshot3 проверен. No reset/restore command
разрешён handler'ом updater; unknown marker/helper/temp bytes сохранены.

Darwin/arm64, Python3.14; new executable SHA256
`ff598499f30ca6da6b3caa4a074fdbe056f0bd77339f49b9e4031895375d3043`;
verified old5454 SHA256
`448602f68a24688b86ba2aaec06a36e064765d22b4bb3b3c5f6d7bd92797e95d`.
New local build provenance отдельно от clean Linux release, не объявляется
VCS-clean публикацией. Report SHA256
`8f490492f1cacf1150992fdf7fbe7635f9193b8b9fb838adeec937cae9ea39ff`.

Source/test manifest неизменен до/после:4Python tests aggregate
`0c154872e29e25b46ef3a730d664fc1f4ca97621cda9fb50d27477180a831ecd`;
39operator `.py/.conf/.in` aggregate
`ded9ec8e842c8a76a29898cf2f77a7358ff1cee7ded58e9e2f4d9ba93fc697f8`.
Форматaggregate: sorted `relative_path + ' ' + file_sha256 + '\n'`.
Все новые файлы до300строк, `git diff --check` PASS. Каждый PASS/FAIL report
сохраняет initial source manifest, source-unchanged и sanitized failure label.

Независимый test review — ожидается reviewer143. Linux actual subprocess run,
production systemd/Next/HTTPS — NOT-RUN этим исполнителем; координатор запускает
после test APPROVE. Physical second PC — NOT-RUN.

## Независимая приёмка тестов

Reviewer143: **APPROVE source/safety/evidence146**. Он пересчитал report SHA и
все43 test/operator hashes: совпадают с текущими файлами. Собственный отдельный
clean rerun reviewer:
`.local/task010-update-qa/run-1792505635949157000/results.json` — **8/8 PASS**.
WORK143 фиксирует и независимый review исправления P1/7snapshot units.
Tester146 source и WORK заморожены для финального immutable release координатора.
Linux/production результаты добавляет координатор; локальные PASS не заменяют
actual systemd/Next/HTTPS и physical second PC остаётся NOT-RUN.
