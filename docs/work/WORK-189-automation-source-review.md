# WORK189 — независимое source/security review TASK014

Reviewer: managed_review. Root поручил готовые185 core/store,187 web и191 infra.
186 integration и193 worker ещё NOT READY и этим review не покрываются.
Область записи reviewer: эта карточка и доказательства foundation в WORK169;
implementation/ops/общая TASK не изменяются. Контракт APPROVE — WORK183.

## Текущий verdict

| Область | Статус | Основание |
| --- | --- | --- |
|191 capable infra/preflight/restore | REQUEST CHANGES | P1 ниже, final revised source gate обязателен |
|187 browser/Next | REQUEST CHANGES | Cached receipt permission DTO mapping неполон, см. ниже |
|185 core/store | PENDING FINAL SOURCE | Author добавляет TaskGeneration/authority hardening и schema P1 fix |
|186 registry/transport/bridge | NOT READY | Не заявляется reviewed |
|193 worker execution/IPC | NOT READY | Не заявляется reviewed |

Manifest19120files и18735files independently rehashed: drift0 на момент initial
review. Web aggregate88efabeb3415f06722e1fbff52a1394d57615ee5c239478cebf6b06f342134c0.
Будущие revisions должны дать новые manifests; прежние fingerprints не являются
approval изменённой реализации. Никаких process/E2E/native automation запусков.

## P1 — нормализация SQL меняет JSON literals и пропускает сломанный fence

Места: infra/accord/projects/automation_schema.py normalized/validate_automation;
internal/projects/automation_storage_preflight.go validateAutomationStorage;
internal/store/schema_validation.go automation index validation (185).

Текущий SQL normalizer lower/EqualFold и split/Fields применяет нормализацию
также внутри quoted JSON paths. SQLite JSON keys чувствительны к регистру и
пробелам. Поэтому повреждённый UNIQUE index с `$.WORKFLOW_ID` либо
`$.workflow_ id` принимается как ожидаемый `$.workflow_id`; аналогично RUN_ID/
run_ id для capabilities. Реальный json_extract даётNULL, UNIQUE не запрещает
дубли workflow/sequence/attempt и run/generation.

Reviewer воспроизвёл оба варианта в isolated SQLite :memory: без source edits:
validate_automation принимает изменённые индексы, после чего вставляются две
одинаковые execution tuples в обе таблицы. Существующие4 AutomationSchema unit
PASS, поэтому они пока не покрывают этот defect. Исправление требуется во всех
трёх validators: exact canonical DDL после TrimSpace либо quote-aware comparison,
сохраняющее literal bytes полностью. Ни lower/EqualFold, ни whitespace deletion
внутри literals недопустимы. Регрессии: wrongcase и literal-internal space в обоих
индексах, отказ до открытия migration writer/публикации snapshot. Initial reviewer
suggestion «case-preserving whitespace-only» была недостаточной; уточнение с
вторым reproduction немедленно отправлено root/185. Автор185 подтвердил exact
TrimSpace fix; independent revised-source verification ещё pending.

Остальная прочитанная191 часть сохраняет capability floor до DDL, historical2–5
snapshots/schema5 immutable migration bytes, additive comparison всех logical rows,
full all-project restore в новый offline root, boot guard и post-reset SQLite proof.
Restore сохраняет policy revisions/workflow evidence/reservations, отзывает policies/
caps/host proofs/credentials и увеличивает epochs; unknown process exit не выдуман.
Production273 pending-https-smoke нельзя обойти поставкой schema6.

##187 initial source observations

Прочитаны browser adapters/forms/projections/actions/recovery tests и production
React forms/control/setup/progress/policy helpers. Применён vercel-react-best-practices
к cache/project boundaries, mutation freezing и derived state; новые зависимости
или произвольные performance refactors не предлагаются.

Положительные границы: same-origin/CSRF и issued action binding до trusted policy
approval, pure permission check cached receipts, original exact payload/session
recovery, strict form keys/CAS, project-qualified routes и read-only projection.
UI выбирает exact local host/workspace consent и role/profile revisions, показывает
queued отдельно от process proof; explicit stale-version adoption и unknown-result
retry сохранены. Four role templates создаются отдельными profile actions без
обещания подключённых native workers. Source fixtures используют реальный SQLite,
но typed registry context в них не является проверкой production registry guards.

Открыто: полный final test/integration review после freeze185/186/193; read DTO
projections/bounded pagination и mixed-provider wire проверяются совместно с core.
Product UX observation: principals вводятся raw ID через запятую; для человеческого
потока желательно имя/label selector из допустимых project/host participants,
оставив IDs только техническим представлением. Это не обнаруженный auth bypass.
Ни source35file review, ни TypeScript/build сами не дают automation runtime PASS.


## P1 — browser cached receipt теряет operation-specific authority fields

Место: internal/web/automation_permission.go checkAutomationPermission. После
изменения core.AutomationPermissionInput adapter не передаёт Workspace, CommandKind
(JSON payload kind), RefreshPolicyID/Version и QuestionID. Два refresh поля
извлекаются только для approval context, но не передаются domain permission check.

Независимо сопоставлены web mapping и current core AutomationPermission:

- cached answer приходит с пустым QuestionID: q=nil, поэтому exact receipt
  назначенного человека отказывается с403 после реально сохранённого ответа;
- cached resume/replan без CommandKind пропускает authorizeWorkflow, поэтому
  current grant/expiry checks для этих операций не выполняются;
- refresh receipt без RefreshPolicyID/Version не проверяет актуальную policy
  revision/settings; policy receipt без Workspace пропускает scope expiry.

Это не утверждение, что cached action вновь запускает процесс: проверка должна
оставаться readonly. Требуется exact forwarding всех сохранённых payload fields и
regressions lost-answer-reply, revoke-before-resume-replay, refresh version change и
workspace consent expiry. Root нашёл этот mapping gap параллельно и сообщил
reviewer; independent code read подтвердил причину/воздействие. Manifest187 должен
обновиться после исправления. Root186/worker193 integration пока не source-ready.

## Передача initial findings

191 и187 сейчас REQUEST CHANGES; source approval всех областей не выдаётся.
Первичные production reads и targeted4infra units выполнены, два isolated SQLite
index reproductions подтверждены. Полный source/tests review185/187/191 с
исправлениями и общая integration186/193 остаются обязательными до новых process/
E2E automation tests. Reviewer освобождает слот авторам исправлений; никаких
native/production/GUI операций в этом review не выполнялось.

## Повторное review готовых185/186/187/191,23.10.2026

191 SQL P1 **исправлен / scoped SOURCE APPROVE**: Python SQL.strip и Go
TrimSpace сравнивают canonical SQL буквально; case/internal spaces внутри JSON
literal больше не нормализуются. Независимо:54 infra unit PASS; четыре isolated
SQLite variants (wrongcase/internalspace в обоих UNIQUE paths) отвергаются.
Targeted projects migration/preflight/reset race PASS5.338s. Manifest191 SHA
0e256caed1c175c2cf02c7e50d2472d627777e4a0c1fd26b0f67f613f85e94d7.
Это не общий source approval ещё меняющейся integration/worker193.

187 cached permission mapping P1 **исправлен**: Workspace/CommandKind/
RefreshPolicyID+Version/QuestionID forwarded из exact stored payload. Прочитаны
пять новых lost-response/current-authority regressions и typed browser intake
factory. Независимый targeted web race PASS5.489s. ClientIntake=false теперь
требует server-only actor/session/action-bound UI proof; произвольная human role
его не заменяет. Полный coupling185/186/193 остаётся отдельным gate.

Прочитан186 registry→legacy→board guard flow, execution credential path/operation
allowlists, forced Describe/session binding, native and browser separation,
read-only archived stop/report plane с current writer authority, scoped MCP
tools/resources и parent IPC injection. Serve scheduler bounded/fair, lifecycle
join предшествует registry close; unexpected scheduler failure stops admission.
Source/tests read продолжается, общий186 APPROVE пока не выдан.

## Новые185 P1 — human resume и post-exit evidence

1. workflowStartOrContinue для interrupted unfinished step использовал новый
   workflowEnqueue вместо exact native resume: новый run/start теряет UUID после
   pause→proven stop→resume либо question→answer. Требуется тот же current step,
   run и providerUUID, новая generation/cap/session/task fencing и учёт обоих
   invocation/continuation budgets; fresh start допустим при отсутствии UUID и
   доказанном terminal/undispatched состоянии. Автор сообщил fix+regressions,
   independent final revised-source check pending.

2. run_exit attestation ранее принималась при ещё running managed attempt:
   attestation checks tuple/UUID/host proof, но не положительный terminal report.
   Ранний cleanHEAD/manifest мог ошибочно стать post-exit proof после последующего
   exit. Требуется durable private positive terminal marker именно accepted
   ManagedReport transaction/current generation; running/unknown/recovery не
   считаются выходом. Автор сообщил TerminalPhase/At fix+negative regression,
   independent final revised-source check pending.

## P1 — count pagination не ограничивает фактический native response

automation_read.go WorkflowList default32 возвращает полные immutable grant
profiles: только instructions32×4×16384 уже2097152 bytes до JSON overhead, при
internal/client MaxResponseBytes1048576 и MCP max event1MiB. WorkflowGet default50
возвращает full step snapshots и планы, поэтому также пересекает cap; JSON HTML
escaping увеличивает допустимые '<' до6 wire bytes. Это допустимые input sizes,
а не malicious secret data. Client intake/status и child workflow resource
становятся нечитаемыми при законной накопленной истории.

Нужны bounded encoded-size projections/pages с truthful cursor и без пропуска
evidence; list не требует полного текста profile instructions. Проверить также
mutation workflowPage responses и policy revision reads. Immutable persisted
snapshots не урезать, global client limit не повышать как замену bounds.
Регрессии должны использовать maximum legal/escaped payload и последовательно
вычитать все pages без lost/duplicate records. Root/185 уведомлены; пока OPEN.

Итог этого review pass: **REQUEST CHANGES для composite TASK014 source**.
Новые automation process/E2E tests ещё не допущены. Real GUI195/production273
проверены отдельно в WORK169; они не являются TASK014 runtime evidence.

## Повторный composite pass: pins и проверенные исправления

Проверялись18560files940a6ae40873f287b0a2ed1679e6571c8cb985c70eb5a347e8249360f3a0182c;
18743files653c91b7b45ae8ec72278806f07b47dcd869bbdd2fe1855fa354819b5d71ce4e;
18629files manifest SHA77de3acaae478c9e0b3f2edaf0fbe27ff841d9ea73dbcf52c1f7a405514ac67a;
19393files78eac6ef1f92618c6f1f7bacf14d00ecaf22a719d143b077b332eb54ccb8dc28.
186/187 manifests independently drift0.185/193 позднее reopened для перечисленных
ниже fixes; эти fingerprints не покрывают будущие изменения.

185 human resume/run_exit fixes **scoped APPROVE**: exact run/UUID сохраняются,
attempt/generation/cap/session/task fence обновляются, continuation budget charged.
TerminalPhase/At в private command выставляются только в положительном managed
report transaction; early start/applied и needs_recovery не являются exit.
Прочитаны human/question regressions, exact stop vs original start и replay guards.
Независимый targeted store race PASS27.852s также покрывает bounded history/list,
immutable wire projections, UTF8 preview и managed selected scopes/native poll.

Workflow/history/list256KiB encoded budgets учитывают HTML escaping и duplicate
MCP structured/text result; независимые cursors каждой коллекции без пропусков,
namespace/version/filter fences и immutable stored snapshots сохранены. Один
непомещающийся элемент даёт explicit RESPONSE_CAPACITY. PolicyGet revisions
отдельно reopened автором по root request, финальная проверка ещё pending.

187 selected host/workspace read **scoped SOURCE APPROVE**: summary явно truncated,
полный ровно один scope получается readonly с current project/host authority.
UI не выводит права из flat provider cross product; late success/error fenced,
abort/unmount/project change не принимают старое consent. Workspace/version adoption
явная, stale role/check bindings сбрасываются. Targeted web race PASS1.999s;
Node automation/host-scope12tests PASS. Новые core DTO CheckAborted/policy revision
pagination ещё требуют согласованной UI projection, это не approval будущих delta.

186 source guards/transport/bridge/scheduler lifecycle и REST/MCP tests прочитаны;
new source-specific auth bypass не обнаружен.29files drift0; final integration
approval зависит от исправлений185/193 и итоговых checks, не выдан авансом.

##193 P1 — неопределённый Git exit теряет ownership boundary

automation_git.go observeGit объединял `runGit(ls-files)` error и index mismatch,
возвращая generic error вместо errOwnership. Preparation мог объявить failure
доказанным и снять checkout lock при неизвестном Git process exit. Кроме того,
executeAutomationCheck post-check Git error лишь делал SourceClean=false; unlike
pre-check/observeAutomationExit, errOwnership не quarantine/stop/retain lock.
Оба пути требуют сохранения cause, needs_recovery и отсутствия автоматического
reuse. Автор подтвердил, fixes/tests pending независимой проверки.

##193 P1 — timeout существует только в локальном status

executeAutomationCheck после deadline/ctx cancellation выставлял только local
ErrorCode=check_deadline_or_stop. exit0 + valid JUnit/TAP + clean source оставались
положительными authenticated observations; workflowTestsGate не видел local code
и мог квалифицировать такой check как PASS. Нужен server-visible negative fact,
сохраняющий реальные exit/count/hash без подмены. Авторы185/193 согласовали
CheckAborted defaultfalse и backend gate; required regression — interrupted check
с нулевым exit и корректным report не может подтверждать passed workflow.

##193 P1 — deferred accepted ACK переживает StopAll

Ack accepted создаёт goroutine, но до получения ею mu StopAll может увеличить
stopEpoch и увидеть empty active set. Поздний apply затем захватывал уже новый
epoch и запускал процесс после глобальной остановки. StopAll Wait мог истечь,
а Close — закрыть journal при живом child. Нужны captured pre-schedule epoch,
отмена queued pre-stop plans и closing/global-stop admission barrier; deterministic
barrier regression, где native spawn после StopAll остаётся0. Автор подтвердил.

##185/193 P1 — terminal ACK после human control отвергается самим worker

Terminal WorkflowAttest законно принимает exact old attempt ControlEpoch после
pause/cancel/question. Однако backend возвращает out.ControlEpoch=current workflow
epoch, а worker sendAttestation требует equality с old input epoch во всех фазах.
После фактического stop server сохраняет run_exit/check_finished, но worker
отвергает ACK, не снимает checkout lock и ломает следующий resume. Fake transport
units echo input epoch и не ловят несовместимость. Требуется единый контракт:
attestation binding exact old tuple, current epoch metadata не выдаёт полномочия
новому запуску; terminal acceptance отдельна от права start. Actual store/HTTP
integration regression pause→terminal report→exit ACK→lock release обязателен.
Root/оба автора уведомлены; independent final fix review pending.

**Composite verdict остаётся REQUEST CHANGES.** Process/E2E TASK014 gate закрыт.
Reviewer не менял source/permissions/production и не запускал новые native модели.

## Final narrow185/186/187 и existing acceptance197 review

197 **SOURCE APPROVE**, только два existing process acceptance files:
backup SHA1620f54f429ff65e37e332bc38e6c24540c8ea01ee81037bd6e24ac64fa6da1c;
migration SHA2f5c145c7c468e419853156c78246c9ae776b7273eb4fc026db8d107637e68ac.
Фактические hashes independently matched. Schema6 assertions отделены от
checksum/history; actual snapshot/migrated SQLite дополнительно проверяются на
9 automation tables/key,data columns, exact metadata(1,1) и два canonical UNIQUE
indexes с неизменными JSON literals. Existing auth/reset/history/fencing checks
сохранены. Допускается targeted исполнение этих existing cases по root GO;
письменный будущий native/pipeline plan197 не означает source/runtime approval.

185 final64 pinf5f236fff43110e3bd8e9721b3f9f89fb60f2090571d3bc4daba3c8a9631ff45
drift0: **scoped SOURCE APPROVE** проверенных core/store corrections. PolicyGet
current policy + bounded revision pages/cursor namespace/currentVersion/filter;
128 revisions читаются без потери. Plan IDs limit32 согласован с32 policy checks;
actual17/32 mandatory variants включают negative omission. CheckAborted сохраняет
настоящие exit0/count/hash, но исключает positive gate для test/lint/build.
Immutable attestation old epoch/current workflow epoch metadata разделены явно.
Independent targeted race PASS7.182s. Fullstore93.220s author относится к прежнему
60file checkpoint; не выдаётся за полный run64file. P2: repeated revoke уже
revoked policy может растить revisions без update128 cap; root/185 уведомлены,
рекомендован dedup/conflict с сохранением первого safety revoke после cap.

186 final31 manifest SHA55d366d105fa9e263c548fb814ba7804d4660e4b74610afd64f3b0a1976531b4,
187 final45 aggregatec018c82567437fccf8a402937fe4e89e6638a97ecb11a1c5484b32f302f70400:
оба independently drift0 и **scoped SOURCE APPROVE**. Новый root in-process
HTTP fixture использует actual SQLite WorkflowAttest + real remote decoder/ACK,
реальный checkout lock/reacquisition и exact core resume; typed registry checkpoint
в fixture не выдаётся за production registry test или native execution.
187 CheckAborted presentation ставит interrupted/not-counted выше status/exit/count,
не превращает pending+exit0 в PASS; independent three pure frontend tests PASS.

193 четырёх P1 final source/tests review ещё pending; общий composite gate
**REQUEST CHANGES**, новых automation native/process/E2E не допускает.

##193 final97 — четыре P1 закрыты

**SOURCE APPROVE193** pin78c3ac53629e191653aa773b9f370856c20923c8470ecdab7491efb2ea96a468:
independent hash recalculation97/97 drift0. Git ls-files error сохраняет ownership
cause; post-check unknown quarantines/retains lock без finished proof. Aborted
exit0/report остаются реальными facts, но authenticated CheckAborted не даёт PASS.
Accept фиксирует stopEpoch; deferred ACK не переживает global stop, new admission
блокируется во время StopAll/Close. Deterministic barriers покрывают manual и
automation. Terminal ACK проверяет exact original attestation tuple; newer current
epoch допустим только для terminal closure, не reserve/start/new authority.
Прочитаны full worker/IPC consent, capabilities, Git/manifest, checks/reports,
stop/watchdogs, durable ACK/restart/compaction paths и новые regressions.
Independent targeted race PASS2.722s, включая actual SQLite→HTTP remote→ACK→
checkout release/reacquisition→exact core resume. Author full worker/IPC/cmd race
12.633/1.366/2.966s и cross-build не выдаются за native Windows/process acceptance.
Composite final verdict ожидает только последнего185 revoke P2 delta/pin;
native permission profiles не менялись, actual write/commit/fullpipeline NOT-RUN.

## Итоговый composite SOURCE APPROVE

185 revoke P2 закрыт: fresh VersionCAS/reason сохраняются, current authority
проверяется перед cached replay. Повторный fresh revoke уже revoked policy — no-op
без новой revision/version/updatedAt/control epoch. Первый safety revoke после
update cap128 разрешён в revision129. Actual SQLite regression проверяет5новых
request IDs/no history growth, stale CAS, denied replay и прежний exact receipt;
independent targeted race PASS2.538s. Final18565files pin
8a499c3b003035432b70690e7a3b76362de23a2b7a29c68707135c9902f1fa6a drift0.

**SOURCE APPROVE composite185/186/187/191/193** на exact перечисленных pins:
18565/8a499c3b;18631/55d366d1;18745/c018c825;19120/0e256cae;
19397/78c3ac53. Все5 manifests повторно independently drift0. Согласованный
197 two-file existing acceptance delta также SOURCE APPROVE. Все перечисленные
blocking findings189 закрыты source fixes и meaningful unit/integration coverage.

Gate допускает отдельному QA авторить новые bounded process/E2E harnesses;
каждый такой source требует собственного review до execution/root GO. Final
full checks на frozen candidate и actual native write/commit/full pipeline,
negative fix loop, UI, capable Linux all-project migration/restore ещё обязательны.
Ни этот source approval, ни SDK/fake process/cross-build не являются их PASS.
Windows/Claude/физический второй ПК, production schema6 rollout — NOT-RUN.
Native permission grants/settings не расширялись reviewer и не выводятся из gate.
Review-card chronology намеренно сохранена целиком; лимит300 строк для этой
истории превысился, production source files этим исключением не оправдываются.
