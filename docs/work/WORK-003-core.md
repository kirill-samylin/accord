# WORK-003 — Ядро и SQLite

- Роль: разработчик; запуск core-2026-10-18.
- Статус: выполнено; передано на независимое ревью. Родитель: TASK-001-pilot.
- Вход: AGENTS.md, исходное ТЗ, контракт WORK-001 (ожидается).
- Запись: internal/core, internal/store, go.mod/go.sum, эта WORK.
- Tandem выключен пользователем; remote/push не выполняются.
- Результат: transactional SQLite domain и unit/integration проверки.
- Модель именованной Claude-роли недоступна, используется модель текущей среды.

## Журнал

- 2026-10-18: прочитаны процесс, профиль разработчика и правила параллельной работы. Начата согласованная реализация; публичный контракт согласуется с архитектором.

## Результат

- Go domain API в internal/core и SQLite store в internal/store.
- Отдельные таблицы tasks/sessions/role_generations/events/idempotency/tokens;
  задача и сессия хранят JSON на строку. SQLite write-lock берётся до чтения.
- Все 11 board-операций, локальные create/auth/revoke токенов, TTL/fencing,
  singleton architect, один захват на сессию, evidence/provenance, эпики,
  current SHA+generation gates, исторические обязательства reviewer/tester,
  циклы и unlink с аудитом, идемпотентность, безопасный cursor.
- API согласован с WORK-001. Владение go.mod/go.sum передано WORK-004 после
  добавления modernc.org/sqlite v1.60.1 и MCP SDK v1.8.0.
- Первая regression lost-response выявила SQLite Scan string→json.RawMessage;
  исправлено сканированием string перед декодированием, последующие прогоны PASS.
- Root выявил разрешение source-requester блокировать чужой ready consumer;
  исправлено ограничением consumer requester/coordinator, добавлен regression
  с отдельным token principal. Live consumer требует owner+generation.

## Фактические проверки

- PASS: targeted `go test ./internal/store ./internal/core`.
- PASS: итоговый `go test -race ./internal/store ./internal/core`:
  store 2.736s, core без отдельного test-файла (контракты проверяются store).
- Проверены: два SQLite-подключения/конкурентный захват, точная граница TTL,
  просрочка без sweep, stale generation, restart, read/failed/replay без renewal,
  singleton, атомарный heartbeat, повторы/конфликт ключей/отсутствие дублей,
  пагинация с фильтром и новыми событиями, отрицательные review/tester,
  stale SHA и same-SHA reopen, provenance, эпик с новым циклом review,
  cycles/unlink audit, цепочка developer→architect→resumed developer,
  запрет чужого consumer, token expiry/revocation/session binding.
- NOT-RUN в этой WORK: process/E2E, реальные Claude Code/Codex,
  две физические машины, все ОС, performance/security production acceptance.
- Независимое ревью выполняет отдельный reviewer; самопроверка не approve.

## Границы и передача

Пилот загружает task/session rows при операции; события/идемпотентные ответы
читаются отдельно и история событий не переписывается. Масштабирование,
нагрузочные цели V1 и автоматическое удаление истории не подтверждены.
DB schema version=1, WAL; поддержка миграции из неопубликованной временной
whole-board схемы не заявлена. Все текущие Go-файлы <=300 строк.
Следующий исполнитель: WORK-005 reviewer, затем tester после approve.
После данного freeze изменения только по отдельному возврату.

## SHA-256 проверенной core/store версии

```text
8ffe8ba1ba6898ae6b3e2934014f04065e7590ee906de3b213d0f47d261baed6  internal/core/types.go
a8ee5a490f93d9219a0f3b304a1b2de04c94e41a00e16313be4ff7c30f8b8b86  internal/store/attempts.go
31e991211961509490208f2abbe4a20a46631df3ebdf1fa491d44f9e546859a3  internal/store/auth.go
1339af28bdb7ee196aafc2435d8385c077303a8c424f6bebc6d25b3033b52147  internal/store/dependencies.go
c6811d2995f58376acdfec6da29de1d70fe3b6973f6acc7ea5753752e16fbc8d  internal/store/events.go
1af2c93771d4c2a22b4fe2a763a366b48800cd4628c728cb348d4ed6834ad45e  internal/store/events_test.go
0410e12d548766bfbfd34152184706c5fed81b7e26c91c4d926e4b086280121b  internal/store/evidence.go
cf4e09bd4089e9fa46906b336c3624c8e66d1878d39e445a62e80b0e1ce7713f  internal/store/helpers_test.go
673ca0177999f3f416607ba5b667cca0f50de77e01499f074a2f76b183a03b99  internal/store/idempotency_test.go
303811e0b51d5a7fec1d8d12c4ab7857daf71cb75034de5d9627adb24dce0f1b  internal/store/leases_test.go
ae4938eb08957469a5a37c2685144a547618f9a46e25c852947ef67c7e8b2554  internal/store/mutation.go
7f0a16ea505dc274bceb67775c016fb791307ca8f41e6ab2385405d0feb2f937  internal/store/persistence.go
3ce94f5dc415040248e044558b83ece5d42363c14c89187fb797175fbea644c2  internal/store/reviews_test.go
87c29d4493bafcd3fe70a5d91f7b845dc4a643bc93a091ec265d2cf98772d360  internal/store/sessions.go
97f4dc1ca3d25e42c124385e223534cfc1c5f97db3a39d30eb965053e502a173  internal/store/store.go
27582bd443bbc48278bd02489493451dda51d4516de17a3641fe107c1a6d8962  internal/store/tasks.go
76a098adbb31f43f73d4625dfd42bc73910d3f1b5af20c19140f5c53d6771006  internal/store/transitions.go
f1bf15239fd8bb6474b131e6852ef87459a28e97ea28115360379da518923ea0  internal/store/workflow_test.go
```
