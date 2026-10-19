# WORK-100 — Paperclip: возможности и контракт первого веб-этапа Accord

- Родитель: REQ-005; исполнитель `/root/paperclip_analysis100`, роль analyst.
- Статус: анализ выполнен; предлагаемый объём этапов передан координатору. Открытые продуктовые решения ниже не объявлены согласованными.
- Вход Accord: `a6501d8621abba558606b7fc26d7407fb3b2065e`, worktree `.local/web-control-099`.
- Reference: `/private/tmp/accord-paperclip-reference`, HEAD `5739a53f5f5e6cccbf21c7ae491ad06cb4b79b7c`, только чтение. [Зафиксированный источник](https://github.com/paperclipai/paperclip/tree/5739a53f5f5e6cccbf21c7ae491ad06cb4b79b7c).
- LICENSE reference: MIT, Copyright (c) 2025 Paperclip AI. При переносе существенных частей кода/стилей сохранить copyright и текст лицензии в поставляемых материалах; бренд Accord остаётся явным.
- Прочитаны профиль analyst, общий процесс, parallel-work и design-first Давинчи; самостоятельный AGENTS Accord имеет приоритет для самостоятельного сервиса. Tandem OFF, Давинчи read-only. Именованные Claude bindings недоступны, модель наследована.
- Область записи: только эта WORK и `/private/tmp/accord-control-099/analysis/`. Код, Git, сервер и общую REQ не менял.

## 1. Вывод для координатора

Paperclip — полноценная система управления агентами, заданиями, выполнениями и организациями, а не только внешний вид доски. Пользовательский запрос «все фишки» сохраняется полным roadmap; первая поставка не должна называться полным переносом Paperclip. Дизайн берётся из зафиксированного исходника, отдельный новый дизайн не создаётся.

Для быстрого первого цельного этапа предлагается реальный control UI существующего Accord: Dashboard → Agents → Tasks/detail → Activity → Projects → Settings. Все эти экраны должны читать настоящий серверный state. Текущий API уже содержит задачи, эпики, события, сессии и статистику, но web Board interface ещё не предоставляет Sessions/Stats. Для Projects/Settings на этой базе доступен один конфигурационный проект и фактические read-only правила. Это не multi-project и не редактор runtime-агентов. Разрешение на следующий этап не следует подменять этой ограниченной первой волной.

Две развилки переданы координатору рано: подключённые внешние агенты против запуска процессов самим Accord; проект как репозиторий против tenant/организации. Вопросы не блокируют read-only обзор текущих данных и сохранение существующих действий, но блокируют новые execution/multi-project мутации.

## 2. Что действительно обнаружено в Paperclip

Источники в таблице относительны HEAD reference. Наличие source/API подтверждает реализацию возможности в reference, но не её проверку в запущенном Paperclip и не доступность всем пользователям. `App.tsx:142–459,760–856` выбирает streamlined/production shell и экспериментальные gates; нельзя перемешать несовместимые варианты экранов и назвать это точной копией одного UI.

| Возможность | Фактический source/UI/API | Соответствие Accord / волна |
| --- | --- | --- |
| Organization switcher, несколько организаций, onboarding | `ui/src/context/CompanyContext.tsx`, `ui/src/pages/Companies.tsx`, `ui/src/api/companies.ts:83–172`, `App.tsx:855` | В Accord одна board с `Config.Project`; организация не равна эпик/area. Волна 2 после модели изоляции |
| Dashboard: метрики, последние задачи, activity, карточки запусков | `pages/Dashboard.tsx`, `components/ActiveAgentsPanel.tsx:1–176`, `api/dashboard.ts`, `routes/dashboard.ts:27–32` | Queue/stats/events/sessions есть в store, wire в web отсутствует. Волна 1 с реальными Accord-метриками |
| Agents: фильтры all/active/paused/error, список/оргструктура, модель, adapter, heartbeat и live run | `pages/Agents.tsx:49–108,194+`, `pages/Agents.production.tsx:255–286,705+`, `api/agents.ts:120–164` | Сессии/роли/машины есть; persistent profile, reporting line, модель и execution adapter отсутствуют. Сессии — волна 1, профили — 3 |
| Настройка агента: permissions, instructions bundle с версиями/rollback, credentials readiness, models/test environment, pause/resume/terminate | `api/agents.ts:150–232,232–280`, `routes/agents.ts:4033–4184,4740–5131,5453–5717` | В Accord есть конфигурация ролей/правил, не запись агента с lifecycle. Не создавать формы с фиктивным сохранением. Волна 3 |
| Реальное выполнение, wake/invoke/cancel, runtime requests, logs/events/provider trace/liveness/recovery | `api/heartbeats.ts`, `routes/agents.ts:6029–6036,6675–7473`, `packages/adapters/codex-local/src/server/index.ts` | Lease/attempt не process run. Нужны отдельные adapter и worker/environment модель. Волна 4 |
| Tasks, личные представления/Inbox, поиск, фильтры, read/unread, labels, checkout/release | `pages/Issues.tsx:1–177`, `api/issues.ts:173–385`, `components/IssuesList.tsx`, `pages/Inbox.tsx` | Фильтры role/state/epic/owner, durable cursor, atomic claim есть. Волна 1 сохраняет текущие фильтры/действия; дополнительные представления — 5 |
| Task detail: thread, комментарии, interactions, планы, документы с revisions/lock, attachments/work products, approvals, privacy | `pages/IssueDetail.tsx`, `api/issues.ts:403–660` | Accord имеет Description/Comments/Attempts/Links/Evidence/Review; human answer/history и design attachments уже есть. Отображение и существующие actions — 1; расширенные collaboration/artifacts — 5 |
| Projects: создание, статус, lead agent, visibility/members, stars, repositories, workspace/runtime policy, budget | `pages/Projects.tsx`, `pages/ProjectDetail.tsx:374–433,493–642,849+`, `api/projects.ts`, `packages/db/src/schema/projects.ts:8–38`, `routes/projects.ts:316–361,540–639` | Project registry/ID/ACL отсутствуют. Текущий проект — 1; настоящие изолированные проекты и workspace refs — 2; runtime — 4 |
| Goals и связь organization → goal → project | `pages/Goals.tsx`, `api/goals.ts`, `packages/db/src/schema/projects.ts:13` | Epic не следует переименовывать в goal без контракта. Волна 5 |
| Approvals: approve/reject/revision/resubmit, комментарии и linked tasks | `pages/Approvals.tsx`, `pages/ApprovalDetail.tsx`, `api/approvals.ts:5–23` | Accord independent review/test evidence/gates имеют другой смысл. Human questions сохраняются; общий approval workflow — 5 |
| Activity/audit, actor/entity filters, run/cost/budget/timeline секции | `pages/audit/CompanyActivity.tsx`, `api/activity.ts:69–85`, `routes/activity.ts:224–363`, `App.tsx:370–396` | Cursor events и immutable decision history есть. Настоящий event feed — 1; runtime timeline/audit расширения — 4/5 |
| Costs/budgets: accounting quality, agent/project/provider breakdown, quotas/subscriptions, hard stops | `pages/Costs.tsx`, `api/costs.ts:29–57`, `api/budgets.ts`, `pages/ProjectDetail.tsx:778–781` | Accord tokens_spent/unknown coverage — не деньги. Реальная существующая статистика — 1; billing и budgets — 6 |
| Routines: manual run, triggers, версии, rollback, run history | `pages/Routines.tsx`, `api/routines.ts:37–83`, `routes/routines.ts:149–200,363–656` | Durable task queue не scheduler. Волна 6 после adapter |
| Settings: организация, branding, governance; members/invites/secrets, environments, adapter/plugin management | `pages/CompanySettings.tsx:72–147,231–398`, `App.tsx:155–260` | Safe read-only current project/roles/TTL/rules — 1; editable validated config и scoped access — 2/3; environments — 4 |
| Skills, instructions studio/sources, connectors/MCP gateways, per-tool permissions, tool audit | `App.tsx:194–235,261–269`, `pages/SkillStudio.tsx`, `pages/apps/`, `routes/tool-gateway.ts`, `routes/company-skills.ts` | Текущий Accord MCP — API координации, а не gateway внешних инструментов. Волна 7 |
| Portability: company export/import, preview, async jobs, activation checklist | `api/companies.ts:120–172`, `pages/CompanyExport.tsx`, `pages/CompanyImport.tsx`, `pages/Dashboard.tsx:53–111` | Существующие Accord backup/import/export сохраняются; перенос multi-project/profiles потребует собственной версии. Волна 7 |
| Дополнительные gated surfaces: agent chat, conference room, cases, status cards, isolated execution workspaces, plugins | `App.tsx:326–352,353–368,398–419`, `components/Sidebar.tsx:97–103,188–274`, `components/Sidebar.production.tsx:174–246` | Явный отдельный backlog; нельзя объявить их общедоступными функциями reference по наличию route. Волна 7 после подтверждения полезного сценария |

## 3. Фактическая база Accord и важные ограничения

- `internal/core/types.go:9–75`: Principal привязан к owner/machine; Session — к principal/role/run/machine и expiry, а не к постоянной карточке агента. SessionView.Project — имя текущего configured project. Нет provider/model/process ID/start/exit и нет last_seen.
- `internal/store/catalog.go:96–178`: Sessions — авторизованный paginated API; чтение не heartbeat. Его `active` означает только `ExpiresAt > now`. Отзыв/expiry токена чужой сессии не включён в этот predicate; `auth.go:59–73` отзывает токен отдельно. До дополнительного server-derived признака нельзя писать «online/готов работать».
- `internal/core/catalog.go`: Stats содержит first/rework completions, returns, tokens, missing coverage, expired leases, wall/blocked duration; `store/stats.go` считает по событиям, queue — по текущему state. Неполные данные сохраняются как incomplete; elapsed time не compute time.
- `internal/web/contracts.go`: нынешний Board interface даёт ListTasks/ListEpics/Questions/Decisions/Events и human operations, но не Sessions/Stats/config snapshot. Их нужно корректно прокинуть через авторизованный backend, а не читать live DB из templates.
- `internal/web/pages.go`, `actions.go`, `server.go`: браузерная сессия — human. GET не Describe/heartbeat; expired native session продолжается только явно. POST actions используют CSRF и durable action/request IDs. Новый shell не отменяет этот контракт.
- `templates/task.html`, `board.html`, `history.html`, `designs.html`: уже существуют human claim/answer/release, optional heartbeat, drafts после lease loss, create task/epic, evidence/attempts/links/comments, история решений и design upload/download. Это обязательная регрессия первой волны.
- Config project/roles/rules в `internal/config/types.go`, canonical snapshot/hash — `types.go:153–168`; safe effective view требует architect contract, raw YAML/секреты в браузер не выдаются. Hardcoded роли из `web/pages.go` не следует объявлять всей реальной configuration.

## 4. Минимальный цельный первый этап — предложение для общего контракта

Пользовательский результат: войти, увидеть состояние текущего проекта, понять какие сессии держат работу, открыть задачу и доказательства, принять/ответить на human question, проследить события и фактические правила. Все основные пункты навигации открывают содержательные страницы. Визуальное соответствие отдельной WORK-102; ниже поведение, не новый дизайн.

| Экран | Обязательное реальное содержимое и действия | Чего не обещать до следующей волны |
| --- | --- | --- |
| Dashboard | Имя проекта; ready/waiting/in_progress с явным смыслом; completed/rework/tokens за указанный период только из Stats; current sessions/leases; последние события/вопросы; ссылки в tasks/agents/activity | Cost в $ без billing; общий итог по одной первой странице; «агент running» по одному TTL; zero вместо unknown |
| Agents | Таблица/карточки session ID, owner, role, machine, run ID, expiry; фильтры active/expired/all и role/owner в реальных допустимых границах; связанные текущие leases и task links; detail с той же provenance | Persistent profile/model/provider, кнопки Start/Pause/Stop/Save, оргиерархия, если backend их не выполняет |
| Tasks/detail | Список и эпики, role/state/epic filters, все страницы; description/links/blocked/areas/owner/attempts/evidence/review/target SHA; создание task/epic; прежние human question actions; history и designs доступны | Изменение чужих leases, прямое присваивание state в обход core, fake approve, новые runtime/chat операции |
| Activity | Server events с типом, временем, task/session/actor attribution только где реально сохранено; ограниченная страница + честное продолжение; фильтры из EventsInput; task links; решения доступны отдельно | Собирать прошедшие события из нынешнего title/answer/owner; бесшумно терять старые/новые события |
| Projects | Настоящий текущий configured project; связанные эпики/задачи/queue; единственный проект так и обозначен. Details config/repository только если доступен безопасный effective source | «Add project» с локальной пустой карточкой, переключение tenant только в UI, изоляция по area/epic |
| Settings | Текущие project/role/lease TTL/heartbeat/rules и config revision/hash, насколько их безопасный effective source предоставляет; пояснение режима чтения; авторизованный owner/session и logout | Полное raw config, токены, серверные filesystem paths/секреты, runtime-agent settings, сохранение без validate/apply |

Операции, которых нет в core, не показываются как действующие. Для next-wave возможностей допустима только честная информация о статусе, не success toast. Пропуски в фактических данных обозначаются «нет данных», отличая от нуля. Закрытие первого этапа не закрывает общий REQ-005.

### Единые состояния и правила

1. Неавторизованный вход ведёт к настоящему login; logout/revoke/expiry работают на всех новых маршрутах. URL/локальное хранение не содержат bearer/CSRF. Прежние session cookies/Origin guards сохраняются.
2. Пустой dataset, пустой результат фильтра, loading/refresh, недоступный backend, forbidden/not found, native session expired различимы. Ошибка чтения не выглядит как «0 агентов» или «всё завершено».
3. Обновление страницы/фильтров не renew native session/lease и не запускает агента. Список обновляется по видимому действию или фоновому чтению; конкретный transport выбирает архитектор. Время последнего успешного чтения страницы можно показывать как «данные обновлены», не как last_seen агента.
4. Пагинация и totals раздельны: loaded N не общий count. Фильтр меняет pass и сбрасывает cursor. Вкладки и ссылки сохраняют контекст без чужого cursor; invalid cursor даёт понятную ошибку/сброс.
5. Claim/answer/release сохраняют проверку generation/session/lease, durable idempotency и защиту от двойной отправки. При конфликте draft сохраняется; неизвестный результат не становится success. Просмотр не делает авто-claim.
6. Русские названия/тексты и Accord identity; layout/tokens/компоненты из выбранного Paperclip variant, не выдуманные дизайн-решения. Состояния с отсутствующим референсом возвращаются design-reviewer/координатору.
7. Read-only pages используют текущую авторизацию. Сервисные и человеческие сессии различимы; свою web-human сессию нельзя представлять работающим coding-агентом.

### Проверяемая приёмка этапа 1

- AC-01: все шесть разделов и task detail работают с authenticated server fixture; deep link/reload/back сохраняют контекст; существующие questions/history/designs доступны.
- AC-02: очередь и totals совпадают с authoritative store для >100 задач, нескольких ролей/эпиков и комбинации closed/blocked; первая страница не выдаётся за весь набор.
- AC-03: две независимые sessions и человеческая сессия отображают реальные owner/role/machine/run/expiry; claim/release/expiry меняют task association. Session с revoked token не называется online. Отдельного process running нет до adapter.
- AC-04: native session expiry не лечится GET; явный continue работает. Listing/auto-refresh не удерживает агента и чужую аренду живыми.
- AC-05: фильтрация и pagination задач/сессий/events не теряют/не дублируют результаты в одном pass; invalid cursor и reset проверены.
- AC-06: create epic/task → human claim → answer → immutable decision history остаётся реальным; double-submit не дублирует, чужой claim/generation conflict не перезаписывает ответ/draft.
- AC-07: evidence/review target SHA/negative verdict не заменяются generic done/success; исторические unknown metadata не подменяются сегодняшними.
- AC-08: настройки отражают effective config/revision, отсутствие поля обозначено; в response/DOM/assets отсутствуют secrets. Projects честно показывает single-project режим.
- AC-09: logout/revoke/auth errors, backend failure, empty/filter-empty, keyboard/focus и узкий viewport проверены. Ошибки не дают fake empty/success.
- AC-10: независимые code/security/design review и фактический UI/runtime QA относятся к той же итоговой версии. Более ранние approve не переносятся автоматически.

## 5. Полный целевой roadmap без скрытого сокращения

| Волна | Цель и готовый результат | Зависимость/граница |
| --- | --- | --- |
| 1 | Цельный truthful control UI из раздела 4, все существующие human workflows | Доступно на существующем core + read contracts; это первая поставка, не весь Paperclip |
| 2 | First-class projects: registry/stable ID, repository/workspace refs, create/connect/archive, project membership/token scope, switcher, миграция существующей board в default project, tests cross-project non-disclosure | Решение company/project/tenant. Каждая задача/session/event/note/design/config/action/cursor привязана к границе; не только UI filter |
| 3 | Persistent agents: profile/name/role/provider/model/instructions, selected adapters/config schema, permissions, effective config validate/preview/apply/history/rollback, credential readiness без утечки | Решение existing vs managed agents; правомочия/применение к активным sessions; безопасные profiles не управляют чужими процессами |
| 4 | Managed execution: реальные queued/running/finished/error runs, worker registration, environment/worktree isolation, wake/manual run, logs/events/cancel/retry/timeout/recovery, live dashboard | Execution adapter и authenticated worker handshake; code/resource/process lifecycle и credential boundary, проверка реальным Codex/Claude в согласованной среде |
| 5 | Глубокая координация: task views/search/inbox, assignment/dependencies/goals, comments/mentions/interactions, artifacts/docs revisions, approvals и delegation/ownership | Существующие atomic claims/independent SHA gates не ослабляются; Paperclip approval не автоматически эквивалентен reviewer approve |
| 6 | Автоматизация и экономика: routines/triggers/versioning/history, scheduler/retries, budgets/limits/hard stops/quotas и accounting quality | Реальные execution/provider usage. Dollars нельзя оценить из неизвестного tokens_spent; missing coverage видна |
| 7 | Расширения: skills/studio, connector/MCP gateway с per-tool permissions/audit, plugins, company/project portability, notifications/channels/chat; оценить gated cases/status/conference-room отдельно | Сквозные ACL/project isolation и secret lifecycle. Эксперименты включаются осознанно, не потому что route найден |

Каждая волна получает отдельный проверяемый контракт до своей реализации. При необходимости очередность 2/3 меняет координатор после пользовательских ответов; сроки/стоимость и безусловный перенос всех экспериментальных возможностей здесь не выдуманы. Общий запрос остаётся открытым до принятого disposition каждой строки inventory: реализовано / запланировано / исключено пользователем.

## 6. Критические вопросы координатору для реестра и пользователя

| ID | Решение и варианты (1 — рекомендуемый старт, не принятое правило) | Что блокирует |
| --- | --- | --- |
| Q100-1 | Сейчас Accord (1) подключает и наблюдает уже запущенные Codex/Claude через свой MCP, управление запуском — следующая волна; (2) сам создаёт/запускает/останавливает агентов уже в этой поставке; (3) смешанный режим с явным типом агента | Start/Pause/Stop, editable provider/model/runtime settings, исполнение и расходы. Волна 1 observability независима |
| Q100-2 | «Проект» = (1) репозиторий/рабочая область внутри общей организации пользователя; (2) полностью изолированная board/tenant; (3) project registry подключает отдельные существующие серверы | Data/auth/migration модель wave 2, назначение switcher; эпик не может молча стать проектом |
| Q100-3 | Какая изоляция для управляемых запусков: (1) отдельный worker/worktree на машине пользователя; (2) процессы на VPS Accord; (3) отдельные sandbox/container окружения | Где credentials/cwd/network/limits/cancel действительно применяются. Это не выбор внешнего вида |
| Q100-4 | «Все фишки» принимается (1) поэтапно с явной матрицей parity и текущей шестистраничной волной; (2) до первой поставки нужны конкретно названные missing features | Не позволяет выдать первый shell за выполненный полный запрос. Наличие «Приступай» не разрешает скрытые исключения |

Q100-1/2 переданы координатору ранним сообщением; Q100-3/4 передаются этой WORK. Вопросы пользователю самостоятельно не отправлял, общий реестр не редактировал. Техническая реализация доступной волны может идти после фиксации координатором её контракта; решения отсутствующих функций не предполагаются.

Уточнение координатора при передаче: пользователь прямо отнёс multi-project к будущему. Текущий implementation scope — один actual configured project и ADR расширения; архитектор готовит DTO. Это снимает необходимость блокировать текущий control UI на Q100-2/3/4; вопросы остаются gates будущих волн. Execution-вопрос координатор уже отправил пользователю, ответ ожидается; зависимые execution-кнопки пока не реализуются. Разработчики могут брать раздел 4 после общего контракта координатора.

## 7. Проверки, ограничения, передача

**PASS (source inspection):** зафиксированы HEAD обеих копий, LICENSE, дерево/маршруты и фактические UI/API/schema Paperclip, current Accord types/store/web/config/templates; сохранены границы сессия/процесс/проект/tenant. `git status --short` до записи показывал только чужую новую REQ-005; сохранена.

**NOT-RUN:** Paperclip build/browser и runtime функций reference; Accord code/build/tests/E2E/deploy — документационная роль их не запускала. Таблица не является runtime PASS или независимым code/design approve. Память использована только для ориентации о самостоятельном Accord/Tandem; текущий AGENTS перечитан и подтверждает границы.

Следующие: координатор переносит согласованную дельту в общий контракт/реестр; архитектор определяет безопасные read models и project/agent extension seams; разработчик получает волну 1 после фиксации контракта; design-reviewer определяет конкретный reference shell/страницы и проверяет actual UI. Снижение объёма без disposition inventory не считать выполнением REQ-005.

## Журнал

- 2026-10-19T19:16:05Z — started: начат read-only анализ, профиль и самостоятельный AGENTS прочитаны; резерв WORK-100 задан координатором.
- 2026-10-19T19:17:11Z — checked: Accord HEAD a6501d8621abba558606b7fc26d7407fb3b2065e; reference HEAD 5739a53f5f5e6cccbf21c7ae491ad06cb4b79b7c, MIT; ранние product gates переданы координатору.
- 2026-10-19T19:17:41Z — completed: inventory, первый поведенческий контракт, AC, roadmap и Q100-1…4 переданы в этой WORK. Принятие контракта/решений и общая приёмка принадлежат координатору.
- 2026-10-19T19:19:51Z — handed_off: финальный файл проверен (128 строк перед этим уточнением), reference source paths подтверждены, `git diff --check` PASS для tracked diff. Предыдущая completed-строка использовала время проверки перед записью; фактическое завершение и передача относятся к этому событию. Новые сторонние untracked документы координатора/архитектора/дизайн-ревьюера не изменялись. Current-control scope уточнён по сообщению координатора, implementation не блокируется будущими multi-project gates.
