# Срез C — независимое code/security review

Дата: 18.10.2026. WORK-037; авторы WORK-032 и WORK-034.
Вердикт: **REQUEST CHANGES**. Нужны три исправления ниже и повторное
независимое review их конкретной версии перед новой process-приёмкой.

## Версия и границы

База `99e722c44f83b1808120bba1fabb04cf2f68a5b5` плюс C WIP.
Source manifest: `.local/evidence/review-037/manifest.sha256`, 85 файлов,
SHA256 `1a2d21487d5a75e3bb2adb70d0057918f334935d53152d0e2acfeb5231961370`.
Проверены additive core, store C/миграция2/backup, REST/MCP/bridge resources,
CLI read consumers, соответствующие unit/integration tests, recovery/restore.
Manifest автора032 совпадает полностью; в manifest034 изменён только docs/api.md:
actual-версия включена в reviewer manifest. Конфигурационный parser D и новые
зависимости в go.mod не входят. Отдельный native proxy fix ещё не интегрирован.

Проверяющий отдельный от обоих авторов. Именованные модели Claude здесь не
закреплены; используется та же доступная модель. Реализация и тесты репозитория
не изменялись. Дополнительные воспроизведения — временный Go overlay в `.local`.

## Блокирующие замечания

### REV037-01 — P1: расходы и результаты приписаны владельцу эпика

Места: `internal/store/metric_events.go:35`, `internal/store/metric_events.go:76`,
`internal/store/stats.go:119`.

`EventData.Owner` всегда берётся из epic.Owner. Участник Bob закрывает задачу
в эпике Alice с 42 tokens: `group_by=owner` начисляет completion и 42 tokens
Alice. Тем самым отчёт §13.2 исходника «по участнику» неверен, а будущий handoff
эпика дополнительно меняет смысл атрибуции разных событий одного исполнителя.

Нужен неизменяемый source fact owner фактической попытки/session/token.owner.
Для очереди использовать реального держателя lease, если он есть; без исполнителя
явно unknown, а не владелец эпика. Для событий без выполняющей сессии допускается
отдельно определённая attribution requester, если это подтверждённый факт.
Historical events не дополнять предположениями. Totals должны сходиться с группами,
включая unknown. Семантику owner уточнить в API и добавить двухучастниковый regression.

Подтверждение: `TestReviewer037ParticipantAttribution` — FAIL,
`credited participant=alice tokens=42`. Координатор подтвердил, что owner здесь
означает участника, а не владельца эпика.

### REV037-02 — P1: допустимый int64 приводит worker в quarantine

Места: `internal/bridge/protocol.go:67`, `internal/bridge/state.go:87`.

`resultJSON` выбирает upstream `StructuredContent`, десериализованный SDK в any
с float64. Новые tokens int64 выше 2^53 теряют точность. При comment с
`tokens_spent_delta=9223372036854775807` store принимает точное значение,
но worker затем маршалит округлённый float64, не может прочитать его в core.Task
и возвращает LEASE_UNCERTAIN после успешной серверной мутации. Последующий
next с такой историей попыток также способен ломать владение.

Нужно сохранить исходное точное JSON-представление при разборе/проксировании
ответа, включая structuredContent на downstream wire; одного изменения accept
недостаточно. Использовать raw JSON text сервиса с проверкой формы либо decode
с UseNumber. Добавить regression через настоящий bridge/HTTP SDK для 2^53+1 и
MaxInt64: точное persisted и клиентское значение, без quarantine, replay unchanged.

Подтверждение: `TestReviewer037ExactInt64ThroughWorker` — FAIL,
валидный comment возвращает tool error LEASE_UNCERTAIN. Request сам передавался
typed int64 без округления; дефект находится в результате.

### REV037-03 — P2: отменённая задача продолжает накапливать blocked duration

Места: `internal/store/metric_events.go:20`, `internal/store/stats.go:46`.

Blocked source fact использует gate predicate независимо от состояния задачи.
Если заблокированную задачу отменить через 10 секунд, source остаётся blocked,
граница окончания не появляется. Через ещё 20 секунд stats показывает 30 секунд
блокировки, хотя задача 20 секунд исключена из рабочей очереди. При никогда не
завершившейся зависимости расход продолжает расти бессрочно.

Отделить queue/operational blocked interval от predicate валидности gate:
отмена должна закрывать интервал. Не ослаблять existing done-result gates;
если для них нужны отдельные интервалы, определить их явно. Regression должен
включать cancellation и [from,to) clipping до/после неё.

Подтверждение: `TestReviewer037CancelledBlockedDuration` — FAIL,
`blocked duration=30 after task cancelled at 10 seconds`.

## Проверки и безопасность

PASS: `go test -race -count=1 ./internal/core ./internal/store
./internal/transport ./internal/bridge ./cmd/agent-board`: core без тестов;
store8.423s, transport2.978s, bridge13.377s, CLI1.705s. PASS targeted vet и
`git diff --check`. Все затронутые Go-файлы укладываются в 300 строк.
Новые unit/integration tests проверяют реальные state/SQLite/SDK поведения.

Модель угроз: bearer — доступ к одной общей board; SID сам по себе не credential;
read не получает admin mint/revoke, изменяющие note/token операции требуют
live own session и idempotency. Проверены чужой SID, source access, note author/
curator, soft-delete audit, quotas в writer-транзакции, numeric high-water/filter
binding и migration backup до DDL. Обходов auth/fencing и ложного продления
lease от store reads не обнаружено. Existing revoke/session guards сохраняются.

Schema1 backup публикуется privately с checksum/fsync до atomic schema2 DDL;
future schema отвергается. Restore поддерживает1..2, schema-changing failed
activation остаётся stopped с inspection guard; автоматического DB rollback нет.

FAIL: три независимых overlay repro выше. Они не добавлены в source как
process/E2E-тесты: это reviewer probes, не заменяющие приёмку тестировщика.
Полный make check не объявляется PASS: прежний process backup fixture и Python
unknown-schema fixture требуют согласованного обновления отдельным tester.
Live C/schema2 deployment, physical second PC, Claude runtime, нагрузочная
приёмка NOT-RUN. UI/OAuth/config D/integrations не проверялись этим срезом.

Следующий исполнитель исправляет единую дельту, сохраняет regression coverage
и обновляет manifest. Затем reviewer перепроверяет все findings, combination с
native proxy fix и затронутые тесты; независимый tester запускает process QA
после актуального approve. Tandem/live/commit/push этим review не выполнялись.
