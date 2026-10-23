# WORK169 — независимая готовность production и Mac GUI

23.10.2026. Reviewer; координатор root выполняет production операции и общую
TASK, author170 готовит Mac worker, отдельный tester172 выполняет приёмку.
Область записи reviewer — только эта карточка. Remote операции, пользовательские
credentials, GUI source и Tandem reviewer не изменяет.

Пользователь явно поручил обновить рабочий Accord и подключить Mac к Давинчи2.0.
Это разрешение нового этапа; прежняя synthetic read-only native приёмка сама по
себе не переносит свои grants на реальный checkout. Повторное разрешение на уже
порученное production обновление не требуется. Взаимный OS control остаётся
ограничен local owner enrollment/controller allowlist, текущим project ACL и
явным локальным workspace/provider/native tools набором.

## Source и поставка

Прочитаны AGENTS, действующие requirements/architecture/multi-project/TASK013,
managed contract/worker guide/native permissions, предыдущий WORK165 и исходники
capable updater, guard, launch, snapshot/migration, Next stage/package.
Product `cmd/internal/infra/web/go.mod/go.sum` diff798e689→273ccbb пуст.

Старый candidate-v6 Linux SHA `9f1b022831672497dd32214b5de99ef8b75a136b8bd371607e56d26999e98a24`
содержит vcs.modified=true; для production root правильно выбрал новую clean
сборку в отдельном worktree, сохранив прежние untracked файлы основного checkout.

Production source `273ccbb15c99dfac91c37f34d2926abf01ca866f`:

- Go Linux/amd64 SHA `5a1e1352895c20ec976feef20cc60328aa9ba19834834e75d3b1e5a46ca7dd60`,
  Go1.27.2, CGO0, trimpath, exact revision, vcs.modified=false.
- Go manifest SHA `f5a30113898267933a6d05c913e69c28487e313340890c40fa85e6c733454f43`;
  независимо проверены все53 файла.
- Next manifest SHA `b18e7fed8678ec48a9e98656facac67863103fcc1f30661c551dd0d384957a41`;
  независимо проверены все1256 файлов. BuildID `EJT-3udsZCJ4tJm9eDhiL`.
- Linux Node24.19.0 SHA `bc17c508ffeed0ec622934f9b7fa72f8e78da65350e63c3eceb56fa688aa5e12`;
  private backend `http://127.0.0.1:7892`, lock-pinned Linux sharp/libvips.

Private evidence: `.local/task013-live/deploy/release-proof.json`, transfer proofs,
stage-runtime-proof.json. Исходники build_release.py/transfer.py/activate_release.py/
collect_runtime.py/process_proof.py прочитаны: **SOURCE APPROVE**.
Новый root-owned release без замены active pointers; manifests проверяются до
активации; mode только stage/apply/resume/finalize текущего projects/upgrade.py.

## План переключения

**SOURCE/PLAN APPROVE**, production apply пока условен до actual Linux rehearsal.
Использовать только capable `projects/upgrade.py`, не initial operate.py,
не старый Next rollback и не одиночный binary switch.

Последовательность проверена: admin/maintenance locks → durable owned startup
hold → stop Go/Next и проверка освобождения7892/7893 → full format2 snapshot
registry/default/всех проектов → новый permanent schema5/managed_agents guard до
DDL → exact Go+Next pointers → offline additive migration всех board DB и
проверка прежних logical rows/immutable bytes → снятие только своего hold →
owned process argv/ELF SHA/readiness → независимые HTTPS/UI/native smoke →
finalize и новый полный backup с повторными process/config/helper проверками.

При ошибке — stop+inspection marker+blocked-roll-forward, те же проверенные
capable artifacts и явный resume. Старые binary/БД не накладывать. Production
tokens и данные в отчёты не выводить; текущий checkpoint update не закрывает
Windows/Claude и второй физический ПК.

Actual Linux stage: **PASS** по прочитанному сохранённому proof: hardened Next
process, MemoryPeak61976576, NRestarts0; helper hashes/effective runtime PASS;
порт7894 освобождён; Go/Next pointers остаются64ed870; recovery markers отсутствуют.
Это stage acceptance, не production apply и не live migration proof.

## Отдельный Linux QA

Private wrapper172 `linux-package-1792752990369561000/run_linux.py`:
**SOURCE APPROVE** до исполнения. Независимо сверены629/629 файлов manifest
с committed273 и historical64ed; source.tar SHA
`47a34272af24310f6690a433bff838fc83a8a0ded0cff07cd18f1e4f15e796b2`,
unsafe/extra/missing archive members0. Synthetic private root, immutable copies
old/new production release executables, fixed SHA; только loopback/test DB,
без production DB/unit/pointer mutation. Browser/native flags выключены.

Existing approved process/update suites не изменены. Wrapper сохраняет whitelist
summary, но его exit0 сам по себе не PASS: перед apply root/reviewer обязаны
проверить suite result/checks/pins/source unchanged/cleanup. Timeout или cleanup
failure оставляют FAIL. Sidecar честно отмечает systemd/Next simulation и старый
static Mac label; stage отдельно доказывает actual hardened Linux Next runtime.

План требует actual Linux updater PASS до apply; process suite проверяет также
Linux ownership/group/recovery behavior. Результаты и Mac170 prelaunch config
review пока **PENDING**. Production apply/finalize, live GUI native control и
actual Windows/второй физический ПК на момент этой записи **NOT-RUN**.

## Mac170 prelaunch

Private `~/.config/accord-gui/managed/worker.json` и user LaunchAgent template
прочитаны без вывода token/origin/project/principal values. **SOURCE/CONFIG
PRELAUNCH APPROVE**: отдельный binary273ccbb с clean VCS и SHA
`e3d4caf6ac13621702ba2ef84155b399d46fe14546b0969611fb7176050e2d3a`;
проверены hash/owner/private0600/0700 и пустой state. Старый GUI bridge сохранён.

Один canonical GUI checkout, Codex workspace-write и exact10 локальных tools:
describe/next/get/events/update/comment/note/transition/release/link. Создание
задач и ручной heartbeat не добавлены. Эти scoped права согласованы новым
поручением реальной работы через Accord; прежнее synthetic согласие отдельно.
QA smoke остаётся чтением GUI, без изменения файлов. Additional controller1
сопоставлен автором с actual membership; owner implicit. До enrollment/run на
обновлённом сервере повторить current project identity/controller проверку.

LaunchAgent явные user HOME/CODEX_HOME/PATH, абсолютные binary/config paths,
private logs, KeepAlive, throttle30, exit15; token в environment отсутствует.
Профиль выбрать с OpenAI provider и пустой model либо exact локально разрешённой
model. Source review не доказывает enrollment/launchd/actual native control.

Linux wrapper172 первый setup остановился SyntaxError до binary copies/tests:
historical harness FAIL, production не затронут. Узкая правка трёх nested
write_text newlines на chr(10) и fresh synthetic root suffix-r1 прочитаны:
**SOURCE RE-APPROVE**, archive и тесты неизменны. Требования actual PASS остаются.

## Production apply gate

**PREAPPLY APPROVE** передан координатору после actual Linux updater **PASS3**:
private `linux-package-1792752990369561000/update-summary.json`, returncode0,
source revision273ccbb, newGo5a1e135..., old actual Linux64ed SHA
`c4b25c45de5204dee55f6bb3e85de03be7e1073267d0304aa04c5c816f918b45`;
source_unchanged и binaries_unchanged true. Approved неизменённый suite проверяет
реальные Linux Go/SQLite/семь DB migration/snapshot/restore/auth-reset; systemd/
Next lifecycle в нём simulated, actual hardened Linux Next отдельно прошёл stage.
Summary прочитан самостоятельно; raw report запрошен tester для final evidence.

Обязательный preapply rehearsal выполнен. Linux process suite ещё выполняется
как дополнительная проверка ownership/recovery перед финальным closure; он не
меняет7892/7893 или production DB. Координатор может выполнять именно reviewed
capable apply с перечисленными выше locks/hold/full backup/preservation guards.
Это не finalize acceptance: независимые HTTPS/UI/native readonlyGUI smoke,
owned actual ELF/config proof и новый schema5 format2 backup остаются обязательны.

Полный `update-raw-results.json` позже независимо прочитан: SHA
`8c1ef3f21cc429b49d291cd47d9fb7d3dd0abe11ede979bc718981908f808fbe`;
все три конкретных preservation/guard/restore/auth-reset checks присутствуют,
source inventory совпадает с уже проверенным committed package. Отдельный
linux-provenance.json SHA `e20194cfbfa045cbf58b5c46d2b2185c04457f82e21df996a6ce8eae7d9247bd`
корректно переопределяет историческую static подпись «one synthetic Mac» на
actual Linux; это не native provider/Windows/physical2PC acceptance.

## Actual apply evidence

После root apply прочитаны operator output и
`deploy/process-pending-https-smoke-proof.json`: **ACTIVATION EVIDENCE APPROVE**.
Фаза pending-https-smoke, exact273 release; actual Go PID461652/ELF5a1e,
Next PID461676/Node ELFbc17/cwd release273/BuildID EJT. Все прежние capability
floors плюс managed_agents1/schema5; effective launch/guard/backup и helpers PASS.
Authority marker uid999/mode0600 и прежние bytes сохранены; recovery markers0.
Pre-snapshot manifest SHA `68861804eadc9ed6f0460ceaa4013cb2dc9fa7a124a2ee60b25fe89f6996b121`
проверен operator/proof до дальнейшей работы. Migration preservation confirms
**5 production DB**, default плюс1 published project, logical rows preserved;
это отдельный факт от synthetic7DB rehearsal. Finalize ещё не принят.

Mac170 live identity/enrollment/LaunchAgent/health proofs прочитаны:
**ENROLLMENT/SERVICE EVIDENCE APPROVE**. Live identity verified, owner и
additional controller сопоставлены; user UID501, LaunchAgent state running,
PID21649, runs1/never exited, stderr0. Active host/current lease, exact provider/
workspace/controller, journal0runs/pending0; GUI baseline3316files unchanged.
Это доказательство подключённого фонового worker, не native GUI acceptance.

Полный Linux `process-raw-results.json` независимо прочитан: **EVIDENCE APPROVE**,
SHA `88f8bf9e3ae09a170038d6c4a8590b7049585eedceb101d1d992fa4f5ddc5e39`;
PASS16, actual platform linux/Node24.19.0, exact273/Go5a1e, source inventory533
совпадает с committed package, source/binary unchanged, cleanup failure нет.
Проверены реально исполненные cases: owned PGID stop и forged proof rejection,
detached/unrelated untouched, same-worker concurrent deadman и offline recovery,
SIGKILL lifeline/retained intent, exact stdio snapshot/terminal handoff, controller
revoke, cross-state checkout lock, reserved stop quotas. Provider deterministic,
native Linux/Windows или второй физический ПК этим не заявляются.

## Live GUI tester source

`.local/task013-live/qa/live_gui.cjs` **SOURCE APPROVE**, independent Node syntax
PASS, frozen SHA `bce45f9595b2a93f7a5c6fd00b18b1a27fc4cd31d749609e2c758adc4d75cabe`.
Три readonly turn: owner UI start и настоящий README hash/claim/handoff;
owner UI resume с длинным анализом без sleep/busy-wait/записи; colleague UI
queued clarification и applied proven stop; owner UI exact-conversation resume,
новая board session/больший fence, semantic clarification/hash receipt/handoff.
QA профиль/эпик/задача временные, запрещены посторонние claims и GUI edits.

Review исправил cleanup failures, которые прежде могли оставить ложный PASS,
и надёжное сохранение final report после browser.close failure. Exact Mac binary
e3d4 upfront, binary/config/adapter before-after hashes, GUI baseline3316,
фактический colleague machine, private-only API reports, user LaunchAgent
exact top-level state, archive/cancel QA-only cleanup теперь проверяются.
Whitespace/case форматирование SHA receipt не ослабляет сравнение expected hash.
Execution только отдельным tester после root GO; SOURCE APPROVE не runtime PASS.

Первый live adapter обнаружил **harness FAIL**: selector использовал
`run.profile` вместо authoritative `run.profile_snapshot`. Reviewer пропустил
это поле в первоначальном source review. Коррекция не требует product change.
Повторно сверены ManagedRun/ManagedCommand/ProfileSnapshot/Task.Attempt DTO:
прочие используемые JSON fields верны. Уже отправленный UI start повторять
нельзя; до нового теста tester обязан найти только собственный QA run по
host/task/profile snapshot, доказать terminal state или отправить stop и
подтвердить его, сохранить первую неудачу и завершить QA-only cleanup.
Selector patch и следующий runtime результат пока PENDING.

Selector-only patch `profile_snapshot.id` плюс exact host/task/created_by
прочитан и **SOURCE RE-APPROVE**, SHA
`3c1e38a585732c94bde9f94a9fb28a952df5b4a198921e44a504770c18150c8b`.
Первая execution `.local/task013-live/qa/live-gui-1792753978491/results.json`
остаётся FAIL; executed adapter copy bce45 сохранён. Cleanup cancelled QA cards,
archived QA profile, GUI baseline/config/binary/adapter unchanged, user worker
running. Однако это не только selector failure: separate diagnostic run shows
accepted start → failed `cannot prepare isolated native MCP connection`, без
provider session или task attempt. Actual GUI native control пока **FAIL**.

До retry author170 исследует только config-only actual launchd environment и
preparation paths. Гипотеза reviewer о смешанном stderr опровергнута current
process_unix.go: stderr io.Discard, stdout writer. Причина не установлена;
deadline/ownership/config verification нельзя ослаблять ради PASS. Новый native
запуск ждёт root cause, source/config re-review и отдельный root GO.

## Diagnostic-only продолжение после публикации

По новому приоритету пользователя сначала выполнена публикация current code+
всей history (WORK173–175), затем native preparation diagnosis возобновлена в
отдельной codex/native-preflight от published main26d833f. Ни прежний synthetic
PASS, ни публикация не закрывают actual GUI preparation FAIL.

Предложение author176 **DIAGNOSTIC CONTRACT APPROVE**, исходники ещё не приняты:
unexported failure wrapper с fixed prep stage и exact wrapped cause; Error()
generic, Unwrap сохраняет errors.Is/As ownership. Только private LocalDiagnostic
может содержать fixed prep code или композицию fixed-stage + уже allowlisted
ownership code с bounded errno/length. Unknown values заменяются fixed fallback.
Нельзя сериализовать cause/error strings, connector names, args, stdout/stderr,
env, paths или credentials. Public Receipt/UI сохраняют прежнюю generic error.

ReadNative5s/1MiB/owned process Wait/census/PGID/isolation и все launch authority
проверки не меняются. Injectable process starter — только unexported unit hook,
default прежний startProcess, без config/CLI/RPC surface. Tests обязаны проверить
no native spawn, exact failure classification, private code/public generic error,
wrapped ownership сохранён и секреты не попадают в receipt. Диагностика не
считается исправлением причины до actual evidence.

Actual failing run accepted→failed около5.19s, в отличие от direct config-only
33/31ms; это основание проверить deadline/Wait boundary, не доказательство причины.
QA177 предложен отдельно reviewed real-Go config-only probe под exact launchd
context, поскольку Python subprocess уже PASS и не использует нашу ownership
обвязку. До source/test approve нет нового provider/GUI retry.

## Diagnostic source176 approval

**SOURCE + AUTHOR UNIT/INTEGRATION TESTS APPROVE** current восьми Go-файлов и
WORK176 после независимого чтения. Closed stage/composite128byte encoding,
Error generic/Unwrap exact cause, errors.Join Wait→Stop→read priority,
atomic timeout classification и private journal/public receipt boundary
соответствуют контракту. Injection seam unexported, production starter прежний.
5s,1MiB, process/PGID/Wait/isolation/authority guards сохранены; отсутствуют
native/service/config/UI изменения или автоматический retry.

Независимый полный `go test -race -count=1 ./internal/worker ./cmd/agent-board`
**PASS**,10.334s/2.815s; diff-check PASS. Новые файлы71/154/108строк. Unit cases
только fake I/O/process:5s timeout, exact1MiB boundary, ownership-wrapped failures,
invalid stages, immutable bridgefile и deadline/pins/controller/no-spawn gates;
осмысленный private persistence/public generic receipt проверен на engine path.

Ограничение диагностики: fatal/closed/state early return после concurrent state
change может вернуть classified error без сохранения нового LocalDiagnostic.
Это не обещание universal crash logging. Обычный accepted preflight failure,
относящийся к текущему случаю, сохраняется через failed receipt + journal save.
Отдельный tester QA177 теперь может подготовить actual config-only process
probe, также с независимым source review. Root cause пока **НЕ УСТАНОВЛЕНА**;
source approval не является разрешением service restart/нового GUI provider test.

## QA177 config-only source gate

**SOURCE APPROVE** private runner/run_probe.py SHA
`79576e21715237e029b095dd3f5e6ada69d6f9997ca572c6238eaa8dec8023ab`,
Go helper qa177_process_test.go SHA
`a9ac82ed703064d4cd46ba58b802a64c86295b4dc8ff08efd56eb6c21e846cdd`,
compiled test binary SHA
`4262a15f11e134950f939c99586fb4eccf53b98f7f7586ec8b0cd08b2c78f71c`.
Независимо проверены363 owning source files в private copy против manifest и
current176: drift0; Python AST PASS. Никакого production CLI entrypoint нет.

По3 config-only samples в interactive и temporary user LaunchAgent контекстах:
actual readNative/verifyCodexMCP используют прежний startProcess. Только
`mcp list --json` и generated inert /usr/bin/false binding; нет exec/resume,
MCP tools/call, модели, board/GUI/config/service mutation. Same configured
EnvironmentVariables/WorkingDirectory, но собственные test binary/label и
inert binding отличаются от production; это явно сохранено как limitation.

Review потребовал исправить report loss при cleanup failures и ложный PASS
после nonzero helper exit. Теперь exact top-level exit0 обоих contexts,
отсутствие manual/timeout flags, оба JSON PASS и все cleanup/pins/baseline/
worker checks обязательны. Сверяется весь source manifest до/после. Temporary
label bootout не выдаётся за доказательство завершения отдельных CLI PGID;
неоднозначность оставляет FAIL/manual inspection, generic pkill запрещён.

Отдельный QA может исполнить этот ограниченный probe после root GO. Фактический
результат ещё PENDING; installed worker273 и прежний GUI native FAIL неизменны.

## QA177 результат и следующий локальный шаг

**CONFIG-ONLY EVIDENCE APPROVE** после независимого чтения comparison SHA
`98ae562b9460851970ab157d5490c5b84f0473bf3fe5010ddcb9e59f02a598ba`:
12phases/6samples PASS, interactive31–56ms/launchd31–45ms, оба process exit0,
без uncertainty/timeout/cleanup flags. Raw reports совпадают с comparison;
все7pins rehashed,363 owning source files совпали copy/current/manifest.
Reviewer read-only launchctl подтвердил отсутствие собственного temporary
label и running persistent worker. GUI baseline/pins unchanged; моделей/MCP
toolcalls0. Первоначальный runtime failure **НЕ ВОСПРОИЗВЕДЁН**, причина неизвестна.

**LOCAL INSTALL/RETRY PLAN APPROVE** для reviewed176: clean exact VCS build,
private backup прежнего binary/config/plist/state; no queued/active/unknown runs
и pending receipts; owned LaunchAgent bootout/worker exit/owned-child check,
atomic binary-only replacement с сохранением0700/owner, bootstrap same user
instance. Config/host proof/consent/provider flags/server/schema/grants прежние;
initial new-instance lease grace до35s допустим. Backup state не копировать
поверх новых receipts/authority после теста, broad process-name kill запрещён.

Existing GUI retry требует узкой test-only замены old hardcoded e3d4/273 pins
на exact новый binary SHA/revision и нового source review до исполнения.
Один ограниченный readonly GUI сценарий после отдельного root GO; прежние FAIL
сохраняются, при новом отказе читать private diagnostic, без автоматического
повторного запуска. Такой PASS диагностического binary не докажет исправление
неустановленной первоначальной причины. Windows/physical2PC ограничения прежние.


## Локальная диагностическая замена176: operations source review

**SOURCE APPROVE** private install script SHA
`42a478595389a91f0740be3ae41b012daabb3b64fde1c15fb134259a1cbcfd44`: AST/hash
независимо проверены. Начальное ревью потребовало исключить bootstrap, если
после bootout прежний owned PID всё ещё существует, и сохранять FAIL proof даже
при исключении bootstrap. Оба fail paths исправлены; state/pid launchctl читаются
только из top-level. Только binary меняется атомарно на clean176 release
`62e354525b83a2b671ab37ede938ffe7e838e24b`, SHA
`10e845ff702e2b1f9d9bcac4ac8cdd278c4aba65e5fd839104dd0cc822c6d429`.
До остановки требуются local pending0, terminal runs, remote idle и отсутствие
owned children. Полная private backup после proven exit; state не восстанавливается.
Config/plist/authority pins и прежние permissions остаются обязательными.
При неоднозначном exit — FAIL/manual inspection, без перекрывающего запуска.

**QA172 TEST SOURCE APPROVE** frozen adapter SHA
`490d13fa844fd3200df4800b320803710658bc411a6785ef7cfc33a608ccb61f`: обратная
замена только binary SHA и новой строки local_worker_revision восстанавливает
точный прежний approved adapter SHA3c1e38; Node syntax PASS. Server revision273
не заменена local worker revision. Scope прежний readonly GUI three-turn сценарий,
execution только после подтверждённой установки и отдельного root GO. На момент
этого review исполнение NOT-RUN, прежний native preparation FAIL сохранён.


**LOCAL INSTALL EVIDENCE APPROVE**: private install proof PASS; reviewer
независимо read-only сверил actual binary/config/plist hashes и UID, exact
top-level running LaunchAgent и ps executable. Current state против full private
backup: все11 authority fields прежние, worker ID новый, epoch вырос, pending0,
все local runs terminal. Clean62e build proof совпадает с установленным10e845.
Повторный readonly native QA172 допускается после отдельного root GO; это
диагностическая замена, не доказательство устранения исходного сбоя.

## Диагностический native retry172: фактический отказ

**FAIL EVIDENCE APPROVE**, results SHA
`859c80b36e718ccc5a5814728dde70bcf152041d2a90fc3e4b49b2e02fcd3c8e`, executed
adapter490d, installed62e/10e. Actual GUI start accepted→failed за5159ms,
provider conversation отсутствует. Новый private allowlisted stage —
`prepare_discovery_timeout`; public error остаётся generic. Reviewer независимо
прочитал actual local journal: тот же stage/failed, pending0. QA cards cancelled,
QA profile archived, GUI baseline/config/binary/adapter/persistent worker preserved.
Новых повторов нет, native control acceptance остаётся FAIL.

Обнаружена граница прежнего QA177: временный LaunchAgent копировал env/cwd,
но не ProcessType=Background установленного worker. Поэтому его PASS не доказывает
поведение в точной resource classification. Installed local man и
[официальный launchd source Apple](https://github.com/apple-oss-distributions/launchd/blob/main/man/launchd.plist.5)
подтверждают разные resource limits у Background и Standard/unset/Interactive.
Это гипотеза для controlled config-only184, а не установленная причина timeout;
таймаут/ownership guards не ослабляются.

## QA184 final source gate

**SOURCE APPROVE**, private runner SHA
`81517722ead73f17d55a1bf8aa7af57648acbe1e2f2f1cfd8e1f92c69400024f`, helper
`653b37345971fcbccaba58425bcac2167dc0ecf9b2da3152f9de575e6b0459b5`, compiled
`d48cacdef69e547d815080bb28ee84c743d7090f592b25bb0e0e52ec27593182`.
Reviewer независимо сверил hashes/Python AST и все363 source files против
manifest и committed62e. Source inventory: только эти363 и reviewed private test;
WORK177 source/binary/report не менялись. Будущая automation working tree в
сборке probe не участвует.

Ownership-uncertain discovery/verify записывает diagnostic и немедленно завершает
Go test, не исполняя остальные samples. Wrapper прекращает последующие modes
при ownership suffix, missing report, outer deadline или несогласованном exit;
finally независимо пытается убрать только own labels и сохранить FAIL proof.
Plain timeout с proven cleanup может продолжить ограниченное сравнение и остаётся
FAIL. Background/Standard/Interactive сохраняют одинаковые operational plist
поля кроме ProcessType; runtime class и top-level exit фиксируются. Модели и
MCP tools/call не запускаются, persistent service/config/plist/GUI не меняются.
Runtime **NOT-RUN** на момент source gate, требуется отдельный root GO.

## QA184 actual evidence и следующий ограниченный probe

**FAIL EVIDENCE APPROVE**, report SHA
`020227282df72ade01848c6e9ef363a0b853bdd4fe4e4acaa0163dca54299450`.
Background runtime class подтверждён, но за100s не появился даже helper test
entry: actual log0, sample report отсутствует. Collection incomplete;
Standard/Interactive **NOT-RUN**. Это не proof native discovery root cause.

Reviewer независимо rehashed все8pins и363 source files: drift0; все3 own labels
отсутствуют, persistent worker running. Fresh same-user OS scan reviewer показал
0 matching helper,0 codex mcp-list inspectors и0 own xpcproxy. Это текущее
наблюдаемое отсутствие, а не восстановленный nonce/PGID proof старого процесса:
pre-bootout PID/PGID не были захвачены, manual-inspection flag и FAIL сохраняются.
Сигналов процессам reviewer не посылал.

**NEXT PROBE PLAN APPROVE**: отдельно Standard и Interactive, fresh private root/
labels, тот же committed62e source/helper, только config-only/no model. Новый
runner до исполнения требует source review; фиксирует top-level PID/UID/comm/
starttime/PGID и family во время poll и непосредственно до own-label bootout,
проверяет post-state без broad kill/numeric PID signals. PID reuse/unknown exit
не объявляются proof; текущая baseline/пины и permanent service сохраняются.
Новый Background запуск не требуется. Scope не включает provider model, worker
config/plist replacement, production/schema changes или автоматизацию TASK014.

## QA188 final source gate

**SOURCE APPROVE**, fresh private Standard/Interactive runner
`66f0acde1d89dcc4e2a67ba00e28c87b714baf19b7c7911d712305d7cb34b351`, read-only
inventory module `a4c7e6e8937a0040327bf01fe78f252ffa61999a098cbcc7a3125c5dc0d750e0`.
Reviewer независимо прочитал оба source, AST/hash PASS. Helper d48cac/private
Go653b373 unchanged;363 source files совпали manifest и committed62e, inventory
содержит только их и reviewed private test. Automation working tree не участвует.

Fresh own labels только Standard/Interactive; Background не повторяется. Во всех
poll/prebootout/postbootout фиксируются только наблюдаемые label-bound PID/UID/
comm/lstart/PGID/family. Полный OS census/args/env не сохраняются. Census error
не отменяет попытку bootout exact own label. Unknown identity, отсутствующий
helper entry, missing result, outer deadline или ownership uncertainty дают
FAIL/manual inspection и прекращают последующие contexts. Numeric PID signals
не используются. Секундная точность ps lstart и ненаблюдённые потомки явно
ограничивают доказательство; sampled family не считается OS containment.

Прежние input pins плюс inventory, GUI baseline и running persistent service
проверяются отдельно; процессные/model/tool actions рабочего проекта не входят
в scope. Actual execution **NOT-RUN** до отдельного root GO. Исторические177/184
reports и manual-inspection flag184 остаются без изменений.

## QA188 actual failure и gated startup plan

**FAIL EVIDENCE APPROVE**, report SHA
`cc2b04fc6db6f51ffa7c58d1f2bc0c3ba003ce0e100688f172a34f1a78ed0eef`.
Reviewer independently verified report/inventory: первый exact-label helper
snapshot UID0/PGID1; через41ms тот же PID/comm/lstart — UID501/PGID=PID.
Startup assertion остановил probe; log0/no result, runtime class не успела
зафиксироваться. Interactive NOT-RUN, native outcomes отсутствуют. Preexec
transition остаётся гипотезой; это не новый proof причины GUI timeout.
Все9pins/363sources unchanged; known family после bootout отсутствует, own labels
отсутствуют, persistent worker running. Manual flag/история FAIL сохраняются.

**GATED STARTUP PLAN APPROVE** для отдельного fresh private probe: перед любым
readNative/verify helper ждёт bounded per-context256bit nonce gate и сверяет
свой runtime PID/UID/PGID с gate tuple. Runner сначала только сохраняет untrusted
startup observations; wrong UID/PGID не усыновляются как owned, сигналов нет.
Private0600 exclusive/atomic gate публикуется лишь после exact fresh-label PID,
UID501, exact helper comm, PGID=PID и stable birth tuple (предпочтительно два
samples). После gate прежняя строгая проверка identity/family сохраняется.
Startup/gate/outer deadline и ambiguity прекращают все дальнейшие contexts,
manual flag остаётся; cleanup только собственного label, без numeric PID signals.
Новый helper/source/binary требуют отдельного source gate до root execution GO.
Background не повторяется, persistent service/config/GUI не меняются.

## QA190 gated probe source review

**SOURCE APPROVE** fresh Standard/Interactive private probe:
runner `469d93b926d7c1b3c84b8970237935f006547cf04aca2c8291752a46cae3ce3e`,
inventory `969ddbc8c5d6bfcc73127cfc188312919c428165f9375c7cd2f2f8a8b12119b5`,
private Go `75ae54889c15f123405005da270503d4243504a1b9f6c7344bb169096a0fe912`,
compiled `81b1827610510bb1804def5a0bb20194393faa92258f049d03aadbe833a9e1fd`.
Reviewer независимо прочитал все sources, Python AST/4hashes PASS; все363 owning
files exact manifest и committed62e, source inventory содержит только их и
reviewed private test. Никакой automation working-tree code не используется.

До любого readNative/verify helper ждёт per-context256bit nonce gate20s и
проверяет exact private path, regular0600/current nonroot owner, Lstat/Fstat
same file,4096byte limit, nonce и собственные PID/UID/PGID. Runner gate25s:
переходные UID/PGID rows только untrusted, два stable exact-label samples
UID501/exact helper/PGID=PID/birth tuple перед exclusive atomic publication0600
после fsync. После gate прежняя строгая identity/family tracking; native5s
Wait/ownership guards неизменны. Gate/outer/missing-result/unknown failures
останавливают последующие contexts, cleanup только own label, FAIL сохраняется.

Nonce/args/env не пишутся в reports, numeric PID signals/model/tool calls и
persistent worker/config/plist/GUI edits отсутствуют. Sampled family/second-precision
birth limitations прежние. Actual execution **NOT-RUN** до отдельного root GO;
исторические184/188FAIL/manual flags не изменяются.

## QA190 actual failure и HOME runtime-artifact plan

**FAIL EVIDENCE APPROVE**, report SHA
`04f0ba65c155e9de7bc966abd33d71dc9604b0d043a8265eb09a61af4c4e4f17`.
Reviewer независимо сверил645 samples, separate/embedded inventory, gate publication
после двух stable own UID/PID/PGID observations. Это publication, не acceptance:
за100s нет Go entry/log/result. Standard actual daemon3, Interactive NOT-RUN;
семейство содержало только helper, postbootout observed family/helper0. Все9pins/
363sources unchanged, own labels absent, persistent worker running. Manualtimeout
и прежние FAIL сохранены. Причина native GUI timeout не установлена.

**NEXT PLAN APPROVE**: fresh runtime helper/spec/gates/logs в private HOME directory,
byte-exact reviewed81b182 helper. WorkingDirectory=/Users/shagi уже был у permanent
и прежних probes: он остаётся прежним вместе с env/Standard/native5s/supervision.
Hardcoded native GUI cwd остаётся неизменным. Только runtime-artifact location
меняется; source/GUI не переносятся, xattr/quarantine/privacy settings не изменяются.
Новый wrapper требует source gate до запуска, новые private labels/nonce обязательны.

[Apple Platform Security](https://support.apple.com/en-gb/guide/security/secddd1d86a6/web)
подтверждает отдельный доступ к Desktop, но это лишь гипотеза. Если helper начнёт
тест и native subprocess откажет, это локализует этап, не доказывает TCC/cwd cause
без controlled contrast либо correlated OS evidence. Допустим отдельный bounded
read-only TCC log extractor только known owned PID+fixed service/result enum/count,
без raw log messages/paths/identifiers/env и без TCC.db/tccutil/setting mutations.
Недоступные logs не выдаются за отсутствие отказа или PASS. Model/native exec/
resume и production изменения этим планом не разрешаются.

## QA192 HOME-artifact source gate

**SOURCE APPROVE**, HOME wrapper
`a93b7cad06ac3f9218a6671142a0129cad7dd8b071180c0d14ab6cee78955b7c`;
unchanged inventory969ddb/private Go75ae54/helper81b182. Reviewer прочитал весь
wrapper, AST/4hashes PASS, copied helper побайтно совпал190. Все363 owning sources
exact manifest и committed62e, extra source только reviewed private test.
Fresh HOME root canonical0700 с current owner, runtime binary/spec/gate/logs
только там. GUI/native cwd и permanent WorkingDirectory=/Users/shagi прежние.

Standard первый; Interactive только после full Standard PASS и отсутствия
ambiguity. Failed context прекращает дальнейшие contexts; exact existing binary
может завершить свои прежние bounded samples, но не получает новый режим после
отказа. Native5s, gate/identity/Wait/PGID/fail-stop и cleanup guards прежние.
Модели/native exec/resume/MCP calls, xattr/privacy/TCC edits и permanent changes
отсутствуют; OS log extractor не входит в scope. Runtime **NOT-RUN** до отдельного
root GO. Результат будущей пробы не переписывает исторические FAIL/manual flags.

## QA192 actual outcomes: конфигурация PASS, весь harness FAIL

**EVIDENCE APPROVE** с раздельными статусами: phase report SHA
`c59ff37c56e7f81c42047681cd7278d022db2bf746bacba1aa6fa067214fbc13` действительно
Standard actual daemon3, startup_gate accepted, шесть native config phases PASS:
discovery1956/33/36ms, verify46/33/34ms. Reviewer independently сверил сам report,
all9 input pins/363sources, own labels absent и running persistent worker.
Это реальные конфигурационные чтения, не native model/task workflow.

Общий report SHA
`7c7210da37d6b12c8cc35e475414a2606454b732dac333dcf37d47f053c2d288` сохраняет
FAIL/incomplete/manual inspection: после завершения helper comm стал parenthesized
при той же sampled birth/UID/PGID. `stat` не фиксировался, поэтому zombie не доказан.
Interactive NOT-RUN. Known family/helper после own-label cleanup отсутствовали;
GUI baseline/config/binary/source unchanged. Исторический отказ не переименован.

Следующее точечное harness improvement может читать ps stat и разрешать terminal
comm difference лишь для уже известной identity с actual stateZ; live mismatch
по-прежнему FAIL. Overall PASS всё равно требует helperJSON PASS/gate accepted,
exact launchd exit0 и проверенный cleanup. Parser units для Z/live mismatch и новый
source review предшествуют любому runtime; state не угадывается по скобкам.
Это план исправления измерителя, не доказательство исправления GUI native timeout.

## WORK195 — real persistent Mac GUI runtime acceptance

**EVIDENCE APPROVE / Mac PASS5** after independent source/evidence readback:
results SHA `3eae9067f6fbae162660d5e6538a20a34bf65b04c2098b307abd230f2dfbaec9`,
safe runtime proof `bcc24865e87e5d9b863eb34a410e08a462b95fc52cb6cb6ce96ce3a4baaf108f`.
Executed adapter exact490d; server273/local diagnostic62e binary10e845 and config
f400 pins verified. Three native turns use one exact conversation UUID and run,
generations1completed→2stopped→3completed. Gen2 colleague UI clarification queued
and stop applied/proven; gen3 owner resume uses same UUID and applies clarification.
Actual README SHA markers initial/final match current GUI README hash.

Есть ровно ДВА real task claims: fences1→2, разные board sessions, оба ended
managed_completed, lease отсутствует. Stopped turn прерван до дополнительного
claim; три turns не выдаются за три leases. Current local journal: own run
completed/gen3, diagnostic empty, pending0, prior records retained. GUI before/
after baseline3316 совпадает исходному, binary/config/executed adapter unchanged;
real persistent LaunchAgent top-level running.

Дополнительно reviewer выполнил normal authenticated readonly browser readback
через unchanged reviewed client: own QA task и epic cancelled/no lease, QA profile
archived. Только browser login/project session open/logout; native/host/task
mutations reviewer не делал. Normal logout завершён.

**Mac native smoke gate для finalize уже установленного273 SATISFIED.** Остальные
owned updater checks actual Linux PID/ELF/schema5/floors/full all-project postbackup
и complete journal выполняет root по ранее reviewed capable updater. Это не gate
для schema6/TASK014 или новой поставки. Windows/Claude/physical second PC NOT-RUN;
два browser principals на одном Mac не являются двумя машинами. Прежние failures
сохраняются; источник intermittent preparation failure не объявлен доказанно
устранённым diagnostic-only binary.

## WORK196 — capable production finalize, независимое readback

**EVIDENCE APPROVE** для уже установленного release273. Reviewer прочитал
process-complete-proof.json и final-all-project-backup-proof.json: updater phase
complete, Go ELF SHA5a1e135 и Next ELF SHA bc17c508/build EJT-3udsZCJ4tJm9eDhiL
соответствуют прежним stage/apply pins; helpers/startup/backup checks PASS,
authority marker600 preserved, recovery markers отсутствуют. Pre-snapshot
validated и logical preservation отмечены отдельно от final snapshot.

Final format2 snapshot validated, manifest SHA
1b47835abe0755ecc26d8081eff313a67d5a9426aaf73993c45435a201f14466;
registry inventory matches current live,4 published boards/9 runtime databases
и5 historical migration databases, всего14 SQLite files. Backup service success.
Старое поле published_projects=1 в process proof относится к original apply
inventory; оно не используется как утверждение о текущих четырёх проектах.
Текущая полнота проверяется отдельным final inventory proof.

Вместе с real GUI195 это закрывает Mac/production gate TASK013 для273 + worker62e.
Это review сохранённых operator proofs, не новая remote mutation reviewer.
Windows/Claude/физический второй ПК по-прежнему NOT-RUN; schema6/TASK014 на сервер
этим gate не допускается. Предыдущие diagnostic failures и неизвестная причина
intermittent timeout остаются историческими результатами.
