# WORK-116 — архитектурный аудит TASK-006

- Роль: отдельный read-only архитектурный аудитор, `/root/architecture_audit`.
- Дата: 20.10.2026. Проверенная база: `d542a09648d183034e8cedb6e0ddd05336082fd6`.
- Запись ограничена этой карточкой; исходники, общая TASK, runtime и deploy не менялись.
- Прочитаны фактический AGENTS, requirements/architecture/api/control-web-contract,
  TASK-005, WORK-100, core/store/web/webui и reference Paperclip из
  `/private/tmp/accord-paperclip-reference`.

## Вывод и предложение координатору

Текущий проект уже имеет полезное разделение `core → store` и независимые
`transport`, browser BFF `web`, presentation `webui`. Массовая смена Go на
TypeScript или перенос всего клиента в React сейчас увеличат объём регрессии
auth, CSRF, idempotency, drafts и deep links; сами по себе они не дадут дизайн
Paperclip или first-class agents/projects. Для текущей поставки сохранить
Go `net/http` + `html/template`, embedded self-hosted CSS/JS и SQLite WAL.
Зависимости в go.mod не требуют обновления ради поставленного UI-результата.

React оправдан следующим решением при появлении сложного общего клиентского
состояния, optimistic coordination или execution screens. Возможна постепенная
встройка отдельной интерактивной области через тот же browser BFF. Это опция,
не обязательный преждевременный отдельный frontend/runtime и не обещание уже
реализованной React-архитектуры. Визуальные tokens/layout/components Paperclip
переносятся независимо от framework.

Официальные источники проверены 20.10.2026:
[html/template](https://pkg.go.dev/html/template) описывает contextual escaping;
[React](https://react.dev/learn/add-react-to-an-existing-project) поддерживает
постепенное добавление к существующему проекту;
[SQLite](https://www.sqlite.org/whentouse.html) описывает ограничения concurrent
writes и условия выбора client/server DB. Проверка не является рекомендацией
обновить компилятор/библиотеки до новых версий.

## Конкретный доступный функциональный объём

1. **Карточка с реальной активностью задачи.** `RecentEvents(TaskKey)` уже
   существует. Сейчас `read()` task branch делает Get/taskActions, а detail
   показывает attempts/comments/evidence, но не immutable event timeline.
   Добавить DESC events с pagination и сохранением task context, получить
   Paperclip-style activity без новых таблиц и без ослабления core.
2. **Полноценная карточка эпика.** Сейчас тот же generic task detail показывает
   эпик, но не его дочерние задачи. `ControlTasks(EpicKey)` даёт существующий
   источник; отдельный child list с явной пагинацией и current/all фильтром
   делает эпик полезной группировкой. Это не project/tenant.
3. **Согласованные фильтры участников.** Не предлагать форму, которая сама
   генерирует серверный validation error; см. ниже.
4. **Paperclip density.** На списке задач использовать компактные rows,
   role/state/priority и владельца; на detail отделить content от facts panel;
   создание оформить отдельным раскрываемым блоком/диалогом с рабочим SSR
   fallback. Сохранять реальные existing forms/action IDs/drafts.

Новый browser mutation для комментариев нельзя делать как «любой может писать»:
`Store.Comment` требует owned live lease/generation и продлевает её. Update и
Link также ограничены requester/coordinator или holder. Возможный следующий
slice должен явно отражать эти права, а не включать generic chat по примеру
reference. Подмена human session ролью coordinator недопустима.

## Подтверждённые source findings

| Находка | Место | Результат и безопасная правка |
| --- | --- | --- |
| Ролевая форма генерирует несовместимые фильтры | `internal/webui/templates/agents.html:4–24`, `internal/web/control_queries.go` | На default kind=agent виден human; его выбор даёт400. Для kind=human видны developer и остальные agent roles, также400. Рендерить совместимые варианты и reconcile role при смене kind; серверную проверку ручного invalid URL сохранить. |
| «Всего задач» открывает список только текущих | `internal/webui/templates/overview.html`, metric TasksTotal href `/web/tasks`; `internal/web/pages.go` IncludeClosed=false по умолчанию | Полный counter и открываемый набор расходятся при done/cancelled. Ссылка полного count должна включать `include_closed=true`; current count/label не выдавать за total. |
| Одна транзакция/весь board на каждое чтение | `internal/store/store.go:transact`, `internal/store/persistence.go:loadBoard/saveBoard` | Read берёт write lock и перечитывает/сериализует все tasks/sessions; это реальный scaling bottleneck, не доказанный runtime FAIL. Не добавлять polling на каждую карточку. |

Первые два пункта подтверждены цепочкой source/form → query validator/store,
но собственный browser reproduction не выполнялся. Это не независимый final
design review и не полный поиск всех визуальных багов.

## Архитектурная уборка и масштабирование

- Сохранить доменные правила в Store; DTO в core, query builders и browser auth
  в web, HTML/CSS/JS в webui. Не подключать browser cookies к native `/api` или
  `/mcp`: строгая Origin-модель сохраняется.
- Объединить query/context helpers и исключить дублирование state/role labels,
  допустимых query keys и ссылок при добавлении новых страниц. Делить handlers
  по реальному use case; большинство текущих файлов уже меньше300 строк.
- Следующий backend этап — измерить нагрузку, выделить immutable event readers
  и индексируемые projections; expiry-materialization оставить транзакционной.
  Переход PostgreSQL допустим по подтверждённой concurrent-write потребности,
  не как косметическая предпосылка хорошего UI.
- Для будущих проектов сначала registry stable project ID + server resolution,
  ACL и изолированные board/DB/webDB/blob roots. Затем scoped session/action/
  cursor/cache namespaces и negative isolation tests. Epic/string label и
  произвольный browser endpoint не заменяют эту границу. Browser выбирает лишь
  разрешённый ID; канонический origin и credential secretRef задаёт оператор
  на сервере. Не принимать upstream URL/token из формы и не делать universal
  proxy. Упрощённый первый registry может отдавать только operator-managed
  links на независимые origins с собственным входом, без передачи credentials.
- Persistent agent profile отделить от session: один profile может иметь много
  исторических сессий. Текущие status/owner/lease показывать честно; provider,
  model, process state остаются неизвестными до отдельного источника данных.
- CSS tokens/components и route/read DTO нужны как extension seams уже сейчас;
  новая multi-tenant schema, workers и execution не включаются молча.

## Проверки и ограничения

**PASS — source audit:** прочитаны указанные контракты и реальные зависимости;
сопоставлены SSR маршруты/readers/forms с Paperclip UI dependency/components;
официальные API источники открыты; рабочий diff до записи сохранялся.

**NOT-RUN:** unit/integration/process/E2E/browser/load/deploy. Исходники не
менялись, новые тесты этой ролью не писались. Удалённые ресурсы, credentials,
Tandem и соседний Давинчи не использовались. Память дала только историческую
ориентацию Accord/Paperclip; все выводы проверены текущими исходниками.

Решение о составе TASK-006 и принятие предложений принадлежит координатору.

Уточнение координатора после аудита: opt-in dashboard refresh может одним
запросом на tick читать те же три bounded источника SSR (overview, шесть
сессий,20событий). Default остаётся исходный counter polling; отдельных
независимых pollers не добавлять. Это допустимый bounded UI-контракт, но не
устраняет full-board load/write-lock каждого reader. Новая пропускная
способность, SLA и соответствие нагрузочным требованиям не доказаны.
