# WORK-118 — TASK-006 browser dashboard BFF и история задачи

Дата: 20.10.2026. Владелец: backend author; источник поручения — координатор
TASK-006. Область записи: `internal/web/**` и этот WORK. `internal/webui/**`,
core/store/native API, общую TASK, Git/remote/deployment автор не изменял.

## Реализовано

- `/web/data/dashboard`: browser-only GET/HEAD, cookie + действующий token и
  board session; любой query отклоняется, включая повторный/невалидный. Другие
  методы дают 405 с `Allow: GET, HEAD`. HEAD сохраняет статус и не выдаёт тело,
  в том числе при ошибке. Чтение не вызывает page/actions/Describe/Heartbeat.
- Внутренний web DTO `DashboardSnapshot`: `overview`, `agents`, `activity`.
  Это реальные ControlOverview, ControlSessionsPage и RecentEventsPage; первые
  6 активных agent-сессий и 20 свежих событий, с действующей pagination metadata.
  Human-сессии не входят в roster; общие counts берутся из authoritative overview.
- Общий `readDashboard(ctx, actor)` используется JSON и SSR overview. При любой
  ошибке возвращается только error, успешные части не публикуются частично.
- `Page.AgentsObservedAt/ActivityObservedAt` сохраняют время своих отдельных
  readers. SSR overview/list/detail/activity/task не подменяют ими общий timestamp.
- Task detail получает последние 10 событий именно своей задачи через
  существующий RecentEvents(TaskKey); ошибка чтения не маскируется пустой историей.
  Эпики продолжают открываться, но timeline для E-* отложен: текущий core reader
  принимает только T-*; контракт core не расширялся.
- `/web/data/overview` оставлен с прежним JSON DTO, guards и чтением только
  ControlOverview. Строгие role/kind conflicts сохранены; UI автор уведомлён,
  что переключение типа должно синхронизировать роль, а не менять backend semantics.

## Ограничения архитектуры

Dashboard — три последовательных авторизованных чтения с собственными
`observed_at`, не единая транзакция. Проверены `control_overview.go`,
`control_sessions.go`, `control_activity.go`, `store.go`: все readers используют
transact (write lock + загрузка board), RecentEvents может фиксировать expiry
события. Поэтому фоновые reads не продлевают аренды/сессии, но сохраняют обычную
доменную обработку expiry. Новый poll должен заменять прежний overview poll,
не добавляться к нему. API не обещает масштабируемость либо атомарный снимок;
устранение тройной транзакции потребует отдельного store/core изменения без
копирования правил effective credential/lease в BFF.

## Файлы

- `control_dashboard.go`: DTO и общий reader.
- `control_data.go`, `control_queries.go`, `http.go`: browser JSON perimeter.
- `control_pages.go`, `contracts.go`: SSR reader, отдельные timestamps, task history.
- `control_dashboard_test.go`, `control_dashboard_boundary_test.go`,
  `control_dashboard_read_test.go`, `control_task_activity_test.go`: regression tests.

Каждый новый файл менее 300 строк. Независимый review и browser/process QA
выполняются другими исполнителями после author handoff.

## Проверки

Проверяемая база: `d542a09` + текущие незакоммиченные изменения TASK-006,
Go `1.27.2 darwin/arm64`. Первоначальный запуск через системный `gofmt` дал
`command not found`; затем использован настроенный `.local/toolchains/go/bin`.

- PASS: полный `go test ./internal/web` после dashboard implementation — 2.121s.
- PASS: targeted `TestDashboard|TestAgentKind|TestControl` — 0.965s.
- PASS: targeted + `TestTaskDetailActivity` — 0.928s.
- PASS: `gofmt` затронутых Go файлов; `git diff --check -- internal/web`.
- PASS: финальный `go test ./internal/web -count=1` после отдельных SSR timestamps
  и task timeline — 2.044s.
- NOT-RUN у автора: новые process/E2E/browser tests, deployment/live, performance,
  несколько физических машин; не являются author PASS.
