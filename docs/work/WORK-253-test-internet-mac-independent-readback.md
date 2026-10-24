# WORK253 — независимый readback Mac/Test_internet

`ops_validation_qa`, 24.10.2026. Вердикт **APPROVE_PRE_ENROLLMENT_ONLY**:
подготовленные файлы WORK251 готовы для следующей отдельной identity/enrollment
операции. Это не подтверждение enrollment, online worker, LaunchAgent или native
готовности. Никакие CLI/helper/runtime/network операции этим review не выполнялись.

Проверен отдельный HOME `/Users/shagi/.config/accord-test-internet/managed`:
обычные собственные каталоги 0700, binary0700, configs/logs0600, собственные
прочитанные файлы без symlink/hardlink. State пустой; два log-файла пусты.
Binary SHA256 `9755ed0fb0de869751635ba4e5adf1a4d742b0a202e6659e871d337fceed27ff`
совпадает с install receipt; worker be95, V3 provenance d98cd715… и independent
source202 receipt80bfcc79… проверены. Все25 source файлов onboard224V5 manifest
`26e1d1088d602c22e9e7518d2227ce4855fa825fc10f54e33a98a2c78321a5a9` совпадают.

Base/draft bytes совпадают с install receipt. Draft добавляет только automation.
Saved preview exit0/signalnull/stderr0, applied:false/requires_local_apply:true.
Текущий draft всё ещё неизменён. `worker.json` отличается от draft только точной
подстановкой check consent из этого preview, controllers пусты (owner implicit).
Expiry соответствует receipt и <=7дней от него. Git/Python hashes проверены
чтением уже pinned executables, без исполнения.

Workspace — настоящий `/Users/shagi/Developer/Test_internet`, не QA checkout.
Actual filesystem dev/ino identity `1000011-55cc618` совпадает с preview и его
process receipt. Следовательно, применён именно новый workspace-bound preview,
а не digest из QA. Это локальный check proposal, не глобальное доверие Codex.
QA219 исключение не применяется. Token разобран только как path reference;
token file/content/current global config не читались. GUI untouched — факт
root receipt, свежий аудит GUI state этим WORK не выполнялся.

Private report `.local/task014/evidence253/pre-enrollment-review.private.json`,
SHA256 `82449c11790b75da9299388621a8386a11968514324ddec133a59e130bf02b7e`.
Index `.local/task014/evidence253/pre-enrollment-index.private.json`,39 файлов,
SHA256 `1e259b56cbd1600035b373144ec3217c335dc081948e8e4590661d635d1c0e55`.
Файлы0600 под0700; секреты не переносились. Позднейшие identity/enroll/status
потребуют отдельного фактического readback; Windows/Claude/второй физический ПК
остаются NOT-RUN.

## Продолжение: actual identity/enrollment, online snapshot, policy/draft

**APPROVE_RECORDED_ENROLLMENT_ONLINE_POLICY_UNSTARTED_DRAFT_ONLY**. Предыдущий
pre-enrollment отчёт сохранён неизменным; live IDs находятся только в ignored
private evidence. Root receipts подтверждают identity/enroll exit0, пустой stderr,
исходные configs unchanged и отсутствие native workflow.

Проверены все три сохранённые before-config копии против прежних хешей и текущие
own configs: единственная semantic разница — token_file на private token внутри
worker HOME. Копия token имеет regular/single-link/owner0600 metadata; содержимое
и его хеш не читались. Равенство байтов/сохранность исходника подтверждаются
**root typed receipt**, а не независимым чтением секрета. Фактический owner principal
в identity, managed readback и policy совпадает; новая роль/credential не заявлены.

Текущий plist600 точно совпадает с HOME-only V2; по сравнению с сохранённым V1
изменён только HOME. Каталоги700, configs600, отдельные label/executable/config/
stdout/stderr paths корректны. Owned daemon receipt и HTTP200 managed snapshot
совпадают: epoch3, automation capability1, active/can_control, future lease
относительно recorded observed_at, runs0/pending0, parent launchd и отсутствие
параллельного foreground. Это сохранённый положительный snapshot; reviewer не
делал нового process query или HTTP poll.

Исторический первый epoch0/offline и неуспешный HOME-only V2 не переписаны в PASS.
Сохранены positive exact bootout/PID-absence receipt и foreground epoch2 с normal
SIGINT/exit0. Blocked open/TCC — ограниченные root наблюдения: точный syscall
pathname/полная причина остаются UNKNOWN. Успех после переноса token path не
выдаётся за полную причинную трассу и не означает изменение системных прав.

Все четыре active profile version1 имеют точные approved224V5 instructions и
provider; model пуст. В policy каждый role binding ссылается на этот profile1 и
native provider Codex. Policy active/version1, owner-only intake/control/client
intake, local commits; limits: 1task/2fix всего и на задачу/12native/
1continuation/300s step/1800s cycle. Первый grant86400 и expiry на следующие сутки
намеренно короче template7days и укладываются в local consent. Project-tests
digest точно совпадает с собственным реальным workspace preview/config.

Сохранённое Python требование — только awaiting_authorization/version1, grant
отсутствует, steps/plans/attestations/questions/reserved/invocations равны нулю.
Native/model/workflow execution и trust writes **NOT-RUN**. Готовность к реальному
запуску не подтверждена: остаются отдельное owner real-Test trust подтверждение,
production finalize verification и явная авторизация native workflow. QA219 сюда
не переносится. Windows/Claude/физический второй ПК по-прежнему NOT-RUN.

Private post-enrollment report:
`.local/task014/evidence253/post-enrollment-review.private.json`, SHA256
`0c9d96db7c65b46f85c5454d57ee4e9ebc319b295be59a764140e8fe1fe57a41`.
Index `.local/task014/evidence253/post-enrollment-index.private.json`,29 файлов,
SHA256 `c9cbe0988971260baa3125e106521f54db6ffd5c0e9f4e84137b241713310282`.
Token присутствует только как metadata-only запись без content/hash.
CLI/helper/network/process/DB/global-settings действия review не выполнял.

## Source-only передача standalone client

Подготовлен отдельный private пакет `.local/task014/client253-handoff-v1/`:
client metadata, matching Claude MCP JSON, byteexact approved231V2 Python launcher,
private installation plan, короткая owner/receiver инструкция и input pins.
Actual server/project/workspace/policy/draft identifiers находятся только внутри
этого ignored пакета. Публичный generic guide не менялся.

Manifest6members SHA256
`f78d2266aba90e5371613f610f43989fcaf89ce1260497b57694a2e449bbd659`.
Launcher остаётся exact231V2
`ebcc148347948c4513818e4f3aa0e6dc48f3e18627d13720b100438bd6bcda28`.
Все файлы600 под700, JSON/Python AST проверены статически. Launcher не импортирован
и не запущен; CLI/model/network/install/token/global config операции NOT-RUN.
Целевой HOME client отсутствует, установка не выполнена.

`launch_ready=false`; дополнительный local namespace-review marker false до
повторного подтверждения root. Actual token_file — ссылка на существующий private
worker HOME token без чтения/хеширования. Root-only остаток: exclusive install3files
без overwrite, контроль pins/modes, production finalize/trust/host/policy/version
gates, отдельная авторизация запуска и только затем local launch-ready marker.
Global MCP/trust/approval/model настройки пакет не меняет.

Первое требование уже сохранено как незапущенный draft: инструкция требует
использовать этот существующий draft, свежие policy/workflow versions и supported
resume с refresh delegation, не отправлять duplicate submit и не authorise автоматически.
Второму человеку нужен индивидуальный credential/machine; Mac token/state/proof
не передаются. Его worker/local-owner enrollment и взаимные host allowlists/policy
grants — отдельный этап; membership/editor не означает OS control. Windows/Claude/
физический второй ПК остаются NOT-RUN.
