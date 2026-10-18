# WORK-062 — ограниченное локальное хранение резервных копий

- Родитель: TASK-003; исполнитель `/root/backup062`, роль DevOps.
- Статус: реализация завершена, на независимой проверке; база `664eec32431342b78180622b7659ec26758e7f55`.
- Рабочая копия `.local/backup-budget-003`, ветка `codex/backup-budget`.
- Запись: `infra/accord/backup.py`, при необходимости helper/service,
  соответствующие stdlib Python unit/integration tests, `docs/backup-retention.md`,
  эта WORK. Go/migrations/recovery/installer/общая TASK вне области.
- Tandem OFF по прямому указанию пользователя и AGENTS этого проекта.
- Общий процесс, parallel-work и профиль DevOps Давинчи прочитаны read-only.
  Named Claude models в этой среде не закреплены; текущая модель унаследована.
- Live deletion/deploy/push/commit запрещены поручением. Соседний проект не меняется.

## Основание

Измеренный годовой synthetic SQLite: 173051904 bytes; прежние168 копий требуют
29072719872 bytes. Переданный размер VPS диска21028065280, free17726021632;
RAM1002659840/available641052672,1CPU. Это вход координатора, не собственное
повторное измерение VPS. Production/Tandem backups здесь не читаются и не меняются.
Нет offsite; DR от потери VPS/диска не обещается.

## Цель

≤24 свежих hourly buckets + newest UTC daily за7дат; managed total≤5GiB,
последние2 verified snapshots защищены. Временная глубина best-effort. До копии
проверить бюджет/свободное место с DB/WAL reserve, удалять только verified owned
regular0600 пары. Чужие/unsafe/повреждённые файлы сохранять. FAIL при невозможности
сохранить минимум/место; durable publication/checksum/fsync и cleanup ошибок.
Новый продуктовый E2E — после независимого source approve, здесь не добавляется.

## Журнал

- 2026-10-18T19:57:58+03:00 started: прочитаны профиль/процесс/repo AGENTS,
  backup/store snapshot contract и installer ownership; найден explicit
  ADMIN_FILES, поэтому runtime останется self-contained в backup.py.

## Результат

- Self-contained wrapper304 строки; исключение сверх ориентира300 —4 строки,
  позволяет не добавлять runtime helper в explicit installer ADMIN_FILES за
  границами WORK. Функции разделены по проверке/плану/публикации; тесты241/202строк.
-24 UTC hourly buckets и7 UTC дат, budget5GiB, latest min(2,count) защищены;
  обе глубины best-effort. Для одного timestamp порядок по mtime_ns затем UUID.
- До мутаций полный безопасный deletion plan и space preflight: DB+WAL+16MiB,
  free≥2*estimate+512MiB. Sparse-aware reclaim; повторный source/free check.
- Owned regular0600 single-link/no-follow, exact bounded SHA256, full integrity;
  fsync/cleanup собственных новых inode, preserve replacements. Post-success
  ротация повторных slots. stdout/stderr child скрыты; fixed fail reason codes.
- Read-only --plan не создаёт lock, не запускает backup и не удаляет файлы.
  Pre-upgrade/pre-migration/неизвестные/повреждённые snapshots не управляются;
  занимают free, требуют отдельного операторского решения. Offsite/DR отсутствует.
- Уточнение координатора: latest2 — запрет pruning min(2,count), а не искусственный
  FAIL bootstrap. Первый durable verified snapshot success, redundancy1/2;
  следующий2/2. Low-disk bootstrap FAIL до child. Прежнее предложение FAIL после
  первой копии было промежуточным и заменено; frozen source/test/doc согласованы.
- Изменены только backup.py,2 новых stdlib unit/integration файла, policy doc,
  эта WORK. Service/timer/resource limits/installer/Go/recovery не менялись.

## Фактические проверки

Окружение macOS27.0.1 arm64, Python3.14.5, SQLite3.50.4.
Final targeted32 PASS, exit0,2.347s; ResourceWarning treated as error, без warnings.
Команда `python3 -W error::ResourceWarning -m unittest discover -s tests/infra
-p 'test_backup*.py' -v` (в одной строке). Покрыты UTC/hour/day boundary,
budget/min2/1/0, sparse/low disk, checksum/integrity corruption, owner/mode/links/
FIFO/sidecars, same-second order, cleanup failed/successful-corrupt/oversized child,
fsync stage faults, operator replacement/collision, read-only plan, source growth,
bootstrap redundancy. Child — отдельный tiny synthetic SQLite copier, не Go CLI.
Новые product E2E не добавлялись.

`py_compile` трёх Python файлов PASS; `git diff --check` PASS, trailing whitespace
в новых документах/тестах отсутствует. Полный прежний infra suite94 PASS/exit0,
1.726s до последней дельты same-second sorting/+3 targeted cases; в этом широком
прогоне были ResourceWarning из существующих иных SQLite fixtures. Их файлы
не изменялись, targeted frozen32 warnings не даёт. Полный suite после финальной
узкой дельты повторно не запускался; root выполнит единый финальный gate.
Go/race/build N/A для изменённых файлов и здесь NOT-RUN; existing Go CLI/process
проверки не выдаются за повторно выполненные. Source-only review и Linux/VPS
runtime/limits/deadline/restore drill NOT-RUN. Production/live deletions/deploy,
Git commit/merge/push NOT-RUN и запрещены поручением.

## Freeze и доказательства

База `664eec32431342b78180622b7659ec26758e7f55`; commit делает root.
Ignored evidence: `.local/evidence/backup062/verification.json`, `freeze.json`,
`targeted-final.log`, `compile.log`, `diff-check.log`, `tracked.diff`.
Широкий прогон94 доступен в tool output этого исполнения; отдельный raw-log
для него не сохранялся. Не приписывать targeted log этому широкому прогону.

| Frozen file | SHA256 |
| --- | --- |
| `infra/accord/backup.py` | `68731d687e9d03ced19142cb4d1089b1b8240eb767d552719090f60372299bba` |
| `tests/infra/test_backup_retention.py` | `b33a8c3bd1a82e72c3b510e4d44f3abad34bf89f3dd0bd4ae8393f4a70a09fe8` |
| `tests/infra/test_backup_integration.py` | `5a4a22b4ee44fc08860add835a9b29e9664cc303abc6a07f944382a0248b2689` |
| `docs/backup-retention.md` | `ebf17492da2fc450c82452e852b34da052e5d4db714434482d7fc0ffe405680a` |

Targeted log SHA256 `778d47afb3ae4728768823bdf468e9332380a407d4a6ad92936ef340664bfd0f`.
Полный final manifest включая эту WORK находится в ignored `freeze-all.json`;
WORK не включает собственный hash, чтобы избежать циклического checksum.

## Ограничения и следующий шаг

- Независимый reviewer/security должен проверить frozen source и тесты;
  автор не выдаёт себе approve. Следующий владелец — координатор root.
- До live root отдельно смотрит --plan проверенного candidate, protected/delete,
  посторонние/pre-upgrade копии, фактический free и timing. Никаких live-удалений
  этим исполнением. Документ deployment с168 в области root; policy doc явно
  отделяет новую неразвёрнутую версию от старой production.
- `/dev/fd` immutable SQLite и5минутный service deadline на Linux/VPS здесь
  NOT-RUN. Приватный0700 каталог/maintenance lock доверяют root и тому же uid;
  atomic POSIX unlink-if-inode не существует. Preflight не физическая квота:
  чужие writers/WAL могут вырасти; повторные проверки/FAIL не обещают DR.
- Resource limits не менялись; при review/изолированном runtime сначала измерить
  стоимость checksum/integrity существующего каталога под текущими лимитами.

## Финальные события

- 2026-10-18T20:06:26+03:00 completed:
  final targeted32/compile/whitespace PASS, implementation→на независимом ревью;
  точные hashes/evidence зафиксированы. Резервирование Tandem не применялось.
  Freeze: запись исходников/тестов/docs прекращена; передача root для review,
  integration и отдельного решения о runtime/deploy.
