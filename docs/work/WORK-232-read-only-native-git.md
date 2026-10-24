# WORK232 — прямой Git для native read-only запросов

Root author,24.10.2026. SOURCE ONLY до независимого review202 и нового GO.
Сервер/worker binaries не менялись; файловые/MCP права не расширялись.

F6V9 actual35543 exit1, report6a15ad78ad23e5210dd412dc2ec1d6732545fa7c4486d4bf600aba07ff13cc6a.
Аналитик и первый разработчик completed; reviewer остановился с вопросом о
предупреждении Git/xcrun при создании /tmp/xcrun_db в read-only среде. Не было
changes_requested, повторного разработчика, повторного review или QA. FAIL
сохранён. Полный terminal cleanup этого failed trial ещё независимо не проверен.

Из exact own synthetic native UUID01a16b07-a62b-71a0-a111-553edb69eee0,
с verified meta.cwd и стабильным SHA всего log, извлечены два настоящих
tool output с xcrun/Operation not permitted. Частный bounded receipt
.local/task014/evidence-f6-v9/root-git-cache-warning.json SHA
90a8a23e90546d8120a7f22df45962d3a29edfdb143c318f12dcbe7536d0dad9.
Код из лога не выполнялся; другие пользовательские session logs не читались.

Владелец выбрал существующий canonical regular executable
/Library/Developer/CommandLineTools/usr/bin/git, SHA
a73bf622a2e470d5d57a4b1d5aef1e8680e67278018d4858a2f93825b7d595c7.
Новая инструкция для native read-only Git queries задаёт этот путь,
GIT_OPTIONAL_LOCKS=0, no-pager/no-replace-objects/fsmonitor=false/
untrackedcache=false. Цель — исключить обращения системного launcher к кэшу;
устранение предупреждения пока NOT-RUN, доказывается только новым actual trial.
Refusal rule и все обязательные evidence predicates сохранены.

F6V10 .local/task014/pipeline210/f6-v10/source-pins.json:
08adb44cc9ee6793015884055d6eed7bf2db841175c35ec40dd4b616307423e4,
80members. Только новый gate, common instruction, отдельный read_only_git tool
pin и lineage. context уже проверяет все tool hashes до/после, новый pin входит
в эту прежнюю проверку. automation Git executable и старые foundations
сохраняются отдельно: historical proof не переименован в proof нового пути.
Все6fixture files byte-exact0644, private source/pins0600;49CJS syntax/JSON/hash PASS.
Число invocations6/fix1/continuation0, проверки, guards, роли и source неизменны.

Onboard224V4 .local/task014/onboard224/v4/source-pins.json:
f238e8cd88b5cb0463d16d5d5da1db0741e8646d7109a2e43fc2361b23fd96b4,
24members. Только тот же append к4профилям и отдельная плановая metadata о
read-only Git. Это не новое неизвестное поле worker config. Mac policy привязана
к этому host; другая ОС требует своего owner-qualified canonical Git.
Scaffold/runner/configs/policy/plist byte-exactV3. Unit13 не повторялся без
изменения runner. Реальная Test trust остаётся pending, не расширяет QA219.

Не запущены новые helpers/provider/native/compiler/browser или workflow;
не изменены global settings, CLI cache permissions, GUI, production и Test code.
Исходный Python bootstrap WORK230 остаётся проверенным тем же scaffold hash.
