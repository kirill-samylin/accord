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
