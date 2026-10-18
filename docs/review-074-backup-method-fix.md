# WORK-074 — независимое ревью исправленного backup test-method

Вердикт: **APPROVE** для пяти delivery files frozen commit
`c8ce77e039c841e0cb0d67b8924950154c3ea495`, base
`7200cc9e1fdfc544dd274ccb8f3991c849112885`. Блокирующее R070-B01/P2 закрыто:
метод принимает законную первую копию другого UTC часа и проверяет её первую
версию данных. Блокирующих замечаний к этой дельте нет. Это approve метода,
не Linux/VPS qualification, не разрешение live pruning и не полный V1 PASS.

Ревьюер `/root/backupreview074` независим от автора `/root/backupfix073`.
Прочитаны профиль reviewer, общий процесс/parallel-work Давинчи и AGENTS
самостоятельного agent-board; Давинчи не изменялся. Tandem OFF по прямому
указанию пользователя. Named Claude bindings здесь недоступны; унаследованная
модель та же, что у автора, без самостоятельного снижения модели проверки.

## Версия и целостность

Проверены три `tests/acceptance/backup_process*.py`,
`docs/backup-method-fix-073.md`, `docs/work/WORK-073-backup-method-fix.md`,
окружающий approved wrapper, acceptance068 и review065/070.
Независимо совпали все5 delivery hashes,12 hashes авторских evidence,
120 source hashes; source checkout clean до/после, diff check PASS.
Freeze073 SHA256 `feda0cd50e48c57ef0b3b2d779bd9795e9df05f44a185d5b2d306c8fbcf4f116`.
Его поле `head` обозначает базу7200, фактический проверенный commit — c8ce77e.

Wrapper неизменён: SHA256
`68731d687e9d03ced19142cb4d1089b1b8240eb767d552719090f60372299bba`.
Проверен готовый C CLI `.local/backup-qa-003/.local/work068/agent-board`, SHA256
`c7a52a3058b8c09b96b84db7a75dc4cdd6def08a5cce52d6d3000fee21501527`.
Исходники и тесты автора ревьюер не менял; production source APPROVE065 остаётся
отдельным заключением с прежними границами.

## Проверенные свойства и точные места

| Область | Заключение |
| --- | --- |
| Независимость oracle | `backup_process_support.py:76–138`: известные fixture identity/mtime/size/hash/origin; декларативное перечисление24 UTC часовых bucket и7 дат, максимум `(timestamp,mtime,name)`, union latest2, budget удаляет старейшие незашищённые. Production module/planner и plan output не используются для выбора expected names. Это самостоятельный oracle того же контракта, а не вызов production planner. |
| Реальные часы | `backup_process.py:48–67`, support129–138: readonly child зажат настоящими start/end; принимаются только точные наборы в допустимых UTC boundaries; clock backwards даёт fixed FAIL. Apply151–174 восстанавливает UTC instant из новой опубликованной пары. Это корректно для неизменённого `backup.py:259,280–283`, использующего один captured now до и после copy. Секундная точность имени сохраняет bucket/date и отношение к известным секундным timestamp. |
| Exact identities и данные | Main151–177: точное равенство наблюдаемого набора результату обеих фаз; ровно одна новая пара; удалённые checksum отсутствуют; known hash и full schema/rows проверяются по origin. Новая копия обязана содержать second-image. First другого часа сохраняется с first-image. Fixed count5 заменён точным набором, а не нижней границей count. |
| Post-plan и ресурсы | Main48–67,168: независимо проверены verified/ignored, latest2, budget5GiB, floor512MiB, DB+WAL+16MiB estimate, reserve2×estimate и отсутствие free-space pressure маленькой fixture. Post-plan также считается по новым часам; прежнее безусловное delete-empty не навязывается после смены UTC boundary. |
| Latest2 и restore | Main94–111,190–207: second_copy не обязана вечно оставаться latest2 при timestamp/mtime/UUID tie. До rotation сохранён private closed recovery seed проверенной second-image. Из него проверяются exact restore, revoke-all/reset, tokens/audit и неизменность source/backups. Этот сценарий честно уже не требует, чтобы seed пережил rotation под исходным именем. |
| Safety/evidence | Main217–250 и support15–28,45–69,141–206: owned0700 fresh temp,0600 O_EXCL, только свой cleanup, closed SQLite copies, token stdout DEVNULL, sanitized child env, no shell/stdin, bounded child process group и reap. Ранний FAIL сохраняет предыдущие cases/exit/reason, raw output/DB rows/tokens не попадают в evidence. Driver меняет лишь STATE/BACKUPS/BINARY; wrapper clock/copier/space/guards не подменены. |
| Регрессии и структура | Runtime test38–86: exact same-hour5/rollover6/midnight/24th-hour/ties/future/budget. Test148–167 проверяет сам prune и отвергает content/loss/extra,4 прежних runtime safeguards сохранены. Main256/support206/test219 строк; новых существенных дублей или необоснованных абстракций нет. UI N/A. |

Новые stdlib конструкции совместимы с Python3.14: aware UTC datetime,
timedelta и frozen dataclass сверены по официальной документации
[datetime3.14](https://docs.python.org/3.14/library/datetime.html) и
[dataclasses3.14](https://docs.python.org/3.14/library/dataclasses.html).
Зависимости не обновлялись.

## Фактические проверки

Независимое окружение: macOS27.0.1 arm64, Python3.14.5, SQLite3.50.4.

| Проверка | Факт и evidence |
| --- | --- |
| Все затронутые unit/method/runtime | `PYTHONDONTWRITEBYTECODE=1 python3 -W error::ResourceWarning -m unittest discover -s tests/acceptance -p backup_process_runtime_test.py -v`:14 PASS/0.186s, exit0. `.local/work074/unit-method-runtime.log`. |
| Независимые method counterfactual probes | `.local/work074/probes.py -v ReviewProbes`:6 PASS/0.042s, exit0. Checked-plan принимает оба exact midnight набора; отвергает omission/unknown/duplicate delete, wrong estimate/protected, backwards clock и readonly mutation. Prune принимает first-protected/second-retired при одном timestamp с mtime tie-break и отвергает оставшийся orphan checksum. `.local/work074/probes.log`. |
| Author real-process, проверен по hashes | JSON SHA`9e3b6584ccb028e9ea87c3b3debbb3e83620460d31380c80a10099d54d7645ad`:6 PASS/0 FAIL,0.585529s,10 child operations, plan9verified/2ignored, apply5retained/5obsolete; exit1 unsafe-source ожидаем; restore/revoke/reset PASS, workspace removed. Это авторский единственный full run, не независимый rerun. |
| Старый7200 regression, проверен по hashes | JSON SHA`9b98f864308e203525fa211410be0ed7e10692068ae54a51ee67db8d10e36375`:3precondition PASS/1 EXPECTED method FAIL `retained_pair_count`, old-hour preserved=true, workspace removed. Старые extracted файлы и probe также совпали с manifest. |
| Freeze/source/diff | `.local/work074/freeze-before.json`, `freeze-after.json`:5+12 hashes и120 source files PASS до/после, checkout clean, `git diff --check BASE HEAD` PASS. |

Полный process suite повторно не запускался: затронутые method fixtures,
counterfactuals и сохранённый hash-bound реальный прогон достаточны для этой
проверки дельты. Controlled fixtures не выдаются за реальное наблюдение
перехода часа/midnight или за Linux/CLI process evidence.

SHA256 независимых артефактов:

- unit log `691a3b0ee763a0ef8902093282ac84779009ed7470e7890a7980aec5c81008ee`;
- probes script `9d3a089a479c54d6b3a0900b75a9422d62913e521237c44516b30afa6facf93a`;
- probes log `428abc12d3c43864bd0ace866e2f96112492b094e7b3abd9a2c4ca4972c11f14`;
- freeze-after `051573763545d33cbf81483b5844c8060e123140dbcb373e6a66dc4880b2ddc5`.

Evidence каталог0700/файлы0600. Проверки ревьюера выполнялись на собственных
synthetic временных данных без чтения реальных БД/токенов.

## Ограничения и передача

**NOT-RUN:** Linux `/dev/fd`, VPS/cgroupCPU50%/Memory256M/Timeout300,
annual173051904-byte fixture/большой inventory, live WAL/writers,
production plan/apply, server/network/login после reset, physical2PC, offsite/DR,
full Go/race/build, квалификация будущего D4 и полный V1. C binary относится
к base d433718; его тождество D4 не заявляется. Low-disk process fault injection
не добавлен; oracle budget unit не заменяет resource-limit qualification.

Отложенная P3 из review070 сохраняется: при раннем FAIL оставшиеся case IDs
ещё не дописываются как NOT-RUN. FAIL видим, предыдущие результаты сохраняются;
эта рекомендация не стала блокером текущей bounded дельты.

Записаны только этот review, WORK074 и ignored evidence. Git mutations/push,
SSH/live/deploy, source fixes, новые агенты и общая TASK не затрагивались.
Следующий владелец — root: интегрировать APPROVE074 и передать ровно frozen
дельту QA072/DevOps при необходимости. Linux/runtime остаётся отдельным gate;
проверку двух физических ПК это заключение не закрывает. Ревьюер074 STOP.
