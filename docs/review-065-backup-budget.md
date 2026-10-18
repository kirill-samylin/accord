# WORK-065 — независимое code/security review WORK-062

Вердикт: **APPROVE** для source/test/doc снимка
`aeb085c0b04829dadd90c7b99687396365b28ad6` относительно
`664eec32431342b78180622b7659ec26758e7f55`. Блокирующих code/security замечаний
в согласованном объёме не обнаружено. Это source approve: Linux/VPS runtime,
ограничения unit, восстановление и live pruning этим заключением не подтверждаются.

Ревьюер `/root/backupreview065` отделён от автора `/root/backup062`.
Профили reviewer/security и процесс Давинчи прочитаны read-only. Именованные
Claude-модели недоступны; использована унаследованная модель без снижения модели
относительно автора. Tandem отключён прямым указанием пользователя; SSH,
production/Tandem данные, deploy, Git mutations и push не выполнялись.

## Проверенная версия и объём

Проверены пять файлов commit: `infra/accord/backup.py`,
`tests/infra/test_backup_retention.py`, `tests/infra/test_backup_integration.py`,
`docs/backup-retention.md`, `docs/work/WORK-062-backup-budget.md`.
Сверены окружающий Go CLI (`internal/store/backup.go`), service и explicit
`ADMIN_FILES` installer, security/service contracts и freeze manifest автора.
Независимый `git diff --exit-code HEAD --` для пяти frozen файлов PASS;
`git diff --check` PASS. Исходники/тесты автора не менялись.

Итоговый SHA256 `infra/accord/backup.py`:
`68731d687e9d03ced19142cb4d1089b1b8240eb767d552719090f60372299bba`.
Полный manifest пяти файлов — `.local/evidence/backup062/freeze-all.json`.

## Проверенные свойства

| Область | Заключение и место |
| --- | --- |
| UTC policy и минимум | `backup.py:127–164`: текущий час +23 предыдущих bucket, текущая UTC дата +6 предыдущих дат; newest verified по timestamp/mtime/name; protected latest min(2,count), future не заполняет slots. Две защиты могут дать дополнительную пару одного часа, что явно описано. |
| Бюджет и свободное место | `backup.py:147–181`: полный план до мутаций; logical DB+SHA budget5GiB; physical reclaim консервативен для sparse; reserve2×(DB+WAL+16MiB)+512MiB; невозможный план FAIL. Unknown/unsafe/corrupt пары не входят в удаления, но занимают фактический free. |
| Перед и после копии | `backup.py:272–285`: повторные DB/WAL/free checks до child, post-copy real size и floor до принятия пары; bootstrap допускает1/2, следующий успех2/2; failed copy сохраняет прежние protected пары. |
| Файловые границы | `backup.py:33–56,92–124,184–192`: uid/mode/regular/nlink, NOFOLLOW/NONBLOCK, dirfd, точный bounded SHA, полная integrity_check, sidecar rejection; удаление требует повторной проверки и identity совпадения. Unsafe/missing пары остаются для оператора. |
| Публикация и cleanup | `backup.py:195–247`: pre-existing DB/SHA/sidecar отклоняются, child output скрыт, checksum O_EXCL, fsync файлов/каталога, финальный verify; отказ удаляет только наблюдённые новые inode. Go CLI до wrapper публикует проверенную БД через hardlink без overwrite и удаляет staging. |
| Readonly plan и lock | `backup.py:250–271`: plan открывает существующий lock readonly, ждёт flock до inventory, child не запускает, имён/содержимого не меняет. Apply использует тот же maintenance lock. Это сериализация запусков wrapper; не заявляется универсальное fencing всех root/same-uid процессов. |
| Ошибки и конфиденциальность | `backup.py:207–208,295–304`: stdout/stderr child подавлены, на entrypoint OSError печатается только тип, BackupError содержит фиксированный код. Независимая инъекция private exception text не попала в stderr. |
| Совместимость и структура | `install.py:21` включает self-contained backup.py, новых runtime зависимостей/файлов не требуется. Unit resource limits и CLI argv не менялись. Source304 строки: исключение+4 явно обосновано в WORK, дополнительный helper потребовал бы несогласованного installer изменения. Tests241/202 строки; новых существенных дублей не обнаружено. |

`immutable=1` допустим только для неизменяемого snapshot: SQLite пропускает
locking/change detection; поэтому source DB проверяется Go online snapshot,
а wrapper проверяет отдельную завершённую копию без sidecars. Контракт API
сверен по [SQLite URI](https://www.sqlite.org/uri.html),
[Python sqlite3](https://docs.python.org/3/library/sqlite3.html),
[os](https://docs.python.org/3/library/os.html) и
[flock](https://docs.python.org/3/library/fcntl.html).

## Фактические проверки

Окружение: macOS27.0.1 arm64, Python3.14.5, SQLite3.50.4.

1. Независимо запущены затронутые tests:
   `python3 -W error::ResourceWarning -m unittest discover -s tests/infra -p 'test_backup*.py' -v`.
   **32 PASS**, exit0,1.939s; warnings не обнаружены. Raw output — tool output
   исполнения WORK-065, не подменяется авторским targeted log.
2. Собственные локальные probes:
   `PYTHONDONTWRITEBYTECODE=1 python3 -W error::ResourceWarning .local/work065/probes.py`.
   **7 PASS**, exit0,0.149s. Это direct wrapper/unit probes на временных synthetic
   данных, без нового product/process E2E, без реального Go child и без VPS.
   Дополнительно проверены:
   - partial checksum write и flush ENOSPC с очисткой только новой пары;
   - отказ финального verify после fsync, сохранение двух прежних пар;
   - missing checksum самой новой DB не вытесняет две verified защиты;
   - readonly verify не меняет DB/SHA bytes/mtime/ctime и не создаёт sidecars;
   - план блокируется на занятом lock до inventory, после release завершается без child;
   - коллизии каждого SQLite sidecar и unsafe source завершаются до child;
   - entrypoint скрывает private текст OSError.
3. `git diff --check` и сверка неизменности frozen пяти файлов: **PASS**.

Ignored evidence: `.local/work065/probes.py`, `.local/work065/probes.log`.
SHA256 probes `66628eb17b577947979da27b37d512818118f38aeda0bfd17de21134e7be6c1d`;
log `9c73adae5f1ae44ecd235e36635af5cb03d1f224766d98178237dccb65c12abd`.

## Ограничения и следующий владелец

**NOT-RUN:** Linux `/dev/fd`/SQLite compatibility, systemd CPU50%/Memory256M/
Timeout5min на целевой машине, реальный каталог/объём/free, live plan/apply,
restore drill этой версии. Go suite/build и полный infra suite здесь не запускались:
Go source не изменён, единый широкий gate выполняет координатор. Авторский прежний
94 PASS не считается независимым final full-suite результатом WORK-065.

До включения policy координатору остаётся отдельный Linux/runtime gate:
измерить inventory+reverify старого каталога под текущими лимитами и readonly plan
проверенной версии, сверить protected/delete/free и только затем решать выпуск.
Локальный source approve не переносится на неизвестную новую дельту или фактическое
удаление backup на VPS. Новые продуктовые E2E теперь могут быть переданы tester
в рамках согласованного задания после этого approve.

Остаточные принятые границы: root/same uid доверены; POSIX unlink не атомарно
условен по inode; preflight не квота от стороннего роста WAL/disk; SIGKILL/power loss
могут оставить orphan. Unknown/pre-upgrade/pre-migration файлы вручную разбирает
оператор. Offsite отсутствует, потеря VPS/диска этим backup не покрыта.

Отложенных обязательных source исправлений нет. Следующий владелец — координатор
root: интеграция заключения, единый gate и независимый tester в разрешённом scope.
