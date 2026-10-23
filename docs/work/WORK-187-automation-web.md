# WORK187 — браузерный запуск автоматического цикла

Автор: automation_web. Основание: independent CONTRACT APPROVE183 для TASK014.

## Область

`internal/web/automation_*.go`, browser Page/interface/routing/actions additions;
`web/src/lib/automation*.ts`, `web/src/features/automation/`, route/action allowlists,
project navigation и screen dispatch. Core/store — author185; projects/transport/
bridge/worker/infra — root. Общие DTO запрашиваются у185 перед wiring.

Экран: требование → выбор одобренной policy/компьютера/ролей/лимитов → workflow,
стадии и task/run delivery, вопросы и отчёты, pause/cancel/resume/clarify. Editable
initial team использует existing profile CRUD и stable per-create browser actions;
не называется подключёнными worker. Authority только trusted browser CSRF action,
project isolation и exact retry неизвестного исхода сохраняются.

## Статус

SOURCE READY для независимого source/security review. Общий TASK014 не закрыт.
Runtime/native/Windows/second physical PC/production NOT-RUN. Native
foundation отдельно; UI не обещает работающий запуск до реальных proofs.

Применён skill vercel-react-best-practices: existing query cache/project boundary,
parallel independent reads, no new dependencies, small composed components.

## Реализация

- Go browser adapter вызывает настоящие Workflow/Policy методы board, не mock
  runtime. Чтение включает current profiles, managed hosts, per-workspace local
  consent, workflow steps/plans/questions/attestations и current run delivery.
- Сохранены same-origin, CSRF, server-issued actions, per-principal action epoch,
  CAS versions, project-qualified routes и durable web-action recovery. Настройки
  policy strict JSON: unknown/duplicate/null/trailing/deep fields отвергаются.
  Trusted policy approval создаётся только после подтверждённой browser action;
  native flag/role/session сами его не дают. Cached receipts проходят readonly
  authoritative permission check; cached rejected action не исполняется заново.
- Next: отдельный экран «Выполнить требование», draft/intake, editable initial
  four-role team с отдельными existing profile:create actions, policy CRUD,
  per-role Codex/Claude + pinned profile revision, exact workspace consent и
  static check references, видимые deadlines/limits/client intake consent.
- Ход выполнения, ссылки на task/run, delivery status отдельно от process state,
  вопросы назначенным людям, plan/history/evidence/test counts/SHA reports,
  pause/cancel/resume/clarify и explicit paused-only replan. Replan wire extension
  принят root как реализация уже согласованного human replan; нужен source review.
- При неизвестном исходе форма/действие заморожены до exact retry. Изменившаяся
  версия управления принимается явно; изменившееся local consent требует явного
  обновления выбора ролей/проверок. Read-only project projection убирает controls.

## Авторские проверки

PASS: TypeScript; весь Node24 unit набор96/96 (8 automation-specific); final Next
production build (Turbopack1.216s + TypeScript280ms). PASS: полный Go web package
с race27.350s. После последнего regression отдельно `go test -race ./internal/web
-run TestAutomation -count=1` PASS2.956s:7 automation tests. `git diff --check` PASS.

Actual SQLite browser integration подтверждает четыре независимых идемпотентных
profile creates, draft без dispatch, exact receipt/collision, trusted browser
policy approval (human native session без approval не создаёт policy), immutable
revision, cached rejection без скрытого исполнения, current host revocation.
Lost committed reply переживает session continuation и возвращает прежний receipt;
неcommitted запрос с истёкшей original session при continuation не исполняется.
Оба обнаруженных core185 нарушения закрыты regression проверками. UI/adapter не
ослабляют auth/epochs/session rules для прохождения тестов.

SQLite тесты не запускают процессы или модели. Registry authority предоставлена
типизированным test context; фактические guards projects wrapper проверяет root.
Новые browser/process/E2E, native analyst→developer→review→QA, negative fix loop,
production deployment и physical second-PC/Windows проверки NOT-RUN; это отдельная
независимая стадия после source approval. Прежний native GUI FAIL не закрыт UI.

Новый test файл automation.test.ts немного больше300строк из-за общих synthetic
fixtures и восьми независимых security/route regressions; production компоненты
разделены по формам/ролям/контролю/истории и остаются около/до300строк.

## Revision после REQUEST CHANGES189

P1 исправлен: cached permission получает exact stored Workspace, payload kind→
CommandKind, RefreshPolicyID/Version и QuestionID. Readonly domain authority
проверяется до сохранённого receipt; новая mutation или повторный запуск для
проверки прав не выполняются.

Добавлен trusted AutomationIntakeApproval только в browser submit и его cached
permission path после CSRF/server-issued action binding. Native Describe(human)
не включает ClientIntake. Wire approval flag отклоняется strict parser.

Пять дополнительных actual SQLite регрессий PASS:

- настоящая domain WorkflowQuestion через synthetic execution capability/session/
  lease, human answer с lost reply → exact retry → cached receipt; последующее
  изменение durable question designation в test fixture запрещает старый receipt;
- cached resume после revoke grant →403;
- cached explicit refresh после policy revision change →409;
- cached policy create после expiry именно selected workspace scope →403;
- ClientIntake=false: browser submit+cached retry работают, native human без proof
  и forged approval form запрещены.

Final revised `go test -race ./internal/web -count=1` PASS32.385s; всего12
Automation-prefixed regressions. `git diff --check` PASS. UI исходники не менялись
после прежних Node96/96, TypeScript и Next build PASS. Source review189 остаётся
обязательным; исправление автора не считается независимым approve.

Question helper не запускает CLI или OS process: это domain/API integration,
не native/second-PC/E2E PASS. SQL designation change — целевое изменение durable
fixture для проверки current read authority, не новая runtime admin операция.

## Узкое уточнение текстов и пагинации перед browser QA

Frontend-only revision: известные причины ожидания/остановки объясняются по-русски
с дальнейшим действием. Для неизвестной причины — безопасный actionable fallback;
raw reason_code не отображается как основной текст ни в деталях, ни в списке.
Фиксированная таблица включает current automation domain reasons, budget/worker/
access/source/test proof errors. Неизвестный server payload не выводится в тексте.

Пустые steps/proofs/list показываются как отсутствие записей на текущей странице,
а не глобальное отсутствие задач или подтверждений. Добавлен переход к началу;
next_cursor обозначает общую следующую страницу истории (включая paged messages),
не обещает только шаги/запуски. Это совместимо с bounded intake preview списка и
удалением длинных profile instructions из read projection; UI их не использует.
Backend187 не изменялся: проверен zero drift всех14 internal/web файлов manifest.

PASS: существующие96 Node24 unit tests, TypeScript и final Next build (compile456ms,
TypeScript192ms). `git diff --check` PASS. Зеркальные тесты для таблицы статичных
текстов не добавлялись. Runtime/browser/native/production acceptance этой малой
редакцией не заявляется; независимый browser QA остаётся следующей стадией.

## Bounded host scope follow-up перед browser QA

Необходимый source revision после сокращения ManagedList summary в185:
`GET /web/automation/host-scope?host_id=...&workspace=...` использует обычные
browser session/project guards, strict single query values и только authoritative
ManagedList(HostID,Workspace). Не перечисляет workflow, не выдаёт web actions,
не меняет host/process/OS consent; viewer projection снимает CanControl.

Policy form берёт aliases из Host.workspaces, затем отдельно читает полный
scope выбранной папки. ScopesTruncated summary не используется как разрешение.
Abort + sequence fence + project/host/workspace/current permission-version key
отвергают поздние ответы и старые errors. До полного ответа роли/проверки/submit
заблокированы; новая selection требует явного принятия current consent version.
Принятие пересоздаёт выбор profiles/checks в пределах exact fetched scope, без
угадывания flat roles/providers/checks. Missing alias/expired/revoked consent
показывают понятное сообщение и retry. Retry неизвестной mutation остаётся exact
frozen action и не принимает новые permission snapshots автоматически.

UI понимает optional intake_truncated preview и показывает многоточие в списке;
full intake читается в деталях. Paged history empty texts остаются локальными.
409 stale cursor предлагает project-qualified начало текущей истории вместо
бесконечного retry прежнего cursor.

PASS:4 новых Node units для stale success/error/project cleanup, full current
scope/control/expiry и cross-project response;2 actual SQLite browser scope
integration tests для strict readonly exact selection и viewer permissions.
Все100 Node tests PASS; TypeScript PASS; final Next build PASS (compile1.436s,
TypeScript515ms). Revised `go test -race ./internal/web -run TestAutomation
-count=1` PASS7.366s (14 Automation-prefixed regressions). Diff check PASS.
Новые browser/process/E2E не авторились и остаются независимой QA стадией.

## CheckAborted wire delta — narrow frontend reopen

Core185 добавил optional WorkflowObservations.check_aborted; web DTO принимает
флаг. Отдельный pure presentation helper даёт ему безусловный приоритет над
recorded passed/completed, exit0 и partial successful test counters. В истории
видно «Проверка прервана, результат не засчитан» и предложение повторить именно
эту проверку; положительные счётчики и код выхода не создают PASS presentation.
Отсутствие флага сохраняет historical compatibility; exit0 сам по себе никогда
не повышает pending status до passed. Optional policy-page next_cursor добавлен
в DTO для соответствия read API; новый revisions UI не создавался.

PASS:3 meaningful pure units для abort precedence/partial counters, отсутствия
promotion по exit0 и legacy observations; TypeScript PASS, diff check PASS.
После этой узкой delta полный Node набор/build не повторялся по поручению root:
общий final build после freeze проведёт координатор. Предыдущие100 Node tests
и scoped14 Go race PASS относятся к предыдущему freeze. Backend187 не менялся;
повторно проверен zero drift его source files. Runtime/browser/native assertions
новой редакцией не заявляются.

## Frozen source handoff

Всего 45 source/test файлов. Aggregate SHA256 по отсортированным строкам
`path + space + sha256 + newline`:
`c018c82567437fccf8a402937fe4e89e6638a97ecb11a1c5484b32f302f70400`. Private manifest `.local/task014-web/source.json`.

| Файл | SHA256 |
| --- | --- |
| `internal/web/actions.go` | `20b2baf6c94e447dc080ada2496407293382239895d0902e37d20eeb12b38041` |
| `internal/web/automation_action.go` | `ab490b9797e47dd493e82c098944e336cc6a0c6565c857131b9bbbc5414c0ea1` |
| `internal/web/automation_browser_test.go` | `b942c56c944d430b6fa47d8e1327e32e08cf03341a4a15a2fcdbd45756adfa9b` |
| `internal/web/automation_form.go` | `b099e7f09a525ca716b872f633f9075c02ed188b9d0144689d25864eec51d71c` |
| `internal/web/automation_form_test.go` | `3bb31bf2b475fb18d215d9a8c2e2935984d3b46ed01d9e4273619373001e3ea7` |
| `internal/web/automation_host_scope.go` | `11a92dc7261f12e17eea1036ae30af2f58260be73901f03aa3bfbed5e9764426` |
| `internal/web/automation_host_scope_test.go` | `1c231df5244c4cea808d33fc7ddf61c7bd2089e27cedc962453f7a739744f53a` |
| `internal/web/automation_json.go` | `02438bfd8f8d237ae91c0ba0b59bda7378214733374818a72a3100bb6dcebb62` |
| `internal/web/automation_page.go` | `3386221d57690db292d5d551516a14defa45482311dc97720540b465e1b52c6b` |
| `internal/web/automation_permission.go` | `ac64f9570b82c60646f481592ddae13d27fb144dd34a3b963de732a324902787` |
| `internal/web/automation_receipt_authority_test.go` | `4c81a2919bdc840e06f09a4dc825a8afaa1325246fb6aacb760cf3708c9a57e9` |
| `internal/web/automation_receipt_fixture_test.go` | `c833de830076775144c248402943bfda1bdb1f9225eeefa958e3466d05d1c5a5` |
| `internal/web/automation_recovery_test.go` | `95bc4a562c9f6de779722b4f996fcee260a8ddb913b7d028b7a754ee0bc10c35` |
| `internal/web/browser_page.go` | `f8ebc90a39eb681bdfa7ac2eedad9b1e4a4db18a9d4611f65a35ccfe8af0782d` |
| `internal/web/contracts.go` | `02ae4527886d2356a6c9afb7c50c83fed21d10d95a8a6d08c0554d7afdec3b25` |
| `internal/web/http.go` | `b18565ccc1d0c195bfbcb45b8158c7087b03b5de4c564c83d4a1aad98a2ec0cc` |
| `web/src/app/globals.css` | `48d0ba10907b633a8937e1e4c55dc53cd52bcd294a39703a1271993736751484` |
| `web/src/app/styles/automation.css` | `1dff3ced586a9d65a7bd2f5ace72a85dd7d500f7e8ee0ae0d6c48ced3c416128` |
| `web/src/features/automation/automation-screen.tsx` | `6629bd462af63a6ffb4afa2aadbeefc0fb35f590525bd17c02065b7f7e0ccf0b` |
| `web/src/features/automation/policy-bindings.tsx` | `6a97a85cbf2742f261db7300358594c97e4c8ae2ce0a646991d33f4375d815ac` |
| `web/src/features/automation/policy-detail.tsx` | `53fc9c7dbcfe0949a78a17c8ea0ed3fd924ee753d682c0de8ba7e5c0daae7563` |
| `web/src/features/automation/policy-dialog.tsx` | `8da45e92a9a89974c2ea4373718cd896f37e1d026f349cb57e75667e565d4c00` |
| `web/src/features/automation/policy-host.tsx` | `a878e68387c31fe774cb680e3d840ea20eda545163fdd645075543f259544314` |
| `web/src/features/automation/policy-limits.tsx` | `b9eba4af50a5e4f8f1d3ebe2ac2312c2f3fc97b554b4e35241b79ab8f1df36c4` |
| `web/src/features/automation/requirement-form.tsx` | `45df7e229aa086fabae7bb339342c653fe0da1825ab8c75f6d519ff6a1c7d434` |
| `web/src/features/automation/team-setup.tsx` | `0a7d803a8e6fd8b96513669e053b7e5fdb2e20edd89adc83044e87ff39b80509` |
| `web/src/features/automation/use-policy-scope.ts` | `401352e566442a22557fe6465df55d09ad64b10d42f71599aee53831555676c9` |
| `web/src/features/automation/workflow-controls.tsx` | `f8871d396cfbe98ddbb86c95e0dec48eabe406a72420b8b01e64efc8f0e80244` |
| `web/src/features/automation/workflow-progress.tsx` | `9f66d6adabd22ee10f1a1017c62e8886b50d559b1ede8abb9c8f4c5b07152094` |
| `web/src/features/automation/workflow-questions.tsx` | `dd36ea4e51a76fd076efb58329fec070a3619c35431c1d549c0da728fd441122` |
| `web/src/features/board-screens.tsx` | `cf766adfa1d471e01343c0c52bbb48240dba8c9bdac96b2840f774bf58b78eb4` |
| `web/src/features/control-app.tsx` | `60440aa2cf1d900bb4df18cc8debda34c4cb23bff551f0acdfe31a992ded3cc0` |
| `web/src/features/navigation.ts` | `9cdfb53035f65c562c071147d665337c271d56a11b7ab55a0c0aa5bed167febb` |
| `web/src/lib/api.ts` | `b10e4e5844e6e946d41cbd4e3e210a4b21859669e1c3f81108974ec6deb9682f` |
| `web/src/lib/automation-evidence.test.ts` | `6a3bca73621e6f3f98a4517f03f856d3a6decd7794600d551bb85bca1feb1029` |
| `web/src/lib/automation-evidence.ts` | `6be887e2eadd7c52c361b9a418057677d06942300d1aca2ff3ec6b97114b2f40` |
| `web/src/lib/automation-host-scope.test.ts` | `b0fe62525de97cc34a7172738b6a3c2fbb64cda24abc84f8bdfba3c0353d371a` |
| `web/src/lib/automation-host-scope.ts` | `a39c8e2639443d799894fd7cac792cd4c0632d250cc68e24c8856f71509c3190` |
| `web/src/lib/automation-policy.ts` | `25d91ab769e6e50bf7aa92399a05458a8ab2265e264b8803d80fc0b727a49dfc` |
| `web/src/lib/automation-reasons.ts` | `ec9df5b9b4157567221c386a94ef4552f1aebb496f4a624f5c64d74386572315` |
| `web/src/lib/automation.test.ts` | `0862eaa94b33f7e607f0d63e421cebc3a7826584588b09e39a7164d55b5bffbd` |
| `web/src/lib/automation.ts` | `8de8be4137a26d57c6723851744079e06b6d4c0d8cbebea59602b926de17cd00` |
| `web/src/lib/managed-agents.ts` | `85bdd2f3b134e1f9e639c3a442068869e0808b6a1cb7a373bd4d9d56df8e0c02` |
| `web/src/lib/project-routes.ts` | `99242a61d467941662a4e46ad054cc74215aa7efa46f363081ff6d25b66b5ef1` |
| `web/src/lib/types.ts` | `b15ae87342c798d0d0354a57145163e912ff67fce31ee166906fccf1c4331938` |
