# WORK257 — независимый source review client253 handoff

24.10.2026. QA `managed_qa`, отдельно от автора пакета WORK253.
**APPROVE_EXCLUSIVE_LOCKED_INSTALL_ONLY**, findings отсутствуют. Разрешение review
ограничено подготовленным планом обратимой exclusive установки трёх файлов;
это не actual installation, не launch-ready/native/workflow acceptance.

Проверен immutable `.local/task014/client253-handoff-v1/`, manifest6members
`f78d2266aba90e5371613f610f43989fcaf89ce1260497b57694a2e449bbd659`.
Все members ordinary/single-link/own0600 под0700, все SHA совпадают; после review
source unchanged. Python AST разобран без import/exec. Launcher byte-exact
approved231V2 `ebcc148347948c4513818e4f3aa0e6dc48f3e18627d13720b100438bd6bcda28`.

Client metadata и matching Claude stdio args согласованы между собой и с
сохранённым accepted251/253 readback: правильные проект, own machine/checkout,
be95/9755 worker и ссылка на перемещённый собственный HOME credential.
Файл credential не открывался, не хэшировался и не stat-проверялся.
Никаких provider/global/trust/config/env contents не читалось. Указанный среди
input pins installed worker config НЕ перечитывался; сравнение основано на
accepted сохранённых reports и metadata packet. Проверены14 non-HOME source/
receipt inputs, включая старый source202 receipt и accepted policy/draft snapshots.

launch_ready и known namespace marker строго false; две process-only disable
keys сохранены, limitation отсутствия полного isolation inventory явная.
Существующий draft в saved snapshot awaiting_authorization/version1/reserved0;
policy/host binding и expiry согласованы с accepted snapshot. Fresh versions и
owner authority остаются обязательными: сохранённый readback не live authority.
Инструкция запрещает duplicate submit и automatic workflow authorization.

Install plan: новая exclusive0700 directory, ровно client.json/launch-client.py/
claude-mcp.json через O_EXCL/NOFOLLOW0600, fsync/inode/mode/SHA readback; failure
не перезаписывает/не удаляет существующие файлы и не запускает retry. Статический
план reviewed; actual installer здесь отсутствует и не выполнялся. Включение
local metadata flag и запуск требуют отдельного root gate после production/
owner trust/host/policy/namespace verification. Global MCP/approval/model/CODEX_HOME
настройки и OS controller grants этим пакетом не меняются.

Private receipt `.local/task014/evidence257/source-review.private.json`, SHA256
`6b7569b076f06de75dab4703f8cb75e24026bf8daf21f80382ecd790c89e03fc`,
0600 под0700. CLI/provider/helper/launcher/model/network/browser/install/token/
trust действия этим QA **NOT-RUN**. Windows/Claude/физический второй ПК NOT-RUN.
Другие source/live файлы и карточки не изменялись.

## Actual locked install — independent readback

Root install receipt SHA256
`af1bd4c5a827055e0a795d9d55e14ff1966b269327a7ad24dfa54f69f6e5b441`
сверен с утверждёнными source/review pins. **APPROVE_ACTUAL_LOCKED_THREE_FILE_INSTALL_ONLY**.
Свежим файловым чтением проверены ровно три установленных файла client metadata,
launcher и Claude MCP: ordinary/single-link/own0600, byte-exact accepted6pins;
родительский scoped каталог и client — own0700/no symlink. Лишних файлов нет.
Python только AST, без import/execution. Оба разрешающих флага строго false.
Root receipt также фиксирует global_settings_changed/runtime_started/
token_contents_read=false. Это recorded факт root, не самостоятельный process query.

Private independent receipt
`.local/task014/evidence257/installed-locked-readback.private.json`, SHA256
`59cdda98d14b06e52c99f0f773da273316e21875a965b744a1678a85957798ab`.
Токен/path target не осматривались и не хэшировались; global/provider configs
не читались, launcher/MCP/model/network не запускались. Установлены только
закрытые клиентские файлы; runtime-ready и workflow completion не заявлены.
