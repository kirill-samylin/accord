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

## Actual candidate freeze

Получен clean committed candidate
`bdcabbd9e97f3aa20ae32aeed1853529be1130dd`. Actual build.private, Go buildinfo,
полные Go55/Next1256 member manifests/SOURCE, production upstream7892 и все
hashes прочитаны и сверены; новых build/CLI действий автор не выполнял.
Original source APPROVE review_changes и build/predeploy QA APPROVE
validate_candidate приняты в их исходных областях проверки. Reviewer reports,
source manifest и producer build hashes проверены; все17 local refs совпали.
QA доказывает указанные regression/typecheck/format результаты, browser и native
запуск нового UI в ней NOT-RUN.

Созданы новые локальные regular-only USTAR archives, затем каждый tar member,
полный member set/path и SHA readback проверены. Go archive
`23c00314a4a84d9e3cc938c5b31b25ef025626d9f4bc9aa0d84d58af892169d1`,
Next archive
`c2949912b422a8f05c5d976bc98c142feb7ad5ff6b4a6888338cdb6bee274d13`.
Actual Go binary SHA
`c3f544812bf47aba17192df66b802ae2afebe74a2aab7dfcec907fe41601739d`,
Go manifest `dd147a67827eb58eaac3d9608d7aa9ca7100ffd2c02e2ee709a0da4ec14e58a9`,
Next manifest
`f12a22d93d4fcac3684d609d9eee3ae96868abb771962fffe282aeffb598f03b`,
BUILD_ID `B1sXF5AaCW_jt_RCH8Orq`; pinned Node hash прежний.

Executable source frozen в manifest
`a3bd6d026cbdbbb62bc97730f3ea471406c5f2d499245d95578b0034511e0248`:
12source/metadata files, 7AST PASS; local-only verify_source/delta_check PASS.
build_ready=true обозначает actual input готовность. Independent operational
source review и per-phase root GO ещё обязательны; фиктивных approval квитанций
автор не создавал. Никаких SSH/HTTP/live SQLite/upload/stage/apply/finalize/native
действий автором не выполнялось. Runtime installation остаётся NOT-RUN.

## Фактическое выполнение координатором

25.10.2026 координатор выполнил exact reviewed frozen source a3bd6d… для
clean candidate bdcabbd9e97f3aa20ae32aeed1853529be1130dd. Автор WORK287 только
прочитал сохранённые receipts и сверил их hashes; новых remote/runtime действий
не выполнял. Все12 frozen source/metadata hashes по-прежнему совпадают.

Upload-go1792931545209925000 и upload-next1792931557612622000 подтвердили
новые inactive releases. Preflight1792931588046221000 и fresh preflight
1792931618280539000 — все counters0 на четырёх boards; UID999 WAL-aware
connections закрыты. Stage1792931598973579000 → staged,
apply1792931635573161000 → pending-https-smoke и audit1792931698166309000
завершены с actual proof. Canonical capable updater сохранил schema6, полные
logical rows9runtimeDB и historical set; старый rollback не применялся.

Original independent ops-source APPROVE b264a769… и postapply PASS/APPROVE
3aba4f2e… проверены вместе с hashes исходных reports. После фактического
external HTTPS/UI принят original independent-https-smoke PASS/APPROVE
`cd89503ee181694d3ce6bcd1a6ed01b20fd4b53a241a7bb55990f463614a2b29`
и independent evidence review PASS/APPROVE
`0408dc1a43ae37863438659e49965243bbad940791dd5409b090c8667a847af9`;
report hashes совпали. Эти receipts относятся к actual bdcabbd server/UI,
новый native workflow/Windows/Claude execution ими не заявляется.

Canonical finalize1792932124963460000 — journal complete, receipt SHA
`aca901caaa080e6fe8d461752d651a8fc2fa4db6fda6a01abc4f1c7631ce923a`.
Final-audit1792932192189420000, receipt SHA
`a39bdc85da10291dfdf1a08a98cb453b9790a39dd90a431176287047a62e4753`,
подтвердил exact owned Go+Next, schema6, logical_rows_preserved=true,
9runtimeDB, history_preserved и новый validated format2 all-project backup.
Final backup manifest
`ed2caaf57bf114d8e6d977f0b20fa83944768239cee056ea5ba6406b174bf1bb`:
четыре проекта,9runtimeDB/9historicalDB, inventory_preserved=true.

Операционная последовательность завершена; отдельное final independent
acceptance проверяет review_changes и фиксирует координатор. Автор этой
карточки не подменяет её собственной положительной квитанцией. Mac worker,
consent/settings/native/Windows/Claude/второй физический ПК не расширялись;
срабатывание scheduled backup timer этим readback не проверялось.
