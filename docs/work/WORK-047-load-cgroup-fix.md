# WORK-047 — узкая правка cgroup metadata load harness

- Родитель: TASK-003; роль: developer + targeted unit/smoke verification.
- Исполнитель: `/root/loadfix047`; статус: ready_for_review / frozen.
- Вход: source APPROVE041/045, QA042 full PASS; единственный blocker
  REV045-01 из [review045](../review-045-catalog-tests.md).
- Запись: `tests/acceptance/catalog_load.py`, новый helper
  `tests/acceptance/catalog_cgroup.py`, `tests/infra/test_catalog_load_cgroup.py`,
  узкое уточнение `docs/acceptance-042-catalog.md`, эта WORK, ignored evidence047.
- Tandem OFF по прямому указанию пользователя. Production/live/Go/README/TASK,
  соседний проект и чужие worktrees не изменяются. Commit/push не входят.
- Общий процесс, developer/tester profiles и parallel-work прочитаны.
  Именованные Claude-модели здесь не закреплены; текущая модель унаследована.

## Метод

Harness передаёт PID собственного сервера после measured requests. Helper читает
его unified membership и mountinfo наблюдателя, убирает cgroup2 mount root из
membership при построении leaf path, декодирует mountinfo escapes, отклоняет
traversal/неоднозначные mounts и проверяет стабильность membership вокруг чтения.
Leaf `cpu.max`/`memory.max` имеют отдельные availability statuses. Не бывает
fallback к root или self membership. Scope и `effective_limits: unavailable`
записаны и при успешном leaf snapshot: ancestor quotas не вычисляются, namespace
может скрывать предков. Снимок не гарантирует постоянство группы за весь прогон.

Официальные источники метода проверены 18.10.2026:
[Linux cgroup-v2 membership и hierarchy](https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html#processes),
[mountinfo fields](https://man7.org/linux/man-pages/man5/proc_pid_mountinfo.5.html).
Raw membership, в отличие от escaped mountinfo, проверен по
[kernel proc_cgroup_show](https://github.com/torvalds/linux/blob/master/kernel/cgroup/cgroup.c).
Helper выделен по ответственности; все три Python-файла меньше300 строк.

## Журнал

- 2026-10-18 — started; границы согласованы координатором, root-read defect
  подтверждён source. Независимый review этой правки ещё требуется.
- 2026-10-18 — implemented; добавлены synthetic proc/mount/controller fixtures:
  nested/root/subtree, escapes, traversal/symlink, no-match/ambiguous, unavailable,
  malformed values, migration during snapshot.
- 2026-10-18 — preliminary unit run FAIL3/14: ожидания теста не учитывали
  macOS `/var` → `/private/var` resolve. Fixture root нормализован; это ошибка
  ожидания нового набора, production и helper не менялись по этому результату.
- 2026-10-18T18:49:34+03:00 — tested; targeted14 PASS0.028с; Mac smoke2/10/10
  PASS на исходном binary042. Source frozen, передача на независимое review.

## Проверки и передача

Выполнены только targeted stdlib tests и Mac smoke2/10/10 на уже сохранённом
проверенном binary042; `git diff --check` PASS. Команды из корня:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/infra -p 'test_catalog_load_cgroup.py' -v
PYTHONDONTWRITEBYTECODE=1 python3 tests/acceptance/catalog_load.py \
  --binary .local/evidence/catalog-acceptance-003/agent-board \
  --sessions 2 --tasks 10 --requests-per-session 10 \
  --output .local/evidence/load-cgroup-047/load-smoke.json
```

14 stdlib unit cases PASS0.028с. Mac smoke:20 measured requests (по10 на
сессию), по4 каждого вида; p95 1.595ms, sampled peak RSS22.578MiB,
populated start13.399ms. macOS27.0.1/arm64,8 logical CPUs; cgroup status
`unavailable / FileNotFoundError`, effective limits `unavailable` — ожидаемая
доступность метода на этой ОС, не доказательство Linux/controller runtime.

Evidence: ignored `.local/evidence/load-cgroup-047/`:
`unit-preliminary.log` (честный первоначальный FAIL), `unit.log`,
`load-smoke.log`, `load-smoke.json`, `source-verification.log`, `freeze.sha256`.
Frozen HEAD `2dfd86bf1627775a8fdd87c650d9a3ada4cae17f`; SHA256:

- load harness `c28d7f556b7e305ddbab6fe9b70bf3f8914726f95420506eb3b0546860bdc608`;
- helper `f3c8253f188b95596e791628dbe637625797859e4440370959eb2c0d4e9b1f27`;
- unit tests `ca393d72254c9ad5678a0fd5cfa55d5abfa562c80d5013f2f76049c6bb37c106`;
- unit log `411e03836af249bd747c62e1d65c00d9a65ae18e043f7dcc06a27e838df19fbe`;
- smoke JSON `3fc2a0189c2d10617ef81e3412d46f326a97fefa5b0a075fbcc3bd3fd93d3483`;
- smoke log `6a624e9440b4f2bb352b45a9d472f7391b0f47a540a1cdd4058b2e33f120a217`;
- binary042 `1ddab9cf59d94ecbf8a32751e6834a025927a7e20fbb919acc52455d592ab107`.

Полный `make check` не повторялся по заданию. Linux/VPS
load20/2000, отдельные systemctl unit/PID limits и независимое review — следующие
шаги координатора; этот автор не выдаёт себе independent APPROVE.
