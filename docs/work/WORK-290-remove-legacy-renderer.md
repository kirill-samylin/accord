# WORK290 — полное удаление Go HTML интерфейса

Исполнитель release_prepare. **IMPLEMENTED / Go tests PASS / runtime NOT-RUN**.
Поручение root основано на прямом требовании пользователя полностью удалить
упрощённый интерфейс `/web/login` из кода. Область: Go `cmd/agent-board`,
`internal/web`, `internal/projects`, полный `internal/webui` и затронутые Go
тесты; эта карточка. Next/Makefile/общую TASK и production исполнитель не менял.

Прочитаны requirements, architecture и next-web. Историческое сохранение SSR
для совместимости заменено новым прямым поручением: единственный UI — Next
`/app`; JSON browser API/auth и binary/native endpoints остаются серверными.
Root согласовал архитектурную границу до реализации. Source review выполняет
отдельный review_changes; новые process/E2E после approve — отдельный tester.

## Реализация

Удалён весь `internal/webui` — 40 отслеживаемых файлов: renderer, embedded HTML
шаблоны, JS/CSS/локальные UI assets и SSR-only Go/JS tests. Скрытого renderer для
старых тестов нет. `web.Renderer`, `Renderer`/`StaticHandler` options, registry
Renderer/Static options и wiring/инициализация из production `serve` и project
runtime удалены. Конструктор browser требует board и прежние безопасные
origin/storage options; слабее auth/storage guard от этого не становится.

Known GET/HEAD `/web/**` без предпочтения JSON выполняют stateless303 к
фиксированным соответствующим `/app/**`. `/web/login` ведёт на `/app/login`,
scoped login также на общий login. Scoped page сохраняет canonical `p_32hex`,
query и область проекта; catalog detail требует canonical ID и не становится
scoped catalog. Task/epic/session/profile IDs проверяются по canonical форме.
Automation host-scope global/scoped сохраняется. Навигация не читает browser DB,
не выдаёт nonce/cookie и не требует действующего login. Неизвестные page paths,
удалённые static assets, traversal/decoded slash/некорректные scoped paths —404;
внешний URL не может стать Location. Данные и binary transfers не редиректятся.

Единый explicit JSON Accept parser с q-priority используется и Go browser, и
registry broker. JSON page projection и logical `/web/...` redirect envelopes
сохранены, включая operation-bound action IDs/CSRF/draft/exact replay/unknown
result semantics. Non-JSON POST после Origin boundary отклоняется406 до чтения
body/парсинга/мутации; foreign Origin сохраняет403. Для существующих JSON
clients bounded draft при Origin error сохраняется. Auth, ACL/epochs/leases,
CSRF, idempotency, native/data/download/preview handlers не заменялись другим
ядром правил. У binary ошибок остаются исходные401/403/404, generic error body
не обращается к renderer и не превращает auth error в406.

## Тесты и проверка

SSR fake Render hooks в Go fixtures заменены test-only observer фактического
public JSON response. Старые security/business/retry assertions работают на
реальной JSON projection; successful mutations проверяют200 JSON вместо303 HTML
form response. Observer не добавляет production callback/renderer abstraction.
Design validation/uncertainty fixture также читает настоящий JSON envelope.
Cmd/project startup tests создают browser без renderer.

Добавлены meaningful unit/integration regressions:

- `internal/web/navigation_test.go`: known GET/HEAD fixed redirects, canonical
  scope/catalog IDs, global/scoped host-scope, query preservation,
  unknown/traversal/static404; JSON login по-прежнему выдаёт nonce.
- `root_navigation_test.go`: stateless navigation даже с revoked cookie и
  закрытой browser DB, без нового cookie.
- non-JSON mutations не читают body, не меняют cookies/Location и не выполняют
  logout; Origin403 сохраняется, subsequent JSON session остаётся валидной.
- `internal/projects/browser_navigation_test.go`: те же registry routes через
  actual transport, q0/high-HTML negotiation, неизменный COUNT preauth,
  canonical/unknown404; scoped/catalog/login/design POST406 и foreign403 доbody.
- `browser_upload_test.go`: real ServeHTTP download/preview anonymous/revoked
 401 при Accept empty/JSON и foreign Origin403; прежняя actual upload/replay/
  byte-for-byte binary/deadline coverage сохранена.

Финальные targeted `go test ./internal/web ./internal/projects ./cmd/agent-board`
— PASS (web5.660s/projects6.043s/cmd0.572s). Полный `go test ./...` — exit0/PASS:
12 пакетов с тестами, core без тестов; acceptance27.083s, store14.216s,
worker12.254s, bridge11.787s, web9.636s, projects9.623s. Некоторые независимые
пакеты могли использовать штатный Go cache. Log в ignored
`.local/task019-remove-legacy-ui/backend/go-full.log`. Использован pinned Go1.27.2
и private normalized TMPDIR: начальная попытка default macOS TMPDIR отказалась
на прежнем unsafe web parent guard; guard не ослаблялся. Промежуточные ошибки
миграции fixtures/Origin draft/navigation и их логи сохранены отдельно.
`gofmt` и `git diff --check` — PASS. Новые production/helper files до300строк;
JSON fixture helper выделен в отдельный54строчный test file, backend fixture264.

Проверенная версия — незакоммиченный source delta поверх
`bdcabbd9e97f3aa20ae32aeed1853529be1130dd`. Это не новый установленный релиз.
Новых runtime/SSH/HTTP/process/E2E/native/commit/push/deploy действий исполнитель
не выполнял. Race/final independent source review/process migration/полная
приёмка и поставка остаются отдельными проверками координатора/исполнителей.
Windows/Claude execution/второй физический ПК этим изменением не проверялись.
