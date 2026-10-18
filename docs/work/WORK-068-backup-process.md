# WORK-068 — переносимая процессная приёмка backup

- Родитель TASK-003, исполнитель `/root/backupqa068`, роль tester.
- Статус: в работе; начало 2026-10-18T20:15:07+03:00.
- База `d4337182b1a1353305b06ab9ffcec591433836ba`, source APPROVE065 для
  `aeb085c0b04829dadd90c7b99687396365b28ad6`; wrapper SHA256
  `68731d687e9d03ced19142cb4d1089b1b8240eb767d552719090f60372299bba`.
- Читаются общий процесс/parallel-work/tester Давинчи и AGENTS Accord.
  Named models не закреплены, модель унаследована. Tandem OFF — прямое
  указание пользователя/AGENTS; Davinci, production и Tandem не меняются.
- Запись: новые `tests/acceptance/backup_process*.py`, при необходимости новые
  их stdlib tests, `docs/acceptance-068-backup.md`, эта WORK. Source/Go/infra,
  старые tests/Makefile/root TASK вне области. Git mutations/push/deploy/SSH нет.

## Сценарии до прогона

Штатный Go CLI создаёт schema/config/token/audit. Отдельный Python driver
настраивает только STATE/BACKUPS/BINARY проверенного wrapper. Два backup с
проверкой 1/2→2/2 и точного содержимого закрытых SQLite; readonly plan сравнивает
имена/байты/mtime; исторические synthetic пары проверяют hourly/daily pruning,
latest2 и сохранение corrupt/unknown/symlink. Детерминированный unsafe-source
отказ без подмены оценок места. Restore в новую закрытую temp DB с CLI recovery.
Каждый прогон использует только собственный новый private0700 workspace.
Linux/VPS/cgroup/большая годовая fixture/два физических ПК здесь NOT-RUN.

## Результат

Созданы5 новых файлов, существующие исходники/тесты не менялись:

- `tests/acceptance/backup_process.py` — 6 последовательных process cases,
  private workspace, CLI-bootstrap, real backup/apply/plan, isolated recovery.
- `tests/acceptance/backup_process_support.py` — маленький shared runtime,
  bounded process groups, sanitized partial JSON, readonly SQLite сравнение.
- `tests/acceptance/backup_process_runtime_test.py` — 4 meaningful stdlib
  проверки partial FAIL/hidden stdout/timeout reap/O_EXCL evidence.
- `docs/acceptance-068-backup.md` — отдельные шаги/ожидания/результаты,
  воспроизведение для готового Linux binary+wrapper и ограничения.
- Эта WORK. Root TASK остаётся исключительной областью координатора.

Оба runtime файла меньше300строк (main222/support139); test63.
CLI secret ни разу не читается; tokencreate stdout/stderr сразу DEVNULL,
никаких содержимых token/hash rows или raw CLI streams в JSON. Сравниваются
все schema/rows закрытых source и snapshots только в памяти. Snapshot SHA и
план синтетических имён допустимы; реальные credentials/данные отсутствуют.
Дочерние процессы ограничены60s, каждая process group принадлежит запуску.
Удаляется только новый mkdtemp workspace, pre-existing каталоги не удаляются.

## Фактические проверки

Окружение из JSON: Darwin27.0.0 arm64, Python3.14.5, SQLite3.50.4.
Go1.27.2 darwin/arm64, GOTOOLCHAIN=local; C binary собран самим исполнителем
из HEAD`d4337182b1a1353305b06ab9ffcec591433836ba`, не future D4.

Build exit0:

```sh
env GOTOOLCHAIN=local \
  GOCACHE=/Users/shagi/Desktop/Project/agent-board/.local/toolchains/cache/build \
  GOMODCACHE=/Users/shagi/Desktop/Project/agent-board/.local/toolchains/cache/mod \
  GOTMPDIR=/Users/shagi/Desktop/Project/agent-board/.local/toolchains/tmp \
  /Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go build \
  -o .local/work068/agent-board ./cmd/agent-board
```

Runtime tests exit0, **4 PASS**,0.130s, ResourceWarning treated as error:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -W error::ResourceWarning -m unittest discover \
  -s tests/acceptance -p 'backup_process_runtime_test.py' -v
```

Единственный process run, без ретраев, exit0, **6 PASS/0 FAIL**,1.224781s:

```sh
env PYTHONDONTWRITEBYTECODE=1 python3 -W error::ResourceWarning \
  tests/acceptance/backup_process.py --binary .local/work068/agent-board \
  --backup-wrapper infra/accord/backup.py --output .local/work068/results-macos.json
```

Фактические B068-01…06: 1→2 tokens и1→2 audit records, schema2/full rows;
redundancy1/2→2/2; inventory9 verified/2 ignored;4 точных obsolete pre-delete;
postapply5 verified с latest2/new preferred, unsafe и unknown неизменны;
readonly bytes/names/mtime; ожидаемый fixed unsafe_regular_file отказ;
restore copy затем Go revoke-all+reset,2 revoked и нужные audit events.
Всего10 прямых child invocations;3 wrapper apply вызывают реальные Go backup,
также2 plan,2 CLI tokencreate,1 ожидаемый failure,2 recovery. Private workspace
удалён. Фактические kernel fsync/no-follow, SQLite /dev/fd integrity, SHA и Go
snapshot пути исполнились; это не fault injection каждого syscall.

Синтаксис3 Python файлов через ast.parse PASS; whitespace5 новых файлов PASS;
`git diff --check` PASS. До начала status clean; после только5 разрешённых новых
файлов.120 existing source files cmd/internal/infra/schema/go.mod/go.sum
совпадают с HEAD; manifest фиксирует head/after hashes. Wrapper SHA совпал и
в первоначальном чтении, и до/после runtime. Full suites/race/старые unit tests
здесь NOT-RUN; source32/review7 не приписываются своему процессному прогону.

## Доказательства и freeze

Ignored `.local/work068/`: `results-macos.json`, `process-summary.log`,
`runtime-tests.log`, `source-unchanged.json`, `freeze.json`. Evidence directory0700,
result/log/manifest0600. Input binary в ignored directory, не публикуется.

| Вход/артефакт | SHA256 |
| --- | --- |
| Wrapper до/после | `68731d687e9d03ced19142cb4d1089b1b8240eb767d552719090f60372299bba` |
| C binary до/после | `c7a52a3058b8c09b96b84db7a75dc4cdd6def08a5cce52d6d3000fee21501527` |
| Harness | `650b3a109f322f5451c73e4ebdd8e63466fa54d6586c9c4f40e4bc4d2e9236c8` |
| Support | `91334d537f36a62214f9334acd72d02c15cd8618d1ba5e87f855548e6abee0ac` |
| Runtime tests | `c99f8f67d03e995c13cfc803fd6bf198c17303943f36b3f9ad4dfd4f5bfe77f3` |
| Process JSON | `867728a267598676fc69f6bed0ae473a7c4100989c6e09d445e2e61930fd0587` |
| Runtime tests log | `66df4f61f40647a4fe94e99043b587c50f13c5213afced53e41aa33d622e7a48` |

Полный delivery manifest включая документы и эту WORK создаётся после финальной
записи в ignored freeze.json, без самоссылки WORK на собственный checksum.

## Ограничения и передача

Набор ожидает независимое code/security review до VPS. Автор тестов не выдаёт
себе approve. Следующий владелец root: назначить отдельного reviewer, затем
передать проверенный portable harness DevOps для изолированного Linux unit
CPU50%/Memory256M/Timeout300. Это остаётся **NOT-RUN**;1.22s маленькой macOS БД
не является resource/deadline evidence для Linux или годовой БД.

Annual fixture не генерировалась; готовую проверенную fixture координатор
передаст отдельно. Набор v1 создаёт собственную небольшую CLI БД и не принимает
произвольный production source. Live WAL/writers/server/network/последующий
login после recovery здесь NOT-RUN; tasks/notes/sessions пусты, сохраняются
schema и фактические token/audit rows. Это isolated restore verification,
не production restore, не два физических ПК и не полный V1. Low disk/budget
представлены предыдущими32 unit/7 probes, здесь только реальный unsafe-source
процессный отказ. Offsite/DR отсутствуют. SSH/VPS/Tandem/deploy/live delete,
Git mutations/commit/push не выполнялись.

## Финальное событие

- 2026-10-18T20:20:15+03:00 completed: tester результат выполнен,6 process+4 runtime PASS;
  передача root для независимого review и Linux gate. Freeze: запись собственных
  source/tests/docs прекращена после manifest. Статус — выполнено, не accepted.
