# WORK-143 — независимый review TASK010

20.10.2026. Reviewer отдельный от backend141/frontend142/root operations144.
Область записи только этот WORK; исходники и общая TASK не изменяются.
Review контрактов, source, авторских тестов и затем отдельного QA evidence.

## Текущий вердикт

Первый проход: Operations REQUEST CHANGES; исправления проверены ниже.
Текущий source verdict: **APPROVE backend/frontend/operations и author tests**.
Это разрешение перейти к отдельным process/browser QA, не production closure.
Production/physical second PC этим reviewer не запускались: NOT-RUN.

## Обязательные findings начального прохода

1. **[P1] Offline update не выполняет миграцию.**
   `infra/accord/projects/upgrade_storage.py:migrate_storage` вызывает
   `sessions list --db`, однако `cmd/agent-board/admin.go:listSessions` открывает
   `store.OpenExisting`, который только проверяет supported schema и не вызывает
   `migrate`. Schema2 остаётся2, следующий `validate.board(...schema3)` отвергает
   storage; каждый реальный TASK009→TASK010 update остановится до запуска.
   Исправить отдельной existing-only offline migration операцией с полной
   preflight/сохранением старых строк и использовать её в updater. Проверить
   настоящий schema2 fixture и новый executable; mock ordering этого не доказывает.
   Координатор принял finding, исправление ожидается.

2. **[P1] Restore старых snapshot ломается на безусловной profile таблице.**
   `internal/store/persistence.go:loadBoard/saveBoard` безусловно используют
   `agent_profiles`, но `OpenExisting` намеренно читает schema1/2 без миграции.
   `projects.ResetOffline` открывает каждую schema2 board именно этим методом,
   затем `RecoverState` вызывает `loadBoard`: no-such-table вместо безопасного
   auth reset. Та же несовместимость касается legacy admin mutations schema1/2.
   Сохранить schema-aware load/save или выполнить явно выбранную безопасную
   миграцию до reset. Проверить старый format2 schema2 snapshot→новый actual
   reset/restore, а также существующие maintenance/revoke операции.
   Координатору передано; исправление ожидается.

3. **[P2] SIGKILL оставляет atomic temporary entries, мешающие resume.**
   `host.atomic` использует `.NAME.next-new` O_EXCL, `host.switch` —
   `.current.next-new`. После остановки процесса между temp fsync/create и
   rename destination может быть правильным старым файлом, но temp остаётся.
   `upgrade.require_helpers` этого не обнаруживает; следующий install/switch
   снова падает. State/data остаются failclosed, однако partial-install resume
   не соответствует обещанному восстановлению. Нужен narrowly owned recovery
   по expected bytes/hash/target либо явная документированная ручная inspection
   процедура, без удаления неизвестных файлов. Регрессия с реальными leftovers.
   Координатору передано; disposition ожидается.

## Проверенный ops source

Отдельный capable-update journal, отсутствие повторного bootstrap-owner, owned
fsynced hold до stop, полный all-project snapshot под maintenance lock, новое
schema3 capability floor до DDL, неизменные ingress/env/unit hashes и exact
resolved systemd arguments сохранены. Требования old-Go/DB rollback не ослаблены.
После запуска проверяются listener owner, stable MainPID, /proc argv и ELF SHA.
Final backup и HTTPS остаются отдельными реальными приёмками.

Исторические migration snapshots имеют exact schema1/2-64hex.db и checksum;
unknown/incomplete/symlink/tamper отвергаются. Published/default copies включают
их в format2 manifest; restore default migration directory добавлен. Pending/
quarantine raw sets сохраняются отдельно. Schema3 snapshot проверяет profile
columns; прежние registry1/web1/2 не меняются. Source hashes не заменяют runtime
evidence. Existing Python compiled caches желательно исключить inventory guard:
`-B` запрещает запись, но само по себе не запрещает загрузку старого __pycache__.

Legacy `/opt/accord/admin/web-backup.py` alias теперь сверяется с установленным
web/backup.py перед stage и по baseline/candidate hashes во время update;
неизвестные bytes не перезаписываются. Эти изменения review прочитаны.

## Фактические проверки этого прохода

`python3 -B -m unittest discover -s tests/infra -v`: **135/135 PASS**,
включая5 новых actual isolated SQLite profile/history cases и4 mocked updater
ordering/failure/resume cases. Python3.14. ResourceWarning старых SQLite fixtures
не преобразован в failure, счётчик не скрыт. Runtime/provider/production из этих
unit/mock результатов не следует. До устранения findings ops APPROVE не выдаётся.

Архитектурный open disposition WORK140 закрыт координатором: MCP profiles/profile
read resources + расширенный Describe, без увеличения12tools. Новые operations
REST/CLI-only. Product review и review авторских tests продолжаются.

## Повторный source review и dispositions

- P1 offline migration: исправлен. `projects migrate-storage` вызывает новый
  existing-only `MigrateStorage`: полный published inventory Preflight до первого
  DDL, затем Store.MigrateExisting с полным source schema validation. Операция
  не открывает listener, не вызывает transact/expiry и не трогает registry ACL.
  Updater проверяет explicit storage_migrated=true и точное число boards, затем
  schema3 и hashes старых logical rows. Повтор уже schema3 не создаёт новую копию.
  Авторский integration проверяет3board (default/active/archived), все старые
  таблицы и registry authority, missing/future/incomplete preflight до DDL.
- P1 old restore/admin: исправлен. Profile load только schema>=3; пустые
  profile maps не генерируют profile SQL при save legacy board. ResetOffline
  проверяет published board/web перед явной MigrateExisting и лишь затем делает
  auth reset. Профили schema3/revisions остаются без изменений. Обязательная
  actual old-snapshot→new-executable restore проверка остаётся в QA gate.
- P2 temp leftovers: исправлен узко. Complete root-owned regular temp с exact
  expected bytes/mode либо root-owned symlink с exact target разрешают resume;
  different/partial/manual entries сохраняются и требуют inspection. Три новых
  filesystem tests покрывают expected file/link и сохранение unknown bytes.
- P2 malformed browser profile query: исправлен ParseQuery+error, отдельный
  regression включает malformed percent и semicolon. Native parser уже строгий.
- Дополнительный ops finding new-path collision исправлен: ранее не известный
  prior manifest candidate path может существовать только с exact candidate hash;
  неизвестный root file не принимается baseline и не перезаписывается. Новый
  regression проверяет сохранение bytes. Old helper drift сверяется с completed
  previous release journal; incomplete prior/alias drift/compiled cache отвергаются.

## Product source review

Рассмотрены все новые profiles/core/store/project/transport/web/CLI/bridge files,
изменённые session/persistence/migrations/preflight/JSON/action цепочки и Next
profile screens/forms/routes/types/styles. Контракт role+ID/version match не
позволяет human Describe обходить viewer guard. ACL/epochs проверяются общим
actorCall до Store и cached browser replay; project archive запрещает writes.
Same-name записи не объединяются. Current profile и immutable revision отдельны;
session snapshot независим от последующих edit/archive. Singleton/lease/SHA gates
не переписаны. Event vocabulary расширен, тексты instructions не идут в events.

Profiles list исключает instructions; общий sessions list также исключает их,
session detail читает сохранённый snapshot явно. DTO provenance содержит session
ID, не token ID. Unknown/duplicate JSON проверяются до canonical decode; CLI и
stdio bridge сохраняют это правило до своего map decode. MCP остаётся12tools,
новые read resources зеркалируются bridge с подстановкой только своей session.

Next source APPROVE: данные backend, quota-bounded lists, plain-text rendering,
история ревизий без write controls, явное разделение sessions/profiles и честные
provider/execution границы. Применён vercel-react-best-practices; новых libraries
нет. Existing project QueryClient/mount/cancel boundaries сохранены. Dialog draft
фиксирует base version, conflict не rebases текст молча; uncertain retry хранит
original body/action/project path. Viewer controls подавлены; серверный ACL остаётся
authoritative. Process/browser interactions ещё требуют отдельной реальной QA.

## Независимые проверки после исправлений

- `make test`: cmd + все internal packages PASS. Два существующих acceptance
  tests FAIL на прежних schema2 ожиданиях (backup metadata и migration assertion);
  это не объявлено общим green. Отдельный tester после source APPROVE адаптирует
  process expectations к3 и проверит их смысл, reviewer затем рассмотрит diff.
- TypeScript PASS и60/60 client units PASS на bundled Node24.19.0.
  Первая попытка web-test системным старым Node закончилась bad-option; повтор
  на закреплённом runtime PASS, dependencies/package.json не менялись.
- Финальный на этом проходе infra144/144 PASS (Python3.14). Новые helper collision/
  prior journal/cache и filesystem temporary tests прочитаны независимо.
- Production build/vet/race автора и координатора не заменены предположением
  об их завершении: итоговый exact revision evidence добавляется при закрытии.

Source snapshot после исправлений: aggregate SHA256 от sorted lines
`relative_path + ' ' + file_sha256 + newline` (не package/build hashes):

| Область | Файлов | SHA256 |
| --- | --- | --- |
| cmd/internal *.go включая author tests |229|cc9742fd2c8ed8de6b20ea512a0572c7f765dd1934a4c77a612d514070c51962|
| web/src все файлы |102|0699b056624152d733119bf6cd59f70147848936a0056aea596686c88ac9ed82|
| infra/accord *.py/*.conf/*.in |39|2c8c191f4c04fd0f80b7c69116d0c0800472e58a678b310871813435afc107ca|
| tests/infra test_*.py |21|69dd710007a5ee1d7e90b3fbb9d3f13f7e4a36f3e48159191c6257216f5e059e|

Открытых source findings в этом snapshot нет. **APPROVE** для перехода к
независимому process/browser tester. Actual Linux upgrade/interrupt/resume/
old-binary guard/restore, production HTTPS/full backup и preservation evidence
остаются обязательными незавершёнными этапами; source APPROVE их не подменяет.

## Первый tester diff и prior operator evidence

Отдельный tester145 после source APPROVE адаптировал ровно два прежних
acceptance expectations к schema3. Review **APPROVE**: backup metadata SHA/time/
private-mode assertions сохранены; migration сохраняет точные исторические
task/event/replay bytes и дополнительно требует обе profile tables. Переход2→3
не скрывает пропущенную миграцию и не ослабляет проверки старых данных.
Независимый targeted повтор этих двух process tests (`-count=1`):2/2 **PASS**.

Root private prior-operators proof прочитан: completed prior journal,17helper
hashes без mismatches, backup alias совпадает, loaded projects/next/web caches0.
Старые rootadmin `__pycache__/web-backup.*.pyc` и `restore.*.pyc` вне новой
importchain: текущие explicit spec paths идут `next/host.py` и `web/backup.py`,
проверяемые подкаталоги уже покрыты reject_compiled_helpers. Расширять отказ на
неиспользуемые manual/legacy entries или удалять их оснований нет. Этот вывод
об import paths проверен source; remote inventory выполнен root и подтверждён
прочитанным private evidence, не выдаётся за независимый SSH login reviewer.

## Новые process/browser tests — промежуточный review

Tester145 пишет tests после source APPROVE; просмотрены начальные harness/
catalog/native/web/sqlite_fixture/run и Chrome forms/isolation/browser. Сценарии
используют настоящие old/new executable, HTTP/CLI/stdio MCP, реальные commits
до lost-response, frozen body retry, две вкладки и held A read. Только synthetic
private fixtures; секретные значения не выводятся. Initial process-only report
12checks PASS прочитан, но не принят за окончательный evidence после дальнейших
изменений tests. Report честно помечает unspecified source revision/dirty status.

Перед окончательным test APPROVE запрошены усиления:

- Restore2/3 должен сверять также tasks/notes/repos/blobs и прежние events при
  разрешённых reset append/auth/lease изменениях: одни revoked counts/markers
  и profile equality не доказывают сохранность остальных данных.
- Damaged3 startup должен проверять, что missing revision table не создана
  заново, помимо отказа запуска и сохранности blobs.
- Actual stdio MCP resources: negative foreign profile и foreign SID checks.

Tester принял замечания; final run/hashguard должен относиться к исправленным
tests. Ошибочные первые предположения tester о native Sessions full snapshot и
необходимости новой capability после validation400 не меняли product source.
