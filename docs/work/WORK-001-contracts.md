# WORK-001 — контракты требований 1.1 и архитектура

- Родитель: [TASK-001](../TASK-001-pilot.md).
- Роль: архитектор с уточнением требований (профили architect/analyst переданы
  путями из соседнего проекта; именованные Claude-модели здесь не закреплены).
- Исполнитель: отдельный субагент `contracts` текущего запуска Codex.
- Статус: выполнено; этап: передан контракт для реализации и независимого ревью.
- Граница записи: только requirements.md, architecture.md, api.md, roadmap.md,
  собственная WORK. Исходное ТЗ не изменяется; чужие WORK/TASK/код не редактируются.
- Источник полномочий: прямое поручение пользователя 18.10.2026 переработать
  ТЗ, начать отдельный сервис в Project; remote позднее, Tandem не использовать.

## Результат

Подготовлены требования 1.1 с полным сохранением V1 и явной границей пилота;
архитектурные решения Go/SQLite и transport; JSON/Store API; дорожная карта.
Исправлены fencing/expiry без sweep, client heartbeat, session binding,
positive review/test gates и SHA freshness, idempotency, pagination cursor,
coordinator handoff, MCP tool-list bootstrap, webhook unknown completeness,
canonical root/area alias и обязательный checkpoint координатора.

Go/SQLite — принятое техническое допущение по поручению переработать ТЗ;
MIT уже установлена владельцем remote Accord (main9537cb7, LICENSE).
Предварительный Apache-2.0 исходника заменён этим фактом. Не приписываем пользователю
отдельного согласования всех решений. UI/OAuth/webhooks/notes/import/stats
остаются V1, не объявляются функциями пилота.

## Проверки и ограничения

- Прочитаны новый AGENTS/TASK/исходник, общий процесс, architect/analyst,
  parallel-work и WORK инструкции; прямое указание про Tandem имеет приоритет.
- Сверены official MCP SDK v1.8.0, protocol/tools/authorization и SQLite
  transaction/WAL, GitHub failed webhooks. Ссылки в architecture.
- Локальная проверка размера документов: каждый менее 300 строк.
- Контракт сверяется с internal/core/types.go и транспортным исполнителем;
  это анализ и проектирование, не независимый approve реализации.
- Unit/integration/E2E, Claude Code/Codex и две машины этим агентом NOT-RUN.
  Reviewer/tester выполняются отдельными агентами после готовности реализации.

## Журнал

| Время (UTC) | Событие | Доказательство |
| --- | --- | --- |
| 2026-10-18 09:41:32 | Зафиксирован первый документальный результат и передан API исполнителям | requirements.md, api.md, architecture.md; сообщения root/core/transport |
| 2026-10-18 09:43:14 | Согласованы core/transport уточнения: bare DTO replay, target generation, историческая обязанность проверки, unlink с аудитом, CLI token revoke | Сообщения координатора и исполнителей; requirements/API/architecture обновлены |
| 2026-10-18 09:43:55 | Сверен переданный пользователем remote и MIT; документы поправлены, локальные ссылки/размер PASS | HEAD9537cb7, origin=kirill-samylin/accord, LICENSE MIT; исходный файл совпадает SHA256 с Downloads |

Время начала до первой записи не восстанавливается задним числом; журнал
отражает фактически считанное время, а не предполагаемую длительность.

## Передача

Core и transport используют [API](../api.md) и проверяют [AC01–12](../requirements.md).
Независимый reviewer проверяет конкретную собранную версию отдельно. Координатор
принимает результат WORK после сверки реализации с контрактом; автор
документов не выдаёт независимое подтверждение реализации.
