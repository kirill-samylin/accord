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
