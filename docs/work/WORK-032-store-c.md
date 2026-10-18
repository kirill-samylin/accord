# WORK-032 — store/core срез C

Статус: реализация передана на ревью; REQUEST CHANGES pending. Исполнитель: `/root/storec003`, роль разработчика.
TASK: [TASK-003](../TASK-003-working-service.md).
Именованные модели Claude в этой среде не закреплены; используется модель
координатора. Tandem отключён по указанию пользователя.

## Объём и границы

`internal/core` additive DTO без Evidence; `internal/store` списки/заметки,
события/статистика/attempt tokens и миграция schema2; schema acceptance для
backup/restore/recovery. Transport/bridge/главная TASK чужие. Срез D и UI не входят.
Источник: service-contract C, config-contract C/D, requirements 1.1.

## Решения

- Новый authenticated `Sessions(ctx, actor, ListInput)` сохраняет прежний
  `ListSessions(ctx)` для локального администратора. Остальные методы:
  ListTasks/ListEpics/ListNotes/Note/Stats. DTO в core/catalog.go.
- Cursor привязан к виду списка и фильтрам; фиксирует numeric creation boundary.
  Это граница идентификаторов, не MVCC snapshot содержимого между запросами.
- Отсутствие tokens отличается от 0. Final total заменяет сумму delta при
  total>=delta; повтор mutation не увеличивает totals.
- Исторические timestamps не синтезируются; старые source events без typed data
  отмечаются incomplete. Текущее состояние не переписывает атрибуцию истории.
- Schema1 получает проверенный private backup до atomic schema2 DDL. Более
  новая схема отвергается до DDL. Offline admin не выполняет миграцию.
- До configurable grants D curators: coordinator; role notes только собственной
  роли. Source доступен своей роли/заказчику/coordinator; epic note только для
  эпика source. Проверяющий должен отдельно оценить эти compatibility predicates.

## Результат и фактические проверки

Реализованы additive task/attempt metadata, idempotent token delta/final total,
returns, стабильная граница paginated lists, role/epic notes с soft deletion,
provenance/quota, describe/next summaries, event-time typed facts и stats.
Код без нового стороннего API: официальные SQLite contracts проверены:
[transactions](https://www.sqlite.org/lang_transaction.html),
[user_version](https://www.sqlite.org/pragma.html#pragma_user_version).

PASS на macOS arm64, Go1.27.2:

- `go test -count=1 ./internal/store`: 1.228s, 16 новых C unit/integration
  сценариев плюс прежние store tests. Конкурентная quota — два DB handles.
- `go test -race -count=1 ./internal/store ./internal/core`: store 8.092s;
  core без собственных test files. Это unit/integration race, не process QA.
- `go vet ./internal/store ./internal/core`; `git diff --check`.
- Python compile restore/recovery, без выполнения live service/systemd действий.
- Новые файлы до 300 строк; исключения не потребовались.

Manifest 47 source/test/infra файлов:
`.local/evidence/store-c-003/manifest.sha256`,
SHA256 `4e42b3d83a1db29980387ca676313400ff2128c3d4b36353963a7e846b3bafd5`.
Race log: `.local/evidence/store-c-003/race.log`,
SHA256 `f8da2aca4b70f0ecf498b5c12f6991e44136d82f24a58da43c1084422407929f`.

## Существенная семантика

Статистика [from,to) считает tokens_spent/attempts/missing_tokens по попыткам,
закрытым в интервале. Активные delta видны в attempts задачи и в эти итоги не
входят. Это итог попыток по периоду завершения, не расход токенов по времени
сообщения. Final total заменяет delta total. Audit содержит отдельные
`tokens_spent_delta`, `tokens_spent_total`, `correction_delta`; они не складываются
как независимые расходы. `current_queue` относится ко времени запроса.
`incomplete_data` true при неизвестных historical facts либо missing tokens;
`incomplete_events` считает события без достоверной полной исторической атрибуции.
Legacy CreatedAt=nil сохраняет неизвестные FirstClaimAt/FirstDoneAt=nil даже
через новые claim/rework. Возвраты legacy учитываются с C без выдумывания истории.

Схема 1 перед DDL получает `<db>.migrations/schema1-<random>.db` и private SHA256
sidecar; директории и файлы fsync до schema2 transaction. Сбой backup/DDL не
меняет schema1. Offline admin/backup принимает 1 и 2 без автоматической миграции;
restore отвергает неизвестную схему, rollback разрешён только при неизменной
совместимой schema fingerprint. Переход 1→2 запрещает автоматический binary
rollback по-прежнему; неизвестная/изменённая schema оставляет guard и stop.

## Изменённые файлы

- `internal/core/types.go`, новый `catalog.go` — additive transport contracts.
- `internal/store`: новые catalog/catalog_sessions/catalog_validation/pagination,
  notes/task_metadata/metric_events/stats/migrations и четыре `*_c_test.go`.
- Existing store integration: store/persistence/sessions/tasks/attempts/
  transitions/evidence/maintenance/backup; два прежних теста обновлены по
  additive schema2 metadata и audit `blocked_changed`.
- `infra/accord/restore.py`, `recovery.py`: schema1/2 acceptance.
- Эта WORK. Common TASK/transport/CLI/docs/api принадлежат координатору.

## Ограничения и передача

Независимый approve ещё не получен. Новые process/E2E — отдельный тестировщик
после approve. Полный make check NOT-RUN этим исполнителем: существующий
`tests/acceptance/backup_process_test.go` пока ожидает schema1, Python infra
fixtures используют schema2 как unknown. Они требуют согласованного обновления
тестировщиком; это известная миграция fixtures, не скрытый PASS.

Schema2 live deploy/Linux runtime, load targets, второй физический ПК и Windows
NOT-RUN. Native99 артефакт этим изменением не развёртывался. C не реализует D:
config persistence/scopes/custom roles/evidence, owner handoff, integrations/UI.
Текущие role predicates compat; их замена принадлежит D store integration.
Backups перед неудачной миграцией сохраняются для осмотра, автоматически не
удаляются. Paged pass фиксирует границу ID, изменения содержимого/фильтруемого
состояния между страницами не являются MVCC snapshot.


## Журнал

- 2026-10-18: created/assigned/started; свежий агент WORK-032, границы от координатора.

- 2026-10-18T18:14:13+03:00: completed/handed_off; source freeze и targeted PASS; независимое ревью WORK-037.

## Состояние после первой обратной связи

WORK-037 подтвердил REV037-01: `group_by=owner` ошибочно использует владельца
эпика вместо участника фактической попытки/session/token.owner. Текущую очередь
также нельзя приписывать epic owner в executor statistics. Это обязательное
исправление, source freeze не является approve. Ревьюер собирает полный пакет;
по указанию координатора исправления выполняет свежий агент следующего шага.
Исходники и manifest WORK-032 после freeze не менялись. Общий TASK остаётся
незавершённым; coordinator принимает полный пакет WORK-037 и назначает fix owner.

- 2026-10-18T18:16:15+03:00: handed_off; REV037-01 подтверждён, REQUEST CHANGES pending; root назначит свежего автора после полного review packet. Source не менялся.
