# Приёмка backup — B068 v1

Набор создан независимым tester после source APPROVE065. Это переносимая
проверка отдельного Python wrapper и настоящего Go CLI. Подмены copier,
SQLite API, места/бюджета, времени wrapper или nofollow/fsync здесь нет.
Исторические копии имеют synthetic timestamp/имена; это fixture retention,
а не многодневное наблюдение. Все данные создаются этим запуском в новом
private0700 каталоге внутри выбранного каталога доказательств и удаляются
только после завершения принадлежащих запуску дочерних процессов.

## Повторение

Нужны POSIX Linux/macOS, Python с sqlite3 и /dev/fd, готовый исполняемый CLI,
проверенный `backup.py` и оба `backup_process*.py` runtime файла рядом.
Go toolchain, сервер, сеть и существующие credentials для запуска не нужны.
Нужны свободные место/права по настоящей политике wrapper: floor512MiB,
2×(source DB+WAL+16MiB). Запускается от обычного изолированного uid;
под root эта приёмка не проводилась.

```sh
mkdir -m 700 /path/to/new-private-evidence
python3 -W error::ResourceWarning /path/to/tests/acceptance/backup_process.py \
  --binary /path/to/reviewed/agent-board \
  --backup-wrapper /path/to/reviewed/backup.py \
  --output /path/to/new-private-evidence/results.json
```

Каталог output уже существует, принадлежит uid и имеет0700. Сам JSON создаётся
O_EXCL0600; существующий результат не перезаписывается. Значение `--timeout`
(по умолчанию60s, максимум300s) ограничивает каждый дочерний процесс, не обещает
общий systemd deadline. При timeout раннер завершает только созданную им новую
process group, ждёт child и фиксирует FAIL. Перезапуск упавшего теста ради PASS
не предусмотрен. До очередного действия сохраняются частичные результаты;
на ошибке остаются ранее завершённые кейсы, PID/operation/exit/timing и класс
ошибки. Содержимое exceptions и raw stdout/stderr в JSON не пишутся.
Вывод token create сразу направляется в DEVNULL, token file не читается.

Driver создаётся внутри temp и импортирует readonly source wrapper,
устанавливая только STATE/BACKUPS/BINARY; вызывает `run()` с той же обработкой
фиксированных ошибок, что entrypoint. Отдельный wrapper process вызывает
настоящий Go CLI `backup`. Это проверка runtime wrapper, не тест установленного
пути `/opt/accord/current` и не systemd/unit/installer smoke.

## Сценарии и ожидаемый результат

| ID | Шаги | Ожидаемый результат |
| --- | --- | --- |
| B068-01 | CLI token create `owner=qa068,machine=synthetic` в новый private файл; wrapper apply; offline integrity/schema/все строки сравнить с закрытым source | schema≥2, один token и audit, одна exact0600 DB+SHA пара, redundancy1/2, нет sidecars |
| B068-02 | Ещё один token create, второй wrapper apply; сравнить обе закрытые копии с их соответствующим source image | первая копия не изменилась; во второй2 token rows и прирост audit; redundancy2/2 |
| B068-03 | Создать7 закрытых synthetic исторических пар: older/newer в одном hourly bucket, older/newer одной старой UTC даты, день−6, день−7, день−40; добавить bad checksum, managed symlink, unknown/pre-upgrade имя; --plan | verified9, ignored2; protected=последние2; exact4 устаревших delete; bytes/names/mtime state и backups неизменны |
| B068-04 | Настоящий wrapper apply с этим каталогом; затем --plan и offline сверка | 4 запланированных пары удалены; после новой копии старейший дубль текущего часа тоже может уйти; verified5, ignored2, delete пуст; prior newest и preferred buckets сохранены; unsafe/unknown bytes и symlink неизменны |
| B068-05 | Только собственному source изменить mode на0640; apply; вернуть0600 | exit1, ровно `unsafe_regular_file`, без новой копии/изменений всех backups |
| B068-06 | Копировать newest verified DB в новый private закрытый restore каталог; full SQLite compare; CLI revoke-all reason restore; maintenance reset reason restore | integrity/schema2,2 токена revoked, audit `tokens_revoked_all` и `database_recovered`; source и backup пары неизменны |

Latest2 защищены относительно текущего verified inventory. После принятия
новой копии защищены новая и предыдущая newest; бывшая третья может стать
устаревшим дублем. Набор не подменяет это правилом вечного хранения первых двух.
Full row/schema сравнения выполняются только в памяти; в JSON идут counts,
fixed reasons, snapshot имена/план и SHA входных файлов, без token/hash rows.

## Результат macOS

Единственный зачётный прогон этой версии: **6 PASS /0 FAIL**,1.224781s,
exit0;2 tokencreate,3 wrapper apply с настоящими childCLI,2 readonly plan,
1 ожидаемый unsafe-source отказ,2 команды offline recovery. Временный workspace
удалён. `results-macos.json` записан в ignored `.local/work068/`.
БД73728bytes, estimate16850944bytes; free в plan244324257792bytes.
Environment: Darwin27.0.0 arm64, Python3.14.5, SQLite3.50.4; C binary собран
из HEAD`d4337182b1a1353305b06ab9ffcec591433836ba` Go1.27.2, GOTOOLCHAIN=local.
Это C build, его тождество будущему D4 не заявляется.

- Wrapper SHA256 до/после:
  `68731d687e9d03ced19142cb4d1089b1b8240eb767d552719090f60372299bba`.
- Binary SHA256 до/после:
  `c7a52a3058b8c09b96b84db7a75dc4cdd6def08a5cce52d6d3000fee21501527`.
- Harness SHA256:
  `650b3a109f322f5451c73e4ebdd8e63466fa54d6586c9c4f40e4bc4d2e9236c8`.
- Support SHA256:
  `91334d537f36a62214f9334acd72d02c15cd8618d1ba5e87f855548e6abee0ac`.

Отдельные meaningful stdlib проверки раннера: **4 PASS**,0.130s. Проверяются
сохранение предыдущего PASS при child exit7, отсутствие private stdout в evidence,
частичный FAIL и reap по timeout, запрет перезаписи существующего отчёта.

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -W error::ResourceWarning -m unittest discover \
  -s tests/acceptance -p 'backup_process_runtime_test.py' -v
```

## Оставшиеся границы

**NOT-RUN:** Linux /dev/fd runtime; Linux/VPS systemd CPUQuota50%/MemoryMax256M/
TimeoutStartSec300; annual173051904-byte fixture и большой накопленный каталог;
production readonly plan/live pruning/deploy; два физических ПК; полныйV1.
Большая fixture не генерировалась. Этот набор всегда создаёт небольшую БД;
будущее измерение annual performance — отдельный прогон/метод DevOps, которому
координатор передаст проверенную годовую fixture. Его результаты здесь не
предвосхищаются. Сам набор сначала должен пройти независимое ревью.

Live WAL/online writers этим прогоном **NOT-RUN**: сервер не запускался,
source создавался/менялся штатными закрывающимися CLI операциями. Исторические
копии синтетические; tasks/notes/sessions пусты. Фактически проверено сохранение
schema, всех таблиц/строк двух состояний, token/audit и isolated recovery.
Не проверяется успешная работа восстановленного сервера/клиентов после reset.
Low-disk/budget fault injection — source32 unit +review7 probes в WORK062/065,
здесь не повторялось и не выдаётся за process result. Offsite/DR отсутствуют.

Использованные stdlib API сверены по официальной документации
[Python subprocess](https://docs.python.org/3/library/subprocess.html),
[tempfile](https://docs.python.org/3/library/tempfile.html),
[sqlite3](https://docs.python.org/3/library/sqlite3.html).
