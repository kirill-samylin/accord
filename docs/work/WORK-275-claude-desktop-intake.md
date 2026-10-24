# WORK275 — Claude Desktop → Accord intake

Дата: 24.10.2026. Author scope: private additive Desktop MCP candidate и helper;
не применение, перезапуск, отправка требования или production mutation.
Поручение пользователя относится к настоящему Claude Desktop chat. Существующий
Codex worker выполняет роли; Claude CLI login/launcher этим не включаются.

## Read-only inventory

Известный `claude_desktop_config.json`: обычный файл, mode0600, собственный
UID/GID,4740байт. Top-level keys: `coworkUserFilesPath`, `preferences`.
`mcpServers` и `accord_test` отсутствуют. Значения preferences/user paths не
публиковались, auth/cache/chats/keychain не читались. SHA конфигурации, токена
и baseline/candidate с существующими preferences не вычислялись.

## SOURCE ONLY

В `.local/task015/claude-desktop-author/` подготовлены private baseline/candidate
и `apply-desktop-mcp.py`. Единственное добавление — `mcpServers.accord_test`:
существующий pinned worker как command; args `mcp --server` рабочего Accord,
точный Test_internet project ID и `--token-file` собственного private файла.
Значение токена отсутствует в JSON/argv/env; helper читает только его metadata.
Desktop JSON использует `command`/`args`, без CLI-only `type`/strict flags.

Существующие bytes/preferences сохраняются: новая root property вставляется
перед заключительной скобкой, семантический readback допускает ровно одну entry.
Duplicate JSON keys/неожиданный существующий mcpServers отвергаются. Это текущий
узкий baseline, не миграция произвольной конфигурации с другими connectors.

Helper требует отдельный root GO с source/metadata SHA, ordinary files,
точные stat snapshots и сравнение baseline/current bytes в памяти. При drift
нужен новый review; автоматических повторов нет. Перед replacement — private
backup0600+fsync, новый temporary в той же папке, compare-before-rename, атомарный
replace, directory fsync и точный readback UID/GID/0600. Это не атомарный
conditional CAS: root должен сначала полностью закрыть Desktop, чтобы он
не переписывал preferences одновременно. Helper не закрывает/не запускает app.

PASS: pure transform одной настоящей private baseline и3negative cases;
исходники syntax-valid. Desktop config writes0, модели0, token-value reads0.
NOT-RUN: independent approve, apply/restart, Desktop MCP catalog/model tool-call,
новый multiply workflow с8тестами и управление из Accord. Это отдельная проверка
от ранее принятого Codex клиента; наличие desktop login проверяет root через UI.

## Источники

24.10.2026 прочитаны официальные
[Claude Desktop local MCP](https://support.claude.com/en/articles/10949351-getting-started-with-local-mcp-servers-on-claude-desktop)
и [MCP local server guide](https://modelcontextprotocol.io/docs/develop/connect-local-servers).
Вторая страница документирует точный macOS config path, command/args JSON,
полный restart после сохранения и обычные per-action approvals. Никакие
permission bypass, extension package, CLI OAuth или глобальная автоаппрувка
этим candidate не добавляются. Фактическую доступность tools подтверждает UI QA.

## Root apply — отдельный фактический этап

После independent SOURCE APPROVE273 (`claude-desktop-source273.json`, findings0)
root выполнил единственный согласованный helper: фактический exit0. Receipt:
`.local/task015/claude-desktop-root/apply-actual.private.json`.
Подтверждены точная entry `accord_test`, равенство всех прочих JSON fields исходным,
backup, сохранённые UID/GID/mode0600. Значения токена и хеши всей конфигурации
не записывались. Это **apply PASS**, отдельно от ранее принятого source.

Root запросил обычный перезапуск Desktop через UI. Само появление рабочего окна,
loaded MCP catalog, model `describe`/`workflow_submit` tool cards и полный новый
workflow ещё не подтверждены этим receipt: **Desktop intake/workflow NOT-RUN**.
Текущая недоступность окна в UI не заменяется CLI/SDK запуском. CLI auth/readiness
остаются отдельным непройденным сценарием. Автор не запускал модель и не менял
конфигурацию; root продолжает foreground/window и native chat QA.
