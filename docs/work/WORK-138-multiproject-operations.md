# WORK-138 — операции и поставка мультипроектности

Owner: root, разработчик infra; независимый reviewer WORK136.
Область: `infra/accord/projects/**`, связанные infra unit, защита legacy
операторов, release packaging; общая TASK и интеграция также root.
Не изменяет Go/frontend исполнителей, не пишет новые process/E2E.

## Контракт

Старый board-only backup не является полным восстановлением registry.
Полная offline копия включает registry, default на прежнем физическом месте,
все опубликованные boards/web/blobs/staging и скрытую provisioning recovery set.
Manifest format2, schemas, hashes, SQLite integrity, private paths, fsync,
atomic publication и seven-copy retention. Несогласованный WAL, orphan storage,
symlink, неизвестная схема или неверный binding завершают backup ошибкой.

Перед каждым стартом отдельный защищённый systemd helper требует capabilities
`project_registry=1`, `project_acl=1`, `registry_schema=[1]`; проверяет и при
потере marker. Drop-in закрепляет `--projects-data`, сохраняет прежние
ExecStartPre guards. После активации нельзя откатить Go до старого binary
или подменить текущие записи старой БД. Поставляются capable recovery/roll-forward.

Nginx меняет только scoped MCP route и точный scoped upload с прежними
ограничениями 50MiB/120s. Next assets сохраняют собственный rate budget;
Host/Origin, bearer, request body и старые native/browser маршруты сохранены.

## Проверки

Авторские units:31 новых PASS, все115 infra PASS (Python3.14; ранее существующие
ResourceWarning SQLite handles не превращены в runtime PASS). Проверены разные
T-1/blobs в A/B, schema1/2, pending journals/quarantine, unknown files/symlinks,
unresolved WAL/rollback journals, guard без marker, ingress byte preservation,
NEW offline staging/reset-failure boot guard, interrupted cutover/resume и
отсутствие выбора старого Go. Последние4 orchestration units — только mocks.
Новые process/E2E, реальная миграция, restore rehearsal, nginx/systemd на Linux,
VPS backup/update и runtime ACL — NOT-RUN. WORK136 подтвердил scoped infra
source/unit APPROVE31/31. Readiness теперь доказывается exact127 endpoint и всеми
PID socket owner == systemd MainPID выбранного unit, свободные порты до запуска;
occupied/foreign/multiple PID отвергаются. Это unit source proof, actual Linux
stage ещё впереди. Offline bootstrap того же owner при resume идемпотентен;
иной существующий owner остаётся conflict (author135 regression).

Все новые файлы менее 300 строк. Snapshot fixtures только synthetic;
реальные данные остаются на VPS, артефакты под `.local/`.

## Actual delivery checkpoint5454eb5

Source/test review APPROVE136; local25process+Chrome и16process-only PASS139.
Root clean Linux release/buildvcs, pinned Node/SRI package, manifests/transfer
PASS. На реальном VPS reviewed synthetic Linux process-only16/16 PASS,
startup loss/default preservation/full snapshot/restore/reset/reject/old guard±.
Root wrapper final plain bind встретил только closed TCP TIME_WAIT: actual
report PASS, ss empty + SO_REUSEADDR bind3100/3101 PASS; product не менялся.

Historical stage/apply returned success; effective mode later FAIL: owned listeners, hardened Linux Node/sharp, nginx
syntax, previous format1 snapshot, explicit offline selected owner, capability
drop-in и capable Go/Next5454 active. Actual executable hashes/Node cwd
подтверждены; restart0. Сравнение pre-upgrade snapshot с live state подтвердило
все прежние строки9 board tables, design rows и blob bytes/default paths.
Реальные данные не скачивались, generated fixtures отдельные private на VPS.

Phase pending-https-smoke: первый generic readonly run harness FAIL, оригинальная
ошибка скрыта dangling cleanup wait, report отсутствует. Product fix пока не
установлен; только tester-owned diagnostics/wait fix и новое review. Полный
HTTPS inspection/finalize-format2 backup ещё NOT-RUN. Root не выдаёт active
version за completion, не вызывает old rollback и не заменяет БД.

## Открыт P1 effective startup mode

Actual systemd90-projects.conf был перекрыт web.conf: новый6950 binary работал
без --projects-data. Прежний apply/readiness/hash PASS относится только к этим
проверкам; production ACL activation FAIL. Go/Next stopped через reviewed
fail_closed; root поставил durable owned START_REQUIRES_INSPECTION0600+fsync.
Root исправляет порядок own override и проверки resolved executable+argv,
permanent guard и actual owned process args, backup command. Новый helper
recovery держит cold-boot guard во всём install/amend окне и сохраняет manual
markers. Source Go/Next и реальные DB frozen, никакого rollback.

Авторские project infra41 unit PASS, общая suite125 PASS, source review136
открыт. Новые operators — отдельный immutable source release; старый5454
app не переписывается. Trusted offline helper amendment допустим только после
review с exact old/new journal+helper hashes, stopped units/ports/owned guard,
O_EXCL/fsync original journal и unchanged revision/Go/Next/baseline/owner.

Root amendment freeze11files `.local/task009/deploy/launch-amendment-freeze.json`.
Новый helper startup_hold.py назван уникально: первая общая suite выявила
collision с прежним recovery.py; scoped+полная suite после rename PASS125.
Permanentguard сопоставляет и executable path guard (защита отsystemd @argv),
и effective backup path. One-off adoption source отдельно reviewed136,
исполняется только non-optimized root/Linux через python3 -I, архивируется
в private serverjournal по своемуSHA. В audit включён егоSHA; old/new helper
maps иполный immutableoperator manifest pinned. Операторapplication5454 не
переписывается; recovery hold ивсе реальные DB остаются до guardedresume.
