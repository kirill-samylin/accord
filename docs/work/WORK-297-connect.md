# WORK297 — `agent-board connect`: участник подключается одной командой

Исполнитель developer, ветка `codex/worker-setup` (после WORK296).
**UNIT PASS / make PASS / connect на Mac PASS / работа по задачам NOT-RUN**.
Поручение пользователя 26.10.2026: «нужна магия» — участник привязанного
проекта должен подключить своего агента без длинных настроек; сценарий:
двое в проекте, один запускает агента, создающего задачи, второй берёт задачи
с доски, не наступая на пятки и не воруя чужие.

## Разбор

Описанный сценарий уже обеспечивает доска через MCP-мост `agent-board mcp` и
аренды `describe → next`; managed worker (TASK013, `worker setup` WORK296)
для него не нужен. Единственное трение — ручная правка MCP-конфига каждого
CLI с абсолютными путями. Команда `connect` убирает именно его.

## Реализованный объём

- `cmd/agent-board/connect.go`: `agent-board connect --token-file PRIVATE
  [--server ORIGIN --project ID]`; строка в `--help`, диспетчер в `main.go`.
  В аргументах только путь и идентификаторы.
- `internal/connect/connect.go`: `Connect` — `worker.ReadToken` (приватный
  файл), сервер/проект из `connection.json` рядом с токеном или флагов
  (`https://` либо loopback), приватный каталог `~/.config/accord/` (0700),
  копия токена `<project>.token` (0600, атомарно, идемпотентно), копия
  текущего бинаря `agent-board` (0700, `worker.ReplaceExecutable`), затем
  регистрация `accord` в первом найденном экземпляре каждого провайдера:
  Codex — `codex mcp remove accord` + `codex mcp add accord -- BIN mcp
  --server … --token-file … --project …`; Claude Code — то же с
  `--scope user`; Gennadiy — `internal/connect/gennadiy.go`: `mcp.accord =
  {type:local, command:[…], enabled:true}` в `~/.config/gennadiy/gennadiy.json`
  с сохранением остальных ключей (включая корпоративные connectors с
  headers) и прав файла; не-JSON (комментарии) не трогается, печатается
  фрагмент для ручной вставки. `Report.Print` — без значения токена.
- `internal/worker/exports.go`: `DiscoverNativeCLIs`, `ConnectionHints`,
  `ReplaceExecutable`, `ProviderEnv` — экспорт существующих helper'ов WORK296.
- Документация: раздел «Подключение одной командой» в `docs/client-setup.md`,
  абзац в AGENTS.md, эта карточка.

## Границы и допущения

- Имя connector — `accord`; повторный запуск заменяет его (remove + add),
  другие connectors не трогаются. `accord_managed` worker'а не пересекается.
- `gennadiy mcp add` у форка интерактивен и не принимает команду аргументами,
  поэтому gennadiy.json правится напрямую; `$schema`, `model`, `permission` и
  прочее переписываются как прочитаны (порядок ключей JSON может измениться).
- Права агента на правку файлов задаёт сам CLI владельца; `connect` их не
  меняет. Токен по-прежнему выдаёт панель.
- Windows: пути `~/.config/accord` и права 0600/0700 не заменяют ACL;
  NOT-RUN.

## Фактические проверки

- `go test ./internal/connect` — PASS: приватные копии токена/бинаря с
  правами, точные команды `codex`/`claude mcp add`, запись `mcp.accord` в
  gennadiy.json с сохранением чужих заголовков и прав, идемпотентный повтор,
  JSONC → manual без изменения файла, создание отсутствующего gennadiy.json,
  отказ на относительный/0644 токен, подсказки `--server`/`--project` без
  `connection.json`, отказ на `http://`.
- `make fmt-check vet test race build infra-test` — PASS (27 пакетов ok,
  202 infra-тестов); веб не менялся, `web-check` PASS из WORK296.
- Реальный прогон на Mac пользователя 26.10.2026 по его поручению
  (`connect --token-file ~/.config/accord-gui/gui.token`, резервные копии
  `*.before-accord-connect` сохранены): токен и клиент установлены в
  `~/.config/accord/` (0700/0600/0700), `codex mcp get accord` — enabled,
  stdio, правильные args; `gennadiy.json` семантически не изменился, кроме
  нового `mcp.accord` (проверено JSON-сравнением); `gennadiy mcp list` —
  **accord connected**; claude CLI не установлен. Корпоративный
  `gennadiy-jira-confluence` падает независимо от нас (SSE error).
- NOT-RUN: реальный `describe → next` из gennadiy/Codex через `accord` и
  совместная работа двух участников.
