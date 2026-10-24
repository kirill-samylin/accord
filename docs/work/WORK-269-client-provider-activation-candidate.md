# WORK269 — отдельное разрешение запуска клиента

Автор: ops_validation_qa. Статус: SOURCE READY; установленный клиент не изменён, interactive runtime NOT-RUN.

Принятая независимая actual-проверка WORK267 подтвердила ровно шесть уникальных записей эффективного Codex MCP-каталога: только `accord_test` включён с точным stdio worker/argv, остальные выключены. Проверка завершилась exit0 с закрытыми потоками, без модели и без изменения глобальной конфигурации. Это доказательство каталога конкретного запуска, а не interactive native PASS и не проверка Claude.

Кандидат только в `.local/task014/client269/v1`:

- Общий `launch_ready=true` и прежний Codex namespace review flag `true`.
- Новый `provider_launch_ready={codex:true,claude:false}`. Общий и выбранный provider флаги требуют literal true; отсутствующие/false/неboolean/неизвестный provider блокируются до доступа к worker executable и его SHA.
- Launcher добавляет одну чистую проверку перед существующими действиями. Worker263, sandbox, approval defaults, точный bridge argv и process-only MCP overrides остаются прежними. Claude MCP JSON byteexact.

Настоящие пути, project и references находятся только в приватных файлах. Токены, их содержимое и хэши не читались. Три установленных файла проверены только по байтам/обычному own-singlelink metadata600 и остались равны WORK267. Глобальные MCP/trust/provider настройки не читались и не менялись.

Python AST parse: PASS. Чистые AST-selected тесты guard/main написаны для положительного Codex и отрицательных Claude/отсутствующих/неboolean/неизвестных флагов, включая sentinel перед worker доступом. Тесты автором NOT-RUN: ожидают независимого source review. Настоящие launcher main/CLI/worker/model не запускались.

Frozen manifest `.local/task014/client269/v1/source-pins.json`, SHA256 `2e82cd3f174d2f53b159d2dbd15f94fa1f2343e712f81368629ae2018039ab86`, шесть участников. Accepted prior actual202 receipt SHA256 `ca04a620bc2f57cc93c575b08844514481513022137f45f3446d9b1ccd8ce4d8`.

Осталось root: после independent SOURCE APPROVE установить только проверенные launcher/metadata с сохранением before bytes, затем провести отдельно разрешённую обычную проверку Codex клиента. Никакой автоматической отправки требования или запуска модели этот кандидат не выполняет. Claude, Windows, второй физический компьютер и текущий desktop chat: NOT-RUN.
