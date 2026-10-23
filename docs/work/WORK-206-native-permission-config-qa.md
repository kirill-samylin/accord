# WORK206 — actual candidate permission-config QA, source preparation

23.10.2026. Отдельный QA. Root поручил только SOURCE PREP после final
SOURCE APPROVE204202, fingerprint23:
eea3f68a765ec122278cda976774493cbc0859e5ea22ee947f3f98cce53c9714.
Независимый reviewer202; выполнение только после отдельного source approve
и exact root GO. **NOT-RUN: compile/process/config RPC/native model.**
Pipeline203 приостановлен и сохранён; результаты197/198/195/196 не меняются.

## Объём и границы

Private sources `.local/task014/qa206/`: run.py, owned.py, fixture.py,
inspection.py, config_snapshot.py, identity/main.go, pins.json, README.
Каждый source файл до300 строк. Entry hardstop MISSING_EXACT_NEW_PIN до любого
mkdir/spawn, пока отсутствуют новые candidate revision/path/SHA, compiled helper
SHA и полный source/harness pin set. Старому3e6 новый inspection не приписывается.
Go helper ещё не compiled и вызывает только существующий exported
worker.AutomationIdentity для обычной fixture .git; production CLI не расширяет.

Fresh private0700 HOME diagnostics root, clone layout с ordinary .git, empty
hooks/template, no remotes и local-only identity; один setup seed commit не
является native commit. Candidate/helper byte-identical copies под HOME.
Настоящие HOME/CODEX_HOME/native auth сохраняются; Git redirect environment
отвергается кроме безвредного GIT_PAGER, fixture calls используют --no-pager.
Все Git/setup/identity children supervised тем же candidate; нет broad kill.

## Exact approved204 invocation

1. Read-only actual Codex `mcp list --json` ≤5s, private inventory.
2. Derive Git identity через exported helper. Profile = accord_git_ плюс первые
   16bytes SHA256(canonical directory + NUL + Git identity), hex32chars.
3. Args совпадают с newConfigInspection: default_permissions=profile;
   permissions={profile={extends=":workspace",filesystem={exact.git={".":write,
   config:read,hooks:read}},network={enabled:false}}}; approval_policy=never;
   whole mcp_servers map с сортированными quoted keys и transport-complete
   disabled tables (stdio command=actual native,args=[]; HTTP loopback port1).
   Нет sandbox_mode, broader permissions или global config changes. ASCII-only
   paths/names дают точное quoted Go/TOML представление; иные fail closed.
4. Actual `mcp list --json` ≤5s на exact args подтверждает тот же inventory,
   все connectors disabled; managed connector не добавляется.
5. **ONE** candidate `worker supervise` с config_inspection и native args
   `app-server --listen stdio://`. Fields: directory, profile,
   executable_sha256, git_identity, sorted connectors (null если inventory пуст).
   Parent не передаёт RPC. Закрытый candidate helper сам выполняет initialize,
   initialized, config/read{cwd,includeLayers:true}, configRequirements/read,
   closes stdin, затем positive natural owned exit0. Actual safe permission
   digest должен прийти от candidate, не echo intended flags из родителя.

Helper сохраняет существующий20s deadline +7s cleanup. Outer bound30s нужен
для наблюдения этого конечного cleanup, не увеличивает candidate deadline.
Затем только EOF/reap/drain reserve. Unknown ownership => FAIL/manual inspection,
никаких следующих contexts/native calls/retries. Unix CLI+owned PGID scope,
не гарантия arbitrary detached descendants.

## Доказательства и cleanup

Сохраняются before/after **байты** config.toml/requirements.toml/
managed_config.toml private0600 и hashes, а не только before hash в памяти198.
Diff выводит лишь known top-level key names/booleans; dynamic project paths,
profile/provider/connector names и raw values не публикуются. Trust-only metadata
отдельно классифицируется и не объявляется permission expansion или авторством
пользователя. Любой config drift остаётся отдельным FAIL; автоматического
restore нет. Raw config RPC responses не выходят из helper.

Candidate natural/proven exit0/!forced и exact executable/Git identity binding,
new safe digest, disabled inventory, все input/source/binary hashes до/после,
fixture HEAD/readme/config/hooks/remotes/clean state проверяются независимо.
Always-save report включает failure stage, per-call ownership и cleanup facts.
Missing/invalid safe proof классифицируется только
permission_proof_unavailable_unclassified: текущий helper не выдаёт raw response,
поэтому отсутствие effective profile и legacy conflict нельзя различить догадкой.
Private fixture/evidence остаются; активные сервисы пользователя не затрагиваются.

Configured grants PASS не native write/commit PASS и не OS-enforcement PASS.
Models/threads/turns/tools/auth/config-write RPC, native Git commit/negative
sentinels, pipeline/UI/SSH/production/GUI и постоянный worker вне WORK206.
На момент source preparation никаких проверок или запусков этой пробы не было.

## Review202 lifecycle correction — v2 source only

Initial private qa206 checkpoint/SHA256SUMS09cfec9 сохранён без изменений.
Новый `.local/task014/qa206-v2/` добавляет общий execution90s и отдельный cleanup
reserve30s (total120), не меняя helper20+7, outer30 или wait15/drain10.
Новый child не стартует без полного per-call budget. TERM/INT/alarm разворачиваются
controlled exception через finally; повторные TERM/INT в cleanup маскируются,
его alarm остаётся независимым. До первого child/до cleanup сохраняется atomic
provisional FAIL report. Unknown/partial Popen без handle честно manual inspection;
no broad signals/retries/new config calls. Остальные sources/identity helper
унаследованы byte-exact. Bytecode отключён до sibling imports даже при hardstop.
Syntax/process/compile/config RPC по-прежнему NOT-RUN; новый source manifest
подлежит повторному202 review до exact root GO.

Root после отдельного202 compile-only approve собрал тот же identity source
d54c8b8… в clean8c43df9 module: helper SHA0d85bf8bf8355c01950ee5d93eca994eb22a9a7f06f98fb56997438041bdbeaa,
executed=false. QA helper не компилировал и не запускал. V2 pins заполнены
root-supplied clean candidate8c43df9f408712526e223796905bba787e336d5e,
Mac SHA8726daa43fbbf93b026215aede6fb3d2e7b98ebdd06644af5cea12dc1e92622f.
Проверяются baseline193 полный source set с override только approved204/205,
каждый исходный manifest тоже pinned. Это source-ready для review, не root GO
и не actual config/native evidence. Pipeline203 остаётся paused.

## Actual206V2 — FAIL before native, preserved

После FINAL SOURCE APPROVE202 e035fb98… и exact root GO выполнен один frozen
v2 run на candidate8c43df9/8726daa/helper0d85bf8. **FAIL fixture_setup**:
MCP0/app-server0/model0, effective permission inspection NOT-RUN. Начало16:50:49Z,
окончание16:50:51Z. Все10 owned fixture Git calls natural/proven exit0/!forced,
supervisor reaped/output drained, неизвестной ownership нет. Pins unchanged;
config.toml before/after bytes одинаковы, requirements/managed config отсутствуют
до/после. Config/global/GUI/production не менялись, restore не выполнялся.

Причина — QA framing bug: обычный candidate Supervise добавляет один LF перед
nonce exit proof. Capture сохранял его в data; git remote без remotes вернул
capture b'\n' вместо ожидаемого b'', fixture assertion остановился. Это не отказ
native CLI и не permission/config proof failure: Codex ещё не запускался.
Нельзя переносить этот факт на actual204 acceptance.

Private runtime qa206-1792774249668107000 содержит results.json, safe-proof.json,
executed-source и весь owned stdout/stderr. Frozen v2 источники не исправлялись.
Предложен отдельно reviewed framing-only delta: убрать ровно один protocol LF
для обычного Supervise, не менять config_inspection output path. Повтор только
после нового SOURCE APPROVE202 и exact root GO; автоматического retry не было.

Root разрешил v3 SOURCE ONLY framing fix: новая qa206-v3, executable delta только
generic Supervise возвращает data[:-1] после всех matched/proven/success/natural0/
!forced checks и обязательного final LF. Отсутствующая граница fail closed;
child-owned LF сохраняются, config_inspection branch byte-exact unchanged.
Owned module SHA8c83d2b7f698d5b2389663277815166cdb174aec9ae2a954fd0259bb09a54d09.
Остальные Python/Go sources byte-exact v2; pins обновляют только owned.py hash.
Compile/process/config RPC не запускались повторно, требуются202 re-review/GO.

Readonly scan paused203 выявил аналогичный риск в incomplete owned.cjs: строка39
сохраняет пустую framing line, command возвращает stdout без нормализации.
203 не менялся; его дальнейший scoped source pass должен учесть byte framing
до actual byte-level assertions. Это не повторная203 SOURCE APPROVE или execution.

## Actual206V3 — permission proof FAIL, cleanup proven

После FINAL SOURCE APPROVE202 e8c5ddda… / pinsaf78c02f… и отдельного exact root GO
выполнен один новый frozen run. Candidate8c43df9/8726daa/helper0d85bf8 прежние.
**FAIL candidate_config_inspection**,16:54:36.838Z→16:54:39.099Z (2.261s).
Setup/identity и оба MCP passes:15 owned calls natural/proven exit0/!forced.
Discovery0.094s; exact overrides verification0.053s:5 discovered/5disabled,
никакой новый managed connector не создавался.

ONE actual app-server invocation0.243s: candidate вернул nonce exit proof
proven=true/success=false/forced=false, child exit_code=-1, wrapper exit0,
output drained. **Safe permission digest отсутствует**. Raw config dialogue
остался внутри candidate; факт успешного прохождения config/read/requirements
не доказан. Unknown ownership/manual inspection нет. Без raw/stage diagnostics
классификация только permission_proof_unavailable_unclassified; missing profile,
active legacy setting или иной protocol/parser cause не установлен.

Models0, ни native commit, ни OS-negative не выполнялись. Pins unchanged;
before/after config bytes identical, snapshots private0600. Последующие success-
only fixture checks NOT-RUN после FAIL. Production/GUI/Testproject/config не
менялись. Повторов/расширения permissions/restore не было.

Private qa206-1792774476837996000/results.json и safe-proof.json связывают
actual facts; executed-source byte copy сохранена, V2FAIL остаётся отдельно.
До нового diagnostic design/source approval/root GO никаких новых вызовов нет.
