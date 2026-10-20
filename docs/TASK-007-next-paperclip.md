# TASK-007 — полноценный Next.js клиент по Paperclip

Дата: 20.10.2026. База: `55071c8`, ветка `codex/next-web`.
Статус: **локальный Next.js этап завершён — PASS / independent APPROVE**.

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
| 127 | Независимый code/security reviewer | Своя WORK, продукт только чтение |
| 128 | Независимый tester/designer | `tests/web/next*.cjs`, своя WORK, `.local/task007-qa` |
| 129/root | Автор исправления network deadline | `mutations/deadline*`, подключение `useAction`, своя WORK |
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

Application-версия **`5cc0a88f74a0040540dcb111466696314810ca8f`**, ветка
`codex/next-web`. Next build `zSw1a3XnegMZCIwt6Wlk_`.
Manifest `.local/task007-freeze/artifact7.json`, SHA256
`e19d708794c944062014a798c4b0d7e6ecda0986ce7fa359106cc5a383bf0231`.
Root и reviewer независимо сверили292/292 исходников и3/3 artifacts.

### Реализовано

- Новый Next/React/TypeScript клиент:10 основных разделов, подробности задачи,
  эпика и агента, поиск Ctrl/Cmd+K, фильтры/пагинация, список/доска и quick look.
- Paperclip Streamlined: единый project switcher, grouped sidebar на всю высоту,
  агенты перед flat KPI, плотные списки, диалоги, Inter, обе темы и mobile drawer.
- Реальные login/logout/continue, create epic/task, edit title/description,
  claim/answer/release/manual heartbeat и материалы3scopes с версиями/download.
- Единый Go domain/auth/action journal: JSON-представление browser handlers,
  native REST/MCP и прежний SSR сохранены. Нет browser bearer/localStorage token.
- Исправлены фактические browser дефекты: root rewrite/slash, mobile cascade и
  accessible names, scope defaults, потеря черновика,8s read/15s POST deadlines,
  backoff/visibility cancellation, полный русский event vocabulary, sidebar
  height, controlled dialog focus и focus готовой страницы после route selection.

### Проверки и версии доказательств

| Проверка | Результат / evidence |
| --- | --- |
| Go fmt/vet/unit/integration, SSR16, infra76, race | PASS полного `make check` на `560b10b`; весь Go source после него неизменен, сверено manifest |
| Финальный Next format/TypeScript/unit/build | PASS42/42 на `5cc0a88`; `.local/task007-artifact7-check.log` |
| Code/security и авторские тесты | Независимые APPROVE всех product delta; [WORK127](work/WORK-127-next-code-review.md) |
| Реальные Go+production Next+Chrome/native MCP | **PASS21/21**, scope `no-deadlines`, `artifact7-final-01/next-results.json` |
| 50МиБ/+1byte,31s body,120s incomplete через Next | Отдельные PASS на `eb01113`, `artifact4-remaining-01`; protected Go/API/proxy/deadline sources после этого идентичны |
| Visual и keyboard |106 PNG,10screens+details/dialogs, обе темы,1440/768/390 и320/767 affected cases; фактический осмотр [WORK128](work/WORK-128-next-qa.md) |

Финальный browser report SHA256
`20bccb1dd526451eba8d7e9a772b694d1b0ea66cc8b05aea99ce2679ab9e6974`:
findings/page errors/CSP-hydration/static errors пусты; native agreement,
112-task paging/search, draft/retry, real200/409 claim race, expiry/continue,
revoke401 и focus на новых маршрутах PASS. Root дополнительно пересчитал27/27
файлов test manifest без расхождений: `.local/task007-final-verification.json`.
Test freeze12:12 новых E2E модулей, SHA256
`cc9ffa2822e92482c7151b099ae9d37f26475ab1279635d3e338c8d85c25129d`.

Физический120s upload завершился через120015ms,31s —31024ms; readiness обоих
сервисов восстановлена, pending/reserved/staging пусты. Старый run имеет общий
FAIL из-за найденных UI/test ошибок; его успешные boundary check IDs сохраняются
самостоятельными доказательствами. Это **не утверждение нового120s прогона на
финальном binary**. Reviewer и tester сверили неизменность protected source;
финальный Next routes manifest совпадает. Новые UI delta проверены отдельно.
Все исторические FAIL/причины и immutable manifests сохранены в `.local/`.

### Границы и передача

Локальный Next UI этап; полная parity Paperclip и полная V1 не заявляются.
PC-01 остаётся контролем уже запущенных агентов. Несколько проектов/изоляция,
постоянные профили и инструкции, execution, budgets/routines/skills/integrations
остаются следующими backend этапами по матрице WORK100.

Production HTTPS/VPS, реальные внешние Codex/Claude Code и второй физический ПК —
**NOT-RUN**. Chrome viewport не равен физическому телефону. Pixel-perfect
сравнение с запущенным pinned Paperclip не выполнялось. Push/merge/release и
переключение установленного сервера не производились; VPS остаётся на TASK-005.
Все QA процессы завершены; реальные токены/БД/снимки находятся только в `.local/`.

Итоговая независимая сверка кода, новых тестов, evidence и closure документов —
**APPROVE**, WORK127. Запуск — [next-web.md](next-web.md). Дальнейших product
изменений в этом этапе нет; финальный Git checkpoint содержит только docs/tests.
