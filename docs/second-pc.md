# Второй компьютер: подключиться к Accord

Обновлено 19.10.2026. Теперь используйте обычный интернет, HTTPS и свой токен.
Tailscale для подключения не нужен. Установленный общий сервер остаётся
на VPS: устанавливать сервер на втором компьютере не требуется.
Если клиент уже настроен, достаточно [сменить адрес](public-https.md).

## 1. Получить три файла

- `Accord-project.zip` — пакет с готовыми клиентами и инструкциями.
- **Обновлённый** `connection.json` — действительный адрес в поле `origin`.
- `pc-2.token` — отдельный секрет доступа этой машины, передаётся защищённо.

Старый токен для этой же машины сохраняется при смене адреса, пока он действителен.
Токен первого компьютера не копируется. Адрес и токен не публикуйте в Git.
Получение архива не заменяет получение двух отдельных файлов доступа.
Старые указания пакета о Tailscale/порте 8443 заменены этой инструкцией.
Состояние релиза и ограничений — [TASK-003](TASK-003-working-service.md).

## 2. Выбрать готовый клиент

Распакуйте `Accord-project.zip`, найдите `clients/` и выберите архив:

| Компьютер | Файл |
| --- | --- |
| Windows x64 | `accord-client-windows-amd64-1e9025c.zip` |
| Mac Apple Silicon | `accord-client-darwin-arm64-1e9025c.zip` |
| Mac Intel | `accord-client-darwin-amd64-1e9025c.zip` |
| Linux x64 | `accord-client-linux-amd64-1e9025c.zip` |
| Linux ARM64 | `accord-client-linux-arm64-1e9025c.zip` |

Windows ARM64 native-архива в пакете нет. Проверьте SHA256 архива по манифесту
пакета, распакуйте и проверьте binary по `SHA256SUMS` его ZIP.
Mac: `shasum -a 256 -c SHA256SUMS`; Linux: `sha256sum --check SHA256SUMS`;
PowerShell: `Get-FileHash .\agent-board.exe -Algorithm SHA256` и сравнение
с выданным SHA256SUMS. Для подключения Go и сборка не нужны.

Поместите binary в `~/accord-client` (`%USERPROFILE%\accord-client` на Windows),
а два файла доступа в `~/.config/accord`. Не помещайте токен в Git checkout.
Откройте `origin` из JSON, добавив `/readyz`, в браузере: ожидается HTTP 200
с нормальной проверкой сертификата. Не отключайте TLS-проверку.

## 3. Настроить Codex на macOS/Linux

После переноса файлов установите права:

```sh
chmod 700 "$HOME/.config/accord"
chmod 600 "$HOME/.config/accord/pc-2.token" "$HOME/.config/accord/connection.json"
chmod 700 "$HOME/accord-client/agent-board"
```

В `~/.codex/config.toml` сохраните существующие настройки и добавьте раздел:

```toml
[mcp_servers.accord_worker]
command = "/ABSOLUTE/PATH/accord-client/agent-board"
args = ["mcp", "--server", "https://ACCORD_ADDRESS", "--token-file", "/ABSOLUTE/PRIVATE/PATH/pc-2.token"]
required = true
```

Замените пути на реальные **абсолютные** пути этого ПК, а адрес-заглушку — на
точное поле `origin` из нового `connection.json`, без добавления `/mcp`.
Клиент не читает JSON автоматически; адрес нужно перенести в `--server`.
Сам токен в TOML не вставляйте. Перезапустите локальный Codex.

## 4. Настроить Codex CLI на Windows

После переноса приватных файлов в `%USERPROFILE%\.config\accord`:

```powershell
$AccordPrivate = Join-Path $env:USERPROFILE '.config\accord'
$AccordUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
icacls $AccordPrivate /inheritance:r /grant:r "${AccordUser}:(OI)(CI)F"
icacls "$AccordPrivate\pc-2.token" /inheritance:r /grant:r "${AccordUser}:F"
icacls "$AccordPrivate\connection.json" /inheritance:r /grant:r "${AccordUser}:F"
icacls "$AccordPrivate\pc-2.token"
```

Если остались явные разрешения другим обычным пользователям, удалите их
через свойства файла «Безопасность»; снятие наследования их не удаляет.
Не изменяйте права чужих каталогов. В `%USERPROFILE%\.codex\config.toml`:

```toml
[mcp_servers.accord_worker]
command = "C:\\Users\\YOUR_USER\\accord-client\\agent-board.exe"
args = ["mcp", "--server", "https://ACCORD_ADDRESS"]
env_vars = ["AGENT_BOARD_TOKEN"]
required = true
```

Замените абсолютный путь и адрес, как описано выше. Из этого PowerShell
запустите новый отдельный Codex CLI:

```powershell
$env:AGENT_BOARD_TOKEN = (Get-Content -Raw "$AccordPrivate\pc-2.token").Trim()
try { codex } finally { Remove-Item Env:AGENT_BOARD_TOKEN -ErrorAction SilentlyContinue }
```

Токен остаётся в памяти процесса, не печатается. Не сохраняйте его значение
в TOML/профиле/командах. Уже открытый GUI не получит это окружение автоматически.
Windows runtime требует фактической проверки на Windows, cross-build её не заменяет.
Настройки: [официальная документация Codex MCP](https://learn.chatgpt.com/docs/extend/mcp).

## 5. Проверить и начать работу

В новом локальном Codex проверьте `codex mcp list`, затем Accord в `/mcp`.
Выполните [проверку подключения с отдельным TEST-эпиком](public-https.md#проверка-именно-с-этого-компьютера):
initialize/catalog 12, describe `machine=pc-2`, create/get/events и отмена только
своего эпика. Новое действие получает новый request_id; required-поля — из схемы.
Сохраните результат без токена и завершите отдельный worker.

Один worker — одна роль и один запуск исполнителя; он поддерживает heartbeat.
Общий MCP для независимых субагентов требует прямого HTTP и явного heartbeat,
его правила — [client-setup.md](client-setup.md). Не включайте оба режима
одновременно для одной работы. При потере аренды остановите локальные изменения.
Два физических ПК отдельно проверяют совместный claim/heartbeat/expiry;
одиночный smoke это не подтверждает. Claude Code требует своей настройки.

Если что-то не работает — [таблица диагностики](public-https.md#если-возникла-ошибка).
Не ищите Tandem MCP и не запускайте `serve`. Разработка передаётся отдельно
по [handoff.md](handoff.md); её исторические частные адреса не используйте.
