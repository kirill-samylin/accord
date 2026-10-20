# TASK-010 — постоянные профили агентов

20.10.2026. Ветка `codex/agent-profiles`, старт clean checkpoint
`867f905d301a508df61f018f6ecda1d17f81539f`. Прямое поручение продолжить
после TASK009 реализацию профилей, затем в новых чатах координацию и запуск.
Статус: IN PROGRESS. Контракт — [agent-profiles.md](agent-profiles.md).

## Последовательность и владельцы

1. Root — контракт, общая TASK, operations/update, интеграция и поставка.
2. WORK140 — независимая архитектура; до реализации disposition координатора.
3. Отдельные backend/frontend авторы с непересекающимися файлами и WORK.
4. Независимый reviewer — source, author tests и ops; обязательные замечания
   исправляются до source APPROVE.
5. Отдельный tester после APPROVE — новые process/E2E; reviewer проверяет тесты.
6. Synthetic Linux update/restore/recovery, полный production backup,
   безопасная поставка и HTTPS-проверка, итоговый clean checkpoint.

## Границы и приёмка

Project-local storage/CRUD/archive/versioned instructions+settings, явная
привязка сессии к immutable snapshot. Не identity, presence или execution.
Существующие проекты/данные/ACL/epochs/cookie/leases/SHA gates сохраняются.
Текущая поставка TASK009 и full backup проверяются до изменения VPS.
Tandem и соседний davinci-gui- не используются. Credentials/runtime только
ignored .local/VPS. Следующий чат создаётся после фактического закрытия этапа.

До выполнения проверки не считаются PASS. Physical second PC, provider workers,
координация и управляемый запуск — NOT-RUN/следующие этапы.

## Архитектурное решение

WORK140: APPROVE. Приняты strict role match и ID/version pair в describe,
immutable session snapshot, actor session ID без credential ID в DTO, quota128
профилей/1024ревизий, мягкий архив, MCP12tools+resources и schema3. Приняты
новый capability floor/отдельный updater/полный backup и строгая сохранность
исторических migration backups schema1/2-64hex.db с checksum. Исполнение,
произвольные commands/env/secrets/provider URL и координация отложены явно.

## Checkpoint source review

Backend141, Next142 и operations144 реализованы; независимый WORK143:
**source APPROVE** после исправлений offline migration, old schema restore/admin,
atomic temporary resume, malformed query и unknown helper collision.
Авторские cmd/internal suites+vet/race PASS, web60units/typecheck/build PASS;
независимо reviewer повторил cmd/internal, web60 и infra144 PASS.
Это не итоговая приёмка: полный make test пока FAIL на двух прежних schema2
expectations в process acceptance. Отдельный tester145 после source APPROVE
адаптирует их и проверяет actual process/Chrome/migration/restore/update.
Linux synthetic, production update/HTTPS/full backup — ещё NOT-RUN.
