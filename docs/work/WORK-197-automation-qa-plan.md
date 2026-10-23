# WORK197 — schema6 acceptance alignment и bounded QA plan

23.10.2026. Отдельный tester; root координатор, reviewer169/189.
Разрешён только source prep двух существующих process acceptance tests на базе
scoped SOURCE APPROVE191 infra/preflight. Composite185/186/187/193 ещё не APPROVE;
новые TASK014 process/native/E2E sources не создаются и не выполняются.
Production196 finalized273 и realMac WORK195 PASS5 сохраняются отдельно.

## Узкий test delta — SOURCE READY / runtime NOT-RUN

Только `tests/acceptance/backup_process_test.go` и `migration_process_test.go`.
Остальные acceptance files/production/GUI/credentials не изменялись.

- Backup metadata schema5→6; schema mismatch имеет собственное сообщение,
  timestamp/checksum проверяются отдельно. Реальный опубликованный snapshot DB
  дополнительно проверяется на фактическую schema6/automation1 authority.
- Legacy migration schema5→6; проверка версии/authority отделена от byte-exact
  retained task/event/idempotency rows. Существующие snapshot/reset/auth/fencing
  assertions не ослаблены.
- Общий private test helper в migration file читает actual SQLite:9 automation
  tables/key,data columns, единственная automation_metadata row(id1/version1),
  два exact UNIQUE execution indexes из independently approved191 canonical DDL.
  Применяется только TrimSpace снаружи; JSON path case/internal bytes сохранены.

Fingerprints:

- backup_process_test.go128 строк:
  `1620f54f429ff65e37e332bc38e6c24540c8ea01ee81037bd6e24ac64fa6da1c`;
- migration_process_test.go214 строк:
  `2f5c145c7c468e419853156c78246c9ae776b7273eb4fc026db8d107637e68ac`.

gofmt и diff-check PASS. **Изменённые process tests NOT-RUN до169 SOURCE APPROVE.**
Root prior `.local/work186/all-packages-race.txt` имел PASS всех product packages
и ровно два failures старых schema5 assertions; это не runtime proof нового delta.
После test approval root назначит frozen candidate и targeted запуск этих двух
existing cases в private TMPDIR, с binary SHA и source pin до/после.

## План native small write + Git commit — только письменный

Gate: composite189 SOURCE APPROVE; QA-authored harness source review; exact root
GO для ограниченной записи и commit в synthetic HOME fixture. Старое разрешение
WORK195 на чтение GUI не используется как разрешение писать в GUI или менять
native grants. Actual execution/new harness пока NOT-RUN.

1. Fresh private0700 HOME fixture, isolated Git repo и начальный commit с одним
   `message.txt` с известными bytes. Git identity только local fixture config,
   без global settings/remote/push/hooks и пользовательских файлов. Явно проверить
   отсутствие inherited hooks до native запуска. Свой loopback project/worker,
   никакого подключения fixture к production board или реальному GUI.
2. Одно bounded actual Codex invocation через source-approved candidate adapter:
   изменить только `message.txt` на точные новые bytes и создать один Git commit.
   Permissions только требуемые fixture checkout; никакого approval bypass,
   глобальных CLI/auth/sandbox изменений или расширения MCP grants ради успеха.
   Если обычные permissions отказывают, сохранить FAIL и точную безопасную стадию,
   остановиться; не перезапускать с более широкими флагами.
3. Лимит180s для native turn, только один attempt. Проверить provider native UUID,
   предметный результат, один новый commit с ожидаемым parent и exact tree/path
   diff, allowlisted bytes и clean worktree после commit. Сопоставить actual
   Git/check evidence с candidate manifests, tuple/generation и process exit;
   CLI exit0 или текст агента сами по себе не PASS.
4. Очистить только owned process group/fixture services по доказанной ownership,
   сохранить private hashes/journal/receipts и результаты. Timeout/unknown остаётся
   FAIL/needs_recovery без ложного exit proof. Публичный отчёт без provider outputs,
   секретов, реальных проектных IDs. Windows/Claude/physicalsecondPC NOT-RUN.

## Последующая QA после composite approval — только план

| Контур | Bounded проверка и требуемый proof |
| --- | --- |
| Intake/pipeline | Один synthetic requirement через UI и exact duplicate retry через scoped MCP; один workflow; actual analyst→validated plan→developer commit→независимые reviewer и QA на exact SHA/generation→result. Distinct executors/principals и реальные task leases, не профили вместо identities. |
| Negative/fix | Один заранее ограниченный дефект в fixture, один negative review/QA и не более одного fix cycle. Stale SHA/epoch/capability evidence rejected; повторное review+QA exact нового commit, лимит/budget overflow требует видимого вмешательства. |
| Human control | Isolated pause/clarify/question→answer/resume/cancel, lost response exact retry и restart. Queued vs applied, exact native UUID, fresh board session/task fencing, stop proof, retained reservation on unknown. Не использовать реальные production assignments. |
| UI | Production Next local build, actual browser requirement/status/questions/controls, current-authority/CAS/lost-response/revoke boundaries; screenshots без секретов. Same-Mac two principals не выдавать за второй физический ПК. |
| Linux restore | Exact reviewed candidate на отдельной private Linux fixture: schema5→6 capable all-project snapshot/migration, все logical rows/ACL/profile revisions/dedup guard retained, automation floor/tables/indexes. Offline restore в новый root: auth/proofs/caps revoked, epochs advanced, unknown reservations/history retained, workflows needs_recovery, ничего не autolaunch. Старый binary rejected. Production untouched. |

Каждый новый process/E2E source отдельно пишет QA после нужного source gate,
reviewer approves до запуска. План не является согласием на production release,
перенос GUI/credentials, глобальный provider config, push или реальный второйПК.
Общую TASK редактирует только root; эта карточка не объявляет TASK014 готовым.
