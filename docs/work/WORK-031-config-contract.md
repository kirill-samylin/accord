# WORK-031 — конкретный контракт config/workflow

- Родитель: [TASK-003](../TASK-003-working-service.md).
- Роль: архитектор, агент configcontract003; статус: выполнено, ожидает accepted.
- Запись: только эта WORK и [config-contract](../config-contract.md).
- Вход: AGENTS, requirements1.1, service-contract C/D, original§5–10,
  core DTO, store transaction/session/queue/evidence/gates/backup, bridge state.
- Общий процесс, parallel-work, design-first и профиль architect соседнего
  проекта прочитаны. Именованные модели Claude здесь не закреплены;
  унаследованная модель, без изменения настроек.
- Tandem/живой VPS не используются; A/B поставка не заблокирована документом.

## Результат

Версии YAML/config/DB разделены; C владеет schema2, D — schema3 последовательно.
Определены atomic миграции/restore, YAML+JSON Schema, материализация defaults,
compat bootstrap и явный Davinci profile, local-admin CAS apply/history,
неизменяемый per-transaction config snapshot без гонки s.cfg.

Даны роли/permissions/scopes/human eligibility, безопасные custom role и
evidence map/gate predicates, draft/update/epic owner/handoff, areas/aliases,
duplicate_of и deterministic similarity, единый DAG/reconcile, notes/stats
зависимости и карта непересекающихся областей следующей реализации.

Явно сохранён конфликт исходника§10.2 с corrected REQ06/12: one-task/cycle off
неподдерживаемы; singleton_role off при singleton конфигурации отклоняется.
Warn/off дополнительной evidence/independence не отключают safety floor.
PR required появляется в явном Davinci profile, не ломает мигрированные задачи.
GitHub/notifications секции не могут успешно применяться без реализации E.
Это ограничения контракта, не заявление full V1 или runtime PASS.

Скрытая точка hardcoded roles передана координатору: bridge/state.go:30;
заменяется проверкой exact requested/server role, а не новым локальным списком.

## Проверки и передача

- PASS: read-only trace исходников и исходника/актуальных требований.
- PASS: embedded JSON синтаксически разбирается; local $ref targets разрешаются.
- PASS: git diff --check для owned files.
- Документ ~326 строк: JSON Schema оставлена рядом с её семантическим
  валидатором/transaction contract, чтобы избежать двух расходящихся версий.
- N/A: runtime/E2E, код и зависимости не изменялись.
- NOT-RUN: новая config реализация, её independent review и live apply.

18.10.2026 started → contract подготовлен → передан координатору для назначения
C/D исполнителей. Независимый review контракта/реализации остаётся следующим
этапом; общая TASK/Git/соседний проект не изменены.
