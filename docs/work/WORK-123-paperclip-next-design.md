# WORK-123 — Paperclip: точный визуальный контракт для Next.js-клиента

- Родитель: TASK-007; роль: независимый designer/analyst, автор приложения другой исполнитель.
- Дата: 20.10.2026. Статус: source-based design handoff; не approve реализации.
- Область записи: эта карточка и `.local/task007-reference/`. Приложение, зависимости, Git, runtime и VPS не менялись.
- Прочитаны: requirements, architecture, control-web-architecture, WORK-100/102/110; актуальные web contracts/pages/templates; pinned Paperclip.
- Paperclip: `/private/tmp/accord-paperclip-reference`, HEAD `5739a53f5f5e6cccbf21c7ae491ad06cb4b79b7c`.

## 1. Решение и проверяемый смысл «как Paperclip»

Выбран **один coherent baseline: current-default Streamlined UI**: `ui/src/components/Layout.tsx`, `Sidebar.tsx`, `ui/src/pages/Agents.tsx`; флаг `useStreamlinedUiEnabled` включён по умолчанию. Не смешивать с `.production.tsx`, старым company icon rail и gated experimental chat. Paperclip frontend сам использует Vite/React, поэтому Next.js — архитектурный выбор Accord для нового React/TypeScript-клиента, а не копирование framework reference.

Из Paperclip переносится конкретная система React primitives, навигационная композиция и плотность данных, а не только серые цвета. Визуальное соответствие требует компактного company/project menu, обычного nav-action «Новая задача», поиска отдельной строкой, сгруппированной навигации, полосы агентов на dashboard, плоских метрик, компактных list rows, line tabs и вторичной панели свойств. Сохраняется узнаваемое имя Accord и текущий домен.

Reference images осмотрены через view_image; копии и SHA256 находятся в `.local/task007-reference/manifest.json`:

| Файл | Источник и доказательная граница |
| --- | --- |
| `paperclip-dashboard-dark-historical.png` | `screenshots/PR-8000-home-flag-off.png`; целый dark desktop shell с sidebar240, agent strip, flat metrics, charts/recent rows. Исторический capture, не current pinned runtime |
| `paperclip-task-components-dark-historical.png` | `docs/pr-screenshots/pr-5291/after-issue-management.png`; contact sheet task list/filter/properties/ledger/quick look. Исторический component fixture, не целая current страница |
| `accord-task006-dashboard-light.png` | `.local/task006-qa/refinement-anchor/overview-1440-light.png`; фактическое прежнее состояние Accord, не будущая Next версия |

Не использовать `docs/pr-screenshots/pr-6384/sidebar-desktop.png` как эталон: WORK-102 установил ошибку Storybook provider. В этой WORK нет запуска Paperclip и нет доказательства pixel identity его сегодняшнего live UI. Главный baseline — зафиксированные sources; изображения подтверждают геометрию и информационную композицию с указанной исторической границей.

## 2. Обязательные изменения против TASK-006

1. Убрать двойной верх sidebar (крупный Accord logo и отдельная project card). Вместо него одна строка высотой60 с project/company dropdown, mark24 и chevron. Полное длинное имя в title и доступном menu.
2. «Новая задача» — обычная nav row с SquarePen16, как `Sidebar.tsx`, без растянутой чёрной плашки. Поиск — соседняя nav row с Cmd/Ctrl+K. Верхняя строка main не превращается в большую поисковую панель.
3. Dashboard начинает секция «Агенты» с compact карточками реальных сессий/занятых задач; после неё плоские KPI. Нынешние четыре крупные bordered cards и большой project-state заголовок не соответствуют композиции.
4. Убрать длинные технические оговорки/checkbox автообновления из основного потока dashboard. Смысл статусов остается точным: подпись «Сессии с доступом», tooltip/help, компактное «Обновлено…» и явная ошибка/повтор. Техническое объяснение про отсутствие process telemetry — в detail/help. Фоновое чтение не продлевает аренды.
5. Task list — плотные строки с key/status/title/role/owner и group header; controls в toolbar/popover, а не огромная постоянно раскрытая форма. Board — самостоятельное сохранённое представление той же выборки.
6. Task detail — главная колонка описания, истории/ответа и secondary properties panel. Метаданные не вставляются большой таблицей между заголовком и человеческим ответом.
7. Agents — avatar/name/role/state/held task/expiry; line tabs и toolbar. Detail имеет identity header, tabs и компактный фактический configuration/session panel; не отдельная длинная карточка со всеми предупреждениями.
8. Projects/settings — настоящая структурированная surface (list/detail и tabbed settings); giant future cards не заменяют содержимое. Отсутствующие provider/profile/execution операции не рисуются действующими.

## 3. Точный shell и tokens

| Элемент | Контракт, источник |
| --- | --- |
| Desktop shell | `Layout.tsx:577+`: h-dvh, main отдельный scroll, sidebar240px; `.SidebarShell.tsx` допускает208–420, но первый baseline240. Main p24. Без маркетингового hero/header/footer |
| Header | `BreadcrumbBar.tsx`: h60, bottom border, px16/md24; breadcrumbs14; правый toolbar небольших icon buttons. Таблица/страница имеет один ясный заголовок, не дублирующие h1 в top bar и main |
| Sidebar | `primary-sidebar-styles.ts`: nav `flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-3 py-2`; group gap2px. Row `mx-2 px-2 py-1.5 gap-2.5 rounded-lg text13 font-medium`, icon16 |
| Sidebar surface | `.primary-sidebar-surface`: light color-mix(border50%,background), dark muted. Active row sidebar-accent; не сине-зелёный brand background |
| Type | `index.css`: InterVariable local regular/italic; 13px nav, 14px rows, 12px metadata, 11px IDs, 10px dense labels. Headings обычно18–24, KPI24/mobile30/desktop |
| Radius | anchor8px; sm4.8, md6.4, lg8, xl11.2. Не добавлять крупные16/24px контейнеры без reference source |
| Light | background/card `oklch(1 0 0)`, foreground `.145`, primary `.205`, muted `.97`, muted-fg `.556`, border `.922`, sidebar `.985`, sidebar-accent `.88` |
| Dark | background/card `.205`, foreground `.985`, muted/accent `.269`, muted-fg `.708`, border white10%, sidebar `.205`, sidebar-accent `.32` |
| Status | blue in-progress, green done, amber ready/waiting, red error/blocker, neutral cancelled/expired. Текст/shape обязательны. Blocked вычисляется, не новый state |
| KPI | `MetricCard.tsx`: borderless `px-4 py-4 sm:px-5 sm:py-5 rounded-lg`; value text2xl/3xl font-semibold tabular; label12/14 muted, secondary12. Hover фон только clickable |
| Dashboard | `Dashboard.tsx`: vertical gap24; active agents first; KPI2col→4 at xl; chart grid2→3 без скрытого priority; recent panels2col md; section title14 uppercase muted tracking-wide |
| Agent cards | `ActiveAgentsPanel.tsx`: grid1→2 sm→4 xl, items-start, gap8/16. Для Accord содержимое session/lease, не поддельный transcript/run status |
| Entity row | `EntityRow.tsx`: px16 py8 gap12 text14 border-bottom; leading avatar32, min-w0 truncate, metadata/right cluster bounded. Hover accent50% |
| Kanban | `KanbanBoard.tsx`: column260, gap12, scroll только board; отсутствие DnD API означает отсутствие drag affordance |
| Mobile | <768: drawer240 с backdrop, sticky h60 header, body scroll; p16; safe-area и opaque bottom nav. Touch controls44, editable fields≥16. Main не имеет horizontal page scroll |
| Motion | drawer100ms, shared fast160ms/instant80ms; reduced-motion отключает движение/pulse. Focus видим; Escape закрывает popover/dialog, focus возвращается инициатору |

Скопировать **нужный subset** `ui/src/index.css`, а не весь файл с более2500 extracted feature tokens. Используемые primitives ссылаются на `--sz-*`, `--tp-*`, `--rad-*`, `--pct-*`, `--e-*`: каждую ссылку перенести/нормализовать явно; неопределённые custom properties — визуальный дефект.

## 4. React-части для прямой адаптации и библиотеки

| Реально пригодный source | Адаптация |
| --- | --- |
| `ui/src/components/ui/button.tsx`, `badge.tsx`, `input.tsx`, `textarea.tsx`, `separator.tsx`, `skeleton.tsx`, `card.tsx`, `label.tsx` | Сохранить class contract/variants и cn; card default py24 не навязывать list surfaces — как reference, применять `block py-0 divide-y` |
| `ui/src/components/ui/dialog.tsx`, `dropdown-menu.tsx`, `popover.tsx`, `tooltip.tsx`, `tabs.tsx`, `command.tsx`, `sheet.tsx` | React client components; Radix обеспечивает semantics/focus. Next Link вместо reference router. CSS animation vars определены до использования |
| `ui/src/lib/utils.ts` | Перенести только cn/class merge, не весь domain/router utilities |
| `EntityRow.tsx`, `MetricCard.tsx`, `PageTabBar.tsx`, `SidebarSection.tsx`, `primary-sidebar-styles.ts` | Низкая domain coupling; Next Link/client API и Accord labels. Это прямые визуальные building blocks |
| `Sidebar.tsx`, `SidebarCompanyMenu.tsx`, `BreadcrumbBar.tsx`, `PropertiesPanel.tsx` | Перенести composition/spacing, заменить Paperclip contexts/API на Accord typed DTO; не копировать plugins/experimental routes |
| `IssueRow.tsx`, `IssuesList.tsx`, `IssueFiltersPopover.tsx`, `KanbanBoard.tsx`, `ActiveAgentsPanel.tsx` | Образцы layout/interaction. Не копировать целиком: они тянут Paperclip shared/runtime/recovery contracts |

Минимальные осмысленные зависимости: Next+React+TypeScript; Tailwind4/PostCSS; `lucide-react`; `clsx`+`tailwind-merge`; `class-variance-authority`; `radix-ui`; `cmdk`. `@tanstack/react-query` уместен для agent/activity refresh, query invalidation и pending state; не второй auth store. `react-markdown`+`remark-gfm` — если настоящий Markdown нужен описаниям; raw HTML выключен. Не нужны workspace adapters, assistant-ui, xterm, yjs, mermaid, dnd-kit, editor и chart package без реализуемого сценария. Простые charts по существующему Stats можно сделать SVG с accessible summary.

Не копировать последние dependency versions Paperclip автоматически: coordinator/author проверяет официальную поддержку выбранных Next/React/Node API и фиксирует совместимые версии. Внешних API/версий эта WORK не назначает.

MIT copyright Paperclip и полный license сохраняются в `docs/reference/paperclip` и поставляемом Next public notice. Inter требует OFL notice; существующий локальный font допустим. Бренд assets Paperclip/агентные capsule artwork не обязательны для структуры Accord.

## 5. Десять содержательных экранов и details

Ниже frontend coverage, **не обещание ten new backend features**. Нынешний `internal/web/control_data.go` открывает JSON только dashboard/overview; для остальных нужен авторизованный page DTO/BFF, иначе нельзя читать Go template HTML как API. Все новые маршруты должны сохранять browser cookie, CSRF/action-id и immutable backend rules.

| Экран | Paperclip-композиция | Реальная база Accord / требуемые действия |
| --- | --- | --- |
| 1 Обзор | Agent strip → flat KPI → реальная status distribution → recent tasks/activity | `readDashboard`, `ControlOverview`, bounded ControlTasks. Sessions≠running process; chart по фактическому state aggregate без invented14-day values |
| 2 Задачи | Line tabs/views, compact toolbar/search/filters, grouped dense rows, list/board toggle | `ControlTasks` + epics pagination, all current filters. Cursor bound to filter. Board сохраняет original query; no client state mutation |
| 3 Вопросы/Inbox | Compact attention rows, mine/all/ready view, count, clear empty | `ListQuestions`, `MyQuestions`; claim только явным действием. Не переименовывать вопрос в общий approval |
| 4 Агенты | Line status tabs, filter popover, dense EntityRows, active lease metadata | `ControlSessions` active/expired/credential_invalid; kind agent/human/all. Нужен JSON DTO; server status не заменять browser expiry heuristic |
| 5 Проекты | Dense project list с mark, status, task progress; project detail | Сейчас один configured project, а не registry. Dropdown содержит реальную board; создания/переключения чужих tenants пока нет |
| 6 Активность | Toolbar filters, date groups и compact audit rows; load more | `RecentEvents` task/session/type/cursor; stable event IDs. Не выводить event actor по нынешнему task owner |
| 7 Статистика | Period/group controls, flat metrics, charts/table | `Stats`: durations/tokens/returns/coverage. Unknown≠zero, tokens≠money, wall time≠compute. No costs/budget screen с пустыми$0 |
| 8 Дизайн/Artifacts | Grid/list assets, inline preview, version/hash metadata; upload sheet | Настоящие DesignAsset/download/upload/version rules. Image preview alt, unsupported binary download. Файлы авторизованы через Go |
| 9 Решения | Timeline/list, filters, decision detail с provenance | `ListDecisions` immutable stored events. Не пересобирать прошлый answer из нынешнего task |
| 10 Настройки | Contextual side navigation + line tabs General/Rules/Session; fact rows | Safe SessionView TTL/heartbeat/rules; role capabilities и session. Конфигурация read-only: нет Save без валидированного apply endpoint |

Details и формы обязательны: task detail, epic detail, agent session detail, project detail, design version detail/preview; create task/epic modal; task question answer/lease panel; command palette; mobile drawer; login/logout/session continuation. У каждой основной коллекции loading, empty, filtered-empty, error/retry и paginated-tail states.

**Task detail:** breadcrumb Tasks/T-N/title; main title и status; center content tabs Описание/Активность/Попытки/Доказательства; right properties status/role/epic/owner/priority/areas/lease/generation. Для human в главной колонке pending question/answer composer с явным claim, countdown и сохранённым draft. Lease conflict не уничтожает текст. На mobile properties открываются sheet/accordion, а не занимают весь верх страницы.

**Agent detail:** identity header(role/name, machine, server status), tabs Обзор/Задачи/События/Сессия. Реальные sessionID/runID/expiry/generation и LastEventAt. Отсутствующие provider/model/process не рисуются настройками. Настройка существующего клиента — только инструкция без токена и fake Save; persistent profile editor требует новой backend сущности.

**Project detail:** current project identity + tasks/epics/activity links и available aggregate. Tab General/Rules использует тот же safe snapshot, не raw config/path/secrets. Company dropdown не объявляется multi-project.

## 6. Что нужно для настоящей parity, а не декоративной имитации

| Будущая возможность Paperclip | Отсутствующий Accord-контракт |
| --- | --- |
| Easy project connection/switching | Registry opaque projectID, membership, отдельные DB/config/credentials, scoped cookies/actions/cursors/uploads/audit; migration текущей board |
| Agent configuration | Persistent profile отдельно Session: role/instructions/access/project membership; validate/apply/revision/audit. Model/provider требуют adapter config и server secret reference |
| Start/pause/stop/chat/live logs | ExecutionRun/worker/adapter, liveness/start/exit/cancel/recovery и отдельные permissions; task lease не заменяет run |
| Org graph/goals | Persisted reporting lines и goal links, не произвольные role labels или rename epic |
| Budgets/costs/routines | Provider accounting+coverage/currency, budget enforcement; scheduler trigger/version/run history |
| Integrations/skills/tool approval | Scoped connector/MCP gateway, tool permissions, audit и secret storage |

Эта таблица сохраняет запрос пользователя «такие же фишки» как открытый объём; frontend migration сама по себе его не закрывает. Публично доступные reference и экспериментально gated features различаются. Не нужен десяток inert пунктов sidebar со словами «скоро».

## 7. Screenshot acceptance для следующего независимого design review

1. Desktop1440×900 light/dark: whole shell+overview; sidebar240, один company menu, New Task/Search обычными rows, strip агентов первой секцией, flat KPI; хотя бы начало recent content в первом viewport. Никаких giant disclaimer blocks/footer.
2. Tasks1440×900: 20 synthetic строк, длинный200-char title/owner/epic, группировка, toolbar и открытый filter popover; board с column260 и horizontal board scroll без scroll всей страницы.
3. Task detail1440×900: длинное описание, center tabs, side properties, видимый primary human action; другой capture — active answer draft + lease countdown; mobile390×844 — composer/CTA не перекрыты bottom nav.
4. Agents1440×900 и390×844: mixed active/expired/invalid + held task, empty-filter и agent detail, не зажатое нулевой шириной имя. Status от сервера, роли и metadata совпадают DTO.
5. Projects/settings/activity/design/history/stats: минимум one real-data capture на route, rules/session details видимы, no fake save/start/cost. Dropdown/focus/tab/mobile sidebar отдельно.
6. Responsive768 и390, light/dark, reduced-motion: page width ≤ viewport; drawer focus trap/Escape/return; touch editable16px, controls44, safe-area; long keys доступны и не ломают layout.
7. Loading/refetch/error/stale/401/403/409/unknown-response — browser checks, не только static screenshots. Login token не попадает в query/localStorage/reports. Действия обновляют query по server result без optimistic domain success.

Это acceptance target; будущие screenshots/E2E — NOT-RUN в данной WORK. Unit/integration author пишет сам; process/E2E после независимого approve — отдельный tester, затем review новых тестов по AGENTS.
