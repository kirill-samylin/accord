# WORK-105 — ранний security-контракт control UI

- Родитель: TASK-005 / REQ-005; исполнитель `/root/control_security105`, роль security.
- Статус: completed, проектирование; **допустимо реализовывать по условиям ниже**.
- База `a6501d8621abba558606b7fc26d7407fb3b2065e`; при чтении backend103/UI104
  продолжают работу. Это не approve текущего diff, приложения или выпуска.
- Запись: только этот WORK и `/private/tmp/accord-control-099/security/`.
- Прочитаны AGENTS Accord, общий процесс/параллельная работа/security-профиль
  Давинчи read-only, REQ/TASK-005, `control-web-contract.md` §1–9, SEC-086 и
  итоговое предыдущее security-review091; существенные выводы сверены с source.
- Tandem OFF по прямому правилу Accord. Source/Git/deploy/SSH/Tandem/Давинчи
  не изменялись. Именованные Claude-модели недоступны; модель унаследована.

## Границы доверия и вердикт

Активы: browser credential, текущая authorisation, board sessions/аренды/generation,
ответы/задачи/immutable события, доступность SQLite и прежних native клиентов.
Недоверенные входы: HTTP/cookie/query/cursor/session ID, project label, owner,
machine/run ID, task title/description, содержимое событий. Сеанс другого агента
в списке — наблюдаемые данные, а не actor/capability браузера.

Действующий pilot допускает авторизованному участнику чтение общего board;
новый roster/search/activity не вводит tenant/admin ACL. Owner и epic — фильтры,
не границы полномочий. Session ID/token ID не заменяют cookie/bearer. В этой
поставке один настоящий проект; runtime adapters/shared login/registry — future.

**Блокирующих пробелов проектирования P1/P2 не найдено.** §8/9 уже задают
необходимые границы; ниже они связаны с последствиями и negative AC для авторов
и будущего независимого reviewer. Условная серьёзность не является заявлением
об обнаруженной уязвимости незавершённого кода. Новых продуктовых gates нет.

## Условия и проверяемые отрицательные сценарии

| Граница / риск при нарушении | Обязательное поведение | Проверка реализации |
| --- | --- | --- |
| C105-01, P1: чтение после отзыва или от чужого session ID | Каждый core reader проходит `readActor` внутри `transact`; browser каждый раз проходит `loadBrowser`/`AuthenticateID`, actor берётся только из сохранённого browser session. Параметр agent detail не становится actor | Без cookie, после logout/revoke/token expiry, с cookie другого credential и чужим board session чтение/JSON запрещены; подмена detail SID даёт только разрешённый просмотр. Истёкшая board session возвращает существующий error code, без автоматического продолжения |
| C105-02, P1: JSON обходит периметр или раскрывает session state | Единственный poll `/web/data/overview` под общим Host/admission/no-store/nosniff, без CORS; GET после browser auth + core session check. JSON — только `ControlOverview`, без browser session/CSRF/actions/bearer/hash. Ошибка non-2xx, `application/json`, стабильный code; нет login redirect или HTML200 | Native API/MCP/metrics с любым присутствующим Origin по-прежнему403; cookie не авторизует native. Плохой Host отвергнут. Auth401 и LEASE_LOST для истёкшей board session (409) не содержат успешного DTO. POST/unknown query не становятся чтением/мутацией. Cookie flags/rotation не меняются |
| C105-03, P1 при authority bypass, P2 для ложного active: отзыв чужого credential скрыт за TTL | `credential_invalid` имеет приоритет над TTL; validity batch из tokens в той же транзакции. Overview и roster используют один effective predicate. Lease badge требует valid credential + live session + live owned lease + matching task generation | Два runs одной роли видны отдельно. Отозвать token A при живой session/lease; читать от B: A не в active count/default list, explicit history A=`credential_invalid`, `Task=nil`, leased count снижен. То же для expired/missing credential. `TasksInProgress` остаётся workflow фактом. Native Sessions TTL semantics не меняется |
| C105-04, P1: фоновые чтения продлевают полномочия; P2: poll создаёт unbounded work | Poll не вызывает `page`, `issueAction`, `Describe`, `ensureSession`, `Heartbeat`; остальные страницы SSR, Stats не poll. 15s, один in-flight, timeout/abort, hidden/offline stop, 429/503 backoff; никаких card pollers | Frozen clock: многократный GET не меняет expiry/session count/generation/lease/action rows. Core materialization уже истёкшей lease допустима, это не renewal. 401/LEASE_LOST останавливает poll, без auto-continue POST; возвращение visible/online делает один refresh. Failed/aborted/HTML/malformed JSON не затирает last-good snapshot, timestamp и draft |
| C105-05, P2: тяжёлый/неоднозначный query, неверная выборка или injection | Allowlist параметров, отказ duplicate/unknown/malformed значений; bounded SID/cursor/q, q <=200 runes/1024 bytes. Search literal case-insensitive до pagination. SQL с bound args, user input не формирует SQL/HTML. Cursor binds normalized filters; recent DESC keyset с SQL filters до LIMIT+1 | Bad SID, oversized cursor, отрицательный/overflow decimal cursor, incompatible kind/role, изменённые filters со старым cursor дают400. `%`, `_`, кавычка и HTML в q — буквальный поиск/текст. Mixed-case кириллица и match за первой страницей находятся. Concurrent append не создаёт skips/duplicates recent pages; native ASC не меняется |
| C105-06, P1: XSS в новых поверхностях | `html/template` contextual escaping; все labels/events/query — plain data. DOM updates через `textContent`; без raw Markdown/HTML, `innerHTML`, eval или conversion data к `template.HTML/JS/URL`. URLs строятся сервером из фиксированного route + escaped identifier/query | `<script>`, quote/attribute break, `<img onerror>` в title/description/owner/machine/run ID/project/event/query выводятся текстом в SSR и после poll. Никаких browser commands, external URL fetch/iframes или token в URL/storage/errors. Existing answer/release/CSRF/action-ID paths и явный editor heartbeat работают без изменения draft |
| C105-07, P1 для active-content upload обхода, P2 для supply chain: иконки/шрифты расширяют поверхность | SVG icons допустимы как доверенные static template literals/allowlisted icon names; не user SVG, URLs или разметка из events. Embedded static handler разрешает точные filenames, не directory/blob serving. Inter self-host; CSP без unsafe-inline/eval/CDN; uploads сохраняют attachment-only active content | Неизвестный static path/traversal не выдаёт произвольный файл, нет external font/script requests. Reference source MIT notice сохранён; Inter — исходный font asset с SHA/provenance и OFL notice/text по §9. Если добавятся copied icon assets/code, фиксируется их actual provenance/license, не предполагается покрытие Paperclip MIT |

GET не требует нового CSRF-токена/Origin ради чтения: это не новый mutation
endpoint. Защита состоит в существующей cookie/auth, no-CORS, no-store и
сохранённой semantics read. Нельзя ослаблять прежние exact Origin/CSRF guards
для login/logout/ответов/upload/heartbeat ради новой панели. Token ID и SID могут
быть разрешённой provenance общего board; запрещены именно secrets/digests,
и использование ID как полномочия. Signed cursors/enterprise RBAC не нужны
как новые gates для нынешнего одного board; cursor не является ACL.

## Future ADR — отдельно от реализуемой безопасности

`control-web-contract.md` §6 задаёт соразмерный последующий путь: отдельные board,
DB/webDB/blobroot, credentials и HTTPS origin; controlled navigation manifest,
без server-side arbitrary URL fetch и переносов token. Для shared login нужны
явные project ACL до выбора Store и отдельные namespaces для actions/cursors/
cache/download/idempotency. Одинаковые task keys/session IDs разных проектов
должны входить в будущие isolation tests. Epic/label/project dropdown этого
не реализуют. Эти tests не gate TASK-005, пока federation отсутствует.

Execution profile не даёт разрешения запускать OS command. PC-01 остаётся
продуктовым вопросом координатора; никаких arbitrary command/workdir/secrets/
browser-supplied endpoints, false running/cost/config success в текущей волне.
Не включать чужую D4/E/schema4 поставку ради формы. Это граница объёма, не
требование добавлять в UI предупреждения или фиктивные настройки.

## Доказательства, ограничения, следующий исполнитель

- PASS: read-only review согласованного контракта и опорного auth/store/web/
  transport/renderer source. `readActor` повторно проверяет credential + session;
  `sessionFor` связывает principal ID; `loadBrowser` повторно вызывает AuthenticateID;
  GET сейчас не вызывает ensureSession; общий transport до Extra защищает Host/
  admission/native Origin; browser CSP/no-store/nosniff действуют централизованно.
- Input manifest: `/private/tmp/accord-control-099/security/input-sha256.txt`.
  Это hashes прочитанных входов в момент ранней проверки, не freeze нового кода.
- Официальные API сверены 19.10.2026: [html/template](https://pkg.go.dev/html/template)
  (данные недоверенные, автор шаблона доверен), [net/http](https://pkg.go.dev/net/http#CrossOriginProtection),
  [ParseQuery](https://pkg.go.dev/net/url#ParseQuery), [Fetch](https://fetch.spec.whatwg.org/).
  Применяемые существующие API не требуют обновления зависимостей.
- NOT-RUN: targeted runtime/unit/browser/process/live/load/release tests. Это
  ранний security contract; source не frozen, тесты авторов и final review идут позже.
  Старый approve091 не распространяется на новое изменение. Нагрузочная стоимость
  SQLite transact/loadBoard известна контракту; V1 capacity этим не подтверждена.
- N/A: новая OAuth/RBAC/tenant/worker реализация, общий старый audit продукта,
  SSH/deploy, изменение backup policy/schema или новые UI security warnings.
- Передача: backend103/UI104 выполняют C105-01…07; координатор выделяет отдельного
  security reviewer на итоговый source/diff + hashes и targeted tests. Затем QA
  по процессу TASK-005; source/unit, browser/process и actual HTTPS evidence различать.
- Память использовалась только для ориентации Accord/Tandem; актуальный OFF и
  границы проверены по текущему AGENTS, исторический live status не использован.

## Журнал

- 2026-10-19T22:24:28+03:00 — completed: фактический `date` перед записью;
  условия и negative AC подготовлены, ранние P1-границы переданы авторам/root.
  Исходники не изменялись; независимый approve реализации не выдавался.
