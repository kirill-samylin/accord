# WORK292 — independent review удаления прежнего HTML UI

Reviewer: review_changes. Область записи — эта карточка и ignored
`.local/task019-remove-legacy-ui/review/`. Product source, чужие карточки и
production reviewer не изменяет. Общую TASK ведёт root.

Прочитаны актуальные requirements/architecture/next-web/browser contracts,
владельцы HTTP/JSON/catalog/download handlers и relevant Next/browser tests;
применяется ранее прочитанный vercel-react-best-practices.

## Уточнённый контракт TASK019

По прямому текущему поручению no-JS HTML UI удаляется из исходников и сборки;
старое обещание SSR compatibility в next-web.md больше не актуально. Однако
Next использует /web JSON как browser backend — эти endpoints, cookies,
CSRF/action IDs, ACL/epochs, leases и exact unknown retry сохраняются.

Root подтвердил финальный вариант: известные non-JSON GET/HEAD legacy pages
дают stateless303 на соответствующий /app без nonce/session mutations; scoped
и catalog URLs/query сохраняют правильный проект. Старые assets404.
Non-JSON POST при допустимом Origin получает406 до чтения body/мутации;
foreign Origin403 сохраняет прежний guard. Binary design download/preview
GET/HEAD и /web/data JSON endpoints сохраняются с прежней auth/ACL.
JSON GET/HEAD/POST использует прежние представления/статусы. Accept negotiation
должен быть одинаковым в single-board и registry ветках. Unknown routes404.

## Предварительное review новой кнопки файла

**Static APPROVE** для LoginForm.tsx + LoginForm.module.css автора
validate_candidate: отдельная доступная клавиатурой кнопка, live status имени
файла, перенос длинного имени, замена/удаление и возврат фокуса. Hidden native
input сохраняет type=file/name=token_file/FormData. Удаление native required
обосновано hidden input: прежние explicit missing/empty/4096 checks остаются.
CSRF/multipart/immutable unknown retry и disabled fieldset не ослаблены; JS не
читает содержимое файла, не сохраняет имя/токен в storage. CSS scoped, использует
существующие theme variables. Actual browser picker QA остаётся отдельным обязательным postapply gate.

## Retired tests / сохранение значимого покрытия

SSR-only browser.cjs/control*/refinement* suites допустимо удалить вместе с
реальным UI и retire их README команды. Shared harness/lifecycle/state-hash и
Next suites сохраняются. Next security/transport/assets и Go JSON tests уже
покрывают CSRF/Origin/revocation/races/leases/stale drafts/idempotent unknown
retry/native session/download isolation. upload-process.cjs сохраняется и
мигрирует на JSON после source approve: direct Go50MiB/overlimit/120s cleanup
отличается от аналогичного Next proxy proof. У next-security.cjs найден stale
heading «Добро пожаловать» — current UI использует «Войти»; сообщено root.

## Найдено в промежуточном backend source

- **P2 source blocker:** raw binary design route не оборачивается JSON writer;
  новый render fallback всегда406 теряет401/403 из loadBrowser/auth failure.
  Нужен generic no-HTML error с исходным error status и ServeHTTP regression
  binary unauth/revoked запросов. Передано backend автору/root.
- Navigation: scoped login генерировал запрещённый Next scoped login; catalog
  detail допускал неканонический project ID/scoped catalog; известная страница
  automation/host-scope отсутствовала в mapping. Передано автору для bounded
  fixes и regressions.

## Финальное source review

**APPROVE / PASS** текущего product/unit source после исправлений автора.
Binary fallback теперь возвращает generic body с исходным error status401/403,
успешный download/preview сохраняет bytes/headers. Scoped login ведёт только на
общий /app/login; catalog/profile/session IDs canonical, scoped catalog/unknown/
traversal/static404; global/scoped host-scope/query сохранены. Registry использует
тот же Accept parser. Новые registry regressions проверяют realtransport routes,
GET/HEAD безcookies/nonce и q-negotiation,406unreadbody/foreign403 и живую сессию
после отвергнутого logout.

Reviewer лично повторил5targeted web tests и2registry tests — PASS. Авторский
full Go log exit0/PASS прочитан:12packages с тестами; targetedweb/projects/cmdPASS.
Token UI typecheck/109unit/format PASS — отдельный actual author log; final
coordinator vet/race/build/process/production gates остаются отдельными.
Test-only JSON observer читает настоящий public envelope, не даёт callbacks или
renderer в product. Реальные security/replay/lease assertions сохранены; error
status и не-JSON early gate не подменяются ради tests.

Прочитаны root narrow fixes stale infra fixtures: future schema7/history6 и
negative oldschema4/5 mockedACK restore. Actual verify_restored не отключён:
недоказанный reset обязан FAIL, guard остаётся, RESET_COMPLETE отсутствует,
исходные/copied bytes сохранены. Это negative unit proof, не successfulrestore6
rehearsal. Production infra source от TASK018 не меняется.

Все40старых internal/webui files удалены; imports/htmltemplate/embed/production
Renderer wiring и Next fallback links отсутствуют, MIT/Paperclip/Inter notices
живого Next сохранены.11SSR browser suites retired, meaningful Go/Next и direct
upload coverage остаются. Новые process/E2E source отдельно review до запуска.

Immutable final source report/manifest/approval находятся только в ignored
.local/task019-remove-legacy-ui/review; эту tracked карточку report receipt не
пинует. Candidate exact SHA будет привязан после commit. Source approval не
является разрешением remote или утверждением установленной версии.
Remote/production, new native Windows/Claude/second physical PC — NOT-RUN.
