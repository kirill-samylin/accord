# WORK202 — независимое продолжение review TASK014

23.10.2026. Reviewer `/root/automation_final_review`; владение только этой
карточкой. Прочитаны AGENTS, requirements/architecture, automation contract,
WORK189/198/199/201 и соответствующие private sources/evidence. Прежний
composite SOURCE APPROVE189 не переименовывается в runtime/production approval.
Product, harness, native config, рабочие данные и production не изменялись;
новые process/model/SSH invocations не выполнялись.

## WORK201 — CONTRACT APPROVE с зафиксированными границами реализации

Default-off local opt-in только для Codex automation developer согласован с
host-local consent и прежним `local_commits` контрактом. Включение через server
policy/profile недопустимо; consent digest/version и server policy должны быть
обновлены обычным явным путём. Manual и остальные роли сохраняют старый режим.
Разрешение Git metadata не выдаётся за whitelist commit-команды или полномочие
push/merge/deploy. Exact canonical checkout/.git, read-only config/hooks и отказ
от external gitdir/linked worktree сохраняют выбранную узкую границу.

Проверены официальные [permission profiles](https://learn.chatgpt.com/docs/permissions)
и [app-server](https://learn.chatgpt.com/docs/app-server). Profiles не сочетаются
с active legacy sandbox settings; layers могут дополнять один named profile.
Поэтому одного имени/default_permissions недостаточно: нужен exact effective
profile с проверкой всех применимых layers/managed constraints. `:workspace`
также имеет baseline system-temp writes: новый grant расширяет только `.git`,
а отрицательный sentinel вне checkout следует выбирать также вне разрешённых
baseline temp roots. Глобальный запрет любой записи вне checkout не заявляется.

Дополнение root/author193 к WORK201 — **фиксированный config-only dialogue
внутри supervised helper** — также CONTRACT APPROVE. Необходимые критерии:

- Child stdin для JSONL dialogue отделён от parent ownership lifeline. Helper
  сам выдаёт только initialize → matching response → initialized → config/read
  и configRequirements/read; внешний вызов не задаёт arbitrary method/params.
- Закрытая state machine проверяет exact request IDs, result/error, порядок и
  ожидаемые shapes. Duplicate/unknown replies, malformed frames, unexpected
  server requests и RPC errors дают fail-closed; helper не отвечает на запросы
  auth, attestation, approvals, tools или configuration writes.
- Ограничены суммарные bytes, frame size/count и полный wall-clock deadline,
  включая stalled writes/reads. EOF lifeline/stop/timeout ведут к owned cleanup
  и drain/reap с nonce-bound positive exit proof. Unknown сохраняет quarantine
  и checkout lock, не трактуется как отсутствие child или разрешение retry.
- Raw config, layers, errors и stderr могут содержать secrets. Они не попадают
  в обычный provider transcript, stdout, server report или журнал. Допустим
  только bounded private transport/capture; наружу — sanitized facts/digest.
- Config proof делается с теми же executable/cwd/environment и generated
  permission overrides, которые будут у native exec. Перед spawn перепроверяются
  authority, consent/version, path/executable identities и актуальность proof;
  несвязанный app-server snapshot не является доказательством native режима.
- Все MCP connectors на inspection выключены whole-map exact verification;
  обычный managed-MCP preflight не заменяется этим config-only proof. Никаких
  thread/model/auth/tool/config-write calls, global settings edits, broader
  fallback или silent downgrade при неподдерживаемой версии CLI.

Это допуск к product реализации и авторским unit/integration tests, **не** к
реальному config RPC/model процессу. Source review, отдельный QA/source review,
config-only proof, actual single commit и OS-enforced negative sentinel checks
остаются обязательными. Отказ текстом модели enforcement proof не заменяет.

## WORK199 V2 — SOURCE APPROVE, runtime NOT-RUN

Проверены `.local/task014/linux-qa199/v2/` пять sources и manifest SHA256
`dfc9428d47e4013193ed2239511e5bbe2ba848149931c26f2eb5e63424ff4236`:
independent hash recalculation 5/5 совпадает. Original manifest
`d02e935426f79c4dff7de76bb96eddb5767c3a8a02978207bbf6499ba90ab240`
сохранён. fixturegen/oracle/lifecycle byte-identical; delta только public
catalog switch seed и безопасный observed HTTP status в failure report.

Fresh CSRF и server-issued switch action запрашиваются перед каждым scoped
GET; owning handler проверяет catalog authority/action, target prefix и вызывает
OpenProject. Cookie тот же, redirect проверяется. Это исправляет отсутствие
ProjectBinding поддерживаемым публичным flow без direct SQL/ACL bypass.
Первоначальный HTTP code остаётся NOT-RECORDED; source-derived409 не выдаётся
за наблюдённый ответ первого run.

Сохранены execution360s + cleanup reserve60s, signal→finally, exact registered
Popen handles, bounded terminate/kill/reap, explicit unknown/manual-recovery,
independent before/after source/binary pins и exclusive durable FAIL report.
Production paths/services/pointers не используются. Storage typed rows остаются
domain-invalid: такой PASS проверяет migration/reset/history, не live workflow
authority, native pipeline или systemd/Next activation.

P2 provenance: текущий runner всё ещё требует `reviewed_by:169` в root GO,
хотя V2 delta проверяет202. До запуска рекомендуется exact marker `202` либо
явный base169+delta202; не приписывать169 review нового V2. Коррекция требует
нового pin/narrow review; runtime только по отдельному exact root GO в fresh root.

Повторный narrow pass: tester изменил только ожидаемый reviewer marker на202.
Финальный V2 manifest
`ddabcc1369c1ef9a482d7e6d0da68a2b6d1527b9557278adc487575e47f3ffbc`,
run_linux `103eccf2c0940bebe25f542e29e24e2b364a8b557f6796aea34b12f0a3a66362`.
Independent5/5 hashes совпали, остальные четыре файла неизменны относительно
предыдущего dfc V2. **FINAL SOURCE APPROVE199V2** на ddabcc; P2 provenance закрыт.
Это допускает отдельный root GO, но не заменяет его и не означает execution PASS.

## WORK198 V2 — EVIDENCE CLASSIFICATION APPROVE, overall FAIL сохранён

Независимо прочитаны safe proof и results; hashes совпали:

- results `4b34c6852a53b479e25fac6058096b8a316928fc08ea637c007cfc6206cc52ff`;
- safe proof `4ff7eec49198a60f01ce72883ac9bb9d1e913b3b4247aeef9ca1df840316c178`.

Observed facts: один настоящий native UUID/turn.completed, exact file write,
owned exit0/drained output, три supervised calls с positive proofs, все пять
MCP connectors выключены, MCP calls нет. HEAD без нового commit. Команды:
pwd, git status, rg, printf; **нет observed git add/commit** и фактического
соответствующего OS error. Поэтому classification — semantic refusal/noncompletion,
не доказанный sandbox Git denial. Одного exit0 недостаточно для результата.

Отдельно harness зафиксировал native user-config hash drift. До-run hash/bytes
не сохранены, причина и автор неизвестны; текущий trust entry не доказывает
причину. Fail сохраняется, config не восстанавливается предположением. Отсутствие
commit и drift — две самостоятельные причины не принимать prerequisite PASS.
Новый opt-in не объявляется доказанным лечением до положительных/отрицательных
actual checks. Полный TASK014 pipeline и production schema6 остаются NOT-RUN.

## WORK199 V3 — узкий SOURCE APPROVE после V2 actual failure

V2 results `54b6fa27f2ca578f6b3aee68b0962da4348de3c2f7aa363e7e37babd85aa9d54`
и readonly postcheck прочитаны. Четыре положительных checks подтверждают actual
Linux seed4boards/9DB, schema5 format2 snapshot и migration5→6 с сохранением
legacy rows/четырёх schema5 copies. Overall FAIL на old-binary-refusal assertion
сохранён. Per-command exit/stderr не записаны; expected public error выведен
из owning273 source, а не представлен как observed V2 error. Restore NOT-RUN.
Оба owned serve children exited0, pins неизменны; postcheck не заменяет FAIL PASS.

**SOURCE APPROVE199V3**: manifest
`c552c0023c8485e120618082a5ea583a90795640952bcdaef0d407584dad184c`,
run_linux `9b2e67e38d2fabf2e05d072f8592ad700d0312da62d0fecfd1cd356318d837af`.
Independent5/5 hashes совпали; остальные четыре sources byte-identical V2.
Owning273 `cmd/agent-board/projects.go` действительно редактирует внутренний
MigrateStorage error в общий публичный текст, а main добавляет prefix/exit1.
Новый matcher exact exit1/stdout-empty/public-stderr соответствует этому API.

Общая ошибка сама по себе недостаточна: сохранены exact old/new executable
hashes, actual old schema5/all previous capability floors, candidate capable
guard и actual schema6 metadata/tables/indexes всех4boards. Все state bytes и
logical rows должны быть неизменны после отказа. Safe observed exit/class/hash
и обе equality facts записываются до assertion; raw stderr не публикуется.
Остальные migration/reset/history assertions и lifecycle не ослаблены.
Execution только по новому exact root GO в fresh root; это source approval,
не результат третьего actual run и не production updater/systemd proof.

## Root documentation — DOC APPROVE

Прочитаны текущие README, TASK014, новая automation-guide и delta managed-worker-guide;
названия основных действий сопоставлены с actual frontend, pause/cancel/replan —
с store control. Исправление stale GUI FAIL на позднейший WORK195 realMacPASS5
согласуется с reviewed evidence189/169 и отдельным TASK013 scope.

Инструкция явно отличает интерфейс source candidate от установленного schema5,
host-local enrollment/OS control от editor membership, browser от scoped MCP,
profile от worker, queued от proven stop, local commit от Git synchronization.
Windows/physical second PC и full automation acceptance не объявлены PASS.
Требования к реальным checks, выбранной policy, вопросам и human controls отражены.
TASK014 сохраняет NOT READY, V2 migration partial PASS и restore NOT-RUN.
README/source-build checks не принимаются за native pipeline/runtime delivery.
**DOC APPROVE** текущего root delta; последующие новые результаты и WORK204
source/QA требуют своих checkpoints и review, это не авансовый approval.

## WORK203 — PLAN APPROVE, executable harness ещё не существует

Independent static plan review: WORK203 SHA
`d38d6cf5ddaf6b1ed026beb319665a45ce928b6c5ffea2e84025a67a895b7d2e`,
private declarative plan.json
`ce11f25173e8c29e9a1d55303ad21b690fceae31ef23f7d93b66d594c954a29f`.
Оба hashes независимо совпали. **PLAN APPROVE203** minimum happy contour:
одна задача, четыре настоящих native роли, отдельные NativeUUID/ManagedRunID/
SessionID/contributor provenance, один owner principal допустим. Никаких fake
человеческих principals для scheduler или пятой summary-модели.

Budget4/zero fixes/zero continuations допустим actual limits validator и
соответствует analyst→developer→reviewer→tester; optional zero budgets source
не нормализует в дополнительные retries. Developer и tester actual checks не
считаются дополнительными native invocations. Native180s/workflow900s и отдельные
setup/cleanup bounds явные; refusal/question/unknown — FAIL, не hidden retry.
Negative fix loop требует своего будущего семистадийного budget/scenario.

UI trusted policy action и exact selected scope соответствуют текущему browser
контракту. Cross-channel replay с тем же action_id/request_id/payload обоснован:
automation idempotency key зависит от principal/access epoch, не board SessionID;
current authorization проверяется до cached replay. ClientIntake=true обязателен
для ordinary MCP channel. Scripted real transport replay не называется native
client intake model proof. Profile openai→policy codex поддерживается source.

Предложенный Python stdlib fixture содержит два meaningful теста и actual JUnit,
fixed preset/source roots/artifact root разделены. Parent не создаёт native
commit, task done, plan, accepted/started receipts, attestation или check counts.
Exact parent/tree/HEAD/source manifest и независимые review/QA на одной версии,
positive owned exit/ACK, actual reports и drained paginated evidence обязательны.
Существующие семь consent tools соответствуют actual automation role union.
Setup helper только для exported identity/digest API также требует source review.

Новые modules/fixture source можно готовить для проверенного3e6 framework.
WORK201/204 extension остаётся prerequisite: ни opt-in API старому binary,
ни effective permission proof не придумываются. Части harness для нового режима
авторятся после204 SOURCE APPROVE. **До reviewed extension, config-only QA,
single commit/OS-negative proofs, нового exact candidate pin/source approval
и отдельного root GO никаких setup/server/worker/model/config RPC запусков.**

Cleanup сохраняет uncertain ownership/journal/reservation/manual inspection,
не удаляет данные ради PASS; оба explicit deadlines и always-save FAIL ещё должны
быть реализованы и проверены в source. PLAN APPROVE допускает authoring, но не
заменяет review будущих executable sources и их каждого изменённого manifest.
Полный pipeline, отдельные negative/control сценарии, Windows/Claude/physical
second PC и production остаются NOT-RUN этим review. Процессы не запускались.

## WORK205 — CONTRACT APPROVE локального readonly preview

Прочитан WORK205 и существующие Config.Load/Validate, identity, executable и
dependency digest helpers. **CONTRACT APPROVE205**: отдельный CLI dispatch до
Open/Engine; strict private draft parsing с прежними ACL/ownership/size/duplicate/
unknown-field правилами; предложения только в independent in-memory copy.
Отсутствующие hash заполняются предложением, неверные непустые binary/dependency
pins дают отказ, не заменяются текущим содержимым. Полный Config.Validate с
локальными role/native consent rules обязателен до формирования success JSON.

Preview не означает consent или execution: applied=false, ручной перенос после
проверки владельцем, затем отдельный worker consent/browser policy update.
Server/controller/provider config, args, raw config, token/state и secret values
не печатаются. Только bounded sanitized JSON aliases/preset IDs/versions/hashes/
digests; полный buffer проверяется до записи, ошибки не дают partial success.
Нет Engine/token/state access, HTTP, процессов, config writes или auto regrant.

Deadline сформулирован корректно как cooperative cancellation context30s:
проверки до/после этапов и между bounded reads/entries, а не гарантия прервать
зависший kernel IO. Existing Config.Validate повторяет некоторые hash reads без
context API: перед/после него context проверяется, но этот вызов нельзя выдавать
за внутренне прерываемый. Лимиты размеров/числа entries должны сохраняться;
preview hashing желательно проверяет cancellation между bounded chunks.

Implementation ownership root: четыре заявленных setup-preview/cmd files и
короткий dispatch; shared204 files не меняются под это поручение. Pure unit/file
tests покрывают mismatch, strict parsing, current flags, bounded output, errors
и отсутствие побочных действий. Это допуск к реализации/авторским unit tests;
SOURCE APPROVE новых файлов ещё нужен. Нативный запуск/Windows/pipeline и
production этим contract review не проверены; процессы reviewer не запускал.

## WORK204 — первый source pass, P1 перед финальным approve

Прочитаны все23 scoped Go files и новые unit sources; initial fingerprint
`82b121e07fc10a86766e6a0f8e2d792a0166fd7a5eafa600bf3640ed8cb7e46e`
независимо совпал23/23. Role/default-off separation, legacy false digest
compatibility, exact Git profile, fixed dialogue/framing/private raw capture,
owned cleanup/errOwnership, double proof и launch authority barriers проверены.
Никакие actual app-server/config/model процессы reviewer не запускал.

**REQUEST CHANGES204 P1:** provePermissionConfig отвергал nonnull legacy sandbox
keys только в layer.config, но effective response.Config допускал sandbox_mode
или sandbox_workspace_write. При непрозрачном resolved/selected legacy mode
это могло принять присутствующий, но не применяемый named profile. Требуется
fail-closed для nonnull effective legacy keys; typed null разрешён. Regression
должен изменять только effective map, не общий alias с clean layer map, включая
empty-layer negative и nullable positive cases. Автор ACK и узкий fix готовит;
новый pin/tests/final source approval пока pending. Первоначальный82 pin не
является допуском к новым process QA.

## WORK204 — FINAL SOURCE APPROVE после P1 correction

Автор исправил effective Config nonnull legacy rejection и добавил независимый
effective-only regression, где layer map остаётся чистым; typed null positive
cases сохранены. Новый scoped23 fingerprint
`eea3f68a765ec122278cda976774493cbc0859e5ea22ee947f3f98cce53c9714`
independently23/23 совпал. Delta ограничен parser и regression source.
Reviewer targeted race13 новых cases PASS1.831s, включая fixed dialogue,
malformed frames, bounded stalled writes/ownership, exact actual launch builder
на mocked no-child process и double-proof/stop-epoch refusal. OS child/model/
config RPC тестами reviewer не запускались. Author full race13.155/1.398/3.105s
и Linux/Windows cross-build отделены от actual native/platform evidence.

**FINAL SOURCE APPROVE204** на eea3 pin. Default false сохраняет прежний consent
digest, true требует новый owner acknowledgement; manual/other-role/provider
grants не расширены. Exact selected effective permission tree/layers/managed
constraints, disabled MCP inventory, fixed config-only helper, private raw
response boundary и preserved uncertainty/lock являются принятым source scope.
Source не обещает отсутствия startup side effects самого CLI; actual config
shape/EOF behavior, single native commit и отрицательные OS sentinel checks
ещё обязательны. Отдельный QA теперь может авторить process probes на этот
reviewed source; каждый executable harness требует своего SOURCE REVIEW и
root GO. Эта карточка не активирует permission mode, не обновляет production
и не закрывает полный TASK014, Windows или физический второй компьютер.

## WORK205 — SOURCE APPROVE локального preview

Проверены6 scoped files, manifest SHA
`0adcb75c5d136d07e342f197eb4a7d339d33f29d2bf1831ea55c6926c7c39129`;
independent6/6 hashes совпали. Новый dispatch входит в preview до LoadConfig/
Engine.Open, strict private decoder не использует token/state/network/process.
Completed proposal строится в clone, nil dependency-map сохраняет прежний digest,
existing nonempty pins не заменяются. Topology проверяется до dependency scan;
executable hash читает bounded chunks с cancellation и проверяет identity/stat.
Полный Validate, post-validation context и1MiB encoded cap стоят до output.

Вывод whitelist DTO applied=false/requires_local_apply=true, flags и presets
не включаются автоматически; raw URL/args/controllers/token/provider config
не выводятся. Существующий digest меняется предложением с consent_changed,
но конфиг/consent не записываются. Cooperative deadline описан без обещания
остановить kernel I/O или внутренние synchronous reads Config.Validate.

Reviewer targeted pure file/unit race PASS worker1.509s/cmd1.500s, включая
actual command dispatch без запуска OS child, missing-token/state proof,
existing dependency/binary mismatch, strict JSON/ACL/symlink и cancelled context.
**SOURCE APPROVE205** на exact6file manifest. Actual installed CLI invocation,
Windows native и полное onboarding/pipeline runtime остаются отдельной QA.

## WORK199 derivative diagnostic — source correction pending

Прочитаны initial probe/support, manifestdfb93ecc. Derivative-only copy, explicit
two-field registry binding correction, immutable original/snapshot maps и
bounded supervised old CLI/lifeline/nonce proof соответствуют узкому root scope.
Raw rejection stderr не выдумывается; aux classification не превращает V3 FAIL
в acceptance PASS и не ослабляет DB/nonempty-WAL/history requirements.

До SOURCE APPROVE нужна provenance correction: current config.source_pins map
проверяется против файлов, но его canonical aggregate не привязан к сохранённому
V3 report.source_pins_sha256. Совместный drift mutable map+helper file иначе
может выглядеть unchanged. Требуется проверить exact original40 map aggregate
и исходный oracle pin до import/действий, сохранить safe proof. Author/root
уведомлены; diagnostic execution не допускается этим pending checkpoint.

## WORK199 derivative diagnostic V2 — FINAL SOURCE APPROVE

Исправлен provenance до oracle import/действий: exact accepted V3 report SHA,
его canonical40pin aggregate и literal `203f7c8cd6a7cdb9744b38e95e89574667abdafae885972506e18c47bc52cdee`
должны совпасть; oracle отдельно bound известным99d1d9fe. Original inputs SHA
сохраняется и перепроверяется после, safe provenance counts/digests — в report.
Все остальные reviewed clone/rebind/owned-proof/immutable-content guards сохранены.

**FINAL SOURCE APPROVE199 diagnostic V2** manifest
`d7fd7da52366454ff30c1128f742501b00c884fd0f79221a9f53aed4ff51cba2`;
probe703770f2/support379c1e71, independent2/2 hashes matched. Scope — один
derivative first-open diagnostic с отдельно root-разрешённой корректировкой
двух clone registry path fields. Original V3 не запускается/не очищается;
DB/nonempty WAL/history/snapshot/source mutation не скрывается aux classification.
Старый missing V3 before-map не реконструируется, его all-byte FAIL сохраняется.
Дальнейший exact root GO обязателен; reviewer новых процессов не запускал.

## Setup guide — узкие замечания перед DOC APPROVE

`automation-worker-setup.md` объясняет private fragment merge, настоящую project
test command/JUnit, selected roots, trusted-owner check mode, separate Git/tester
flags, preview→manual apply→consent→browser policy и отсутствие auto regrant.
Windows/Claude runtime и пока не установленная preview-команда обозначены явно.

Два малых исправления перед финальным DOC APPROVE: показать два literal
backslash в примере JSON Windows path вместо четырёх; для Claude-only ролей
явно указать codex_git_metadata_writes=false (true требует Codex developer).
Новые файлы reviewer не редактирует, root уведомлён.

Узкий повторный DOC pass: оба замечания исправлены. Windows JSON показывает
ровно два literal backslash, Claude-only developer явно требует Git flagfalse.
Setup guide SHA `0418d33d395884871f8e41819517402335fba0bd0aed284dc315a83beb9ff655`.
Ссылка из automation-guide ведёт на существующий файл. **DOC APPROVE** новой
инструкции и ссылки; source2056 pins повторно unchanged. Full Go race root
PASS — отдельное source/check evidence, не новый native/runtime PASS.

## WORK206 — initial source review, lifecycle correction pending

Прочитаны все пять Python sources, readonly Go identity helper, pins и README;
manifest `09cfec9eb293ad1c2f83220031e07216b4b79183b9c65a2e146818f6cf02ab11`
independent8/8 listed hashes совпал. Override/profile derivation соответствует
204 для допустимого ASCII subset; иных names/paths не угадывает. Exact disabled
inventory проверяется перед единственной закрытой config inspection. Parent не
передаёт RPC, proof приходит от candidate, secret-bearing raw config остаётся
внутри helper, private snapshots сохраняют before/after bytes и hashes.
Unclassified missing proof не превращается в guessed legacy/shape причину.

**REQUEST CHANGES206 lifecycle:** per-call deadlines и EOF/reap/drain уже есть,
но нет общего execution bound и SIGTERM/INT handler. Default signal может обойти
Call.run finally и итоговый always-save report, как ранее исправленный199.
Нужны явный whole-execution budget/cleanup reserve и signal→exception→finally,
с masking repeated signals только на cleanup. Existing helper20s/7s не менять.
Unknown proof сохраняется manual-inspection FAIL. Дополнительно не обещать
hardstop до любого mkdir, если sibling imports могут писать __pycache__: -B
entry либо dont_write_bytecode до imports делает это утверждение корректным.
Новый pin/source review до process execution; root/author уведомлены.

Отдельно **SOURCE APPROVE compile-only identity/main.go**
`d54c8b8b2b581f65bcd2ec6060d0514aafdf531818b47246726f5972bd412aaf`:
один canonical ordinary synthetic .git под HOME diagnostics, exported readonly
AutomationIdentity, bounded one-field JSON, no Engine/token/server/process.
Root может собрать exact source из своего clean ignored path и сохранить binary
pin, если это нужно параллельно lifecycle fix. Это не разрешение исполнить helper
или harness: exact new candidate/helper/input pins и отдельный root GO ещё нужны.

## WORK199 diagnostic V2 — EVIDENCE APPROVE

Report SHA `3076f352b1e3fef18cacc5635e9b7e3542db8a476a4e3aff596ca0c2a8ac66d4`
независимо совпал; executed probe/support hashes соответствуют approvedd7fd
manifest. Прочитан actual nonce proof: proven=true/exit1/success=false/forced=false;
operator exit0/outer_timeout=false. Это наблюдение отказа старого CLI, не AI run.

Из retained before/after maps независимо пересчитаны ровно четыре новых пути:
default board и registry SHM32768bytes, их два empty WAL0 с hash(empty).
Clone facts9DB до/после byte-equal, original state/snapshot maps тоже equal.
Report фиксирует unchanged source40/inputs/binaries/probe, no manual recovery.
**EVIDENCE APPROVE diagnostic only**: controlled derivative first-open показал
этот aux-creation mechanism. Он не восстанавливает missing original V3 map,
не меняет его overall FAIL и не подтверждает ещё не выполненный restore.
Primary [SQLite WAL documentation](https://sqlite.org/wal.html) допускает
создание WAL/SHM при readonly database open в writable directory; ненулевой WAL
остаётся частью persistent state и не исключается из проверок.

## WORK199 V4 — SOURCE APPROVE узкого aux oracle

Manifest `7480521c86f3f26f04a676d1fed9e1891550677f3003ee867be25ef2cbfcc150`
independent5/5 совпал. Только run969f98ca/oracle11764480 изменены; остальные
три sources byte-identical approvedV3. Source проверяет все прежние migration/
snapshot/authority-reset/history/reservation gates, lifecycle360+60 и pins.

Новая root-разрешённая observability allowance ограничена четырьмя exact путями
default board/registry: SHM create/change32768, только empty WAL create либо
unchanged. Удаления, иные пути, main DB/nonempty WAL/history/snapshot mutation
запрещены. Original all-byte predicate сохраняет правдивое false; отдельный
exact-aux predicate вместе с actual public refusal/exit1 и strict9DB logical/
DDL/integrity/floors решает новый gate. Полные карты и changed facts сохраняются
до assertions, предыдущие FAIL не переписываются.

**SOURCE APPROVE199V4** на exact748 pin; отдельный root GO и fresh fixture ещё
обязательны. P2 nonblocking diagnostic note: after-state immutable logical read
вычисляется до отказа по nonempty-WAL predicate. Final PASS защищён обязательным
exact-aux predicate, но для будущего negative report лучше не вычислять logical
equal при nonempty WAL/journal, а сохранить checkpointed=false/NOT-RUN. Этот
комментарий не разрешает исключать persisted WAL или менять oracle молча.
Runtime restore/typed history/systemd/production остаются непроверенными этим pass.

## WORK206 V2 — FINAL SOURCE APPROVE config-only harness

Lifecycle correction прочитана: Budget90s execution+30s cleanup, controlled
TERM/INT/alarm→finally, repeated external signals masked during cleanup,
independent cleanup alarm, atomic provisional FAIL до первого child/cleanup.
Call EOF/reap/drain и unknown-handle facts сохранены, global PID signals/retries
не добавлены. Bytecode отключён до sibling imports. Истечение cleanup reserve
сохраняет FAIL/manual inspection, не объявляет отсутствия живых процессов.

**FINAL SOURCE APPROVE206V2** manifest
`e035fb98c1c126975177dbdc2c6658c10cb2909e56589b313368d8c747f2948d`,
pins `e90dba69bc9ccfbf7ef99dbd3ad0e1f149b9051c2ffde1105c0e851dde17a63e`.
Harness entries и три исходных manifest hashes независимо проверены; baseline193
дополняется только approved204/205 overrides. Exact candidate8c43df9 SHA8726daa
и compile-only helper0d85bf8 совпали с локальными файлами; helper source d54c
сохранился. Принимается только исходный actual safe proof от закрытого candidate,
не reconstructed parent digest. Отдельный root GO ещё обязателен.

Scope допускает ровно один actual config inspection после supervised synthetic
Git setup/identity и двух5s MCP config calls. Models/tools/thread/turn/auth/config
writes, native commit/OS-negative sentinels/full pipeline остаются вне206.
Любой config drift сохраняется отдельным FAIL без restore; current helper отказ
без digest остаётся unclassified, причина не выводится догадкой. Readonly source
review не является config RPC runtime PASS; reviewer harness не выполнял.

## WORK199 V4-P2 — FINAL narrow SOURCE APPROVE

Отдельный manifest
`4f793a7217c8d3f5c7ab747dd5dcf52aff654858596fe30786fd9d2dd523a96c`
independent5/5 matched. Только run6a352bc0 изменён; остальные четыре файла
byte-identical approved748. После сохранения всех maps/receipt вычисляется
checkpointed_after. При nonempty WAL/journal immutable logical/schema reads
не выполняются: equality=null, explicit NOT-OBSERVED reason, strict FAIL.
Before-WAL0 guard, exact aux allowlist и все остальные assertions сохранены.
P2 diagnostic concern закрыт без ослабления persisted-content gates.
**FINAL SOURCE APPROVE199V4-P2**; execution только по новому exact root GO,
runtime/restore этим source pass не проверялись.

## WORK206 V2 evidence и V3 framing correction

**EVIDENCE CLASSIFICATION APPROVE206V2**: results06c987ca и safe-proofb34cf41
hashes независимо совпали. Все10 fixture Git calls natural/proven0/!forced,
helper exit0/output drained; config snapshots unchanged, source/binary pins
unchanged. MCP/app-server/model calls0. Raw десятый stdout действительно
содержит только framing LF и nonce proof: generic Supervise добавляет LF перед
proof даже для пустого child stdout. Fixture remote-empty assertion поэтому
FAIL; native permission/config path ещё NOT-RUN. Это не native CLI refusal.

V3 executable delta только owned.py: после matched positive nonce proof,
natural exit0 и !forced generic branch требует final LF и удаляет ровно один.
Это убирает только Supervisor framing, сохраняет child-owned newline; inspection
branch не меняется. Все прочие Python/Go sources, permissions/budgets/guards
сохраняются. **FINAL SOURCE APPROVE206V3** manifest
`e8c5dddacde7dc6b24c8aa737617435a4cb0aae5a7cb5c88e8a8836350215c01`,
pins `af78c02f4566a42975881f5aa3fc8c3d80fc5c3211dd693f1d653735717eb7c4`.
Independent harness hashes matched; owned8c83d2b7 — единственная executable
delta. Original V2 sources/report не изменены. Новый exact root GO и fresh root
обязательны до повторения; reviewer harness/CLI не запускал.

## WORK199 V4-P2 — EVIDENCE APPROVE PASS7, только Linux storage

Results SHA `e0eb6cba41711cb163f5136afa43193a9521a63a0388a1ccf4f9ddd9c469bc26`
independently matched. Report содержит7 положительных checks: actual4projects/
9runtimeDB seed, schema5 pre-format2 snapshot, migration5→6/legacy rows/four
historical copies, old schema5 refusal с exact public receipt и final offline
restore/reset/history/reservation checks. Refusal maps независимо показывают
только board.db-shm32768 и empty board.db-wal0 creation; snapshot maps equal.
Checkpointed before/after true; all9 logical/fullDDL/integrity comparisons true.
Оба owned serve exited0 без forced/unknown, source40/three binary pins unchanged.

**EVIDENCE APPROVE199 PASS7** для actual Linux Go/SQLite migration/snapshot/
offline restore reset. Typed offline rows имеют domain-invalid refs; это не
live automation policy/capability proof. Systemd/Next updater activation,
production upgrade, native pipeline и physical second PC не проверены этим run.
V1/V2/V3 failures и их прежние ограничения сохранены, не переписаны новым PASS.

## WORK206 V3 — EVIDENCE APPROVE классификации FAIL

Независимо matched results
`14dc5baa4ca661ff3ebbe1f9a6afb2924f81322e3b7d331d0c2e93c4de348283`
и safe proof467dfdf3. Fifteen preceding setup/identity/MCP calls proved natural0;
actual MCP5/5 disabled. Единственная candidate inspection вернула proven=true,
success=false, forced=false, native exit_code=-1, wrapper exit0/drained.
Raw parent stdout159bytes содержит только failed nonce proof: safe digest нет.

Config before/after bytes independently equal, reported source/binary pins
unchanged, manual inspection=false, model invocations0. **FAIL сохранён**;
тип/этап RPC failure неизвестен, -1/!forced не называется natural successful exit.
Без raw response/stderr нельзя назвать legacy/profile/protocol причиной.
Native commit/OS-negative/pipeline всё ещё NOT-RUN. EVIDENCE APPROVE означает
точность этой классификации, не acceptance нового permission режима.

## WORK207 — CONTRACT APPROVE закрытого private diagnostic broker

Прочитан актуальный207 с private stderr addendum и separate child exit facts.
**CONTRACT APPROVE207** допускает отдельному QA авторить источники: один broker
под existing generic candidate supervisor; native child в той же owned group,
без new session/group, broker не выходит до known reap/drain. Nonce proof broker
не подменяет native Popen wait status/stdout+stderr EOF. Partial spawn/unknown
даёт manual FAIL, никакого PID adoption/retry или нового grant.

Fixed initialize→initialized→config/read→requirements/read→EOF, без внешнего
RPC pass-through/server-request replies; unknown notifications только bounded
private capture. RPC/parse/oversize/deadline errors не открывают другие методы.
Raw frames2MiB/4MiB/35 и fixed outgoing8KiB, native stderr private≤256KiB;
exclusive0600 под fresh0700 HOME, никаких raw errors/transcripts/stdout/server/Git.
Root safe projection ограничена known fields/types/error codes и expected
profile equality, без secret values/dynamic identifiers. Same argv/env/cwd,
before/after config/source/binary proof, budgets90+30 и native20+7 сохраняются.

Это source-authoring approval, не runtime GO и не accepted permission proof.
Нужны review всех executable sources/pins и отдельный exact root GO на один
config invocation. Product8c не меняется, никаких config writes/permissions
broaden/model/thread/auth/tool/commit/OS-negative вызовов. Reviewer новых
процессов не запускал.

## WORK207 — FINAL SOURCE APPROVE private broker

Прочитаны четыре новых Python modules, reused206 dependencies, exact pins и
README. Manifest `6c5881ea966635304ce8766b18fcff74f284a202acab160a786338ffc2d93fbc`,
pins `5cf70ff4b2a0c6617c7c9f0e5ebba3fb39a731076abdc2db9b1e843947916279`
независимо matched; new/reused module hashes и Pythonbd349 executable совпали.

Native Popen наследует broker PGID, после spawn это проверяется; candidate
generic supervisor владеет группой. Отдельны actual native wait/returncode,
stdout/stderr EOF/drain и nonce broker/supervisor proof. Broker сохраняет
safe provisional/final status; при unknown child/reader quarantines, parent
deadline→lifeline EOF→owned group Stop, не выдумывая native0. Arbitrary detached
descendants не объявляются contained. No new PID/group adoption/retry.

Фиксированы initialize1, initialized, config/read2 и requirements/read3 с exact
cwd/args/environment206; внешнего RPC/pass-through нет, server requests не
обслуживаются. Unknown notifications только bounded private frames, не команды.
Source задаёт native20s+7s reserve и parent90+30. Raw stdout2MiB/frame/4MiB/35frames,
stderr256KiB exclusive0600 и flush, output public только finite codes/stages/counts/
exit facts. Raw exception/config/provider/path values не печатаются. Snapshot/
source/binary guards unchanged, CAPTURED явно не accepted permission proof.

**FINAL SOURCE APPROVE207** на exact6c5881ea. Нужен отдельный exact root GO
на один config invocation; новые процессы reviewer не запускал. Permission
activation/model/thread/tools/auth/config-write/commit/OS-negative/pipeline
остаются вне этого диагностического source scope.

Setup-doc narrow repeat: текущий Windows token программно имеет ровно два
literal backslash; пример содержит по два на каждый JSON separator. **DOC APPROVE**
`automation-worker-setup.md` SHA
`a010cfdc28f37edee7787bbaccfa3d643d2d47f1b348e8229f318abfdc63cb11`.

Лимит300строк для этой review-карточки превышен намеренно: append-only chronology
сохраняет старые REQUEST CHANGES, exact pins и отдельные source/evidence verdicts
длительного TASK014. Это исключение отчёта, не product source file exception.

## WORK207 первый actual run — EVIDENCE APPROVE классификации FAIL

Results `b95deddcb1f0ba56fd96d1bd2f9ec5b77e692671e5970b2079e1589d472b510c`,
broker-safe2d4b106a и safe-proof7bf4592a independently matched. One actual native
same-PGID spawn: initialize response1 observed, failure unexpected_frame_shape
на config_read. Native actual wait известен0 после stdin EOF cleanup, оба
streams EOF/drained; normal dialogue не завершён. Broker exit1, candidate group
proof proven/!success/!forced/exit1, wrapper0. Unknown/manual нет. Model0,
MCP5disabled, config/source/binary pins unchanged; permission proof не получен.

Reviewer отдельно прочитал raw frames только через whitelist projection:
initialize id1/result содержит ровно четыре known fields; frame2 имеет только
method/params/emittedAtMs, без id. Method equality с известным
remoteControl/status/changed=true, metadata type integer. Ни raw params/values,
ни native stderr в output не выводились. Generated installed CLI schema
ServerNotification.json содержит этот method и optional emittedAtMs integer/int64.
207 protocol whitelist не включал эту metadata, поэтому отверг frame до config
response. Это наблюдённая причина207 failure, не доказательство точной скрытой
206 cause. Product parser также ограничивает notification method=configWarning;
будущая compatibility correction требует собственного source review.

**EVIDENCE CLASSIFICATION APPROVE207 FAIL**. Разрешена только дальнейшая
source-only diagnostic correction по отдельному поручению root; никакого
automatic retry/config changes/model/native commit. Старый FAIL сохраняется,
config permission/full pipeline всё ещё не приняты.

## WORK207 V2 — FINAL narrow SOURCE APPROVE metadata

Manifest `672247aa67cc17d8d773c33f2bf6ffde563c75a1263951c29961021e40a6970c`,
pins `ae849b7e2e588dd93d2ca23f263646f0a3fe76f61f19d1949491b07fbc513ea0`
independently matched. Единственная executable delta — protocol3b5b7047;
broker/raw_capture/run byte-identical прежним approved sources. Optional
emittedAtMs разрешён только при notification method без id, integer/not bool,
в signed64 bounds. Replies/server requests/ошибочные типы не получают исключения;
после EOF действуют те же правила. Raw capture/budgets/same-PGID/exit/privacy
и три фиксированных RPC не изменены, никакой authority не добавлено.

**FINAL SOURCE APPROVE207V2** для diagnostic capture. Отдельный exact root GO
на fresh fixture всё ещё обязателен. Старый207 FAIL сохранён, product8c не
исправлялся, accepted permission proof/native execution отсутствуют. Reviewer
новых процессов/config RPC не запускал.

## WORK207 V2 — EVIDENCE APPROVE CAPTURED, не permission proof

Results `cddd2f6555a723661899b693887869440a99e15da1fdda6ef2973fa7aff8abe7`
и broker-safed4898d88/safe-proofc75bf2e6 independently matched. Actual native
same-PGID known natural0, stdout/stderr EOF/drained, три responses/одна notification,
broker/candidate/supervisor success0/proven/!forced. Model0, MCP5disabled,
fixture/pins/config unchanged. **EVIDENCE APPROVE CAPTURED diagnostic only**;
product proof, commit, OS-enforcement и workflow этим не принимаются.

Reviewer safe projection raw capture выполнена без публикации values: expected
default profile и never=true; legacy values null; exact known disabled connector
set=true; requirements=null. SessionFlags selected profile exact intent=true.
Effective selected profile exact-before=false; удаление только15 JSON-null map
entries рекурсивно даёт exact expected equality=true. Ни non-null, array или
empty-map значения projection не меняла. Это прямое наблюдение serialized
shape, не догадка о широком grant или причина ранее скрытого206 payload.

## WORK208 — CONTRACT APPROVE compatibility correction

Согласован узкий proposed delta: optional int64 emittedAtMs только на двух
известных passive notifications configWarning/remoteControl/status/changed;
никаких command/reply/authority действий по их params. Допускается такой же
bounded drain после stdin EOF. Unknown method/server request/reply metadata,
malformed/null/bool/string/overflow timestamp остаются FAIL. Existing framing/
byte/frame/deadline/lifeline/ownership limits сохраняются.

Normalization применяется **только к effective selected profile**: рекурсивно
убрать entries maps со значением null, оставить все non-null values и структуры,
затем полное exact equality с прежним expected profile. False/0/empty string/
maps/arrays не выбрасываются и типы не преобразуются. Altered config/hooks/
network/extends либо meaningful extra/root/unknown grant по-прежнему запрещён.
Raw layer projection/digest, managed requirements, оба proof passes и default-off
owner consent не меняются; overrides/native/global config не расширяются.

**CONTRACT APPROVE208** для separate author204 и авторских unit/integration
regressions по заявленным files. Source review обязателен до отдельного QA,
нового candidate и обычного accepted206 proof. Actual native commit/OS-negative/
full pipeline и capable production activation остаются необходимыми. Reviewer
модели/process/config RPC не запускал и source продукта не менял.

## WORK208 — SOURCE APPROVE узкой compatibility correction

Независимо прочитаны все5 Go files и полный delta относительно8c43df9;
пересчитаны hashes/length каждого файла и canonical fingerprint:
`00e72442007dfc90f3a50d73564b8f513893f11dca586dd68b49331e4ee18b9f`.
**SOURCE APPROVE208**: optional timestamp разрешён только на двух passive
notification methods, int64/null/bool/reply metadata/unknown actions остаются
строгими; tail после stdin close использует тот же35-frame/byte budget.
Copy-only normalization касается исключительно effective selected profile,
удаляет только null map entries и сохраняет arrays/types/non-null значения.
Exact expected equality, raw layer digest, managed constraints и остальные
authority/launch/default-off guards не менялись. Новые regression fixtures
не alias raw layers и проверяют15 null defaults, meaningful extras/altered
grants, digest sensitivity, malformed metadata и точный frame ceiling.

Author reports full race worker12.705s/IPC1.277s/cmd2.853s и Linux/Windows
cross-build PASS. Reviewer initial independent pure-unit repeat не начался:
`go` отсутствует в shell PATH (exit127); путь toolchain запрошен автору.
Это не отрицательный test result продукта. Native/config RPC/model/process QA,
actual Windows и весь pipeline reviewer не запускал. Новый candidate и обычный
accepted206 runtime proof ещё обязательны; source approval не runtime GO.

## WORK203 — полный SOURCE REVIEW: REQUEST CHANGES

Независимо прочитаны17 CJS modules, Python fixture/tests/real JUnit runner,
README/plan/pins и owning product DTO/API/auth paths. Все28 hashes manifest
`.local/task014/pipeline203/source-pins.json` совпали; manifest SHA
`a19c28c26bf1412cf06e63cfedfc65a606f9d2f6569dee86221a36874bb3eecb`.
Никакой module main/import, fixture, setup/browser/server/MCP/model/config RPC
или process execution reviewer не выполнял. Unconditional EXTENSION_PENDING
сохраняет runtime закрытым;8c pins требуют будущего repin208 и нового review.

Положительно: public205 preview с explicit owner apply; реальные UI profiles,
policy/intake и scripted stdio MCP stable same-action replay; report cursors;
четыре различных bound native sessions/runs/roles; task/check/run_exit/ACK tuples;
один Git commit с frozen tests/helpers; настоящий unittest-derived JUnit,
config/source before-after hashes и private evidence. Replay не выдаётся за
пятый native агент, single Mac не выдаётся за Windows/второй физический ПК.

**REQUEST CHANGES203**, три блокирующих source findings:

1. **P1, coordinator.cjs146–152 / setup.cjs44–65:** fixture создаёт только legacy
   token, затем cleanup передаёт его raw tokenID в `projects token-revoke`.
   Owning `internal/projects/auth.go47–54` импортирует legacy identity без
   credentials row; `RevokeCredential`118–129 обновляет credentials и требует
   ровно1row. Поэтому happy cleanup всегда FAIL до legacy revoke. Исправить
   на фактическое legacy-only revocation либо отдельно создать и сохранить
   catalog credential/её правильный ID; не заявлять несуществующую revoke.
2. **P1, owned.cjs31–41 / mcp.cjs24–28:** readline собирает неограниченную line
   до проверки8MiB; private stdout/stderr logs пишутся без total cap. Это не
   bounded output: возможны неограниченная allocation/диск и unhandled callback
   errors до финального report. Нужны byte-chunk frame/total/log limits перед
   накоплением, terminal FAIL и controlled EOF/owned cleanup при overflow.
   Аналогично HTTP/API response.text проверяется только после накопления body;
   shared bounded reader должен ограничивать bytes во время чтения.
3. **P1, coordinator.cjs120–157 / owned.cjs11–14 / gate.cjs:** Promise.race
   ограничивает ожидание, но не отменяет/join cleanup continuation. Вечный
   cleanupToken продолжает разрешать commands/HTTP после60s и сохранения final
   report; late operations могут менять state/report уже после результата.
   Нужны общий cleanup deadline/fence, отмена pending readers/UI/HTTP и join
   либо явно retained unknown/manual с запретом новых side effects после fence.

Findings переданы root и source author. Автор подтвердил понимание; исходный
a19 manifest immutable, reopening требует отдельного ownership root. Gate
остаётся закрытым. Source review ещё не production approval и не разрешение
нового runtime; независимый rereview нового frozen source обязателен.

## WORK208 — initial unit PASS; final candidate HOLD; addendum CONTRACT APPROVE

Через repo toolchain `.local/toolchains/go/bin/go` и private0700 TMPDIR
независимо повторены5 новых pure mock/file compatibility regressions с race:
**PASS1.663s**. Первоначальный shell exit127 был отсутствием PATH, не тестом.

До candidate root выявил ещё один effective key из существующего207V2 capture:
`include_permissions_instructions:true`, который прежний permission-key guard
отклоняет. Initial00e724 SOURCE APPROVE выше относится только к проверенному
notification/null-normalization delta; **final candidate/runtime HOLD** до
нового frozen manifest и независимого review дополнения. Новых native calls
для этого анализа не запускалось, старые sources/evidence сохраняются.

**CONTRACT ADDENDUM APPROVE208**: точное имя допускается только при boolean true,
null/absent сохраняют прежнюю absent-default обработку. False/любые другие
типы и unknown похожие permission keys остаются FAIL. Explicit true из active
layer обязателен в raw relevant projection/digest; effective grant exactness,
managed constraints, no-overrides/default-off/две проверки не меняются.
Reviewer проверил первичные [ConfigToml](https://github.com/openai/codex/blob/main/codex-rs/config/src/config_toml.rs)
и [packaged defaults](https://github.com/openai/codex/blob/main/codex-rs/config/defaults.toml):
это добавление developer permissions-instruction block, default true, а не
новый OS write grant. Approval разрешает узкий author source delta/units после
root ownership GO; не распространяется на новый process QA или запуск моделей.

## WORK209 — PLAN REVIEW, один narrow P2 до final approval

Прочитан plan-only WORK209 SHA
`b3740605aca4936515cb97d263095a88d35b07ec5b4d7ff7dc27121eb0dd7a2a`;
hash независимо совпал. A ordinary accepted product proof → C direct OS
positive/negative models0 → B ONE actual native commit — допустимый порядок
с отдельными source/evidence/root GO. Max1model/510s total, exact shared
profile/env/cwd, second matching config proof, real EPERM/EACCES plus ordinary
DAC-control, outside-temp sentinel, no fallback/hidden retry и owned wait/drain
limitations сформулированы корректно. Foundation не выдаётся за four-role pipeline.

**P2 до PLAN APPROVE**: B требует branch/refs unchanged, но ordinary-branch
commit обязан изменить current branch ref. Уточнить единственный разрешённый
переход `refs/heads/<current>` seed HEAD→exact new HEAD; symbolic HEAD/имя branch
и все остальные refs/remotes/config/hooks сохраняются. Это correction oracle,
не ослабление single-commit proof. Root и QA уведомлены; root принял finding.
Новый plan hash/review ожидаются, harness authoring до final208/source/root GO
не разрешён. Reviewer CLI/help/sandbox/model/config/process не запускал.

## WORK209 — PLAN APPROVE после narrow P2

Независимо проверен новый SHA WORK209
`d2dc083a0842748290acf977a78737c6dd52f01e27dbb4f5f022d559c89db9ef`.
B теперь сохраняет symbolic HEAD/branch name, допускает единственный заранее
bound current branch ref seedHEAD→exact newHEAD; все другие refs/remotes/config/
hooks immutable. Detached fallback отсутствует. Root порядок A→C→B и отдельные
source/evidence/GO закреплены. **PLAN APPROVE209**, только план; authoring после
final SOURCE APPROVE208, любой runtime после отдельного SOURCE review и root GO.

## WORK208 — c3d addendum SOURCE REQUEST CHANGES, P1 совместимости

Проверены все5 source hashes/length и fingerprint
`c3d8785329db52dd4f2891d2543330a6f1228f2f65198742e55565d0d94a42d5`;
от initial00e724 изменились только permission_projection.go и normalization
tests. Exact known key true-only/null-absent handling и active raw projection/
digest реализованы верно. **SOURCE REQUEST CHANGES208**, финальный HOLD сохраняется.

**P1:** новый blanket `strings.HasPrefix(lower,"include_")` reject в
permission_projection.go отклоняет четыре legitimate non-null поля effective
config из exact207V2 capture: include_apps_instructions,
include_collaboration_mode_instructions, include_environment_context и
include_environment_context_time. Reviewer independently прочёл bounded private
capture и вывел только typed boolean predicates заранее известных official
ключей; все4 присутствуют и true. Raw contents/другие values не опубликованы,
нового native invocation не было. Official defaults, уже проверенные выше,
содержат те же поля. До addendum эти unrelated non-permission fields guard
игнорировал; новый blanket reject снова гарантирует native proof failure.

Рекомендация root: уточнить контракт, сохранить прежнюю обработку unrelated
include_* fields; убрать только generic prefix rejection, оставив exact
include_permissions_instructions true-only и прежний strict permission/sandbox/
approval-key guard. Unknown похожие permission keys остаются FAIL, общая
allowlist/grant exception не требуется. Соответствующую negative unit generic
include_unknown_flag заменить проверкой прежнего поведения unrelated fields,
добавить actual-shape synthetic fixture четырёх known fields. Root и author
уведомлены; c3d freeze сохраняется до нового approved source delta/manifest.

## WORK208 — CONTRACT clarification APPROVE после observed-shape P1

Root уточнил неоднозначное unknowninclude_* wording. **CONTRACT APPROVE**:
удаляется только новый generic include_ prefix reject; unrelated non-permission
settings сохраняют прежнюю обработку. Exact include_permissions_instructions
true-only/null-absent и explicit active-layer raw projection/digest остаются.
Неизвестные permission/sandbox/approval names по-прежнему FAIL. Обязательны
pure synthetic regression четырёх наблюдённых штатных true fields и negatives
похожих permission keys; новый frozen manifest/rereview до candidate. Это
исправление compatibility regression без override/grant/API expansion. Runtime
закрыт; c3d tests, даже PASS, не снимают выявленный actual-shape blocker.

## WORK210 — PLAN APPROVE remaining native/control cases

Прочитана полная221-line plan-only карточка WORK210; SHA независимо совпал:
`870523cf1735a80a816be1e7c5e5fdf43ebc7f664145f71e9ca315465d1a68ad`.
**PLAN APPROVE210**, не executable-source/runtime approval. F6 primary genuine
review-negative route сохраняет independent reviewer decision и обе Git/SHA
истории; Q7 optional без навязанного approve/подмены tests/parent outcomes,
NOT-EXERCISED выделяется отдельно. Exact budgets6/7/2/1/2 invocation, без hidden
continuations; setup/workflow/cleanup и policy/local ceilings явно ограничены.

C2 требует positive owned stop+ACK/lock release до exact native UUID resume,
fresh board session/generation/epoch и canary observed native delivery. H1
требует real native question и authenticated designated-human answer без auto
resume, immutable history/idempotent receipt. B2 использует разные credentials/
contexts/actions, настоящее local-owner controller grant/revoke и distinguishes
forbidden от offline/stale rejection; single physical Mac так и обозначен.
Always-save private evidence, config drift FAIL/no restore, bounded cleanup и
203 P1 correction обязательны. Applicable final208/normal206/native commit/
OS-negative prerequisites ещё закрываются независимо.

При отдельном SOURCE review проверять достижимость live-pause checkpoint,
защищённое capture реально выданного gen1 credential и точный owning auth-check
order для B2. Плановые ожидаемые tuples не становятся runtime proof. Ни новых
fixtures/source/harness, ни browser/HTTP/MCP/CLI/config/model execution reviewer
в этой PLAN проверке не создавал/не выполнял. Root и plan author уведомлены.

## WORK208 — FINAL SOURCE APPROVE10cd после clarification

Независимо проверены hashes/length всех5 Go files, canonical fingerprint
`10cd3bbc4754ceb675f0169e94e9bc74ec3721a2f454ee4d298c6d8c64e0b3c4`.
От c3d изменены только permission_projection.go и его normalization tests;
historical00e724/c3d manifests сохранены отдельно. **FINAL SOURCE APPROVE208**:
generic include_ rejection удалён, unrelated fields сохраняют old behavior.
Exact include_permissions_instructions true-only/null-absent и raw active-layer
projection/digest неизменны; unknown permission/sandbox/approval names остаются
FAIL. Notifications/frame limits/effective selected-profile null normalization
и все прежние proof/authority/default-off guards не менялись.

New unit независимо разделяет effective/layer maps, воспроизводит четыре
observed known true fields и generic unrelated metadata types, сравнивает exact
baseline proof. Unknown similar permission/sandbox/approval names отрицательные;
false/wrong-type known flag отказаны. Независимые pure mock/file race checks:
compatibility regression group **PASS1.615s**, отдельный новый unrelated-field
test **PASS1.334s**. Scoped diff-check PASS. Author full worker/IPC/cmd race и
Linux/Windows cross-build PASS отмечены как авторские, не native Windows proof.

P1 c3d compatibility blocker исправлен, final source HOLD снят только для этого
exact source set. Новый clean candidate и ordinary accepted product206 proof,
direct OS-positive/negative209C, actual native commit209B и отдельно203 SOURCE/
runtime всё ещё обязательны. Никаких native/model/config RPC/processE2E reviewer
не запускал; product source/harness не менял. Root и author уведомлены.
