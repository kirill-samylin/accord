# Второй ПК: подключение к Accord

Обновлено18.10.2026. Общий Accord установлен на VPS; удалённый сценарий
с настоящим Codex на Mac прошёл. Физическую проверку второго ПК пользователь
выполнит позже. Точный статус — [TASK-003](TASK-003-working-service.md),
проверенная версия — [Codex003](codex-003.md).

## 1. Получить доступ

Подключите второй ПК к той же частной Tailscale-сети. Администратор выдаёт
частный HTTPS-адрес Accord с портом8443 и **отдельный токен для machine=pc-2**.
Адрес старого Tandem на443 и код его доски для Accord не подходят.
Не запускайте `serve` на втором ПК: оба клиента используют одну серверную БД.

Администратор выпускает токен на VPS штатным helper:

```sh
python3 /opt/accord/admin/token.py --owner SECOND_OPERATOR --machine pc-2 --days 30
```

SECOND_OPERATOR заменяют именем участника. Helper выводит путь private-файла,
значение в терминал не печатает. Передайте файл защищённым каналом, сохраните
ID для отзыва, после доставки удалите серверную plaintext-копию.
Токен Mac не копируйте; секрет не помещайте в чат, Git или config.toml.

На втором ПК сохраните файл `~/.config/accord/pc-2.token`. В macOS/Linux
каталог должен иметь0700, файл0600. Windows: ACL только текущей учётной записи;
POSIX chmod не является проверкой Windows ACL.

Откройте `https://YOUR_PRIVATE_ACCORD:8443/readyz`: ожидается200 с обычной
проверкой TLS. YOUR_PRIVATE_ACCORD заменяется выданным именем. Сертификатную
проверку не отключайте. Readiness без токена ещё не проверяет доступ к задачам.

## 2. Собрать локальный клиент

Нужны Git, Go1.26.1+ и установленный авторизованный Codex. На проверенном Mac
использован Codex CLI0.163.0-alpha.10.2. Сборка на других ОС не заменяет запуск.

```sh
git clone --branch codex/pilot https://github.com/kirill-samylin/accord.git
cd accord
go build -o bin/agent-board ./cmd/agent-board
```

На Windows последняя команда:

```powershell
go build -o bin/agent-board.exe ./cmd/agent-board
```

## 3. Подключить отдельного worker

В конфигурацию MCP клиента добавьте раздел, сохранив существующие настройки.
Замените пути и адрес на реальные абсолютные значения. Содержимое токена
в конфигурацию не вставляется.

macOS/Linux:

```toml
[mcp_servers.accord_worker]
command = "/ABSOLUTE/PATH/accord/bin/agent-board"
args = ["mcp", "--server", "https://YOUR_PRIVATE_ACCORD:8443", "--token-file", "/ABSOLUTE/PRIVATE/PATH/pc-2.token"]
required = true
```

Windows использует защищённый файл и передачу в окружение запускающего терминала:

```powershell
$env:AGENT_BOARD_TOKEN = (Get-Content -Raw "$env:USERPROFILE\.config\accord\pc-2.token").Trim()
codex
Remove-Item Env:AGENT_BOARD_TOKEN
```

Соответствующий раздел MCP Windows:

```toml
[mcp_servers.accord_worker]
command = "C:\\ABSOLUTE\\PATH\\accord\\bin\\agent-board.exe"
args = ["mcp", "--server", "https://YOUR_PRIVATE_ACCORD:8443"]
env_vars = ["AGENT_BOARD_TOKEN"]
required = true
```

Уже открытое GUI-приложение не получает окружение нового терминала автоматически.
Эти настройки сверены с [официальным Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli).

**Один worker — одна роль и один запуск исполнителя.** Он поддерживает heartbeat,
пока процесс жив, и должен завершаться вместе с исполнителем. Не используйте
один долгоживущий worker для всех параллельных ролей внутри IDE. Для клиентов,
которые разделяют один MCP между субагентами, требуется прямой HTTP с отдельными
board-сессиями и явным heartbeat; пример в [client-setup](client-setup.md).

## 4. Проверить подключение

Откройте новый отдельный запуск Codex и поручите ему:

> Используй только Accord. Открой describe с role=coordinator, machine=pc-2,
> свежими run_id/request_id. Создай синтетический epic с явным словом TEST,
> прочитай его через get и events. Отмени только этот epic с причиной
> synthetic cleanup. Сообщи результаты вызовов, ключ и next_cursor без секретов.

Все вызовы должны завершиться успешно. Запишите ОС/версию клиента, ключи и
курсоры. Через worker session_id можно опускать: он подставляет только свой.
Для другой роли нужен новый процесс worker и новый describe.

Позже совместно с первым Mac проверьте отдельную synthetic developer-задачу:
два разных worker одновременно вызывают next на один task_key; только один
получает аренду. Проверьте работу дольше15минут, heartbeat, release/retry,
завершение worker, истечение аренды и следующий generation. Это пока NOT-RUN
на двух физических ПК; локальные процессные тесты его не заменяют.

## Если возникла ошибка

- 401: проверьте срок/отзыв токена и правильный сервер; новый токен выдаёт администратор.
- machine mismatch: describe.machine должен совпадать с меткой выданного токена.
- SESSION_ALREADY_OPEN: для смены роли перезапустите worker.
- LEASE_UNCERTAIN/LEASE_LOST: прекратите локальные изменения, завершите worker,
  откройте новую сессию и сверьте владение. Не продолжайте старое generation.
- Нет связи: проверьте Tailscale и readiness; не публикуйте незащищённый HTTP.

Accord защищает операции очереди, но не может остановить локальный редактор
или запретить запись файлов на другом ПК. Рабочие копии и границы файлов
согласуются отдельно. Claude Code и Windows runtime требуют своей проверки;
совместимость Codex на Mac не считается их подтверждением.
