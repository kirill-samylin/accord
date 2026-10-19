# WORK-104 — интерфейс управления по Paperclip

- Родитель: TASK-005 / REQ-005. Роль: developer, профиль Давинчи передан путём и прочитан вместе с общим процессом/parallel-work read-only; модели именованных Claude-ролей в этой среде не закреплены.
- Начало: 19.10.2026, текущий авторский сеанс. Статус: реализация готова к независимому ревью, исходник заморожен для передачи координатору.
- Контракт: `control-web-contract.md` §8/9 с приоритетом §9; визуальный baseline — WORK-102, current streamlined Paperclip sources HEAD `5739a53f5f5e6cccbf21c7ae491ad06cb4b79b7c`. Снимки upstream не выдаются за пиксели текущего Accord.
- Границы: `internal/webui/**` и эта WORK. Backend/общие docs/лицензионное исследование — другие владельцы. Git, Tandem, VPS и Давинчи не изменялись. Tandem OFF согласно самостоятельному Accord AGENTS и подтверждению координатора.

## Результат

Реализован постоянный sidebar 240px, идентичность одного реального проекта, breadcrumb, компактные строки/карточки, статусы, фильтры, светлая/тёмная тема и мобильный drawer. Тема — реальное локальное предпочтение без токенов; при запрете storage действует до закрытия страницы. Drawer поддерживает Escape, Tab trap, фон и возврат фокуса; без JavaScript навигация остаётся обычным доступным списком ссылок.

Реальные страницы: обзор, список сессий агентов/людей и detail, журнал событий, один текущий проект, параметры сервера/правила и статистика. Статусы сессии не названы работающими процессами; LastEventAt — последнее записанное событие. Недоступные модели/провайдеры/ревизия явно неизвестны. Полный count только из Overview; roster6/activity20 — выборки со ссылками на полный SSR. Обзор не показывает не загруженную backend выборку задач: к реальному списку ведут карточки счётчиков.

Задачи имеют настоящий поиск q, список/доску `layout`, фильтры и сохранение контекста пагинации. Колонки относятся к текущей странице; drag/drop не изображён как действующая операция. Вопросы, карточки, ответ/возврат/явное продление, загрузка дизайна и история сохранены. `app.js`, защищённые POST actions, IDs и data selectors не менялись. Пустые/ошибочные состояния остаются серверными; ошибка не заменяется пустой очередью.

В `control.js` изолирован read-only poll только `/web/data/overview`: 15 секунд, visible+online, одна in-flight, timeout8s, abort, ограниченный backoff, сохранение последних значений при сети/429/503/невалидном JSON. 401 и 409/LEASE_LOST терминальны до явной навигации. BFCache pageshow восстанавливает свежий GET, но не снимает терминальную ошибку доступа. Меняются только `textContent` счётчиков и их дата; SSR roster/activity имеют отдельную статическую свежесть, формы/черновики/heartbeat не затрагиваются.

Статистика показывает настоящие токены, MissingTokens/долю и IncompleteData/число неполных событий. Стоимость/модель не выдумываются. Проекты и параметры не содержат фиктивных переключателей, создания проектов или настройки execution adapters. Ответ пользователя PC-01 «Подключение и контроль уже запущенных агентов» соответствует этой границе и не требует добавления управления запуском.

## Файлы и происхождение

- `render.go`, `control_helpers.go`, `links.go`: новые template names, безопасные helpers, query context и точный Static allowlist/MIME/nosniff.
- Templates: layout/icons/overview/agents/activity/settings/stats, board/list/kanban и вынесенная create форма; прежние task/designs/history переформатированы и используют общий shell/styles.
- CSS разделён по base/cards/forms/sidebar/shell/control/tasks; текстовые исходники ниже300 строк, без уплотнения объявлений. Цвета, радиусы, размеры и геометрия адаптированы из источников WORK-102; новых brand gradients/illustrations/анимаций для украшения нет.
- `InterVariable.woff2` скопирован напрямую из reference `ui/public/fonts/InterVariable.woff2`; local embed, CDN нет. Полные Paperclip MIT и Inter SIL OFL скопированы из согласованных root docs в embedded `license-paperclip.txt`/`license-inter.txt`, доступны в footer бинарной поставки. Проверка/сохранение официальных notice — root.
- `icons.html`: вручную составленные статические геометрические SVG paths Accord; icon libraries, Lucide и сторонние SVG assets не копировались. Пользовательские SVG/paths не принимаются.
- `control_test.go` — render/query/static regression unit tests; `control.test.cjs` — чистый unit harness JS через Node VM без браузера и сервера.

## Фактические проверки

PASS 19.10.2026:

- `.local/toolchains/go/bin/go test ./internal/webui`: existing protected forms/rendering + новые страницы/экранирование/полные counts/неполные токены/list-board/agent detail scope/asset MIME.
- `node --test internal/webui/control.test.cjs`: 6 unit tests — valid snapshot, failure/type/shape retention+backoff, 401/409 terminal через BFCache, hidden/offline/restoration, local expiry, single-flight/timeout abort.
- `node --check internal/webui/assets/control.js`: синтаксис.
- `gofmt` для затронутых Go файлов; `git diff --check -- internal/webui` после удаления лишней пустой строки в конце board template.
- SHA256 Inter font совпал с reference: `693b77d4f32ee9b8bfc995589b5fad5e99adf2832738661f5402f9978429a8e3`.

Первый прогон нового assertions обнаружил только чувствительность теста к форматированию HTML вокруг счётчика. Проверка нормализует переносы строк; финальный прогон PASS. Предварительные замечания независимого reviewer106 до freeze учтены: scope agent pagination, отсутствие незагруженной overview.Tasks, BFCache resumption. Это не итоговый approve автора.

Для выбора браузерных API сверены документации [inert](https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/inert), [pageshow/BFCache](https://developer.mozilla.org/en-US/docs/Web/API/Window/pageshow_event), [AbortController](https://developer.mozilla.org/en-US/docs/Web/API/AbortController). Это проверка контракта API, а не browser QA.

NOT-RUN: новые process/browser E2E до code approve, actual UI screenshots/независимый visual verdict, полный make check (один раз у root), deployment/HTTPS/release. Дизайн-approve и code/security review выдаются отдельными агентами. Проверенная автором версия — рабочий diff UI104; координатор создаёт общий frozen commit, SHA автор не назначает.

## Передача

Нужны независимые code/security review общего frozen состояния, затем tester browser/process сценарии и дизайн-ревью desktop/mobile/light/dark/long labels/keyboard/empty/error/offline/два sessions. Особое внимание: старые fenced формы/400 retry/409 draft и отсутствие продления от polling. После новых исправлений повторяются затронутые unit tests и актуализируется review.
