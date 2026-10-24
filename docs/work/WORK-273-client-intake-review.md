# WORK273 — независимая проверка native client intake

Дата: 24.10.2026. Reviewer отдельно от автора WORK272, process/E2E QA WORK274
и root, выполняющего реальные UI/API/клиентские действия. Владение: эта карточка
и игнорируемая `.local/task015/review/`. Product/Test/installed client не менялись.

## SOURCE APPROVE и PLAN APPROVE

Проверен frozen source WORK272: manifest
`73c9e0ee36b76c8fc4cbe709c6bca5c93f7d9ade2fc0b5289405a08c1168bf7f`.
Все шесть members совпали, private files0600; launcher AST разобран без импорта.
Независимо сравнён узкий diff к принятому WORK269 и SHA двух native executables.
Новые проверки exact provider path/SHA выполняются после selected-provider gate.
Codex сохранил read-only sandbox, process-only MCP namespace и обычные approvals.
Claude использует подтверждённые сохранённым installed help `--restricted`,
`--tools ""`, `--strict-mcp-config`, точный private MCP config. Это ограничение
клиента-координатора; полной ОС/plugin изоляции здесь не утверждается.
Новых approval/model/bypass/autoapprove/global settings flags нет.

Claude readiness остаётся **false**, первоначальная auth — не готова. SOURCE
APPROVE не включает Claude и не доказывает вход. Обычный вход владельца и узкое
подтверждение readiness/activation нужны отдельно; credential contents не нужны
reviewer. Codex-only installed TASK014 этим review не переименован в actual intake.

Принят декларативный план WORK274: private manifest
`fa6807cc9c761b686153a7b88ce94fe5d76faea450abf033779bfae924afd78d`.
Это план, не executable harness и не operational GO. Exact request texts, текущие
policy/version/request IDs и запуск каждого клиента остаются у root.

## Два последовательных результата

1. Codex model сам делает MCP describe/coordinator и workflow_submit: subtract
   и три обычных unittest; actual developer/tester JUnit должен содержать пять
   тестов с исходными двумя.
2. После terminal/ACK/reservation0/clean первого Claude model аналогично подаёт
   multiply и ещё три теста; всего восемь. Подтверждённые subtract/test bytes
   становятся защищённой baseline второго цикла.

Каждый цикл: одна задача, один meaningful local commit; исходные add, два теста,
runner/AGENTS/README/.gitignore сохраняются. Нет push/merge/deploy. Исполнители
A/D/R/T — существующий Codex worker: Claude intake не означает Claude roles.

Native origin требует actual client UUID/turn, реальные model tool call и MCP
response с request/policy/text binding к тому же серверному workflow ID. SDK,
REST submit от root, текст обещания или один каталог tools этого не доказывают.
Literal intake_channel отсутствует в поддерживаемом DTO; поле N/A, не выдумывается.
Intake UUID/counters отделяются от managed role UUID/invocations. При неясном
результате submit сохраняется фактическая неопределённость: нет fresh-ID retry.

## Управление и завершение

На одном действительно живом аналитике: observed native UUID/lease → настоящие
UI pause → отдельная proven stop/exit/5ACK/lease release → clarify → resume точного
UUID с новым managed generation/session/fence. Исходные 300 секунд этапа и policy
бюджеты не продлеваются. При гонке с завершением/вопросом/deadline coverage честно
NOT-EXERCISED/FAIL; новый UUID/replan не выдаётся за same-session resume.
Новый cancel согласован как необязательный: оба workflow должны завершиться.
Историческое cancel evidence TASK014 остаётся отдельно со своим scope.

Terminal acceptance: полный A/D/R/T и epic done; один exact SHA/generation у
реальных commit/review approve/tester passed; независимые роли; реальные persisted
project-tests v1/digest, JUnit5 затем8, typed exit0, failures/errors/skips0 и literal
false truncation. Проверяются source manifest/clean/target bindings, вся история
native попыток, пять ACK, pending/reservation/questions/leases0. Исторические
forced/delivery_unknown статусы сохраняются, не называются graceful/applied.

## Границы запуска и доказательств

Каждый client intake ограничен180s+30s own cleanup, capture16MiB/frame8MiB;
обычный interactive PTY и разовые точные разрешения допустимы. Отказ native
approval не обходится policy/flags/global edits. Source approval не является GO.
Нет скрытого второго client/model retry, параллельных workflow или расширения
policy/OS/checks. Текущая авторизация/host/policy проверяются root перед каждым.

Private evidence — finite own tuples, tool/request/workflow relation и immutable
JUnit; без raw production journal, auth/host_secret/IPC/provider env/token или его
fingerprint. Root/отдельная QA готовят пакет; reviewer проверяет actual независимо.
Новый executable capture/collector, если потребуется, проходит отдельный review.
Windows, второй физический ПК и desktop-wide registration остаются NOT-RUN.

Фактически выполнено этим review: чтение source/help/плана, AST/static diff,
member hashes/modes и binary SHA. CLI/модель/browser/HTTP/SSH/Test mutations,
импорт launcher, auth reads и runtime tests **не выполнялись**. Actual intake,
два полных новых цикла и новое UI управление пока NOT-RUN. Ограничений/находок,
требующих изменения frozen source или декларативного плана, нет.

## Actual Codex case — отдельный final evidence verdict

24.10.2026: **APPROVE — PASS_CODEX_CASE_WITH_DECLARED_GEN1_CAPTURE_LIMITATION**.
Общий TASK015 остаётся PARTIAL: реальный Claude intake/второй цикл NOT-RUN,
обычная авторизация владельца ещё нужна. Это не общий PASS двух клиентов.

Независимо проверены 25 indexed artifacts QA274 (SHA/размер/inode), реальные
собственные native JSONL без выполнения содержащегося code-mode текста, exact
model describe/submit и tool response. Request/text/policy/requester/intakehash
совпадают с frozen prompt и тем же completed workflow. Native client exit0
подтверждён отдельным root receipt; один normal approval на call, не Always.
Первый TERM startup FAIL остаётся в истории и не выдаётся за model turn.

Четыре текущие роли completed, workflow version32/epoch3; пять native invocations
включают analyst continuation. Все этапы завершены до своих deadline. Отдельные
board authors, reviewer approve и tester passed относятся к одному фактическому
SHA/generation. Два immutable XML независимо разобраны: по5 testcase, failures/
errors/skips0; typed exit0, literalfalse truncation, source_clean, target/manifest,
project-tests v1/digest и public attestations совпали. Все четыре текущих local
runs имеют пять ACK/exit/lock flags true; pending/reservation/questions/leases0.

Известные Git objects и checkout прочитаны без Git CLI: один parent, один commit,
ровно implementation и новый subtract test file. Пять защищённых файлов и AST
исходной add сохранены, tracked bytes совпали с деревом exact принятого commit.

Для управления actual analyst JSONL содержит два настоящих turns одного UUID,
разные board sessions, gen1→gen2 и сохранённое уточнение во втором user turn.
Paused stopped/stop applied, retained stopped receipt и gen1 run_exit attestation
согласуются с успешно продолженным guarded gen2. **Прямой полный local5ACK snapshot
первой generation до resume НЕ архивирован**; indirect corroboration им не
называется. Initial delivery_unknown/last_error сохранён. Literal OS resume argv
observer NOT-RUN; новый cancel в этом задании не требовался.

Native TUI действительно изменил два служебных config поля. Принятый one-shot
helper исполнен root после own native idle: actual exit0, backup изменённого
состояния и точный восстановленный baseline независимо проверены по байтам.
Это changed-then-restored, не unchanged за весь запуск; сохранение informational
provenance не заявляется. Auth/token files reviewer не открывал.

QA274 actual report SHA
`710ded65734dc07b3173975205ddc0a3bbf7b5f6c344d9ec59222c7e8241443a`,
index SHA `19c970aacdb69ad970cb23fb6a75524311b544c76941e9dadc50b4e9133d4155`.
Final private verdict/report и exact семь document hashes находятся в
`.local/task015/review/`. Новый server/worker release, control call, model/test
или иные runtime этой независимой проверкой не запускались. Codex scope принят;
Claude/Windows/физический второй ПК/Claude execution roles им не подтверждаются.

## Claude Desktop — actual final verdict и завершение выбранного scope

24.10.2026 пользователь выбрал уже авторизованный Claude Desktop вместо отдельного
Claude Code CLI. **APPROVE — PASS_CLAUDE_DESKTOP_INTAKE_AND_CODEX_PIPELINE**.
Совместно с ранее принятым Codex case принят текущий TASK015 scope: два настоящих
клиента подали требования, оба полных цикла завершены; управление переиспользует
прежний Codex pause/clarify/resume с его объявленным gen1 capture ограничением.
Это не повторное испытание управления из Desktop и не общий V1/platform PASS.

Независимо проверены все32 members immutable Desktop packet по manifest
`97555579ca719f70c60968e6e02a2b401ae6740adf197e7ac32861579d4ae39e`.
Просмотрены собственные screenshots Describe, Submit head/start и законченный
ответ модели. Видны actual Describe coordinator/request/run/machine и выданная
boardSession/project, actual Submit request/policyv1/полный текст/queued workflow
и intakehash. Те же текст/hash/requester/grant/workflow совпали с server report.
Сохранённые describe.ax.txt **и** submit.ax.txt — только stale/nochange notices;
они не являются доказательством arguments/fulltext. Неверный AX pointer исходной
metadata не переписан: достаточное первичное подтверждение содержат PNG.

Desktop model turn завершён; CLI exit к нему N/A. В UI сохранён существующий
режим «Авто», individual dialogs не наблюдались; нового Always/permission режима
не приписываем. CLI OAuth/gate этим не включены. Apply source/ordinary root exit0
и readback подтверждают add-only bridge; reviewer не читал/не хешировал Desktop
preferences/config/token/authcache. Desktop chat — клиент подачи; все роли Codex.

Server workflow completed/version25/epoch1; четыре независимых роли/nativeUUID,
четыре invocation, одна задача, fixes0, этапы завершены до своих deadline.
Review approve/tester passed привязаны к exact commit/generation. Два immutable
JUnit независимо разобраны: по8testcases, failures/errors/skips0; typedexit0,
literalfalse truncation, source_clean/manifest/target/preset-v1 digest и actual
attestations согласованы. Current4run5ACK true, pending/reservation/questions0,
полные applied command pages, tasks/epic done и leasesclosed.

Известные локальные Git objects прочитаны без Git CLI: один parent — принятый
Codex commit, один новый commit; только math_ops.py и новый multiply test.
Шесть защищённых файлов совпали; AST add/subtract неизменён, три новых ordinary
метода теста. Working bytes совпали с committed tree, saved Git status clean.
Новых тестов/моделей/API/UI/helper/network/control/поставки reviewer не выполнял.

Независимая QA274 receipt SHA
`458c48b624bc580a0cff9d8a8e30d312ed71c97e53b353a070bee63c2c1741e6`.
Final273 private verdict — `.local/task015/review/claude-desktop-final273.json`.
Claude Code CLI authentication/nativeintake, Claude execution roles, Windows и
физический второй ПК остаются NOT-RUN; исходные FAIL/ограничения не переоценены.
Сервер/worker/ACL/consent/checks/лимиты не менялись этим review. Root canonical
receipts-only итоговая строка после этого verdict не расширяет принятый scope.
