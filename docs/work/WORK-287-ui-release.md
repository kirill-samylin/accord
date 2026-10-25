# WORK287 — подготовка поставки обновлённого UI

Исполнитель release_prepare. **SOURCE PREPARED / runtime NOT-RUN**.
Область записи: эта карточка и новая ignored `.local/task018-ui-release/ops/`.
Общую TASK и чужие исходники/receipts исполнитель не меняет.

Прочитаны AGENTS, operational README, WORK284, PLAN и exact reviewed/executed
TASK017 drivers. Все members исходного manifest a577b0… совпали. Original
independent final receipt d80040cf546ff92320eccac0f84ca586a3b4b6f6, report,
finalize и final-audit hashes сверены локальным чтением; новый live readback
автор не выполнял. Исторический accepted baseline не выдаётся за current VPS.

Скопированы семь скриптов в новую область; bounded delta — TASK018 label,
служебные import labels и diagnostic WORK287 tags. Canonical capable
`infra/accord/projects/upgrade.py` и прежний pinned SSH transport сохранены,
новый updater/framework не вводится. Baseline pins d800 взяты из original
accepted final receipt без retagging старых native/browser gates. Original
TASK017 sources/receipts не изменялись.

Candidate ещё pending: build_ready=false, actual revision/source/build/QA hashes,
reviewer и remote_go_files отсутствуют; archives placeholders null и executable
manifest.json отсутствует. Это fail-closed подготовка, не approval. WIP source
manifest `02a0ed2e6137b48a78c26078a2c191fa1ac888601bea392d3f5cff03a6ae1565`
не разрешает транспорт. PLAN содержит точные build entrypoints и per-phase
commands, candidate receipt shape и порядок independent review → upload только
NEW inactive → preflight/stage/fresh preflight/apply/audit → external HTTPS QA →
finalize/final-audit. Полный Go+Next собирается с одного clean committed SHA,
production upstream7892; текущий pinned Linux Node и LICENSE имеются локально.

Обязательные guards, ACL/schema6/floors, UID999 WAL-aware closed query,
4boards/9runtimeDB/9historicalDB, exact logical rows/history, immutable format2
all-project backup, root ownership, durable UNKNOWN-before-transport и
roll-forward сохранены. Busy/unknown блокирует, принудительная остановка
пользовательской работы и автоматические retry/cleanup/старый rollback запрещены.

Фактические проверки: семь Python AST PASS; исходный manifest members PASS;
original d800 report/finalize/final-audit hashes PASS; шесть baseline/local
reference hashes сохранены. Build, SSH, HTTP, live SQLite, upload, stage/apply,
finalize, native/control и изменение runtime NOT-RUN. Windows/Claude/второй
физический ПК этим этапом не проверяются.

От root требуются actual clean candidate SHA, Go+Next bundle paths/manifests,
buildinfo и producer report, независимые source/build/predeploy QA receipts.
После их поступления можно заполнить actual pins, создать exact regular-only
USTAR archives и freeze source для независимого operational review. Затем
координатор даёт точный per-phase GO в рамках пользовательского deployment
поручения; отдельного нового запроса пользователю этот документ не вводит.
