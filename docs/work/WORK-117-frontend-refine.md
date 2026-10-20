# WORK-117 — доработка веб-клиента TASK-006

- Автор: frontend_refine117, отдельный исполнитель от backend118 и reviewer119.
- Дата: 20.10.2026. Статус: реализация и авторские unit/integration PASS;
  независимое ревью и browser/process QA ведут отдельные участники.
- Проверенная база: `d542a09648d183034e8cedb6e0ddd05336082fd6`,
  ветка `codex/control-refinement`, незакоммиченный согласованный TASK006 delta.
- Область записи: только `internal/webui/**` и эта карточка.
- Прочитаны AGENTS, requirements, architecture, API, TASK005 и control contract.
  Paperclip reference `/private/tmp/accord-paperclip-reference` использован read-only:
  компактные нейтральные поверхности, sidebar hierarchy, command navigation.
  Новый raster/design asset или зависимость не добавлялись.

## Реализовано

- Нативная command palette: поиск задач реальным GET, переходы/создание,
  Cmd/Ctrl+K, стрелки, native dialog Escape/focus containment, возврат фокуса.
  Доступна с телефона; при отсутствии JS/dialog остаётся обычная GET-форма.
- Нативные быстрые представления задач: текущие/все/в работе/очередь/ревью.
  «Все» и overview total включают завершённые; ревью — роль `reviewer`,
  не вымышленное состояние. Layout сохраняется, cursor/старые фильтры сбрасываются.
- Эпики свёрнуты в native details, чтобы длинный каталог не вытеснял задачи.
  Компактные desktop task rows; длинные заголовки ограничены двумя строками,
  полный текст сохранён в ссылке и title. Улучшены metrics/dashboard hierarchy.
- Фильтры агентов: нативные kind tabs сбрасывают несовместимую роль;
  role selector предлагает только подходящие роли. Работает без JavaScript,
  строгий backend 400 не ослаблен.
- Tablet 768–1100: roster/detail/settings/toolbar не требуют desktop minimum.
  Mobile search работает и без JS; заголовки/действия/фильтры переносятся.
  `[hidden]` явно соблюдается независимо от author display declarations.
- Согласованное с backend118 opt-in автообновление агентов и событий: один
  polling loop выбирает `/web/data/dashboard` вместо `/web/data/overview`,
  второй запрос на тик не добавляется. Default — прежние overview counters.
  Отдельные observed_at, пределы 6 сессий/20 событий, последний хороший снимок,
  строгая schema validation; auth terminal/pause/abort/backoff сохранены.
  DOM создаётся только через createElement/textContent, ссылки валидируются
  перед построением. Коллекция с keyboard focus не заменяется; её timestamp
  остаётся прежним, status объясняет отложенное обновление.
- В карточке T-задачи — до 10 реальных событий и ссылка на полную историю.
  Для эпиков E-* такой reader не поддерживается и секция не показывается.
- Theme/drawer/poller/commands/dashboard разделены на самостоятельные JS assets;
  allowlist содержит точные MIME. CSS распределён по feature ownership:
  commands — только palette, shell — toolbar, sidebar — sidebar, cards — общие
  нативные tabs, control — dashboard/agents, tasks — task lists/epics.
  Изменённые текстовые assets/templates/helpers меньше 300 строк.

## Изменённые файлы

- JS: `assets/control.js`, новые `theme.js`, `drawer.js`, `commands.js`,
  `dashboard.js`; `control.test.cjs`, новые `commands.test.cjs`,
  `dashboard.test.cjs`.
- CSS: `assets/app.css`, `cards.css`, `shell.css`, `sidebar.css`, `control.css`,
  `tasks.css`, новый `commands.css`.
- Templates: `layout.html`, `icons.html`, `board.html`, `agents.html`,
  `overview.html`, `task.html`, новый `commands.html`.
- Go: `render.go`, `control_helpers.go`, `links.go`, новый `refine_test.go`.
  Пути выше относительно `internal/webui/`.

## Фактические проверки

- **PASS** `.local/toolchains/go/bin/go test ./internal/webui ./internal/web`:
  старые protected forms/escaping/allowlist и новые navigation/role-kind/
  отдельные timestamps/opt-in controls/scoped timeline assertions.
- **PASS** `node --test internal/webui/*.test.cjs`: 16/16 unit:
  прежние polling6, новые opt-in polling3, dashboard4, command palette3.
- **PASS** `git diff --check` на текущем совместном дереве.
- Новые process/E2E тесты автором **не создавались и не запускались**;
  browser actual-layout/native-dialog проверка — отдельный tester после approve.
- Зависимости, auth/CSRF/action IDs/idempotency, POST flows, leases и native API
  не изменены в этой области записи. Production/deploy/remote этим исполнителем
  не выполнялись.

## Границы

Сессия не выдаётся за OS process; execution/model/cost данные не выдумываются.
Полная Paperclip parity, запуск агентов и tenant isolation — будущие слои
с отдельными backend контрактами. Текущий worker остаётся одним проектом.
Визуальный/браузерный PASS на этой карточке не заявляется до внешнего QA.


## Дополнительное исправление по visual QA121

`assets/forms.css`: `#create` получает `scroll-margin-top: 128px`, чтобы
нативный переход к созданию задачи сохранял заголовок ниже sticky toolbar,
включая узкую noJS форму поиска. Остальной layout не менялся.
Авторские targeted Go webui и frontend unit16/16 — PASS; фактическую геометрию
заголовка после перехода повторно проверяет независимый tester.
