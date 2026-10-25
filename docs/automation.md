# Автоматическое выполнение требования — TASK014

Проектируемый контракт; не описание уже работающего функционала. Основание —
[TASK014](TASK-014-automation.md). Полный контур: требование → native аналитик →
задачи → native разработчик → независимое review → реальные проверки тестировщика
→ ограниченные исправления → отчёт либо явный вопрос/остановка. Оба входа, из
Accord и Codex/Claude, используют одни сущности и gate evaluator.
## 1. Фактическая база и новые полномочия
В current core уже существуют Task/Lease/Attempt/Evidence/Review, profile revisions,
TaskAssignment, ManagedHost/Run/Command. Profile не identity, assignment не lease,
CLI completed не task done. `completion` проверяет формат SHA и evidence, но не Git;
`CheckCreate` атомарно фиксирует developer SHA/generation и prerequisite review для QA.
`successful/latestVerification/epicClosable` остаются единственными core gates.
`ManagedStart/Command` сейчас требуют отдельную human session; bridge фиксирует
managed run/generation/session/profile. Автоматического scheduler сейчас нет.

Ограничение TASK011 о ручном создании pipeline расширено новым поручением.
Нельзя реализовать scheduler поддельной human/coordinator session, произвольным
`Describe(role=human)`, сохранённым bearer человека или обходом managedMutate.
Новый внутренний delegated domain path авторизует только операции конкретного
workflow. Обычные native core и ручные managed APIs сохраняют прежние правила.
Identity registry, project-local principal, server session, workflow/step, managed
run, native conversation и физический ПК различны. Legacy issuer проверяется через
свою действующую credential authority, не по owner label. IDs другого проекта не
переиспользуются. Третий editor получает site access, а не OS/automation control.
## 2. Явная policy и authority checkpoint
Human owner/editor с текущим control каждого выбранного host создаёт versioned
AutomationPolicy через browser/CLI admin: project, issuer principal+access_epoch,
expires_at (default24h/max7d, bounded local consent), intake/control principals,
один host/workspace; для каждой роли analyst/
developer/reviewer/tester — binding provider+profile revision, local check preset IDs,
Git policy local_commits, budgets. По умолчанию client intake выключен. Включение
явно разрешает указанным principals подавать задания в этих пределах без нового
подтверждения каждой стадии. Без policy вход создаёт только draft/awaiting_authorization.
Policy ID — ссылка, не credential. Mixed Codex/Claude роли допустимы в том же checkout при local enrollment обоих CLI.
Модель не передаёт host/path/argv/env/profile override. Policy approval требует
trusted browser action/CSRF context либо локального CLI consent с host proof;
обычный native bearer и Describe(role=human) этого контекста не создают.

Workflow получает неизменяемый grant snapshot: policy ID/version, policy issuer
и requester local IDs/epochs, host owner ID/epoch, host consent version, aliases/profiles,
лимиты. Raw bearer/provider login/host proof туда не записываются. Grant исполняется,
только пока policy active/current, issuer и requester credential+membership write/live epochs,
host owner credential+membership/epoch, project active и обе стороны host control
действительны. Requester также обязан быть current host controller в server/local
allowlists; policy intake allowlist не расширяет OS control. Policy update/archive/
revoke приостанавливает старые workflows;
regrant/новая membership epoch их автоматически не оживляют.

Перед каждым advance и enqueue проверяются registry → legacy authority → board
под тем же порядком write guards, что `projects/service_managed.go` и
`GuardManagedCredentials`; guards удерживаются до commit board transaction. Scheduler
получает unexported typed authority из этого checkpoint, а не wire Actor/SessionID.
Общие create/check/reopen/managed enqueue helpers выделяются из текущих методов;
они сохраняют все проверки, но administrative delegation не подделывает native
completion/session. Requester и command.CreatedBy остаются настоящим initiating
requester principal, CreatorEpoch — его epoch; audit отдельно фиксирует
workflow/step/grant revision, без выдуманной human session в sessions table.

Dispatch, accepted и started повторно проверяют grant/issuer/requester/owner epochs.
Новые workflow bindings managed run/command (workflow_id,step_id,grant_version)
сервер формирует сам. При revoke/pause command становится неисполняемой. Для уже
выданной работы используется существующий owner-authenticated ManagedPoll failsafe:
он формирует только stop под реальным owner management actor. Отозванный issuer
не используется для новой команды. Это не даёт scheduler права запустить процесс.
Stopped/completed/recovery proof принимаются для своей fenced попытки после revoke;
отсутствие положительного exit proof никогда не освобождает защиту.
## 3. Локальное согласие и workspace
На host нужен отдельный default-off automation opt-in для project/workspace:
разрешённые роли/provider, local runtime presets, maximum attempt duration и
фиксированные verification presets. Enrollment/consent с host proof публикует
только безопасный snapshot aliases/capabilities/version. Worker повторно пересекает
его с текущим local config и controller allowlist перед каждой попыткой.
Server policy не расширяет локальные paths/executables/models/permissions.
Старый worker без automation capability не получает workflow execution dispatch.

Default — один workflow последовательно в одном разрешённом checkout, включая
readers. Durable server reservation host/workspace запрещает interleaving другого
workflow/manual run; ручное вмешательство сначала pause+proven stop. Physical
filesystem identity/global local lock TASK013 дополнительно защищает alias/re-enrollment;
серверный alias не доказывает совпадение физических каталогов на разных hosts.
Workspace должен быть чистым в начале: иначе awaiting_human, без auto stash/reset/
clean/commit чужих изменений. Нужен отдельный заранее разрешённый checkout, если
обычная рабочая копия занята. Git push/merge/deploy не входят в local_commits.

Role presets задаёт владелец: reviewer без source edits; tester пишет только
согласованные build/artifact/test paths. Это local native presets+последующая
проверка source hash, не новая выдуманная FS sandbox: текущий workspace-write
сам по себе не запрещает reviewer запись. Строгий read-only выбирается локально,
а необходимые tester build writes явно разрешаются. После стадии фиксируется Git HEAD,
clean/changed-source state и проверяемый SHA до освобождения local lock. Для
review/test target SHA обязан совпасть с фактическим checkout; worker не переключает
ветку и не скачивает произвольный репозиторий по команде модели. Без доступного
точного SHA — blocked, не проверка другого дерева. Unix гарантия остаётся managed
CLI+owned PGID, Windows Job stronger; detached writers не выдаются за contained.
## 4. Durable данные и scheduler
Предлагаемая board schema6 добавляет automation_policies/revisions, workflows,
workflow_steps, workflow_plans, workflow_questions, workflow_attestations,
workflow_reservations, execution_capabilities и automation_metadata(version1). Workflow хранит ID/version,
intake channel/request/hash, bounded requirement, grant snapshot, state/reason,
plan revision/epic, current step, epoch, counters/deadlines и итоговые evidence refs.
Step хранит stable ID/type/sequence, task+expected version/generation, profile
snapshot, target SHA, attempt number, managed run ID/generation, status и proofs.
Это новые поля, их нет в текущих core DTO. История попыток/планов append-only.

`Advance(workflow)` — bounded deterministic board transaction: проверить authority,
CAS workflow version/epoch, core gates, budgets/reservation → записать step + task/
check/command + audit + durable response атомарно. Никакого AI/network ожидания
в SQL transaction. Уникальный execution key (workflow,step,attempt) защищён SQL
UNIQUE; один task/check/run не создаётся дважды. Two server processes/restart
конкурируют тем же write lock/CAS, отдельный leader без authoritative fencing не нужен.
Wake после report/plan/question/answer и periodic bounded scan раз в2s; повтор
вычисляет сохранённое состояние, не повторяет внешний spawn. Native spawn остаётся
только за TASK013 journal/accepted ACK/worker epoch/lifeline/owned exit proof.

Intake idempotency key: project+requester principal+его epoch+request_id; canonical
payload включает policy version/channel-independent requirement fields. Current
access проверяется до replay. Точный retry из другого транспорта возвращает тот же
workflow; изменённый payload конфликтует. Session expiry допускает fresh authorized
session для чтения прежнего receipt, но не новую authority. Новый request_id —
новая заявка, similar-work warning не считается гарантией дедупликации смысла.
## 5. Действительный pipeline

| Стадия | Наблюдаемый результат и следующий переход |
| --- | --- |
| intake | Валидный grant+reservation; создаются epic и pinned analyst task/run |
| analysis | Native analyst делает describe→next, предлагает structured plan и завершает свою task с artifact+summary; нужны plan validation и proven exit |
| materialize | Одной транзакцией создаются developer tasks и DAG plan refs; роли/profiles берутся только из grant, duplicate strict не обходится force |
| develop | Native developer получает lease, реализует и делает локальный commit; core done evidence + worker Git attestation + proven exit нужны вместе |
| review | Новый managed run/conversation/board session; общий CheckCreate фиксирует точный developer SHA/generation; findings+approve/changes_requested/blocked |
| test | Только latest current approve; новый CheckCreate tester с blocks/approved; native tester выполняет реальные разрешённые checks и сдаёт evidence |
| fix | Negative review/test сохраняется; общий reopen developer с reason+evidence_ref отрицательной done проверки; новый generation/SHA, заново review и QA |
| finish | Все plan tasks имеют актуальные positive gates и qualifying worker proofs, нет questions/leases/runs; epicClosable, затем done + итоговый отчёт |

Plan: maximum16 task entries, default8; local IDs, title/description/areas,
acceptance, prerequisites DAG, required local check IDs: ≥1 required kind:test
на каждую developer task и весь policy mandatory set, иначе reject. Не содержит executable,
команд, credentials, host, provider или agent instructions overrides. Native analyst
подаёт его только для своей analysis lease через bound workflow_plan. Core analysis
evidence ссылается на immutable plan resource/hash. Неполный/цикличный план не
материализуется, CLI exit0 без plan/task evidence даёт blocked/no_outcome.

Стадии используют разные managed runs, реальные provider conversations и board
sessions. Resume допустим лишь внутри того же role/step и с точным сохранённым
provider ID; reuse developer conversation для review/test запрещён. Весь author
attempt/contributor history исключается из reviewer sessions; core session/SHA/
generation independent gates остаются обязательны. Это независимость процессов/
сессий, не доказательство разных людей или независимого мышления модели.

Task.done сам по себе проверяет существующий evidence format, а не Git/команды.
Поэтому workflow step сначала evidence_pending: worker после proven exit, пока
checkout lock удерживается, записывает authenticated attestation фактического HEAD,
target SHA, source-clean state и attempt binding. SHA модели, text output, event
или CLI exit0 не заменяют её. Mismatch/недоступный Git/непроверяемое дерево — blocked;
историческая task evidence сохраняется и не переписывается scheduler.

Реальные проверки требуют отдельного default-off isolation consent для preset:
enforced launcher либо явно trusted_owner_process (WORK181); native workspace-write
не разрешает unsandboxed repo-code checks. Fixed local config задаёт executable,
argv/cwd/timeout. Bound workflow_run_check принимает
только preset ID; private per-attempt IPC к worker сверяет run/gen/session/lease.
Worker запускает свой supervised preset, сохраняет exit/timing/target SHA и
bounded normalized artifact hashes, без remote shell/добавочных args. Native tester
читает результат и пишет conclusion. `passed` qualifying только если все required
check IDs реально выполнены на этом SHA с exit0 и current review; not_run/N/A не
автоматический успех. Наличие preset проверяется до workflow start; missing требует
владельца, а не обхода permissions. Full provider transcript остаётся локально.

IPC конкретно: Unix domain socket0600 в private attempt directory0700; Windows
Named Pipe с random128bit name, current-user/SYSTEM DACL и REJECT_REMOTE_CLIENTS
через имеющийся x/sys. Listener создаётся worker до native child, не наследуется.
В immutable private bridge config добавляются endpoint+random256bit nonce; nonce
не идёт в argv/env/server/logs. Frame: uint32 big-endian length≤64KiB + strict JSON
{v,nonce,request_id,workflow_id,step_id,control_epoch,run_id,generation,
session_id,task_key,task_generation,preset_id}.
Каждый frame сверяется с этим endpoint/attempt и текущим grant/lease checkpoint;
более широкого local RPC нет. Check request сначала резервируется на сервере
через owner proof/epoch, локальный intent fsync предшествует spawn фиксированного
preset. Key(run,generation,request_id)+payload hash: повтор возвращает тот же
check ID/status/result, неизвестный запуск не повторяется. Tool отвечает bounded
queued/running/check_id; результат читается resource, status не подменяет exit.
Result/exit proof сначала fsync в journal под checkout lock, затем idempotent report;
crash/lost ACK остаётся pending/recovery. Детали Git/artifact/check proof — WORK181.
Parent shutdown/pause/revoke/deadline отменяют native и все check handles; terminal
run proof требует exit всех owned checks. IPC закрывается при attempt end, nonce
не оживляет новый worker. Network/IO/check spawn вне global worker mutex; legacy
25s/30s fencing и independent watchdog действуют. Windows transport требует своей
реальной проверки; доступность WinAPI не считается native PASS.

Если нужны новые process/E2E tests: tester после source approve авторит ограниченный
verification artifact/test patch; это test_design, не passed. После proven exit
test_design оставляет source checkout чистым, отдавая hash-bound patch artifact;
отдельный developer применяет/фиксирует его с сохранением tester provenance, новый
reviewer проверяет новый commit (включая tests), затем tester исполняет их. Старые
SHA approvals не переносятся. Цикл учитывается в budgets; неизвестные методы/
права/локальные presets выносятся человеку. Создание новых tests не имитируется
запуском echo и не совмещается с их независимым approve.
## 6. Управление, вопросы и пределы
States: draft/awaiting_authorization, queued, running, pausing, paused,
awaiting_human, stopping, cancelled, completed, blocked, needs_recovery.
Stage и run delivery status показываются отдельно; queued/accepted/offline не applied.
Pause блокирует advance и запрашивает owned stop, paused только после proof.
Cancel при needs_recovery запоминается (`cancel_requested`), resume/новый dispatch
отвергаются, пока ownership неизвестна; после terminal/recovered_stopped всех runs —
cancelled+release. Без cancel needs_recovery → paused/`native_recovered`. Каждый
reason_code сопровождает bounded redacted `reason_message` (у native_* — LastError).
Invocations считаются по started report; human resume не тратит continuation budget.
Поздний `accepted`/`started` команды, dispatched до pause/cancel/вопроса/срока,
от live owner этого dispatch принимается (native session и invocation учитываются),
но run остаётся `stopping`, owned stop — pending, шаг не возобновляется; при
потере authority инициатора — по-прежнему `403`. Run в `needs_recovery` не
получает повторную доставку своих команд (их закрывает `worker recover` →
`recovered_stopped`) и не делает весь poll `409` для других runs этого host:
отказ остаётся только для dispatched работы с неизвестной судьбой. Live-owner
`recovered_stopped` не продлевает lease (остаётся действующей для run_exit).
Шаг без dispatch, истёкший при открытом workflow deadline, на resume заменяется новым.
Queued scan при занятом workspace не меняет version без новых фактов.
Cancel также снимает лишь незахваченные workflow задачи после остановки; историю/
check evidence не удаляет. Resume требует current grant/epoch/profiles, разрешённый
checkpoint и отсутствие ambiguity; новый control epoch отвергает late results. Profile edit/archive блокирует новые
claims/steps до явного rebind; живой snapshot не переписывается.
Сообщения имеют `message_meta{source,task_key?,defect_key?}`: human_clarification/
answer/replan идут всем следующим turns, reviewer/tester_findings и test_design — только
developer step этой задачи как findings; legacy без меток — по серверным префиксам.
Clarify сохраняет ordered сообщения: текущему run — next-turn TASK013 semantics,
изменяющее план/acceptance уточнение — pause, analyst replan новой revision и явное
согласование уже сделанного. Цель/SHA начатого review не переписывается.

workflow_question доступен любой своей native стадии: durable вопрос с reason,
вариантами/контекстом и отдельная human task, затем awaiting_human+stop. Ответ
только назначенного current human principal через существующие claim/answer lease
правила либо общий equivalent domain helper; агент не отвечает за человека.
Ответ и explicit resume входят в один проверяемый control action. Вопрос без
ответа не теряется по TTL и не разрешается по истечению времени.

Visible defaults: max8 plan tasks,2 fix cycles per task и2 total fix cycles,
40 native invocations включая continuation,1 continuation per step, step30min,
workflow8h и grant24h. Hard maxima:16 tasks,5 cycles/task и10 total; invocations≤min(128,
фактически доступные зарезервированные slots),
step2h/workflow24h/grant7d. Локальный maximum может быть строже. Runtime deadline
также исполняет worker monotonic watchdog, не только server scheduler. Превышение
→ pausing→awaiting_human/budget_exhausted, без автоматического повышения лимита.
Token/cost usage показывается только если native CLI реально сообщает её;
неизвестная стоимость не объявляется нулевой или доказанным денежным hard budget.

Квоты (TASK022): admission считает только live — ≤128 active runs+reserved run budget
всех workflows, 8 active, ≤32 non-terminal workflows; 256 commands/run (+stop), 16
queued, 256 steps/workflow. Lifetime-счёт навсегда блокировал проект. История не
удаляется, но ограничена в live state, т.к. каждая transaction грузит (tick клонирует)
весь board, а run держит до 64KiB output+16KiB prompt: когда новый workflow/run
требует slot и terminal history ≥512 runs или ≥128 workflows, в той же transaction
старейшие idle terminal записи (без active run/queued/accepted command/live session/
lease; automation runs — только вместе со своим terminal workflow, steps/plans/
questions/attestations/capabilities) переносятся в archive rows тех же таблиц
(key `archive/…`, без DDL/schema bump) до 448/96. Это автоматическая, но явная и
аудитируемая ротация (event `automation_history_archived` с числами), не silent GC:
`AutomationArchive{kind:workflows|managed_runs,cursor,limit≤32}` пагинирует её —
read-only для owner/admin проекта: REST POST `/api/v1/p/{id}/automation/archive`,
MCP resource `accord://automation-archive{?session_id,kind,cursor,limit}` (см.
[api.md](api.md)). Перенос пишет финальное значение записи; capabilities архивного
workflow в archive row уже `revoked`, а restore (`RecoverState`) отзывает и archive
rows прежних версий. Нет архивируемого — `capacity` с причиной. Archived run не resume/clarify; задачи
archived workflow — обычная terminal история. Старый binary archive не понимает.
## 7. Общие интерфейсы и границы исполнителей
Все новые DTO ниже proposed; точные поля/safety helpers —
[WORK181](work/WORK-181-automation-contract.md#единые-dto-и-safety-context).
Project scope всегда path/authority, не поле клиента.
Mutation: request_id+expected version; read: bounded cursor/limit. Body≤64KiB,
requirement≤16KiB, question/clarify≤4KiB; output/redaction floors TASK013 сохраняются.

| DTO / метод | Доступ и смысл | Владелец реализации |
| --- | --- | --- |
| PolicyCreate/Update/Revoke | human admin UI/CLI, current host control; нет MCP policy CRUD | root transport/UI, core/store author |
| WorkflowSubmit{policy_id,policy_version,text} | UI или unbound native session с policy intake grant | core/store author, root transport/bridge/UI |
| WorkflowList/Get/Report | project read ACL, metadata list/details/reports с пагинацией | core/store author, root reads/UI |
| WorkflowCommand{workflow_id,version,kind,text?} | current authorized control; pause/cancel/resume/clarify | core/store author, root controls |
| WorkflowPlan{version,plan} | только own bound analyst lease; binding выводится сервером | core/store author, root bridge |
| WorkflowQuestion{question,options?} / Answer | own bound stage / designated human; durable inbox | core/store author, root bridge/UI |
| WorkflowAttest{run_id,generation,task_generation,proof} | только owner worker+host proof/epoch, typed observed facts | worker owner + core/store author |
| WorkflowAdvance internal | registry/legacy checkpoint + atomic domain helpers/outbox | core/store author, root registry scheduler wiring |
| workflow_run_check{preset_id} | локальный bound tester/test-design bridge, fixed IPC | worker owner, root bridge integration |

Files: core/automation.go и store/automation_* + shared in-board guards/migrations
— core/store author; projects/service_automation, transport/automation, bridge,
web/Next и scheduler wiring — root; worker IPC/executor — отдельно назначенный owner.
REST/CLI namespace `/api/v1/p/{id}/automation/...`; методы отражают таблицу.
MCP explicitly добавляет workflow_submit, workflow_command, workflow_plan,
workflow_question и local workflow_run_check; read resources
`accord://workflows`, `/workflows/{id}`, `/workflow-inbox`, `/workflow-reports`.
Исторический12-tool каталог больше не обещается неизменным: names/version schemas
и bridge accept/quarantine logic обновляются явно; local native consent enums не
расширяют grants автоматически. Unset/empty grants по-прежнему ничего не разрешают.
Unbound Codex/Claude intake bridge может submit/control лишь policy-scoped workflow.
Automated child получает новый per-attempt scoped ExecutionCapability: server хранит
только digest, owner bearer/host proof остаются parent. Middleware фиксирует project/
workflow/step/run/gen/role/profile/session даже при omitted binding; next только
own task, нет unbound Describe/admin/submit (подробный протокол WORK181).
Managed child bridge не рекламирует submit/policy/admin/host control и отвергает их
сервером даже при guessed names; у него только own-step plan/question/check/read.
Нельзя рекурсивно породить workflow из child prompt или смоделировать новый grant.

Первый запуск: «Подготовить команду» создаёт4 editable role profiles через existing
CRUD с явными partial/unknown receipts; до всех valid bindings policy не approve.
UI: требование+policy/пределы до запуска, progress по стадиям и task/run delivery,
вопросы/ответы, история решений/исправлений, причины blocked/recovery/capacity,
пауза/отмена/продолжение/уточнение и итоговый отчёт с actual SHA/check results.
Same-origin/CSRF/action capability, project query/draft isolation и unknown-outcome
recovery сохраняются. Ручные managed controls остаются и не обходят reservation.
## 8. Миграция и приёмка

Schema6 + permanent capability `automation:1` вместе с registry/ACL/profiles/
coordination/deduplication/managed_agents floors; registry1/web2 пока не меняются.
До DDL durable schema5 snapshot, unchanged legacy rows, strict incomplete/future
schema refusal. Старый schema5 executable запрещён до переключения; обновление
только capable all-project backup/upgrade/restore/roll-forward. Host dispatch
требует новую advertised worker automation capability, не guessed binary name.

Приёмка: unit/integration → independent source/security APPROVE → отдельные QA
process/browser/native tests и review самих новых tests → Linux capable restore/
upgrade → отдельно production. Обязательны оба intake/replay, full real pipeline,
negative fix loop+new SHA, stale/forged outcomes, pause/revoke/regrant race, crashes/
lost responses/epoch takeover refusal, budgets/history capacity и human question.
176 остаётся diagnostic-only; config-only actual probe177 и настоящий native
start/stop/resume prerequisite не отменяются этим контрактом. Mac, Windows Claude,
Codex/Claude direct intake и разные физические ПК проверяются отдельно.

Полный sequential contour обязателен; deferred только parallel worktrees/merge,
autonomous push/deploy, OAuth/webhook triggers, provider switching и accurate cost
ceilings без native metering. Это не сокращает analyst→developer→review→test→fix.
