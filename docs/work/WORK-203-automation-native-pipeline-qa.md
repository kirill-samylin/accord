# WORK203 — full native pipeline QA, source plan

23.10.2026. Отдельный QA; root координатор, независимый reviewer202.
**SOURCE COMPLETE / 202 SOURCE REVIEW PENDING / EXECUTION NOT-RUN.**
Ни servers/workers/models/MCP/config RPC, SSH, production или GUI не запускались
и не менялись. Разделы1–8 сохраняют историю PLAN/partial checkpoint; текущее
source продолжение и exact manifest указаны в разделе9. Runtime gate закрыт.

## 1. Версия и жёсткие prerequisites

API foundation — composite APPROVE189 candidate
3e6ac81ee81909e3c33a2c86e593f4781357878d. Base Mac binary SHA
7207d53679dbc0e91132ec3bc206f70e860516d3cc818aab093a2ce023d84dae;
Next manifest946c79a21ca86314f704dc75fe04ae265370c20b37d56e6954981ec33d215fd1.
Это source/build provenance, не native pipeline PASS. WORK197 existing schema6
checks, WORK198 failures и production195/196 outcomes не переписываются.

**EXTENSION_PENDING: developer Git permissions.** В3e6 нет нового opt-in201/204.
Нельзя выдумать рабочий config/API или включить его флагом в старом binary.
До reviewed implementation, config-only permission proof, actual single-commit
и OS-negative enforcement checks, затем нового exact candidate pin202 — stop
перед любым setup/spawn. Contract field name сам по себе не реализация.

Второй prerequisite: отдельное local tester artifact-write согласие. План
использует существующий `allow_tester_artifact_writes=true` только в synthetic
workspace; оно входит в digest и требует workspace-write. Source/HEAD guards
сохраняются. Это observed artifact/source boundary, не OS artifact-only sandbox.
Fixed trusted-owner check permission отдельное от native tester разрешения.

Следующие gates: root принимает plan → QA пишет небольшие modules/fixtures →
202 SOURCE APPROVE всех новых sources/pins → отдельный exact root GO. Пока нет
ни runnable harness, ни worker config, ни config RPC реализации в этой работе.

## 2. Minimum fixture и budgets

Fresh private0700 HOME root; own bin/state/registry/web/check artifacts/credentials
и один ordinary Git checkout `.git` directory. Нет linked worktrees, symlinks,
remotes/hooks, внешних packages, push/merge/deploy. Real HOME/native auth сохраняются;
старый TASK013 synthetic harness с подменой HOME нельзя копировать для real CLI.
CLI/candidate/Node/Python/Git/Chrome hashes фиксируются до/после. Go binary запускается
из exact verified HOME copy, не свежим Desktop artifact.

Предварительно review202 frozen unit fixture:

- `src/math_ops.py`: маленькая функция add, seed NotImplementedError;
- `tests/test_math_ops.py`: два meaningful unittest cases positive/negative inputs;
- `tools/run_checks.py`: stdlib unittest execution и JUnit с actual test cases,
  errors/failures/skips/counts; не echo/static fake report;
- README.md, AGENTS.md с synthetic scope и `.gitignore` для `.artifacts`/pycache.

Явные SourceRoots: src, tests, tools, README.md, AGENTS.md, .gitignore.
SourceRoots="." пересекается с ArtifactRoots и не применяется. ArtifactRoots
только `.artifacts`; dependencies empty. Python запускается с -B. Один seed
commit делает setup, один следующий commit должен сделать native developer,
изменяя только src/math_ops.py. Tests/runner/source helpers seed review frozen.

Happy policy budgets: plan_tasks1, native_invocations4, fix_per_task0, fix_total0,
continuation_per_step0, step_seconds180, workflow_seconds900, grant_seconds1200.
Local maxima совпадают либо уже; consent expiration30min. Весь happy execution
bounded900s + cleanup reserve60s; setup отдельно bounded180s без model turns.
Любой вопрос, refusal, timeout, extra invocation или recovery — FAIL/stop,
не новый attempt. Wire idempotent retry не является native restart.

## 3. Изолированный setup через поддержанные входы

Использовать реальные CLI/HTTP registry paths, не typed store fixtures/SQL writes:
initial default board/legacy owner token → owned legacy serve только для storage
init → graceful stop → offline projects bootstrap-owner → registry-enabled serve.
Далее normal browser catalog create/open synthetic project; токены private0600.
Все пути новые private, производственная registry/DB/host не используются.

Go serve включает --projects-data/--web-data/--web-origin; без registry scheduler
и AutomationService недоступны. Base Next rewrite уже compiled на loopback7892,
поэтому Go7892/Next7891 проверяются free BEFORE HTTP; занятый порт — stop, не kill
пользовательского процесса. При новой сборке expected origin берётся из её
manifest, не молчаливой runtime подменой build-time rewrite.

Один fixture owner token/current principal обслуживает policy и роли — это
нормально. Controller allowlist явно только нужный fixture principal. Своё
worker enroll/run с отдельным state_dir; никакой persistent GUI LaunchAgent.
Local AutomationConfig содержит four roles→codex, version/expiry/maxima и один
fixed required test preset `python-unit`, isolation=trusted_owner_process.
Executable canonical pinned Python; Args=[-B,tools/run_checks.py], Directory=.,
TimeoutSeconds30, ReportFormat=junit, ReportPath=.artifacts/junit.xml.

Digest/identity получают маленьким QA-owned setup helper, который вызывает
существующие exported worker.AutomationIdentity/AutomationPresetDigest;
worker CLI identity — principal lookup, не filesystem identity. Public CLI
"preset digest" не выдумывается. Helper source также202-reviewed до compile/run.

Native consent exact union из actual validateAutomationNativeConsent:
describe,next,get,transition,workflow_question,workflow_plan,workflow_run_check.
Старый GUI10 grant не переносится. Модель не получает parent owner bearer/host
proof; только per-attempt execution credential/IPC binding. Все остальные MCP
connectors отключаются existing managed preflight, Tandem не используется.

## 4. UI policy, requirement и один replay

Playwright создаёт свой headless browser/context, normal token-file login,
catalog project switch и current same-origin CSRF/action IDs. UI создаёт четыре
profiles role analyst/developer/reviewer/tester, provider=openai, model empty,
immutable revisions1; role bindings policy provider=codex. Никаких fake human
sessions для scheduler. Current host-scope берётся supported GET с host/workspace,
не из truncated list DTO.

Real PolicyDialog требует explicit consent; submit POST
`/web/p/{project}/automation/policies/create` содержит csrf/action_id/settings.
Settings: exact host/consent version/workspace, four profile revisions, только
fixture intake/control principal, client_intake=true, python-unit ref/version/
digest, git_policy=local_commits и указанные limits/expiry. Native bearer или
describe(role=human) не подменяет browser approval.

RequirementForm отправляет `/web/p/{project}/automation/submit` с exact policy
ID/version/text и issued action_id. Capture request только private. Затем scripted
**real MCP transport**, не пятая модель: candidate mcp/initialize→describe ordinary
client session→workflow_submit с тем же action_id как request_id, теми же payload
и current principal/epoch. ClientIntake=true нужен для второго канала. Проверить
один workflow ID, одну analyst task/run, неизменный intake hash. Scripted replay
не объявляется отдельным native Codex intake proof. Browser session и MCP client
session могут различаться; requester authority должна совпасть.

## 5. Четыре actual native стадии

| Стадия | Обязательные действия и принимаемый proof |
| --- | --- |
| Analyst1 | Fresh describe/next exact assigned task; structured workflow_plan одного client_key с title/description/areas/acceptance/prerequisites=[]/required_check_ids=[python-unit]. Plan version/task generation реальные. transition done с summary и artifact `workflow-plan:<actual immutable hash>`. Native proven exit + plan.status materialized, без writes. |
| Developer1 | Собственный run/UUID/session/lease; реализовать add, actual один local commit поверх seed. После clean committed HEAD вызвать workflow_run_check python-unit, ждать completed, затем transition done с actual full SHA и run_summary. Отсутствие commit/semantic refusal блокирует workflow. Parent не дописывает результат. |
| Reviewer1 | Новый run/native UUID/board session; read-only exact developer target SHA/generation. transition done outcome=approve, evidence.verified_commit/findings; contributor history исключает own review. HEAD/source manifest неизменны. |
| Tester1 | Новый run/UUID/session; approved exact SHA. workflow_run_check только request_id+preset_id; poll accord://workflow-check либо same exact request until completed. Actual fixed Python check обязан исполнить2 tests, JUnit hash/exit0/passed>=1/failed0/skipped0, не aborted/truncated. transition done outcome=passed + genuine summary/verified SHA. HEAD/source manifest прежние. |

Роли различаются по actual NativeUUID/ManagedRunID/SessionID и contributor
provenance; один owner principal допустим. Policy counters started/invocations
не более4, continuations/fixes0, ни hidden auto retry, ни пятого summary turn.
Native/check API writes выполняют только настоящие дети/worker: harness не
подделывает plan/task done/attestation/check counts или accepted/started receipts.

Summary — поддержанный workflow_report/WorkflowPage, не выдуманный text DTO:
workflow completed, epic done, four completed steps, one materialized plan,
actual target SHA и current review/QA outcomes, qualifying run_exit/check
attestations с exact tuple/digest/source clean, нет questions/leases/active runs,
reserved_runs0. UI показывает это состояние и те же evidence; own final report
связывает actual facts. Пагинация дочитывается до конца, truncated list не proof.

## 6. Provenance, failure и cleanup

До/после сохранить candidate/source/harness/fixture/tool hashes, effective safe
permission proof digests и отдельные hashes исходных raw config snapshots private,
не повторять утрату before-config proof198. Не считать project trust metadata
доказанным расширением прав или установленным авторством drift. Секреты/полные
transcripts/host proof/credential values не выводить. Safe public proof только
counts/statuses/hash bindings; IDs/UUID и raw logs private.

Server/Next/worker/browser/MCP client регистрируются own process handles. Native
и trusted checks управляются реальными worker supervised helpers/lifelines;
positive nonce exit+terminal ACK нужны до освобождения checkout. Unix owned PGID
не arbitrary tree containment. Raw JUnit/observations на exact reviewed SHA,
post-exit source manifests и native IDs проверяются независимо от слов модели.

На FAIL сначала own workflow cancel CAS под current controller, далее worker
graceful stop и owned cleanup в60s reserve; deadman/unknown сохраняют needs_recovery/
reservation/private journal, не удаляются ради PASS. Не restore/править пользовательские
configs. Never generic pkill/killall или PID из старого disk record. Always-save
FAIL report и конкретные manual-inspection limits при неопределённости.

На PASS убедиться terminal/no lease/no pending proofs; затем normal revoke своей
policy/disable fixture host/archive своих profiles, graceful stop own worker,
MCP/browser/Next/Go. Сохранить runtime DB/history/artifacts private; credentials
отозвать только в своём fixture offline после остановки. Не cancel/done реальные
задачи и не переносить результаты в production.

## 7. Следующие отдельные сценарии, пока без source

После happy PASS отдельно reviewed negative/fix fixture с одним bounded cycle
и новым budget: negative QA вариант требует до7 native invocations
(analysis/dev/review/test_failed/dev_fix/review/test_pass), не прячется в happy4.
Fault injection/ожидаемый negative outcome должны быть явно202-reviewed; чужой
target SHA не меняется вне developer commit. Pause/resume/question/answer,
clarify/cancel/restart/unknown и два site principals intake/control — отдельные
fresh scenarios с exact budgets. Linux restore WORK199 — отдельный storage proof,
не native pipeline. Windows/Claude/physical second PC NOT-RUN.

Private `.local/task014/pipeline203/plan.json` — декларация, не executable harness.
Никакие новые процессы/config RPC/model вызовы в WORK203 пока не выполнялись.

## 8. Partial source checkpoint после PLAN APPROVE202 / root source-prep GO

Root попросил освободить concurrency slot для независимого review204/199/205.
Подготовлены только private source modules: gate/main, private filesystem/pins,
owned supervisor lifecycle, Python fixture+JUnit, public registry/setup HTTP,
cookie/CSRF session, UI profiles/policy/requirement, real MCP same-action replay,
role instructions. Каждый файл меньше300 строк. README в private directory
фиксирует incomplete части; SHA256SUMS фиксирует текущий partial checkpoint.

**UNRUN / INCOMPLETE:** identity/preset Go helper, baseline config builder,
candidate/tool pin resolution, future201/204 extension hook, workflow/Git/JUnit/
terminal assertions, pagination, observer и always-save final cleanup coordinator
ещё не написаны/не связаны. Нет полного runnable harness. Entry hardstop
EXTENSION_PENDING без env bypass; модули не выполняют actions при import.
Syntax checks/compile/process/model/HTTP/MCP/config RPC не запускались. PLAN
APPROVE не переносится автоматически на эти новые source modules: после
завершения полного source требуется отдельный WORK202 SOURCE review и root GO.

## 9. Source completion после root ownership handoff, без runtime

Root передал полную запись `.local/task014/pipeline203/` и этой WORK-карточки
новому отдельному QA author; прежний QA198 туда не пишет. Продолжение основано
на PLAN APPROVE202 и независимых SOURCE APPROVE product204/205. Никакого setup,
spawn сервиса/браузера/worker/native, HTTP/MCP/config RPC, model turn, compile
или тестовых writes в synthetic runtime здесь не выполнялось. Разрешённые
syntax/static subprocesses проверяли только исходники.

**Жёсткая граница:** WORK206V3 FAIL остаётся. WORK207 выявляет protocol compatibility,
не подтверждает permissions/OS grants. Требуются disposition207 → normal206 PASS
→ actual single native developer commit и OS-negative enforcement → exact
candidate/source approval202 и отдельный root GO. `gate.authorize()` всегда
throws EXTENSION_PENDING до чтения input; нет env bypass/exported permit issuer.
Наличие complete source, public API205 или candidate8c не открывает этот gate.

Current explicit descriptor `pins.json`: revision
`8c43df9f408712526e223796905bba787e336d5e`, Mac binary SHA
`8726daa43fbbf93b026215aede6fb3d2e7b98ebdd06644af5cea12dc1e92622f`, Next
manifest SHA `e1e751583e1ae1c959675534d98645db9d8e278e1cc2346d4c9a832043a3cc6b`,
build `W34afpK9YF9oWNEIULNXf`. Старые3e6 поля plan.json помечены historical only;
execution_candidate теперь отдельный explicit8c descriptor. Новый product fix
потребует нового descriptor/review, silent reuse отсутствует. Mac Node24.19,
Codex/Git/Python/Chrome/Playwright hashes прочитаны статически; Linux Node из
Next payload не выбирается для запуска на Mac.179Playwright library files имеют
отдельный exact manifest; source/tool/fixture/Next pins проверяются до/после.

Complete source wiring (каждый executable source меньше300строк):

- Public `worker automation-preview --config PRIVATE` строит workspace identity
  и preset digest, проверяет unchanged draft/applied=false; Go helper не нужен.
  Будущий отдельный local apply включает только выбранные developer Git metadata
  и tester artifact opt-ins. Реальные config RPC выполняет только продукт worker
  при настоящем launch; новый QA inspection path не написан/не менялся.
- Actual public registry bootstrap, ordinary Git seed/empty hooks, browser login/
  project/четыре profile revisions1 и policy consent. Future worker config
  содержит реальные supported DTOs. Host active+lease и полный текущий host-scope
  читаются отдельно; truncated list не становится grant.
- Normal UI requirement + один real stdio MCP same-action replay; parent script
  не пишет workflow_plan, done/review/check/attestation и не создаёт пятую модель.
  Full report/list cursors дочитываются до конца; меняющаяся внутри пагинации
  version вызывает FAIL, не выдаётся неполная история за полную.
- Actual workflow/Git/JUnit assertions: одна materialized задача, четыре completed
  роли, разные nativeUUID/managedRun/boardSession, matching attempt tuples,
  reviewer/tester вне contributor runs разработчика, один developer commit с
  seed parent/единственным src/math_ops.py diff, byte/mode frozen tests/helpers,
  exact clean SHA/source manifest при review/test/exit. Два meaningful Python
  unittest cases реально запустит fixed preset; parser сверит raw JUnit hash,
  actual names/count2 и failed/errors/skipped0, exact check digest/SHA/generation.
- Server run_exit + command applied receipts дополняются exact local worker journal
  native_exited/exit_durable/managed_ack/exit_ack/lock_released и нулевыми pending;
  developer permission digest/executable/Git-directory identity отдельно связаны.
  Generic owned supervisor удаляет ровно один framing LF только после positive
  natural proof; config_inspection output path этим wrapper не обслуживается.
- Coordinator сохраняет initial/final FAIL/result в private root, raw config bytes
  before/after и private observations. Setup180s, happy900s, cleanup reserve60s;
  expired normal permit блокирует следующие actions, owned cleanup отдельно.
  НаFAIL — свой current workflow cancel CAS и own lifeline cleanup; unknown
  constructor/exit/ACK/manual не превращается в no-process/PASS. Нет PID/name kill.
  НаPASS — normal own policy revoke/host disable/profile archive, все owned exits,
  затем supported offline revoke только own fixture credentials. Runtime/journal/
  evidence сохраняются, user config restore/production mutations не выполняются.

Future root-owned input/normalized prerequisite receipt schema описана в private
README. Эти wrappers не product API и не существуют как подтверждённые PASS;
они должны происходить из independently reviewed actual receipts. Пока gate
не выдаёт permit, их нельзя использовать для обхода prerequisites. Source202
receipt должен дополнительно связать exact source manifest. Negative/fix/control
сценарии не включались и требуют отдельных budgets/source. Windows/Claude/
физический второйПК и production NOT-RUN.

Freeze: `.local/task014/pipeline203/source-pins.json`,28files, SHA
`a19c28c26bf1412cf06e63cfedfc65a606f9d2f6569dee86221a36874bb3eecb`.
17CJS `node --check` PASS,3Python AST PASS; max executable source172lines.
`SHA256SUMS` соответствует новому full set, прежний partial manifest сохранён
как `partial-SHA256SUMS`. Python tests не запускались, modules/main не выполнялись.
202 запрошен полный независимый SOURCE review; runtime остаётся NOT-RUN.

## 10. REQUEST CHANGES202 → V2 source-only correction

202 независимо REQUEST CHANGES originala19 по трёмP1: фиктивный catalog revoke
для legacy-only fixture; unbounded readline/private log buffering; Promise.race
cleanup continuation после deadline/final report. Root разрешил узкий SOURCE
REOPEN после завершения PLAN210. PLAN210 SHA870523cf… получил отдельный PLAN
APPROVE202; это не runtime approval203 и не перенос native prerequisites.

Original28files/source manifest
`a19c28c26bf1412cf06e63cfedfc65a606f9d2f6569dee86221a36874bb3eecb`
сохранены byte-exact в parent directory и перепроверены. V2 — отдельная
`.local/task014/pipeline203/v2/`; никаких product/API/config/provider изменений.

Исправления исходников, пока без process/unit/E2E runtime:

1. Credential lifecycle соответствует owning `projects/auth.go importLegacy`:
   setup создаёт legacy token, import не создаёт catalog credentials. V2 не
   вызывает projects token-revoke с raw legacy ID. После positive own server/
   worker exits выполняются supported CLI `token list`→`token revoke --id`→
   `token list` на своей defaultDB; exact actual ID и revoked=false→true проверяются,
   остальные returned token fields неизменны. Generic owned exit/receipt тоже
   нужен. Секрет не печатается, новый catalog credential не выдумывается.
2. Новый streams.cjs обрабатывает byte chunks до concat/UTF8 decode. Frame≤1MiB,
   stdout≤8MiB, stderr≤2MiB, общий private log≤10MiB на process; MCP inflight≤16.
   Общий log budget учитывает оба stream. Overflow/invalid framing/IO/callback
   failure ловятся внутри callback, ставят FAIL и закрывают только свой lifeline
   EOF. После cap данные drain/discard без роста буфера; logfd/handlers закрываются
   идемпотентно. Generic framingLF нормализуется только после positive proof;
   inspection path по-прежнему не введён/не менялся.
3. Gate хранит отдельные abortable normal/cleanup scopes. Setup/workflow phases
   tracked; HTTP/body операции получают scope signal и clamped timeout. Новый
   cleanup.cjs не запускает racing whole-cleanup continuation: общий mutable
   budget45s, далее scope seal запрещает новые CLI/HTTP business calls и aborts
   pending requests. Последние15s — closure только уже известных handles и join
   main phase. Все cleanup tasks await/settle; unmatched phase/constructor/exit
   остаётся incomplete/unknown/manual FAIL. Нет new commands после fence.
   Перед owner-only final serialization logs/streams/listeners detached, report
   глубоко frozen; unjoined body не может записать новый report или разрешённую
   mutation с fenced permit. Pending already-issued distributed action не
   объявляется завершённой: unknown status сохраняется, no-process не выдумывается.

Read-only after-pins/config evidence IO следует после resource cleanup и под
внешним whole-coordinator bound оператора; мутирующий scope не открывается вновь.
After config writes ограничены private report snapshots; отдельный owner-only
final writer может записать только results.json после истечения execution scope.
Все real native prerequisites и unconditional EXTENSION_PENDING сохранены.
Explicit execution descriptor всё ещё8c source provenance, protocol issue207
не трактуется как разрешённые права;208 pins молча не перенесены.

V2 freeze:30files,19CJS syntax checks PASS,3Python AST PASS, max136executable lines.
Manifest `.local/task014/pipeline203/v2/source-pins.json` SHA
`08bcbda11b2cc6c05d25c2c9591cf1d0b94c4fe0c004cb1cc6e79d90e5663aee`.
Private sibling v2-source-diff.patch сохранён отдельно. **202 SOURCE REVIEW
PENDING / runtime NOT-RUN**. Ни models/servers/browser/MCP/config RPC/help,
ни compile/Python tests не запускались; только syntax/static checks разрешены.

## 11. V3 narrow HTTP/UI follow-up — source-only

202 перепроверил все30V2pins и preservation a19, но запросил оставшиеся поправки:
HTTP/API response.text до cap, повтор ms budget в UI.closeOwned и orphaned
waitForResponse rejection при ошибке trigger. Root разрешил отдельный SOURCE
REOPEN V3; V2/08bcb… и a19… sources/manifests сохранены и сверены byte-exact.

Новая `.local/task014/pipeline203/v3/`:

- Общий response-text.cjs читает response.body по byte chunks. Проверка8MiB
  происходит **до copy/concat/UTF8 decode**, в том числе для decompressed fetch
  bytes. HTTP/API передают тот же exact AbortSignal и absolute end, что fetch;
  overflow/abort/error инициирует reader.cancel, его rejection сразу handled,
  bounded cancellation await/releaseLock выполняются в finally. response.text
  для этих двух клиентов удалён; бесконечный response не копится до проверки.
- UI.closeOwned фиксирует один absolute end до первого await. Остаток передаётся
  в ожидание SDK closure и затем actual child exit; near-deadline первый wait
  не даёт второму новый полный ms. Exit остаётся положительно проверяемым;
  timeout/unknown нельзя превратить в no-process/PASS. Close listener снимается
  в finally.
- UI.mutation использует управляемый response observer: immediate rejection
  anchor, Promise.all response+trigger с handlers в том же turn, scope abort/
  page close/deadline и безопасное удаление listeners/timer в finally. Trigger
  failure/timeout не оставляет orphaned rejected waiter, callback exceptions не
  выходят за managed error path. Это не меняет CSRF/action/payload/approval gates.

Права/ролевая цепочка/fixed tests/credential/stream caps и normal/cleanup fences
не расширялись. Exact8c descriptor теперь явно historical_source_only/runtime
blocked; текущий a32/Mac e9fc/Next fee16/build1tnR9 не подставлялся до foundation
proofs. Unconditional EXTENSION_PENDING не открывался. No inspection/native
config RPC path changes, no setup/server/browser/model/compile/test execution.

V3 full manifest `.local/task014/pipeline203/v3/source-pins.json` SHA
`49df98c83d9d5dff66013fa93f47620d639bd04e52582919104d9afdc467358a`;
31files,20CJS syntax PASS,3Python AST PASS, max147executable source lines.
Private sibling v3-source-diff.patch230lines; original a19 + V2 exact pins PASS.
**SOURCE READY → 202 review; EXECUTION NOT-RUN.** Новый runtime потребует final
source approval, actual foundation prerequisites, новый candidate descriptor и
отдельный exact root GO.

## V4 source adaptation219 — actual a32 foundations, execution still NOT-RUN

По ROOT SOURCE GO219 отдельный `.local/task014/pipeline203/v4/`; original a19,
V208bc и V349df immutable, V3 все31pins matched. Новый manifest33sources SHA
`8d76904a5cf55d17d712b79ea7a6e4f74917327fe3cd144ab91ce3a169f3ceb2`.
Static23CJS syntax+3Python AST+local requires PASS/max147lines. Candidate/tool/
Next1256files/library179files hashes matched; это readonly/static, не выполнение.
Sibling v4-source-diff.patch/v4-static.private.json сохранены private.

Pins теперь exact clean a32db22973d5b43d4fb227a89337bf5b82a4969f / Mac
 e9fcade02b8ea4b8e6acdb70d315e77b5e955ef27743920f96b33662f9afcaec,
Next build1tnR9bTxsi1TBLZ3mgtCe. Foundations читаются DIRECTLY из actual JSON
A2095af7cd0d…, C2167e35715d…, N2185b5092df…: exact candidate, positive owned
natural exit0/drained, strictconfig, реальные MCP/OS5/native event/Git поля.
Нет generic fabricated PASS receipts. Common49ed foundation digest относится
только к OLD Git identity; new fixture actual worker permission proof/digest
и ordinary.git identity остаются обязательными runtime assertions.

Shared219 один exact external manifest14795546…/4files, без divergent checker.
Root сообщил independent202 source APPROVE и pure14PASS; author выполнял лишь AST.
Parent сохраняет actual raw0600 config baseline/metadata до первого child,
заранее связывает единственный fresh private0700 own HOME checkout/internal.git,
проверяет ownership/ancestors/no symlinks/localconfig absence до native. Никаких
settings/trust writes/restore/RPC. Whole-cycle postcomparison после proven own
stop, bounded supervised Python3.14<=20s внутри45s cleanup BEFORE seal; final15s
не запускают helper. Только STRICT_UNCHANGED/QA_TRUST_ADDITION_ALLOWED по219;
любая missing observation/drift FAIL. Raw evidence private, public hashes/counts.

Human mutations используют actual owner browser JSON human session, MCP session
только scripted replay. Cleanup revoke и managed:disable — supported browser
CSRF/current action/version +authoritative REST readback; human nativeREST alone
не обходит localHostProof. Product authorization не менялась. Actual offline
legacy list/revoke/list и bounded streams/fences/owned joins сохранены.

Prospective gate требует private root-created source202 receipt +exact source/
shared checker/candidate/scenario/input SHA/expiry<=15min/64hex nonce root GO.
Atomic O_EXCL fsynced one-use claim ДО setup/spawn; replay невозможен. Fixed paths
в README; author НЕ создавал receipt/GO. Missing files EXTENSION_PENDING.
4native roles/one developer commit/independent exactSHA review+QA/real2tests/JUnit/
terminal ACK, budgets180+900+60 unchanged; own fixture actor approval и final proof
всё равно надо получить реальным worker. SOURCE READY / SOURCE REVIEW PENDING.
No setup/helper/native/UI/MCP/configRPC/model/compile/runtime executed; no settings
read/write. F6/C2/H1/B2/Windows/Claude/secondPC/production/GUI/Test remain NOT-RUN.

## V5 narrow202 fixes — no runtime

V4 full33pins/8d769 preserved. ROOT SOURCE REOPEN203 permitted exactly two
review findings. Separate `.local/task014/pipeline203/v5/` manifest35files SHA
`ff499bfc01790ef9ddc874ebeed2ac33e999d53ce2477face28b550e6bed769e`;
static25CJS+3AST/local requires PASS/max149lines. Pure31 synthetic receipt/gate
cases SOURCE ONLY/NOT-RUN by author, to independent202 after source approval.

P1 общий pure owned-success predicate требует wrapper exit0/no signal,
proof.proven===true/success===true/forced===false/typed integer native exit_code0,
no spawn/manual/stream failure и actual closed/drained stream. Его используют
Owned.facts и cleanup (SHA d3b038452edb96acb02a81bef84bbf5fe9ca3a89f33b88c25f0fb8a8fc26feaf).
Wrapper0 с native1/false/missing/forced/unknown больше не successful stop/cleanup.
Будущие F6/C2/H1/B2 adapters должны унаследовать reviewed correction; исторические
freezes не менялись. Pure cases не выдают mocks за реальные process evidence.

P2 перед219helper обязательны constructionKnown, successful UI/context/browser
close +actual browser0/no signal, MCP EOF0, allOwned strict success, phaseDone,
все предыдущие cleanup steps proven. Rejected browser close теперь не теряется
в allSettled. Неизвестный constructor/ошибка close/любой failed cleanup step даёт
explicit NOT_OBSERVED/FAIL/manual, без нового helper, затем seal+owned drain only.
Перед prerequisite assertion сохраняются actual safe booleans. Root GO/receipt
fixed names теперь203-v5; никто из author не создаёт действующее разрешение.
Все остальные grants/a32/shared219/foundations/flow бюджеты прежние.
SOURCE READY202; никакого runtime/helper/settings/model/CLI/import tests.

## V6 explicit positive UI/MCP tuples — source-only clarification

После ROOT P2 clarification V5ff499 full35pins сохранены. Отдельный V6 manifest
`32eda9a1fd1701b09da1a6034b5d5ceb2ef7dd5f6295f7190ffa02240c6b6b73`,
35files/.local/task014/pipeline203/v6; syntax25CJS+AST3 PASS/max150lines.
Actual UI.closeOwned возвращает Chrome code0/signalnull/fulfilled close calls/
pageerrors0; actual MCP EOF возвращает code0/signalnull/protocolfalse/streamsfalse/
drained. comparisonReady проверяет сохранённые реальные tuples сверх closure flags,
construction/steps/mainphase/allOwned gate. Добавлены11 source-only pure negative
cases actual tuple shape (absent/nonzero/signal/rejected close/pageerror/protocol/
failedstream/undrained), всего42 synthetic cases NOT-RUN by author до review.
Общий reviewed-pending predicate SHA
124da0d612f25937c7448f5ae650d14ea0877a2f68555111155de29a36c59989.
Fixed future GO/source202 names203-v6; никаких receipt/GO author не создавал.
Все прочие a32/foundations/shared219/grants/бюджеты прежние. SOURCE READY202;
runtime/helper/model/config/settings/CLI NOT-RUN, historical sources immutable.

## Actual V6 FAIL сохраняется; V7 narrow fixture-mode source correction

Root выполнил ровно один approvedV6 trial, results SHA
f78b65c6fe77619b820725c2d4252eee18850399f77a27ba62d9a3baf8c33f3a.
По root observation FAIL phase seed до любого model/server/bootstrap: umask077
превратил все6 requested644 trackedfiles в actual600; strict GitProof inventory
верно остановил сценарий. Byte SHA unchanged,12seedGit owned calls actual0/drained,
config STRICT_UNCHANGED094/manualfalse. Это не automation/runtime PASS; original
runtime/source/nonceclaim сохранены, automatic retry не выполнялся.

ROOT SOURCE REOPEN203V7: отдельный `.local/task014/pipeline203/v7/`, manifest
`ef6de7300890b28fc39431da06f8cca02922518f9bf0a53b31f1f56617aef2a4`,
37files; syntax27CJS+3AST/local requires PASS/max150lines; V6 все35pins unchanged.
Новый fixture-copy SHAe5f60fcf99299274d1f310252487149efc38aa5b4bbf430266bc3659c189d877
создаёт только fresh own tracked file под canonical0700 checkout, проверяет
parent ownership/no links/0700, O_EXCL|O_NOFOLLOW, пишет exact reviewed bytes,
затем fchmod(record.mode) на том же новом fd и fstat regular/owner/nlink1/mode/
size; последующие lstat identity/mode+rawSHA и full inventory equality ДО seedGit.
Не меняется umask077, assertions/modeguard не ослаблены, existingfile не трогается.

PureFS regression SOURCE ONLY/NOT-RUN by author: свой fresh private temp при077,
копирование всех6 reviewedfiles, exact manifest bytes/modes/no symlinks, отказ
overwrite; удаляется только этот временный test root. Никаких Git/CLI/spawn/
server/model/network/settings. Нужен202 source review затем independent pureFS run.
Future F6/C2/H1/B2 adapters обязаны перенести approved fixture-copy correction,
исторические freezes не редактируются. New rootGO/source202 names203-v7, actual
receipt/GO не создавались. SOURCE READY202; actual new runtime GO отсутствует.

### V9 source freeze — exact wrapping-label controls and ordered browser closure

Root reopened SOURCE ONLY after actual V8 FAIL report `1ec207812d7ee08579c8e6ce40412b68b9af197c3eca32ba0a27125f2d9fcd50`: diagnostic stage TEAM_ANALYST_PROVIDER/selectOption, zero native/model/MCP/worker attempts. Original V8 evidence/source remain immutable. Pinned Playwright coreBundle getElementLabels/elementText uses full descendant label text; CSS text-is uses exact normalized immediate text nodes. Actual JSX wraps SELECT/OPTION and nonempty TEXTAREA in labels. V9 uses scoped `label:text-is(exact static literal) > select/textarea` for all six SELECT and both TEXTAREA callers, preserving strict uniqueness/actions and existing POST/payload assertions. Product source unchanged.

Browser close now awaits context close before server close within one absolute deadline. Server remains attempted after context rejection/timeout when budget remains. Finite context/server statuses are retained in cleanup; rejection/timeout remains FAIL. Existing strict actual browser exit0/signalnull, drainage, generic owned-success and Next-only closure predicates remain unchanged. Parallel-close causality of the old initial AssertionError remains unproven; final join0 does not convert its first failed step to PASS.

Frozen `.local/task014/pipeline203/v9/source-pins.json`: SHA256 `27711ee68da01137a708029acd89813f39cb514a4d086f40485dcbf891543035`, 47 files; V8 all44 unchanged verified. Syntax36 CJS/AST3 PASS; max158 lines. Eight pure boundary tests authored NOT-RUN by author, independent202 review requested. New source202/rootGO paths use 203-v9 names, existing exact scenario/candidate/nonce/input/expiry schema unchanged. No author runtime, setup, server/browser/HTTP/CLI/native/model/helper/settings or actual tests. Native gate remains CLOSED until exact independent approval and new ONE root GO.

### V10 — source fixture packaging metadata correction

V9 actual trial FAIL/context before any child/config/model, report `ccaf389740443aea8d48364f3e51bf651a2ec8d98feed65e470ba915350c2d59`, preserved. Author blanket chmod600 during V9 freeze incorrectly changed all six source fixture modes; exact bytes remained unchanged and inventory guard correctly failed. V10 retains source fixture descriptor0644 (420) under0700 parents, and explicitly asserts complete six-file inventory SHA+mode equality BEFORE freeze and afterward. Guards, descriptor modes, all business/API/UI/close helpers are byteexact V9. Only README/version plan/auth filenames change bytes; new fixed gates203-v10 require independent review/new ROOT GO. H1/F6 authors notified to preserve source fixture modes in new copies.

V10 manifest `.local/task014/pipeline203/v10/source-pins.json` SHA `26534f4a996b2884c764d3bafc67a3f7d807f937343998d055e3fb9340a1b149`,47pins; V9 original47 hashes unchanged. Static36 CJS/3AST PASS, complete fixture inventory exact, seven tool/179 library/1256 Next files plus three foundations/shared checker source hashes verified read-only. V9 pure tests byteexact reused, no author execution/import/runtime/model/browser/HTTP/settings/helper/CLI. SOURCE review202 requested; new runtime gate CLOSED.

### V11 — strict equivalent UTC-minute expiry representation

Root actual V10 trial passed four profiles/enrollment/worker and policy POST200, then strict payload comparison failed before intake/models. Root forensic of the closed own V10 fixture establishes only expiry representation difference, same UTC instant/minute. Runtime was pipeline203-1792791570662904000. A completed workflow is not established. V10 source/evidence preserved.

New V11 saves original private own policy submission receipt BEFORE JSON parsing/assertion. Shared policy-settings helper retains strict deepEqual for every setting, only permitting expires_at exact valid UTC-minute forms `YYYY-MM-DDTHH:mm:00Z` and `YYYY-MM-DDTHH:mm:00.000Z` at the identical instant. Exact spelling length, calendar round-trip and unchanged full settings required; offsets/fractions/nonzero seconds/rollovers/missing/type/any other field differences FAIL. No product/authority/deadline change. All other business/UI/close helpers unchanged.

Frozen `.local/task014/pipeline203/v11/source-pins.json` SHA `3bed25742e8aef53aaab5130143f5bd7798f63749518beb55999db3305a42f97`,49pins. Static38 CJS/3AST PASS, six actual source fixture0644/SHA inventory exact; seven tools/179libraries/1256Next/three foundations/shared219 pinned inputs matched read-only. Six pure boundary tests authored NOT-RUN by author, independent202 review requested. New fixed203-v11 review/GO paths; runtime remains CLOSED until separate exact approval/rootGO. No author native/model/browser/server/HTTP/CLI/helper/settings execution.

### V12 — separate patched worker with old exact a32 server/QA outer supervisor

Per approved WORK222/223, new worker be95daed7153f4abeb72012286cc748d0596b479/MacSHA9755ed0fb0de869751635ba4e5adf1a4d742b0a202e6659e871d337fceed27ff is a distinct artifact. Server/bootstrap/token/scriptedMCP/Next/QA outer supervisor remain exact a32. Worker identity/automation-preview/enroll/run use privately copied workerCandidate, old c.candidate supervises it. Worker self-exec supervisor/bridge physically use NEW binary via unchanged os.Executable sources. Report separately labels server/QAouter/worker/selfexec revisions and SHAs; old foundations unchanged and never relabelled as new worker proof. Profiles/prompts/limits4/UI/cleanup/assertions untouched.

V3 provenance d98cd715 and independent V3 receipt80bfcc79 bind source2/live+immutable snapshots, unchangedinner3/live+snapshots, exact base→worker patch08bd, immutable contract3476, buildinfo/race/report. Context verifies original/copiedworker and provenance before/after; each worker command checks pinned executable. RootGO/input exactworker_revision/worker_sha256 mandatory; new203-v12 source202 receipt additionally binds worker_provenance_sha256 and worker_source202_sha256. No author GO/receipt or runtime.

Frozen `.local/task014/pipeline203/v12/source-pins.json` SHA `76bb7e5a460249026a3e83569c8b683bc09e99709f58e9cb3061829ddbd25604`,51pins. Static40CJS/3AST/max187 PASS; fixture six actual0644/SHA inventory exact; alltool/libs/Next/foundations/shared219/worker provenance inputs verified read-only, V11immutable. Two pure worker-binding test functions with19 negative mutations authored NOT-RUN. Whole source delta sent202, gate CLOSED pending exact approval/new ONE root GO. H1/F6 authors informed canonical workerCandidate/helper/schema for later reviewed inheritance.

### V13 — complete synthetic positive worker-binding unit baseline

202 V12 full review found only missing three mandatory receipt fields in pure sample: source_files (independent structuredClone), base_to_worker_diff_sha256 and contract_path. V13 fixes exactly synthetic baseline plus new version/auth names; worker-proof/context/worker routing production helpers remain byteexact V12. Frozen manifest `.local/task014/pipeline203/v13/source-pins.json` SHA `2ce92a5c4a6c2567664ee14148adff03b4b9460e5015436be8dde62286005fc5`,51pins; static40CJS/3AST/fixture6actual0644 inventory/pinned inputs PASS, V12immutable.

Root separately authorized author pure metadata-only groups. First external VM loader FAIL preserved at v13-pure-1792794014593982000 (strict arrays crossed separate VM realms; source unchanged). Correct same-realm CommonJS custom-require loader exposes only metadata, private runtime methods throw; PASS2 groups/19 negatives,20 metadata calls/0 forbidden, exit0. Receipt `.local/task014/pipeline203/v13-pure-cjs-1792794049731127000/receipt.json` SHA `249ac26672f8429a3134cc246d9659421d629262a75a57c1ebc8aab01295df01`. Frozen manifest/guard/test hashes unchanged before/after. No validate/executable/context/main, candidateCLI/native/model/browser/server/HTTP/settings execution. Independent202 narrow source review/repeat requested; runtime still requires fresh rootGO.

### V14 — minimal workflow-list contract correction; recovery source stopped

Actual V13 remains FAIL (`430ac29c0896fa35f7d1f81cbdb40cb8d106a02191499570845f5ec0dd100b14`), despite four actual native roles completing. Independent ops evidence `71ac00d38dbb4f508bfc626978897ef30e8b59ca9bf985b4d73d6fd8af436b87` classifies the first terminal read as QA workflow-list limit100 versus supported max50. Its closed-fixture verification of subsequent data does not retroactively execute missing original assertions/finalUI/revokes.

Root stopped the larger additive readback recovery at SOURCE_WIP_NOT_REVIEWED_NOT_RUN. Its CJS/three Python drafts remain untouched, no imports/helper/SQLite/clone/server/browser/native execution. Private checkpoint `.local/task014/pipeline203/readback-v14-source-wip.json` SHA `9a07d5fadf28583663aae71f1c376b8455860162d80e324c25aab59c494c7d78` preserves all source hashes; this WIP is not part of an acceptance gate.

New minimal `.local/task014/pipeline203/v14/source-pins.json` SHA `19436f9bf1df8951db5d5ccb5dea9f06136442bb8a852688c61b07a9e8a4c08d`,52pins. Sole API behavior delta: workflows list100→50; report100/cursor/duplicate guards unchanged. Fixed private review/GO names203-v14 and source version/README updated; pagination pure6 test copied byteexact from WORK225 (`3ec819c950a8087c767205431d309415289620f95ad9427ebc312abf2482b5e9`). All worker/provenance/routes/profiles/assertions/Git/journal/UI/cleanup remain byteexact approved V13. Optional common profile change deliberately omitted.

Static PASS41CJS syntax/3Python AST; full six-file source fixture actual0644/bytes exact; seven tools/library/Next/foundations/shared219/worker provenance and immutable inputs1503 matched read-only. Original V13 all51 source hashes unchanged. Private diff/static records retained. Pure test/runtime/config/helper/CLI/browser/HTTP/model NOT-RUN by author. SOURCE READY202; exact independent review and separate fresh one-use root GO required for another four-role trial. Author wrote no review/GO receipt.
