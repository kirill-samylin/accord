# WORK191 — capable schema6 automation infrastructure

Автор: automation_infra. TASK014, CONTRACT APPROVE183. Изменяет только capable
all-project helpers, offline storage preflight и fixtures, capabilities output.
Production, пользовательский worker, GUI и исходный deploy-task012 не изменяются.

Статус: SOURCE READY для независимого review. Process/Linux QA ещё NOT-RUN.

Контур: новый startup guard требует automation1 + board_schema[6] вместе со всеми
прежними floors; snapshot format2 принимает historical2–5 и strict6; schema5
migration backup сохраняется; upgrade применяет только capable6 и сравнивает
каждую историческую logical table/immutable byte; restore остаётся new/offline,
проверяет authority reset и сохраняет unknown workspace reservation/history.

Unit checks:54 infra project tests PASS, включая strict schema6 metadata/index/
future refusal, historical3–5+6 snapshots/schema5 checksums, full history/reservation
restore и отрицательные unchanged authority/ACL epoch/host proof/forged exit cases.
Go projects и cmd/agent-board PASS с private TMPDIR; default macOS TMPDIR отклонён
существующей проверкой небезопасного parent (guards не ослаблены). Race: projects PASS18.403s и cmd PASS2.765s.
Нет process/native/platform proof, mock orchestration не объявляется runtime.

## Файлы и поведение

- Infra: guard.py, layout.py, validate.py, migration_files.py, upgrade_storage.py,
  upgrade.py, restore.py, README.md; новые automation_schema.py/restore_authority.py.
- Go preflight: storage_preflight.go, automation_storage_preflight.go,
  automation_storage_preflight_test.go, migrate_storage_test.go.
- Capabilities: новые cmd/agent-board/capabilities.go и capabilities_test.go;
  прежний projects_test.go fixture обновлён до automation1/schema6. main.go меняет
  координатор, не автор этой карточки.
- Unit: tests/infra/test_projects_guard.py, test_projects_restore.py,
  новый test_projects_automation.py.

Restore проверяет результат maintenance reset на SQLite до complete marker:
новый registry signing key, текущие credentials revoked, sessions/actions/epochs
очищены, membership access epochs увеличены; board tokens revoked, managed host
proofs очищены/worker epochs увеличены, queued commands cancelled, неизвестный
process state needs_recovery. Automation policy revisions/steps/plans/questions/
attestations/reservations сохраняются, policies/capabilities revoked, unfinished
workflows needs_recovery с новым control epoch. Restoration не заявляет exit proof.

Updater baseline_helpers сохраняет требование предыдущего complete journal.
Текущий production273 pending-https-smoke нельзя обойти ради schema6: сначала
реальная native приёмка TASK013/finalize, затем новый capable stage/update.
Реальные данные, сервисы и runtime здесь не трогались. Новых process/E2E tests
автор не создавал: отдельный QA после source APPROVE остаётся обязательным.

## P1 independent review correction

REQUEST CHANGES189: прежнее case folding и удаление whitespace внутри SQL могло
принять UNIQUE index с JSON path `$.WORKFLOW_ID` либо `$.workflow_ id`, где SQLite
получает NULL вместо нужного поля. Исправлено: сравнение canonical migration DDL
после удаления только внешнего whitespace. Регистр и каждый внутренний byte SQL,
включая literals, сохраняются. Обе malformed path регрессии добавлены в Python
validator и Go offline preflight.

Повтор:54 infra units PASS; targeted Go projects race preflight/offline migration/
reset/restore PASS6.607s. Full projects race пока FAIL в чужом managed_test.go64
(viewer worker: expected200, actual401 owner lost access); cmd race NOT-RUN compile
blocked незавершённой WORK193 (releaseAutomationExit/executeAutomationCheck). Старые
full race PASS выше относятся к предыдущему checkpoint и не подменяют текущий.
Product/test files frozen заново для independent rereview; process/deploy NOT-RUN.
