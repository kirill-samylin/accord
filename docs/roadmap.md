# Дорожная карта agent-board

Дата: 18.10.2026. Требования [1.1](requirements.md) заменяют конфликтующие
правила исходника; V1 не сокращена до пилота. Даты завершения и оценки в днях
не назначаются без результатов реального пилота. Статус этого документа —
план, фактическая готовность — [TASK-001](TASK-001-pilot.md).

## Этап 0A — воспроизводимое локальное ядро (текущий объём)

Результат: отдельный локальный Go-репозиторий, SQLite, токены, сессии,
эпики/задачи/связи, положительные gates, fencing/expiry, idempotency,
безопасные events, REST/MCP/CLI. Coordinator не singleton; architect singleton.
Предпосылки: контракт 1.1, доступный toolchain и official pinned MCP SDK.

Проверки: unit/integration разработчика; независимое ревью конкретной версии;
после актуального approve — новые process/E2E отдельного тестировщика и их
ревью. Два подключения к БД проверяют contention, перезапуск — durability,
fake clock — точную TTL-границу. SDK smoke отдельно от реальных MCP-клиентов.
Закрытие: AC01–12 на доступном локальном стенде с явным списком NOT-RUN.
Остаток 0B не позволяет назвать всю первую версию или весь пилот готовыми.

## Этап 0B — реальные инструменты и две машины

Результат: Claude Code и Codex подключаются к удалённому защищённому пилоту
с локальным bearer способом там, где клиенты его поддерживают. Отдельно
проверяется способ клиента поддерживать heartbeat во время молчаливой работы;
инструмент heartbeat сам по себе фонового агента не создаёт.

Сценарии: конкурентный next; developer → architect → новый developer;
потеря связи, истечение без sweep, возврат stale клиента, потерянный ответ и
retry, restart сервера. В журнале фиксируются версии клиентов, ОС и машин.
Недоступный клиент или вторая машина — NOT-RUN и открытая зависимость.
Нельзя заменять два физических ПК двумя процессами на одном компьютере.

## Этап 1 — полное ядро и конфигурация V1

REQ V1-02/03/04: версионируемые YAML/JSON Schema/config validate/apply,
custom roles и evidence schemas, reopen матрица, mode block|warn|off,
epic ownership/handoff, human eligibility, области/aliases, duplicate warnings,
duplicate_of, links эпиков, notes/limits, comments, tokens/returns/stats.
Все динамические изменения сохраняют audit и не ломают живые leases.

До реализации: точный config migration/compatibility contract, семантика
отмены обязательного gate, retention idempotency/events, token role scopes.
Singleton planner остаётся отдельным открытым решением, не включается молча.
Приёмка: сценарии 9.1–9.8 исходника в границах доступных интерфейсов,
configuration invalidation/reload tests, alias-parent-child overlap tests.

## Этап 2 — авторизация, интеграции и данные

REQ V1-01/05/06: GitHub login allowlist/org/team, OAuth MCP с metadata,
audience/PKCE, revocation/session lifecycle, токены UI/CLI; signed incoming
GitHub webhooks, delivery dedup, completeness unknown/manual redelivery;
signed outgoing notifications с retries; escalation; JSONL import/export;
SQLite consistent backup/restore; мигратор Давинчи как отдельный проектный script.

Зависимости: модель угроз и профильный security review, проверенная матрица
клиентов. Не объявлять OAuth реализованным выдачей произвольного bearer token.
Без GitHub API не обещать автоматическую сверку пропущенных webhooks.
Перенос реальных карточек не выполняется без отдельного указания пользователя.

## Этап 3 — веб для человека

REQ V1-07: «Моя очередь», «Состояние», human claim/answer/reopen, токены,
read-only конфигурация и история, аудит, статистика. Минимальный SSR допустим,
конкретный UI определяется согласованными макетами. Сервисные контракты
готовятся до UI; полноценная продуктовая визуальная часть после макетов.
Независимые code/security/design review и browser/E2E на актуальной версии.

## Этап 4 — выпуск V1

REQ V1-08/09: целевые ОС/архитектуры, binary/container/systemd,
TLS deployment/Origin/proxy hardening, metrics/rate limits/secret scan,
EN README + ключевые RU документы, template Давинчи, checkpoint/handoff,
нагрузочная методика и backup/restore drill. MIT установлена владельцем в
подключённом remote Accord; push и публичный выпуск требуют отдельного поручения.

Приёмка полной V1: все 12 критериев §18 исходника с исправленной семантикой
аренд, независимости и положительных проверок; один эпик завершён на двух
машинах с реальными инструментами, актуальным review и tester evidence;
измерены performance/RSS/start цели на описанном стенде.

## Ближайшие решения и ограничения

- Пользователь передал `https://github.com/kirill-samylin/accord.git`; origin
  подключён координатором, базовая main9537cb7 содержит MIT/LICENSE владельца.
  TASK-002 разрешает публикацию проверенной рабочей ветки; merge/main и выпуск
  отдельно не поручены. Tandem отключён по прямому указанию.
- Go/SQLite и клиентский heartbeat — принятые технические допущения для
  0A; не переоткрывать их без новых фактов, но не выдавать за отдельное согласование.
- Planner singleton и eligibility для production требуют решения
  до соответствующих этапов, но не блокируют безопасное локальное ядро.
- Доказательства команд в evidence заявлены агентом; сервис не исполняет
  команды и не проверяет их содержание. Exact-SHA gate не является CI runner.
- Координатор клиента сохраняет checkpoint; events/my_tasks его не заменяют.
