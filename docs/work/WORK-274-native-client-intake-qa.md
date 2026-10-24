# WORK274 — native Codex/Claude intake → команда → UI control

Владелец отдельного process/E2E QA: gui_connection_qa. Root — исполнитель actual;
client_intake_author — клиентский source; independent reviewer273 — applicable
source/QA review. **PLAN READY, executable adapter NOT AUTHORED, actual NOT-RUN.**
Владение: эта карточка и `.local/task015/qa/`. Test/worker/policy/server/GUI/global
settings автор не меняет. Предыдущие261/269/270 доказательства не переименовываются
в nativeclient intake:269 проверял каталог,261 подавался из настоящего UI.

## Два коротких последовательных случая

1. Настоящий Codex через model MCP describe(coordinator) → workflow_submit одного
   exact requirement с действующей policy/version → получает workflowID → завершает
   клиент. Команда добавляет subtract в src/math_ops.py и tests/test_subtract.py:
   три обычных unittest positive/negative/zero; всего5 вместе с исходными2.
2. Только после полного terminal/ACK/reservation0/clean первого — настоящий Claude
   аналогично подаёт multiply + tests/test_multiply.py, ещё3 ordinary cases; всего8.
   Клиентская Claude auth/readiness требует отдельного фактического подтверждения.

Оба intake клиента только координируют, не пишут код и не запускают тесты. Четыре
роли исполняет существующий **Codex worker**; Claude intake не является Clauderoles
proof. Exact requirement text root ещё замораживает, новых требований QA не подаёт.
Один локальный developer commit на случай; никакого push/merge/deploy.

## Что доказывает настоящий intake

Нужны actual native CLI executable/version/UUID, обычный provider turn, настоящие
model toolcall describe/request_id/run_id → unbound coordinator board session,
затем actual workflow_submit request_id/policy/version/exact text и успешный MCP
response. WorkflowID/requester/grant/intakehash совпадают с fresh scoped server
readback; workflow_submitted event/session дополняет привязку, если доступен.
Ни SDK/stdiodirect replay, ни REST submit от helper не заменяют этот путь.

В текущих core.Workflow и owning WorkflowSubmit нет literal intake_channel поля.
Root подтвердил: поле N/A; native origin доказывается реальной цепочкой выше.
Нельзя выдумать workflow.channel или объявить ordinary GET клиентским intake.
workflow_get — read resource, не новый MCP tool; bridge подставляет session.
Один bridge — одна fresh session, managed scoped child не вправе submit.

Current269 launcher только interactive codex|claude, Codex ready/Claude blocked.
Новые print/exec/capture/normalexit flags не угадываются: exact argv и capturemode
берутся из финального independently approved273. Допустим обычный root interactive
PTY и явное **разовое** разрешение точного call. No allowall/always/autoapprove,
settings/auth bypass или ослабление read-only. MCP refusal при never-policy — FAIL,
сохранённый отказ, а не повод переписать permissions.

## Ограничения и root-run порядок

Каждый случай — один client process, intake<=180s + owned cleanup<=30s; stdout+
stderr<=16MiB/frame<=8MiB, bounded captures/private0600/own0700. Нет autoretry.
Клиент возвращает workflowID до положительного natural exit/drains; закрытие
клиента не требует остановки рабочего daemon. Неизвестная closure — FAIL/manual.
Provider token/cost считаются только при actual emitted данных, иначе unavailable.

Перед каждым root проверяет current host online/consent/policy/intake grant,
собственный requester и approved source/CLI path+SHA. Существующие policy limits:
300s/этап,1800s/цикл,12native,2fix; не продлеваются и не увеличиваются QA. Случаи
строго последовательны в одном checkout, poll2s; разрешённые дополнительные
continuations/fixes сохраняются целиком и входят в обычный budget.

На первом workflow root в настоящем UI может выполнить early analyst pause:
сначала observed running nativeUUID/boardSession/live tasklease → «Пауза» →
отдельный applied stop + durable exit/5ACK/lease release → «Передать уточнение» →
«Продолжить» с точным старым UUID, новой managedgeneration/boardSession/taskfence.
Human operation timebox60s, исходный300s deadline остаётся. Уточнение не меняет
функциональный scope: сохранить add/исходные tests/runner, только localcommit.
Если native завершился/вопрос/timeout опередил контроль, coverage NOT-EXERCISED
или FAIL с фактической причиной, не fakepause/no newretry. Replan требует отдельного
root решения; нельзя назвать новым UUID same-session resume. Forced/unknown stop
не именуется graceful PASS; все raw факты сохраняются. Второй случай — наблюдение
полного UI lifecycle без обязательной второй паузы.

## Terminal acceptance каждого workflow

- Полные страницы без cursors/truncation; текущие A/D/R/T tasks и epic done.
- Actual developer localcommit/fullSHA и taskgeneration; reviewer approve/tester
  passed относятся к этому SHA/generation, независимые nativeUUID/boardauthors.
- Реальные developer+tester project-tests/version1/digest;5 затем8 testcase nodes,
  failures/errors/skips0; typed exit0, literalfalse truncation; reportSHA actual
  persisted attempt XML, source_clean/manifest/target/run/generation bindings.
- Все native попытки и controls в истории: actual UUID/run/gen/epoch/profile;
  native_exited/exit_durable/managed_ack/exit_ack/lock_released=true,
  scopedpending0/reservations0/closedleases/questions. Receipt phase не serverACK
  сама по себе, numeric nativeexitcode в нём не выдумывается.
- Clean Git, parent baseline и только разрешённые implementation/newtest изменения.
  Исходная add function source и5 protected файлов неизменны. Перед Claude новая
  accepted Codexbaseline дополнительно фиксирует subtract source/test bytes.
  Runner discover(test_*.py) прочитан readonly и поддерживает5/8 без изменения.

## Практический reuse без нового большого harness

269 launcher/installed approval — process-only transport contract;273 даёт свежие
native flags/provider activation. Root обычным клиентом выполняет запуск, QA затем
независимо анализирует сохранённые finite packets. 261 — whitelist tuple/ACK/JUnit
pattern, **не импортировать** старый collector с hardcoded workflowID/plan;235 —
исторические stop proof категории;270 — принятый genuine catalog/navigation/closure.
Новое executable glue/collector, если потребуется, отдельно source-reviewed до
imports/run. Private EXECUTION-RECORD.md — декларативный data adapter/root checklist,
не executable скрипт и не готовое разрешение выполнения.

Root и final independent evidence reviewer различны. PASS требует обеих настоящих
client цепочек и terminal proof; отказ auth/permission/coverage остаётся FAIL или
NOT-RUN/NOT-EXERCISED, не workaround. Windows/второй физический ПК/current desktop
registration этим Mac процессом не подтверждаются. Токен/его хэш не читать,
не сохранять auth/host_secret/IPC; нет broad FS/process scans.

До runtime: approved273 candidate+QAplan, exact two requirement/policy/request IDs,
Claude normal auth/activation, fresh baseline и отдельный root GO на каждый случай.
Private plan-v1 source manifest передан root/reviewer; фактических CLI/model/
network/browser/control/Testwrites этой работой нет.

## Codex V1 startup FAIL и V2 compact startup adapter

Root V1 actual exit1 до модели: TERM=dumb refusal и heredoc/EOF; прежний лог и GO
сохранены. Fresh root API preflight показал только прежний completed workflow,
нового submit нет; original request_id сохраняется для точной идемпотентности.
Это не permission/model/Accord business failure и не завершённый native intake.

По ROOT SOURCE GO подготовлен единственный95-line stdlib adapter
`.local/task015/qa/launch-codex-v2.py` SHA
`03e2c380dd6ab3554124329bb0c742dcb65e087156f50b67155abb52ef12c01e`;
manifest `codex-v2-source-pins.json` SHA
`69ec79e22dce943c0e3a60cbc4be65e8c039c01affb5fe7c9a8c15f1f14e6d56`.
AST parse PASS, module/main/CLI/model НЕ исполнялись.

Adapter проверяет exact273source/capture approvals, installed3bytes600, approved
argv52897616/prompt71715/installreceipt, реальный Codex/worker executable SHA и
script binary. Не открывает token/config auth/global settings. Требует реальный
inherited tty fd0/fd1 (запуск Python FILE, не heredoc), fresh root V2 GO с adapterSHA,
originalrequestID/noapprovaloverride/nomodeloverride/180+30+16MiB и возрастом<=180s;
one-use claim/fresh600 V2log. exec заменяет Python на /usr/bin/script -q с exact
capturedargv + approved --no-alt-screen/positionalprompt. Только child environment
TERM=xterm-256color; исходные fd0/1/2 наследуются без pipe/EOF/autoapproval.

Root запускает `python3 -I -B /ABS/.local/task015/qa/launch-codex-v2.py` в собственном
PTY и сам соблюдает утверждённые180s/30s/output caps, сохраняет actualexit/drains,
normal menu input/percall approvals. Source adapter не фоновый runner/watchdog,
не обещает internal timeout; обычный /exit или CtrlD только после реального
workflow MCP receipt, не доказательство toolcall сам по себе. V1 GO не переписывается.
Новая independent273 SOURCE APPROVE и отдельный root V2 GO обязательны.

## Actual Codex intake — частичное evidence, цикл ещё не принят

Root V2 выполнил настоящий native Codex с обычными разовыми Allow для Describe
и Submit; клиент вернул workflowID и root наблюдал normal CtrlD shutdown/exit0.
Это не stdioSDK simulation. Независимо прочитан только точный собственный native
session JSONL по переданному UUID (без broadscan): session_meta exactUUID,
actual model tools.mcp__accord_test__describe/coordinator и workflow_submit с
original request_id/policyv1, successful toolresponses/boardSession/workflowID/
intakeSHA совпали; native task_complete присутствует. Raw /status accountline
не выводилась/не копировалась. Persisted root OSexit receipt запрошен отдельно:
пока actualexit0 — явно root-observed, не вывод из modeltask_complete.

Private finite partial receipt `.local/task015/qa/codex-intake-native.partial.private.json`
SHA `59cf5cc20d44ebff45d72ea977d007193acc1f55c80ffd72bf294c5e8cba8f01`:
NATIVE_INTAKE_PROVEN_WORKFLOW_PENDING. Это не fullworkflowPASS и не ClaudePASS.

Paused root API packet подтверждает workflowv8/epoch2/paused и original analyst
managedgen1/stopped; отдельный stop applied, start demoted delivery_unknown/public
last_error сохранены. Требуются отдельные положительные local5ACK/pending0/gen1
proof и актуальный gen2sameUUID/newboardSession/taskfence/terminal packet; пока
sameUUIDresume не принят. Root сообщил UI pause→clarify→resume до deadline, но
пока нет полного сохранённого результата. Historical category235 применяется
только при всех exact tuple/currentauthority/stop guards, не ignore last_error.

У root запрошены конечные full workflow/task/run pages(no truncation), точные
review/QA SHA+generation, genuine JUnit5/0/0/0 developer+tester, Git/protected files,
все nativeattempt tuples/5ACK/reservations/leases/questions. QA не выполнял новых
process/helper/model/API/control/Testwrites; только whitelisted saved evidence.
Claude auth/activation и весь его случай остаются NOT-RUN.

## Codex случай — independent actual QA PASS с указанной границей

Полный frozen root packet codex-final-1792831207436901000 проверен по всем pinned
member SHA; native intake partial/exit receipt и точный собственный native analyst
JSONL прочитаны отдельно. Private review `.local/task015/qa/codex-final-review/review.private.json`
SHA `710ded65734dc07b3173975205ddc0a3bbf7b5f6c344d9ec59222c7e8241443a`;
25-file index SHA `19c970aacdb69ad970cb23fb6a75524311b544c76941e9dadc50b4e9133d4155`.
**PASS_CODEX_CASE_WITH_DECLARED_GEN1_CAPTURE_LIMITATION**; TASK015 overall PARTIAL,
Claude native/model NOT-RUN/auth pending. Отдельная final evidence review273 нужна.

Actual native intake Describe/Submit, original request/policyv1/returnedworkflowID,
server intakehash/requester и normal individual approvals подтверждены. Root exit
receipt содержит actualscript/native0 после CtrlD; account/status raw остаётся
private. Workflow completedv32/epoch3, четыре текущие роли done,4roleUUID/5workflow
native invocations (analystcontinuation), reservations0/questions0/leasesabsent.
Все4 current localruns пятьACKфлагов true/pending0; полные publiccommand страницы.

Native analyst ownJSONL содержит ОДИН native UUID и ДВА реальных task_started turn:
managed Describe gen1 иgen2 с разными board session; managed/taskgeneration1→2,
continuations1. Точный UI clarification текст из persisted workflow.messages
найден во втором native userturn. Поэтому actual same-dialog continuation
подтверждён conversation history; OS literalresumeargv observer в TASK015 NOT-RUN.

До resume полный gen1local5ACK snapshot **не архивирован**. Не приписываем ему
несуществующее наблюдение. Есть paused API/stopped run/stop applied, retained
started/stopped receipt/gen1 run_exit attestation, затем guarded sameUUIDgen2 с
успешным окончанием/пятьACK текущейгенерации. Это corroborated closure, отдельно
от direct pre-resume all5flags. Исторический startdelivery_unknown/publiclast_error
не исправлены/не называются applied. Никакого повторного native ради счётчика.

Actual commit `2a2a530c50c51290eb7653bcfadb979956f772ff`, parent fc5cc899…;
root Git saved clean, diff только src/math_ops.py+tests/test_subtract.py, add и
пять protectedbaseline файлов сохранены. Reviewer approve и tester passed на
T-11generation1/exactSHA, независимые author sessions. Оба настоящих persisted
JUnit прочитаны:5 testcase/0failure/0error/0skip; XML SHA совпали с localcheck и
public attestation. Typedexit0/literalfalse truncation/source_clean/manifest/
project-tests v1 digest/target/run/gen/epoch3 gates сверены.

Global Codex config во время native TUI НЕ был unchanged: два обычных bookkeeping
ключа сохранены отдельным root delta. Принятый pure restore выполнен root только
после own native idle, actual receipt включён в index; это не скрытый обход approval
или новая native проверка. ПервыйTERM=dumbFAIL и ранний expiredobserver409 capture
не переписаны. Автор QA не выполнял API/provider/process/Git/test/control/import;
только сохранённые exact packets/ownnativeJSONL, finite projections и hashes.

## Новый явно выбранный target: запущенный Claude Desktop

Пользователь выбрал повторную проверку через уже вошедший **Claude Desktop**.
Это новый target второго случая, не фиктивное завершение Claude Code CLI auth:
CLI остаётся NOT-RUN. Codex actual273 PASS и controls не повторяются. До этого
append точный прежний WORK274 сохранён private immutable snapshot в
`.local/task015/claude-desktop-qa/WORK274-before-desktop-plan.md`.

PLAN READY only, actual NOT-RUN: private source manifest SHA
`538676bc8c56cb409ba44e8bc9682a81048813f0651a4a54a2db1730e31f2a7f` (3members),
plan SHA `7724c296263395252199e841cd67d6b5cf2a4152612d95222230da00e9886cba`.
Никакого runner/нового executable/helper/CLI/model/UI управления автором не создано
или запущено. Root делает настоящие CUA действия в собственном новом testchat
только после additive MCP source275/independent273 и fresh rootGO/preflight.

Неиспользованный prompt17d17598/request_id сохраняются exact. Реальная модель
Desktop должна вызвать Describe/coordinator и workflow_submit с exact args,
получить boardSession/workflowID/state/link и закончить turn. Снимки только своих
toolcards/ответов + fresh scoped API sameID/requester/grant/intakehash и текущая
Accord карточка доказывают origin; регистрация MCP/список tools не заменяют вызов.
Native conversationID берётся только если штатно виден в ownchat; иначе честное
NOT_EXPOSED, не случайный UUID/boardSession и не чтение caches/чужих чатов.
Root сохраняет только own test content, исключая sidebar/account/credentials.
Для Desktop не требуется закрытие самого приложения; CLIexit N/A этому target.

Далее существующая Codexкоманда: multiply+3ordinaryunitcases/всего8, один localcommit
parent2a2a530…, exactSHA/generation reviewapprove+QApased, developer/tester настоящие
JUnit8/0/0/0 и strict source/digest/terminal5ACK/pending0/reservations0/leasesclosed.
Защищены add/subtract/ранние tests/runner/AGENTS/README/.gitignore; scopes/policy/
consent/worker/server/300s/1800s/12native не меняются. Дополнительный controltrial
не нужен: используются принятые Codex pause/clarify/sameUUID proof, а новый workflow
наблюдается в том же Accord. No push/merge/deploy/новых разрешений/autoapprove.

Новая Desktop actual приёмка и итог TASK015 — отдельное решение root+273 после
всех фактических доказательств. Claude role execution, Windows/физический второйПК
и Claude Code CLI этим Desktop случаем не подтверждаются.

## Desktop MCP apply — независимый readonly readback PASS

После approved275/root helper actualexit0 выполнено только чтение exact known
baseline/backup/candidate/текущего Desktop config. Сравнения произведены в памяти:
baseline равен saved beforebackup; текущие bytes равны approved candidate и exact
additive transform. Единственное добавление — одна mcpServers.accord_test entry
с точными approved command/args. Все прежние JSON fields равны, исходные bytes
сохранены вставкой перед final brace; UID/GID/0600 сохранены. Duplicate JSON keys,
необычный file/owner/mode/drift отвергались; фактический readback их не обнаружил.

Private finite receipt `.local/task015/claude-desktop-qa/apply-readback.private.json`,
SHA `d249203a64a6f07e24072528dbe829966aa5af1dc0609dfabd91c64ab9aec5b8`:
**PASS_ADD_ONLY_CONFIG_APPLY_DESKTOP_INTAKE_NOT_RUN**. Hash baseline/candidate/
Desktop configuration/token НЕ вычислялся, token не открывался; preferences/user
paths/content не копировались в receipt или публичный отчёт. Helper/module/app/
provider/model/network/CLI автором QA не запускались; никакой новый harness.

Root сообщил stale/недоступное рабочее окно и попросил владельца вывести Claude
на передний план. Установка config сама по себе не доказывает загруженный MCP
catalog, actualmodel Describe/Submit или multiply workflow. Все эти Desktop
runtime gates пока **NOT-RUN**; новых workflow/role/test результатов не заявлено.
CLI Claude остаётся отдельным NOT-RUN target, прежний Codex PASS не повторялся.

## Claude Desktop actual — independent QA PASS

Принят frozen final-1792847822075850000: manifest SHA
`97555579ca719f70c60968e6e02a2b401ae6740adf197e7ac32861579d4ae39e`,
все32 member hashes проверены. Compact independent receipt
`.local/task015/claude-desktop-qa/final-actual-qa.private.json` SHA
`458c48b624bc580a0cff9d8a8e30d312ed71c97e53b353a070bee63c2c1741e6`:
**PASS_CLAUDE_DESKTOP_NATIVE_INTAKE_AND_CODEX_ROLE_PIPELINE**.
Root после обновления canonical docs передаёт отдельную final evidence review273;
эта QA не подменяет итоговое root/273 решение TASK015.

Непосредственно просмотрены pixels пяти собственных chat screenshots: Describe,
Submit head/start/tail и финальный ответ. Видны actual Describe exactrequest/
coordinator/run_id/machine+возвращённая boardSession/project; actual Submit
original request/policyv1/полный exactmultiplytext и queued/version1 workflowID/
intakehash. Их server completed payload содержит тот же текст/hash/requester/grant.
Собственный Desktop chatID виден в native AX receipt; модель закончила turn и
вернула фактический ID. Это не MCP catalog/source/SDK simulation.

Граница capture уточнена по independent273 readback: оба сохранённых файла
**describe.ax.txt и submit.ax.txt** содержат только stale «nochange»/focused control,
без полного набора аргументов. Утверждение о Text500 не подтверждается именно
архивированным AX и не используется как evidence. Точные Describe args/session
проверены по describe.png; полный Submit text/request/policy/returned ID/hash —
по submit-response-head/start pixels и server exacttext/hash. UI режим «Авто»
остался прежним; individual per-call dialogs не наблюдались, отдельное нажатие
Allow не приписывается. Model display — только UI label, не серверная attestation
конкретного underlying model. Аккаунты/auth/cache/чужие чаты не читались.

Workflow completed/version25/epoch1:4 роли/4nativeUUID/4invocations/0fix, все tasks
и epic done; независимые4author sessions. Полные commandpages, start applied,
все4 current local native_exited/exit_durable/managedACK/exitACK/lockrelease=true,
pending0/reservations0/leasesabsent/questions0. Нового Desktop pause trial не было
и не требуется: принимается уже подтверждённое Codex управление с сохранённой
gen1 capture оговоркой; тот тест не повторялся и не переименован в Desktop control.

Developer commit `bf77b6db0fadfc5a18da3045f0f4fd5bf510a314`, parent
`2a2a530c50c51290eb7653bcfadb979956f772ff`, clean; только src/math_ops.py и новый
tests/test_multiply.py. Saved baseline/after AST add/subtract сравнены независимо,
6 protected файлов совпали, новый тест содержит3ordinary methods. Reviewapprove
и testerpassed относятся к T-16generation1/exactSHA, author sessions независимы.

Два настоящих persisted developer/tester XML прочитаны: **8+8testcases**, ни
failure/error/skipped; reportSHA совпали с local check и public attestation.
Typedexit0/literalfalse truncation/source_clean/manifest/target/run/gen/epoch1/
project-tests-v1 digest сверены. Saved Codexconfig равен принятому c17baseline;
Claude CLI gate false. SHA Desktopconfig/token не вычислялись, новых config writes,
provider/UI/API/network/process/test/helper/model действий QA не выполнял.

Новый подтверждённый клиент — **Claude Desktop**, роли — прежние Codex на одном
Mac. Claude Code CLI auth/nativeintake, Claude execution roles, Windows и второй
физический ПК остаются NOT-RUN. No push/merge/deploy; обе прежние истории сохранены.
