# WORK-042 — приёмка C catalog / notes / stats / migration

Набор **042.1**,18.10.2026; независимый тестировщик `/root/catalogqa042`.
Входной source APPROVE: [WORK-041](work/WORK-041-recheck.md).
Версия production source: `2dfd86bf1627775a8fdd87c650d9a3ada4cae17f`.
Запущен настоящий CLI, его дочерний stdio worker и настоящие loopback sockets.
Clock/store/auth/HTTP handler не заменены. Данные, токены и SQLite синтетические,
в private temporary directories; каждый процесс завершён owned cleanup.

## Кейсы для воспроизведения

| ID | Проверка / шаги | Ожидаемый результат |
| --- | --- | --- |
| C-01 | `TestCatalogNotesProcessRestartPaginationResource`: CLI create3 notes с provenance; прочитать limit1; перезапустить сервер; replay исходного create; добавить четвёртую; продолжить старый cursor; прочитать stdio resource; delete исходной; снова restart и replay delete. | Байты replay сохранены; исходный проход содержит ровно3 исходные notes, без дублей/поздней записи. Resource совпадает с новым CLI проходом из4 notes. После удаления3 active/4 include_deleted; provenance/reason сохранены. |
| C-02 | `TestCatalogExactTokensStdioChildWire`: stdio child describe/claim; comment delta9007199254740993, replay; stats до закрытия; transition done total9223372036854775807, replay/get/events; изучить stdout structuredContent; restart; stats с границами на фактическом task_done. | В текстовом JSON и5 структурированных ответах exact int64. По одному delta/final audit, correction=total−delta. Активная delta не попадает в closed-attempt stats; после закрытия ровно1 attempt/first completion, все tokens у участника `metrics-participant`; [from,to) исключает верхнюю и включает нижнюю границу. |
| C-03 | `TestCatalogCancelledBlockedDurationProcess`: link blocks, cancel blocked target, получить event timestamps, restart; stats до и после cancel и с from посередине span. | Span равен наблюдённому времени block→cancel, обрезается по from и перестаёт расти после cancel/restart. Dependency predicate blocked сохраняется у cancelled задачи. |
| C-04 | `TestCatalogLegacySchemaProcessMigrationBackupRestart`: создать canonical SQL schema1 fixture; запустить actual server; проверить schema2, backup/hash/modes/integrity и исторические rows; открыть токен/сессию; stats/get; создать schema2 note и restart. | Backup остаётся schema1, source становится2; исторические task/event/replay bytes сохранены на момент миграции; timestamps не выдуманы, старое событие incomplete. Schema2 note и incomplete history сохраняются после restart; второй backup не создаётся. |
| C-05 | `TestCatalogMigrationProcessFailuresAtomic`: schema1 с препятствием вместо backup directory; затем конфликт notes DDL; отдельно future schema99; попытаться actual serve. | Быстрый отказ до listening; schema и historical task не изменены; note_metadata не появилась. DDL conflict сохраняет исходную таблицу и валидный schema1 backup. |
| C-06 | `TestCLIOnlineSnapshotAndOfflineRecovery`: существующий live-WAL snapshot/reset дополнен schema2 note. | Metadata schema2 и SHA256 соответствуют backup; task/events/notes/provenance точны в копии. Reset отзывает старые токены/сессии/lease, сохраняет notes и историю; новый claim повышает generation. |
| C-07 | `test_recovery.py`: unknown schema99; прежняя schema2 без DDL изменений и failure candidate start. | Unknown compatibility fails closed; schema2 допускает восстановление прежнего executable только при неизменной схеме. Это mocked service commands с реальными SQLite/filesystem, не systemd runtime. |
| C-08 | Существующий `TestProxyProcessConfiguredHost` после интеграции C. | Tools/list через настоящий local TLS proxy содержит12 tools. Отдельное native039 доказательство на11 tools не переписано. |

Schema1 fixture `tests/acceptance/testdata/schema1.sql` воспроизводит DDL из
`dc9c74c:internal/store/store.go`; строки вымышленные. Это явно **не** БД,
полученная запуском старого executable. Исторический idempotency payload
проверен побайтно в SQLite/backup, а create/delete note replay — по actual CLI.
Старый executable не объявляется проверенным на schema2.

## Протокол и результаты

**PASS** всех C-01…C-08 в указанной области. Независимое review новых тестов
ещё требуется перед общей приёмкой. Новый production дефект не найден.

Окружение: macOS27.0.1, darwin/arm64, Go1.27.2; CLI SHA256
`1ddab9cf59d94ecbf8a32751e6834a025927a7e20fbb919acc52455d592ab107`.
SDK consumer — test harness, это не Codex/Claude Code. SDK any decoder может
округлять >2^53; C-02 использует typed JSON text и отдельное чтение actual child
stdout structuredContent, не выдаёт SDK any за точный numeric контракт.
`-race` инструментирует harness/связанные Go packages; child CLI обычный.

- Targeted C:5 top-level tests +3 migration subtests, PASS2.380s.
- Infra targeted:15 tests PASS,0.035s.
- C-06 после добавления notes: PASS0.74s.
- Полный свежий `make check` после очистки test cache: fmt/vet, normal Go,
  infra25, race Go, build — PASS. Финальный acceptance normal16.172s,
  race17.829s, infra25 tests0.050s; ни один Go package не reported cached.
- Первоначальный полный PASS до усиления C-06 сохранён отдельным preliminary
  log; итоговый полный проход выполнен после последней Go-test дельты.
- Python3.14 показывает ResourceWarning о незакрытых SQLite connections в
  synthetic infra fixtures;25 tests остаются OK. Это ограничение fixture
  hygiene, не доказательство runtime leak сервиса.

Промежуточные незачётные попытки выявили ошибки самого нового набора:
read helper добавлял request_id к catalog GET; после note на legacy task
ожидаемо добавлялся ещё один incomplete event. Helper исправлен, сохранение
incomplete сравнивается до/после restart; production не менялся.

Воспроизведение C subset из корня проекта при настроенном Go runtime:

```sh
go test -count=1 -v ./tests/acceptance -run 'TestCatalog|TestCLIOnlineSnapshotAndOfflineRecovery'
python3 -m unittest discover -s tests/infra -p 'test_recovery.py' -v
make check
```

Makefile использует локальный toolchain/cache, если он установлен. Для прямого
go subset в данном стенде использованы те же GOTOOLCHAIN/GOCACHE/GOMODCACHE/
GOTMPDIR, что в Makefile. CLI создаётся TestMain из текущего source.

## Нагрузка: отдельный runnable опыт

`tests/acceptance/catalog_load.py` самостоятельно поднимает и завершает
изолированный CLI; не принимает URL существующего сервиса. Все токены private,
реальных данных нет. По умолчанию20 sessions/2000 tasks, затем20 участников
одновременно начинают по20 последовательных HTTP requests на сессию:400 всего,
до20 одновременно; поровну get/tasks20/events20/stats(owner)/heartbeat.
Seed time исключён из HTTP latency. После seed измеряется restart с2000 tasks.
RSS — sampled server RSS каждые50ms, клиент исключён; p95 nearest rank.
JSON содержит окружение, binary hash, снимок leaf cgroup-v2 файлов процесса
сервера, per-operation latency и наблюдённые пороги. TLS и200000 events/year
не моделируются.

Уточнение метода WORK-047 после REV045-01: после measured requests читается
`/proc/<child-pid>/cgroup`, а соответствующий cgroup2 mount определяется через
`/proc/self/mountinfo` наблюдателя с учётом mount root и экранирования путей.
`cgroup_limits` записывает PID, membership path, mount root/mountpoint, leaf path
и отдельные статусы/значения `cpu.max`, `memory.max`. Корень иерархии не
подставляется вместо leaf; membership `/` может означать namespace root.
Нет однозначного mount, доступа,
валидного значения или стабильного membership — явно `unavailable` (при одном
доступном файле — `partial`). На macOS без procfs — `unavailable`.
Это наблюдённые leaf values, **не effective limits**: ancestor quotas не
читаются, namespace может скрывать предков; `max` не доказывает отсутствие
наследуемых лимитов. Снимок не доказывает неизменность группы на всём интервале.
Для целевого VPS нужны отдельно привязанные к этому запуску unit/PID limits.
Генератор в том же ограниченном unit тоже расходует CPU/память, его RSS не
включён в server RSS. Прежний smoke042 ниже сохранён как историческое evidence;
новая дельта и её smoke — в [WORK-047](work/WORK-047-load-cgroup-fix.md).

```sh
python3 tests/acceptance/catalog_load.py \
  --binary /absolute/path/to/verified-linux-agent-board \
  --output /private/experiment/load-20-2000.json
```

Директория output должна существовать; файл создаётся эксклюзивно. Запускать
на выделенном VPS под заранее заданными1vCPU/512MiB, без других полных прогонов.
Достижение порогов этого смешанного сценария не доказывает все возможные
workloads или годовой event volume.

Mac smoke метода:2 sessions,10 tasks,10 measured requests, PASS. Наблюдения:
p95≈1.77ms, sampled RSS≈22.11MiB, populated start≈13.51ms. Это лишь проверка
запуска сценария на8 logical CPUs; целевые20/2000 на1vCPU/512MiB — **NOT-RUN**
в этой WORK и должны измеряться отдельно координатором на изолированном VPS.

## Доказательства и ограничения

Ignored `.local/evidence/catalog-acceptance-003/`: `source.sha256`,
`test-suite.sha256`, `run-metadata.json`, `targeted-final.log`,
`infra-targeted-final.log`, `backup-notes-final.log`, `make-check.log`,
`make-check-pre-backup-note.log`, `load-smoke.log`, `load-smoke.json`,
`results.json`, `evidence.sha256`, сохранённый actual CLI `agent-board`.
`results.json` связывает результаты с хешами source/test/log/binary;
manifests позволяют reviewer воспроизвести2–3 subset без ещё одного full run.

- Production source manifest108 files SHA256:
  `92e8c848648a22278e506673c4e8bc2208cffc1b202bb8eced9c57933d29903b`.
- Final test manifest18 files SHA256:
  `b9fe6407eeae52168b799dd309f79055d087191419054f18cdb867bd336cd2c4`.
- Results SHA256:
  `4f83e02d3908726366c3e2b54a17e69694c398d9b508675470efeb7bac974084`.

Не выполнялись: mutation live service/VPS, deploy/push/commit, новый TLS/Codex
remote runtime C, второй физический ПК, OAuth, target resource load, Windows,
crash/power-loss fsync durability. Нет UI: screenshots/design review N/A.
Canonical SQL fixture + реальные process migrations не доказывают запуск
исторического executable, откат schema2 к schema1 либо crash-fault durability.
Набор не закрывает полный V1 и не переносит C source PASS на установленный VPS.
