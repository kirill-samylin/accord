> Обновление 25.10.2026: репозиторий DenisShagi/accord удалён; единственный
> репозиторий — [kirill-samylin/accord](https://github.com/kirill-samylin/accord),
> текущий код в ветке `codex/native-preflight`, релизы — теги `v*` с готовыми
> артефактами (см. [release-process.md](release-process.md)). Для передачи
> исходников: `git clone --branch codex/native-preflight https://github.com/kirill-samylin/accord.git`.
> Объединение истории 23.10.2026 описано в [repository-consolidation.md](repository-consolidation.md).
> Инструкции старого bundle/`codex/pilot` ниже — исторические. Для современного
> schema5 worker используйте [managed-worker-guide.md](managed-worker-guide.md)
> и фактический [TASK013](TASK-013-managed-agents.md); native GUI проверка пока FAIL.

> Обновление 19.10.2026: клиентский доступ Accord теперь через обычный публичный HTTPS.

> Дополнение веб-версии19.10.2026: человеку достаточно открыть существующий
> HTTPS-адрес и войти по своему токену/файлу токена. Четыре раздела описаны в
> [руководстве веба](web-guide.md). Установка MCP нужна только агенту.
> Для ресурсов загруженного дизайна обновите старый клиент1e9025c до6245829
> из проверенного платформенного архива и перезапустите MCP; адрес и токен
> остаются прежними. Старый клиент сохраняет прежние12 инструментов, но
> не показывает новые design resources. Таблица старых архивов ниже относится
> к предыдущему выпуску и не является пакетом обновлённого веб-клиента.

> Tailscale на клиенте не требуется. Актуальные шаги — [public-https.md](public-https.md)
> и [second-pc.md](second-pc.md); прежние клиентские указания Tailscale/8443 ниже исторические.
> Приватные серверные маршруты сохранены для совместимости и Tandem.

# Передача Accord: второй компьютер и разработка

Снимок на 18.10.2026. Сначала выберите цель: разделы 1–6 подключают компьютер
к существующему серверу; разделы 7–9 передают разработку. Сервер уже работает,
поднимать второй сервер и переносить БД для подключения клиента не требуется.

**Используем Accord. Tandem временно отключён по указанию пользователя.**
Не ищите Tandem MCP, canvas code и задачи его доски. DNS-имя существующего
хоста может содержать слово tandem: сервис определяется также портом и путём.
Accord использует частный HTTPS8443; прежний Tandem на443 сохраняется.
Отсутствие Accord в списке инструментов нового агента означает, что требуется
настроить MCP в его локальном приложении. Клонирование репозитория этого не делает.

## 1. Что передаётся

| Материал | Назначение |
| --- | --- |
| `Accord-project.zip` | Общий пакет: bundle, инструкции, клиенты, process-reference и SHA256SUMS |
| `accord.bundle` внутри пакета | Исходники и сохранённая Git-история/ветки; автономное клонирование |
| `clients/accord-client-<OS>-<ARCH>-1e9025c.zip` | Готовый клиент; Go для подключения не нужен |
| Контрольные суммы пакета и `SHA256SUMS` внутри client ZIP | Проверка целостности архивов и исполняемого файла |
| Эта инструкция и [receiver-agent.md](receiver-agent.md) | Действия человека и готовое задание агенту |
| `process-reference/` в пакете передачи | Справочные профили ролей/процесс, без зависимости от папки Давинчи |
| **Отдельно: `connection.json`** | Действительный частный адрес в поле `origin`, machine и версия сервера |
| **Отдельно: `pc-2.token`** | Секрет доступа второго ПК; в общий архив проекта не входит |

В исходном Mac приватные файлы находятся в
`.local/accord/second-pc/` относительно корня Accord. Токен передайте отдельно
по защищённому каналу. Не отправляйте его в чат и не вставляйте в Git, TOML,
командную строку, логи, скриншоты или профиль оболочки. Файл `connection.json`
не содержит самого токена; частный адрес также не публикуйте в репозитории.
После распаковки общего ZIP положите отдельно полученный `connection.json`
в корень пакета рядом с инструкциями; `pc-2.token` доставьте сразу в приватный
каталог клиента. В общем ZIP обоих файлов нет. Команды ниже выполняются из
корня распакованного пакета, затем metadata копируется в приватный каталог.

Текущий credential выдан для `machine=pc-2`, `owner=operator-a` на30дней.
Срок отсчитывается от выдачи, а не первого запуска. Для другого компьютера
или после истечения нужен отдельный токен администратора; токен первого Mac
не копируется. Одну машинную метку не используйте для двух физических ПК.

Пакет не содержит SSH-ключей, Codex/OpenAI credentials, рабочей БД, серверных
backups и локальных кэшей/секретов. Это передача исходников и клиентского
доступа. Смена владельца VPS, Tailscale и резервных копий — отдельная операция.

## 2. Проверить сеть и выбрать клиент

Установите/включите Tailscale и подключитесь к той же частной сети, что сервер.
Приглашение и разрешение доступа выдаёт владелец сети. VPN другой учётной
записи недостаточно. На момент передачи с Mac и VPS видны только основной Mac
и сервер; физический второй ПК из них не виден. Его подключение ещё не доказано.

| Компьютер | Архив |
| --- | --- |
| Windows x64 | `accord-client-windows-amd64-1e9025c.zip` |
| Mac Apple Silicon | `accord-client-darwin-arm64-1e9025c.zip` |
| Mac Intel | `accord-client-darwin-amd64-1e9025c.zip` |
| Linux x64 | `accord-client-linux-amd64-1e9025c.zip` |
| Linux ARM64 | `accord-client-linux-arm64-1e9025c.zip` |

Windows ARM64 native-клиент в этом пакете отсутствует. Проверьте SHA256 архива
по манифесту передачи, распакуйте свой архив, затем проверьте бинарник по
его `SHA256SUMS`. Например: `Get-FileHash .\agent-board.exe -Algorithm SHA256`
в PowerShell; `shasum -a 256 -c SHA256SUMS` на Mac;
`sha256sum --check SHA256SUMS` на Linux. Ожидаемое значение берите из файла,
а не из случайного сообщения. Проверки других ОС пока были cross-build,
нативный запуск подтверждён только на macOS arm64.

## 3. Разместить файлы и проверить HTTPS

Сохраните клиент в отдельном каталоге, например `~/accord-client` либо
`%USERPROFILE%\accord-client`. Приватные `connection.json` и `pc-2.token`
поместите в `~/.config/accord`. Не кладите их в checkout.

macOS/Linux, после переноса файлов:

```sh
mkdir -p "$HOME/.config/accord"
chmod 700 "$HOME/.config/accord"
cp ./connection.json "$HOME/.config/accord/connection.json"
chmod 600 "$HOME/.config/accord/pc-2.token" "$HOME/.config/accord/connection.json"
chmod 700 "$HOME/accord-client/agent-board"
read -r ACCORD_ORIGIN
```

Последняя команда ждёт ввода: вставьте **только значение поля `origin`** из
`connection.json`, затем Enter. Это адрес, не токен. Далее:

```sh
curl --fail --show-error --max-time 15 "$ACCORD_ORIGIN/readyz"
```

Windows PowerShell, после переноса файлов:

```powershell
$AccordPrivate = Join-Path $env:USERPROFILE '.config\accord'
New-Item -ItemType Directory -Force -Path $AccordPrivate | Out-Null
$AccordUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
icacls $AccordPrivate /inheritance:r /grant:r "${AccordUser}:(OI)(CI)F"
Copy-Item .\connection.json "$AccordPrivate\connection.json"
icacls "$AccordPrivate\pc-2.token" /inheritance:r /grant:r "${AccordUser}:F"
icacls "$AccordPrivate\connection.json" /inheritance:r /grant:r "${AccordUser}:F"
$AccordOrigin = (Get-Content -Raw "$AccordPrivate\connection.json" | ConvertFrom-Json).origin
(Invoke-WebRequest -UseBasicParsing -Uri "$AccordOrigin/readyz" -TimeoutSec 15).StatusCode
```

Проверьте `icacls "$AccordPrivate\pc-2.token"`: у других обычных учётных записей
не должно оставаться доступа. Команды снимают наследование, но не удаляют
ранее выданные явные права посторонним пользователям — их должен устранить
владелец файла через свойства «Безопасность». Не меняйте ACL чужих каталогов.

Ожидается HTTP200 с действительной проверкой TLS. Не применяйте `-k`,
`--insecure` и обход сертификата. Не заменяйте DNS-имя IP-адресом: проверка
Host точная. Readiness не требует bearer и ещё не доказывает доступ к задачам.

## 4. Подключить MCP к приложению агента

Рекомендуемый путь для **одного отдельного запуска исполнителя** — локальный
stdio worker `agent-board mcp`, работающий с существующим HTTPS-сервером.
Не запускайте `agent-board serve` на втором ПК.

Для Codex сохраните существующую конфигурацию и добавьте один раздел в
`~/.codex/config.toml`. Замените пути на абсолютные пути этого компьютера,
адрес — на `origin` из переданного `connection.json`:

```toml
[mcp_servers.accord_worker]
command = "/ABSOLUTE/PATH/accord-client/agent-board"
args = ["mcp", "--server", "ORIGIN_FROM_CONNECTION_JSON", "--token-file", "/ABSOLUTE/PRIVATE/PATH/pc-2.token"]
required = true
```

Это конфигурация macOS/Linux. Вручную запускать `mcp` в терминале для проверки
не требуется: он ожидает MCP JSON-RPC на stdin, его запускает приложение.

На Windows клиент использует окружение: POSIX-проверка private-файла не
подтверждает Windows ACL. Раздел в TOML:

```toml
[mcp_servers.accord_worker]
command = "C:\\Users\\YOUR_USER\\accord-client\\agent-board.exe"
args = ["mcp", "--server", "ORIGIN_FROM_CONNECTION_JSON"]
env_vars = ["AGENT_BOARD_TOKEN"]
required = true
```

Из подготовленного PowerShell запустите **новый отдельный Codex CLI**:

```powershell
$env:AGENT_BOARD_TOKEN = (Get-Content -Raw "$AccordPrivate\pc-2.token").Trim()
try { codex } finally { Remove-Item Env:AGENT_BOARD_TOKEN -ErrorAction SilentlyContinue }
```

Значение читается в память и не печатается. Не используйте CLI-параметр
`--env AGENT_BOARD_TOKEN=<secret>` и не печатайте окружение. Уже открытое GUI
не получает окружение этого терминала. Для Windows сначала проверьте CLI;
GUI и Claude Code требуют отдельной настройки/фактической проверки.

После изменения настроек перезапустите локальный клиент; проверьте
`codex mcp list` и наличие Accord в `/mcp`. Настройки stdio и `env_vars`
сверены с [официальной документацией Codex](https://learn.chatgpt.com/docs/extend/mcp).
Наличие записи в конфиге ещё не равно успешному соединению.

**Один worker — одна роль, одна board-сессия, один запуск исполнителя.**
После `describe` worker поддерживает heartbeat; завершайте его вместе с
исполнителем. Не используйте общий долгоживущий worker для независимых
параллельных агентов IDE. При смене роли нужен новый процесс.
Для приложения с общим MCP между агентами предусмотрен прямой HTTP `/mcp`:
каждому агенту отдельные `describe`, `session_id` и явный heartbeat по интервалу
сервера. Автоматический heartbeat там отсутствует. Настройка —
[client-setup.md](client-setup.md); одновременно оба режима для одной работы
не включайте. Для первого подключения используйте описанный worker.

## 5. Приёмка именно на втором ПК

Передайте агенту [готовое задание](receiver-agent.md). Минимальная проверка:

1. MCP initialize и tools/list успешны. Каталог live содержит12имён:
   `comment, create, describe, events, get, heartbeat, link, next, note,
   release, transition, update`.
2. Вызвать `describe` с `role=coordinator`, `machine=pc-2`, новыми уникальными
   `run_id` и `request_id`. Сохранить `session_id` без токена.
3. Через `create` создать **собственный** epic и сохранить выданный `E-N`:
   `{request_id:"NEW_UNIQUE_ID",type:"epic",title:"TEST pc-2 connection",description:"Проверка подключения",role:"",priority:2}`.
4. Прочитать его через `get {task_key:"E-N"}` и
   `events {task_key:"E-N",since_cursor:0,limit:100}`. Записать next_cursor.
5. Отменить **только созданный здесь** epic через `transition`:
   `{request_id:"NEW_UNIQUE_ID",task_key:"E-N",generation:0,state:"cancelled",reason:"synthetic cleanup"}`.
6. Завершить отдельный worker/исполнителя. Публичной команды мгновенного
   закрытия board-сессии здесь нет; после остановки heartbeat она истечёт по TTL.
   Если случайно взята своя synthetic task, сначала `release` с её точными
   `task_key`, `generation`, новым `request_id` и причиной. Чужие аренды не снимать.

Передавайте все required-поля схемы tools/list. У свежего собственного epic
нет task lease: для его отмены `generation=0`, а не generation чужой задачи.
Через stdio worker `session_id` можно опускать: он подставляет свою сессию.
Прямой HTTP требует session_id явно. Повтор того же request_id допустим только
для точного replay прежнего запроса; новый шаг получает новый идентификатор.

Отчёт: ОС/архитектура, версия клиента, SHA бинарника, дата, readiness status,
MCP initialize/catalog, созданный/отменённый ключ, next_cursor, завершение worker.
Не выводить токен или полный config с секретами. Затем с владельцем первого Mac
выполнить отдельный тест двух физических машин: одновременный `next` на одну
synthetic developer-task даёт ровно одного владельца; длительность>15минут,
heartbeat, release/истечение и новый generation проверяются совместно.
Первый одиночный smoke этот конкурентный сценарий не закрывает.

## 6. Если подключение не получилось

| Признак | Следующее действие |
| --- | --- |
| Нет инструмента Accord | Настроить MCP именно в приложении/на хосте агента, перезапустить его; не искать Tandem |
| DNS/timeout | Проверить Tailscale, ту же сеть/ACL, точный origin и порт8443 |
| Readiness200, MCP403 | Администратор сверяет точный external-host; токен не обходит Host/Origin |
|401 | Проверить срок/отзыв и файл нужной машины; новый токен выдаёт администратор |
| Machine mismatch | В describe должно быть ровно `pc-2` |
| SESSION_ALREADY_OPEN | Новый worker для новой роли/сессии |
| LEASE_UNCERTAIN/LEASE_LOST | Остановить локальные изменения, переподключиться и сверить владение; старое generation не продолжать |

Accord не останавливает редактор или shell при потере аренды. Раздельные Git
checkout и непересекающиеся области файлов согласуйте до параллельной работы.

## 7. Восстановить исходники и историю

В новом каталоге, рядом с переданным bundle:

```sh
git bundle list-heads ./accord.bundle
git clone ./accord.bundle accord
cd accord
git bundle verify ../accord.bundle
git checkout codex/pilot
git status --short
git log -1 --oneline
git branch -a
```

Если локальная codex/pilot не создана автоматически:
`git checkout -b codex/pilot origin/codex/pilot`.
При clone из bundle origin указывает на локальный файл. Для последующей связи
с GitHub, только если получателю поручена работа с ним:
`git remote set-url origin https://github.com/kirill-samylin/accord.git`.
Не отправляйте все ветки и не сливайте их в main автоматически. Online clone
одного GitHub-состояния не заменяет bundle с ещё не отправленными checkpoint.

Сначала прочитать `AGENTS.md`, [TASK-003](TASK-003-working-service.md),
[service-contract.md](service-contract.md), затем требования затронутого среза.
`process-reference/` содержит справочные инструкции ролей без необходимости
получать соседний Davinci. Приоритет у локального AGENTS Accord: Tandem off;
не запускать процессы/задачи Davinci. Фактические модели Claude в другой среде
не считаются доступными автоматически. Историю поручений и WORK сохранять.

## 8. Что работает, а что ещё не выпущено

| Срез | Версия и фактическое состояние |
| --- | --- |
| Общий сервер live C2, schema2 | `1e9025cd6e3122f0adf6165e09845202736d004a`; native bearer/MCP12, service и backup timer активны |
| `codex/pilot` | Основная передаваемая ветка с C2 и документацией; это не main и не автоматическая установка будущих срезов |
| `codex/performance-qa` | D4 checkpoint `f94c95d19ab7516511e32533f99b1c6430c705b3`; локальный make check PASS:37process и106infra; новые тесты072 получили независимый APPROVE076 |
| `codex/integration-config` | `d1d3fb87d4c6d55895a9b0701dd6ae1112121e75`; approved pure config/interchange, E runtime/schema5 отсутствуют |
| Новый bounded backup helper | Source approved065/074; live всё ещё прежняя политика168пар, новый helper не установлен |
| Второй физический ПК | NOT-RUN; с Mac новым pc-2 token проверены HTTPS и MCP initialize/catalog12, board-сессия pc-2 тогда не создавалась |

Окончательные HEAD и состав bundle сверять с манифестом передачи и TASK:
более поздний документационный commit может идти после указанных checkpoint.
**Полная V1 не завершена.** Старый целевой VPS-тест049: p95≈10025мс при цели50мс,
FAIL. Новые локальные оптимизации этого результата не отменяют; годовой объём
200тыс.событий и целевая VPS-приёмка D4 ещё не завершены. D4 на сервер не установлен.
GitHub OAuth, E runtime/webhooks/import/export wiring, часть native-контрактов071,
реальный Windows/Claude и два физических ПК остаются отдельными шагами.

## 9. Разработка и эксплуатация после передачи

Для сборки: Go≥1.26.1 (здесь фактически1.27.2), зависимости из go.mod/go.sum;
не обновлять их только ради запуска. Для make check нужны make, Python3 stdlib
и C toolchain для race detector. Сборка клиента из корня checkout:

```sh
go build -o bin/agent-board ./cmd/agent-board
```

Windows: имя `agent-board.exe`. Проверки: `go mod verify`,
`make check`. Окружение/кэши — [development.md](development.md); локальный Go
из `.local/toolchains` в переносимый Git-пакет не входит. Для тестов не брать
живую БД и не запускать maintenance reset/migration на общем сервере.

Одна общая рабочая БД остаётся на VPS. Root-owned releases лежат в
`/opt/accord/releases`, current — `/opt/accord/current`, состояние —
`/var/lib/accord`, snapshots — `/var/backups/accord`. Эксплуатация и отзыв токенов —
[deployment.md](deployment.md); новые условия хранения —
`docs/backup-retention.md` в ветке `codex/backup-budget` (в pilot не установлены).
Для клиентского подключения SSH не нужен.
Администратор передаёт административный доступ отдельным защищённым порядком.
Backup на том же VPS не заменяет offsite; подтверждённого offsite нет.

Следующий координатор принимает незавершённый этап из TASK, сохраняет чужие
изменения, назначает непересекающиеся области и отдельного ревьюера. Перед
выпуском нужны проверки конкретного итогового SHA и совместимости схемы.
В рамках обычной передачи не менять VPS, лимиты, HTTPS443, Tandem, main,
данные и credentials других машин. Установка нового кода и перенос владения
сервером выполняются как явно отдельная согласованная операция.
