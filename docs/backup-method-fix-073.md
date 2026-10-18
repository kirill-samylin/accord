# WORK-073 — метод backup после R070-B01

Эта дельта исправляет standalone harness WORK-068 на базе
`7200cc9e1fdfc544dd274ccb8f3991c849112885`. Исходная frozen приёмка068 не
переписывается. Source APPROVE065 и wrapper не изменены; config067 вне области.
Независимый review070 нашёл ложный FAIL при bootstrap через UTC час: корректная
первая копия остаётся отдельным bucket с первой версией данных, поэтому число
может быть6, а не5. Свежий независимый review дельты ещё требуется.

## Что изменилось

- Harness объявляет известные snapshots сразу после bootstrap и создания
  synthetic истории: имя, UTC timestamp, mtime ns, логический размер DB+SHA,
  SHA256 и происхождение содержимого (`first` либо `second`). Полные schema/rows
  остаются только в памяти. План wrapper не формирует ожидаемые identities.
- Независимый oracle перечисляет24 UTC часовых интервала и7 календарных дат,
  выбирает максимум `(timestamp, mtime_ns, name)` в каждом, объединяет с latest2.
  При logical budget pressure удаляет старейшие незашищённые preferred пары;
  невозможный budget остаётся ошибкой. Production planner не импортируется и
  не вызывается как oracle. Размеры/лимиты проверяются отдельно:5GiB,
  source DB+WAL+16MiB, reserve2×estimate, floor512MiB.
- `--plan` проверяется относительно узкого реального интервала до/после child:
  допускается только один из точных наборов на входящих в него UTC boundaries.
  Это учитывает midnight и конец часового окна без фиксированного delete count.
  Изменение часов назад даёт явный `fixture_clock_monotonic` FAIL.
- Apply проверяет точный набор известных имён после pre-copy и post-copy
  политики. Новая фактически опубликованная копия даёт timestamp единственного
  UTC instant, который wrapper использует в обеих фазах. Все сохранённые пары
  сверяются по заранее известному hash и полному содержимому их версии;
  новая копия также обязана иметь полное содержимое второй версии.
- Удалены фиксированный count5 и безусловное ожидание сохранения second-copy.
  Restore использует отдельную закрытую копию второй версии, сохранённую до
  rotation внутри собственного private temp workspace, и по-прежнему проверяет
  Go revoke-all/reset, source и каталог backups после recovery.

Непроверенные/unknown/symlink пары по-прежнему обязаны остаться побайтно и по
mtime неизменными. Реальный driver меняет только STATE/BACKUPS/BINARY; wrapper,
его clock, SQLite copier, filesystem guards и available-space API не подменены.
Маленькая process fixture требует отсутствия free-space pressure. Low-disk
process fault injection этой дельтой не добавлен; budget oracle проверен unit.

## Воспроизведение

Запуск и ограничения CLI из [приёмки068](acceptance-068-backup.md) сохраняются;
новый suite marker — `backup-process-068-v2-work073`.
Нужны три runtime/test файла этой дельты, проверенный wrapper и готовый CLI.
Для method/runtime unit:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -W error::ResourceWarning -m unittest discover \
  -s tests/acceptance -p backup_process_runtime_test.py -v
```

Новый output JSON должен отсутствовать; родительский каталог owned0700.
Полный real-process run (один запуск, без ретраев ради PASS):

```sh
env PYTHONDONTWRITEBYTECODE=1 python3 -W error::ResourceWarning \
  tests/acceptance/backup_process.py \
  --binary /path/to/prebuilt/agent-board \
  --backup-wrapper infra/accord/backup.py \
  --output /path/to/private0700/results.json
```

## Фактические проверки073

| Проверка | Факт |
| --- | --- |
| Старый7200 hour-boundary probe | EXPECTED FAIL: `retained_pair_count`,3 precondition PASS/1 finding FAIL; старый час законно сохранён; собственный temp удалён |
| Final Python unit/method/runtime |14 PASS,0.197s;4 прежних runtime safeguards +7 oracle +3 method tests |
| Controlled method fixtures | Реальный метод `prune` принимает точные6 на переходе часа и5 на midnight; отклоняет неверное содержимое первой/новой копии, потерю первой и лишнее имя |
| Clock/oracle regressions | Same-hour5 / rollover6;7-я дата истекает в midnight;24-й час истекает отдельно от календарной даты; mtime до UUID, затем UUID при точном равенстве; future snapshots; latest2 при budget pressure |
| Полный локальный real-process |6 PASS/0 FAIL,0.585529s,10 прямых child invocations; закрытые SQLite schema/rows; plan9 verified/2ignored; apply5 retained/5 obsolete deleted; private workspace удалён |
| Среда | Darwin27.0.0 arm64, Python3.14.5, SQLite3.50.4 |
| Source guard |120 файлов cmd/internal/infra/schema/go.mod/go.sum неизменны; wrapper и C binary SHA до/после совпали |
| Статические проверки | ast.parse3 файлов PASS; git diff --check PASS; main256/support206/tests219 строк |

Clock-boundary fixtures детерминированы заданными UTC временами, без sleep или
ожидания реального часа. Они проверяют test-method на scripted file fixture;
это отдельно от полного запуска с настоящими wrapper/CLI. Финальный полный
процессный запуск не выдаётся за наблюдение реального перехода часа или суток.
Старый probe использует реальные bootstrap copies в своём private workspace и
переименовывает только первую fixture-пару в предыдущий час с точной SHA line.

## Доказательства и границы

Ignored `.local/work073/`: `results-macos.json`, `process-summary.log`,
`runtime-tests-final.log`, `old-hour-boundary/results.json`, старый probe и
извлечённые через read-only git show файлы7200, `source-before.json`,
`source-unchanged.json`, `freeze.json`. Все evidence файлы0600, каталоги0700;
реальный CLI остаётся в исходном ignored пути, в Git не копируется.

| Проверенный вход/результат | SHA256 |
| --- | --- |
| Wrapper | `68731d687e9d03ced19142cb4d1089b1b8240eb767d552719090f60372299bba` |
| C binary068 | `c7a52a3058b8c09b96b84db7a75dc4cdd6def08a5cce52d6d3000fee21501527` |
| backup_process.py | `fb9e39fcc2a0ea71ed9e92cfcf95a8ab68255eb36788a58b6c6d261c217aef8c` |
| backup_process_support.py | `ec104719bf0bb72043c4e6e3ea3cbaf418be97e6ed558aaec1f323bb871dbf7a` |
| backup_process_runtime_test.py | `9fb0820260d541976c8c63be75205b4024757eb5b91bef3eedace7d63e137aa7` |
| Final process JSON | `9e3b6584ccb028e9ea87c3b3debbb3e83620460d31380c80a10099d54d7645ad` |
| Final unit log | `cce3907d3f708401b21c6a4aff19b3971d4cfdb0f70eb5aa8a3f4f48af45f4fb` |

Linux/VPS/cgroup CPU50%/Memory256M/Timeout300, annual fixture, большой inventory,
live WAL/writers, server/network/login, два физических ПК, offsite/DR — **NOT-RUN**.
Full Go/race/build не запускались: изменён только Python test-method. Binary
относится к C base d433718, будущий D4 этим запуском не квалифицирован.
Tasks/notes/sessions пусты; реальные token/audit rows проверены только в памяти.
Новый набор не проверяет disk-pressure процессно, не меняет backup policy,
не выпускается на VPS и не закрывает полный V1.

Следующий владелец — root: передать ровно5 frozen delivery files на свежий
независимый review. После approve можно передать эту дельту QA072/DevOps;
их старую копию harness исполнитель073 не менял. Автор не выдаёт себе APPROVE.
