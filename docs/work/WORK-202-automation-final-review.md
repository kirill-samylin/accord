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
