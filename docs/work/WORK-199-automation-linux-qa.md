# WORK199 — отдельная Linux QA schema6 migration/snapshot/restore

23.10.2026. Отдельный tester после composite SOURCE APPROVE189
(185/186/187/191/193 exact pins). Scope: только эта карточка и игнорируемая
private `.local/task014/linux-qa199/` (0700). Общей TASK владеет root.
Production TASK013273, GUI, соседние QA fixtures, workers и provider settings
не изменяются. Source harness не выполняется до169 review и отдельного root GO.

## Исследование и выбранная граница

Прочитаны requirements/architecture, automation/TASK014, WORK189/191/196/197,
actual helper sources и предыдущие TASK011/013 fixtures. Production updater
`projects/upgrade.py` и host/launch primitives используют канонические
`/opt/accord`, `/var/lib/accord`, systemd unit names, active pointers и7892/7893.
На этом VPS настоящий updater невозможно изолировать простой передачей другого
root: эти paths не параметры. Подмена systemd/Next callbacks из старого
Environment дала бы simulated activation, а не production-like proof.

Поэтому этот bounded harness **не вызывает** upgrade apply/finalize, host helper
install, staging, systemctl, active pointers, реальные DB/services или native
агентов. Будет проверять real Linux Go CLI `projects migrate-storage`, unchanged
format2 snapshot/restore helpers и actual candidate maintenance-reset внутри
нового private synthetic root. Systemd/Next activation остаётся **NOT-RUN** и
будет отдельным этапом, не скрывается за CLI PASS.

Разрешённое readonly исследование через pinned transport подтвердило Linux
x86_64 / Python3.12.3, private `/root` и ~13.3GiB свободно. Из immutable273
прочитан только executable hash/capabilities: SHA256
`5a1e1352895c20ec976feef20cc60328aa9ba19834834e75d3b1e5a46ca7dd60`, schema5 и
managed_agents1 вместе с прежними floors. Реальные DB/env/credentials не читались.
WORK196 baseline4active/9runtimeDB/journalcomplete используется как контекст,
не как якобы повторная live DB проверка этим tester.

## Candidate и source plan

Root сообщил frozen clean commit `3e6ac81ee81909e3c33a2c86e593f4781357878d`,
Linux binary SHA256
`6c259cd672b5c56b89c625b159c233cafb1ecbd2487a739cf1826efbf71d072f`, bundle
manifest SHA256 `7f02d9b117d43373a7e307672d010e372d1ae329b33d77f09d659d5f17d796c7`.
Clean candidate module для будущей сборки typed fixture generator находится
в root-managed worktree; tester tracked files там не меняет. Сборка только
после source review/root GO, исходник — private `fixturegen.go`.

Новые QA sources, каждый до300 строк:

- `run_linux.py`: строгий Linux/owned-root/input/hash/GO guard, реальная CLI
  миграция, snapshots/restore и pin checks. Ни transfer, ни setup production нет.
- `seed.py`: через actual prior273 CLI/loopback HTTP создаёт собственные4active
  проекта/9runtimeDB, tasks/notes/profile revisions/живые leases и managed dispatch
  без worker/provider запуска. Synthetic token только в своём root, clean child
  environment, proxies/redirects отключены.
- `fixturegen.go`: компилируется внутри frozen candidate module. Выдаёт canonical
  Go DTO JSON для девяти automation tables, включая unfinished/terminal workflows,
  plan/question/attestation history, capability и unknown workspace reservation.
  Это **offline typed storage fixture**, не real pipeline или native proof.
- `oracle.py`: собственные direct SQLite row/byte/schema/index проверки до/после;
  не доверяет только helper success flag или exit0.
- `lifecycle.py`: execution deadline360s, SIGALRM/SIGTERM/SIGINT через finally,
  только зарегистрированные до constructor/readiness owned child handles,
  bounded terminate8s/kill3s и явный exit postcheck. Pin errors на missing/changed
  files собираются в FAIL, не прерывают попытку durable report writing.

Reuse только unchanged reviewed
`tests/process/task011-update/fixture.py` для child ownership/readiness/CLI
и candidate infra snapshot/restore dependencies. Source manifest фиксирует
каждый Python helper, эту fixture и все5новых QA sources до/после. Child env
allowlist подставляется явно; operator/store поведение не mock/patch.

## Bounded будущий запуск

1. После169 source review/root GO — новый `/root/accord-task014-qa199-<unique>`
   (0700), собственные inputs/harness/helpers/binaries/fixturegen, ни symlinks,
   ни production data. Old executable копируется read-only из immutable273;
   candidate из проверенного clean bundle. Exact SHA проверяются до/после.
   `inputs.json` и `root-go.json` (0600) должны bind candidate revision/SHA,
   reviewed source-pins digest и root/169 approval. Повтор существующего run
   root/state/report запрещён; unknown outcome не получает слепой retry.
2. Actual schema6 capability guard отказывает old schema5 executable **до DDL**.
   Actual old CLI seeding4project/9DB завершается; writer останавливается своим
   child PID. Произвольные реальные listener/PID не останавливаются.
3. Unchanged snapshot helper создаёт и validates format2 pre-snapshot всех4
   schema5 проектов. Actual candidate CLI migrates4boards до6. Каждый старый
   table row во всех9runtimeDB сравнивается byte-exact; отдельно проверяются
   schema6 automation metadata/tables/exact UNIQUE JSON indexes,4durable schema5
   migration DB+checksum и неизменность pre-snapshot.
4. Actual old273 CLI против **своей** migrated fixture должен отказать с
   конкретной unsupported-schema причиной; bytes/logical rows остаются прежними.
   Ни listen, ни fallback/rollback не используются.
5. Только offline в своей schema6 fixture вставляются canonical typed synthetic
   automation history/authority rows во все4boards, сохраняются synthetic
   blob/staging bytes. Unchanged helper создаёт schema6 format2 snapshot.
   Эти rows намеренно имеют synthetic domain-invalid references,
   `GitPolicy=isolated` и bindings только одной роли. Их Go decoding корректен,
   но они **не являются действительной live policy/workflow authority**.
   Проверяется storage reset/retention, не разрешение исполнения или pipeline.
6. Actual restore helper в **новый offline sibling root** вызывает настоящий
   candidate reset. Independent oracle проверяет credentials/proofs/caps revoked,
   новые registry signing key/ACL/host/workflow epochs, cleared session/action
   authority, cancelled dispatch, unknown process needs_recovery; reservation,
   policies revisions, steps/plans/questions/attestations и immutable legacy
   history сохраняются. Terminal workflows неизменны; exit не выдумывается.
7. Snapshot и source fixture bytes неизменны. Boot guards остаются, restored root
   нигде не служится и не auto-launch. Все процессы ограничены owned fixture;
   приватный report содержит hashes/counts/status, без DB rows/секретов/provider
   output. Внутренний execution deadline360s оставляет reserve на bounded
   owned-child cleanup/report внутри внешнего420s. SIGTERM/INT запускают тот же
   finally; source-pin failures сохраняют FAIL report. Polling/yield≤30s.

## Статус

**SOURCE PREP / REVIEW PENDING.** Новые harnesses пока **NOT-RUN**; transfer,
fixture writes на Linux, сборка fixturegen и migration/restore не выполнялись.
Root candidate fullchecks и197 existing acceptance PASS — отдельные evidence,
не результаты этого нового harness. Physical second PC/Windows/Claude,
автоматизация pipeline и production schema6 rollout этой карточкой не закрываются.

## Узкий REQUEST CHANGES169 — execution readiness

Reviewer указал, что одного обещания внешнего420s недостаточно: default SIGTERM
мог обойти finally и оставить собственный fixture server, а ошибка pin check
в finally — не дать сохранить FAIL report. Добавлен описанный `lifecycle.py`,
exact handle ownership и exit proof; timer/signals и source/binary-pin ошибки
обрабатываются отдельно от основного результата. Typed fixture limitations
уточнены без объявления её domain-valid. Это source correction, **не runtime
PASS**. Повторное169 review/root GO ещё требуются; executions/compile/transfer нет.

## Final source approval и единственный actual Linux run

169 **FINAL SOURCE APPROVE** пяти source files: manifest SHA256
`d02e935426f79c4dff7de76bb96eddb5767c3a8a02978207bbf6499ba90ab240`,
lifecycle `33d7ffbb…`. Последние уточнения: незавершённый constructor без
наблюдаемого Popen handle считается unknown/manual-recovery FAIL; manual flag
также true для известного, но не завершившегося child. Ложного no-process proof нет.

Получен отдельный **ROOT GO ONE bounded execution**. В clean candidate module
собран только synthetic fixture generator в разрешённой ignored папке;
tracked worktree остался clean. Generator SHA256
`9a3b37f194321a15f803574cd43e7e49be57228b33035b02198d4c8b055b4a59`.
Это cross-build вспомогательного storage DTO инструмента, не native AI proof.

В новый private Linux root передан проверенный package63files/40sourcepins.
Candidate/old binary hashes совпали с указанными выше. Expanded source-pins
digest `4207ac604dfad20e0a564e4f11fe0066be44e2da4327859fc90aa4deeb0da24d`.
Запуск выполнен **ровно один раз**, при FAIL не повторялся и не обходился.

Результат — **FAIL на подготовке `old_cli_seed` / HTTPError**:

- **PASS:** actual schema6 capability guard отверг actual old273 executable до DDL.
- Synthetic old CLI создал4active проекта/9runtimeDB, один собственный catalog
  credential и два editor grants. Полная history seed успела только в default
  board:2tasks/profile/managed host+dispatch.
- **NOT-RUN:** candidate schema5→6 migration, pre/post format2 snapshots,
  schema5 historical-copy proof, actual old CLI declared6 rejection и full restore.
  Readonly postcheck подтвердил snapshot_count0 и отсутствие restored root.
- Оба собственных serve child завершились graceful exit0, forced=false;
  all_owned_children_exited=true, manual_recovery_required=false,
  cleanup_failures0, outer_timeout=false. Никакие native agents не запускались.
- Source pins и оба binary hashes после запуска неизменны. Дополнительный
  independent readonly postcheck40pins также PASS. Original HTTP status/body
  не сохранялись: actual HTTP code **NOT-RECORDED**, report не выдумывает его.

Report SHA256:
`bee44e2258bcb1c3d2c56edbf1f5c29e3d7200df210496ebf3b8c2fc86cd0ccf`.
Private local evidence:
`.local/task014/linux-qa199/run-1792770475854420000/` — package/compile provenance,
execution-summary/results, readonly failure diagnosis/postcheck. Собственная
Linux fixture сохранена для review; автоматический cleanup данных/retry не делался.

## Readonly разбор остановки

Source seed открывает scoped nondefault `/tasks` без явного catalog project-open
action. Owning273 BrowserRouter требует existing ProjectBinding и при его
отсутствии возвращает PROJECT_SESSION_REQUIRED; default получает binding при
login. Readonly SQL своей fixture подтвердил default browser_bindings1,
остальные3boards —0, и остановку после второго grant. Это указывает на пробел
seeding harness перед nondefault page read, **не является runtime доказательством
дефекта нового candidate**. HTTP409 — ожидаемый по owning source ответ, не
наблюдённый/сохранённый статус первоначального HTTPError.

Новый retry потребует узкого source fix, повторного169 review и нового root GO
в fresh fixture. На этом checkpoint harness source не менялся. Linux migration/
restore acceptance остаётся **незавершённой**; production schema6 и TASK014 PASS
не заявляются. Production data/services/pointers/helpers/GUI не изменялись.

## V2 source-only seeding correction — review pending

Root разрешил узкую доработку нового harness после FAIL; первый Linux root,
report, package и пять исполненных source pins сохранены неизменными.
V2 находится отдельно в private `.local/task014/linux-qa199/v2/`.

По owning273 `catalogHTTP → switchProject → OpenProject` выбран публичный
project-open flow. Перед каждым scoped task page GET теперь запрашивается
fresh catalog CSRF + server-issued `switch:<project>` action, затем POST
`/web/catalog/switch` с явными project_id и scoped target. Success/redirect
проверяются, и только после этого выполняется GET с той же browser cookie.
Проектный binding не создаётся direct SQL; ACL/CSRF/epochs остаются серверными.
Так все4boards должны получить настоящие browser sessions до migration/reset.

В failure report добавлен только безопасный observed HTTP status, если exception
его содержит; response body/URL/credential не выводятся. Код первого HTTPError
остаётся NOT-RECORDED и не восстанавливается догадкой. Остальные migration/reset/
authority/history assertions и источники без изменений.

V2 AST PASS; **runtime/transfer/compile NOT-RUN**, новые202 source review и root GO
обязательны до второго fresh run. Это исправление harness, не product fix и
не доказательство готовности Linux acceptance.

V2 delta теперь проверяет отдельный reviewer WORK202 поверх исходного approve169.
Чтобы marker правдиво указывал reviewer именно исполняемого V2, его exact
`root-go.json` ожидает `reviewed_by: "202"`; исходный D02 marker/source первого
run остаётся169. До explicit SOURCE APPROVE202 и нового root GO выполнение V2
по-прежнему запрещено.

Повторная процедура подготовлена только как порядок действий: новый private
local V2 run и новый Linux root; никаких перезаписей первого execution.json,
package/source/report. Typed fixture generator source не менялся, поэтому
можно использовать прежний exact SHA-проверенный helper с прежним clean-module
build proof. Source package/pins собираются заново из V2 и frozen candidate
infra, root GO marker связывает202/new expanded pins/candidate SHA. Один запуск,
внутренний360s/cleanup reserve и внешний420s; при unknown outcome — сохранение
fixture/manual FAIL, без blind retry. Перед выполнением нужны свежие root
подтверждения candidate revision/SHA и hash всех пяти V2 sources.

## V2 actual Linux run и V3 source correction

202 SOURCE APPROVE V2 `ddabcc13…` и отдельный root GO получены. One fresh run
на exact3e6/6c259cd6, без новой сборки primary source: **FAIL** на assertion
`actual_old_binary_refusal`. До него4PASS: полный actual old273 seed4boards/9DB,
format2 schema5 snapshot, real candidate schema5→6 и точное сохранение всех старых
logical rows +4durable schema5 migration copies/checksums. Independent readonly
postcheck подтвердил9runtimeDB/4schema6boards/4historycopies, old rows unchanged,
source40pins unchanged и snapshot manifest/files unchanged.

Оба owned serve child graceful exit0, forced=false; all_owned_children_exited=true,
manual_recovery_required=false, cleanup_failures0, outer_timeout=false. Source и
binaries unchanged. Restore и typed automation insertion **NOT-RUN**. Report SHA256
`54b6fa27f2ca578f6b3aee68b0962da4348de3c2f7aa363e7e37babd85aa9d54`.
Private local evidence: `v2/run-v2-1792771499140621000/`. Первый run и V2 сохранены;
никакой автоматический retry или изменение production не выполнялись.

Причина assertion mismatch подтверждена owning273 source: `cmd/agent-board/projects.go`
возвращает публичное `offline storage migration failed; keep startup guard and inspect`,
а `main.go` пишет `agent-board: <error>` и exit1. Harness ошибочно ожидал внутреннее
`published board schema unsupported`. Конкретный returncode/stderr самого rejection
в V2 не сохранялись; по ним нельзя задним числом объявить observed PASS.

Root разрешил source-only V3 отдельно в `.local/task014/linux-qa199/v3/`.
Меняется только `run_linux.py`: перед old CLI повторно проверяются exact old/new
hashes, реальные old capabilities schema5 со всеми прежними floors, capable
candidate automation1+previous floors и actual declared schema6 всех4boards.
Проверяется документированное публичное CLI сообщение + exit1 + пустой stdout;
byte-exact всех state files и exact logical rows до/после остаются обязательны.
Observed exit/stdout-empty/stderr-class/hash и обе equality записываются в report
до assertions; raw stderr/DB/user data не выводятся.

V3 требует самостоятельного202 review и нового root GO. Пока source-only:
новых compile/transfer/execute нет. Все предыдущие FAIL остаются в истории,
а Linux restore gate и полный TASK014 acceptance ещё не закрыты.

## V3 actual refusal и отдельный source-pending first-open diagnostic

V3 получил202 SOURCE APPROVE `c552c002…` и отдельный root GO. One fresh run
exact3e6/6c259cd6 снова **FAIL** на mandatory all-state byte equality. До него
4migration checks PASS. Теперь сохранены actual old CLI facts: exit1,
stdout_empty=true, stderr_class=documented_public_migration_refusal,
stderr SHA256 `5577ba45e12b0e7dfc6392ad8e647a1c3cde2763450deb2934af2d2b433b43d3`;
all_logical_rows_unchanged=true, all_state_bytes_unchanged=false.
Original all-byte assertion не ослаблена и остаётся FAIL.

Снова оба own serve child exit0/forcedfalse; all_owned_children_exited=true,
manual_recovery_required=false, cleanup_failures0, outer_timeout=false;
source40/binaries unchanged. Report SHA256
`e0b60998147a34bd056eaea5dbfe7388aaac6f066ffe32fc3c8afaa5a2cba78b`,
private evidence `v3/run-v3-1792771968080981000/`.
Independent readonly postcheck сохранил9DB/4schema6/4schema5history copies,
legacy rows/source/snapshot unchanged. Все найденные WAL0; SHM32768. Конкретные
before/after path maps V3 не сохранял, поэтому названия изменившихся файлов
**NOT-RECORDED**. Mode=ro в owning273 preflight и SQLite WAL semantics дают
возможный механизм, не доказательство того, какие bytes изменились тогда.
Restore/typed automation insertion остаются NOT-RUN.

Root поручил отдельный narrow diagnostic source, без автоматического retry
harness. Private `diagnostic/probe.py` + `support.py` (каждый до300строк):

1. Exact retained V3 root/report SHA, original40source pins, candidate/old binary
   hashes и previous proven child exits проверяются до действий. Original state
   и backup сохраняют полный private path→SHA/size map до/после всего probe.
   No-open-handle census смотрит только fd targets собственной fixture; не
   читает чужие argv/env и не ищет процессы по именам/PID для остановки.
2. Original никогда не запускается повторно и не очищается. После9DB integrity/
   schema/floors и WAL/journal0 proof создаётся **новая собственная derivative**
   copy, исключая только SHM и нулевые WAL. До SQL exact copied DB/file bytes
   сравниваются с original. Неопознанный nonempty journal останавливает probe.
3. Root отдельно разрешил единственное metadata-only SQL исключение в копии:
   update только `registry_meta.default_db/default_web` на canonical clone paths,
   с old-value WHERE и rowcount1. Вся строка и все9DB logical tables/schema
   сравниваются; кроме этих двух полей изменения запрещены. ACL, tokens,
   epochs, policies, marker/floors и остальные файлы не меняются. Собственные
   нулевые clone WAL/SHM могут быть удалены только после close/no-open-handles/
   integrity/logical-equality proof; original sidecars не трогаются.
4. Новый before baseline сохраняется после rebind. Exact old CLI `projects
   migrate-storage` применяется один раз к clone через existing candidate
   `worker supervise`: fresh nonce + parent-stdin lifeline, Proven=true,
   expected ExitCode1, forced=false и natural helper exit0. Это ownership helper,
   не worker enrollment/provider/native AI запуск. Helper намеренно discards
   stderr; probe не подделывает новый stderr receipt, прежний V3 receipt сохранён.
5. After path maps сохраняют **все** изменения и классифицируют SHM, нулевой WAL,
   nonempty WAL, DB/migration/other bytes. Все9DB schema/logical facts,
   original snapshot/history и source/binary pins повторяются. DB/nonempty WAL/
   immutable content mutation означает FAIL, не исключается из сравнения.
   Classification aux служит диагностике; исходный V3 all-byte FAIL сохраняется.
6. Timer18s + bounded8s lifeline cleanup внутри внешнего30s. SIGALRM/TERM/INT
   unwinds finally, repeats masked. Missing/unproven exit — manual recovery FAIL,
   данные остаются. Ошибки source-pin/readback не отменяют попытку report.
   Результат `DIAGNOSTIC_COMPLETE` означает завершение этого эксперимента,
   не PASS migration/restore acceptance и не реконструкцию отсутствующего V3 map.

Новый probe пока **SOURCE-ONLY / 202 REVIEW PENDING**: AST PASS; transfer/run
не выполнялись. Derivative rebind — только test-fixture adjustment, не supported
production restore и не разрешение на SQL изменения рабочего сервиса. Нужны
отдельные SOURCE APPROVE202 и root GO с exact probe manifest до исполнения.

202 запросил узкую provenance correction до approval: текущие config map+files
нельзя считать сохранёнными, если они изменились согласованно. Source-only
diagnostic V2 в отдельной `diagnostic/v2/` теперь связывает canonical sortedJSON
40-pin aggregate с accepted V3 report (его exact SHA e0b60998… проверяется),
известным expanded pin203f7c8c… и exact original oracle99d1d9fe…. Hash исходного
inputs.json фиксируется и повторяется после probe. Report сохраняет только
безопасные provenance hashes/count. Первый diagnostic source dfb93ecc… сохранён;
новый probe всё ещё не переносился и не выполнялся.
