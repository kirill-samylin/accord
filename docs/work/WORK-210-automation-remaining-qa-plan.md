# WORK210 — планы remaining native fix/control/UI QA

23.10.2026. Отдельный QA plan author; root координатор, reviewer202 независим.
**PLAN ONLY / SOURCE REVIEW PENDING / EXECUTION NOT-RUN.** Новые executable
sources/fixtures не написаны; server/browser/worker/native/model/RPC/help/compile
не запускались. Production, Давинчи GUI, реальные задания и configs не менялись.
203 sourcea19… остаётся frozen с REQUEST CHANGES202; эти планы его не открывают.

## 1. Общая граница и gates

Порядок допуска каждого case: согласованный PLAN210 → applicable product SOURCE
APPROVE и закрытые native prerequisites → отдельный tester пишет case sources →
независимый202 SOURCE APPROVE всех sources/fixtures/pins → exact root GO одного
fresh execution. Candidate208 ещё pending; никакие3e6/8c pins или permissions
не переносятся молча. Normal206, actual developer commit и OS-negative enforcement
должны быть реальными PASS на exact execution candidate. Diagnostic207 этого
не заменяет. До нового source-review нельзя включать case env-переключателем.

Каждый case — новая private0700 fixture, отдельные DB/registry/web/state/workspace,
локальные token0600, обычный Git `.git`, empty hooks/remotes, frozen source/check
roots. Real HOME/native auth сохраняются, user config raw before/after private;
никакого restore/исправления configs ради PASS. Bounded реальный CLI supervisor,
nonce/lifeline/owned exit, actual managed terminal ACK и local lock release
обязательны. Offline/queued/accepted не означают stopped/applied/completed.

UI — реальные локальные browser contexts и CSRF/action IDs; собственный server
и Next из exact pinned copies, готовность/свободные порты до запросов. Это не
physical second PC и не browser test реального рабочего Accord/Давинчи. Источник
требования — UI; дополнительные transport reads/idempotent retries не являются
новыми native turns. Ни parent, ни mock не пишет успешные native outcomes,
attestations, done/review/QA или фиктивные JUnit counts.

## 2. Exact budgets — отдельные executions

| Case | Expected native/model turns | Policy/local maxima | Step/workflow/grant seconds | Setup / cleanup |
| --- | --- | --- | --- | --- |
| F6 primary review-negative fix | A1,D1,R1,D2,R2,T2 =6 | tasks1, invocations6, fixes_per_task1,total1, continuation0 |180/1200/1800|180/60|
| Q7 optional QA-negative fix | A1,D1,R1,T1,D2,R2,T2 =7 | tasks1, invocations7, fixes_per_task1,total1, continuation0 |180/1500/1800|180/60|
| C2 pause/clarify/exact resume | same analyst native conversation,2 turns | tasks1, invocations2, fixes0, continuation1 |240/600/900|180/60|
| H1 question/answer/cancel | analyst1, answer/cancel0 | tasks1, invocations1, fixes0, continuation0 |180/360/600|180/60|
| B2 second-principal browser controls | same analyst conversation,2 turns | tasks1, invocations2, fixes0, continuation1 |240/600/900|180/60|

Local maxima не шире указанных; consent expiry максимум45min для F6/Q7 и30min
для остальных. Setup180s не содержит model turns. Workflow deadline охватывает
все её попытки, fixed check max30s, cleanup reserve60s; отдельный outer bound
setup+workflow+cleanup. Счётчики started/invocations должны совпасть с фактом,
не только быть ниже потолка. Fifth/seventh/eighth лишний turn, automatic restart,
скрытый continuation/summary/fix — FAIL. Для C2/B2 stable native UUID один,
но model invocations две; 0-count human/API actions показаны отдельно.

## 3. F6 — основной actual fix pipeline, negative review

Fresh small Python fixture с двумя meaningful tests и заранее reviewed edge
failure (negative/zero input). Fault contract и первый native developer prompt
заранее source-reviewed: controlled initial faulty commit A допустим **только
как явный synthetic negative result**, не production success. Developer summary
обязан честно описать фактический diff/тесты, не заявить PASS при failing tests.
Reviewer получает полное требование/код/tests и независимо оценивает его.
Если native отказывается завершить synthetic faulty attempt либо исправляет всё
сразу, expected negative route НЕ доказан: NOT-EXERCISED/FAIL coverage по заранее
approved классификации, без parent transition, скрытой подмены prompt/patch/retry.

Обязательная фактическая цепочка:

1. Analyst materializes ровно одну developer task с обязательным fixed Python check.
2. D1 делает actual commit A поверх seed, только разрешённый source diff; clean HEAD.
3. R1 — новый native UUID/run/board session, exact A/generation1. Настоящий
   `changes_requested` с конкретным defect evidence; это не harness-patched JSON.
4. Сервер сохраняет R1 done/negative evidence и reopened developer generation2
   с reason/evidence_ref. D2 — новая попытка; делает реальный новый commit B,
   parent=A, HEAD B !=A. Никакой QA/source patch после frozen review target.
5. R2 — свежий independent run/session, exact B/generation2, actual approve.
6. T2 — свежий independent run/session, exact approved B/generation2, actual fixed
   test execution:2cases, exit0, failure/error/skip0, genuine JUnit hash. Затем
   actual passed и native/check terminal proofs; workflow completed/epic done.

Proof:6 invocations,1fix,1reopen,2developer commits, immutable R1 negative history,
latest R2/T2 target B, actual reviewed/tested/source manifest B. R1 дляA не
становится approval дляB. Parent только читает; source/HEAD final hashes совпадают
с qualifying attestation. Здесь initial tester T1 **не запускался**: результат
называется review-negative fix + renewed independent review/QA, не QA-negative.

## 4. Q7 — необязательная genuine QA-negative ветвь

Root отдельно подтвердил: честность independent review важнее достигаемости
сценария. Нельзя заставлять R1 одобрить известный дефект, подменять уже reviewed A,
подделывать failing JUnit или писать parent outcome ради маршрута из7turns.

Только если реальная цепочка A1→D1(commitA)→R1(actual approveA) дошла до T1 и
реальная fixed verification обнаружила failure, ожидается T1 actual failed с
genuine nonzero/test failure evidence; далее D2(commitB parentA)→R2(approveB)→
T2(real passedB). Ровно7 invocations, fixes1, full same-SHA/generation bindings,
обе raw JUnit/check histories сохранены; первые failure counts не стираются.

Если R1 сам ловит defect или initial T1 проходит, Q7 **NOT-EXERCISED**, не PASS
этой ветви. Никакого runtime навязывания approve, late source mutation, повторного
seed или дополнительного восьмого model turn. После safe own cancel/cleanup
это отдельный сохраняемый outcome. F6/native coverage и existing typed backend
QA-negative tests отчёт различает; typed тесты не являются native Q7 proof.

## 5. C2 — pause → proven stop → clarify → exact UUID resume

Fresh workflow, один analyst step, две разрешённые native invocations. Profile
просит на первом turn взять именно свою lease и не материализовать/закрывать
задачу до уточнения человека. Это ожидание надо source-review как bounded prompt;
если native закончит раньше/уйдёт в question, нужный live-pause route не доказан,
не создаётся принудительный duplicate run для исправления timing.

1. Readiness: actual native started+UUID и bound describe/next с live task lease.
   UI pause с current workflow version. Receipt pausing/stop queued сам по себе
   не PASS. Сразу после этого resume до positive stop должен быть rejected,
   без generation/invocation/command increase.
2. Дождаться actual owned CLI/PGID exit, terminal stop command applied, run_exit
   exact stopped-command tuple/старый immutable epoch, worker durable managed ACK,
   native_exit/exit_ack/lock release. Лишь после этого paused, старый claim закрыт.
3. UI clarify добавляет один unique canary ordered message, не меняя acceptance/
   план/target SHA. Уточнение не считается применённым до второго native turn.
4. UI resume current version. Тот же managed run ID и provider native UUID,
   generation2/attempt2, новый control epoch, fresh board session и новая lease
   generation >generation1. CLI resume argument должен содержать **тот же**
   реально сохранённый UUID; одного текста server prompt недостаточно.
5. Второй native действительно делает fresh describe/next и отправляет свой
   workflow_question с canary уточнения (без plan/done). Это observed delivery,
   не предположение по очереди. Затем только human cancel, без третьей модели.

Fencing proof: terminal original tuple не переписывается новым epoch; новая
execution capability имеет новый run generation/control epoch. Одно read-only
GET со **собственным ранее реально выданным** generation1 child credential может
проверить отказ после pause/resume. Его защищённое capture/retention/path и exact
expected error берутся из source review перед исполнением: credential не
минтится/не подменяется owner bearer, не печатается, parent не завершает task.
Без валидного original credential нельзя объявить этот negative RPC PASS;
metadata/new-session/generation observations и NOT-RUN RPC выделяются отдельно.

Финал C2 — safely cancelled control fixture, не completed business pipeline;
2native invocations/1UUID/2board sessions/continuation1 и question/canary evidence.
Никакой replan или приема другой роли в прежней conversation.

## 6. H1 — genuine question, authenticated answer без resume, cancel

Одна native analyst invocation реально вызывает workflow_question для своей
lease; durable question с reason/options и отдельной human task. Проверяются
awaiting_human, epoch advance, old execution revoked, actual stop/terminal ACK.
Открытый question не разрешается TTL и не теряется после UI refresh.

Назначенный fixture human principal через собственный UI/current CSRF отвечает
`Resume=false`: настоящая authenticated human claim+completion, answer evidence,
human-task author/attempt/session, question answered/version+1/answer_ref. Перед
ответом UI resume при open question должен дать supported human_required отказ
без нового model turn. Чужой неназначенный fixture principal (если добавлен для
этого auth subcase) получает отказ и не меняет question/human task.

Exact retry того же answer action должен сохранить один receipt/одну completion;
изменённый answer с тем же id конфликтует. Answer без resume оставляет paused
после proven stop, counters.invocations=1. Затем назначенный human cancel current
version; terminal cancelled, reserved0, no active runs/leases/pending proofs.
Question+answer/history сохраняются. Не заявлять, что отвечал native agent:
это автоматизированное действие тестового human/browser actor.

## 7. B2 — второй fixture principal, remote UI control, один физический Mac

Свежие A(owner host) и B(editor), разные catalog credentials/project-local IDs,
разные browser contexts/cookies/session/actions. Owner bearer/cookie никогда не
используется для действия B. B не получает worker secret/execution credentials.

До запуска моделей: host enrolled только A, локальные controllers=[A], B имеет
лишь project editor membership. UI B не предлагает разрешённый OS start/control;
один supported собственный managed_start/control attempt B должен получить
forbidden на реальной authority path, без created run/spawn/counters. Offline
host/readiness rejection не подменяет доказательство forbidden; expected check
order сверяется по owning source при написании case.

Затем **сам владелец локально**, при остановленном/unlocked собственном worker,
меняет только reviewed fixture controller allowlist [A,B] и вызывает supported
worker consent. Ни браузер B, ни роль editor не добавляют grant. Current server
host consent/epoch/controllers подтверждаются readback. Worker запускается с
тем же явно разрешённым workspace. A через UI создаёт policy с intake/control
[A,B], immutable versions/current host consent; лишних principals/providers нет.

Теперь A submit запускает analyst1; B из собственного UI выполняет C2 pause,
ожидает proven stop, clarify canary и exact UUID resume analyst2. A независимо
видит тот же workflow/commands/evidence. Второй native подтверждает canary своим
question; B как designated current controller делает safe cancel. Итого ровно
2model turns, не два ПК. Current permissions/actions читаются заново; cached
кнопка/200 receipt не заменяют applied/terminal outcome.

После всех native proven exits worker остановлен. A локально удаляет B из
consent и читает новую version/allowlist; cached B control action отвергается
current-authority fence и не создаёт command/run. Не приписывать rejection
конкретному ACL vs stale-action слою без фактического error/provenance. Этот
subcase показывает grant и revoke в собственной fixture, не изменения доступа
реальных участников. Scope Windows/Claude/second physical machine NOT-RUN.

## 8. Report, cleanup и следующие source reviews

Каждый case always-save private initial/final result, stage/failure, before/after
pins и native config raw snapshots, browser action receipts/screenshots, actual
CLI/run/step/session/epoch/SHA/check/JUnit tuples; full pages/cursors и changed
source maps, без секретов в stdout/Git. Safe public report — counts/status/hash
и точные NOT-RUN/NOT-EXERCISED границы. Assertions и typed/unit proofs отмечаются
отдельно от actual native/browser/OS observations. Expected negative outcomes
не являются error masking: должны быть получены заданным actor/путём/tuple.

Новый harness обязан исправить203 reviewer P1 до повторного использования:
корректный credential lifecycle, bounded raw-byte frame+total/private-log caps,
реальный cleanup fence/deadline и join/abort без late actions после final report.
Неконтролируемый helper/unknown constructor/exit/ACK сохраняет fixture/reservation/
needs_recovery/manual-inspection, не освобождает её ради PASS. No generic PID kill,
no SQL mutations, no permission weakening, no auto retry/fix нового harness.

НаFAIL — только own workflow current cancel CAS, затем bounded own worker/browser/
MCP/Next/Go lifeline cleanup и положительные proofs. НаPASS case — native/lease/
reservation terminal, свои grants/policy/profiles закрыты, процессы proven exited,
только затем supported offline revoke фактически существующих own credentials.
Не выдумывать catalog credential для legacy-only setup; реальные IDs фиксируются.
Данные/артефакты/private history не удаляются; родитель решает дальнейший execution.

**PLAN READY → 202/root review.** Никаких новых executable sources или runtime
операций по210 нет. Перед каждой реализацией согласовать frozen fixture/prompt,
exact candidate и applicable source approvals; после source review нужен новый
root GO. Review-negative F6 — primary; Q7 optional по прямому решению root.
