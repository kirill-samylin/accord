# TASK-013 — управление локальными и удалёнными агентами

22.10.2026. Ветка `codex/managed-agents`, исходный HEAD
`d9f9eba6fdecb131e36f49907d421a2df43cccbd`. Прежний рабочий VPS release
`64ed870b749289b545d29cd2ec7813c6266de54d` этой записью не обновляется.

## Поручение и результат

Пользователь: «Давай теперь реализуем управление агентами из Accord: запускать,
останавливать, продолжать работу и передавать уточнения, в том числе агентам
на компьютере второго человека».

Это следующий прямо порученный этап после закрытых TASK011/012 и подключения
GUI (WORK158–161). Реализуется управление собственными процессами локального
worker, который владелец добровольно устанавливает на каждом ПК. Первая связка:
Mac с Codex CLI и Windows с Claude Code. Модели, подписки/provider credentials
остаются локальными. Физического доступа к Windows в текущей среде нет.

Контракт — [managed-agents.md](managed-agents.md). Profile, account identity,
board session/lease, process/run и provider conversation остаются разными
понятиями. Предыдущие инструкции, запрещавшие реализацию managed launch до
closure координации, теперь не блокируют прямо порученный новый этап.

## Объём

- Отдельное разрешение управления компьютером поверх project membership:
  локальный enrollment, controller allowlist, разрешённые workspace/provider.
  Редактор задач не получает произвольное исполнение на чужом ПК.
- Durable start/stop/resume/clarify commands; состояния очереди, получения и
  исполнения отражают подтверждённые факты, offline не считается stopped.
- Outbound-only worker, private journal и ownership lock, bounded supervision
  своего дерева процессов, exact native session continuation и fail-closed
  recovery без blind relaunch после неоднозначного crash.
- Понятная панель машин/запусков/уточнений с ACL, историей команд и ошибками,
  без фальшивых online/process/delivery состояний.
- Schema5/capability floor и совместимость capable upgrade, полного backup и
  restore всех проектов без старого binary, DB rollback или ослабления ACL.
- Инструкции запуска worker на Mac/Windows и воспроизводимые проверки.

Пользователь выбрал: **оба управляют агентами обоих ПК**. На каждом компьютере
владелец локально подтверждает allowlist двух project-local controllers. До
локального enrollment чужой компьютер не подключается автоматически.
Implementation использует явный список controllers, а не имена людей.

## Явные границы

Команды относятся к процессам, созданным worker. Присоединение уже запущенного
терминала/IDE не имитируется. Один writable run на разрешённый checkout;
другой параллельный checkout разрешает локальный владелец. Live task lease
по-прежнему выдаёт native `next`; запуск процесса сам её не создаёт.

Уточнение первой реализации доставляется на следующей границе CLI turn и
показывается в очереди до подтверждения подачи. Остановить текущую работу
можно отдельной командой. Продолжение использует точный provider session ID;
неудача восстановления не заменяется новым чатом. Offline remote stop требует
связи; локальный worker останавливает процессы при потере control heartbeat.

Нет произвольной remote shell/argv/path/env из браузера, серверного clone,
provider key storage, обхода native approvals или выдачи task done/review approve
по одному exit0. Автономное планирование/автогенерация продуктовых задач не
вводятся этим поручением. Tandem и соседний GUI-код не изменяются.

Граница остановки на Unix — управляемый CLI и принадлежащая ему группа PGID.
`setsid`/double-fork может отделить фоновый процесс; такие службы требуют
локальной проверки владельца. Checkout mutex кооперативный, не системный запрет
всех фоновых писателей. Windows Job Object удерживает своё дерево. Arbitrary
descendant containment/отсутствие любых писателей на Unix этим этапом не
объявляется PASS; QA отдельно проверяет группу и отделившийся процесс.

## Ответственность

- Координатор/root — эта TASK, интеграция областей, фактический статус.
- WORK162 — архитектура/контракт; WORK163 — backend/schema/native/browser API.
- WORK164 — локальный worker, supervision и native provider adapters.
- WORK165 — независимый contract/code/test/evidence review.
- Панель и infra/QA получат отдельные WORK до начала изменения своих файлов.

Изменения до начала: в Accord untracked WORK158–161/GUI onboarding и чужой
`deploy-task012.command`. Они сохраняются; текущий этап не выдаёт их за новую
продуктовую реализацию и не исполняет посторонний deployment script.

## Критерии приёмки

1. Author unit/integration: ACL/epochs/replay, quotas/version conflicts,
   ordered durable commands, dispatch/ACK/crash windows, lease/deadman,
   exact session continuation, owned process tree и redaction.
2. Независимый source APPROVE реализации и авторских тестов.
3. Отдельный tester после approve: реальные процессы/HTTP/Next browser,
   просмотр состояний/ошибок, lost response/restart/offline/stop/resume/clarify;
   source review новых tests и независимая проверка evidence.
4. Schema4→5/all-project capable migration, сохранность прежних строк и ACL,
   durable guard, recovery и full restore на synthetic данных.
5. Реальный native Codex и native Windows Claude учитываются раздельно.
   Stub/SDK/cross-build не закрывает native acceptance или второй физический ПК.
6. Полная сборка/проверки и secret scan; точная проверенная версия указана.
   Source/staging не означает установленный production release.

## Фактическая приёмка

Проверенный локальный candidate v5 — `3ea1f8b16ca7bb81d6a2639f67b9f2d78fdc2f1a`,
Mac binary SHA256 `7d2864cc488d573463c69f9670100c989a21bf5915c4f8319977592d57a864ca`.
Точные private evidence paths — WORK168; независимый source/evidence review —
WORK165. Source/binary unchanged подтверждены каждым законченным suite.
Полный `make check` product source PASS на `85fbc50`; v5 меняет только запрос
закрытых сессий в QA tests. Go/race/acceptance, infra164, web88,
TypeScript/format и production Next build проверены.

| Проверка | Результат | Граница результата |
| --- | --- | --- |
| Next/Chrome, запуск/остановка/resume/уточнение, ACL, lost response | PASS5, 8 снимков | Детерминированный CLI fixture, один Mac |
| Capable all-project migration4→5 и restore/auth-reset | PASS3 | Настоящие Go/SQLite/snapshot; systemd/ingress simulated |
| Основной process suite, включая оба offline сценария | 3 × PASS16 | Три независимых private runtime, owned PGID на одном Mac |
| Настоящий Mac Codex | FAIL на MCP approval | Native conversation создан, Describe отклонён policy never; task attempts0 |
| Windows runtime, native Claude | NOT-RUN | Cross-build не заменяет запуск на Windows |
| Второй физический ПК | NOT-RUN | Два synthetic host на одном Mac не являются двумя ПК |

Codex MCP isolation, positive terminal handoff и независимость watchdog от
подготовки запуска independently approved и проверены actual fixture процессами.
Config-only probe установленного Codex подтвердил изоляцию подключений; это
не заменяет native task acceptance. Более ранний ownership unknown не повторился
в трёх полных v5 runtime; его исходная причина остаётся недоказанной. Unknown
по-прежнему сохраняет checkout intent и требует локального восстановления.

Отказ настоящего Codex сохранён без bypass. Подготовлена выключенная по умолчанию
локальная опция точного native MCP consent —
[native-agent-permissions.md](native-agent-permissions.md). Source review и
проверки этой новой опции учитываются отдельно; разрешение человека до повторного
native запуска с grant обязательно и ещё не получено.

Статус: **IN PROGRESS — итоговая приёмка и исправления по QA**. Remote
push/merge/release не выполнены; VPS остаётся TASK012. Непройденная обязательная
native/Windows проверка оставляет этап незавершённым.
