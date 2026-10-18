# WORK-045 — независимое review C process tests / load harness

18.10.2026; reviewer `/root/qatestreivew045`, отдельный от автора source и
тестировщика042. Профиль reviewer и общий процесс прочитаны; закрепление
именованных Claude-моделей здесь недоступно, модель унаследована.

## Вердикт и версия

**Go process tests + Python recovery delta042: APPROVE.** Новых блокеров в них
нет. **README: APPROVE** в пределах current-source и раздельно указанного
deployed/native evidence. **Load harness: REQUEST CHANGES**: один блокер
REV045-01 до целевого VPS-замера. Это не отзыв source APPROVE041. Общий
набор042 пока не получает безусловный approve из-за метода нагрузки.

Проверен HEAD `2dfd86bf1627775a8fdd87c650d9a3ada4cae17f` и незакоммиченная
QA042 suite042.1. Побайтно сверены108 source и18 test файлов:

- source manifest SHA256 `92e8c848648a22278e506673c4e8bc2208cffc1b202bb8eced9c57933d29903b`;
- test manifest SHA256 `b9fe6407eeae52168b799dd309f79055d087191419054f18cdb867bd336cd2c4`;
- results042 SHA256 `4f83e02d3908726366c3e2b54a17e69694c398d9b508675470efeb7bac974084`;
- binary042 и независимо собранный child совпали:
  `1ddab9cf59d94ecbf8a32751e6834a025927a7e20fbb919acc52455d592ab107`;
- README snapshot SHA256 `2035d5000faa92dda27328e12470c1e00b998f608e6f0436167121736c8366e4`.

Все12 записей `evidence.sha256` и все artifact references из results042 совпали.
Production/tests проверяющий не менял. D/E worktrees не проверялись.

## Блокирующее замечание — полный пакет за один проход

**REV045-01 [P2] — неверный scope ресурсных лимитов.**
`tests/acceptance/catalog_load.py:80–86,230` читает только
`/sys/fs/cgroup/{cpu.max,memory.max}`. При обычном transient systemd unit
процесс и дочерний CLI принадлежат вложенной cgroup. Корень может показывать
`max` либо не иметь этих файлов, хотя unit ограничен1CPU/512MiB. Отчёт не
записывает membership/path, а `docs/acceptance-042-catalog.md` называет значения
фактическими cgroup limits. При последующей приёмке такой JSON не доказывает
условия замера. Membership и наследование cgroup описаны в
[официальной документации Linux](https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html#processes).

Воспроизведено без VPS: synthetic root `max 100000`/`max`, membership
`0::/system.slice/accord-load.scope`, nested `100000 100000`/`536870912`.
Вызов настоящей функции с перенаправленными файловыми путями вернул root max
и ни разу не прочитал membership; evidence `cgroup-repro.json`.

Требуемый результат: до зачётного запуска либо читать cgroup v2 membership
из `/proc/<child-pid>/cgroup` (или self с подтверждённым наследованием),
сохранять path и наблюдённые leaf limits с оговоркой о mount/ancestor limits;
либо явно назвать поле снимком корня и привязать к тому же запуску отдельное
доказательство лимитов unit/PID. Не превращать root max/отсутствие файла в
подтверждение1CPU/512MiB. Добавить targeted synthetic nested/root/unavailable
fixtures; для этой дельты полный `make check` повторно не нужен.

## Осмысленность тестов и безопасность

- C-01 проверяет exact create/delete replay после restart, high-water cursor
  с поздней вставкой и без дублей, notes resource против CLI, soft-delete и
  provenance. Утверждения различают потерю/дублирование/позднюю запись.
- C-02 использует typed JSON text и отдельно разбирает фактический stdout
  дочернего stdio worker. Пять structured Task frames должны содержать
  2^53+1/MaxInt64; replay проверяется числом comments/audit events. Проверяются
  correction delta, closed-attempt окно [from,to), participant owner вместо
  координатора и persistence. SDK any не выдаётся за exact numbers.
- C-03 берёт ожидаемую длительность из двух наблюдённых server events,
  проверяет clipping и прекращение роста после cancellation/restart. Нет
  фиктивного clock и несвязанного sleep; gate predicate проверяется отдельно.
- C-04 schema1 DDL сопоставлен с `dc9c74c:internal/store/store.go`: структура
  canonical, rows synthetic. Реальный новый server делает миграцию; проверены
  schema2, integrity/checksum/modes backup, исторические task/event/replay bytes,
  отсутствие придуманных timestamps, incomplete history и restart notes.
  Это не запуск старого executable и не legacy replay API test.
- C-05 различает backup obstruction, DDL conflict и future schema99;
  требует отказа до listening/за5с и отсутствия частичного DDL/изменения task.
  Общая command deadline12с ограничивает зависание; превышение5с не даёт PASS.
- C-06 дополнение проверяет schema2 metadata, notes/provenance в live-WAL
  snapshot и после reset; прежние auth/history/fencing assertions сохранены.
  C-07 unknown99 и неизменённая schema2 проверяют recovery policy на настоящих
  SQLite/filesystem с mocked system commands. C-08 tools count12 согласован
  с source; прежнее11-tool native039 evidence не переписано.

Затронутые файлы до300 строк; новых необоснованных дублей не найдено.
CLI/process helpers переиспользуются, read helper не добавляет mutation
request_id к catalog GET. Нет подавления проваленных assertions, безусловных
PASS или skip вместо обязательной проверки. Токены/БД synthetic/private;
подробный credential output подавлен, stdio проверяется на отсутствие bearer.
Cleanup ограничен owned processes. Существующий сервис/Tandem не затрагивался.

## Независимое воспроизведение и зачётные логи

С GOTOOLCHAIN=local, GOCACHE/GOMODCACHE/GOTMPDIR из Makefile запущено:

```sh
.local/toolchains/go/bin/go test -count=1 -v ./tests/acceptance \
  -run '^(TestCatalogExactTokensStdioChildWire|TestCatalogLegacySchemaProcessMigrationBackupRestart|TestCatalogMigrationProcessFailuresAtomic)$'
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/infra -p 'test_recovery.py' -v
```

**PASS** три top-level process tests и три failure subtests,1.662с;
**PASS**15 Python fixtures,0.034с. Это настоящий CLI, stdio child, loopback и
SQLite; systemd/Linux runtime не эмулируется как PASS.

Остальное сверено по manifests/results/logs042. Финальный full `make check`:
normal acceptance16.172с, race acceptance17.829с, infra25/0.050с,
vet/fmt/build; cached Go results отсутствуют. Первый full до усиления backup
notes сохранён отдельно, второй после последней Go дельты; последовательность042
не скрывает повторный запуск. Reviewer полный набор повторно не запускал.
Race инструментирует harness/связанные packages; child binary обычный.

Evidence045: `.local/evidence/review-045/manifest-verification.json`,
`log-verification.json`, `process-subset.log`, `infra-recovery.log`,
`cgroup-repro.json`. SHA256 subset log
`a50bc6bf62958cd8a98b7eede4e6c66e74f88aa1c46d674db67ff09ab23ac9db`,
infra log `d9a56e85c3e2fb0b60564808c025f96ab905e5b4890ad1b24193db7f2d1d80bb`,
repro `1df3a0c72cb20d9cc4876ce4203373123408fd23b65af0ff3ff8df8bb8853e01`.

## Нагрузка, README и границы выводов

Harness принимает только binary, создаёт private isolated DB/токены и свой
loopback server; URL существующего сервиса не принимается. По умолчанию20
sessions/2000 tasks seed и populated restart; barrier запускает20 участников,
у каждого20 последовательных requests:400 всего, до20 одновременно. По80
get/tasks20/events20/stats(owner)/heartbeat. Seed исключён из HTTP latency;
p95 nearest rank, per-operation metrics есть. RSS sampled у child каждые50мс.
Клиентский RSS исключён; генератор в том же ограниченном unit также потребляет
CPU/память. Это свойство метода нужно сохранять при сравнении результатов.
Threshold flags — наблюдения; завершение сценария не означает достижения порогов.

Mac2/10/10 smoke042 подтверждает запуск метода; это не20/2000, не1CPU/512MiB,
не TLS/200000 events/year/длительная нагрузка. Target load корректно NOT-RUN.
README явно разделяет source/deployed evidence и оставляет fullV1, physical
second PC/Windows/OAuth и target load открытыми. Текущие source12 tools не
выдаются за переиспытание native11 tools. README approve относится к снимку выше.

NOT-RUN reviewer: VPS/deploy/live mutations, target load, повторный TLS/Codex,
второй физический ПК, OAuth, historical executable, crash/power-loss durability.
UI/design review N/A. Commit/push не выполнялись.

## Отложенные рекомендации P3 — необязательные

- `tests/infra/test_recovery.py`: закрывать SQLite connections явно при следующем
  изменении fixtures. Context manager sqlite3 управляет transaction, но не
  закрывает connection; ResourceWarning Python3.14 уже честно указан042.
  Это fixture hygiene, не доказанный leak сервиса и не blocker.
- В описании метода уточнить «20 участников ×20 последовательных запросов»:
  это исключит прочтение как400 одновременных requests. Новый прогон не нужен.
