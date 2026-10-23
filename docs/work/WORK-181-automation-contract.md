# WORK-181 — полный durable automation contract

Автор: managed_worker. Область записи: эта WORK и `docs/automation.md`.
Статус: CONTRACT SOURCE READY для независимого architecture/security review169;
реализация TASK014 не начата. Root владеет общей TASK014 и распределяет source.

Прочитана действующая модель core/managed/coordination/profiles; Store.Create/
Next/Transition/completion/CheckCreate/gates; managed session/poll/report/terminal,
registry managedCall/GuardManagedCredentials, transport operations и bridge binding.
Контракт фиксирует новые типы явно: существующих workflow fields в current core нет.
176 остаётся SOURCE APPROVE diagnostic-only, а не fixed root cause. Actual
config-only177 PASS не заменяет native pipeline; реальный повтор GUI failed до
conversation с prepare_discovery_timeout без ownership suffix. Причина разницы
между helper probe и actual engine ещё не установлена; это отдельная приёмка.

## Принятый полный контур

Одна заявка из Accord либо unbound Codex/Claude client, разрешённая ограниченной
policy, запускает настоящего аналитика; его план материализуется в задачи,
которые по очереди исполняются разработчиком, независимым reviewer и tester.
Negative outcomes автоматически возвращают работу с defect evidence в пределах
лимитов; вопросы/неясные outcomes/превышения останавливаются у человека.
Это не fake очередь: каждый переход требует task evidence, current core gates,
реальную native conversation/lease и qualifying worker observations.

Default sequential checkout — режим исполнения полного контура, не сокращение
до демонстрационного analyst/developer MVP. Parallel worktrees/merge, external
triggers, remote push/deploy и неподтверждённые денежные hard budgets отложены явно.

## Единые DTO и safety context

Все перечисленные DTO proposed для `internal/core/automation.go`; field names —
wire contract, неизвестные/повторные/null-required поля отклоняются. Project ID
приходит только из scoped route/authority. Никаких произвольных paths/actors/argv.
Refs ниже — safe IDs, не bearer/host proof. Общие defaults/limits в automation.md.

| Тип | Поля |
| --- | --- |
| RoleBinding | provider:codex/claude, profile_id, profile_version; role фиксирован ключом mapping; provider/profile сопоставляются managedSnapshot rules |
| CheckPresetRef | preset_id, local_version, local_digest, kind:test/lint/build, required |
| AutomationLimits | plan_tasks, fix_per_task, fix_total, native_invocations, continuation_per_step, step_seconds, workflow_seconds, grant_seconds |
| AutomationPolicy | id, version, state, name, issuer_principal_id, issuer_epoch, host_id, host_owner_epoch, host_consent_version, workspace, role_bindings{analyst/developer/reviewer/tester}, intake_principals[], control_principals[], client_intake, checks[], git_policy:local_commits, limits, expires_at, created_at, updated_at |
| PolicyCreateInput | request_id, name, host_id, host_consent_version, workspace, role_bindings, intake_principals, control_principals, client_intake, checks, git_policy, limits, expires_at |
| PolicyUpdateInput | PolicyCreateInput + policy_id, version; new immutable revision, never silent workflow rebind |
| PolicyRevokeInput | request_id, policy_id, version, reason |
| WorkflowSubmitInput | request_id, policy_id?, policy_version?, text; missing policy→draft only; channel attribution server-generated |
| WorkflowGrant | policy_id/version, issuer_id/epoch, requester_id/epoch, host_owner_id/epoch, host_consent_version, selected snapshots/check refs, limits, approved_at, expires_at |
| Workflow | id, version, control_epoch, state, reason_code, intake_text/hash, requester_id, grant?, epic_key?, plan_revision?, current_step_id?, counters, deadline_at, created_at, updated_at, report_ref? |
| WorkflowStep | id, workflow_id, sequence, kind, status, attempt_no, task_key/version/generation, profile_snapshot, target_sha?, managed_run_id/generation?, native_session_id?, started_at?, deadline_at?, ended_at?, evidence_refs[], contributor_run_ids[] |
| WorkflowPlanTask | client_key, title, description, areas[], acceptance[], prerequisites[], required_check_ids[]; developer role implicit |
| WorkflowPlanInput | request_id, workflow_id, version, task_generation, tasks[]; workflow/step binding must equal authenticated native session |
| WorkflowQuestionInput | request_id, workflow_id, version, task_generation, question, options[]?; only own bound stage |
| WorkflowQuestion | id, workflow_id, step_id, version, human_task_key, designated_principals[], state, question, options, answer_ref?, created_at |
| WorkflowAnswerInput | request_id, workflow_id, version, question_id, question_version, answer, resume; human claim/answer generation handled atomically by shared domain helper |
| WorkflowCommandInput | request_id, workflow_id, version, kind:pause/cancel/resume/clarify, text?, refresh_policy_id/version?; refresh requires explicit current human approval |
| WorkflowReadInput/Page | workflow_id? / state?, cursor?, limit?; list max50, detail steps max100/page, opaque project-scoped cursor |
| WorkflowAttestInput | ManagedPollInput + command_id, dispatch_id, run_id, generation, workflow_id, step_id, control_epoch, task_key, task_generation, native_session_id, kind, check_request_id?, preset_id/version/digest?, observations; request replay as worker report, host_secret hashed before storage |
| WorkflowObservations | actual_head?, target_sha?, source_clean?, manifest_sha256?, check_exit_code?, checks_passed/failed/skipped?, report_sha256?, artifact_refs[], outcome_bytes_truncated; no raw command/path/env/transcript |
| WorkflowAttestResult | check_id?, status, control_epoch, max_runtime_seconds, reservation_ttl_seconds, replay flag; durations constrain local monotonic horizons |
| ExecutionCapability | id, digest, host_id, workflow_id, step_id, control_epoch, run_id/generation, worker_epoch, role, profile_id/version, state, operations[], created_at, expires_at; project implicit board scope |
| ExecutionCapabilityRegisterInput | ManagedPollInput + command_id, dispatch_id, run_id, generation, workflow_id, step_id, control_epoch, capability_id, digest; scope/deadline derived server-side, no secret |
| WorkflowAdvanceInput/Result | workflow_id, expected_version, control_epoch / workflow snapshot + next_due_at; internal only, no native/browser binding |

Policy expires_at обязателен: default24h, hard maximum7d и не позднее
локального automation consent expires_at. Update — новое явное approval, не
неявное продление. WorkflowGrant.expires_at=min(policy.expires_at,
approved_at+grant_seconds,local_consent.expires_at); replay не продлевает ничего.
Policy approval требует хотя бы1 required kind:test preset. Каждый developer
plan task содержит ≥1 такой preset и весь policy mandatory set; пустой список,
optional-only, omission и lint/build-only reject, а не vacuous QA passed.
Budget counters резервируются при atomic enqueue; replay не расходует их снова.
Фактический started count ведётся отдельно, отсутствие ACK не возвращает budget.
Fix counters растут при reopen один раз; task budget считает созданные задачи
всех plan revisions, cancelled/history не дают бесконечно освободить квоту.
Дополнительно max8 plan revisions и16 questions/workflow, options≤8;
human answer≤16KiB. Capacity/refill — только явное current human action.

Kind attestation: check_reserved/check_started/check_finished/run_exit.
`run_exit` допустим только вместе с existing managed terminal proof для точного
worker epoch/command dispatch/run generation. Check reservation требует active
workflow/step/grant/task lease. Report не может выбрать другой task/session и
не является новым способом получить аренду. Core lease generation и managed
run generation — разные поля. Native session ID — observed provider UUID.

Safety helpers — server-only context, по образцу WithManagedPrincipals:

- `WithAutomationAuthority(ctx, snapshot)` предоставляет current project/registry
  write state и map current project-local principals→epochs/legacy validity.
  Его создаёт только projects registry wrapper, пока держит registry→legacy locks;
  внешнего JSON/HTTP setter нет. Snapshot не является сохранённым credential.
- `WithAutomationPolicyApproval(ctx, approval)` подтверждает trusted browser
  issued action+CSRF/current actor либо locally proven CLI owner consent.
  Role human, строка actor или native Describe сами не создают этот context.
- `authorizeWorkflow(b,ctx,w,operation,now)` сопоставляет current caller/requester/
  issuer/owner epochs, policy revision/expiry, host consent/control и exact scope
  до replay и внутри advance; возвращает unexported narrow domain capability.
- `workflowBoundSession(b,a,workflow,step,task_generation)` выводит bindings из
  stored session→managed run→step и проверяет текущую own task lease/profile.
  Managed bridge injects immutable fields; поля модели никогда не authoritative.
- `createWorkflowTask/check/reopen/enqueueInBoard` разделяют проверенную логику
  current Create/CheckCreate/Transition/managedNewCommand; нет nested transactions,
  fabricated Session или обхода completion. Task finish только от native holder. Административные shared helpers принимают
  explicit role/capability context; не конструируют *core.Session даже временно.
- `qualifyWorkflowStep` вызывает current successful/latestVerification/epicClosable
  и сопоставляет task SHA/generation/evidence с worker attestation; не отдельная
  ослабленная копия core gate evaluator.

Requester должен быть и policy intake principal, и current server/local host
controller; policy issuer/owner также остаются действующими. Третьему editor
этот contract не выдаёт control автоматически. Command.CreatedBy — initiating
requester; audit дополнительно хранит policy approver/epoch, не подменяет его.

## Child credential и initial setup

Для automated child прежний TokenFile owner bearer заменяется private execution
capability file. Worker сам генерирует random256bit secret и fsync private file
до регистрации через owner bearer+human management session+host proof. Сервер
хранит digest и tuple(project,workflow,step,control_epoch,run,generation,worker_epoch,
role,profile_id/version), deadline≤attempt deadline и allowed operation set.
Регистрация идемпотентна по run/gen/request/hash; raw secret не возвращается/
не сохраняется сервером, поэтому lost ACK повторяет тот же digest. Новый run/gen
не использует старый cap. UNIQUE(run_id,generation): один cap, повтор registration
с иным digest конфликтует. Maximum128 active и4096 total cap records/project,
включая revoked history; capacity заранее видна, auto-delete/reuse отсутствует.

Transport различает execution token от project bearer; native principal сохраняет
действующий host owner local identity, а auth context несёт immutable capability ID.
Session получает required execution_capability_id, сопоставляемый при каждой
операции: cap одного child не открывает/не использует session другого. Describe
с omitted binding всё равно forced-bound, mismatch reject; role/profile нельзя
выбрать заново. Routes registry/catalog/host/worker/policy/intake/control для cap
закрыты; own workflow task/plan/question/check reads и permitted native mutation
проверяются не только tools/list, но на сервере. Смена session не расширяет cap.

Worker/owner lease, host state, grant/current epochs и deadline проверяются при
каждом native request; process terminal/revoke/pause invalidates cap. Parent owner
bearer и host proof не передаются в child config/env/argv. Private nonce IPC
отдельный от execution cap. Серверная scoped-credential гарантия не является
защитой от malicious same-OS-user, способного украсть иной private owner token;
не обещается hard OS credential isolation при trusted_owner_process режиме.

Один host/workspace на workflow, но каждый RoleBinding выбирает Codex либо Claude
из local enrolled providers. Mixed roles не требуют cross-PC Git transfer.
Первый shared intake UI при отсутствии profiles показывает «Подготовить команду»:
четыре editable defaults analyst/developer/reviewer/tester, provider выбирается
владельцем; existing profile CRUD не выдаёт OS rights. Для каждого create свой
stable action/request receipt; partial success показывает созданные exact IDs,
unknown outcome сохраняет исходный receipt, не создаёт duplicate под новым ID.
После expiry/unrecoverable unknown пользователь выбирает existing IDs явно;
имена не уникальны и не используются как recovery key. Policy approval и start
возможны лишь после проверки всех четырёх current active RoleBindings.

## Worker proof, IPC и права проверок

Parent worker создаёт per-attempt socket/named pipe и nonce, записывает binding
в immutable private bridge config с fsync до native child. IPC frame не принимается
на browser/HTTP endpoint. Один request≤64KiB, ответ≤16KiB, protocol v1, strict fields,
nonce comparison constant-time; максимум4 одновременных connections/1 active check
на step. Local lifetime/auth и schema подробно определены в automation.md.
Run/workflow/step/control epoch/task/session/lease generation проверяются на каждом
frame. Нет endpoint «выполнить команду», перечисления env или чтения произвольного
файла. Доступ того же OS owner к private nonce не объявляется защитой от malware.

Tool check запускается асинхронно: server authorization/reservation → local fsync
intent → owned static process → journal fsync result → idempotent report. Потерянный
IPC/HTTP ответ повторяет тот же key/payload; ambiguous intent требует recovery,
не второй spawn. Output/result resources bounded/redacted, raw transcript local.
Actual check result требует exact preset digest, target SHA и current lease scope.
Все check subprocess handles принадлежат worker; StopAll/host revoke/workflow pause/
local deadline остановят и native CLI, и checks. Result valid только после proven
exit; terminal handoff ждёт все owned scopes и удерживает checkout lock до durable
proof. Crash после результата, но до ACK, сохраняет pending; crash без proof не
объявляет exit/finished или свободный workspace. Network/IPC не держат global mu.

**Отдельный security boundary:** fixed argv запускает repository-controlled код.
Это не наследует sandbox/approval Codex или Claude. Каждый local CheckPreset
по умолчанию disabled и обязан иметь явный owner-approved isolation mode:

- `enforced_launcher`: конкретный проверенный platform sandbox/container launcher,
  immutable local executable/argv и его declared boundary. Неподдержанная ОС или
  неподтверждённый launcher→blocked; сама настройка имени не доказывает containment.
- `trusted_owner_process`: отдельное видимое согласие исполнять trusted repo code
  с правами OS owner для этого workspace+конкретного preset digest. Default off,
  не выводится из workspace-write, member/controller или native tool grant.
  PGID/Job supervision не означает filesystem/network sandbox; риски доступа
  процесса за пределами checkout и Unix detached descendants указаны прямо.

Server policy выбирает только уже разрешённый preset. Включить unrestricted mode,
сменить executable/args/isolation, повысить timeout или переписать local consent из
модели/браузера нельзя. UI показывает фактический mode и необходимость local setup.
Минимальный полный contour допускает explicit trusted-code presets после такого
consent; enforced cross-platform sandbox не объявляется уже реализованным.

Git/source proof: fixed local Git executable, canonical repo/workspace identity,
full HEAD SHA, tracked files равны HEAD, manifest source roots; untracked/ignored
файлы вне явно разрешённых dependency/cache/artifact roots входят в baseline и
не могут неожиданно стать новым кодом. Artifact roots должны быть non-tracked и
не пересекаться с source roots; модель не добавляет exclusions. Dependencies не
выдаются за reviewed source: их approved local boundary/version фиксируются.
Reviewer/tester source mutation блокирует advance, даже при passed/approve;
ничего не reset/clean автоматически. Worker читает manifests до/после check/step,
а attestation фиксирует exact artifact scope и truncation, не «все файлы ОС».

Обычные unit/integration tests пишет developer до review; test_design включается
только при необходимости новых process/E2E после независимого source approve.
Новый test patch сперва artifact-only, source checkout остаётся чистым. Developer
применяет/коммитит patch, reviewer проверяет новый commit и включение тестов в
preset, затем tester исполняет reviewed tests на этом SHA. Attestation checks:
required test preset требует bounded parsed JUnit/TAP report, хотя бы1 выполненный
test и отсутствие failures; zero-tests/invalid report→not_run/blocked. Exit-only
lint/build presets разрешаются явно, но одни не квалифицируют testing stage.
Worker parsing не доказывает содержательность assertions; независимое review
и native tester findings обязательны. Заранее отсутствующий preset→вопрос владельцу.

## Миграции, восстановление и допуск

Schema6 содержит DDL и validators всех новых tables/required bindings, loadBoard/
persistBoard/clone/replay paths, quotas и unique execution keys; old rows unchanged.
Capabilities automation1 + все действующие floors. Миграция только capable
all-project schema5 prebackup/restore/roll-forward, без old-binary/DB rollback.

Full restore сохраняет workflow/task/plan/evidence/audit history, но инвалидирует
policy/delegation/host execution authority в том же capable reset контракте:
registry epochs растут, hosts disabled/proofs revoked, active workflow grants
revoked, control epochs растут. In-flight flows/runs маркируются needs_recovery/
paused с unknown ownership, не auto-dispatch. Снятие task lease/reset само по себе
не доказывает остановку физического worker или свободу checkout/reservation.
Нужны local owned-process recovery, fresh enrollment/consent, явный human grant
refresh при resume; старые proofs остаются историческими, late reports fenced.
Terminal completed history сохраняется как история, не право нового исполнения.

Одинаковое explicit refresh требуется после policy revision/issuer/requester/owner
access epoch changes. Допустим прежний plan/checkpoint, если он заново проверен;
сроки/профили/grants не переписываются задним числом. Нельзя автоматически оживить
старые очередь, token, nonce, request replay или applied status при новом доступе.

Implementation split после contract APPROVE:

| Автор core/store | Root / отдельно назначенные owners |
| --- | --- |
| core/automation.go DTO, shared binding additions | projects registry authority snapshot + scheduler lifecycle |
| store/automation_* state/CAS/plan/questions/control/grants/evidence/reservations | transport/automation.go, REST/CLI/MCP schemas/adapters |
| shared in-board create/check/reopen/managed authorize helpers | bridge fixed workflow scope + local check IPC adapter |
| migration/schema/load/persist/backup signatures, domain unit/integration | worker static checks/attestations, infra capable all-project update |
| no writes в web/worker/project wrappers без нового scope | web/browser/Next UI/action recovery; отдельный QA process/E2E |

Contract review отдельно от реализации. После approve координатор фиксирует
точные DTO/helpers и non-overlapping source ownership перед параллельной работой.
Полная приёмка включает оба intake, actual analyst plan+developer commit+separate
review+actual tests+negative fix cycle, questions/control, crashes/revocation/
restore/quotas и native Windows/разные физические ПК как отдельные статусы.
Ни один doc/schema/test fixture не объявляет pipeline работающим.
