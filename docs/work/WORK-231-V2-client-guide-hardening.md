# WORK231 V2 — исправление инструкции standalone client

24.10.2026. Узкий SOURCE READY replacement по двум P2 независимого WORK202.
Область: новая private `.local/task014/client231/v2/` и эта карточка.
Исторический packet/manifest `365ced9b…` и все пять исходных members сохранены
byte-exact; launcher, client metadata, Claude MCP и references в V2 также
byte-exact. Из исходных пяти файлов изменён только GUIDE.md.

Mac-блок теперь отдельный fail-fast subshell: успешный exclusive mkdir 0700
предшествует всем копиям, noclobber и umask 077 защищают новые файлы; ошибка не
позволяет продолжить копирование поверх существующей установки. Частично
созданная новая папка остаётся для ручной проверки, автоматического удаления нет.

Windows-пример принимает только case-sensitive bare TOML paths из ровно трёх
сегментов mcp_servers.NAME.enabled или пяти
plugins.PLUGIN.mcp_servers.NAME.enabled. Собственный accord_test, arbitrary
plugin/sandbox leaves, newline, =, quotes и лишние сегменты отклоняются.
Это более узкий поддерживаемый bare-name subset Python disabled_key; runtime
Python helper не изменён. Нет новых permissions или provider flags.

Pure тест на pinned Python 3.14: **PASS 5 accepted / 18 rejected**, плюс два
non-MCP plugin leaf negatives для извлечённой чистой Python-функции. Только AST
этой функции выполнялся; launcher module/main не импортировался. Структура
fail-fast shell блока проверена статически. PowerShell и install shell блоки
**NOT-RUN**, provider/helper/CLI/native/model/HTTP/browser/config/token reads —
**NOT-RUN**. JSON/AST, неизменность четырёх файлов, oldpacket hashes и режимы
root0700/files0600 проверены. Ничего не установлено; launch_ready=false сохранён.

Freeze 8 members:
`2f9edf82862547a719fe9e48475622cd489c5921caa1c7c2ae92d39b0eb3680b`.
SHA256SUMS `7de0d113e4bd1a4f94064e8d1b746d5e79af93445518af7add0569824c9e874b`.
Guide `8e97f2cfce2eb0bfd7b4d28c297fb9d83608a999142c6812870439df6db60a44`.
Pure source `6302eac51e6a924a38f7bd6211bf455cad3cc10326904d0f31dae6aa551d76a9`;
evidence `982b82699d9af34ed235dc30cf2374450997288ac3b820c789d2e4fcdc6e0d35`.
Ожидается независимый SOURCE review202; установка/подключение не объявлены PASS.
