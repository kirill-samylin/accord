# Управляемые агенты — TASK013

22.10.2026. Контракт запуска, остановки, продолжения и уточнений из Accord, включая Windows-компьютер коллеги.
Фактический статус и проверки фиксируются в TASK/WORK; этот документ не PASS.
Расширяет прежнее ограничение REQ-01 о запуске, сохраняя все ACL, lease,
generation, idempotency, profile snapshot, coordination и deduplication gates.

## Границы и понятия

Участник явно устанавливает локальный `agent-board worker` и разрешает конкретный
проект, workspace и provider. Worker только исходяще опрашивает Accord по HTTPS;
открытого порта, SSH, удалённой shell, серверного clone или provider credentials нет.
Первый сценарий: macOS + native Codex CLI и Windows + native Claude Code CLI.
Третий провайдер — корпоративный `gennadiy` (fork OpenCode): headless `run`,
тот же worker, те же ACL/lease/bridge; см. раздел адаптеров.
Accord управляет только процессами, которые запустил этот worker. Уже открытый
терминал/IDE/desktop-чат нельзя выдать за подключённый управляемый процесс.

Host — разрешённая установка worker в одном проекте. Profile — versioned
инструкции/роль. Run — одна управляемая provider conversation с последовательными
execution attempts. Board session — отдельная серверная роль/identity/аренды.
Provider session ID — реальный ID native CLI conversation. Эти ID не заменяют
друг друга, online означает только свежий authenticated heartbeat worker.

## Локальное согласие и полномочия

Локальный private config фиксирует `server_url,project_id,host_id`, file reference
на credential, workspace aliases→canonical absolute roots, provider→absolute
executable, разрешённые model/permission presets и controller principal IDs.
Секреты не попадают в репозиторий, process argv, browser, logs или server response.
Provider login остаётся у владельца ПК; worker запускается от его OS account.
Workspace и executable разрешает только локальный владелец. Из панели передаются
alias/provider/profile/text, но не path, shell script, env, executable или argv.
Symlink/reparse/UNC/network root не принимается; cwd снова проверяется перед spawn.

Каждая human mutation требует live project owner/editor и независимое право
управления host. Viewer, profile role и coordinator role сами права не дают.
Пользователь выбрал «оба управляют агентами обоих ПК»; это оформляется
взаимными explicit controller grants. Каждый grant требует явного
локального согласия владельца host. Project owner не получает control чужого ПК
из одной роли проекта. Серверный список controllers пересекается с локальным
allowlist. Выбор «оба управляют обоими» оформляется такими grants, а не ослаблением
ACL. После revoke/epoch change replay тоже проверяет current auth.

Worker использует owner bearer + отдельную management human X-Session и host proof.
Host proof отдельно не разрешает catalog/task CRUD; локальный project bearer
приватен и используется отдельно для per-run bridge. Host scoped к owner epoch.
Каждый poll/report проверяет
current owner membership и host state. Отзыв owner membership/host останавливает
выдачу команд; worker при отказе авторизации останавливает owned process scope ниже.
Локальный аварийный stop и shutdown worker доступны независимо от сети.
Нельзя обещать мгновенный remote stop при выключенном/недоступном ПК.

## Данные и транзакции

Project-local board schema5 добавляет `managed_hosts(key,data)`,
`managed_runs(key,data)`, `managed_commands(key,data)` и
`managed_metadata(id=1,version=1)`. Registry1/web2 сохраняются.
Unknown/current incomplete schema fail closed, не восстанавливается CREATE.
До DDL — durable private schema4 snapshot; прежние rows не переписываются.
Индексы/поля JSON — implementation detail, но квоты/authorization атомарны.

`Host={id,owner_principal_id,name,version,state,controllers[],workspaces[],
providers[],worker_version,last_seen_at,worker_epoch}`. States active/disabled;
online вычисляется по last_seen (≤30s), не хранится как доказательство процесса.
Server хранит hash отдельного worker credential, никогда provider login.
Workspace view содержит alias/display name, не абсолютный локальный путь.

`Run={id,host_id,version,provider,workspace,profile_snapshot?,task_key?,
created_by,state,generation,provider_session_id?,board_session_id?,
created_at,updated_at,last_error?,output_cursor,output_truncated}`.
States queued/starting/running/stopping/stopped/completed/failed/needs_recovery.
Native process exit0 означает completed turn, а не task done или review approve.
Provider ID immutable после первого подтверждения; resume обязан совпасть.
Generation возрастает при каждом новом provider execution attempt, никогда при
простом poll/replay. PID и OS process birth identity остаются локально.

`Command={id,request_id,run_id,sequence,run_generation,kind,text?,created_by,
created_at,status,dispatch_id?,accepted_at?,applied_at?,result?,error?}`.
Kinds start/stop/resume/clarify. Status queued/accepted/applied/failed/
delivery_unknown/cancelled. Один accepted execution-changing command на run.
Статус accepted означает durable local journal receipt, не исполнение. Command
фиксирует issuer access_epoch; current credential+membership+тот же epoch нужны
перед dispatch/accepted/start, revoke/regrant не оживляет старую очередь.
Start/resume applied — подтверждённый provider start с real session ID;
stop applied — подтверждённый exit managed CLI и owned PGID (Unix) / Job (Windows);
clarify applied — наблюдённый native turn после подачи именно этого сообщения.
Applied не означает, что агент согласился или выполнил просьбу.

Mutation атомарно проверяет auth→idempotency→expected version/state→изменение+
command+audit+native response. Exact request_id+payload replay возвращает то же;
иной payload конфликтует. Sequence server-generated. Worker report проверяет
host/run/generation/dispatch_id и monotonic command phase, stale/foreign ACK
отклоняется. Transport ACK не меняет queued напрямую в applied.

## Start, stop, resume и уточнение

Start принимает `host_id,workspace,provider,profile_id,profile_version,prompt`
и optional `task_key`. Точная immutable profile snapshot pinned на создание;
provider/model должны соответствовать локальному allowlist. Task reference —
контекст, не claim: существующий native `describe→next` сам проверяет assignment,
gates и generation. Не получилось получить нужную задачу — процесс не пишет код
под её видом. Нельзя запускать второй run в занятом workspace alias; явный
отдельный локально разрешённый checkout нужен для параллельной записи.

Stop queued start отменяет его до dispatch. Stop running переводит stopping;
worker прекращает новые provider turns и завершает managed CLI + owned PGID/Job,
после bounded platform-specific grace принудительно завершает остатки
(Unix до3s; Windows Job Object termination сразу forced). Forced stop отображается
отдельно. Валидные completed/stopped/recovered_stopped (failed только с exit proof)
атомарно releases task leases и истекают только authoritative run/gen/hostowner sessions.
Попытка закрывается, task ready; generation/evidence/done сохраняются, аудит released/
managed_session_closed. Новый next/role binding получает новое поколение; unknown и
обычный EOF не дают release/expiry, произвольного human release чужой session нет.

Resume допустим только после доказанной остановки/exit и с сохранённым exact
provider session ID. Conversation продолжается на том же host/workspace/provider.
Resume не означает возврат истёкшей task/role lease: bridge обязан открыть новую
board session при утрате и снова пройти next/gates. Нет session ID — явный failure
resume_unavailable; нельзя молча открыть новый чат и назвать продолжением.

Clarify к running run сохраняется в durable mailbox и отображается «В очереди».
Первая реализация использует границу CLI turn: worker ожидает завершение текущего
turn и запускает exact native resume с сообщением. Кнопка «Остановить» позволяет
прервать текущую работу отдельно. Уточнение не отменяет работу скрыто и не
имитирует live injection в headless CLI без документированного provider API.
Для stopped/completed run clarify само не стартует процесс: «Продолжить» отправит
очередь в порядке sequence, включая текст продолжения. Failed/needs_recovery
не стартует до явного recovery. Конфликтующие новые команды получают409.

## Poll, процесс и восстановление

Worker poll раз в2s (backoff2..30s), report heartbeat раз в5s. HTTPS certificate
проверяется; явный local test mode допускает только loopback HTTP. Redirect на
другой origin и любой downgrade отвергается. Endpoints имеют bounded bodies/
timeouts; одна poll instance под OS exclusive lock на private host state dir.

Dispatch сохраняется сервером перед ответом. Повтор poll возвращает тот же
dispatch_id; accepted command не превращается в новый start по timeout/offline.
Worker fsync записывает receipt и launch intent до spawn, затем process identity,
provider session events и ACK progress. Output/network failure не даёт повторно
spawn того же dispatch. Удаление локального state требует нового enrollment.
Два worker с одним private state конкурируют за OS lock; скопированный credential
не разрешает независимый launch: worker epoch/poll ownership конфликтует.

На macOS/Linux отдельная process group; stop относится к owned PGID и managed
CLI, с проверкой принадлежности supervisor/run. Это первая заявленная граница:
`setsid`/double-fork/detached subprocess может выйти из группы и пережить stop.
Полный descendant tree, OS containment и остановка всех writers не обещаются.
Локальный workspace mutex кооперативен, не OS-барьер для любого writer. Фоновые
сервисы/отделившиеся процессы требуют локальной проверки владельцем.
На Windows native Job Object с kill-on-close,
provider создаётся suspended→assign job→resume: дочерний процесс не ускользает до
assign. Нельзя убивать по одному сохранённому PID или по имени claude/codex.
Если birth identity/ownership доказать нельзя — needs_recovery, без blind kill.
Worker exit/deadman прекращает owned scope и fsync terminal state; stopped
означает именно этот scope, не доказательство отсутствия detached writers.

После restart worker сначала reconciles private journal, не получает новый start.
Доказан живой собственный процесс — восстановить observation/stop; доказан exit —
отправить прежний terminal report. Crash между spawn и сохранением identity,
между native message и receipt, потерянный provider ID или неизвестный owned scope —
needs_recovery/delivery_unknown. Автоповтор side effect в этом окне запрещён;
владелец подтверждает отсутствие старого процесса до нового attempt.
Нет автоматического переноса run на другой host и распределённого exactly-once
provider execution. Offline UI показывает последнее подтверждение и timestamp.
При потере control heartbeat на30s worker останавливает owned scope; accepted report
сохраняется локально и повторяется после сети. Возобновление требует команды.

## Native adapters и board bridge

Worker использует exec argument array без shell. Prompt передаётся stdin/private
file с stdin redirect, never shell interpolation. Worker сам собирает allowlisted
flags из локальных presets; панель не разрешает bypass permissions или произвольный
config override. Permission denial становится видимой ошибкой/блокировкой.
Локальный provider preset сохраняет согласованную политику чтения/записи и approvals.
Отдельный default-off native MCP consent — [native-agent-permissions.md](native-agent-permissions.md).

Codex: `codex exec --json -`, затем `codex exec resume --json UUID -` с cwd workspace.
`thread.started.thread_id` сохраняется до success ACK; проверяются turn.started,
turn.completed/turn.failed и exit status. Claude: `claude -p --output-format
stream-json --verbose`, stdin prompt; resume добавляет `--resume UUID`.
`system/init.session_id` либо документированный result session_id подтверждает
conversation; success требует terminal result без error и успешного exit.
Gennadiy (OpenCode fork): `gennadiy run --format json --agent build`, prompt
через stdin; resume добавляет `--session ses_…`. Каждое событие несёт `sessionID`
формата `ses_` + base62; `step_start` подтверждает начало turn, `step_finish`
без последующего `error` и exit0 — завершение. Flag `--mcp-config` отсутствует:
worker передаёт только этому child переменную `OPENCODE_CONFIG_CONTENT` с
записью `accord_managed` этой попытки, `enabled=false` для каждого обнаруженного
через `debug config` чужого connector (URL/headers не копируются) и
pinned permission preset. Headless `run` автоматически отклоняет любой
native «ask», поэтому preset явный: `edit`/`bash` allow только при локальном
`gennadiy_permission_mode=workspace-write` (automation — только developer и
tester с artifact consent), `external_directory` deny, bridge tools по
`native_mcp_approved_tools.gennadiy` (allow выбранные, deny остальные: native
default для MCP — allow, а не prompt). Второе config-only чтение с инъекцией
проверяет точное binding и policy до spawn. `--dangerously-skip-permissions`,
`--continue`, `--fork`, `OPENCODE_CONFIG_DIR` и подмена глобального config не
используются; ambient `OPENCODE_CONFIG*` окружения worker не наследует.
OS sandbox у OpenCode нет: workspace-write означает обычный shell владельца.
Нет `--last`, `--continue`, implicit model fallback или поиска чужого чата по имени.
Префлайт проверяет установленную версию/flags; unsupported provider не получает
фиктивную capability. Windows native executable и process tree проверяются реально.

Для каждого run/новой board session создаётся отдельный stdio bridge private state
dir. Provider MCP config фиксирует project scope и этот bridge. Один bridge не
делится автором/reviewer/tester, worker не подставляет родительскую session.
Bridge credentials берёт из локального project credential file. Все операции идут
через прежний native API и ACL; worker presence не даёт право task completion.
Profile instructions и run prompt включают правило прекращения записи при lease
loss; heartbeat мостом не превращается в бессрочное право после потери worker.
Конкретная native проверка обязательна, SDK/stub не подтверждает Claude/Codex.
Bridge binding `managed_run_id/managed_generation` фиксирован в immutable
`runs/ID/attempt-N/bridge.json` и инжектируется bridge, не выбирается моделью.
Managed Describe требует current host owner/epoch, live worker lease, точный run
и accepted/applied execution текущего generation; берёт только server run snapshot.
Edit/archive профиля не меняет этот snapshot. Ordinary Describe требует current
active профиль, а task assignment/next/SHA/lease gates остаются прежними.

## API, browser и ограничения

Native scoped: `managed_list`, `managed_run`, `managed_host_enroll`, `managed_host_update`,
`managed_host_disable`, `managed_start`, `managed_command`; worker-only `managed_poll/report`.
Каждая human write содержит request_id; update/command — expected version.
Worker fields: host_id, worker_epoch, dispatch_id, run_id, run_generation,
report_id, phase, provider_session_id?, output entries?, error?. Report IDs
идемпотентны; credential/owner/project сервер выводит из auth, не payload.
Routes `/api/v1/p/{id}/managed/...`; browser `/web/p/{id}/managed` и
`/managed/runs/{id}`; Next `/app/p/{id}/managed`. MCP сохраняет прежние12tools.
Browser POST использует issued action_id/CSRF/same-origin и прежний uncertain
action recovery; query/cache keys и pending commands всегда scoped project.
PageData capabilities проверяют actor+host ACL; server повторяет authoritative
check на mutation/replay. UI отдельно показывает host online, process state,
command/delivery status, board lease/task и provider conversation ID.

Квоты:32hosts/project,128 live runs+workflow reservations/project (terminal history
≤512 в live state, старше — audited archive, automation.md §6),8active,1active/workspace,
256обычных commands/run с отдельным257-м reserve stop и запретом concurrent
stop;16pending обычных команд/run, prompt≤16KiB, clarify≤4KiB, list limit1..100.
Output: сервер только нормализованные короткие events/assistant excerpts,
до64KiB/run в bounded tail; ≤4KiB/event,≤16KiB/report. Raw terminal/tool/env/
reasoning output не загружается. Worker отбрасывает ANSI/control sequences,
редактирует известные credentials/bearer/url credentials до отправки и ограничивает
строки; redaction не считается гарантией скрыть произвольный секрет из кода.
Показываются `truncated`/dropped count. Full provider transcript остаётся локальным. Список runs содержит только metadata,
без prompt/output/profile instructions/description; detail отдаёт последние≤100
commands с encoded JSON budget256KiB и `commands_truncated`, полный audit durable.
Private journal хранит metadata/mailbox, не бесконечный stdout; rotation≤10MiB/host.
Quota overflow явный409, command/audit история не удаляется молча ради нового start.

### Точный wire и ownership handshake

Native routes ниже POST JSON; GET `/managed` читает тот же список, что
POST `/managed/list {}`. Строгие unknown/null/
duplicate fields и64KiB body. Scope `/api/v1/p/{id}/managed/`. Existing bearer и
`X-Session` guards обязательны. Native management human session не per-run session.
`hosts/enroll`: `{request_id,name,host_secret,controllers,workspaces,providers}`.
Worker CLI генерирует random256bit host_secret и private local state до запроса;
сервер хранит только SHA256, response `{host}` без secret. Request idempotency hash
содержит hash proof, не raw payload. Ownership principal/epoch сервер берёт из
auth. Host enrollment/controller/workspace changes только native CLI с existing
proof для update; браузер этого operation не имеет. Controllers — project-local
principal IDs, не подписи owner/machine; CLI показывает перед consent понятные имена.
`hosts/update`: `{request_id,host_id,host_secret,version,name,state,controllers,
workspaces,providers}`. Новый owner epoch требует нового explicit enrollment.
`list {}` → `{actor_principal_id,observed_at,hosts,runs}`; `run {run_id}` →
`{run,commands}`. `start`: `{request_id,host_id,workspace,provider,profile_id,profile_version,prompt,
task_key?}` → `{run,command}`. Snapshot не client supplied.
`command`: `{request_id,run_id,version,kind,text?}` → `{run,command}`.
`poll`: `{request_id,host_id,host_secret,worker_id,worker_epoch?}` →
`{host,worker_epoch,lease_expires_at,command?,run?}`. Worker ID random local startup
nonce, не PID. Первый poll atomic bind owner; same worker renews30s lease. Other
worker ID при живом/ambiguous run получает409 (`MANAGED_UNCERTAIN`/`LEASE_HELD`), не automatic takeover по timeout. Исключение: run в `needs_recovery` не блокирует новый worker того же journal (host secret + exclusive state lock); сам run остаётся в карантине до `worker recover`.
Полученный dispatch_id навсегда связан с command/run generation/worker ID/epoch.
Повтор poll того же worker возвращает этот dispatch; accepted command после receipt
не повторяет side effect. Новый worker после restart обязан локально reconcile:
`report phase=recovered_stopped` допускает только terminal old dispatch, proof и
новый worker ID; он НЕ доказывает отсутствие процесса сам по сети. Локальная
проверка обязательна, неоднозначность phase=delivery_unknown и needs_recovery.
Recovered_stopped текущего live lease holder (тот же worker ID+epoch, живая lease:
local deadline, `worker recover`) — не takeover: lease/ID/epoch сохраняются, его
run_exit принимается; другой worker ID — только после истечения lease, lease сброшена.
Workflow pause/cancel/question/timeout после dispatch только добавляет owned stop:
start/resume сохраняет статус, его failed/stopped принимается; delivery_unknown —
лишь при потере полномочий инициатора/grant. accepted/started по-прежнему 403.
После terminal reconciliation нового epoch выдаёт следующий explicit command.
`report`: `{request_id,host_id,host_secret,worker_id,worker_epoch,command_id,
dispatch_id,run_id,generation,phase,provider_session_id?,error?,output?}` →
`{run,command}`. Receipt phase=accepted: queued→accepted без нового process state;
started: start/resume accepted→applied + run running + exact session ID;
stopped: stop accepted→applied + run stopped; completed: run running→completed,
для clarify accepted→applied только after native turn receipt; failed: command→
failed/run failed; delivery_unknown: command→delivery_unknown/run needs_recovery.
Final report replays exact response после current auth/proof/grants без повторного
продления lease; временные phase/fence checks относятся только к новой mutation.
Backward/foreign transitions409. Reports с
истёкшим ownership lease не утверждают running; terminal own-dispatch recovery
может лишь остановить/пометить неизвестность. Stop до первого poll cancels queued
start. После dispatch stop отдельная queued команда, старый start не исчезает и
не перевыдаётся; worker сначала локально cancels/terminates, затем ACK обоих.
Worker локальный monotonic deadline не позже request-start+30s-grace, stop owned scope до
него. Истечение серверной lease показывает ownership unknown, никогда stopped.

## Capable поставка и приёмка

Capabilities `managed_agents:1,board_schema:[5]`; permanent independent floor
сохраняет registry/ACL/profiles/coordination/deduplication floors. Старый schema4
binary запрещён до первого managed write/DDL. Только capable projects/upgrade.py:
owned startup hold→all-project backup→install guards→schema5 migration→preservation
→owned restart→HTTPS smoke→full backup. Registry1/web2, all projects/quarantine,
ACL и незатронутые rows сохраняются; backup/restore/preflight понимают schema5.
Initial TASK009 apply, downgrade, board-only restore/rollback запрещены.

Приёмка: автор unit/integration→независимый APPROVE→отдельный tester process/
browser и review новых тестов→synthetic capable upgrade/recovery/full restore.
Обязательны double poll/retry, ACL/revoke/epoch, owned PGID/Job stop/PID reuse,
Unix setsid probe (owned group killed, detached и unrelated untouched), crash spawn/
ACK windows, offline, version conflicts, quota/redaction, native exact resume и
реальный bridge/session isolation. Mac native и Windows native — отдельные факты.
Второй физический ПК и Windows без доступа остаются NOT-RUN; staging не production.

Первичные API проверены22.10.2026: [Codex non-interactive](https://learn.chatgpt.com/docs/non-interactive-mode),
[Codex CLI](https://learn.chatgpt.com/docs/cli/reference),
[Claude programmatic CLI](https://code.claude.com/docs/en/headless),
[Claude CLI reference](https://code.claude.com/docs/en/cli-reference).
