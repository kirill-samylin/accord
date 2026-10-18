# WORK-018 — контракт оставшегося объёма Accord

- Родитель: [TASK-003](../TASK-003-working-service.md).
- Роль: архитектор/аналитик, агент contract003; статус: выполнено.
- База: `966dda698da5e615b1517faec853c0b7271b2f9c`.
- Область записи: эта карточка и [service-contract](../service-contract.md).
- Профили architect/analyst, общий процесс, parallel-work/design-first,
  AGENTS/requirements/architecture/API/roadmap/исходное ТЗ прочитаны.
- Именованные модели Claude в этой среде не закреплены; используется
  унаследованная модель. Tandem отключён по прямому указанию пользователя.

## Результат

Сопоставлены все разделы исходника и V1-01–09 с текущей реализацией.
Отдельно выделены работающий bearer выпуск и полная V1 без сокращения объёма.
Исполнимые контракты: stdio bridge/automatic heartbeat/quarantine; безопасный
remote operations/backup/tokens; lists/notes/stats; config/roles/rules;
webhooks/notifications/import/export и условие начала UI.

Уточнения пользователя сохранены: tokens-first вместо отсутствующего OAuth
App, physical2PC позже. Остальные функции не отменены. Согласование простого
functionalSSR вместо макетов принадлежит пользователю; координатор уточняет.
Singleton planner не вводится. Предложения технических деталей (DTO, schema,
порядок миграций) требуют сверки исполнителем и независимого ревью среза.

## Проверки

- PASS: source/doc read-only trace по Go DTO/store/transport/CLI и ТЗ.
- PASS: diff --check и проверка размеров документальных файлов.
- N/A: runtime/E2E приложения — только контракт, код/зависимости не изменены.
- NOT-RUN: новый remote bridge/VPS/physical2PC; это задания реализации/приёмки.

## Журнал и передача

- 18.10.2026 started: проверены база/входные документы и текущие DTO/SQLite.
- 18.10.2026 handoff: координатору передан bridge/ops контракт до полного map;
  координатор подтвердил его для следующего developer WORK.
- 18.10.2026 completed: итоговый service-contract покрывает остаток V1,
  acceptance и ограничения. Следующий владелец — координатор: назначение
  независимых A/B реализаций и затем их code/security review.

Исходники, общая TASK, сервер, соседний проект и Git не изменялись.
