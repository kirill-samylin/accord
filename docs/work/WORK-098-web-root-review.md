# WORK-098 — независимое ревью навигации через корень веб-Accord

- Роль: reviewer, отдельный от автора агент `/root/webrootreview098`.
- Статус: **выполнено — APPROVE** узкой дельты. Блокирующих замечаний нет.
- Проверенная версия: HEAD `63d6561` плюс незакоммиченные `internal/web/http.go` и `internal/web/root_navigation_test.go` (SHA256 ниже). Остальной ранее принятый код повторно целиком не проверялся.
- Область записи: только эта WORK. Реализация, тесты, Git, deploy, SSH, Tandem и Давинчи не изменялись.
- Прочитаны AGENTS самостоятельного Accord, профиль reviewer, общий процесс, parallel-work и code-structure Давинчи. Tandem OFF по прямому поручению самостоятельного проекта. Именованные Claude-модели недоступны; наследованная модель соответствует автору, независимость исполнителя сохранена.

## Вывод

Root GET/HEAD теперь проверяет текущую браузерную авторизацию существующим `loadBrowser`: действующая сессия ведёт 303 на `/web/`, отсутствие или отзыв авторизации — 303 на `/web/login`. Неавторизационные ошибки не подменяются формой входа: SQL/JSON errors проходят через существующий `fail` и дают 503. Обе цели постоянны и не зависят от входного URL, поэтому open redirect не добавлен. No-store и остальные общие защитные заголовки устанавливаются до ветвления.

Цепочка `loadBrowser` → SELECT browser_sessions → `AuthenticateID` → SELECT active token не пишет сессию, не вызывает Describe/heartbeat и не продлевает lease. Проверяются единственность cookie, длина, хеш, срок browser session и актуальный активный токен. Новая ветка не выдаёт cookie и не меняет безопасные cookie flags. Нативный API и web POST Origin/CSRF-проверки остаются вне дельты и сохраняют прежнее поведение. HEAD использует ту же новую ветку по явному условию; отдельно в тесте root HEAD не запускался.

Новый тест проверяет наблюдаемые статус/Location анонимного, вошедшего и отозванного посетителя, затем закрывает web DB и проверяет 503 вместо ложного перехода на login. Это регрессия исходного ошибочного маршрута `/` → `/web/` → 401. Сценарий конечной формы после HTTP redirect и HTTPS/browser поведение проверяет координатор после нового build/deploy; здесь они не объявляются пройденными.

Переиспользован существующий auth/error path, существенных новых дублей и лишних абстракций нет. Файлы 183 и 34 строки, исключение по декомпозиции не требуется. UI-kit отдельного Давинчи неприменим: UI-компоненты и визуальный дизайн дельтой не менялись. Новые API/зависимости не вводились.

## Фактически выполнено

**PASS** локальным Go из `.local/toolchains/go/bin/go`, GOTOOLCHAIN=local, один узкий прогон:

```text
go test ./internal/web -run 'TestRootOpensLoginAndRetainsSignedInNavigation|TestGetDoesNotDescribeOrRenewAndContinueIsExplicit|TestBrowserOriginCSRFAndDraftBoundaries|TestBrowserCookieRotationRevokeAndLogout' -count=1 -v
```

Все четыре теста PASS, package `agent-board/internal/web` 1.042s. Также PASS `git diff --check` для двух проверенных файлов. Read-only сверены `loadBrowser`, `AuthenticateID`, `errorStatus`/`fail`, cookies и прежний POST auth/CSRF flow.

| Проверенный файл | SHA256 |
| --- | --- |
| internal/web/http.go | 53c21516f5f95e274b991c1de0aa90a312e7d3db6589dc4dbcc76e5922258bfe |
| internal/web/root_navigation_test.go | caed5b396c675d9305a2c79676a614698ddf088407c2edfb5cf5b5227305cae0 |

**NOT-RUN reviewer:** полный Go/race/E2E, новый release build/stage, actual HTTPS root→login browser smoke, live apply/finalize/rollback. Это границы данного code approve, не PASS выпуска. Предыдущие QA/backup/security approve относятся к своим версиям и не подменяют проверку нового выпуска.

## Передача

Координатор может зафиксировать именно эту дельту, пересобрать и stage новый чистый release, затем выполнить согласованный guarded apply и actual HTTPS/browser root smoke. Установка не считается завершённой до проверки конечной login страницы для свежего посетителя, действующей сессии и защищённости прежних native/web сценариев. Изменение проверенных файлов требует затронутого re-review.

## Журнал

- 2026-10-19T18:46:55Z — reviewed: узкая дельта и auth/error цепочка проверены, четыре регрессии PASS.
- 2026-10-19T18:46:55Z — completed: **APPROVE**, передано координатору для нового build/stage и guarded выпуска с отдельным runtime smoke.
