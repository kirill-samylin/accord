# WORK-070 — независимое code/security review метода WORK-068

Вердикт: **CHANGES REQUESTED** для standalone метода на commit
`7200cc9e1fdfc544dd274ccb8f3991c849112885`, base
`d4337182b1a1353305b06ab9ffcec591433836ba`. Один блокирующий дефект ожиданий
теста приведён ниже. Это не дефект или отзыв source APPROVE065 для backup wrapper.

Ревьюер `/root/review070` независим от автора068; профиль/процесс и ограничения
те же, что в [отдельном review067](review-070-integration-config.md).
Проверены frozen5 delivery files +4 evidence files, CLI и source wrapper:
все11 hashes совпали, worktree clean до/после. Manifest.head — входная база,
проверенный фактический HEAD указан выше. Сверка: `.local/work070/backup-freeze.json`.

## Блокирующее замечание R070-B01 — P2

Место: `tests/acceptance/backup_process.py:129–141`, особенно139–141;
связанные допущения — bootstrap52–78 и history82–99.

Метод предполагает, что обе bootstrap copies и retention apply попали в один
UTC hour. Если первая copy создана до границы часа, а вторая после неё,
после третьего apply первая законно сохраняется как newest в отдельном hourly
bucket. Верных retained copies становится6, а не5. Тест выдаёт
`retained_pair_count`; даже без этой проверки следующий цикл ошибочно сравнил бы
первую copy с second_image, хотя её корректное содержимое — first_image.
На ограниченном Linux запуске граница ещё вероятнее из-за меньшего CPU бюджета.
Аналогичные абсолютные ожидания должны учитывать смену UTC даты при построении
history/исполнении wrapper. Это ложный FAIL исправного backup, а не потеря данных.

Независимо воспроизведено без изменения wrapper, CLI или их часов:
собственные реальные bootstrap copies были получены штатным CLI; только имя
первой собственной fixture-пары перенесено в предыдущий UTC hour с пересчётом
точной checksum line. Это допустимое состояние, которое получается при настоящем
переходе часа. History readonly plan прошёл; apply корректно сохранил старый hour,
но метод упал с `retained_pair_count`. Изолированный workspace затем удалён.
Evidence: `.local/work070/backup-hour-boundary/results.json`,
probe `.local/work070/backup_hour_boundary_probe.py`;
status `CONFIRMED_METHOD_FALSE_FAILURE`, old_hour_snapshot_preserved=true.

Нужное исправление: ожидания retained identities и per-copy contents должны
соответствовать реальным timestamp buckets выполненных bootstrap copies и
реальной применимой retention политике. Не удалять корректный первый bucket
ради фиксированного числа и не ослаблять production policy/clock/authority guards.
Добавить meaningful regression для bootstrap через UTC hour, сохранив проверку
содержимого каждой версии. После дельты — новый freeze/hash, targeted author
retest и независимый review затронутой области до передачи метода на VPS.

## Что уже подтверждено

- Driver импортирует reviewed wrapper read-only и меняет только STATE/BACKUPS/
  BINARY. Его run вызывает настоящий Go backup child; SQLite copier/space policy/
  time/fsync/nofollow не заменены. Это qualification wrapper+CLI, без installer.
- Все файлы данных создаются в собственном mkdtemp0700. Root/production/source
  paths не принимаются как DB; удаляется только собственный temp tree.
  Commands не используют shell и stdin, не наследуют credentials; child group
  создаётся start_new_session, штатный timeout завершает её и reap child.
- Token create идёт сразу в DEVNULL, token file не читается. Full schema/rows
  сравниваются только в памяти; в JSON только counts/static reasons/plan hashes,
  bytes counts/timings/exit codes. Реальные secrets/PII/DB rows не записаны.
- Evidence создаётся0600 O_EXCL, перезапись существующего result запрещена.
  Partial report сохраняет предыдущие PASS, failing child exit и case reason;
  timeout/hidden output подтверждены отдельно. Все3 Python файла <300строк,
  shared runtime вынесен, новых существенных дублей не найдено.
- B06803/04 код и авторский hash-bound result проверены:9verified/2ignored,
 4 planned obsolete pairs, unsafe/unknown/symlink unchanged; B06805 — fixed
  unsafe_regular_file отказ без новых backups. Их авторский PASS относится
  к фактически одному UTC hour, не доказывает найденную граничную ветку.

## Фактические проверки

Окружение macOS/Darwin arm64, Python3.14.5, SQLite3.50.4.
Wrapper SHA256 `68731d687e9d03ced19142cb4d1089b1b8240eb767d552719090f60372299bba`;
C CLI SHA256 `c7a52a3058b8c09b96b84db7a75dc4cdd6def08a5cce52d6d3000fee21501527`.
CLI из C base d433718, тождество будущему D4 не заявляется.

| Проверка | Результат |
| --- | --- |
| Авторский frozen process JSON SHA867728a267598676fc69f6bed0ae473a7c4100989c6e09d445e2e61930fd0587 | Проверен;6PASS,0FAIL, workspace removed |
| `PYTHONDONTWRITEBYTECODE=1 python3 -W error::ResourceWarning -m unittest discover -s tests/acceptance -p backup_process_runtime_test.py -v` | Независимо4PASS,0.133s |
| Existing-method subset B06801/02/06 через `.local/work070/backup_review_subset.py` | Независимо3PASS,0FAIL,0.316310s; real CLI+wrapper, restore/revoke/reset |
| UTC hour fixture probe через `.local/work070/backup_hour_boundary_probe.py` | Подтверждён ложный FAIL метода;3 precondition cases PASS/1 expected finding case FAIL |
| SHA/input hashes после запусков, `git diff --check BASE HEAD` | PASS; source/test checkout clean |

Subset JSON — `.local/work070/backup-subset/results.json`, support log —
`.local/work070/backup-support-tests.log`. Полный acceptance повторно не гонялся;
проверяющий воспроизвёл3 существующих cases и отдельную адресную диагностику.

## Ограничения и рекомендации

Оба frozen отчёта честно отделяют source/library checks от process run:
**NOT-RUN** Linux `/dev/fd`, VPS cgroupCPU50%/Memory256M/Timeout300,
annual173051904bytes, большой inventory, production plan/apply, liveWAL/writers,
server/network/login после reset, physical2PC, offsite/DR, полныйV1.
Tasks/notes/sessions пусты; runtime доказаны schema/token/audit rows и isolated
closed restore. Предыдущие low-disk source unit не объявлены process PASS.
Это допустимые границы метода; Linux остаётся последующим обязательным gate.

Отложенная **P3 рекомендация**: при раннем отказе сохранять оставшиеся
предусмотренные case IDs как NOT-RUN, чтобы counts отражали полный suite,
а не только уже начатые cases. Текущий FAIL status видим, ложного PASS это
не создаёт; отдельную дельту ради этой рекомендации делать не требуется.

Проверяющий не менял ни harness, ни wrapper/source, не выполнял Git mutations,
SSH/live/deploy/production delete. Следующий владелец — координатор root:
вернуть одну bounded дельту автору068, source wrapper оставить неизменным.
