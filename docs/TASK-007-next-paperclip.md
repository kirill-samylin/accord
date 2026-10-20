# TASK-007 — полноценный Next.js клиент по Paperclip

Дата: 20.10.2026. База: `55071c8`, ветка `codex/next-web`.
Статус: **реализация**.

Источник: пользователь повторно указал, что нужен аналогичный Paperclip дизайн
и возможности, и спросил о Next.js. Предыдущий TASK-006 был локальным улучшением
старого SSR-клиента; он не закрывает этот более широкий запрос.
Next.js + TypeScript выбран координатором как техническое решение по уточнению,
а не как утверждение, будто сам Paperclip использует Next.js.

## Продуктовый результат

Самостоятельный React-клиент со структурой и взаимодействиями Paperclip:
компактное рабочее пространство, единый переключатель проекта, сгруппированная
навигация, диалоги поиска и создания задач, плотные списки/доска, карточки и
панели подробностей, вкладки, агенты перед метриками на dashboard, светлая и
тёмная темы, мобильная навигация. Основа — выбранный **Streamlined/current-default**
вариант, не смесь с историческим `.production` интерфейсом.

Все существующие реальные возможности Accord должны работать в новом клиенте:
обзор, задачи/эпики, агенты/сессии, вопросы человеку, дизайн/версии/скачивание,
история решений, активность, статистика, проект и настройки; подробности задач
и агентов, поиск/фильтры/пагинация. Создание, claim/answer/release, явный heartbeat,
вход/выход/продолжение и upload выполняются настоящим Go backend. Успех после
неизвестного сетевого исхода не придумывается, черновики не стираются.

Добавляется реальное редактирование заголовка и описания через существующий
`Store.Update`: собственные незахваченные задачи/эпики либо собственная живая
human-аренда с generation. `can_update` и capability выдаёт сервер; чужое
владение не обходится. Состояние/роль/приоритет этот native Update не меняет,
поэтому такие свойства не изображаются редактируемыми. Пустое описание при
update разрешено действующим native контрактом, лимит 16 КиБ сохраняется.

Матрица остальных возможностей Paperclip остаётся [WORK-100](work/WORK-100-paperclip-analysis.md).
Ранее выбранное PC-01 — подключение и контроль уже запущенных агентов — действует.
Постоянные профили, несколько изолированных проектов, execution, бюджеты,
skills/routines/integrations требуют своих backend-контрактов. Их нельзя
подменить работающими на вид формами. Полная parity не заявляется по одному UI.

## Архитектура и контракты

- Next.js **16.4.2**, React **19.3.1**, TypeScript; App Router и Node runtime.
  Go остаётся единственным владельцем авторизации, аренд и бизнес-переходов.
- UI — `/app`, Go — прежние `/web/**`; фиксированный server-configured loopback
  rewrite, без произвольного upstream URL/токена из браузера.
- `Accept: application/json` на существующих browser GET/POST даёт JSON-проекцию
  тех же Page builders/handlers. Native REST/MCP и обычный SSR не меняют протокол.
- JSON GET: `{ok:true,page:{...}}`. JSON POST: `{ok:true,redirect:"/web/..."}`;
  heartbeat содержит `session`/`next_action_id`. Ошибки сохраняют исходный HTTP
  status, безопасные `error`, `draft`, `field_errors`, `actions` и признак
  `retry_same_action`. Детальный DTO — [WORK124](work/WORK-124-next-browser-bridge.md).
- Cookie/Origin/CSRF и durable action IDs сохраняются. Browser bearer/localStorage
  токена нет. Переход `/web` → `/app` разрешён только для известных локальных путей.
- Next получает собственную script nonce CSP; matcher только `/app/**`.
  Старый Go CSP не ослабляется. Проверка исходников Next 16.4.2 показала, что
  external rewrites клонируют тело даже без совпадения Proxy matcher: явно
  настроены `proxyClientMaxBodySize=51MiB` и `proxyTimeout=130000ms`, чтобы
  исходный Go лимит файла 50 МиБ, multipart overhead и deadline 120 секунд
  оставались достижимы. Это буферизующий Node rewrite, не zero-copy streaming.
  Для будущей установки `/web/**` рекомендуется направлять reverse proxy
  прямо в Go; текущий локальный путь проверяется и через Next.
- `.local/next` внутри `web` — build output, `web/node_modules` — игнорируемый
  стандартный результат установки. БД, токены, логи и evidence остаются в корневой
  `.local`. Это явно описанное исключение для дерева зависимостей, не секреты в Git.

Детальная [архитектура WORK122](work/WORK-122-next-architecture.md) и
[дизайн WORK123](work/WORK-123-paperclip-next-design.md) прочитаны до реализации.
Версии новых пакетов проверены по публичному npm; старые Go-зависимости не
обновляются. Применяется skill `vercel-react-best-practices`.

## Владение

| WORK | Роль | Область записи |
| --- | --- | --- |
| 122 | Архитектор | Своя WORK, source только чтение |
| 123 | Независимый designer/analyst | Своя WORK и `.local/task007-reference` |
| 124 | Backend developer | `internal/web/**`, своя WORK |
| 125 | Next frontend developer | `web/**` кроме mutations, своя WORK |
| 126 | Forms developer | `web/src/features/mutations/**`, своя WORK |
| root | Coordinator | Общая TASK, docs/индексы, root Make/ignore/Git |

Все используют наследованную модель. Независимое code/security и design review
назначаются после реализации; новые process/E2E пишет отдельный tester только
после APPROVE, затем новые тесты проходят отдельное review. Tandem OFF.

## Обязательная приёмка

1. Go unit/integration + старые SSR regression, новые frontend typecheck/unit/build.
2. Независимый review кода и авторских тестов конкретной версии.
3. Реальные Go + production Next build + Chrome: login, все экраны, создание,
   human workflow, upload/download, native agreement, pagination/search, ошибки,
   исходы retry, drafts/expiry, csrf/Origin/revoke и CSP/hydration.
4. Независимый visual осмотр относительно одного зафиксированного Paperclip
   варианта: desktop/tablet/mobile, обе темы, пустые/ошибочные/длинные состояния.
5. Production/физический второй ПК не входят в эту локальную поставку; NOT-RUN
   не превращается в PASS. До обязательной приёмки этап остаётся открытым.

## Результат

Будет заполнен после реализации и независимой приёмки. Ни новый runtime,
ни наличие Next package.json сами по себе не являются готовым продуктом.
