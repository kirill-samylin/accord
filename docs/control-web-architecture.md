# Архитектура веб-клиента и масштабирование Accord

> Уточнение 20.10.2026: пользователь повторно запросил аналогичный Paperclip UI
> и возможности, спросив о Next.js. Для следующей поставки выбран отдельный
> Next.js/TypeScript клиент поверх Go. Актуальное решение —
> [TASK-007](TASK-007-next-paperclip.md) и [WORK122](work/WORK-122-next-architecture.md).
> В TASK019 прежний SSR полностью удалён из кода; ниже только историческое
> решение TASK-006. Текущий клиент и JSON-only контракт — [next-web.md](next-web.md).

20.10.2026. Решение для [TASK-006](TASK-006-control-refinement.md).
Существующие [инварианты](architecture.md) и [API](api.md) сохраняются.
Аудит исходников — [WORK-116](work/WORK-116-architecture-audit.md).

## Выбранные технологии

| Слой | Сейчас | Основание / граница |
| --- | --- | --- |
| Домен и сервер | Go, `net/http`, `html/template` | Один путь правил для web/REST/MCP, экранирование текста |
| Хранение | Закреплённый `modernc.org/sqlite`, WAL | Один экземпляр проекта; транзакции и fencing |
| Интерфейс | SSR HTML, CSS tokens, небольшие JS-модули | Первый экран и формы без JS; без второго auth/router слоя |
| Визуальная система | Paperclip neutral tokens, Inter, SVG | Выбранный референс, локальные assets и лицензии |
| Взаимодействия | Нативный dialog, ссылки/формы, Fetch/AbortController | Клавиатура/focus, bounded reads, отмена запросов |
| Проверки | Go unit/integration/race, Playwright/Chrome | Реальные сервер/браузер, synthetic данные, независимые review/QA |

Новых runtime-зависимостей этот выбор не требует. React/TypeScript/Vite уместны
при переходе к сложному редактору профилей/графу/потокам исполнения; сейчас
перенос форм, drafts и сессий создаёт отдельную большую миграцию.
Добавление библиотеки должно решать конкретную измеримую задачу.

## Ответственности

`internal/core` хранит типы и правила; `internal/store` применяет их в
транзакциях; `internal/web` управляет browser auth/actions и read-моделями;
`internal/webui` отображает их и обслуживает allowlist embedded assets.
Клиент не повторяет бизнес-авторизацию и не владеет состоянием аренд.

SSR и JSON обзора читают общий dashboard reader. Каждый результат содержит
время чтения; счётчики не объявляются атомарным снимком с лентой.
Клиентские theme, navigation, commands, polling и безопасный DOM-renderer
имеют отдельные файлы. Скрипты подключаются явным allowlist; CSP
`script-src 'self'` остаётся без inline/unsafe-eval. Формы и ответы сохраняют
существующие CSRF/action-id/generation контракты.

## Подключение нескольких проектов

Целевая сущность project — отдельная область данных, доступа и конфигурации,
а не эпик. Первый безопасный путь — server-side registry изолированных boards
с собственными БД и авторизацией; браузер обращается к одному BFF по opaque
project ID. Registry проверяет членство до каждого чтения и мутации. Нельзя
принимать произвольный upstream URL/токен от browser query.

До реализации нужны контракт ID/membership, миграция существующего single-board,
project-scoped cookies/action IDs/cursors/uploads/audit и отрицательные тесты
межпроектного доступа. Универсальный токен для всех boards не вводится.
Tenant routing и registry — следующий backend этап, не скрытая настройка UI.

Постоянный agent profile отдельно от сессии: роль, описание, разрешённые
проекты и настройки подключения. Session подтверждает участие и доступ;
lease — владение задачей; execution run — отдельная будущая сущность worker
с подтверждёнными стартом/выходом/heartbeat. Модель/provider/cost не выводятся
из имени сессии. Настройка и запуск требуют execution-контракта.

## Ограничения производительности

Dashboard ограничивает возвращаемые коллекции, но это не доказательство
ограниченной стоимости store: текущие read-модели используют общий
транзакционный путь, а часть чтений загружает полный state. До большого числа
проектов/агентов нужны read-only SQL-проекции, индексы, нагрузочные измерения,
пагинация и отдельная проверка изоляции. Несколько writer-реплик поверх общей
сетевой SQLite БД не являются целевым масштабированием.

PostgreSQL рассматривается при подтверждённой потребности в общей multi-tenant
БД/конкурентных writers; миграция обязана сохранить fencing, atomic claim,
idempotency и SHA/generation gates. Redis/очередь/контейнерные workers не нужны
для текущего просмотра сессий и не устанавливаются заранее.

## Первичные источники

- [Go server project layout](https://go.dev/doc/modules/layout#server-project).
- [SQLite: appropriate uses и конкурентная запись](https://www.sqlite.org/whentouse.html).
- [Нативный dialog](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/dialog).
- [Paperclip source](https://github.com/paperclipai/paperclip/tree/5739a53f5f5e6cccbf21c7ae491ad06cb4b79b7c),
  локальная provenance/лицензия — `docs/reference/paperclip/`.
