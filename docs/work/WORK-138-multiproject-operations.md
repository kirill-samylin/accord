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
