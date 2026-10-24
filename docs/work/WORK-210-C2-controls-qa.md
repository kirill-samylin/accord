# WORK210 C2 — pause/proven stop/clarify/exact native resume QA source

23.10.2026. QA author gui_connection_qa; root координатор; reviewer202 независим.
**SOURCE ONLY / SOURCE REVIEW PENDING / EXECUTION NOT-RUN.** Это C2 по approved
PLAN210870523cf…, отдельный от WORK216 OS-Cv2. Исторический source checkpoint
сохранён в `.local/task014/pipeline210/c2-source-checkpoint/` (manifest49332af8).
После F6V2 CONDITIONAL SOURCE APPROVE root дал SOURCE GO; после CONTRACT217
APPROVE202 отдельно разрешил C observer. F6V2/f2d2 и203V3/49df immutable.

## Новые источники и gates

Private `.local/task014/pipeline210/c2-v1/`. Unconditional EXTENSION_PENDING до
чтения input, без env bypass/issuer. Historical8c candidate не заменён на текущий
и не разрешает runtime. Дополнительный observer status NOT_BUILT_SOURCE_ONLY
блокирует setup. Реальный compile/link/ABI probe helper НЕ выполнен; нет binary.
После SOURCE APPROVE202 требуется отдельный root BUILD GO с exact source/SDK/
compiler pins; actual build receipt/path/SHA вводятся только в новом reviewed
freeze. Затем actual candidate/native prerequisites+source review и отдельный
exact root runtime GO. Harness ничего не компилирует и не открывает свои gates.

## Actual C2, который будущий запуск обязан доказать

Один analyst task/step, тот же managed run/native UUID, максимум2model turns.
Policy tasks1/native2/fix0/continuation1, step240/workflow600/grant900; scope240/
600/2, consent≤30min. Setup180+workflow600+cleanup60=outer840sec. Четыре immutable
профиля создаёт обычный UI; реально должен запускаться только analyst. Native
MCP allowlist describe/next/get/workflow_get/workflow_question; read-only, без
Git metadata/tester artifact opt-ins, plan/done/check/tool-shell permissions.

1. Early worker.child Popen anchor PID/birth/PPID/UID фиксируется до требования/
   первой модели. UI intake + обычный stdio MCP replay дают ровно один workflow.
2. Реальные describe/next/native UUID/live lease/session gen1. Ограниченная
   read-only инспекция до8cycles даёт возможность UI pause; ранний exit/automatic
   continuation — coverage FAIL, не human-resume. Дополнительная модель не чинитrace.
3. UI pause/current CSRF+action/version. Immediate REST resume должен увидеть
   pausing/stopping и exact409 CONFLICT «Сначала подтвердите остановку».
   Stale-version409 не засчитывается. Counters/generation/command history не
   должны получить resume до positive stop.
4. Ожидание server stopped + actual stop command/dispatch/native UUID/gen1 и
   native_exited/exit_durable/managed_ack/exit_ack/lock_released, run_exit clean
   seedSHA/manifest. Gen1 Attempt закрыта, lease/session освобождены. Full private
   evidence snapshot и hash сохраняются до перезаписи journal поколением2.
5. UI clarify добавляет unique canary, UI resume того же policy/run/UUID — epoch3,
   managed gen2/attempt2/continuation1. Native реально делает fresh describe/next,
   новая board session/lease generation2 и workflow_question с доставленным
   canary. Gen2 fresh claim доказывает actual Attempt/child operation даже если
   live GET-окно уже закончилось. Parent никогда не пишет question/done за native.
6. Реальный gen1 child credential берётся только из собственного pinned worker
   attempt1 token path, mode/UID/digest match registered journal. GET своей task
   до pause должен дать200; после resume тот же token/session —401 UNAUTHORIZED.
   Никакой fake token/owner substitution/generic error PASS. Секреты private0600.
7. Самостоятельный217 observer обязан доказать actual argv vector resume с
   первоначальным UUID; одного provider event/command receipt недостаточно.
8. Question epoch4 вызывает второй actual stop; те же exit/ACK/lock/SHA gates.
   Gen1 original proof остаётся byte/hash/tuple-identical. UI human cancel затем
   epoch5/cancelled/reserved0, две закрытые Attempts, cancelled human question,
   no lease/active run/pending. Ровно4native commands start1/stop1/resume2/stop2,
   no third model. Итог control-only, не completed implementation pipeline.

## Observer217 — primary SDK definitions, source NOT-COMPILED

Три C файла и два header; все<300lines. `observer-abi.json` pins resolved installed
SDK: libproc/sys/proc_info/sysctl/CommonCrypto и используемые ABI headers/
SDKSettings, actual compiler file hash, Darwin release/machine/SystemVersion hash.
Нет ctypes, guessed offsets/struct sizes/constants. `KERN_PROCARGS2` объявлен
Apple под `__APPLE_API_UNSTABLE`, это не stable-public-API обещание. Primary Apple
links и SDK provenance — в private README/ABI descriptor; compiler version/help
и C syntax/link/runtime не запускались.

Observer фиксирует тот же живой Popen child до/после; numeric BFS только его
потомков (без global inventory/argv), same UID+real UID, stable PID/birth/PPID
ancestor chain, proc_pidpath и ordinary executable file identity/mode/SHA before/
after. PID только readonly query, без adoption/pgrep/child signals/permission changes.
Limits128descendants/childlist<256/polls32/budgetchecks20000, общий20s remaining
включает wrapper pins; self setitimer+monotonic gates ограничивают OS queries.
Own core dump disabled; helper получает только expected tuple, не чужие secrets.

Buffer≤1MiB, argc1..1024, strict NUL/UTF8, только ровноargc argv после execpath/
padding. Environment tail никогда не parse/copy/save/log. Volatile raw wipe на
любом return/timeout. Exact `exec,resume,--json,originalUUID,-` в реальных argv
границах; не flattened ps. Nonmatching valid vector можно ограниченно ждать;
unsupported/denied/malformed/drift/overflow сразу NOT_OBSERVED/coverageFAIL, без
fallback. Сохраняются лишь enum/booleans/counts и private early numeric anchor.
Самостоятельный helper runtime запрещён; future scenario запускает pinned copy
через настоящий candidate supervisor/nonce/lifeline/owned terminal proof.

## Evidence, cleanup и границы

Inherited203V3 stream/log/HTTP byte caps, shared abort/end, handled UI response
observer, absolute browser close budget. Cleanup45s own mutable actions +15s
sealed owned closure/main AND observer promise join; no late business mutation.
Source/Git complete inventory/bytes/modes/config/hooks/remotes/seed HEAD и real
native configs before/after строго неизменны. Raw observations сохраняются до
финальных assertions; ACK reconciliation ожидается, queued не становится stopped.
Unknown/manual/failed native/extra invocation/source drift остаются FAIL, fixture
и diagnostics retained. Success отзывает только свои policy/host/profiles и реально
существующий legacy token list/revoke/list, без catalog credential fiction.

Статические проверки не означают actual runtime. Windows/Claude/physical second
PC/production/GUI/H1/B2/Q7 — NOT-RUN. Ни normal prerequisite, ни permissions/OS
control этого кейса не выведены из чужого PASS либо historical candidate descriptor.

## Freeze C2-V1 → независимый202

Full manifest `.local/task014/pipeline210/c2-v1/source-pins.json` SHA
`b27849f9f6f344e5bf6d7092a55627c26fa1fd4717fb533f7bdab39758bea434`;
41sources. Static PASS23CJS syntax+3Python AST+local require targets+14SDK/header
hashes; max159executable lines. C3sources/H2headers **NOT-COMPILED**, compiler
version/help/link/ABI/helper/control/native/model/UI/MCP/runtime **NOT-RUN**.
Original203V3 31pins, F6V1 35pins, F6V2 36pins, PLAN210 и CONTRACT217 hashes
совпали. Private sibling c2-v1-source-diff.patch/static.private.json сохранены.
**SOURCE READY / SOURCE REVIEW PENDING / runtime gate CLOSED.** Ни одного
самостоятельного helper запуска. Нужен202 full review; BUILD и runtime GO отдельны.

## C2-V2 и V3 source-only corrections

V1/b278 полный freeze сохранён. V2 narrow P2 errno fix: manifest
`748cb2682da3ae19e6189c4c4bd49e66cca6c5832ef58c23fe52b698b86a45d0`,
41pins: Apple proc_listchildpids zero-on-error не признаётся empty-success.
Перед запросом errno=0, captured errno немедленно NOT_OBSERVED без следующего poll.
V2 static23CJS/3AST; compiler/helper/runtime NOT-RUN.

Отдельный V3 — `.local/task014/pipeline210/c2-v3/`, manifest
`0ea75f3d538059f1dd373121a1b46c72fe95baeda3c8ff9dda8bb7bf88c115cf`,
42pins. Source authority finding от root/B2: API использует actual owner browser
human session, не MCP coordinator. Отзыв policy через nativeREST даже с human
требует localHostProof; root согласовал supported browser CSRF/action nonce revoke
и managed:disable. Новый browser-cleanup читает fresh scoped pages, проверяет
тот же actual human session и выполняет действия с текущими versions, потом
проверяет revoked/disabled через scoped REST. Используются только собственные
уже существующие browser cookies в cleanup permit/abort/deadline; новые proof
или secrets не экспортируются. MCP session остаётся scripted intake replay.
No product authorization changes. Static24CJS+3AST/local requires PASS; max159lines.
V1/V2 immutable; V3 SOURCE READY / REVIEW PENDING / compiler/runtime NOT-RUN.
Historical candidate, observer NOT_BUILT и root execution gates закрыты; trust219
будет отдельной адаптацией, текущий C2 strict config observer не переоценивается.

## C2-V4 warning/source follow-up — BUILD readiness only

Root выполнил V3 BUILDONLY после202approve; compilerexit0, общий FAIL из-за3
signedness warnings +common alignment reduction32768→16384. Actual receipt
074cb06dccae3300c05c631f632d1c0c0315c903537df7a002677946d907ab68 сохранён,
helper ни разу не выполнялся. Не переоцениваем FAIL по наличию binary.

По ROOT SOURCE REOPEN отдельный `.local/task014/pipeline210/c2-v4/`, manifest
`28d13228b22cff5f1e9a97070de9cbd3c1c989882f6fa385f405afcc07eaf410`,
44pins. V3 все42pins unchanged. C changes только положительные/range-checked
PID conversions к uint32_t, положительная length к size_t и C11 _Alignas(16)
для raw byte buffer; argc по-прежнему memcpy, volatile wipe и environment exclusion
не менялись. SDK14headers/observer-ABI exact unchanged. Никаких дополнительных
flags/warning suppression/new OS query/fallback. C compile/helper/native NOT-RUN.

Общий strict predicate exact SOURCE APPROVED203V6 SHA124da0d6… плюс Owned/MCP/
42pure cases copied byte-exact (4sources); UI close block identical strict approved
implementation. Cleanup own legacy commands требуют constructionKnown, actual
positive UI/MCP closure tuples и strict allOwned native0/success/proven/!forced/
drained. Wrapper0 с native failure не PASS. Pure cases тут не запускались;
203V6 independent20242PASS — только provenance одинакового pure source.

Static26CJS syntax+3AST/local requires PASS/max162 executable lines. Sibling
c2-v4-source-diff.patch/static.private.json retained. SOURCE READY202, нужен exact
source approve и отдельный BUILD GO. Runtime CLOSED: historical8c/unconditional
hardstop/observerNOTBUILT оставлены явно. CanonicalWORK217 получил root append
actual buildFAIL и не совпадает со старым lineage SHA; runtime не сможет пройти
этот historical gate. Currenta32/shared219/foundations/one-useGO +accepted actual
buildreceipt и frozen contract snapshot нужны отдельной последующей source
адаптации, без silent rebind. Actual203V6/shared checker/libraries/configs во время
root's ONE four-role run не менялись. GUI/Test/production/ports/agents untouched.

## C2-V5 runtime-adapter CHECKPOINT — paused for203 UI diagnosis

ROOT SOURCE GO после202 C2V4 build evidence APPROVE95079a8c…/binary96564be5….
Separate `.local/task014/pipeline210/c2-v5/` checkpoint manifest49files SHA
`c7a838e4ad72f08dde432c01897b2eb46a51e2b159bebee91d0aab094d71d189`.
Static30CJS+3AST/local requires PASS/max162lines; V4 все44pins unchanged.
Observer6built source files/14SDKheaders/ABI/binary/buildreceipt exact matched.
Accepted receipt fields resultPASS/build_only/helper_executedfalse/abi_sha256/
observer_sources consumed directly, не fake BUILT wrapper. Currenta32/Next1256/
library179/tool/foundation source hash verification PASS; runtime NOT-RUN.

Copied exact6reviewed203V6 shared consumer/foundations/owned/MCP/purecases files,
plus exact3reviewed203V7 fixture files AFTER202 pureFS PASS6 confirmed. Shared219
one external14795546 manifest; whole-cycle baseline/prebound own checkout;
preseal comparison requires strictUI/MCP/allOwned/main AND observer phase settled.
New private rootGO/source202 names210-c2-v5 (no author-created receipt/GO), exact
source/shared/candidate/scenario/input/expiry/atomicnonce plus acceptedobserverhash/
buildreceipt. Source code exists; execution authorization ABSENT.

Original approvedWORK217 snapshot0c0d8ccb copied byteexact into own contracts/
from readonlyB2checkpoint. Source lineage separately records changed canonical
WORK217 hash as documentary only, without silent rebind of original authority.
Old canonical append no longer invalidates immutable approved snapshot gate.

Root then ordered pause because actual203V7 FAIL ui_team before native and Next
stop known130 requires separate reviewed lifecycle disposition. This checkpoint
still inherits strict generic0 (correctly rejects130), no Next exception/source
instrumentation assumed. NOT source/runtime ready for actualC2 until that shared
reviewed fix and UI diagnosis resolve. Fullfreeze/diff/static retained; don'trun/
compile/helper/settings/native/ports/models. 203V7 originalFAIL/source/GO claim
untouched. Root prioritizes separate203 instrumentation; no automatic retry.

### C2V6 — reviewed203V10 adapter source freeze

Root source assignment only, runtime gate remains CLOSED. Immutable C2V5c7a838e4 and203V10 source26534f4a verified. New `.local/task014/pipeline210/c2-v6/source-pins.json` SHA `9d836bd1c1d62a58d9ad18b05f078a150b62706e7bc585bc0622ee682d548c06`,58pins. Direct inheritance: strict generic native0, scoped exact immediate-label controls, sequential bounded browser context/server close, separate exact known Next130 closure/intent/readiness, finite UI diagnostics/read joins, source fixture exact0644 modes under0700. API/business product unchanged.

Declared C2-only integration: own QA210 name/machine namespace; existing control method gets exact immediate-label textarea and finite per-control stages. History cap256 permits four controls/page reopens; diagnostics root regex only pipeline210-c2. Cleanup keeps observer phase preseal guard and final bounded join plus inherited diagnostic joins, and coordinator supplies private failure reference. Context pins/rechecks exact V10 sources and Next launch bindings before/after. Existing scenario/probe/control-proof/observer sources byteexact C2V5: native2/continuation1/sameUUID/literalargv/newboardsession/leasegeneration/early409/credential200→401/positiveACKs/clarification/cancel unchanged.

Accepted observer binary96564be5 and buildreceipt95079a8c, six observer sources/14SDK headers/ABI verified read-only. Static39CJS/3AST/max169 PASS; six sourcefixture fullinventory bytes+actual0644 exact descriptor BEFORE freeze. Seven tools/179libraries/1256Next files/three direct actual foundations/shared219 pins matched read-only. No helper/compiler/native/model/browser/server/HTTP/settings/CLI/pure-test execution by author. Historical freeze files untouched. Independent202 full review requested; prospective gates210-c2-v6-source202.json/210-c2-v6.json require new exact root GO and atomic one-use claim. No runtime, Windows/Claude/second-PC or business workflow PASS inferred.

### C2V7 — exact approved V11 policy expiry wire helper

New source-only adapter preserves immutable V6/9d836bd1 and V11/3bed2574. Exact V11 policy-settings helper/tests copied; private original policy UI receipt saved before parse/assert. Only equivalent strict valid UTC-minute expiry forms accepted, all other settings remain deepEqual. V6 literal argv/sameUUID/early409/credential200→401/native2/control/cap256/observerbuild/Next130/diagnostic joins unchanged. New auth names210-c2-v7 require full review/new root GO; V6 full review was not completed.

Frozen `.local/task014/pipeline210/c2-v7/source-pins.json` SHA `a233d336ce8f4f715abba6424868577481f18ad50887918e4ff5ccefb82d687c`,60pins. Static41 CJS/3AST/max171 PASS; six-file sourcefixture full SHA/actual0644 inventory exact beforefreeze. Tools/libraries/Next/direct foundations/shared219/observer binary/buildreceipt/six observer sources/14SDK pinned inputs matched readonly. No author runtime/model/CLI/helper/compiler/browser/HTTP/settings or pure-test execution. Full packet sent202, gate CLOSED.

### C2V8 — mandatory exact false outcome truncation evidence

202 full V7 review found one P2: truthiness accepted missing/nonboolean outcome_bytes_truncated. Root narrow source reopen: V8 common stopped proof (gen1 and gen2) now calls exact approved F6V6 requireCompleteOutcome. HelperSHA `acf2106849a6db1b80391bb8665ab603529617c7b7cfc6daf94cbdabe274354a`, pure-testSHA `4a8d6bc76e9908e465109c0aa900cebca37b29bf09f07dd80b80be67caf3ccbe`; requires mandatory literal false. No other control semantics changed; original V7 remains immutable.

New `.local/task014/pipeline210/c2-v8/source-pins.json` SHA `afa913fb9557122f067a613aafe9352b88654e25c2d7a2dc900584d2c4b67252`,62pins; static43 CJS/3AST PASS, six sourcefixture bytes+actual0644 inventory exact beforefreeze. New gate names210-c2-v8, own historical contract snapshot path updated. Independent202 narrow review requested; author executed no pure tests/runtime/model/browser/helper/compiler/CLI/HTTP/settings, GO remains root-only.

### C2V9 — exact approved V13 dual server/worker binding

New source-only adapter from immutable approved C2V8. Exact203V13 worker-proof/test copied, separate pins.worker be95/9755 with V3d98 provenance/receipt80bf. Server/Next/scriptedMCP/QA outer supervisor stay a32; worker identity/preview/enroll/run execute privately copied workerCandidate through old QA supervisor. Worker self-exec supervisor/bridge physically NEW executable via unchanged source. Context validates original/privateworker/source2+snapshots/inner3+snapshots/diff/contract/report before/after; report separately labels server/QAouter/worker/selfexec. New210-c2-v9 GO/input/source202 worker tuple bindings match reviewedV13, retaining observer tuple.

Scenario/probe/control-proof/observer/native2/sameUUID/literalargv/early409/200→401/mandatory false/finitecontrols/pre219 observer settlement+joins/UI/cleanup byteexact C2V8. Frozen `.local/task014/pipeline210/c2-v9/source-pins.json` SHA `3c53b88d74131dcdef62e1d53c731a0c045058430ec02454d28ec188fab12327`,64pins; static45CJS/3AST/max171 PASS. Six actual0644/SHA sourcefixture inventory exact beforefreeze. All context inputs/workerV3/observer six sources14SDK/build95079/binary96564 matched read-only. OldV8/V13 immutable. No author pure-test execution/helper/CLI/native/model/config/browser/HTTP/runtime/GO; independent202 review then separate exact ROOT GO required.
