# WORK193 — локальный исполнитель автоматизации

Автор: `/root/automation_worker_impl`. Основание: TASK014, automation.md,
WORK181 и independent contract APPROVE183. Только worker, per-attempt IPC и
`cmd/agent-board/worker.go`; core/store/transport/bridge/infra/web принадлежат
другим исполнителям. Product SOURCE READY ожидает независимого review; это не
native/production PASS.

## Реализовано

- Default-off `AutomationConfig`: четыре local role/provider allowlists,
  workspace identity, срок, пределы времени/числа invocation, source/artifact/
  pinned dependency roots. Static test/lint/build presets имеют exact local
  executable SHA256, argv, cwd, timeout, format/path результата. Только явно
  подтверждённый `trusted_owner_process`; unsupported enforced launcher отклонён.
  Digest связывает preset, workspace identity, roots/dependency hashes и отдельное
  artifact write разрешение. Рабочие ручные конфигурации не получают automation.
- Новые MCP grants `workflow_plan`, `workflow_question`, `workflow_run_check`
  добавлены в closed local enum без автоматического добавления в старый grant.
  Automation-enabled роли требуют своих explicit grants. Child не получает
  `workflow_submit`/`workflow_command` или policy/host control.
- Per-attempt capability: random256 secret с ExecutionTokenPrefix, digest полного
  secret, `ec_` +32hex ID. Private immutable token file fsync предшествует owner/
  host-proof registration. Lost response повторяет тот же request/digest; child
  bridge.json содержит только scoped token path и fixed execution binding, не
  parent owner bearer или host proof.
- Отдельный private IPC: Unix socket0600 в directory0700; длинный путь использует
  owner-checked `~/.accord-ipc/<random>` с теми же правами и cleanup после close.
  Windows Named Pipe имеет current-user/SYSTEM DACL, REJECT_REMOTE_CLIENTS,
  random128 name и overlapped bounded I/O. Random256 nonce только в private
  config. Strict uint32+JSON frame ≤64KiB, constant-time nonce, fixed workflow/
  step/run/gen/control epoch. Start/status принимают только preset/request IDs;
  status не создаёт execution. Maximum4 connections/1 active owned check.
- До проверки — actual owner GET own lease checkpoint, exact preset/hash/source
  verification, authenticated server reservation, durable local intent. Затем
  supervised static child, actual OS exit code, fsync bounded raw JUnit/TAP
  artifact и typed observed result, durable queue, idempotent server ACK.
  Resource не объявляет queued/running/report_pending завершённым.
- JUnit/TAP require реально описанные executed cases и полную структуру; пустой,
  zero-tests, all-skipped, invalid/TODO/bailout не дают PASS. Lint/build могут
  иметь exit-only proof, но не заменяют test preset. Parsed counts и digest
  относятся к одним прочитанным bytes; raw report сохраняется private immutable.
- Git использует pinned canonical executable, fixed read-only commands и
  GIT_NO_LAZY_FETCH. Full HEAD/tree/index проверяются независимо от
  assume-unchanged. Фактические tracked bytes/mode равны HEAD; source manifest
  включает untracked/ignored input вне явных exclusions. Dependencies pinned
  целиком, внутренние относительные links допускаются только внутри того же
  dependency root; внешние links, submodules/unsupported source symlinks blocked.
  Reviewer/tester требуют exact target SHA и неизменённый source manifest/HEAD;
  developer — clean actual HEAD, совпадающий с завершённой task evidence.
- Parent ownership poll остаётся независимым от подготовки/Git/IPC/HTTP checks.
  Native и checks получают общий stop/revoke/shutdown watchdog, отдельные
  monotonic attempt/check/workflow deadlines. Checkout lock держится до proven
  всех owned exits, durable observations и точного terminal+attestation ACK.
  completed ACK при server stopping не terminal; pending proof привязывается
  к реально подтверждённой stop command. Ambiguous intent/failed save/ownership
  не освобождают checkout, restart сохраняет pending facts и needs_recovery.
- Только после terminal+attestation ACK, доказанного выхода всех scopes и release
  компактируются дубли Prompt/Profile instructions/Output. IDs, generations,
  full snapshot hashes, receipts и observed proofs сохраняются. Не подтверждённая
  история и pending queue не сокращаются. Текущий 8MiB journal остаётся hard
  fail-closed capacity; automatic deletion/rotation отсутствует.

## Отдельное test_design согласие

`allow_tester_artifact_writes` default false. Для Codex true допустим только
вместе с уже явным `codex_sandbox: workspace-write`; изменение флага меняет
preset acknowledgement. False оставляет tester read-only. Reviewer/analyst
остаются read-only Codex. При true tester может предложить новый bounded `.patch`
только в approved artifact roots; source/HEAD всё равно обязаны остаться ровно
reviewed SHA. New valid patch hash + native `not_run` дают лишь test_design:
разработчик должен применить/закоммитить, затем independent review и настоящий
запуск. Это наблюдаемый source boundary в trusted-owner режиме, не OS filesystem
sandbox против malicious same-user/native code. Missing write consent/invalid
patch не выдаются за выполненные тесты.

## Файлы и API сопряжения

Новые worker-файлы `automation_*` разделяют config/consent, launch/capability,
Git/manifest, checks/reports/artifacts, terminal ACK/exit, compaction/recovery.
`receipts.go` вынесен из engine.go, чтобы не расширять coordinator mutex логику.
`shutdown.go` отдельно хранит global stop/admission barrier и закрытие journal.
Изменены engine/execution/preparation/runner/stop/journal/http/bridge_config,
process status/supervisor и native consent. Все новые source файлы менее300 строк.

`internal/automationipc`: Config{Endpoint,Nonce,Binding}; Request{Operation=start|
status,RequestID,PresetID,SessionID,TaskKey,TaskGeneration + fixed binding};
Response{CheckID,Status,Result?,Error?}. `Call(ctx,config,request)` выполняет bounded
exchange. `Listen(directory,binding,handler)` / Config / Close принадлежат parent.
Root bridge и cmd/mcp внедряют immutable data; model не определяет scope.

`BridgeConfig.Automation` + ExecutionCapabilityID передаются через root-owned
`runMCPWithAutomationBinding`. Worker пишет только `/automation/capabilities/register`
и `/automation/attest`, GET own `/tasks/{key}` с parent owner session. Локальные
read-only setup helpers AutomationIdentity/AutomationDependencyDigest/
AutomationPresetDigest возвращают только факты/digest, не сохраняют consent.

## Проверки

- PASS: author unit/integration `go test -race ./internal/worker ./internal/automationipc`.
  Покрыты exact nonce/frame/binding, duplicate/unknown fields, длинный Unix path,
  stale report removal, source/index/ignored/dependency link boundaries,
  nonempty JUnit/TAP, static-check replay, status-without-start, foreign lease,
  actual-report/exit без source immutability, explicit grants/test_design flag,
  scoped credential isolation, both owned scopes stop, terminal ACK binding,
  restart pending proof quarantine и bounded completed-history compaction.
- Linux/Windows amd64 test cross-build: PASS до финального pin, повтор перед review.
  Это не runtime Windows или второй физический компьютер.
- Native CLI/model/process/E2E, реальный полный pipeline, crash/Windows process,
  production install: NOT-RUN автором. Новые process/E2E — отдельный QA после
  source APPROVE; существующий manual Mac worker/GUI/VPS не менялись.
- Cmd tests без canonical TMPDIR воспроизводят старый macOS symlink temp-root
  отказ `unsafe web data parent`; final run использует private canonical .local
  temp. Это не скрывается ослаблением storage guards.

Официальный Git контракт optional-locks/no-lazy-fetch:
[git documentation](https://git-scm.com/docs/git). Windows транспорт использует
существующий x/sys/windows, обновления зависимостей отсутствуют. Unix PGID/Windows
Job supervision не доказывает containment Unix detached descendants; physical
Windows и OS/code trust ограничения остаются отдельной приёмкой.

## Исправления независимого review169

- Ownership failure при Git `ls-files` сохраняет исходный `errOwnership`, а не
  превращается в обычный dirty-index отказ. Неопределённый post-check Git exit
  переводит attempt/check в needs_recovery, останавливает native и сохраняет
  checkout intent; положительное или обычное negative finished proof не уходит.
- Deadline/stop, даже если программа обработала interruption и вернула exit0 с
  валидным passing TAP/JUnit, передаются authenticated `CheckAborted=true`.
  Настоящие exit/count/hash/source факты сохраняются; core185 gate запрещает
  qualifying PASS для любого required preset с этим флагом.
- Deferred accepted ACK привязан к stop epoch при приёме, а не при позднем
  запуске. Global StopAll блокирует новое admission, отменяет прежние deferred
  планы; Close закрывает admission до остановки. Детерминированный barrier test
  проверяет отсутствие spawn и для manual, и для automation dispatch.
- Terminal ACK проверяет точный исходный bound epoch внутри attestation;
  текущий workflow epoch может быть новее после human pause/cancel. Это metadata
  не продлевает authority. Reserve/start требуют текущий exact epoch; чужой
  bound terminal proof отвергается. Реальный SQLite+HTTP pause→terminal ACK→
  release→exact resume regression принадлежит root186 и прошёл общий race.

Final source scope: 97 Go files, fingerprint
`78c3ac53629e191653aa773b9f370856c20923c8470ecdab7491efb2ea96a468`.
Private pin manifest: `.local/work193/source-manifest.json`; алгоритм — sorted
relative path + NUL + SHA256 + newline. Root-owned
`internal/worker/automation_http_fixture_test.go` и
`internal/worker/automation_http_integration_test.go` исключены из193 pin и
включены root186. Ни одного файла scope свыше300 строк.

Final author checks: общий worker/IPC/cmd race PASS; Linux/Windows amd64 worker
test cross-build и Windows IPC cross-build PASS; `git diff --check` PASS.
Native Git commit permissions остаются непроверенными: разрешения native CLI
этим reopen не расширены; реальный commit/pipeline — отдельный QA после review.

## Read-only addendum — узкий native Git commit

Это проект решения, не изменение frozen product или разрешение на повторный
native запуск. По сообщению root, WORK198v2 доказал edit/owned exit0/native UUID,
но commit отсутствует; точная Git ошибка ещё исследуется QA. Изменение hash
пользовательского config требует отдельного объяснения, а не автоматического
restore. До этих фактов причина отказа не назначается.

Установленный CLI `0.163.0-alpha.11.1`: проверены только version/help/exec help/
debug help/doctor help. `--approve-for-me` существует; `--profile` загружает
`$CODEX_HOME/<name>.config.toml` поверх user config. `exec --ignore-user-config`
пропускает user config, сохраняя CODEX_HOME для auth. Модель, doctor diagnostics,
config parse и policy activation не запускались. Документация подтверждает
приоритет CLI над trusted project config, затем profile/user/system; пропуск
user config сам по себе не доказывает отсутствие остальных слоёв.
[Config basics](https://learn.chatgpt.com/docs/config-file/config-basic),
[non-interactive mode](https://learn.chatgpt.com/docs/non-interactive-mode).

**Предпочтительный узкий filesystem вариант — отдельный owner opt-in профиль.**
Предлагается оставить approvals `never`, расширить `:workspace` ровно на
canonical Git metadata одного consented checkout, сохранив config/hooks и
`.agents/.codex/.aws` read-only, native command network выключенным. Набросок
ниже не проверен парсером и не установлен:

```toml
default_permissions = "accord-local-commit"
approval_policy = "never"
[permissions.accord-local-commit]
extends = ":workspace"
[permissions.accord-local-commit.filesystem."/EXACT/CONSENTED/CHECKOUT/.git"]
"." = "write"
"config" = "read"
"hooks" = "read"
[permissions.accord-local-commit.network]
enabled = false
```

Предусловие: реальная directory `.git` внутри этого checkout; external gitdir/
worktree/symlink требуют отдельного рассмотрения. Список metadata можно сузить
после доказанного Git поведения. Это разрешение файловых записей, а не запрет
всех Git операций кроме commit. Все native commands остаются внутри OS sandbox;
сохранность локальной истории проверяется отдельно. Profiles beta; текущий193
всегда добавляет `-c sandbox_mode=…`, а любой загруженный legacy sandbox выбирает
старую модель вместо profiles. Следовательно, нужен отдельно согласованный
invocation/config-layer путь без legacy settings; простого `--profile` либо
добавления `.git` в writable_roots недостаточно.
[Permissions](https://learn.chatgpt.com/docs/permissions).

**Альтернатива — automatic approval review, только явно согласованный риск.**
`--approve-for-me` использует workspace-write + on-request + auto_review и
конфликтует с `--sandbox`; совместимость с текущим193 invocation не доказана.
Это общий reviewer для эскалаций, не Git-only capability.
[Developer commands](https://learn.chatgpt.com/docs/developer-commands).
Короткий `[auto_review].policy` применять нельзя: он заменяет активную политику;
документация требует сохранить полный active policy, иначе не переопределять.
Для добавочного ограничения есть `extra_policy`, но managed политика старше
локальной. Reviewer вероятностный; инструкцией нельзя гарантировать hard deny.
[Auto-review](https://learn.chatgpt.com/docs/sandboxing/auto-review),
[config reference](https://learn.chatgpt.com/docs/config-file/config-reference).

Если root выберет этот вариант, design требует отдельный default-off developer
opt-in, связанный с executable hash/version, workspace identity, сроком и policy
digest. Reviewer допускает только отдельные exact-path local add/commit в этом
checkout после read-only проверки diff/index/config/hooks; остальные escalations,
compound shell, произвольные helpers, push/network, auth/globalconfig writes,
destructive Git и неоднозначные действия отклоняются. Native отказ не обходится.
Нужен полный запрет остальных review categories там, где installed granular
semantics доказаны; иначе вариант не объявляется узким. MCP grants не расширяются.
При требовании детерминированного «только commit» нужен отдельный parent-owned
fixed Git operation, что выходит за этот read-only design.

Перед реализацией: independent contract review, затем отдельно разрешённый
config-only/effective-policy proof. После source approval — QA positive local
commit с real UUID/owned exit и сохранением exact resume; negative outside-root/
global config/hooks write, push/network и unrelated escalation; policy denial
должен остановить попытку, не включить fallback. Сравнить config/rules/auth hashes
до/после без вывода содержимого. Native QA и Windows physical здесь NOT-RUN.
Production/settings/product source не менялись;193 pin сохраняется.

## WORK201 feasibility — supervised config-only dialogue

Read-only review: `supervisor.go`, Unix/Windows process implementations,
installed schema201 и `codex app-server --help`. Текущий `launchSpec.prompt`
одноразовый: Unix использует конечный Reader, Windows пишет строку и закрывает
child stdin. Supervisor stdin остаётся отдельным parent lifeline; bytes после
handshake выбрасываются. Его нельзя использовать для native JSON-RPC.

Официальный app-server использует stdio JSONL, сначала initialize/initialized,
затем остальные методы. Schema201 JSONRPCMessage допускает отдельный object,
не batch-array; метода shutdown в ClientRequest/ClientNotification нет. Для
pipeline четырёх строк с немедленным EOF не найдена гарантия обработки запросов
и ответов до завершения. Такой batch shortcut не считаю доказанным.
[App-server protocol](https://learn.chatgpt.com/docs/app-server).

Минимальное предлагаемое расширение supervisor — закрытый `config_inspection`
режим, а не общий interactive RPC proxy. Внутри helper создаётся отдельный
child stdin pipe, работающий с прежними OS ownership/Stop/Wait primitives.
Parent lifeline сохраняет прежнюю семантику; parent API не получает произвольный
WriteStdin, model/remote не передают методы или JSON. Legacy prompt mode неизменён.

Фиксированная state machine внутри helper:

1. Spawn pinned executable с фиксированным `app-server --listen stdio://` и
   согласованными config overrides; без proxy/daemon/socket listener. Сразу
   владеть child, привязать cancellation к runtime.ctx/stop/revoke и local epoch.
2. Отправить initialize id1 с fixed clientInfo, без experimental/attestation/
   elicitation capabilities; дождаться единственного корректного ответа id1.
3. Отправить initialized notification, затем config/read id2 с exact canonical
   cwd/includeLayers:true; дождаться единственного ответа id2.
4. Отправить configRequirements/read id3 с пустыми params; дождаться ответа id3.
5. Только после этих ответов закрыть child stdin и дождаться natural exit,
   полностью дочитав stdout. Positive proof требует owned exit0, без force.
   Если EOF не завершает actual CLI корректно, timeout/Stop не превращаются в
   положительный результат: режим blocked до отдельного согласованного решения.

Предлагаемые hard limits: requests суммарно≤8KiB, line≤2MiB, stdout≤4MiB,
notifications≤32, dialogue≤20s либо раньше attempt/worker deadline; exit grace
≤2s входит в тот же deadline. Cleanup использует прежний bounded owned Stop;
его время учитывается отдельно и не даёт late native spawn. Duplicate JSON
keys/IDs, extra response/result+error, неизвестный ID, malformed/truncated/oversize
frame или любая server-initiated request — отказ без ответного tool/auth действия.
Notifications только из reviewed exact allowlist, bounded и без сохранения raw
params; неизвестный method блокирует. Никаких произвольных RPC после отказа.

Raw config/ошибки сервера не пересылаются как output и не сохраняются. Helper
проверяет permissions projection и возвращает только fixed safe result/enum
error, digest и согласованные identity/version факты; итог принимается лишь с
прежним nonce exit proof. На timeout/parent EOF helper закрывает child stdin,
останавливает owned scope и сообщает ownership uncertainty честно. Сохранённый
digest не заменяет повторную authority/config/path проверку перед native spawn.
Все blocking действия вне Engine.mu; active preparation не исчезает до cleanup.

Schema201 нюанс: config/read `Config` имеет additionalProperties:true, а raw
default_permissions/permissions не входят в его typed properties; layers[].config
вообще arbitrary JSON. layers name — tagged source object, version string,
disabledReason optional/null. Нельзя считать typed sandbox_mode:null активной
legacy настройкой либо предполагать, что разрешения вернутся по факту CLI flags.
Нужны actual raw selected-profile/layer facts; отсутствие/unknown shape блокирует
режим. Requirements дают allowedPermissionProfiles/defaultPermissions, но не
доказывают OS-compiled filesystem policy. Configured grants и negative OS QA —
разные доказательства. Отключение connectors при inspection не должно менять
permission projection относительно будущего native invocation.

Реализация helper, interactive process plumbing, parser и авторские unit требуют
contract approval201. Config RPC, native model и новые process sources здесь
NOT-RUN; их отдельный QA следует после независимого source approve. Возможные
startup side effects app-server не объявляются отсутствующими без проверки;
global config hash drift не восстанавливается. Source pin193 не изменён.
