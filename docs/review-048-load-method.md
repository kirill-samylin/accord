# WORK-048 — независимое повторное review load cgroup method

18.10.2026; reviewer `/root/loadreview048`, отдельный от автора047.
Профиль reviewer, общий процесс, parallel-work и локальный AGENTS прочитаны.
Именованные Claude-модели здесь не закреплены; модель унаследована, как у
автора. Tandem OFF по прямому указанию пользователя.

## Вердикт и проверенная версия

**APPROVE узкой дельты WORK-047. REV045-01 закрыт.** Блокирующих замечаний
и новых P3 в проверенной области нет. Метод пригоден для следующего отдельного
изолированного Linux/VPS опыта20 sessions/2000 tasks под systemd с явно заданными
CPUQuota100% и MemoryMax512MiB и отдельным привязанным к запуску unit/PID evidence.
Это approve корректности метода и безопасности сценария, **не performance PASS**
и не подтверждение фактических ресурсов будущего VPS.

Входной HEAD по freeze047: `2dfd86bf1627775a8fdd87c650d9a3ada4cae17f`.
Побайтно сверены все5 записей `.local/evidence/load-cgroup-047/freeze.sha256`:

| Файл | SHA256 |
| --- | --- |
| `tests/acceptance/catalog_load.py` | `c28d7f556b7e305ddbab6fe9b70bf3f8914726f95420506eb3b0546860bdc608` |
| `tests/acceptance/catalog_cgroup.py` | `f3c8253f188b95596e791628dbe637625797859e4440370959eb2c0d4e9b1f27` |
| `tests/infra/test_catalog_load_cgroup.py` | `ca393d72254c9ad5678a0fd5cfa55d5abfa562c80d5013f2f76049c6bb37c106` |
| `docs/acceptance-042-catalog.md` | `ab2221aadfe4e874a5ab535e4879f5ed6dbe40551067063c4e803372b814bdf0` |
| `docs/work/WORK-047-load-cgroup-fix.md` | `0289e910870292bdf81d4147be5dee395bdc62a9394b7e2e808ff0fc39035fb9` |

Freeze manifest SHA256:
`c599bb8fac02019f5b1e13d26ce90ae974de4497de32bc7a01782522b989601e`.
Также108/108 файлов source manifest042 совпали; manifest SHA256
`92e8c848648a22278e506673c4e8bc2208cffc1b202bb8eced9c57933d29903b`.
Production, process tests, README и прежние APPROVE041/045 не переоткрывались.

## Проверка метода

- `catalog_load.py:223` передаёт PID текущего собственного сервера после
  измеряемых requests, включая новый PID после populated restart. Чтение root
  cgroup и observer/self membership не служит fallback.
- `catalog_cgroup.py:32–68` разбирает unified membership и cgroup2 mountinfo,
  сопоставляет полные компоненты path, вычитает mount root, декодирует только
  допустимые mountinfo escapes. Raw membership не подвергается этому decoding.
  Неоднозначные/no-match mounts и traversal дают unavailable. Strict resolve
  не позволяет leaf directory уйти за mountpoint через symlink.
- `catalog_cgroup.py:75–123` сохраняет controller values только после проверки
  формата. Missing/permission/malformed values получают свои availability
  statuses; при доступности одного файла — partial. Повторное чтение membership
  отбрасывает values при обнаруженной миграции. Это snapshot, не мониторинг
  постоянства группы; документация не обещает большего.
- Всегда сохранён `effective_limits.status=unavailable`, включая root membership,
  leaf max и доступные leaf values. Ancestor quotas, hidden namespace ancestors,
  cpuset и конкуренция не превращаются в доказательство effective budget.
  `thresholds_observed` относится к измеренной latency/RSS/startup и не содержит
  фиктивного resource-environment PASS.

Метод сопоставлен с [cgroup-v2 kernel documentation](https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html#processes),
[mountinfo fields](https://man7.org/linux/man-pages/man5/proc_pid_mountinfo.5.html)
и [cgroup namespace semantics](https://man7.org/linux/man-pages/man7/cgroup_namespaces.7.html)
18.10.2026: membership читается относительно cgroup namespace наблюдателя;
mount root и mountpoint — разные поля; ограничения предков остаются значимыми.
Неподдержанный namespace path с `..` консервативно unavailable.

Safety проверен по окружающему harness: URL существующего сервиса не принимается,
только owned child с loopback ephemeral port, private temporary SQLite/tokens и
синтетические данные; cleanup завершает свой child. Cgroup helper только читает
файлы. Root-запуск на отдельном VPS не расширяет адресат операций harness.
Генератор в том же unit расходует общий CPU/memory budget, а server RSS исключает
клиент; это честно указано в acceptance042. Переиспользование/декомпозиция проверены:
helper выделен по отдельной ответственности, новых смысловых дублей нет;
load265/helper123/unit190 строк, UI N/A.

## Независимые проверки

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/infra -p 'test_catalog_load_cgroup.py' -v
```

**PASS14**,0.029с. Synthetic cases проверяют nested/root/subtree, escapes,
traversal/symlink, ambiguity/no-match, malformed/unavailable и migration.
Дополнительно независимо воспроизведены3 проверки в private temporary fixture:

1. Membership в точности равен escaped subtree mount root с пробелом/обратной
   косой чертой: leaf равен mountpoint, available; effective unavailable.
2. Два разных корректных covering mounts (`/` и `/slice`): unavailable с
   `No unique matching cgroup2 mount`, values отсутствуют.
3. Restrictive parent10000/100000 +32MiB, leaf max/max: возвращаются только leaf
   max, effective unavailable, никакого вывода об unlimited effective budget.

Все3 — **PASS**. Это synthetic filesystem/proc fixtures на macOS, не Linux
kernel runtime. Smoke047 не повторялся: сохранённые unit/smoke logs и JSON
сверены с WORK047 hashes. Smoke047 —2 sessions/10 tasks/по10 requests,
20 measured requests, cgroup unavailable/FileNotFoundError на macOS; его
наблюдённые thresholds не подтверждают1CPU/512MiB.

Evidence048: ignored `.local/evidence/review-048/unit.log`, `probes.json`,
`verification.json`. SHA256 unit log
`9025182c93443d2721969c49c6107cb0681804dba24f101f7807bfdb49bfa24c`;
probes JSON `e4800b7fc80e644bb8759578299f2ac3213c423c15409b7e319faa3165633978`.

## Передача и ограничения

Следующий исполнитель — координатор/DevOps. Для зачётного целевого опыта
сохранить binary hash, unit configuration/runtime properties и actual child PID
с его membership из того же запуска; подтвердить1CPU/512MiB отдельно, сверив
leaf и доступные ancestor/host условия. При unavailable/partial metadata не
объявлять ресурсные условия PASS без отдельного достаточного evidence.
Root/systemd не означает отсутствия конкурирующих процессов или hidden limits.

Linux/VPS20/2000, systemd runtime, target performance, deployment/live, TLS,
годовой event volume, второй физический ПК и полный V1 здесь **NOT-RUN**.
Go/full suite не повторялись по границам узкого review; их прежний зачёт042
не подменяет target load. Source/tests, Git и соседний проект не изменялись.
