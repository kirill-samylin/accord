# WORK212 — source-only подготовка capable activation TASK014

Автор: automation_infra. Область записи только private .local/task014/ops212/ и
эта карточка. Product/infra/source/GUI/worker/production не изменяются.

Прочитаны AGENTS, TASK014/automation, WORK191/196/199/200/208/209/211, canonical
projects updater/guard/backup/layout/host helpers и предыдущие reviewed private
transport/activation/runtime scripts. Подготовлен узкий wrapper вокруг существующего
capable operator; новый updater не проектируется. Все скрипты — только исходники. Exact remote Go files/ancestors/hashes проверяются
перед любым импортом candidate helpers, включая readonly preflight.

Candidate: a32db22973d5b43d4fb227a89337bf5b82a4969f. Exact Linux/macOS/Next/Node
pins и55Go member hashes закреплены private pins. Baseline273/schema5/complete;
GUI worker62e не меняется. Infra191 и Linux199V4-P2 actual PASS7 с independent
APPROVE — prerequisites, не утверждение установленной автоматизации.

Состав: PLAN.md, pins.json, phase.py, preflight_remote.py, operation_remote.py,
postapply_remote.py, verify_release_remote.py, manifest.json. Каждый script<300lines. Future preflight
только readonly counts/hashes текущих projects/DB/leases/runs, никаких stop/cancel
чужой работы. Busy/unknown блокирует stage/apply, root решает дальнейший порядок.

Hard gates до ANY production stage/apply: exact independent SOURCE approval +
actual approved native A/C/B, fullpipeline, fix/newSHA, control/recovery, оба intake,
UI, Linux199. Отдельный root GO для каждой фазы. C/B/pipeline/UI pending, поэтому
stage/apply здесь заблокированы. Finalize дополнительно требует независимый
strict HTTPS smoke/evidence receipt. Environment bypass/auto-resume отсутствуют.

Будущий capable apply сохраняет полный all-project format2 precopy и exact old
logical rows/immutable bytes; post-audit проверяет owned runtime/schema6/floors/
marker/history. Finalize создаёт новый full schema6 backup и complete journal;
последующий audit проверяет inventory/runtime/history. Restore/DB rollback и
initial009 apply не используются. Timeout/unknown не называется positive exit.

Проверки подготовки: AST parse Python sources, bounded line counts/hash manifest,
diff check. Runtime CLI/process/native/SSH/HTTP/systemd/stage/apply/finalize не
выполнялись. SOURCE REVIEW202 и отдельное пофазное root GO ещё NOT-RUN.

Frozen source manifest SHA256:
`eb56ba6cfec08a2cf7898f3d41ac140e04371f4439b59a20cf2c8bace97589d5`.
5 Python AST PASS, maximum160lines/script;8 private files, all0600, directory0700.
Ни один подготовленный script не запускался/не импортировался при этих проверках.

## V2 после independent REQUEST CHANGES202

P1: root mode=ro над live SQLite мог создать root-owned SHM/WAL и нарушить
последующий writerUID999. V2 полностью убрал root SQLite connects: отдельный
fixed query child работает UID999/actualprimaryGID, extra_groups=(), WAL-aware
mode=ro/query_only/BEGIN, private metadata/ancestor/auxiliary checks до+после.
Missing WAL/SHM при WALheader2 блокирует read, без immutable/repair fallback.
Parent тоже проверяет UID/mode после child. Это источник, child не выполнялся.

P2: connection context не закрывал sqlite DB. V2 использует contextlib.closing
для всех live connections и positive child exit до следующей фазы.

P2: отсутствовал durable след прерванного SSH. V2 пишет immutable UNKNOWN attempt
перед transport import/SSH, fsync file+directory; SIGINT/TERM/exception/timeout
оставляют отдельный final UNKNOWN. Hardkill не стирает provisional. Локальное
прерывание не доказывает remote stopped; никакого auto retry/resume.

Полный предыдущий8file eb56 source сохранён private versions/eb56ba6c/ с exact
старым manifest. Новые исходники требуют независимого rereview; production273,
GUI и candidate не менялись. C/B/fullpipeline/fix/control/UI gates HARD CLOSED.

V2 frozen manifest SHA256:
`95e334fe2913e7b3436ba0b1e5f48212002c5f2778ab1a13736dda45731e2464`.
6 Python AST PASS, max190lines/script;9 current private source/manifest files.
Static checks: parent scripts не импортируют sqlite3, live child имеет closing,
provisional write предшествует remote(), old eb56 fullsource hashes exact.
Никаких import/process/DB/SSH/HTTP/systemd execution; V2 runtime NOT-RUN.

## ONE readonly preflight actual — PASS, activation CLOSED

После независимых SOURCE APPROVE20295e334fe… и focused215 FINAL EVIDENCE
APPROVE LinuxPASS5/localSignalPASS root выдал отдельный GO **только preflight**.
SOURCE receipt7177d863… binds exact source/local refs. Phase driver сохранил
provisional attempt до transport; один SSH/remote Python завершён successfully,
все live SQLite opens — UID999 с primaryGID/groups[], closed до final projection.

Baseline273ccbb15c99dfac91c37f34d2926abf01ca866f:4boards/9runtimeDB/5historicalDB;
leases0/managed-active-or-unknown0/workflows0. Go/Next listeners принадлежат своим
units, unitsactive/spaceverified, pointers/helpers/nginx/marker/candidate bundles
совпали с pinned состоянием, active_changed=false. Никаких stage/apply/перезапуска/
переключения/DB migration/backup replacement. Actual source95e334fe unchanged.

Private result-preflight-1792785143927679000.json SHA
e44c0d9d7472520f08a5cfffb722e0cea48bb1fbfa78f3409a9ba4ae12a50512.
Этот receipt не разрешает activation: stage/apply требуют всех native full/fix/
control/intakes/UI gates и нового fresh≤120s preflight/собственного root GO.
Старая preflight давность не продлевается и не скрывается.
