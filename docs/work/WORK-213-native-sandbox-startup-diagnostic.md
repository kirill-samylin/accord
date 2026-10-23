# WORK213 — закрытая диагностика failed OS probe209C

23.10.2026, root. CONTRACT DRAFT, source/runtime NOT-RUN.
209C один approved run: normal product proof refresh A=C PASS, затем direct
sandbox command exited71 до syscall case output. Results
`599f0a9959a9023179bdd94ddd6d1c773730cada7b80123c1ab0b0ff2c7b31be`;
new frozen environment SHA
`fa10f8a83f1ad16b534dee41a12b00811c56b846a997dd625230e2815f73d2a8`.
Owned proof proven/!forced, source/config/env unchanged, model0. Ни OS-denial,
ни ошибку аргументов, ни конкретное ограничение ОС это пока не доказывает.
B и pipeline runtime закрыты.

## Ограниченный диагностический контур

Product code a32 не меняется. startProcess намеренно отбрасывает native stderr;
изменять это глобальное поведение ради теста нельзя. Separate QA пишет private
broker по уже reviewed207 pattern: generic candidate supervise владеет одной
PGID broker+native child, Popen без new session/group, native wait/EOF/drain
учитываются отдельно. Никаких numeric PID adoption/killall/чужих процессов.
Native stdin — immediate EOF, как GenericSupervise с empty Prompt через
strings.NewReader(""); broker использует DEVNULL, а не удерживаемый open PIPE.

Разрешается подготовить **один** повтор exact C sandbox command только для
private diagnostics: тот же pinned native/Python/probe source/spec, A checkout,
named-profile overrides и exact frozen C environment file/hash. Никаких CLI
fallback/правок global config/расширения прав/model/thread/auth/tools/MCP/RPC.
Сам probe обращается только к ранее consented synthetic files; user files вне
scope. Это новый диагностический запуск, исходный209C FAIL не переписывается.

Native stderr — только exclusive0600 private file≤256KiB, stdout — separate
exclusive0600≤4MiB, без raw stdout/stderr/report/tracebacks. Broker public output
содержит только fixed stages/counts/known native exit/drain facts; native nonzero
не превращается в success из-за broker/supervisor exit0. JSON case result при
наличии остаётся private. Root читает лишь минимальную safe projection известных
error enums/фраз и типов, без secret/path/URL/argument values.

Общий90s+30s, native20s+7s cleanup; provisional atomic FAIL до child. Parent
lifeline прежний. Broker не выходит добровольно, пока native не reaped/readers
не drained; по неизвестному состоянию quarantine до bounded owned-group Stop,
manual FAIL, без новых команд. Positive group proof не заменяет неизвестный
native wait. Повторные cleanup signals маскируются, независимый deadline жив.

Before-after immutable C env/source/binary/probe/spec/A fixture/Git refs/config
и private user config snapshots обязательны. Никаких silent retries/restore.
Если получится syscall output, это диагностические факты для independent
classification, а не автоматический обход первоначального C FAIL/gates.

## Ownership и порядок

Separate QA209 пишет только private `.local/task014/qa213/` и этот WORK после
CONTRACT APPROVE202. Root TASK/контракт, reviewer202 source/evidence отдельно.
Сначала contract approve, затем SOURCE review точных files/pins и отдельный
root GO на ONE diagnostic command. Никаких executions/compile в source prep.
Source213 не меняет203/210/212, GUI/Test/production. После причины нужны отдельные
corrective source gates и нормальный C acceptance до native B/pipeline.

## QA SOURCE READY — runtime NOT-RUN

После CONTRACT APPROVE202 и ROOT SOURCE GO подготовлены только private sources
`.local/task014/qa213/`: run.py174lines, broker.py152, raw_capture.py49,
continuity.py87, pins.json/README/SHA256SUMS. Новые executable sources не
импортировались/не запускались; только AST-only parse4PASS и read-only SHA
сопоставление141 immutable inputs (включая114 merged product files). Никаких
setup/compile/CLI/help/helper/config RPC/model/OSprobe/launchctl вызовов нет.

Frozen source manifest **aeb4254a1e234263c1df2e511dc7a7b9f32c66093b7fe33c032f0ca8e2a6f5e7**;
pins.json SHA **33cdb7bf5a3f3df712791099a0ab895bc1baef7bff6d7998bd3160e6ce97feef**.
Exact C command canonical hash81e62feb5a95f2384d670d2841b2b7ea651ee3e63ce4613d54965dc04c71ab66
и frozen C envfa10f8a8… независимо проверяет broker; никаких новых overrides
или discovery. Candidate a32/e9fc, native9394, Pythonbd349, identity0d85 прежние.
Raw env/command inputs не печатаются; A/C files/results/spec/source immutable.

Новый broker без new session/group наследует candidate-owned PGID и запускает
ровно одну exact native command; сохраняет capped private stdout/stderr и
отдельные native wait/drain факты. Native nonzero остаётся explicit FAIL даже
при broker/supervisor0; CAPTURED только diagnostic status, OS_acceptance всегда
NOT-PROVEN. По unknown ownership quarantine/manual, а не подмена native wait
родительским group proof. Bounds20+7/30/90+30, signal/finally/provisionalFAIL,
broker-safe.json в parent finally и config/source/env pins сохранены.

До и после — source-pinned initial hashes/refs/head checks и полный pure
filesystem snapshot checkout/.git/hooks/refs/config/own sentinel/identities.
Новых Git/helper commands не требуется. Сохранность и config byte equality
проверяются независимо от diagnostic result; ничего не restore/удаляется.
Independent202 SOURCE review и отдельный ROOT GO обязательны до ONE execution.
Предыдущий209C FAIL, A_PASS и closure B/pipeline остаются без изменений.

## SOURCE REOPEN — исправление stdin boundary, execution NOT-RUN

Reviewer202 нашёл P1: первый broker ошибочно удерживал native stdin PIPE открытым.
Исходный GenericSupervise передаёт empty Prompt в strings.NewReader("") и после
zero-byte copy закрывает child stdin (EOF). Диагностический broker теперь передаёт
DEVNULL, сохраняя exact argv/env/cwd/PGID. Это уточнение process boundary, не
расширение native прав. Полный old manifest/source/pins checkpoint aeb4254a…
byte-exact сохранён в qa213-first-freeze-aeb4254a; прежний source не исполнялся.

Полный findings batch202 завершён: других blockers нет. До replacement freeze
выполняется только AST regression: ровно один native Popen, его stdin обязательно
subprocess.DEVNULL, отсутствие новых session/group options. Runtime всё ещё
NOT-RUN; нужны новый exact SOURCE APPROVE и отдельный ROOT GO.

Replacement frozen manifest **0c4af3842e4ca390a01fa1b612a6e951c56690009864253c71cb6472d7a1faf5**;
pins **82e6919ca09ba5899d2d0bdbb38863c03c94b1c34ee7f313fd859354170b21a0**;
broker **620058d39251406b2d24ec0d8fd1c29ee8562a7eb020bca09a42f08d6e75ebed**.
AST-only regression PASS4files/onePopen/DEVNULL/no group options сохранён в
static-checks.json SHA db8f0560f3971ff306ec249f4115c649ddee311b79eb73abd4f6520e3e66e945.
Command/env/product inputs прежние; CLI/helper/native/process не запускались.

## Actual213 — diagnostic CAPTURED, native command FAIL71

После FINAL SOURCE APPROVE202 exact replacement manifest0c4af384… / pins82e6919c…
и отдельного ROOT GO выполнен **один** frozen213:18:25:06.933Z→18:25:08.204Z,
1.272s. GO consumed; повторов нет. Broker/candidate call0.862s.

**CAPTURED** означает только успешную закрытую диагностическую запись.
Native command **FAIL**, known native wait exit71, natural_native_exit0=false.
Native остался в той же owned PGID; stdout0bytes, stderr68bytes; оба EOF/drains
подтверждены, capture errors/manual/unknown нет. Broker/supervisor exit0 и
positive nonceproof proven/!forced отдельно подтверждают exit broker/group,
а не подменяют native71. Native syscall output отсутствует, OS acceptance
**NOT-PROVEN**. Models/RPC/MCP0; ровно один native sandbox command.

Raw stderr QA семантически не читал и не выводил: сохранены только private600
bytes/count/hash, причина **UNCLASSIFIED**. Для дальнейшего решения root/reviewer
разрешена только оговорённая минимальная safe projection из private capture;
никаких fallback/новых прав/CLI probe/native retry QA не выполняет.

Exact frozen C envfa10f8a8…, command81e62feb…, source/binary inputs before/after
unchanged. Полный pure-filesystem checkout/.git/refs/config/hooks/sentinels
snapshot и identities unchanged. Native user config raw snapshots byte-equal,
ничего не restore. Старые A/C sources/results/environment неизменны, GUI/Test/
production вне scope. B/pipeline runtime остаётся CLOSED.

Private HOME qa206-1792779906932542000, все files600; executed-source + reused206
и exact approved manifest0c4af384… сохранены. results.json SHA
ec963745c3066c0cb3746827b3779a037d68373f288e3b00d1aad686c3abd0f3;
broker-safe.json SHA757892e54b7450479eda47929dbdff6bea78fe141a58069b2998f25d0e319bd8;
native-stderr-private.bin SHA8b5d77dee0f5066511ccb9ec46b3935c56ff60937c94cd8f28dc813ddffe4614
(68bytes); native-stdout-private.bin empty SHAe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855.
Independent202 EVIDENCE review запрошен; corrective decision отдельно.
