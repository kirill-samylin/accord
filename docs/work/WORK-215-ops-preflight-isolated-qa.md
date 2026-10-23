# WORK215 — isolated Linux QA operator212

23.10.2026, root. CONTRACT DRAFT; source/runtime NOT-RUN.
212V2 SOURCE APPROVE202 manifest95e334fe2913e7b3436ba0b1e5f48212002c5f2778ab1a13736dda45731e2464
не заменяет actual focused validation перед operational GO. Separate tester,
не автор212, готовит private `.local/task014/qa215/` после CONTRACT APPROVE.
Root owns TASK/контракт; reviewer202 independently reviews source/evidence.

## Linux fixture scope

Разрешается подготовить ограниченный isolated Linux тест будущего service-UID
SQLite reader. В текущей среде Linux доступен на прежнем VPS через тот же pinned
transport199; eventual runtime требует отдельного source review/GO на transfer и
ONE fixture run. Никаких live database reads, real registry, services, pointers,
nginx, updater stage/apply/finalize или production credentials в fixture.

Новый exclusive root-owned prefix `/var/lib/accord-qa215-<nonce>` вне
`/var/lib/accord`, mode0710/group actual accord primaryGID: только service group
может пройти к synthetic state0700/UID999, root reports отдельно0700. Ancestors
root/service-owned и без group/world write. Не менять существующие пути. До
первого child durable private provisional FAIL/UNKNOWN, bounded transfer/run и
cleanup; локальные/remote source hashes и actual exit/EOF/drain обязательны.
Interrupted transport не доказывает remote exit; no retry/adoption/killall.
Unknown требует сохранения prefix и manual inspection.

Исполняется pinned212 live_inventory_remote.py с единственной source substitution
точного STATE literal на новый fixture path, expected one project. Source hash,
ровно одна literal replacement, unchanged remaining bytes фиксируются. Других
monkeypatch/guard bypass нет. Child actual UID999/его passwd primaryGID/groups[],
fixed env, query_only/WAL-aware read и closing такие же, как approved212.

Минимальные cases:

1. Positive existing0600 service-owned WAL/SHM: actual query видит committed row,
   которая ещё находится в WAL, не только в main DB. Это обязана быть строка,
   реально влияющая на unchanged reader projection, например busy count;
   exact expected projected value проверяется. Keeper открывает только synthetic
   DB, остаётся жив до завершения reader и затем positively closes. После reader
   child known exit и все SQLite handles закрыты. Prospective clarification после
   independent review: POSITIVE может менять только content hash трёх уже
   существующих fixture `-shm` файлов. File set/dev/inode/UID/GID/mode/size тех же
   файлов сохраняются; mainDB/WAL/прочие bytes неизменны. Все differences сначала
   сохраняются и явно отражаются в отчёте без утверждения причины. Новые aux,
   deletion/identity/mode changes и generic WAL allowance запрещены. Для четырёх
   NEGATIVES остаётся полная byte equality. Старый FAIL не меняется: он случился
   до reader body и не доказывал SHM drift.
2. Missing WAL, missing SHM, wrong-owner SHM, wrong-mode auxiliary — четыре
   isolated negative cases. Guard отвергает ДО connect: nonzero child сам по себе
   недостаточен, source trace/точный guard location и preserved files обязательны
   без instrumentation changes в reader. До/после file set,
   content hashes, UID/mode неизменны. Никаких repairs/aux creation/chown existing
   production/immutable fallback. Negative fixture mutation выполняет только
   родитель в собственном prefix до baseline каждого case.
3. Один local driver signal case с безопасным transport stub, без SSH: durable
   initial UNKNOWN уже существует до stub dispatch; SIGTERM либо SIGINT приводит
   к final UNKNOWN без remote-stop claim/automatic retry. Test source разрешает
   только точную проверяемую замену transport import на pinned local stub в
   private copy phase.py; production driver/source неизменен. Детали substitution
   и controlled child lifecycle требуют SOURCE review до execution.

Успех каждого reader child подтверждается separately known process wait/drains,
не строкой success от родителя. До/после immutable212/source/configs и fixture
metadata; known-owned cleanup bounded, никаких чужих процессов. Общий budget
execution120s+cleanup30s, child≤45s, raw bounded private0600 stdout/stderr≤1MiB,
никаких секретов/production paths values в public reports. Отсутствие нужного
service UID, safe prefix или валидной WAL fixture — FAIL/BLOCKED, не fallback.

## Gates

Только source preparation после CONTRACT APPROVE202. Затем exact manifest,
independent SOURCE APPROVE и отдельный root GO до запуска/transfer/SSH/import
transport. PASS215 нужен для operational212; он не заменяет native C/B/pipeline/
fix/control/UI и не разрешает production activation. Linux fixture не является
проверкой Windows/второго физического компьютера.

## SOURCE_READY215 V1 — отдельный тестировщик ops_validation_qa

Прочитаны AGENTS, approved контракт215/prefix0710 clarification, WORK212V2 и
independent SOURCE APPROVE202. Запись только private `.local/task014/qa215/` и
эта append-секция. Исходники212/product/TASK/production не изменены.

Подготовлены Linux bootstrap/one-shot launcher, service keeper, unchanged-reader
trace/audit wrapper, bounded capture/keeper drain, local SIGTERM/stub driver.
Original212 nine-file input copy сохранён с exact manifest
`95e334fe2913e7b3436ba0b1e5f48212002c5f2778ab1a13736dda45731e2464`.
Reader получает ровно STATE literal substitution; service UID999/actual GID/
groups[] и existing600 auxiliary guards сохраняются. Positive row влияет на
real unchanged busy projection: main busy0 после checkpoint, committed WAL busy1.
Negative cases: missing WAL, missing SHM, root-owned SHM, mode0640 WAL; audit
считает SQLite connect events, trace фиксирует exact original guard location,
не меняя functions/connect/module semantics. Negative fileset/content/UID/mode
before/after должен совпасть; positive тоже имеет строгую hash equality (может
честно FAIL на SHM read marks, это не runtime-наблюдение).

Signal fixture копирует phase/source/references в отдельное private дерево,
заменяет ровно transport-import path literal на pinned no-network local stub;
original driver не меняется. Synthetic manifest/source approval/rootGO fixture-only.
Initial durable UNKNOWN обязан существовать до stub child dispatch; один SIGTERM
exact owned phase child должен оставить final UNKNOWN/remote_exit_proven=false.
Local known wait/EOF/drains/group absence не выдаются за remote-stop proof.

Linux execution120+cleanup30/child45; one-shot transport150+exact owned cleanup,
streams<=1MiB/private600, reports/prefix сохраняются при unknown; нет retry,
PID adoption, killall, live STATE reads, service/pointer/updater действий.

Frozen V1 manifest SHA256:
`b6c5c6936ee88d81169c3b92124babe0ff52c5adfd90d110f4880fc71542a702`.
Pins SHA256:
`b63004a8536f0fa4f94dfe5ce467570a455693bc5cc3f84b2ae40b2b4a5ebe9a`.
21 manifest members,16 Python AST-only PASS, max190 lines/script, private600.
При подготовке scripts/import/transport/CLI/compile/process/test/model/RPC/HTTP/
SSH не выполнялись. SOURCE REVIEW202 pending; runtime NOT-RUN до отдельного GO.

## V2 source fixes — полный REQUEST CHANGES202 batch

По отдельному ROOT SOURCE REOPEN весь V1 byte-exact сохранён в private
`versions/b6c5c693/` (все21 members + manifest verified). Исправлены все3 findings:

1. P1 prefix: umask077 больше не оставляет exclusive prefix0700. Новый directory
   FD/O_NOFOLLOW получает fchmod0710, затем fstat проверяет root/actualGID/0710.
   Существующие каталоги не меняются.
2. P1 lifecycle: deadline задаётся перед Popen; stdin отправляется nonblocking
   selector chunks параллельно bounded stdout/stderr. Handle сохраняется перед
   pipe setup; partial setup/interrupt/overflow/timeout ведут в отдельную bounded
   cleanup и независимые попытки save обоих raw captures/process receipt.
   wait/drain/forced/unknown отдельно записаны; forced/unknown не становятся PASS.
   Wrapper SIGINT/TERM/ALRM теперь controlled. Cleanup отключает raising alarm,
   использует monotonic deadline и nested finally для final receipt; Linux120+30
   ceiling сохраняется даже после capture cleanup. Keeper UNKNOWN intent пишется
   до Popen; отсутствующий handle остаётся manual/UNKNOWN. Keeper EOF/drain/wait
   также имеет отдельные bounded private capture/receipt на failure.
3. P2 evidence: before maps записаны перед reader; after maps/trace/exit факты и
   полный набор predicates записаны ДО assert. Negatives требуют первого exact
   exception origin metadata на исходных guard lines42/43/32/33 соответственно,
   zero SQLite audit events и отсутствие connect line51. Failed guard/hash drift
   сохраняет evidence. Ожидания и212 guards не ослаблены.

V2 manifest SHA256:
`796d39dbc414d3d35cba9f7f669d55fc69bba56ed43af6108d85f4dc086b35ad`.
Pins SHA256:
`709cb83d54a2c3b1b46db89184c84fbc22f02cc765111f3e0c3a5240b73bb4ff`.
23 members/14 immutable references independently hash-checked by static script;
18 Python AST-only PASS, max242lines. V1/current212 inputs unchanged.
Никакого source import/compile/process/test/SSH/transport/model/RPC/HTTP запуска.
V2 SOURCE REVIEW202 pending; оба runtime gate закрыты до отдельного exact ROOT GO.

## V3 narrow source fix — signal argv path

202 full rereview подтвердил исправление прежних3findings, но обнаружил новый
P1: `phase` был local prepare(), а main ссылался на отсутствующее имя. По narrow
ROOT SOURCE REOPEN V2 сохранён полным exact checkpoint `versions/796d39db/`.
Единственное изменение Python source: argv теперь использует явный
`str(ops / 'phase.py')`. Other execution semantics/guards не менялись.

Static AST regression без выполнения fixture проверяет отсутствие Name phase
в main и точную AST форму argv[3] = str(ops / 'phase.py'); private
static-regression.json сохраняет эти факты и source hash. AST-only18 PASS/max242.
V3 manifest24members SHA:
`481422936569d02c09b809d8130c741a3d08aeef6d5926a2b06be5019d345105`.
Pins SHA:
`b670c466ca8f4cb46ee90e0f39c3f664ec6401a14cb3dccb48c311b5729202ec`.
Runtime/helper/SSH/transport import/model/process/tests NOT-RUN; final delta
SOURCE review202 и отдельный ROOT GO обязательны.

## Actual V3 ONE Linux — FAIL, stop-on-failure

После FINAL SOURCE APPROVE202 и отдельного ROOT GO выполнен ровно один
reviewed launch_linux сценарий. Private local evidence:
`.local/task014/qa215/run-linux-1792782062779539000/`.
`results.json` SHA2354f6a1abf173ba35555f329c280a5c8afc62a041d1d650b11d599c3b3dde9c;
`safe.json` SHA49305fd0ad3322123a3506d5970be196d1b0a07c60eae1fabcabf6c5e7e83ec3.

SSH child separately known wait1/forcedfalse/both EOF+drained, captured stdout4017
bytes/stderr0. Wrapper outcomeUNKNOWN/manualtrue сохранён как записан. Remote
fixture stdout сообщает FAIL после первого positive case: reader exit1,
private reader stderr289bytes/no trace. Before/after fileset/content/UID/GID/mode
совпали; source hashes local+remote unchanged. Remote cleanup_errors[] и
manual_recovery_required=false. Keeper intent statusUNKNOWN в основном report;
positive keeper wait/drain факт нужно независимо прочитать в private cleanup
receipt, не домысливать из пустого cleanup_errors.

Fresh prefix `/var/lib/accord-qa215-1792782065538992012` сохранён. Remote results
SHAc719d67036092dc438f6e14a00412b88085015704f04881c0ed98111ff9abbbb.
Причина reader startup error пока UNKNOWN: raw stderr находится только в private
remote prefix и не выводился. Запрошен отдельный readonly exact-prefix evidence
collection GO. Ни real STATE, ни registry/services/updater/pointers не трогались.

По stop-on-FAIL дальнейшие4negative и local signal NOT-RUN. Исправлений, retry,
fallback/repair нет; operational212/native/production gates остаются закрыты.

## Readonly collection — SOURCE_READY, execution pending review

Отдельный ROOT GO разрешил read-only сбор existing exact fixture prefix. Для
bounded capture потребовался новый private collector helper, поэтому до его
исполнения выполнен source-only freeze и запрошен202 SOURCE review.
`.local/task014/qa215/collection-v1/manifest.json` SHA
414fae990c53bb17bc250ce7b43c095b42acaff0e93a344bc2cb2f46f8579f5e;
pins SHAd7b27d81c5e7620fef5c8b69f8ebe8884a71a95b9a64828a8efd0ba2250b4118.
3members/27references/2AST-only. Нового execution пока нет.

Remote script делает только fixed-path ordinary/no-follow bounded reads уже
существующих reports (включая фактические positive/process.json и keeper cleanup
receipts) и hashes pinned source files; SQLite/state/fixture/import remote source/
services не выполняются. Raw base64 остаётся в local private600 файлах. Semantic
stderr projection только после verified byte hash и только известные exception
classes/numeric <string> line, без message/path values. Reused approved215 capture/
Signals дают45+8 budget и always-attempt receipt. Исходный FAIL не меняется.

## Readonly collection actual — CAPTURED, original FAIL unchanged

После SOURCE APPROVE202414fae99 и отдельного exact ROOT GO выполнен ровно один
collector. Private folder:
`.local/task014/qa215/collection-v1/run-1792782956725789000/`.
Results SHA4ce7853863d87eb54ed79b03567d1e357424126dedfedaa619832d13bc9635e7;
evidence-index SHA636278d9932398cb7633f264de1d6bc036da9f2894f6a5e3c791ace4e039235d.
SSH knownwait0/forcedfalse/bothEOFdrained, private raw saved0600. Все25remote
source hashes exact. Fixture/DB/services/new tests не исполнялись.

Private raw/reports/positive/stderr.bin —289bytes,
SHAca0b685b807d1a112961551dc10ae9077c660a0049248c9e31c63e9827c829b5.
После byte-hash verification safe projection обнаружила только known class
PermissionError и pinned <string>line31. Это соответствует reader_trace.py31
`runpy.run_path`. Specific denied object пока UNCLASSIFIED; arbitrary stderr
message/pathvalues не выводились и не интерпретировались вне разрешённой schema.

Actual positive/process.json: reader knownwait1/forcedfalse/EOFdrained/manualfalse.
Actual keeper-cleanup-0/process.json: keeper knownwait0/EOFdrained/manualfalse;
его stdout проверен exact JSON closed3, все три SQLite handles закрыты.
Основной исторический report intentUNKNOWN не переписывается; новый collection
добавляет независимые close/wait facts. Original Linux FAIL остаётся FAIL,
4negative/local signal NOT-RUN, retry/repair отсутствуют. Evidence отправлен202.

## V4 source correction after complete independent batch

По ROOT SOURCE REOPEN после202 fullbatch (одна P1) сохранены exact V3 source
checkpoint `versions/48142293/`, полный collector+actual collection evidence
checkpoint `versions/collector-414fae99/`, historical file hashes27entries.
Все старые evidence bytes unchanged; original FAIL не меняется.

Исправлена capture keyword collision: control parameter теперь strict bool
`new_session`, а Unix `group` остаётся в kwargs и передаётся Popen без изменений
вместе с user/extra_groups. Единственный owned-session caller использует
new_session=True. Pure Popen-stub regression source проверяет forwarding
user999/group988/groups[] при default false и explicit true, а также отказ до
Popen на non-boolean control. Stub никогда не создаёт OS child/native CLI.
Regression sources написаны, но по source-only GO НЕ запускались.

Prospective positive-only SHM clarification root/202 реализована отдельно:
metadata snapshot теперь включает dev/ino/size вместе с UID/GID/mode/hash.
Positive допускает только hash отличия exact трёх existing fixture SHM при
неизменных остальных metadata/file-set; все mainDB/WAL/other bytes strict.
NEGATIVES остаются all-byte strict. before/after/differences/predicates сохраняются
до assertions; причины observed differences не приписываются. Pure map regression
source покрывает scope/identity/main/WAL/negative boundaries, без запуска.
212 original reader/guards и State-only substitution не менялись.

V4 manifest SHA95d13f553c2e3caf1580b2df8b2e60db3d9e1c39e466240f1e48c457e6cdef20;
pins SHAc6be926df4f9c9b5bdbee3d2434342a6ec40ad55c1d6b46ed748346cb14f032d.
27members/21AST-only/max246; static AST дополнительно доказывает отсутствие formal
capture group, наличие new_session и literal caller new_session=True. Старые
27history hashes и V3 full source independently static-verified unchanged.
Ни tests/helper/fixture/SSH/transport/stub/native/process execution для V4 нет.
Полный SOURCE REVIEW202 pending; actual execution требует отдельного ROOT GO.

## Actual V4 — Linux PASS5 + local signal PASS, evidence review pending

После FINAL SOURCE APPROVE20295d13f55, независимых6pure unit PASS и отдельного
root-go-v4 ACTUAL GO выполнен ровно один новый Linux fixture. Exact manifest/pins
проверены до helper import. Старый prefix и V3 FAIL не менялись.

Local Linux results:
`.local/task014/qa215/run-linux-1792784543316931000/results.json`,
SHA0cdc3ea5fa4ca1f3c17244911f274d6b01bec95f8037c58f4acb4caf17a1e871.
Fresh remote prefix `/var/lib/accord-qa215-1792784545081357289`;
remote report SHAd685f9d994ed2da7508d57a42ff62e9458d36376b9f658e554706641f8a2247c.

PASS5: positive actual serviceUID999/actualGID reader делает4connects и видит
busy1 из committed WAL после mainbusy0 checkpoint. Различился только contenthash
board.db-shm; file-set/dev/ino/UID/GID/mode/size и main/WAL/прочие bytes unchanged.
Причина hash differences не утверждается. Все4negative PASS с zero connect и
exact original guard origins42/43/32/33; их before/after maps полностью равны.
Все5keeper knownexit0/EOFdrained/по3closed SQLite handles; remote manualfalse,
cleanup_errors[], source unchanged. Local SSH known0/forcedfalse/EOFdrained.
Отдельные reader process receipts существуют в remote reports; автоматически
не собирались/не объявляются независимо прочитанными, общий actual report+trace
сохранены в private local stdout capture.

После actual Linux PASS, как разрешено conditional GO, выполнен ровно один local
SIGTERM stub case. Results:
`.local/task014/qa215/run-signal-1792784566781773000/results.json`,
SHAfa26bbe4ff1900446c9ef6b7b53b0d0898e020c04dcbbdda81e4d82afe2a9c7c.
PASS: fsynced initialUNKNOWN предшествует dispatch, exact owned phase послеSIGTERM
knownexit1/EOFdrained, finalUNKNOWN/manual_journal_inspection сохраняется;
local owned group absent, sources unchanged. remote_exit_proven=false и
no_remote_stop_claim=true, никакой SSH/remote stop этим case не выполнялся.

Combined safe facts SHA2ec8812ad5b4255079576a908c668442effc1062804f36ac7da53d28486dc8c4
в local signal folder/combined-safe.json. Current27source members afterhash exact;
никаких retries/repairs/source edits или production/operational212 действий.
Все evidence сохранены, отправлены root/202 для independent evidence review.
Это focused215 PASS observations, не закрытие native/pipeline/UI/production.

## Readonly collector V2 — SOURCE_READY for new PASS prefix receipts

По отдельному root narrow SOURCE PREP подготовлен private collection-v2 на базе
approved414fae99, oldcollector source/evidence сохранены exact. Единственный
newprefix `/var/lib/accord-qa215-1792784545081357289`; fixed five-case reports,
reader process receipts, keeper-close receipts и pinned source hashes. No DB,
fixture execution/import reader/services/operational212. Runtime NOT-RUN до
SOURCE review202 и отдельного root GO.
Manifest SHA cc649f88ac6788fc28d6725f81dc4d03de193e113cc4e09703e6729e49562b44;
pins SHAb8c9d3a7db145bc883acd5e2aa932c8f8c0ac92a5deb9b23d0d48e7b10ff7da8.
3members/30refs/2AST-only. Передано202; H1 source correction продолжается отдельно.

## Actual readonly collector V2 — CAPTURED5 reader lifecycle receipts

После SOURCE APPROVE202cc649f88 и отдельного exact ROOT GO выполнен ровно один
read-only collector V2; не fixture/test повтор. Private folder:
`.local/task014/qa215/collection-v2/run-1792784969249808000/`.
Results SHAfc498e754e2bc142c210b880f5b97573c0750607e01cb765acf50af72af21155;
index SHAc280dac084ddf043838f917f32e2626db4fda4b8e364bae6edffa2eb9edd2284.
51fixed raw files saved private0600,28source hashes exact, SSH known0/nonforced/
EOFdrained. Все5 reader process receipts отдельно прочитаны: exit0/1/1/1/1,
known wait/forcedfalse/bothEOFdrained/manualfalse/stderr0. Все5 keeper-close
receipts known0/drained/manualfalse и exact stdout closed3.
Positive reader receipt SHAead3b5cec942139261071ecef34f005314aec926b967f002b38108e1c86e6841;
negative receipts each SHA0a9e08b41002df5afc10d9812503d8e8d4d412f67a19b327c945258b3b41ac1e;
keeper receipts each SHAac5e57f3a38015705a2af30d318286f40585903bd9e4831402b4a7bae4a32ae4.
Это добавляет actual independently-readable ownership proofs к PASS5, не вывод
из родительской строки success. No DB/source execution/services/retry. Raw paths
и hashes отправлены202/root для evidence closure. Operational212 не запускался.
