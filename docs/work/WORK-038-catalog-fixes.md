# WORK-038 — исправления пакета review C

Родитель: TASK-003. Исполнитель: `/root/catalogfix003`, разработчик.
Статус: выполнено / передано на независимое повторное review; собственный
approve не заявляется. Зависимость WORK-037 завершена с REQUEST CHANGES.
Профиль разработчика и общий процесс прочитаны; именованные модели Claude
здесь не закреплены. Tandem отключён пользователем.

## Границы

Единая дельта REV037-01/02/03: неизменяемая атрибуция статистики фактическому
участнику, точный int64 через HTTP SDK/worker и закрытие operational blocked
интервалов на terminal state. Store/bridge и unit/integration regressions;
transport/API docs, native proxy fix, live, Git и process E2E вне области.

## Журнал

- 2026-10-18: назначено; прочитан окончательный полный пакет WORK-037 до правок.
- 2026-10-18T15:22:24Z: completed/handed_off; единая дельта трёх findings,
  targeted race/vet PASS, source freeze ниже.

## Исправления

- REV037-01: event owner — owner сохранённой server session участника события;
  при закрытии попытки источник строго её SessionID, включая expiry/recovery.
  Current queue использует только текущую lease session. Нет исполнителя —
  группа с пустым key, без attribution epic owner. Исторические события не
  переписываются; интервалы closed-attempt accounting остаются прежними.
- REV037-02: worker разбирает оригинальный JSON text сервиса, проверяет JSON
  object и восстанавливает structuredContent через UseNumber. И local typed
  validation, и downstream wire сохраняют int64; upstream SDK float64 больше
  не является источником бизнес-ответа. Локальные ошибки bridge также имеют
  JSON text, сохраняя code/message и прежнее fail-closed поведение.
- REV037-03: отдельный operational metric predicate исключает done/cancelled;
  terminal event закрывает уже открытый интервал, включая старый typed boundary
  без отдельного blocked_changed. Restore в ready начинает новый интервал,
  если dependency всё ещё не удовлетворена. Gate evaluator не менялся.

## Файлы

`internal/store/metric_events.go`, `stats.go`, `metrics_regression_test.go`,
`epic_gates_test.go`; `internal/bridge/protocol.go`, `bridge.go`,
`precision_test.go`; эта WORK. В bridge.go сохранён ранее существовавший C
resource mirroring. Чужие transport/config/native файлы не менялись.

## Проверки

PASS на macOS arm64 / Go1.27.2:

- Новые unit/integration regressions: Bob42/Alice epic, totals и unknown queue,
  persistence через второй store, release/expiry/offline recovery participant;
  blocked cancellation [from,to) clipping, ready restore и повторная cancellation.
- Настоящие HTTP SDK→worker→stdio SDK: 2^53+1 и MaxInt64, exact client JSON text,
  persisted task, replay unchanged, events delta, release и next с историей.
  Отдельно проверяется actual downstream structuredContent wire, поскольку
  consumer SDK сам десериализует any числа в float64 вне власти bridge.
- Первый race: bridge13.625s, transport2.910s, CLI1.768s PASS; store FAIL только
  из-за старого epic_gates_test, ожидавшего operational blocked_changed у done.
  Assertion обновлён на linked+epic_reopened и metric blocked=false; прежние
  gate/history/replay/atomicity assertions сохранены. Это исправленная семантика
  terminal metrics, а не подавление ошибки.
- Финальный `go test -race -count=1 ./internal/store ./internal/bridge`:
  store8.449s, bridge13.590s PASS.
- `go vet ./internal/store ./internal/bridge ./internal/transport ./cmd/agent-board`
  и `git diff --check` PASS. Все изменённые файлы менее 300 строк.

Freeze7 source/test файлов: `.local/evidence/catalog-fixes-003/manifest.sha256`,
SHA256 `375f062953a26e578e6e6bef1479bed095571fb704d1c3aaf424c9be32b7edb6`.
Первый race log SHA256
`d7a708ea1bf66fe38a6a472c188ef22d29456ddcdbd1c46288640e6265598dc9`;
финальный race log SHA256
`5479fda663f50179141ae40a1983ed85b10229fdca68eeb3b2eab4c428a0c7d7`.

Полный make check NOT-RUN: прежние process/schema fixtures ожидают обновления
тестировщиком после независимого approve. Новые process/E2E автором не писались.
Live C/schema2, второй физический ПК, Windows/Claude runtime NOT-RUN. Native
proxy fix интегрирует координатор отдельно; мой freeze не подтверждает его.
Root обновляет API owner semantics; следующий шаг — reviewer всех трёх findings
и собранной версии, затем tester. Git/live/Tandem не выполнялись.
