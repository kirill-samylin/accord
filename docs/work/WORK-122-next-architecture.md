# WORK-122 — контракт миграции веб-клиента на Next.js

20.10.2026. Роль: независимый архитектор. Область записи: этот WORK и
`.local/task007-architecture/`; продуктовые файлы не изменялись.
Основание: поручение пользователя «Почему не next.js для фронта? Дизайн сделай
аналогичным их, такие же фишки нужны». Этот документ — рекомендация координатору,
а не утверждение, что приведённый ниже API уже реализован.

## Прочитанные контракты и вывод

Прочитаны `AGENTS.md`, `docs/requirements.md`, `docs/architecture.md`,
`docs/api.md`, `docs/control-web-architecture.md`, TASK-006 и реальные
`internal/web/{http,server,login,contracts,pages,control_pages,control_queries,
control_data,actions,mutations,design_http,files_upload,files_form}.go`,
`internal/core/types.go`, `internal/transport/http_host.go`.

**Рекомендация: Next.js App Router + TypeScript как основной новый клиент;
Go остаётся единственным владельцем browser auth, доменных мутаций и SQLite.**
Прежний SSR имел разумную стоимость для ограниченной панели, но запрос на
Paperclip-подобный богатый клиент теперь оправдывает отдельный frontend.
Перенос доменных правил в Node, замена SQLite и переписывание native API
для этой миграции не требуются. Существующий SSR сохраняется как рабочий
совместимый интерфейс на время проверки нового клиента.

## Маршруты и процессы

Без `basePath`. Публичный origin один; пример локальной fixture:
`http://127.0.0.1:3100` → Next, Go слушает `127.0.0.1:3101`.
Go `Options.Origin` равен **публичному** origin с портом3100.

| Путь в браузере | Владелец | Контракт |
| --- | --- | --- |
| `/app`, `/app/login`, `/app/tasks`, `/app/tasks/T-*` | Next | Новый интерфейс |
| `/app/questions`, `/app/agents`, `/app/agents/:id` | Next | Новый интерфейс |
| `/app/projects`, `/app/designs`, `/app/history` | Next | Новый интерфейс |
| `/app/activity`, `/app/stats`, `/app/settings` | Next | Новый интерфейс |
| `/web/:path*` | Go через фиксированный Next external rewrite | SSR, browser API, формы и файлы |
| `/api/v1/**`, `/mcp`, health/metrics | Go | Прежние native клиенты без изменения |

В fixture native клиенты продолжают использовать3101. Публикация native путей
на общем ingress — отдельная явная конфигурация, не catch-all rewrite.
Не переписывать `/` Go на Next до принятия нового клиента; при входе через
Next `/` можно направить на `/app`. Существующие Go ссылки остаются `/web`.
Клиент сопоставляет только разрешённые внутренние `/web` ссылки с `/app`;
download/preview URL остаются `/web/designs/...`.

Upstream — серверная настройка с фиксированным loopback адресом. Браузер не
передаёт upstream, bearer или токен проекта. Rewrite сохраняет исходный Origin
публичного сайта; Go сравнивает его с `Options.Origin`. Не подменять Origin
адресом Go. Host в upstream остаётся loopback и проходит существующий host
guard. Проверить это реальным login через rewrite, а не только mock request.
Перепроверять query, Set-Cookie rotation/deletion, Location и HEAD на wire.

## Browser JSON API

Координатор выбрал content negotiation на существующих screen routes:
`GET /web/tasks?state=ready` с `Accept:application/json`. Аналогично работают
`/web/`, `/web/overview`, `/web/questions`, `/web/agents`, `/web/agents/:validID`,
`/web/tasks/:validKey`, `/web/projects`, `/web/designs`, `/web/history`,
`/web/activity`, `/web/stats`, `/web/settings`. Произвольного path query нет.
Query проходит текущий `controlQuery` либо `designFilter` конкретного экрана.
Ошибки — JSON400/404. `Vary:Accept` у negotiated representations вместе с
no-store. HEAD — те же проверки и заголовки, пустое тело; лучше не выдавать
неиспользованные action IDs для HEAD. Download/preview не превращаются в Page
JSON из-за Accept; они сохраняют бинарное представление и защитные заголовки.

Реализация выделяет общий builder страницы из `read`/`handleDesigns`,
а SSR renderer и DTO projector используют один результат. Нельзя собирать
JSON повторным HTTP GET к SSR или парсить готовый HTML. Существующие
`/web/data/overview` и `/web/data/dashboard` не меняют wire-контракт.

Пример envelope: `{ "ok": true, "page": { ... } }`. `page` получает явные
snake_case имена, optional отсутствуют либо null, коллекции — массивы.
Это отдельный DTO, не экспорт private `browserSession`. Нужные поля:

- `view,title,notice,error,csrf,actions,can_continue,can_edit`;
- `principal:{id,owner,machine}` и текущий `session` из `core.SessionView`;
- `overview,agents,agent,activity,stats,task,tasks,epics,questions,my_questions`;
- `designs,decisions,legacy_decisions`;
- `observed_at,agents_observed_at,activity_observed_at`;
- `next_cursor,epics_next_cursor,query,agent_state,epic_key,role,state`;
- `roles,states,categories,complexities,draft,field_errors` при применимости.

Nested core DTO сохраняют существующие JSON tags. Не отдавать session cookie,
TokenID, DB hash, PendingDescribe, action payloads или исходный bearer.
CSRF/action IDs — видимые browser capabilities, их выдача нужна форме.
Все персональные ответы `Cache-Control:no-store`; JSON content type и nosniff.

Чтение browser сессии сохраняет AuthenticateID/revocation check. GET не
вызывает ensureSession/heartbeat. Истёкшая board session при живой browser
cookie возвращает shell+`can_continue:true` без нового describe. Native read
errors не маскируются успешной пустой коллекцией. Времена трёх dashboard reads
остаются отдельными; согласованный атомарный снимок не обещается.

`s.page` уже создаёт operation-bound IDs и ограничивает unused IDs2048.
Новый read не должен бесконечно polling-выдавать IDs: polling использует
dashboard, форму перечитывают явно после принятой мутации/перехода.

## Login и mutation JSON

`GET /web/login` с `Accept: application/json` выдаёт preauth cookie и JSON
`{ok:true,page:{csrf,title,...}}` тем же алгоритмом, что HTML loginPage.
`POST /web/login` использует прежний urlencoded либо multipart token-file body,
тот же preauth/CSRF/Origin/rate-limit/Authenticate/rotation; изменяется только
представление результата. Никакого хранения токена в localStorage, URL,
query cache или telemetry. После ответа очищать input и File reference.

Прежние POST URL сохраняются:

| Операция | URL и обязательные поля |
| --- | --- |
| Создание | `/web/tasks`, текущие create fields, csrf, action_id |
| Захват | `/web/tasks/:key/claim`, csrf, action_id |
| Ответ | `/web/tasks/:key/answer`, csrf, action_id, generation, answer |
| Освобождение | `/web/tasks/:key/release`, csrf, action_id, generation, reason |
| Продление владения | `/web/heartbeat`, csrf, action_id, task_key, generation |
| Продолжение сессии | `/web/session/continue`, csrf, action_id |
| Выход | `/web/logout`, csrf, action_id |
| Файл | `/web/designs`, прежний streaming multipart |

JSON выбирается явным `Accept:application/json`; body стандартных действий
остаётся `application/x-www-form-urlencoded`, 32КиБ, запрет duplicate fields.
Не вводить вторую реализацию runAction. Все ранние ошибки в ServeHTTP
(Origin, метод, parseForm, browser auth, CSRF, unknown path) тоже получают
JSON для JSON-запроса. Изменить только serialization/redirect presentation.

Успех: `{ok:true,redirect:"/web/tasks/T-1",session?,next_action_id?}`.
Статус200 вместо303 для JSON; SSR сохраняет303. Redirect только серверный
разрешённый `/web` path; браузер проверяет origin/allowlist до перехода.

Ошибка: `{ok:false,error:{code,message},draft?,field_errors?,actions?,
action_id?,retry_same_action:boolean}` с фактическим HTTP400/401/403/409/
413/429/503. Draft не содержит token/token_file/csrf. Ошибка login не отражает
введённый токен. Поле message — публичное сообщение из текущего контракта.

При известной domain/validation ошибке обновлённая форма получает новый
operation ID. При неизвестном результате503 сохраняются **тот же ID и
точное body**; кнопка повтора повторяет их, без автоматической новой мутации.
После сетевого обрыва клиент считает исход неизвестным. При live draft
пользовательский текст не заменяется polling response. Не делать optimistic
state transitions/claim; только сервер подтверждает generation и владение.

Heartbeat включается только для явно захваченного human вопроса; новый
next_action_id заменяет прежний лишь после подтверждённого результата.
Read polling не продлевает аренду. При401 polling прекращается; при409
LEASE_LOST редактор сохраняет draft, блокирует новую отправку и предлагает
явно обновить состояние. Автоматический новый describe запрещён.

## Файлы, proxy, CSP

50МиБ file +64КиБ envelope уже ограничены Go; multipart порядок сохраняется:
csrf, action_id, метаданные, единственный file последним. При ошибке клиент
сохраняет metadata и File, если браузер ещё держит его, не обещая восстановить
файл после reload. Не повышать Go лимит ради Next. Existing download/preview
идут по cookie, без bearer/native endpoint, с прежними disposition/CSP.

`proxy.ts` для nonce/headers должен match **только `/app/:path*`** и не читать
body. **Уточнение после проверки установленного Next16.4.2:** исключение `/web`
из matcher само по себе не отключает clone/buffer. В
`next/dist/server/lib/router-utils/resolve-routes.js` создаётся clonableBody;
`router-server.js` вызывает cloneBodyStream для external rewrites независимо
от matcher. `body-streams.js` по умолчанию обрезает поток после10МиБ.
Прежнее предположение о bypass buffering только matcher было неверным.

Для локального fixed rewrite нужен `experimental.proxyClientMaxBodySize`
ровно51МиБ и `proxyTimeout:130000`; серверные пределы Go50МиБ+64КиБ и120секунд
не меняются. Next может держать копию запроса в памяти до51МиБ на запрос;
этот путь нельзя называть zero-buffer/zero-copy. Официальная опция experimental;
её фактическая семантика сверена по коду той же установленной версии.
Production ingress должен направлять browser `/web/**` напрямую Go в общем
origin, обходя Node clone; текущий production не переключён.
Обязательны реальные50МиБ upload PASS,50МиБ+1 byte file413,31секунд успешный
поток и поведение около/после120секунд Go deadlines. Не подменять boundary
проверку мини-файлом. Upstream/network error не считать успехом.

Go `script-src 'self'` нельзя механически применить к Next document: hydration
использует framework inline scripts. Для `/app` — request nonce, dynamic
rendering, CSP nonce на request/response и framework scripts, `object-src none`,
`frame-ancestors none`, `form-action self`, `base-uri self`. Production без
unsafe-inline/unsafe-eval; dev исключения отдельно. Не добавлять nonce/Next
headers в ответы Go: preview sandbox и Go политики остаются их собственными.

## Runtime и библиотеки

Node runtime `next build` + `next start`/standalone. `output:export` не подходит
для этого контракта: runtime rewrites, cookies и неизвестные заранее task/agent
paths не поддерживаются статическим экспортом. Это новая стоимость запуска,
build/deploy/rollback и RSS; Go512МБ цели нельзя автоматически переносить на
Next процесс. Production deployment пока NOT-RUN.

Минимум: Next/React/TypeScript, локальный Inter; Lucide для единой иконографии,
Radix primitives для доступных dialog/menu при необходимости. TanStack Query
уместен для bounded reads, invalidation и отмены запросов; actions retry:false.
Не переносить весь Paperclip монорепозиторий, Express/Drizzle и его auth.
UI токены и лицензии Paperclip/Inter сохранить, использовать актуальную
проверенную версию Next с lockfile. Установка зависимостей этим WORK не делалась.

Зависимости `node_modules` игнорируются как воспроизводимый installation output;
DB, logs, fixtures, runtime evidence и builds должны лежать в игнорируемой
`.local/`. Для Next задать `distDir:'.local/next'` внутри frontend workspace:
не выносить distDir за его пределы. Production build script/start должны
согласованно использовать этот путь. Lockfile и package manifest входят в Git.

Client structure: `app/` только routes/layout; `features/` по workflows;
`components/ui/` primitives; `lib/api/` DTO/fetch/error/route mapping;
`lib/query/` keys; `styles/` tokens. Go web adapter отдаёт read DTO и mutation
presentation; core/store не знают о Next. Файлы стремятся к300 строкам.

## Приёмка и честная граница parity

Обязательны все10 существующих разделов, детали task/agent, login/token-file,
создание эпика/задачи, claim/answer/release/heartbeat, continue/logout,
design upload/version/preview/download, decision history, pagination/filters,
пустые/ошибочные/expired состояния и длинный текст на mobile/tablet/desktop.
Paperclip-подобный shell и компоненты должны использовать реальные данные.
Состояние «running» означает живую session/lease; оно не доказывает OS run.
Projects показывает текущий project и честную границу текущего backend.

Универсальные execution adapters, budget/cost, постоянные профили, multi-project
membership/registry, provider secrets и запуск агентов требуют своих доменных
контрактов. Их нельзя реализовать фиктивными активными controls. Совпадение
дизайна и перенос текущих workflows не равно полной Paperclip parity.

Разработчик пишет unit/integration negative auth/CSRF/origin/idempotency/
expiry tests. После независимого APPROVE отдельный tester создаёт E2E:
реальный Next+Go browser login/mutations/upload, повтор неизвестного результата,
два tabs, revoked cookie token, no read renew,50МиБ/oversize,120sec boundary,
CSP console, visual screenshots. Новый тестовый код тоже проходит review.
Проверки реализации здесь **NOT-RUN**: документ не является code review или QA.

## Первичные источники, проверены20.10.2026

- [Next rewrites](https://nextjs.org/docs/app/api-reference/config/next-config-js/rewrites):
  external URL rewrite поддерживает постепенную миграцию.
- [Static export](https://nextjs.org/docs/app/guides/static-exports):
  limitations включают cookies, rewrites и dynamic routes без static params.
- [Self-hosting](https://nextjs.org/docs/app/guides/self-hosting):
  Node self-hosting поддерживается; перед публичным Next рекомендован reverse proxy.
- [CSP](https://nextjs.org/docs/app/guides/content-security-policy):
  nonce policy требует динамического рендера и отдельной настройки Next scripts.
- [proxyClientMaxBodySize](https://nextjs.org/docs/app/api-reference/config/next-config-js/proxyClientMaxBodySize):
  экспериментальное ограничение относится к Next Proxy body buffering.
