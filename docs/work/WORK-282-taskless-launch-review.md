# WORK282 — architecture/security review запуска без задачи

24.10.2026. Независимый reviewer, read-only source/runtime/server.
Source baseline `d2af9c882683da7067056351f72218b03b141659`.
Область записи: только эта карточка. Реализация не выполнялась.

## Диагноз

Private `.local/task017-taskless-launch/run-before.private.json` подтверждает:
`mr_93545b960e7c112be4e7a6e87c37cd22` completed, task_key отсутствует,
native UUID `01a16e47-17e1-78c1-aa74-a83489a55c8b`.
Текст результата прямо сообщает `task: null`, отсутствие lease и отказ создавать
root `c1.js`; commit и проверки не выполнялись. Это успешное завершение native
turn с корректным отказом писать без задачи, а не исполненное требование.

`managed-agents.md` и `execution.go:prompt` определяют optional task reference
как контекст, который не выдаёт lease. Перед записью агент получает задачу через
describe/next; несовпадение requested task запрещает работу под её identity.
`ManagedStart` не является intake планирования, не создаёт task автоматически.
Текущая форма `start-dialog.tsx` принимает свободный текст и optional task,
практически обещая больше, чем этот механизм выполняет.

## Bounded architecture verdict — APPROVE UX repair при условиях

1. На экране управления явно предложить project-scoped переход
   «Выполнить требование» в существующий `/app/automation`:
   анализ → разработка → независимый review → QA. Использовать ProjectLink,
   не терять выбранный проект. Не отправлять requirement при навигации.
2. Использовать существующий RequirementForm и явный выбор действующей policy
   с exact current version. Нет policy — черновик/объяснение, без скрытого создания
   policy, owner consent, расширения grants, бюджетов или продления expiry.
   Stale/revoked/expired/foreign policy fail closed обычным server путём.
3. Обычный manual запуск объяснить как работу по существующей задаче.
   Для намерения редактировать UI требует task reference, но ясно сообщает:
   это ещё не lease; агент отдельно получает именно эту задачу.
   TASK013 wire contract optional task не меняется этим UX исправлением.
4. Taskless путь показывать только как явно выбранный запрос без изменения
   файлов. Не пытаться угадать намерение по словам prompt или output.
   Требовать осознанный выбор/подтверждение такого режима вместо случайного
   пустого task input. Это UX restriction, не новая OS sandbox гарантия:
   текущий manual native sandbox остаётся локально согласованным workspace-write.
   Если реализация обещает технически enforced read-only, потребуется отдельный
   end-to-end контракт server/run mode/worker native flags и независимый review.
5. Состояние completed прямо назвать завершением процесса/шага и рядом показать,
   что выполнение поручения не проверено автоматически. Applied start означает
   запуск; текст модели «успех» или exit0 не становятся task done/review approve.
   Не переписывать исторический completed в failed по NLP-анализу output.
6. Формы сохраняют текущие CSRF/action_id/idempotency/version/project guards,
   snapshot профиля и exact scope. Переходы/подсказки не открывают обход ACL.
   Перед реализацией учитывается ответ пользователя о выбранном пути.

Уточнённый вариант автора также APPROVE: RequirementDialog загружает существующий
project-scoped readPage(`/app/automation`) и использует RequirementForm с
initialText. Primary CTA — полный цикл, secondary — manual. Исходный текст
передаётся только в памяти, без URL/localStorage. Захваченный project ID
проверяется при получении async response; поздний ответ другого проекта
игнорируется. Dialog размонтируется при смене проекта, переход/закрытие
блокируются во время sending/uncertain в соответствии с useAction.

Рекомендуемая минимальная область реализации: managed screen/start dialog/detail
и существующая общая managed validation/copy при необходимости. Никакого нового
scheduler, shell исполнения, security bypass, автосоздания task или скрытой
конвертации completed run в workflow не требуется.

## Повтор exact c1.js через существующий workflow — BLOCKED текущим scope

Selective read-only текущего worker config показывает local source roots:
`src`, `tests`, `tools`, `README.md`, `AGENTS.md`, `.gitignore`.
Файл `c1.js` в корне не входит ни в один из них. `sourceManifest` в
`automation_manifest.go` отклоняет tracked source вне approved roots.
После local commit с root c1.js Git proof будет rejected; завершённый workflow
с таким результатом в прежнем consent безопасно обещать нельзя.

Недопустимо молча заменить путь на `src/c1.js`, добавить широкое `.` в roots,
отключить manifest, поменять SHA gates или выдать новые права за прежнюю policy.
Здесь нет architecture approve исполнения exact root c1.js старым workflow.

До дальнейшего уточнения рассмотрены следующие пути:

- Manual: создать/назначить обычную developer task на exact root c1.js,
  выполнить после actual task lease с существующим локальным manual consent.
  Это отдельный управляемый запуск, не полный принятый workflow.
- Automation: отдельное явное согласие владельца на минимальный root `c1.js`,
  пересчёт затронутых preset consent digests/версий и нужный current host/policy
  grant через штатные механизмы. Это расширение scope, не часть данного review.

### Уточнение пользователя: выбран полный автоматический цикл

Пользователь повторно поручил exact root c1.js и выбрал полный цикл автоматически.
Это основание для минимального расширения source scope только этим файлом;
повторный вопрос о тех же полномочиях не нужен. Не следует применять старый
consent, под которым файл запрещён. Bounded план восстановления — APPROVE:

1. Подготовить private draft: добавить единственный exact root `c1.js`.
   Повысить automation host/workspace/check versions по требованиям контракта,
   пересчитать затронутые TrustedOwnerConsent digest; current files/source
   этому не изменять. Предъявить reviewer конкретный diff/hash до apply.
2. Все остальные controllers, native tools, providers, paths, model permissions,
   roots, deadlines, checks/args, budgets/expiry сохранить. Не добавлять `.`.
3. Проверить отсутствие active/uncertain attempts и pending; private backup;
   штатно остановить только Test_internet worker; применить обычный local
   config/worker consent, restart. Не править pins/journal/server DB вручную.
4. Обновить существующую policy через новый current revision и exact consent/
   check refs. Roles/profiles/intake/controllers/expiry/лимиты прежние.
   Предпочесть update, чтобы не создавать дублирующий активный grant.
   До согласованного host/policy checkpoint новый workflow не submit.
5. Один новый workflow и request ID на исходное требование; current auth/
   policy/grant guards обязательны. Старые workflows автоматически не resume.

Это architecture APPROVE плана, не доказательство применённого consent или
успешного запуска; конкретный draft и фактические receipts проверяются отдельно.

### Concrete draft review — APPROVE перед apply

Reviewer независимо сравнил private `worker-c1.draft.private.json` с текущим
live config. Exact SHA256 draft:
`d89d76e939693ceb21c37ef56c8962f48af5c7e7c42281e0b1e9c9a5969fa305`.
Mode0600. Полный structural diff содержит только пять путей:

- automation.version1→2;
- test_internet workspace.version1→2;
- project-tests.version1→2;
- source_roots append единственного `c1.js`;
- project-tests.trusted_owner_consent →
  `7ad47fbced84c365d97769035effefa53bd908bc2e820a7d73074a7ce748ec49`.

Других изменений нет. Existing installed worker preview подтверждает applied=false,
requires_local_apply=true и тот же digest при workspace identity
`1000011-55cc618`; Git/runtime executable hashes остаются прежними.
Preview путь по source не открывает Engine/credentials/state/network.

APPROVE применения именно этого draft после повторного idle/pending0 и private
backup; atomic config replacement сохраняет mode/UID/GID. Штатные consent/restart
только Test worker, без ручного journal update. Если current config изменился с
момента review, пересравнить его с draft до замены, сохранив чужие изменения.

APPROVE следующего bounded update ТОЛЬКО существующей policy
`apolicy_ca6a1de12e24aefd97a96f42d85960fd`, version1→2:
HostConsentVersion2 и exact check preset/version2/local_digest, полученный из
применённого host workspace snapshot. Остальные policy settings должны быть
полностью равны прежним, включая expiry `2026-10-25T01:05:15.4Z`, budgets,
role/profile bindings, intake/control principals и client-intake. Не клонировать.
Current owner CLI HostProof передаётся только в памяти через штатный
AutomationPolicyLocalUpdate; source transport отделяет proof от idempotency
payload. Тело запроса/ошибки/proof не журналировать. Public result/settings
сравнить после apply. Все три прежние workflows completed, не refresh/resume.

Это разрешение конкретного ограниченного применения координатором, не
утверждение, что reviewer менял или уже проверил установленный runtime.

Если выбранный путь позволяет exact файл, ограничить результат одним новым файлом
`c1.js` с `console.log(1)` и одним local commit, без dependencies/push/merge/deploy.
Предварительно проверить чистоту checkout/отсутствие c1.js и чужих изменений;
не выполнять auto stash/reset/clean. Reviewer проверяет exact commit/blob/diff,
QA сохраняет SHA/generation gates и честные evidence.

Текущий единственный preset `project-tests` выполняет Python
`-B tools/run_checks.py`; его existing tests не доказывают JS runtime output.
Их PASS можно предъявлять как regression evidence; exact blob проверяется
reviewer отдельно. Выполнение Node-проверки не включено в нынешний preset:
native workflow роли не получают права direct shell verification.
После workflow completed отдельный независимый QA вправе проверить exact
reviewed blob через уже установленный Node, без установки зависимостей,
с совпадением target SHA и проверкой stdout `1\n`/exit0/неизменности checkout.
Это отдельная QA evidence, а не выдуманный workflow preset результат.
Если Node недоступен, JS runtime check остаётся NOT-RUN; read-only exact blob
и existing tests подтверждают только заявленный ими объём.

## Статус и пределы

UX architecture/security план — APPROVE с перечисленными условиями.
Root c1.js workflow со старым local source scope запускать нельзя; пользователь
выбрал полный цикл, узкий план нового source consent выше APPROVE условно
до конкретного draft review и штатного применения.
### Source implementation review — APPROVE

Независимо прочитан полный diff managed screen/start/detail, shared managed
validation/copy, RequirementForm optional props, новый RequirementDialog и новый
unit regression. QueryBoundary уже обеспечивает unmount/cancel по project scope;
readPage проверяет expectedProjectID; useAction замораживает pending payload и
синхронно сообщает lock через ref. Исходный prompt остаётся только в component
state/prop; policy не выбирается и submission не происходит автоматически.
Основной CTA на списке ведёт в существующий automation screen; в run detail
диалог переносит исходный prompt. Manual editing требует task reference;
taskless диагностический prefix также запрещает захват задач/аренд.
OS read-only гарантия не заявляется. Отображение completed явно process-only.

После замечания reviewer автор усилил diagnostic prefix запретом task/lease
acquisition и добавил соответствующий regression assertion. Новые tests
проверяют обязательность task, explicit taskless choice, no retained task leakage,
allowlist/workspace exclusion и byte quota с диагностическим prefix.

Exact inspected SHA256:

- `managed-agents.ts`:
  `105278d1a5f2f4862a031ac984f12516c3df7020a2367081bb3c61af4606ab7a`;
- `managed-agents.test.ts`:
  `b400eb81a2e0b45d1c53da7f6f6e9a1d2844853e543c5664cd4ec37788d331f0`;
- `requirement-dialog.tsx`:
  `a8e728e5d2477145ea36b4fdadbe6859d5005d1971f910f75a2b93949a538ddb`.

Reviewer применил vercel-react-best-practices; существенных performance замечаний
нет. Независимо выполнены Node24.19.0 typecheck — PASS, 23 targeted unit tests
managed-agents/automation/mutation contracts — PASS, git diff check — PASS.
Стандартный MODULE_TYPELESS_PACKAGE_JSON warning не связан с изменением;
package/dependencies не менялись.

Production rollout, browser/process/E2E и выполнение пользовательского c1.js —
NOT-RUN этим reviewer. Независимый QA вправе начинать после данного approve.
Reviewer не менял runtime/server/config и не запускал native агентов.
Historical failed/completed runs сохраняются.

### Clean candidate source и предварительный ops review

Candidate `d80040cf546ff92320eccac0f84ca586a3b4b6f6` независимо проверен в clean
worktree. Full source manifest охватывает1260tracked regular files; каждый
проверен против Git blob и SHA256. Backend/infra/dependencies относительно
accepted4ae не менялись; runtime source delta — семь web файлов с unit test.
Private source receipt `review/source-approval.json`, kind
`independent-task017-source`, runtime_go=false, APPROVE:

- receipt SHA256 `d257ea3354fa99dc9485c69939b42dcc39cd3d408b1bc9d468454e32d9e625b0`;
- full source manifest `18ac77960e7bec49efe0d82658238174fb79719fb05273d41bcda58c7b9ea856`;
- immutable report `621a6f780ae5f8e685f89971af51220758b5f066d0c4c1908bafe510f3d5d310`.

WORK284/ops WIP прочитан полностью: adapter delta к accepted268, все семь
скриптов, source-wip manifest и pinned local references. AST7 PASS, все manifest
members/local refs exact. Предварительных blocking findings нет. Capability/
inventory/history/WAL-aware UID999/closed readers/UNKNOWN receipts/per-phase GO/
canonical all-project roll-forward guards сохранены. Исполнимого manifest ещё
нет, build_ready=false; WIP безопасно блокирует транспорт. Build и окончательный
ops approval остаются NOT-RUN до actual build/QA pins и immutable source freeze.

### Actual build review — APPROVE

Независимо проверены actual файлы coherent Go+Next candidate d80040c:
55Go members и1256Next members, exact set+каждый hash, regular-only;
все packaged infra assets равны candidate. Actual Go buildinfo readback равен
saved: go1.27.2/Linux amd64/CGO0/full revision/vcs.modified=false. ELF и SOURCE
проверены. Next SOURCE/lock/BUILD_ID/rewrite upstream7892 и Node checksum
согласованы с actual bundle. Producer logs exact hashes:104web tests PASS,
typecheck/Go build/production Next build/package PASS. Initial package refusal
по dev upstream3101 сохранён отдельно, не выдан за PASS или установленный output.

Private `review/build-approval.json`, kind `independent-task017-build`,
runtime_go=false, APPROVE, exact d80040cf546ff92320eccac0f84ca586a3b4b6f6:

- receipt `1deec12be571f82b201f4e0fb30407115307bc15d22a3797ffa4b54b215dbc09`;
- immutable report `7ec0583e42888814d9f3eb44a61951435d44687c75e8ffca2be703974f5a98a0`;
- Go binary `f46e5365231c29a3843003810701d53f9a25add7f6023fd9e57d931f98e26e95`;
- Go manifest `e42f92c0d172473a3fdf8ab371169369a3d2c6e32ae948b1bed3f7348e42ccb4`;
- Next manifest `9cb3472434620b810aef19622717d70a085659539df2e1574ec7583f3a140019`;
- BUILD_ID `JhofWwPcTj3Bzr4jsJ4Hh`;
- Node `bc17c508ffeed0ec622934f9b7fa72f8e78da65350e63c3eceb56fa688aa5e12`.

Build receipt не разрешает runtime execution. Browser/HTTPS/native workflow и
production-проверки не подменены build или unit tests.

### Final frozen operational source — APPROVE

Exact ops manifest `a577b0b141187a3dc5262215b00422b9e986b69a2b72be5116978ef1e4080d52`:
12frozen members/16local refs independently exact; весь delta и7scripts прочитаны,
AST PASS. Source/build/QA original receipt chains/report hashes/pins проверены,
QA reviewer metadata присутствует; predeploy browser NOT-RUN сохранён.
Все56Go и1257Next USTAR member paths/types/hashes прочитаны независимо:
regular-only exact set, без symlink/traversal/PAX; archive/manifests exact.

Private `review/ops-source-approval.json`, independent-task017-ops-source,
reviewer=review_repair, exactd800, runtime_go=false:

- receipt `f6f4fb288fae4764c89f97dc21a527bbdbab1dbbbcac3732c1442d21cbceb9d3`;
- report `437b3d59a6969a82613c116554faf9db315c6cccbef622e1dd0ec7c08e891a9f`.

Это SOURCE APPROVE, не phase GO. Quiet window/native c1 terminal/all-project busy0,
actual preflight/owned process/journal guards и independent external HTTPS newUI
gate перед finalize обязательны. Таймаут остаётся UNKNOWN без auto retry.
Phase main/SSH/HTTP/SQLite/service/native reviewer не выполнял.

### Native c1 workflow final audit — PASS/APPROVE

Независимо проверены QA283 report/snapshots,4real native transcripts/session_meta,
Git/filesystem, current installed worker buildinfo/config и OS children.
Underlying server4ae, workerdfcef clean Darwin arm64/SHAf629; этот кейс не retag
candidate d800 UI. `wf_80c5b653534825ea69f11a5a2bbcd715` completed25,4distinct UUID,
policy2/consent2/check2; один local commit489c123250237254608e8a16bdd492b60c8d6ee5,
только A c1.js exact `console.log(1)\n`, прежние8tracked bytes сохранены/clean.
Current/latest/successful review+QA exactSHA/gen1,2JUnit по8passed; separate
Node QA stdout1/exit0 — отдельное evidence, не workflow preset. Все5ACK каждой
попытки/pending0/no leases/reservations/owned children. Exact full server scope
read после submit честно сохранён как ограничение preflight evidence.

Private `review/native-approval.json`, runtime_go=false, original server4ae:
receipt `e257b0340d06391b87af1b55ca58ed5f91991cb43a194fe47b1bb55279149825`,
report `33a6f2fbffde5fc70c1898ff699f5d5b1938a8efdd78c97d04194f7bac27f8ae`,
25input manifest `21cfeec3a9c2440d7b7bc1f441566e592e48ac5b82e9e96f69bf7c183e5f5386`.
Native/browser candidate d800 и release installation этим receipt не доказаны.

### Final production acceptance — PASS/APPROVE

Actual d800 coherent Go+Next release: canonical finalize/final-audit complete,
owned runtime, exact logical rows9DB/schema6 preserved, format2final backup всех
4projects/9runtimeDB/9historicalDB validated; final backup manifest
`93f29fd5bb33f7197d7b5f4174a7d45ed8ba43c81d0eaf5f887ac6c44004fb81`.
Actual external HTTPS/Safari newUI gate независимо принят; original QA512672…
и own HTTPS reviewe960… сохранены отдельно, readonly/error/новый dialog submit
NOT-RUN честно остаются ограничениями. Native case4ae не retag d800.
Все9actual operation results/UNKNOWN attempt pairs и frozeninputs exact.
Private final-approval.json, independent-finaldelivery-audit/runtime_go=false:
`49eb418e6256cfa7189dd331addd159ff0c427b794f75dbc75cf6d90cd3397d3`;
report `333baa5ef76de8fb9e024b8f13a534c9209944eb760a44e4a28331a534544a8b`;
37input manifest `e19ed5a17fcb0735fddb6e19f37f906f4522661e43926039f7d69e9bad062ff5`.
Исключение300строк: карточка сохраняет последовательные независимые source,
consent, build, native и production verdicts; полные immutable receipts в .local.
