# WORK-185 — automation core/store

Автор: managed_worker. CONTRACT APPROVE183: docs/automation.md и WORK181.
Статус: SOURCE READY / независимое ревью ожидается. Код frozen; author proofs ниже.
Это готовность core/store к ревью, не TASK014 closure и не native PASS.

## Реализованный контур

Schema6 и automation metadata/tables/UNIQUE guards с pre-schema5 backup+checksum;
строгая проверка полного schema6 и точного SQL constraint включая quoted JSON paths.
Load/save/reset сохраняют задачи, grants, plans, steps, questions, worker facts,
capability history и idempotency. Restore отзывает policy/caps, увеличивает control
fences, оставляет needs_recovery и workspace reservations без выдуманного exit.

Policy CRUD требует current guarded identity/epoch и human approval (browser issued
CSRF/action либо local owner proof). Actor/authority/approval не external DTO.
AutomationHostConsent.Scopes[alias] — authoritative intersection; flat summary
не даёт прав. Общий version и private host consent watermark предотвращают повторное
использование старой версии после disable/reenable. Профили/поставщики/check refs,
expires/limits сверяются именно с выбранным workspace. Policy/grant <=7days,
local consent может быть дольше, но не расширяет policy/grant. Явный optional zero
fix/continuation budget не увеличивается нормализацией.

Требование UI/unbound native -> server-owned workflow -> native analyst lease/plan
-> атомарная materialization DAG -> developer -> independent review -> tester
-> negative evidence/reopen/new generation -> review/test again -> report/completion.
Применяются реальные существующие task/lease/generation/SHA/verification gates.
Create/CheckCreate/Reopen используют shared in-board helpers с explicit role;
никаких fabricated Session. Завершение задачи остаётся только у native holder.

Положительный agent outcome/CLI exit0 не достаточны: нужны server task evidence,
worker clean-source/actualHEAD/manifest + run-exit proof; для QA каждый обязательный
check должен реально закончиться, kind:test требует JUnit/TAP >=1 passed, 0 failed.
Check intents имеют immutable preset/version/digest/native/task tuple; summaries
не содержат argv/paths/env. Artifact refs — sha256:/artifact: hashes. Test-design
artifact outcome отправляет в developer fix -> новый review -> execution, не PASS.

Intake replay project/principal/epoch/request stable между сессиями/transport.
Current permission проверяется до replay. Истёкшая original session может вернуть
только существующий receipt; fresh mutation требует live session, replay её не
оживляет. Capacity/history fail закрывает advance без удаления audit. За lifetime
128 managed runs резервируется весь оставшийся бюджет; очереди одного workspace
исполняются последовательно, manual launch уважает reservation/capacity.

## Контроль и scoped native credential

Pause/cancel/revoke/archive запрашивают owned stop; paused/cancelled только после
proven exit. Unknown ownership остаётся needs_recovery. Archived guarded scheduler
не dispatch, но видит очередь, отзывает caps и обслуживает stop/terminal plane.
Worker finish/run_exit facts могут закрыть УЖЕ owned attempt после pause/archive;
новый reserved/started check требует current write grant + live own task lease.

ExecutionCapability: digest всего ac_exec1_ token, секрет остаётся local worker.
Exact workflow/step/control/run/gen/worker/role/profile/session scope enforced при
native Describe/Next/Update/Transition; нет child recursion/admin/unbound Describe,
другой workflow не захватывает задачу даже с тем же profile. Native conversations
для разных roles/runs отдельные, continuation сохраняет точный provider ID.
ManagedAutomationBinding.TaskGeneration — expected NEXT lease generation; до
claim actual=expected-1, после own Next=expected. Same capability не может снова
claim после истечения task lease. Step actual generation обновляется server Next.
GuardAutomationCredentials маскирует target cap только для DEFAULT legacy proof;
callback сохраняет исходный target scope. Viewer read snapshot отдельный от writers.

Question создаёт durable designated human task и останавливает current native.
Ответ — реальный human claim/completion, generic task answer синхронизирует inbox;
открытый вопрос нельзя миновать resume. Answer до stop не делает paused.

Согласованное root уточнение реализации: clarify ordered audit, next-turn delivery;
Step.DeliveredMessages меняется только после observed native started соответствующего
prompt MessageCount. Unfinished task -> controlled exact-session continuation с
fresh board session/cap/generation и budget; completed + pending clarify -> paused
clarification_after_completion, нужен explicit human replan. Replan только paused,
с причиной/согласованием прежнего результата; старые epic/tasks/plan/steps/evidence
сохраняются как история, бюджеты не обнуляются. Никакого queued==applied.

## Интерфейсы и ownership

Core DTO: automation_{policy,workflow,evidence,capability}.go. Store exports:
AutomationPolicyCreate/Update/Revoke/Get, AutomationPermission;
WorkflowSubmit/List/Get/Report/Command/Plan/Question/Answer;
AutomationCandidates/AutomationTick; ExecutionCapabilityRegister/
AuthenticateExecutionCapability; WorkflowAttest.
Authority getters/setters и trusted Approval server-only context. Registry lock/
legacy guard/root transport/worker/UI/infra принадлежат другим исполнителям.
AutomationPermission содержит Workspace/CommandKind/RefreshPolicyID+Version/
QuestionID для текущих cached browser permissions, не mutation/CAS retry.

## Проверки и границы

Авторские unit/integration — настоящая SQLite и native board protocol с synthetic
worker facts: positive complete pipeline, negative review fix, missing real test
proof, wrongSHA, current auth/replay, child/manual-child recursion and task scope,
no-expired-lease reclaim, policy local scope/approval, human question, sequential
workspace/cancel/regrant, clarify controlled continuation, archive stop plane,
terminal check proof, schema5 backup/preservation, malformed schema/index literals,
restore authority/history. Новые process/E2E тесты автор не создавал.

Финальный author race frozen core/store: PASS 68.589s; web integration: PASS29.958s.
В том же запуске projects был FAIL401-vs-403 в существующем membership regression;
root исправил actor wrapper вне185 и отдельно сообщил full projects race PASS23.13s.
Авторский targeted race recheck этого regression после root fix: PASS2.192s;
исходный FAIL artifact не переписан. Ранний общий race был PASS66.4/20.1/30.2s.
Actual native CLI pipeline / Windows / physical second PC / production TASK014:
NOT-RUN в WORK185. Worker static executor/IPC и transport/UI integration отдельные
WORK186/193/187/191 и отдельные review/QA. TASK013 GUI native preflight timeout
не исправлен этой серверной реализацией. Deployment/service/config не менялись.

Все новые Go файлы <=300 строк. Shared исторические файлы сохраняют имеющуюся
структуру. Exact source manifest (код+unit tests, без этой WORK):
`.local/task014/core-store185/review-p1/manifest.sha256`.
Source fingerprint SHA256(manifest): `8a499c3b003035432b70690e7a3b76362de23a2b7a29c68707135c9902f1fa6a` (65 source/test files).
Final author evidence: `.local/task014/core-store185/author-final-race.txt`;
projects targeted recheck: `.local/task014/core-store185/author-projects-recheck.txt`.

## Точный список source/test файлов185

- `internal/core/automation_capability.go`
- `internal/core/automation_evidence.go`
- `internal/core/automation_policy.go`
- `internal/core/automation_workflow.go`
- `internal/core/managed.go`
- `internal/core/types.go`
- `internal/store/auth.go`
- `internal/store/automation_aborted_test.go`
- `internal/store/automation_advance.go`
- `internal/store/automation_attestation.go`
- `internal/store/automation_authority.go`
- `internal/store/automation_capability.go`
- `internal/store/automation_clarify_test.go`
- `internal/store/automation_context.go`
- `internal/store/automation_control.go`
- `internal/store/automation_controls_test.go`
- `internal/store/automation_fix_test.go`
- `internal/store/automation_fixture_test.go`
- `internal/store/automation_gates.go`
- `internal/store/automation_list_wire_test.go`
- `internal/store/automation_list.go`
- `internal/store/automation_migration_test.go`
- `internal/store/automation_mutation.go`
- `internal/store/automation_pipeline_test.go`
- `internal/store/automation_plan_checks_test.go`
- `internal/store/automation_plan.go`
- `internal/store/automation_policy_wire.go`
- `internal/store/automation_policy_wire_test.go`
- `internal/store/automation_policy_revoke_test.go`
- `internal/store/automation_policy.go`
- `internal/store/automation_questions.go`
- `internal/store/automation_read.go`
- `internal/store/automation_resume_proof_test.go`
- `internal/store/automation_security_test.go`
- `internal/store/automation_steps.go`
- `internal/store/automation_storage.go`
- `internal/store/automation_submit.go`
- `internal/store/automation_task_scope.go`
- `internal/store/automation_terminal_test.go`
- `internal/store/automation_wire_test.go`
- `internal/store/automation_wire.go`
- `internal/store/automation_validation.go`
- `internal/store/backup.go`
- `internal/store/check_create.go`
- `internal/store/coordination_migration_test.go`
- `internal/store/helpers_test.go`
- `internal/store/maintenance.go`
- `internal/store/managed_automation_wire_test.go`
- `internal/store/managed_hosts.go`
- `internal/store/managed_migration_test.go`
- `internal/store/managed_poll.go`
- `internal/store/managed_registry.go`
- `internal/store/managed_report.go`
- `internal/store/managed_runs.go`
- `internal/store/managed_storage.go`
- `internal/store/migrations.go`
- `internal/store/migrations_c_test.go`
- `internal/store/mutation.go`
- `internal/store/persistence.go`
- `internal/store/profiles_test.go`
- `internal/store/schema_validation.go`
- `internal/store/sessions.go`
- `internal/store/store.go`
- `internal/store/tasks.go`
- `internal/store/transitions.go`

Review P1 ClientIntake закрыт узким согласованным delta: при ClientIntake=false
нужен typed AutomationIntakeApproval{PrincipalID,SessionID,ActionID}, выдаваемый
только browser adapter после CSRF/issued action. Describe(human) недостаточен;
native/REST не создают proof. Same-actor/session/nonempty-action проверяется и
в WorkflowSubmit, и AutomationPermission. Negative wrong actor/session/action
и positive typedUI tests: targeted race PASS1.623s. Все TestAutomation после
этого delta: race PASS15.143s, proof automation-after-intake-race.txt.
Ранее fullstore68.589s относится к base503b1e; полный final source послеdelta
не выдаётся за повторно пройденный fullsuite, добавочный охват указан явно.


## Повторное source review P1 — 23.10.2026

После f332aa3 сформирован 60-file checkpoint940a6; дальнейшие узкие изменения
PolicyGet/consistency перечислены ниже. Итоговый64-file source manifest указан выше. Pause/proven-stop/resume и question/answer/resume незавершённого
шага сохраняют тот же ManagedRun и точный native UUID. Новая generation/board
session/capability/task fence и бюджет обязательны; новый pinned profile/provider
не подменяется внутри прежней беседы — для такого изменения нужен explicit replan.
Fresh start допустим для terminal попытки без UUID. Пауза не обнуляет budgets/deadline.

Run-exit attestation creation/replay требует уже committed terminal ManagedReport
этой command/generation/worker: original completed/failed либо отдельная owned stop
command/stopped. Private TerminalPhase/TerminalAt сохраняют этот факт, applied/started
недостаточно. Running/accepted/needs_recovery не превращаются в source proof.

Workflow read/mutation pages ограничены 256КиБ фактического json.Marshal с HTML
escaping. Это оставляет место обеим копиям domain payload в MCP result и envelope
до неизменного 1MiB client limit. Immutable grant/step snapshot instructions/description
удалены только из wire views; ID/version/role/provider/model и native prompt сохранены.
List intake — корректный UTF8 preview до512 bytes с intake_truncated; detail и
mutation сохраняют весь intake. Messages, steps, plans, questions, attestations
имеют отдельные cursor positions; list paginates и policies, и workflows. Namespace,
filter и version/digest fences отвергают устаревший/чужой cursor. Новый элемент,
не помещающийся на страницу, остаётся следующим; oversized single record даёт
явный RESPONSE_CAPACITY. История и доказательства не удаляются ради лимита ответа.

ManagedList по умолчанию/HostID-only выдаёт Automation summary без authoritative
Scopes и legacy flat duplicates; scopes_truncated=true явно отмечает сокращение.
HostID+Workspace возвращает один полный выбранный scope и outer consent version.
Workspace без HostID/несуществующий scope отклоняются. Run.Automation.Checks убраны
только из list view; native ManagedPoll и storage остаются полными. Truncated consent
нельзя повторно выдать как локальный grant.

Новые author regressions — unit/actual SQLite integration, без процессов/model calls:
exact resume, separate stop proof, до/после terminal report и cached replay;
HTML-escaped histories/MCP envelope, все5 коллекций без skip/duplicate, stale/foreign
cursors, policy+workflow list pagination, UTF8 preview и immutable prompts;
32hosts ×32scopes ×32checks и128run bindings с bounded summaries и full native poll.
Targeted escaped/projection tests PASS2.258s. Ранний pre-wire full-store race
PASS66.544s лежит в review-p1/full-store-race.txt; он не final-proof wire изменения.
Финальный полный store race PASS93.220s: review-p1/final-store-race.txt; manifest drift check PASS60/60. Проверен pre-policy-delta 60-file checkpoint940a6; его exact manifest сохранён как review-p1/pre-policy-manifest.sha256. Последний narrow policy delta проверен targeted race ниже.
Process/E2E/native/Windows/release остаются NOT-RUN автором185; source reviewer
и отдельный QA должны подтвердить следующие gates. TASK014 этим не закрыт.


## Последний узкий PolicyGet bounds delta

AutomationPolicyInput получил optional Cursor/Limit, AutomationPolicyPage — optional
NextCursor. Current Policy всегда полный authoritative; Revisions — deterministic
страница по возрастанию version, default limit50/max100, фактический encoded budget
256КиБ включает current policy. Exact Version остаётся immutable filter. Cursor
привязан к project namespace, policy ID, current version, filter version и position;
смена policy version даёт CONFLICT, foreign/filter mismatch — VALIDATION. Oversized
single record даёт RESPONSE_CAPACITY, ревизия не пропускается. Stored history и
CRUD approval/authorization не менялись; старый вызов получает первую страницу.

Actual128revision escaped-payload + MCP double-encoding envelope/no-skip regression,
SQLite store exactVersion/continuation/stale/foreign cursor и oversized item:
**targeted race PASS3.939s**, proof review-p1/policy-get/targeted-race.txt.
Изолированный diff четырёх файлов: review-p1/policy-get/narrow-delta.patch.
Полный store race не повторялся по согласованному narrow scope: предыдущий
PASS93.220s относится к checkpoint940a6, новый delta подтверждён targeted race.
Checkpoint62files c47c05b8 сохранён в pre-consistency-manifest.sha256;
дальнейшие согласованные consistency изменения описаны ниже.


## Финальные consistency/terminal ACK уточнения

Доказанное несоответствие: policy/local consent допускали32checks, а plan требовал
RequiredCheckIDs<=16. При17+ mandatory checks никакой план не мог пройти. Увеличен
только этот предел до32; max16task entries и общий64КиБ encoded plan не изменены.
Actual SQLite owner consent update → policy update → scoped analyst plan regression
для17и32 requiredchecks: доfix VALIDATION (plan-red.txt), послеfix PASS; пропуск
любого обязательного check по-прежнему VALIDATION. Включена wire-size проверка.

WorkflowObservations.CheckAborted (check_aborted,omitempty) честно отмечает отмену
или deadline/localstop. QA gate отклоняет такой check для всех required presets,
включая lint/build, даже actual exit0 и валидный JUnit/TAP. Хранилище сохраняет
реальные exit/counts/hash/sourceclean без подделки отрицательного результата.
Regression positive control + abortedtest/lint/build gate и actual SQLite fullpipeline
abortedexit0→blocked/test_evidence_missing; итоговые observations неизменны.

WorkflowAttestResult.Attestation всегда содержит immutable original attempt tuple,
включая исходный control epoch. Result.ControlEpoch — current workflow metadata,
не grant: после pause/revoke terminal ACK может иметь более новый current epoch.
Nonterminal current fence не ослаблен. Actual SQLite pause→owned stop terminal→
run_exit ACK проверяет оба epoch и exactrun/gen, затем exactUUID resume с новыми
session/cap/task fences. Worker193 отдельно проверяет sender/HTTP/lock integration.

Targeted final consistency + PolicyGet + positive/negative pipeline race PASS9.978s:
review-p1/final-consistency/targeted-race.txt. TerminalACK/resume race PASS3.363s:
review-p1/final-consistency/terminal-ack-race.txt. Изолированные per-fix diffs лежат
рядом: plan-checks-delta.patch, check-aborted-delta.patch, terminal-ack-delta.patch.
64-file checkpointf5f236 сохранён в pre-revoke-manifest.sha256;
последний узкий storage-capacity delta указан ниже. Полный store race
93.220s остаётся доказательством checkpoint940a6, новые узкие изменения покрыты
перечисленными targeted races. Independent source review и processQA ещё обязательны.


## Review P2 — already-revoked history capacity

Новый revoke уже отозванной policy после currentVersion CAS и reason validation
возвращает current policy без изменения version/updatedAt/history/control epochs.
Current authority по-прежнему проверяется до receipt replay; прежняя idempotent
response доступна текущему авторизованному actor без повторного применения старого CAS.
Первый safety revoke active policy на update-cap128 допускается и создаёт final129.
Общий запрет revoke-at-cap не добавлялся: безопасный отзыв не блокируется лимитом.

Два source/test файла: automation_policy.go и automation_policy_revoke_test.go.
Actual SQLite regression:128revisions→revoke129→5новых request IDs остаются129,
stale freshVersion rejected, current authority loss rejects original receipt replay,
valid original replay сохранён. RED выявил исходный рост до130; первый targeted
прогон нашёл ошибку нового test expectation (FORBIDDEN вместо существующего
FORBIDDEN_ROLE), assertion исправлен, guard не ослаблен. Финальный targeted race
**PASS4.915s**, proof review-p1/revoke-cap/targeted-race.txt, narrow-delta.patch рядом.
Итоговый65-file manifest/fingerprint указан выше; drift check65/65 PASS, source frozen.
Полный store race не повторялся для этого узкого fix согласно scope координатора.
