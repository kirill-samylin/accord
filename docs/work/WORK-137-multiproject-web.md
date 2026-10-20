# WORK137 — мультипроектный Next.js клиент

TASK009, 20.10.2026. Роль: frontend architect/developer. До отдельного root GO
владение только этой WORK; продуктовый код не изменялся. Root ведёт TASK009,
WORK135 готовит backend-контракт, WORK136 — независимую критику и review.

Прочитаны AGENTS, requirements/architecture/api, next-web, TASK008, существующие
Next screens/navigation/forms/query hooks и Go browserPage. Применён прочитанный
`vercel-react-best-practices`: cache isolation, AbortSignal, отсутствие лишних
waterfalls и сохранение explicit prefetch=false. Зависимости не обновлялись.

## Фактические ограничения текущего клиента

- `SessionView.project` — имя, не идентификатор. Browser principal пока содержит
  только owner/machine; это не ключ для управления членством.
- `lib/api.ts` принимает только `/app/...` и `/web/...` одного board; allowlist
  действий и безопасных redirect/download URL рассчитаны на один проект.
- Query keys без project scope: page, dashboard, command, create-options, peek,
  recent-tasks. Смена отображаемого имени не создаёт безопасной границы cache.
- Ссылки, forms, polling и бинарные URL используют общий `/web`. Нужна единая
  маршрутизация, включая downloads/previews и все mutation retry paths.
- `useAction` замораживает точный path/body для неизвестного результата. Это
  свойство необходимо сохранить при смене проекта; перенос action ID в другой
  board недопустим даже при одинаковом T-1.

## Предлагаемый UX и семантика API — ещё не принятый wire-контракт

1. Верхний Paperclip-style switcher показывает только реально разрешённые
   проекты с именем, признаком архива и текущим выбором. Переход открывает обзор
   целевого проекта; фильтры, курсоры и открытая задача прежнего не переносятся.
2. «Проекты» — настоящий серверный каталог. «Создать проект» сохраняет имя и
   ссылку на репозиторий, затем открывает выделенный проект. Нельзя создавать
   локальную карточку до server success или показывать чужой проект по имени.
3. Настройки проекта: основные сведения, подключённый репозиторий и участники/
   доступы. Редактирование выводится только по server capabilities; прямой POST
   всё равно обязан независимо проверяться сервером.
4. Для участников требуются stable opaque principal ID, display owner/machine,
   membership role и допустимые изменения. Одинаковые display names не должны
   сливать личности. Удаление последнего управляющего отклоняет backend.
5. Архивирование — явное действие с подтверждением. Архив не означает удаление
   данных; разрешённые чтения/мутации и восстановление должны быть определены
   backend-контрактом до UI. Статус не хранится только в браузере.
6. Repo connection отделяет сохранённую связь от проверки удалённого доступа.
   Без соответствующего worker/GitHub-контракта UI не пишет «доступ проверен».
   Пароли, токены и private keys не принимаются полем URL и не выводятся в DOM.

Требуемые данные: current project ID/name/status/membership/capabilities,
доступный каталог (либо отдельный authenticated endpoint), repository refs,
members и server-generated action IDs/CSRF. Требуемые действия: create,
repository update, membership grant/change/revoke, archive/restore. Точные
paths, DTO и ACL определяет canonical TASK009 после критики WORK136.

Предпочтение для стабильных deep links: `/app/p/{opaqueID}/tasks/T-1` и
`/web/p/{opaqueID}/tasks/T-1`. Старые paths остаются закреплёнными за прежним
default board. Проект в URL авторитетен; глобальный «active project» cookie
не должен менять смысл соседней вкладки или уже сформированного POST.

## Границы клиента

- Pure route helpers выделяют project ID и локальный screen path, принимают
  только canonical opaque IDs и allowlisted routes, не произвольный upstream.
  Все ссылки/redirects/forms/polling/downloads проходят согласованную границу.
- Отдельный QueryClient/remount boundary для project identity. Ключи включают
  project ID, даже когда путь уже scoped. Старые queries отменяются, прежние
  payloads/placeholderData/драфты не появляются в новом проекте. Локальный
  вид/тема может сохраняться; private data в localStorage/sessionStorage нет.
- Отложенный ответ проекта A не меняет B и не вызывает в B навигацию/очистку
  новой identity. Unauthorized notification обязана учитывать исходный scope;
  membership denial не превращается в глобальный logout.
- Во время sending/uncertain switcher запрещает UI-переключение с понятным
  объяснением и возможностью retry именно старого действия. Captured mutation
  path/body/action ID не переписывается текущим project context. Ручной уход
  со страницы не может перенаправить уже отправленный запрос в другой board.
- После успешного переключения Shell, search, quick peek, create/upload forms,
  cursors и recent-tasks state сбрасываются. Все28 ссылок сохраняют отсутствие
  автоматической viewport/hover предзагрузки из TASK008.
- Project-qualified binary URLs проверяются по ожидаемому текущему project ID;
  чужой download/preview не отображается как текущий. ACL проверяет сервер.

## Предлагаемое владение после root GO

`web/src/**`: route/context/API/query helpers и их unit tests, project switcher,
catalog/settings/forms, обновление существующих links/screens/mutations.
Новый код делить по ответственности; Shell305 уже имеет зафиксированное
исключение TASK008, выделение project switcher поможет сократить этот файл.
Go/infra/TASK/Git/SSH не редактировать. Backend DTO согласовывать с WORK135.
Новых process/E2E автор не пишет: отдельный tester после independent APPROVE.

## Предлагаемая приёмка

Unit автора: qualified route/redirect allowlist и traversal, collision T-1 между
двумя IDs, response scope mismatch, сохранность captured replay path/body,
download/preview project boundary, project-aware mutation deadline.

После review отдельный tester: создание второго проекта/репозитория, switch/
reload/back/deep links, две вкладки разных проектов, ACL grant/revoke/archive,
поздний ответ A после перехода B, unknown-result retry только в A, cache/draft/
cursor/search/preview isolation и default migration с прежними данными.
Используются отдельные synthetic fixtures, реальные Go/Next/Chrome, ошибки
не маскируются empty/success. Физическая вторая машина этим не доказывается.

## Реализация после root GO

Root принял WORK135 и разрешил frontend implementation. Принятый wire:
`/app/p/{p_32hex}/...` → `/web/p/{id}/...`, глобальный каталог `/web/projects`,
детали `/web/catalog/projects/{id}`, серверные create/switch/update/repositories/
members и registry CSRF/action IDs. Create передаёт `repository_url`, repository
add/update — `url`; switch target — полный безопасный scoped `/web/p/{id}/...`.
Core Session не подделывается для глобального каталога.

Реализованы реальный переключатель проектов, каталог, создание, переименование,
архив/восстановление, ссылки и версии репозиториев, роли owner/editor/viewer,
выдача/изменение/отзыв доступа по stable identity ID. Репозиторий явно описан как
сохранённая связь, без обещания клонирования или проверенного удалённого доступа.
При deep link без binding показывается явное «Открыть проект» с POST switch.
Старые URL остаются закреплены за прежним проектом.

Клиент получает отдельные QueryClient/remount boundaries для каждого проекта и
каждой страницы настроек каталога; secondary queries, polling, поиск и peek
содержат scope. При уходе отменяются чтения и очищается cache; late401 прежнего
scope не сбрасывает новый. Ответ с чужим project ID отвергается до query/cache.
Просмотрщик и архив не получают доступные write controls. Все binary URLs
проверяются по текущему проекту. Mutation endpoint/body заморожены; отправка или
неизвестный исход блокируют переход, повтор сохраняет точный прежний payload.
Project-qualified upload сохраняет 130-секундный deadline прежнего upload.

Согласованное замечание review136: отсутствие Project на rejected 4xx больше
не означает unknown outcome. Unqualified error page не сливается в state;
публичные field errors/actions сохраняются, подтверждённый FORBIDDEN блокирует
новые отправки. Чужой ID и success page без ожидаемого ID по-прежнему отклоняются.
Backend135 подтвердил Project для разрешённых validation4xx и отсутствие такой
страницы при отказе авторизации/членства. Добавлена regression для обоих случаев.

## Изменённые файлы и владение

- `web/src/lib/{projects,project-routes,project-assets,catalog-action}.ts` — DTO,
  маршруты, бинарная граница и snapshot registry actions; обновлены `api.ts`,
  `types.ts`, dashboard API/navigation и frontend unit tests.
- `web/src/features/projects/*` — scope/context, catalog action/form, switcher,
  каталог, детали, репозитории, участники, явное открытие проекта.
- `web/src/components/project-link.tsx` и существующие ссылки/screens/hooks/
  mutations — единая qualified navigation, ACL flags и scoped reads/actions.
- `web/src/features/{control-app,board-screens,shell}.tsx`, appearance hook,
  `web/src/app/styles/projects.css` и import в globals — отдельный catalog layout
  без вымышленной Session, существующий Paperclip-style shell сохранён.
- `docs/work/WORK-137-multiproject-web.md` — эта карточка. Go/infra/shared TASK,
  зависимости, Git, VPS и реальные данные этим исполнителем не изменялись.

Новые модули до 300 строк; Shell сокращён до 274, ControlApp — 262. Существующие
cohesive list screens tasks308/designs307 оставлены вместе с row/list rendering:
здесь только scoped link/import, readonly visibility и asset guard; разделение
ради 7–8 строк затронуло бы несвязанный list presentation. Это ограниченное
исключение требования «стремиться к 300 строкам».

## Проверки и статус

Версия: локальный frontend source TASK009/WORK137 поверх рабочего checkout root,
20.10.2026; frontend commit/релиз этот исполнитель не создаёт. Node24 из
подготовленного runtime, существующий Next.js16.4.2, зависимости не обновлялись.

- PASS — Prettier check, TypeScript noEmit, 53/53 frontend unit tests.
- PASS — production Next build с локальным loopback `ACCORD_GO_ORIGIN`.
- Unit regression: одинаковый T-1 разных проектов, traversal/foreign routes,
  scoped assets, чужой board/catalog response, definitive scoped400/403,
  readonly flags, exact retry endpoint/body, aborted late body, registry
  capability snapshot и scoped upload deadline.
- NOT-RUN — совместный Go/Next runtime, браузер/ACL/migration/two-tab acceptance,
  VPS rollout и физическая вторая машина. Это последующие этапы root/отдельного
  tester; frontend unit/build не являются их доказательством.

Приватный лог: `.local/task009-frontend-final-check.log` (не для Git). Независимый
review136 получает frozen frontend; новые process/E2E автором не создавались.

### Bounded review fix: архив исходного проекта

После review136/root принят дополнительный server capability `can_archive`:
owner, active и не default. ProjectDetails показывает архивирование только
при `can_archive === true`. Для исходного проекта вместо неработающей кнопки
выведено объяснение сохранения прежних данных. Восстановление archived проекта
остаётся доступно управляющему независимо от `can_archive`; rename/repositories/
membership и ACL visibility не менялись. Client DTO дополнен обязательным bool.
Повторные format/typecheck/53 unit — PASS; браузерная проверка — NOT-RUN.
