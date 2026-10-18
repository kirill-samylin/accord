# Code review TASK-003

Итоговый вердикт: **APPROVE** для implementation срезов A/B и native
hardening TASK-003. Проверена незакоммиченная дельта к966dda6 после freeze
WORK021/022/023/025. Блокирующих замечаний к этой версии не осталось.
Это implementation approve перед отдельным process/E2E и live acceptance;
он не объявляет ещё не выполненное развёртывание или полную V1 готовыми.

Независимый reviewer026 прочитал изменения к966dda6: bridge/session ownership,
fresh heartbeat и исторический replay, bounded request ledger, irreversible
quarantine, EOF; общий HTTPS/bearer/privatefile policy; atomic SQLite snapshots
с private stage/fsync/no-overwrite, token/audit/recovery transactions; native
Origin/headers и fixed-size admission/metrics. Блокирующих application дефектов
первым проходом не найдено. Дополнительно проверены TTL safety horizons,
запрет второго скрытого describe, отсутствие renewal/reacquire от replay,
неизменность generation после quarantine и независимость фонового pulse
от модели. Каталог и request ledger ограничены; исходники используемых
StreamableClientTransport/IOTransport API сверены с закреплённым SDK1.8.0.

Независимый race-прогон пяти затронутых Go packages PASS, результаты —
[WORK-026](work/WORK-026-review.md). Это unit/integration, не реальные
Codex/Claude, физические два ПК или Linux/systemd acceptance.

Финальные infra025 повторно прочитаны: install/rollback имеют единый
activation lifecycle; backup/start/readiness failure возвращают прежний
executable только при неизменной schema1, не накатывая DB. Неизвестная/
изменившаяся схема и неудачный возврат оставляют остановленный процесс
с durable inspection guard. Отдельный restore guard fsync-нут вместе
с каталогом до stop/замены DB. Оба запрета проверяет systemd unit.
Первичные SEC003-01/02 по прочитанному исправлению закрыты для code review;
профильное security заключение и live SEC01–12 остаются отдельными воротами.

## REV003-01 — повтор общей логики активации executable

На промежуточной версии `infra/accord/install.py` и `infra/accord/rollback.py` повторяли
`switch()` и `healthy()`: атомарное переключение managed current и один и
тот же loopback readiness polling без proxyenv. Изменение безопасного timeout,
URL или публикации ссылки может разойтись между install и rollback. Профиль
reviewer требует устранить новый семантически одинаковый дубль до approve.
Исправлено WORK025: общий implementation в recovery.py; оба caller передают
BASE и используют общий readiness. Независимо повторены17 mock/failure-
injection tests — PASS. Возврат REV003-01 закрыт. Все проверенные Go/Python/
shell source-файлы ≤300 строк; новых обязательных дублей не обнаружено.

## Фактические проверки

- Независимый `go test -race -count=1 -timeout 90s` пяти application packages:
  bridge13.093s/client1.487s/store6.515s/transport2.773s/cmd1.698s — PASS.
- Дополнительные три целевые regression проверки: network timeout →
  quarantine/no mutation recovery; atomic rollback при отказе recovery audit;
  chunked response overflow — PASS,3.146s/1.557s/1.497s соответственно.
-17 infra failure-injection tests повторены независимо — PASS,0.039s.
  Они используют mock subprocess/private synthetic SQLite, не systemd/VPS.
- `git diff --check` — PASS. Критические README/requirements/API/architecture,
  service/security contracts/deployment прочитаны; в них bearer не назван
  OAuth, stdio bridge не обещает запретить локальные edits, physical2PC явно
  отложен. TASK003 прочитан как изменяемый progress checkpoint.

Новые process/E2E пишет отдельный tester после этого approve, затем они
независимо проверяются. Windows ACL/runtime, удалённый реальный Codex/Claude,
physical2PC, Linux systemd/backup/restore и ресурсная нагрузка не подменяются
этим code review. Infra mock tests сейчас в ignored evidence; durable
regression suite рекомендован tester-этапу для воспроизведения после clone.

## Точная версия

Manifest86 (все cmd/internal Go source+unit tests, infra scripts/units,
go.mod/sum/Makefile, AGENTS/README и критические контрактные документы)
сохранён в `.local/evidence/review-003/manifest.sha256`:
SHA256 `586e59d8e4a25d5deccea8660d5bf7ab9143d5c62c066da46c6f3624eb1166c6`.
TASK003 и сами review/status docs не включены как immutable source.
Изменение application/infra/контрактов требует recheck соответствующей дельты.

Targeted log SHA256:
`be6a229c4324376995e9d648bed5ee82ab318633d4281d38a8ac7462e11d4fdb`.
Infra log SHA256:
`ca026d92f85ea1fc3d7e4cd6e81571a1a4b44e99ffc47e5959f27842ccf54d8d`.
