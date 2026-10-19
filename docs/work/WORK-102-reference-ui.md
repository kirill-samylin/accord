# WORK-102 — UI-контракт Paperclip → Accord

- Родитель: [REQ-005](../REQ-005-paperclip-control.md). Роль: design-reviewer, ранний reference audit; автор реализации — другой исполнитель.
- Статус: handoff готов; это не дизайн-approve будущего Accord.
- Начало: 2026-10-19T19:16:28Z. Фактическая дата проверки: 19.10.2026 UTC.
- Профиль/процесс Давинчи прочитаны read-only; Accord AGENTS имеет приоритет: Tandem OFF, Davinci не меняется.
- Границы записи: этот файл и `/private/tmp/accord-control-099/reference-ui/`. Продуктовый код, Git, сервер, очередь не изменялись.
- Ограничение среды: профиль передан путём; закреплённые модели Claude не применяются. Ранний аудит не подтверждает качество кода или E2E.

## Версии и сила доказательств

Paperclip: `/private/tmp/accord-paperclip-reference`, HEAD `5739a53f5f5e6cccbf21c7ae491ad06cb4b79b7c`, рабочее дерево на момент чтения чистое. Accord: `/Users/shagi/Desktop/Project/agent-board/.local/web-control-099`, HEAD `a6501d8621abba558606b7fc26d7407fb3b2065e`; согласованный вход — REQ-005 и действующий web-guide. Соседние изменения участников не входят в этот аудит.

Все пути Paperclip ниже относительно reference checkout. Постоянная ссылка на исходник имеет вид `https://github.com/paperclipai/paperclip/blob/5739a53f5f5e6cccbf21c7ae491ad06cb4b79b7c/<путь>`.

Осмотрены реальные пиксели пяти сохранённых изображений. Копии и SHA256 — `/private/tmp/accord-control-099/reference-ui/evidence-manifest.json`. Это upstream captures/fixtures, не новый запуск приложения на данном HEAD. ОС текущего аудита macOS; viewport и browser zoom исходных upstream captures не подтверждены; размеры изображения не означают CSS viewport.

| Изображение Paperclip | Доказательство / ограничение |
| --- | --- |
| `doc/assets/pap-3368/mobile-planning-detail.png` | Пригодный сохранённый mobile UI: breadcrumb, заголовок, длинный ключ, tabs, empty chat, composer, bottom nav; 390×1003 пикселя; исторический shell |
| `docs/pr-screenshots/pr-15773/overview-view-light.png` | Пригодный светлый detail: header/actions, ограниченная ширина текста, code block; отдельная routine surface, без общего sidebar |
| `docs/pr-screenshots/pr-5291/after-navigation-layout.png` | Пригодный fixture contact sheet: sidebar, organization switcher, breadcrumbs, tabs, mobile nav, focus/shortcuts. Старое поколение с icon rail |
| `docs/pr-screenshots/pr-5291/after-issue-management.png` | Пригодный fixture contact sheet: dense task list, filters, detail/properties, documents, run ledger; не screenshot сегодняшнего live dashboard |
| `docs/pr-screenshots/pr-5428/assigned-backlog-light.png` | Fixture component board: status chips, предупреждения, disabled/parked semantics; продуктовые правила относятся только к Paperclip |

Исключено после осмотра: `docs/pr-screenshots/pr-6384/sidebar-desktop.png` содержит Storybook error `usePluginLauncherRuntime must be used within PluginLauncherProvider`; это не эталон интерфейса. Часть других screenshots не проверена; наличие PNG не доказывает пригодность.

## Какое поколение брать

В `ui/src/App.tsx:855` shell выбирается **runtime** флагом `enableStreamlinedUi`, не названием build mode. `ui/src/hooks/useStreamlinedUiEnabled.ts` возвращает true, если настройка не равна false: current default — `Layout.tsx`, `Sidebar.tsx`, `Agents.tsx`; `.production.tsx` — альтернативный legacy surface. Не смешивать произвольно оба поколения.

Предлагаемый baseline для handoff — current default sources. В `ui/src/context/SidebarContext.tsx:37-73` breakpoint 768px, desktop nav постоянно раскрыта; `collapsed`, `peeking`, setters сохранены как inert compatibility API. Поэтому icon rail из старого contact sheet не является обязательным текущим поведением. `SidebarShell.tsx:15-18` задаёт expanded width 240px, min208/max420, шаг16px; legacy комментарии про rail в shell не отменяют действующий context.

## Контракт поверхностей и адаптация данных

| Поверхность | Фактический Paperclip source | Перенос в Accord / проверяемая граница |
| --- | --- | --- |
| Shell | `components/Layout.tsx:577-743`, `SidebarShell.tsx`, `primary-sidebar-styles.ts` | Слева постоянная навигация desktop, справа рабочая область, breadcrumb/top bar, отдельный scroll; main padding16 mobile/24 desktop. Сохранить skip link, `aria-current`, роли сообщений. Не делать ещё один широкий marketing header |
| Navigation | `components/Sidebar.tsx`, `SidebarNavItem.tsx:125-145`, `SidebarSection.tsx` | Компактные сгруппированные ссылки, 16px icons, 13px labels, active row fill, counters по данным. Accord сохраняет доступ к Вопросам, Задачам, Дизайну, Истории решений; новые Обзор/Агенты/Проекты добавляет по контракту координатора |
| Organization/project identity | `components/SidebarCompanyMenu.tsx`, `CompanySwitcher.tsx`, `context/CompanyContext.tsx` | Компактное имя/значок и dropdown со selected state. Paperclip company — верхняя граница данных, project — внутри company. В Accord `.Session.Project` сейчас одна project identity; визуальный switcher сам по себе не реализует multi-project. Список только реально доступных проектов; создание/настройки ведут в настоящие сценарии |
| Dashboard | `pages/Dashboard.tsx:298-325,400-550`, `components/ActiveAgentsPanel.tsx` | Компактные counters, recent tasks/activity, agent run cards; grid2→4 counters и md2 panels. Источник Paperclip liveness — heartbeat live-runs endpoint, а не имя агента или task.status. В Accord цифры должны быть из server aggregates или явно ограниченного набора; нельзя выдавать количество текущей страницы за весь проект |
| Agent roster | `pages/Agents.tsx:49-96,333-411,493-515`, `EntityRow.tsx` | List rows: avatar32, имя/роль, status chip, metadata; status filter tabs, count; overflow truncation c доступным полным значением. Для Accord session roster показывать owner/machine/role/run_id/expiry/назначенную задачу, если API даёт эти факты. Не выдавать TTL сессии за работающий OS/LLM process |
| Task board/list | `components/IssuesList.tsx`, `IssueRow.tsx`, `IssueFiltersPopover.tsx`, `KanbanBoard.tsx:36-46,217,465-478` | Dense list либо status columns шириной260px и отдельный horizontal scroll, gap12px; key/title/status/role/owner/epic. Paperclip seven statuses и drag/drop не копируются как бизнес-правило: Accord сохраняет собственные state machine, leases и evidence. Drag/drop запрещено показывать работающим без авторизованной transition операции |
| Task detail | `pages/IssueDetail.tsx`, `components/IssueProperties.tsx`, `PropertiesPanel.tsx`, `PageTabBar.tsx` | Breadcrumb/key/status/title + content и secondary properties panel; secondary metadata не заглушает основное действие. В Accord остаются явное «Взять и ответить», lease countdown, generation/conflict handling, сохранение черновика, доказательства, attempts/comments. Перенос tab style не создаёт chat/run history, если данных нет |
| Create/configure | `components/NewIssueDialog.tsx`, `pages/NewAgent.tsx`, `components/new-agent/NewAgentSetup.tsx` | Компактные формы и видимые pending/error состояния можно адаптировать; agent profile/configuration и настоящий execution adapter — разные возможности. «Создать задачу» не должно называться «Запустить агента» |
| Mobile | `context/SidebarContext.tsx`, `Layout.tsx:595-624,680-743`, `MobileBottomNav.tsx` | <768px off-canvas sidebar240px + backdrop, sticky top bar, safe-area/bottom padding, single-column content. Reference mobile drawer transform100ms. Bottom nav содержит только реальные назначения/действия Accord; нельзя оставлять composer под фиксированной панелью |

## Токены, типографика, геометрия

Источники: `ui/src/index.css:5-18,40-91,107-145,408-449,1123-1129,2579-2581`, `motion-tokens.css`.

| Токен / правило | Фактическое значение референса |
| --- | --- |
| Typeface | InterVariable variable100–900, regular/italic локальные WOFF2; system sans fallback; monospace только IDs/code/metadata по смыслу |
| Text scale | nano10px, micro11px, compact13px; стандартные xs/sm/base у Tailwind —12/14/16px; не масштабировать весь UI до старых Accord h1=32/h2=22/card padding24 |
| Radius | anchor8px (`--radius:.5rem`); sm4.8/md6.4/lg8/xl11.2px; один согласованный scale |
| Light palette | background/card white `oklch(1 0 0)`; foreground `.145`; muted `.97`; muted-fg `.556`; border `.922`; sidebar `.985`, active sidebar `.88` |
| Dark palette | background/card `.205`; foreground `.985`; muted/accent `.269`; muted-fg `.708`; border white10%; sidebar `.205`, active sidebar `.32` |
| Current sidebar surface | Light `color-mix(in srgb,var(--border) 50%,var(--background))`; dark `var(--muted)`; source `.primary-sidebar-surface` |
| Status hue | running/in_progress blue#2563eb; paused/todo amber#f59e0b; error/blocked red#dc2626; review violet#7c3aed; done green#22c55e; inert gray#a8aeb2. Всегда подпись/shape дополнительно к цвету |
| Spacing | shell padding16/24; nav outer horizontal12, row margin8 + inner8, gap10; row vertical6; panel sections16/24; task columns260 + gap12; не превращать плотный список в массивные карточки |
| Motion | instant80ms, fast160ms; ease standard cubic-bezier(.4,0,.2,1), expo(.16,1,.3,1); mobile drawer100ms. Sidebar pin/resize в source shell намеренно без width transition. `index.css` содержит reduced-motion overrides |

Это перенос визуального языка с адаптацией по требованиям, не требование встроить React/Tailwind/Radix в Go-приложение. Согласованный общий UI Accord реализует его собственный слой; `packages/ui-kit` Давинчи сюда не переносится.

## Состояния: что есть и что нельзя потерять

| Состояние | Paperclip evidence | Требование к проверке Accord |
| --- | --- | --- |
| Normal | Перечисленные sources и upstream captures | Screenshot целого shell + dashboard/roster/list/detail на конкретной версии; не только отдельные cards |
| Loading | `PageSkeleton.tsx` варианты dashboard/list/detail; Dashboard:311, Agents:337 | При server navigation может быть native pending; для async refresh нужен pending, без fake0 и мигания empty. Полноценный screenshot нужного нового async состояния ещё отсутствует |
| Empty | `EmptyState.tsx`; Dashboard no organization/no tasks; Agents empty roster и no matching status | Различать пустой проект, нет совпадений, нет доступных проектов; CTA только если реально работает |
| Error/retry | Dashboard:325 и Agents:493 выводят error; `CompanySwitcher.tsx` отличает list unavailable от empty и даёт retry | Не интерпретировать сбой API как «нет агентов». Сохранить `.Error`, drafts, доступный retry; единообразный error/retry layout для всех новых поверхностей по reference capture не подтверждён |
| Offline/stale | В текущем осмотре нет полного offline screenshot reference | В Accord `assets/app.js` уже запрещает отправку offline, сохраняет draft, останавливает heartbeat; новый UI должен сохранять это. При потерянном refresh — последняя дата данных/ошибка, без online pulse по старому snapshot |
| Disabled/pending | Reference формы/кнопки; existing Accord `aria-busy` и disabled submit | Повтор submit блокируется; неподдерживаемые действия не показывают фиктивный success. Реально недоступную настройку объяснить, не выдавать за готовую функцию |
| Hover/focus/active | SidebarNavItem source; upstream nav contact sheet | Keyboard visible focus, активный route, tooltip/accessibility name у icon-only action; проверить Escape/focus return если появился drawer/dialog |
| Long text/overflow | Truncate в roster/sidebar; mobile capture содержит длинный issue key | 200-character title, длинный project/owner/machine, multi-line description, узкая ширина. Полное значение остаётся доступным; без page-wide horizontal scroll |
| Expired/401/403/conflict | Не переносить Paperclip permissions как Accord rules | Существующий Accord task.html/app.js: renewal явно, потеря lease401/403/409 без тихого перезахвата и потери ответа; cookie/session boundary и logout сохраняются |
| Reduced motion | Несколько overrides в index.css | Проверить реальные transitions/pulse при reduce; static screenshot не подтверждает motion |

Нет доказательства, что каждый отсутствующий reference capture означает отсутствие реализации. NOT-RUN отдельно: запуск Paperclip, browser interaction/keyboard, live dashboard, current-default whole-shell screenshot, полный accessibility audit. Никакие agents/reference серверы не запускались, зависимости не ставились.

## Разница с текущим Accord и риски

Текущий `internal/webui/templates/layout.html` имеет горизонтальные header/nav и main max1180; `assets/app.css` — зелёный accent#176b61, h1=32px, cards24px; `board.html` — серверные filters, epic grid и task cards; `task.html` — факты/lease/ответы/результаты. Это рабочие сценарии, которые новый визуальный shell должен сохранить. `docs/web-guide.md` остаётся функциональной границей вопросов/дизайна/истории.

| ID | Риск / условие исправления | Адресат |
| --- | --- | --- |
| UI102-01 | Смешать исторический icon rail и current default shell. Зафиксировать один baseline sources и выбранные адаптации в общем контракте | coordinator/developer |
| UI102-02 | Pulse «работает» от sessionTTL или task.in_progress. У UI должно быть честное имя «сессия активна/задача во владении», а process-running — только от execution adapter | architect/developer |
| UI102-03 | Fake project switcher/agent setup/run/cost controls. Каждое action связано с реальным endpoint и подтверждённым результатом; неподдерживаемое вынесено в будущий объём явно | analyst/developer |
| UI102-04 | Из-за редизайна исчезают человеко-вопросы, upload/version/hash, decision history, explicit lease/retry и draft retention. Пройти существующие сценарии web-guide после нового shell | developer/tester |
| UI102-05 | Клонирование Paperclip статусов и DnD ломает Accord ownership/review workflow. Только текущие domain labels/transitions, без client-side success до серверного ответа | architect/reviewer |
| UI102-06 | Счётчики total/running берутся из неполной страницы. Использовать авторитетные aggregates либо явную надпись о неполном наборе; сбой и unknown отличать от0 | developer/reviewer |

Core evidence: `internal/core/types.go` Session/Lease/Task/Attempt; `core/admin.go` SessionInfo; `core/catalog.go` SessionsPage/QueueSummary и StatsMetrics; `internal/web/contracts.go` текущий Page ещё не имеет roster/run surface. Наличие `run_id` — имя запуска сессии, не доказательство живого внешнего процесса. `Attempt.TokensSpent` nullable: неизвестное значение нельзя превращать в нулевую стоимость.

## Источник и attribution

Reference repository — Paperclip AI; корневой `LICENSE`: MIT, Copyright(c)2025 Paperclip AI. При переносе существенных частей кода/CSS сохранить copyright и полный license notice; собственная MIT Accord не заменяет атрибуцию заимствования. Копирование конкретных brand logos/маскотов не требуется для принятого визуального языка. Весь reference backend/SDK не является необходимой зависимостью UI.

`ui/public/fonts/NOTICE.md` отдельно указывает Inter v4.1, SIL Open Font License 1.1, upstream URLs. При переносе WOFF2 перенести соответствующее уведомление и полный license text, не считать шрифты покрытыми только Paperclip MIT. Для визуальной близости рекомендован self-host `InterVariable.woff2` с SHA256 и official OFL; system stack достаточен как рабочий fallback, но меняет ширины строк, переносы и геометрию и поэтому сам по себе не подтверждает близость типографики. Italic WOFF2 нужен, если italic действительно используется. Координатор ведёт THIRD_PARTY/manifest; Paperclip brand logo не переносится.

У сторонних адаптерных logo assets отдельные источники/LICENSE — без необходимости их не копировать. Это инвентаризация notices из источников, не утверждение о правах на любой artwork.

## Проверки, передача и продолжение

- PASS: роль/процесс/Accord AGENTS/REQ/web-guide и перечисленные source chains прочитаны; HEAD зафиксированы;5 пригодных saved captures осмотрены, copied/hash manifest сохранён;1 broken capture исключён.
- NOT-RUN: новый runnable Accord UI, screenshot comparison, browser/keyboard/motion/responsive interactions, E2E, живой Paperclip. На раннем этапе этого ещё нет; визуальный approve **не выдан**.
- Изменён только этот WORK и локальные review artifacts. Независимые source/backend/contracts и другие WORK не редактировались.
- Следующий исполнитель: coordinator фиксирует baseline/границу реальных функций с analyst/architect; developer получает этот контракт. После реализации design-reviewer осматривает конкретный commit/diff и actual UI минимум1440×900,768px boundary,390px mobile, loading/empty/error/offline/expired/long text; zoom100% и reduced-motion фиксируются в новом ревью.

Журнал: 19:16Z started/read-only inventory →19:18Z reference source/captures inspected →handoff prepared. Время финальной записи отражено фактическим filesystem timestamp; длительность/approve не выдумываются.
