# WORK-014 — Read-only аудит VPS и прежнего Tandem

- Родитель: [TASK-002](../TASK-002-publication-readiness.md), запрос пользователя
  проверить ПК/проект/сервер и подготовить подключение второго ПК.
- Роль: DevOps, отдельный агент `server002`; статус: выполнено в границах аудита.
- Запись: только эта WORK и `docs/server-readiness.md`. Код, чужие документы,
  соседний проект и серверная конфигурация не изменяются. Commit/push не выполнялись.
- Входы: AGENTS, TASK-002, requirements/architecture; общий процесс,
  parallel-work и профиль DevOps из davinci-gui-; прежний Tandem runbook,
  WORK-268/274/275/276. Прямое указание пользователя отключить Tandem
  приоритетно над старым требованием board claim; MCP/доска не использовались.
- Модель профиля Claude здесь не закреплена: применяется доступная модель
  среды. Это read-only эксплуатационный аудит, не независимое ревью кода.
- Исходная версия: Accord `21cb3e0`; проверенный удалённый образ приведён в отчёте.

## Журнал и результат

- 2026-10-18T13:00:03Z — SSH-проверка с существующим приватным ключом,
  BatchMode, ConnectTimeout и StrictHostKeyChecking=yes с pinned known_hosts.
  Чтение system/Docker/Serve/listeners/backup metadata; секреты не выводились.
- 2026-10-18T13:00:23Z — уточнено фактическое Compose-имя контейнера
  `davinci-coordination-tandem-1`; проверены inspect selected fields,
  auth/TTL без вывода других env, checksum/возраст архива.
- 2026-10-18T13:00:24Z — с Mac HTTPS health200 с обычным SSL context.
  Локальный приватный конфиг имеет права0600; endpoint/capability не выводились.
- PASS: VPS доступен; Tandem healthy, loopback-only API, private HTTPS,
  Funnel=false, OOM=false/restart0; SHA256 backup совпал.
- NOT-RUN: физический второй ПК, Tandem MCP/claim, Linux runtime Accord,
  новый backup/restore, нагрузочный benchmark, развёртывание/remoteTLS Accord.
- В проверенных контейнерах/systemd/стандартных /opt путях Accord не найден.
  Последний backup прежнего Tandem —17.10 19:42 UTC (~17,3ч);
  root cron/named systemd timer для него не найден.
- Возврат координатору: [server-readiness](../server-readiness.md).
  Следующий шаг зависит от выбора общего сервиса и ОС второго ПК;
  server deploy/замена Tandem данным аудитом не разрешаются.
