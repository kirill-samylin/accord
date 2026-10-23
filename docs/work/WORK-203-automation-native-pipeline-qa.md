# WORK203 — full native pipeline QA, source plan

23.10.2026. Отдельный QA; root координатор, независимый reviewer202.
**SOURCE PREP ONLY / EXECUTION NOT-RUN.** Ни servers/workers/models/MCP/config RPC,
SSH, production или GUI не запускаются и не меняются. Большой harness пока
не реализован: сначала root/202 проверяют этот minimum plan.

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
