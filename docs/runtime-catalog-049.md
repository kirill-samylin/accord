# Фактическое C обновление Accord и изолированная нагрузка

18.10.2026, WORK-049. Живое обновление C/schema2 и авторизованные проверки
нового каталога **PASS**. Изолированный опыт20/2000/20 завершён: наблюдённый
latency target **FAIL**, RSS/startup thresholds **PASS**. Полная V1 этим отчётом
не закрывается; для задержки требуется отдельная performance работа.

## Поставка и миграция

- Установлен только C: `1e9025cd6e3122f0adf6165e09845202736d004a`.
- Linux/amd64 binary SHA256:
  `91ac8396cda97ee0577be6074eb9523f8ab8ce8beb7579cc52ec9684116ff1a2`.
- Immutable release построен координатором из чистого committed дерева;
  WORK049 независимо проверила SOURCE и все13 SHA256SUMS локально и на VPS.
  Все108 production source files совпали с manifest QA042
  `92e8c848648a22278e506673c4e8bc2208cffc1b202bb8eced9c57933d29903b`.
- Reviewed installer заменил `dc9c74cd692e287173def9fb5743dbf18ffdef21`/schema1
  на C/schema2. D/config runtime и E delivery runtime не включены.
- Existing exact external-host env сохранён побайтно; приватный root0600.
  Private HTTPS8443 readiness200 с обычной проверкой TLS с Mac.
- Installer остановил только Accord и создал schema1 snapshot перед сменой
  executable. Затем новый executable создал отдельную schema1 migration copy
  до DDL schema2. Обе копии имеют SHA256
  `57402cef67d43101ea1c34e244b15f10e6c108a297d2ee8721a5de77c0038a9b`;
  checksum matches, SQLite integrity=ok, файлы0600 в каталогах0700.
- Первый post-install snapshot — schema2, integrity=ok, SHA256
  `6df77bbf0def6be4147ad1b163f6a3ec2f47cebd45dc7b91fe523ad0be9bce95`.
  Рабочая БД schema2/integrity=ok; ровно одна migration copy на момент проверки.
- Automatic DB rollback не выполнялся. При ошибке после изменения схемы
  reviewed installer должен оставлять сервис остановленным с inspection guard;
  такой отказ на живой БД не инъецировался. Старый binary не запускался на schema2.

## Живые проверки

| Проверка | Результат |
| --- | --- |
| Systemd active/running, enabled, NRestarts=0 | PASS |
| User=accord, NoNewPrivileges, strict filesystem, private tmp/devices/home | PASS |
| Live service MemoryMax256MiB / CPUQuota50% | Конфигурация подтверждена; нагрузка запускалась отдельно |
| Loopback7892 only; root-owned binary/admin, accord не может писать /opt/accord | PASS |
| State/backups0700, DB/WAL/SHM0600, корректные владельцы | PASS |
| Hourly persistent backup timer enabled/active; backup service success | PASS |
| Missing bearer / present Origin / foreign Host с spoofed forwarded Host | 401 /403 /403 |
| HTTPS MCP initialize и tools/list | PASS,12 tools, protocol2025-11-25 |
| Авторизованный scoped task list | PASS, только собственный synthetic epic |
| Note create/list/provenance в собственном epic | PASS |
| Авторизованный stats(owner) | PASS, только структура ответа; чужие данные не сохранялись |
| Собственная note soft-delete, task/epic cancelled | PASS |
| Отдельный synthetic token revoke → повторный запрос | 401, PASS; plaintext удалён |
| Существующие E-1/T-2 | cancelled до/после; не изменялись тестовыми операциями |
| Прежние token rows и working Mac credential/client metadata | Сохранены; working bearer /metrics200 |
| Bearer headers в Accord journal | Не найдены ограниченной проверкой |

MCP smoke — прямой protocol consumer через private HTTPS, не новый actual
Codex/Claude Code smoke. Note удаляется мягко; synthetic cancelled records и
audit остаются штатной историей. Чужие задачи не захватывались/не изменялись.

До/после обновления совпали весь Serve JSON и selected Tandem metadata:
container/image ID, StartedAt, RestartCount, OOMKilled, mounts, healthy status.
Tandem local health200 и private HTTPS443 с Mac200. Funnel выключен; команды
доски и запуска/остановки Tandem не вызывались. Ключ и exact known_hosts только
читались; StrictHostKeyChecking=yes сохранялся.

## Метод изолированной нагрузки

Reviewed `catalog_load.py`/`catalog_cgroup.py` WORK047/APPROVE048 перенесены
побайтно; SHA256 соответственно
`c28d7f556b7e305ddbab6fe9b70bf3f8914726f95420506eb3b0546860bdc608` и
`f3c8253f188b95596e791628dbe637625797859e4440370959eb2c0d4e9b1f27`.
Собственный transient `accord-load-049.service`, uid accord, NoNewPrivileges,
CPUQuota100%, MemoryMax512MiB, RuntimeMaxSec600; root-owned helper files,
fixture/output directory0700, output0600. Генератор создал свои temporary DB,
tokens и loopback ephemeral server из того же immutable binary. Existing live
DB/API не использовались нагрузкой. Генератор и child делят лимиты unit.

20 sessions,2000 tasks плюс synthetic epic; после HTTP seed — populated
restart. Затем400 measured requests,20 на сессию, до20 concurrent: get,
tasks(limit20),events(limit20),stats(owner),heartbeat поровну. Seed исключён из
latency. Server RSS sampled50ms, client RSS исключён. Внешний monitor связывает
actual child PID/membership с unit MainPID/ControlGroup и leaf limits;
effective ancestor limits остаются unavailable. Exit0 сам по себе не зачёт
порогов: значения `thresholds_observed` оцениваются отдельно.

## Фактическая нагрузка

Все400 measured requests завершились HTTP200 без изменения размеров сценария.
Seed2000 tasks занял237.626с; измеряемая фаза91.800с. Unit завершился штатно:
Result=success/ExecMainStatus=0. Это завершение опыта, а не performance PASS.

| Показатель | Наблюдение | Порог | Результат |
| --- | --- | --- | --- |
| Общий HTTP p95 | 10025.109мс | ≤50мс | **FAIL**, примерно200.5× выше |
| Общий HTTP p50 /p99 /max | 3463.879 /12798.825 /17163.349мс | — | Измерено |
| Sampled peak server RSS | 35.723MiB | ≤128MiB | PASS наблюдённого порога |
| Populated startup | 28.268мс | ≤1000мс | PASS наблюдённого порога |
| Empty startup | 16.805мс | — | Измерено |

| Операция | Количество | p95, мс |
| --- | --- | --- |
| get | 80 | 10410.056 |
| tasks(limit20) | 80 | 8745.219 |
| events(limit20) | 80 | 10371.267 |
| stats(owner) | 80 | 10157.894 |
| heartbeat | 80 | 9301.890 |

Host — Linux6.8.0-142/Ubuntu glibc2.39, x86_64,1 logical CPU,
MemTotal979160KiB, совместно с Tandem. Transient main Python PID87884;
измеряемый server child PID91493/PPid87884 независимо наблюдался в
`/system.slice/accord-load-049.service`. Тот же PID91493 сохранён harness
после measured requests; membership/leaf values совпали с независимым снимком.
Unit CPUQuotaPerSecUSec=1s, MemoryMax=536870912; leaf cpu.max=`100000 100000`,
memory.max=`536870912`. User=accord/NoNewPrivileges=yes подтверждены runtime.
Cgroup MemoryPeak64974848bytes≈61.965MiB включает generator/cache и не
приравнивается к RSS35.723MiB. Unit CPUUsage199.130с относится ко всему опыту.

Эти значения доказывают заданные unit/leaf limits и identity child, но
`effective_limits.status=unavailable` сохранён: ancestor/cpuset/hidden quotas
не исследованы, выделенного CPU нет. Результат нельзя назвать подтверждённым
PASS окружения1CPU/512MiB либо сертификацией исходного ТЗ.

Длительное SSH-соединение отдельного monitor разорвалось до выдачи его
in-memory JSON. Systemd load продолжился без перезапуска. Последующие короткие
снимки сохранены сразу; они доказывают identity/limits до завершения и
Tandem200, cumulative memory.events oom/oom_kill/oom_group_kill=0 в доступных
снимках. Непрерывный health timeline не заявляется. После завершения и cleanup
повторена полная сверка Tandem/Serve/TLS443 и живого Accord.

Собственный transient unit, `/opt/accord-load-049` helpers и temporary
fixture DB/tokens удалены после фиксации evidence. Private
`/var/lib/accord-load-049/result.json` сохранён0600 в каталоге0700; локальная
копия также0600. Production service и его resource limits не изменялись
нагрузкой; источник не оптимизировался ради результата.

## Ограничения и доказательства

- Shared VPS с существующим Tandem: конкуренция и монитор влияют на результат.
- Leaf/unit values не доказывают отсутствие hidden ancestor quotas/cpuset.
- Loopback HTTP без TLS; годовой объём200000 events не моделировался.
- RSS sampling может пропустить кратковременные пики; это не cgroup memory.
- Один опыт не сертифицирует остальные workloads/другие машины.
- Второй физический ПК, actual Claude Code, новый actual Codex C smoke,
  reboot/power-loss, destructive live rollback/restore — NOT-RUN.
- OAuth/UI, D/E runtime и полный V1 не закрыты; offsite backup/alerting
  этим шагом не добавлялись. Визуальная проверка N/A, UI не менялся.

Private evidence: ignored `.local/evidence/runtime-catalog-049/`:
`release-verification.json`, `source-verification.json`, `before.json`,
`state-before.json`, `transfer-verify.log`, `install.log`, `after-install.json`,
`state-after-install.json`, `migration.json`, `smoke.json`, `client-before.json`,
`final-check.json`, `state-final.json`, `after-final.json`, `load-method.json`,
`load-start.log`, `load-result.json`, `load-summary.json`, `load-unit-monitor.json`,
`load-snapshot-*.json`, `load-progress-*.json`, `load-monitor-interruption.json`,
`load-result-permissions.json`, `load-journal.log`, `cleanup.json`,
`evidence.sha256`. Длительный monitor JSON в summary явно обозначен как
восстановленный из отдельных сохранённых snapshots.
Remote/local `load-result.json` побайтно совпадают, SHA256
`f1419a5b97d9f3b8ec6f8f588eb357c1b7cc4b5f8e90fbca1e90aeeb670126b5`.
Manifest35 evidence files SHA256
`8aea1009faa4acb5f7f2e0f9e91bb6cac6ec40d7676609994b0cbb6e8e8543bb`.
Private origin/hostnames, secrets и реальные пользовательские payloads в
отчёт не включены. Исполнитель WORK049 не редактировал shared TASK, source и
старое evidence; commit/push не выполнял. Параллельные чужие изменения сохранены.
