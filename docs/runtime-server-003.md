# Фактическая серверная приёмка Accord

18.10.2026, WORK-029/035/036. **HTTP/эксплуатация и remote MCP PASS** после
исправления Host policy. Настоящий Codex прошёл10вызовов: [отчёт](codex-003.md).
Полный остаток V1 остаётся в TASK-003.

## Проверенная поставка

- Текущая Git revision: `dc9c74cd692e287173def9fb5743dbf18ffdef21`.
- Linux/amd64 binary SHA256:
  `301b4e7e330b69b6d0d3ff304ac717335dde1aaea63888f7cfc2ff300b240c0d`.
- Использован immutable `.local/releases/<revision>`; SOURCE и все
  SHA256SUMS проверены после защищённого переноса на VPS.
- Реальная установка reviewed `install.py`, публикация reviewed `publish.py`.
- Ubuntu24.04/systemd255; root-owned `/opt/accord/releases/<revision>`,
  ссылка `current`; отдельный системный пользователь accord без shell.
- Runtime loopback `127.0.0.1:7892`, частный Tailscale HTTPS8443, Funnel выключен.
  Частный адрес и credentials в публичный отчёт не включены.

## Фактические проверки

| Проверка | Результат |
| --- | --- |
| После установки local readiness | 200, PASS |
| Systemd active/running, автозапуск enabled | PASS, NRestarts=0 |
| User=accord, NoNewPrivileges, ProtectSystem=strict, ProtectHome, PrivateTmp/Devices | PASS |
| MemoryMax256MiB, CPUQuota50% | PASS, конфигурация проверена |
| Root-owned binary/admin; accord не может писать binary или `/opt/accord` | PASS |
| State/backups0700, DB/WAL/SHM0600, владелец accord | PASS |
| HTTP слушает только127.0.0.1:7892 | PASS |
| Mac → private HTTPS, обычная проверка TLS | PASS, readiness200 |
| Первоначальный99e722c: Codex → remote MCP | FAIL, SDK Host rejection; устранено WORK035 |
| Текущийdc9c74c: Codex → remote MCP через native bridge | PASS,10вызовов, cleanup+revoke проверены |
| `/metrics` без bearer / своим bearer | 401 /200, PASS |
| Present Origin на native bearer endpoint | 403, PASS |
| Отдельный ephemeral credential: действующий → revoke → повторный доступ | 200 →401, PASS |
| Hourly persistent timer enabled/active | PASS, реальный trigger18:00:19MSK |
| Timer next fire и jitter | 19:00:31MSK, до120сек; снимок18.10 |
| Online snapshot штатным CLI, checksum и SQLite integrity | PASS, schema1, integrity=ok |
| Проверка snapshot reviewed restore.py `--check-only` | PASS |
| Изолированный Linux restore/reset drill | PASS, детали ниже |
| Журнал Accord содержит bearer headers | Нет; PASS ограниченной проверки |

При первой проверке VmRSS процесса14412KiB; это idle-снимок,
не целевой benchmark под20сессиями/2000задачами. MemoryCurrent cgroup
не приравнивается к RSS. Измеренные лимиты не являются доказательством p95.

В `/var/backups/accord` наблюдались три завершённые пары на момент проверки;
контрольная копия SHA256
`cdca77c45d28ca2f5126fdc3ee4ef5dcffc617ee0222bc304e8c830c18589243`.
Дальнейшие копии и их хеши закономерно меняются.

## Изолированное восстановление

На том же Linux запущен отдельный synthetic процесс штатного binary в отдельной
DB/каталоге и временном systemd unit; порт не публиковался в Tailscale.
Исполнялись реальные reviewed функции `restore.py` с переназначением только
fixture-каталогов и имён lifecycle units. Pre-restore backup также использовал
реальные функции backup.py с fixture-путями. Это не буквальный запуск hardcoded
production restore command, который затронул бы общий сервис.

- Созданы2synthetic задачи,2сессии, действующий lease generation1;
  проверен idempotent replay claim.
- Штатный CLI создал online snapshot; после snapshot credential отозван.
- Restore восстановил ровно2tasks/2sessions/5idempotency rows/cursor6;
  pre-restore state сохранён отдельным каталогом.
- Durable restore marker оставлен; настоящий systemd отказал в запуске fixture.
- Offline maintenance reset:1token revoked,2sessions expired,1lease released.
- После reset прежний bearer даёт401, прежний SID с новым bearer даёт401.
- Новый сеанс захватывает ту же задачу с generation2.
- Snapshot SHA256:
  `d6de1325715ac8886f83d4ed6580d1bda52c28b46a37808806ff36f35f82937a`.
- Fixture listener/helper/units/timer удалены; его credentials отозваны,
  plaintext удалён. Private synthetic DB сохранена как локальное VPS evidence.

Рабочая DB Accord не заменялась и не откатывалась во время drill.

Первоначальная установка и restore drill выполнены на99e722c (binary SHA
`da3fcda1528543a2af3dad74709789fa3167c558460cc3b125cdd3c3ec5da30a`).
Обновление доdc9c74c прошло reviewed installer: pre-upgrade snapshot,
private root0600 server.env с явным external DNS:port, readiness и MCP200.
Схема осталась1; это не проверка будущей миграции C/schema2. Состояние всех
Serve routes и metadata Tandem до/после обновления полностью совпало.

## Сохранность Tandem

До установки и после всех проверок совпали container ID, image ID, StartedAt,
RestartCount, OOMKilled и volume mounts прежнего Tandem. Healthy и health HTTP200;
HTTPS443 с Mac также200 с обычной TLS-проверкой. Всё Tailscale Serve состояние,
кроме добавленного собственного HTTPS8443, совпало. Funnel остался выключен.
Команды доски, старта/остановки Tandem или замены его DB не выполнялись.

## Клиент и оставшиеся ограничения

Рабочий токен выпущен отдельно для operator-a/mac-main на30дней,
защищённо доставлен на Mac в `.local/accord/mac.token`0600, каталог0700;
`.local/accord/client.json` содержит только origin/путь/token ID и метаданные.
Серверная plaintext-копия после доставки удалена. Секрет не выводился в чат,
аргументы или отчёты. Для actual Codex smoke отдельно выдан synthetic token.
Первоначальный отказ SDK Host protection исправлен явной exact-host policy
WORK035; независимое036 APPROVE и последующий реальный10-call smoke PASS.
Synthetic token после retest отозван; повторный запрос получил401, локальная
plaintext-копия удалена. Рабочий credential Mac сохранён отдельно.

- Второй физический ПК/его ОС: **NOT-RUN**, пользователь проверит позже.
- Реальный Claude Code: **NOT-RUN**.
- Полный reboot VPS/аварийное отключение питания: **NOT-RUN**;
  enabled units и persistent guards проверены без остановки Tandem.
- Первоначальная99e722c не имела предыдущей установленной версии; сейчас она
  сохранена как предыдущий executable. Живой rollback не запускался.
  Fault regressions относятся к acceptance-003; на рабочей DB
  destructive upgrade/restore не выполнялись.
- Offsite backup и внешнее уведомление о failed backup не настроены;
  серверные копии не защищают от потери всего VPS.
- Полная V1, UI/OAuth и нагрузочные критерии не закрываются этой приёмкой.

Private evidence: `.local/evidence/runtime-server-003/`:
`before.json`, `after-install.json`, `after-final.json`, `transfer-verify.log`,
`install.log`, `publish.log`, `runtime.json`, `mac-connection.json`,
`restore-drill.json`, `final-check.json`, `proxy-update.json`,
`proxy-runtime-security.json`. Credentials в evidence не включены.
