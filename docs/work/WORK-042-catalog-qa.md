# WORK-042 — C process / migration acceptance

- Родитель: TASK-003; роль: тестировщик; исполнитель `/root/catalogqa042`.
- Статус: выполнено / handed_off; независимое review новых тестов — следующий этап.
- Вход: C/native source `2dfd86bf1627775a8fdd87c650d9a3ada4cae17f`, APPROVE041;
  parser033 APPROVE036. Production source заморожен координатором.
- Запись: `tests/acceptance/**`, необходимые C fixtures в `tests/infra/**`,
  эта WORK, `docs/acceptance-042-catalog.md`, ignored evidence
  `.local/evidence/catalog-acceptance-003/`. Production/Makefile/TASK/live/Git
  не изменяются. Tandem OFF по указанию пользователя.
- Модель: унаследованная; закрепление Claude-моделей в этой среде недоступно.

## Проверки

Notes CRUD/replay/pagination/resource и перезапуск реального CLI; exact int64
delta/final total через дочерний stdio worker; participant stats/closed interval;
cancelled blocked duration; canonical legacy schema1 fixture → schema2 с backup,
hash, atomic failure и restart; backup/reset schema2. Полный `make check` один
раз после заморозки набора; race относится к harness, дочерний CLI обычный.
Benchmark 20 sessions / 2000 tasks — отдельный опыт, Mac не целевой VPS.

## Результат

**C-01…C-08 PASS** в [протоколе042](../acceptance-042-catalog.md). Production
source108 files остался побайтно неизменным. Проверяемая версия
`2dfd86bf1627775a8fdd87c650d9a3ada4cae17f`; бинарь tester SHA256
`1ddab9cf59d94ecbf8a32751e6834a025927a7e20fbb919acc52455d592ab107`.

Изменены4 существующих tests: backup process schema2+notes/provenance,
catalog GET helper, proxy12tools, recovery schema99/accepted2. Добавлены
3 Go process suites, canonical synthetic `schema1.sql` и самостоятельный
`catalog_load.py`272 строки; остальные новые Go files95/176/177 строк.
Fixture не выдана за runtime старого executable.

Фактически: targeted C5+3 PASS2.380s; infra15 PASS; backup notes targeted
PASS0.74s; итоговый полный свежий `make check` PASS — acceptance normal16.172s,
race17.829s, infra25 tests0.050s, fmt/vet/build PASS, без cached Go results.
Python3.14 выводит ResourceWarning о synthetic SQLite connections в infra
fixtures; отмечено в отчёте, suite OK. `git diff --check` PASS.

Последовательность сохранена честно: первый полный PASS был до усиления C-06;
после notes/provenance assertion выполнен targeted backup PASS, затем запущен
и завершён второй полный PASS. Указание координатора не повторять full пришло
после завершения этого уже запущенного прохода; новых прогонов не выполнялось.
Race final включает последний Go-test delta; child CLI ordinary, не race build.

Нагрузочный harness Mac smoke2 sessions/10 tasks/10 requests PASS.
Целевой20/2000 на1vCPU/512MiB **NOT-RUN**: root выполняет отдельно на точной
release, метод и ограничения описаны в протоколе. VPS/live/Tandem/remote/Git
не изменялись. Model binding Claude в этой среде недоступен, роль наследуемая.

Evidence `.local/evidence/catalog-acceptance-003/`: source manifest
`92e8c848648a22278e506673c4e8bc2208cffc1b202bb8eced9c57933d29903b`,
final test manifest18 files
`b9fe6407eeae52168b799dd309f79055d087191419054f18cdb867bd336cd2c4`,
results JSON
`4f83e02d3908726366c3e2b54a17e69694c398d9b508675470efeb7bac974084`.
Независимое test/harness review PENDING, назначает root. До approve новые
тесты не считаются независимо принятыми; WORK передана без commit/push.

## Журнал

- 2026-10-18T18:30:05+03:00 — started; прочитаны AGENTS Accord, tester profile,
  общий процесс и parallel-work; подтверждён APPROVE041, запись ограничена
  назначенной областью. Существующие fixtures ожидали schema1/11tools и будут
  актуализированы на schema2/12tools; unknown fixture станет schema99.
- 2026-10-18T18:35:27+03:00 — tested; новый C subset PASS и первый полный
  make check PASS; production остался прежним. Затем усилен schema2
  backup/reset notes/provenance, targeted PASS; load harness Mac smoke PASS.
- 2026-10-18T18:39:33+03:00 — completed/handed_off; финальный свежий full PASS,
  source/test/results manifests сохранены, reviewer получает frozen tests
  и standalone load harness. Общую приёмку/выпуск ведёт root; V1 не объявлен PASS.
