# WORK190 — gated Standard/Interactive config-only comparison

23.10.2026. Отдельный tester; root координатор, reviewer169. По169 PLAN APPROVE
подготовлен fresh private probe. Runtime **NOT-RUN**: нужны SOURCE APPROVE
новых helper/binary/runner и отдельный exact root GO.

## Почему новый probe

WORK188 остановился до samples: первый label-bound OS census показал UID0 и
PGID1, следующий через41ms — UID501 и собственный PGID. Это возможный startup
transition, но не доказанный root cause. Исторические188FAIL и184FAIL/manual
flags не меняются. Background не повторяется; только Standard/Interactive.

## Startup gate

Перед любым `readNative`/`verifyCodexMCP` private Go test до20s ждёт собственный
gate file. Spec содержит fresh256bit nonce для каждого context; nonce/args/env
не попадают в reports. Helper проверяет gate path, regular0600 file/current
nonroot owner, Lstat/Fstat same-file, bounded4096byte read, nonce и точное
совпадение ожидаемых PID/UID/PGID с runtime; PGID должен равняться PID.
Только после этого разрешены прежние3 discovery/verification samples.
В report сохраняется `startup_gate=pending/accepted`, без capability value.

Runner до публикации gate записывает label-bound PID/UID/comm/starttime/PGID
как **pending/untrusted**, не принимает эти rows в family и не сигнализирует
PID. Два последовательных census должны показать один birth identity,
UID501, exact fresh helper path, PGID=PID. Тогда gate публикуется exclusive
атомарным hardlink из созданного0600 file после fsync. Existing destination
или pending collision дают FAIL; чужой pending file не удаляется.

После gate действует строгая прежняя family tracking: stable label identity,
UID501/PGID=PID/exact helper, observed descendants, immediately-prebootout census
и bounded postbootout absence. Нет numeric PID/group signals. Только own label
bootout. Pending transition не является ослаблением native ownership guards:
до gate native процессы вообще не создаются.

Runner gate25s, helper gate20s и outer100s deadlines bounded; native5s deadline,
Wait/PGID/ownership suffix fail-fast прежние. Gate failure/outertimeout/missing
report/ownership uncertainty останавливают все последующие contexts, оставляя
FAIL/manual inspection. Cleanup/pins/baseline/service checks и always-report
сохранены. `ps lstart` секундной точности не kernel birth nonce; unobserved
descendants не объявляются proven absent. Возможности containment не расширены.

## Private source-ready

Root `.local/task013-live/qa/qa190-1792762373483083000/` содержит неизменные363
owning sources committed `62e354525b83a2b671ab37ede938ffe7e838e24b`, прежний
manifest и один обновлённый private QA Go test вне owning manifest.
Persistent EnvironmentVariables/WorkingDirectory/ExitTimeOut15/ThrottleInterval
копируются как ранее; отличаются Standard/Interactive ProcessType и собственные
label/args/logs/one-shot lifecycle. Current production tree не используется.

- `run_classes.py`216 строк — `469d93b926d7c1b3c84b8970237935f006547cf04aca2c8291752a46cae3ce3e`;
- `process_inventory.py`154 строки — `969ddbc8c5d6bfcc73127cfc188312919c428165f9375c7cd2f2f8a8b12119b5`;
- private `qa177_process_test.go`177 строк — `75ae54889c15f123405005da270503d4243504a1b9f6c7344bb169096a0fe912`;
- compiled `qa190-worker.test` — `81b1827610510bb1804def5a0bb20194393faa92258f049d03aadbe833a9e1fd`.

AST parse, gofmt, compile-only `go test -c`,363 source pins PASS. Новый helper
не выполнялся, LaunchAgents не создавались; native CLI/model/MCP tool calls0.
GUI, production и постоянный worker/config/plist не изменялись.

## Actual execution — FAIL, collection incomplete

169 independently SOURCE APPROVE и отдельный exact root GO получены на четыре
вышеуказанных hashes. Единственный запуск 23.10.2026 в13:37:07 UTC создал
requested Standard job, actual spawn type `daemon (3)`. Два стабильных OS
samples подтвердили exact helper PID48335/UID501/PGID48335/birth; gate file
опубликован в13:37:08 UTC. Это publication proof, не helper acceptance:
Go-test entry/log/result не появились за100s. Helper `startup_gate=accepted`
не получен. Native configuration samples отсутствуют.

Outer deadline корректно остановил runner exit1; **Interactive NOT-RUN**,
повторов/Background/model/MCP actions нет. Private log0bytes. В645 safe census
samples tracked family содержала только helper; native descendants не
наблюдались. Не делался stack/xattr probe и не менялись quarantine/permissions.
Root cause, включая возможную macOS startup policy, не доказан.

Сразу перед own-label bootout в13:38:48.010 UTC зафиксирован прежний exact
helper PID/UID/PGID/starttime. Postbootout в13:38:48.053 UTC observed family
и exact helper отсутствуют; own labels absent. Cleanup failures отсутствуют,
все9 input pins/source363 и GUI3316 baseline unchanged, persistent worker
top-level running. Manual/timeout flags остаются true; sampled-process
limitations из source contract не снимаются. Numeric PID signals не было.

Private `.local/task013-live/qa/qa190-1792762373483083000/comparison.json`:
SHA256 `04f0ba65c155e9de7bc966abd33d71dc9604b0d043a8265eb09a61af4c4e4f17`.
`result=FAIL`, `collection_complete=false`, `collection_failure=RuntimeError`.
Process proof — `standard/process-inventory-private.json` в том же root.
WORK184/188 исторические failures сохранены. Дальнейшая диагностика требует
отдельного reviewed scope и root GO.
