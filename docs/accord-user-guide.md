# Accord: от требования до результата

Выберите **«Проекты» → Test_internet → «Выполнить требование»**. Здесь требование
проходит аналитика, разработчика, независимых ревьюера и тестировщика; вы можете
приостанавливать работу, уточнять задание, отвечать на вопросы и продолжать цикл.

**Срез 24.10.2026:** Mac/Test_internet подключён: фоновый worker онлайн,
четыре профиля и ограниченное разрешение созданы. Первое Python-требование
сохранено без запуска и ожидает доверия владельца к папке проекта.
Рабочая веб-проверка прошла, поставка сервера завершена с сохранением проектов.
Файлы standalone client231V2 установлены на Mac; оба флага запуска пока false,
клиент и модель им не запускались. Индивидуальные ID и закрытые пути оператор
передаёт лично. Версии: сервер/Next a32db22, Mac worker be95dae.
Текущий checkpoint: [TASK014](TASK-014-automation.md).

## Карточка подключения

| Что получить от оператора | Значение |
|---|---|
| HTTPS-адрес и установленная версия Accord | Заполнить после поставки |
| Проект | `Test_internet`; точный project ID из индивидуального подключения |
| Свой token file и `machine` | Каждому участнику отдельно защищённым каналом |
| Рабочая копия / alias папки | Фактический путь / подготовленный `test_internet` |
| Worker для своей ОС, версия и SHA | Заполнить по установленному artifact |
| Host / версия согласия / policy ID и версия | Получить из фактически подключённого проекта |
| Установка, доверие real Test и первый результат | Пока не заполнено — не подтверждено |

Токен Mac-владельца, state/host proof и вход провайдера второму человеку не передают.
У каждого свои credentials. Временное разрешение QA не распространяется на обычную
папку Test_internet; `launch_ready:false` нельзя включать только по наличию файлов.

## Один раз подготовить компьютер и команду

Для просмотра достаточно собственного доступа к проекту. Чтобы управлять чужим
ПК, нужны разрешение его владельца и разрешение команды; роль «Редактор» этого
не даёт. Чтобы выполнять роли на своём ПК, дополнительно нужны Git-копия проекта,
worker и native Codex/Claude с собственным входом. Для одной отправки требования
из MCP-клиента собственный worker не обязателен: роли выполнит разрешённый host.

1. Оператор готовит private `worker.json`: свой origin/project/token file/state,
   абсолютные пути проекта/CLI и разрешённые providers/controllers.
   На Mac каталог0700/файлы0600; на Windows настоящие DACL и native `.exe`.
   Шаблоны: [worker guide](managed-worker-guide.md).
2. Владелец отдельно подтверждает доверие этой рабочей копии обычным способом CLI.
   Настройки общего sandbox/approval и существующего GUI для этого не переписывают.
3. Настройте реальные тесты и локальное согласие: [automation setup](automation-worker-setup.md).
   В Python-старте есть два unittest и JUnit runner; исходный `NotImplementedError`
   должен дать два errors/exit1, а не PASS. Проверки не заменяют печатью «успешно».
4. Выполните команды установленным клиентом с абсолютным `PRIVATE_WORKER_JSON`:

```text
agent-board worker identity --config PRIVATE_WORKER_JSON
agent-board worker automation-preview --config PRIVATE_WORKER_JSON
agent-board worker enroll --config PRIVATE_WORKER_JSON
agent-board worker consent --config PRIVATE_WORKER_JSON
agent-board worker run --config PRIVATE_WORKER_JSON
```

На Windows вместо `agent-board` используйте `& 'C:\ABSOLUTE\agent-board.exe'`.
`identity` создаёт локальное состояние/сессию и возвращает project-local код.
Preview только предлагает hashes/digests (`applied:false`, `requires_local_apply:true`);
владелец проверяет и переносит их в config перед enroll/consent. Чужие QA digests
и GUI ID не подходят. Новый host enroll выполняют один раз; существующий обновляют
через consent после подтверждённой остановки и штатного завершения worker.

Owner-only запуск не ждёт второго ПК: владелец управляет своим host при пустом
`controllers`. Для взаимного управления обменяйтесь project-local кодами именно
Test_internet, добавьте второго участника в локальные controllers на каждом ПК,
выполните consent и обновите policy в UI. Третий участник автоматически не добавляется.
Оставьте worker работающим; `worker run` не устанавливает автозапуск. Если оператор
установил отдельный Test_internet LaunchAgent, второй worker с тем же state не запускают.

На проверенном Mac отдельный LaunchAgent использует явный `HOME` и закрытую
копию собственного token file в папке настроек worker. Исходный файл сохранён;
доступ ко всему диску не выдавался. Это проверенное подключение одного Mac,
а не подтверждение автозапуска на Windows. Не переносите этот token второму человеку.

В приложении откройте **«Выполнить требование» → «Подготовить команду»**, создайте
четыре профиля ролей. Затем **«Настроить запуск»**: выберите **«Компьютер»**,
**«Рабочая папка»**, **«Принять текущее согласие · версия …»**, provider и профиль/
версию для каждой роли, реальные проверки, участников, сроки и лимиты.
Для клиента включите **«Разрешить задания из Codex / Claude указанным участникам»**.
Первый подготовленный вариант — owner-only, четыре Codex с моделью по умолчанию;
Claude выбирают только при подходящем локальном provider/профиле.

Стартовый план: до7дней согласия,300с/попытку,1800с/цикл,12 запусков, одна задача,
до2 исправлений — фактические сроки ограничены credential/local consent.
Один цикл закреплён за одним host/checkout; роли не распределяются между ПК.
Основная страница: `/app/p/PROJECT_ID/automation`; разрешение —
`/app/p/PROJECT_ID/automation/policies/POLICY_ID`, требование —
`/app/p/PROJECT_ID/automation/workflows/WORKFLOW_ID`. ID берите из своей ссылки,
версию — из карточки разрешения; чужую QA policy использовать нельзя.
Git/тестовые разрешения принимаются локально; push/merge/deploy в них не входят.

## Подать требование из приложения

После фактической подготовки Test_internet используйте выбранный Python-пример:

> Реализовать add(left, right) в src/math_ops.py для положительных, отрицательных
> и нулевых целых. Сохранить два фиксированных теста и runner. Сделать один
> реальный локальный developer commit. Независимые review и QA должны проверить
> точный SHA с project-tests/JUnit. Не менять другие проекты и настройки,
> не выполнять push/merge/deploy. Готовность: реальные тесты проходят,
> review approve и QA passed относятся к этому SHA.

Вставьте текст в **«Требование»**, выберите актуальную **«Команда и разрешение»**,
нажмите **«Начать выполнение»**. **«Сохранить черновик без запуска»** только сохраняет текст.
В карточке **«Ход выполнения»** показывает роли, попытки, задачи и процессы;
**«Результаты и подтверждения»** — план, SHA, review и тесты. Замечания запускают
исправления в пределах разрешения. Старт процесса или ответ модели не равен готовности.

ПК должен оставаться включённым, worker — онлайн; закрытие браузера цикл не отменяет.
Результат остаётся в выбранном локальном Git. Доска не синхронизирует файлы двух ПК;
передача коммитов, push/merge/deploy — отдельные действия.

## Управлять и отвечать на вопросы

| Действие | Что произойдёт |
|---|---|
| **Пауза** | Запрос остановки. Дождитесь подтверждённого состояния процесса: очередь/offline/«отправлено» не означают stopped. |
| **Передать уточнение** | Текст из «Уточнение или причина» доставляется на следующем ходе; для прерывания текущего сначала пауза. |
| **Продолжить** | После остановки использует точный сохранённый native диалог и новую сессию/аренду задачи. Потерянный диалог не подменяется новым чатом. |
| **Пересмотреть план** | Явно пересматривает план с учётом результатов, если меняется само требование. |
| **Отменить цикл** | Запрашивает выход процессов и завершение цикла; проверьте подтверждение и освобождение задач. |

В **«Вопросы команде»** назначенный человек заполняет **«Ваш ответ»** и нажимает
**«Сохранить ответ»**. Вариант ответа только заполняет поле. Флажок **«Продолжить
работу после сохранения ответа»** изначально выключен: можно сохранить ответ
без нового запуска, затем продолжить отдельно при наличии прав.
Агент не отвечает за человека; обычное resume открытый вопрос не закрывает.

При смене состояния обновите форму и примите текущую версию; после изменения
consent/policy явно выберите актуальное разрешение при продолжении. Если исход
отправки неизвестен, сначала сверьте состояние. Не создавайте дубликат новым запросом.
Доставку команды и состояние процесса смотрите отдельно. При «Требуется восстановление»
обратитесь к владельцу ПК: удаление state/запуск второго worker не доказывает остановку.

## Отправить требование из нового Codex / Claude

Это отдельный CLI-процесс с MCP, не автоматическое подключение текущего desktop-чата.
После реальной установки принятого client231V2 владелец Mac может запускать:

```sh
python3 -I -B "$HOME/.config/accord-test-internet/client/launch-client.py" codex
python3 -I -B "$HOME/.config/accord-test-internet/client/launch-client.py" claude
```

Нужен Python3.11+. Оператор проверяет executable paths/SHA, реальную policy,
`launch_ready` и effective MCP/plugins/Tandem. Launcher не меняет global registration;
Codex-координатор получает read-only sandbox. У Claude остаются обычные approvals,
что само по себе не является read-only ОС-изоляцией.

Поддержанная команда stdio bridge (подставьте значения из подключения):

```text
agent-board mcp --server https://ACCORD_ORIGIN --project p_PROJECT_ID --token-file /ABSOLUTE/PRIVATE/OWN.token
```

У `mcp` нет `--config`.
На Windows standalone принят процессный `AGENT_BOARD_TOKEN` из собственного
DACL-защищённого файла. В private `claude-mcp.json` оператор подставляет свои значения:

```json
{"mcpServers":{"accord_test":{"type":"stdio","command":"C:\\ABSOLUTE\\agent-board.exe","args":["mcp","--server","https://ACCORD_ORIGIN","--project","p_PROJECT_ID"]}}}
```

Windows + Claude, после заполнения путей:

```powershell
Set-Location 'C:\ABSOLUTE\Test_internet'
$ownToken = 'C:\ABSOLUTE\receiver.token'
$env:AGENT_BOARD_TOKEN = (Get-Content -Raw $ownToken).Trim()
try { claude --strict-mcp-config --mcp-config 'C:\ABSOLUTE\claude-mcp.json' }
finally { Remove-Item Env:\AGENT_BOARD_TOKEN -ErrorAction SilentlyContinue }
```

Windows Codex: оператор сначала проверяет effective MCP и заполняет только свои
точные отключаемые MCP keys; значения не переносят с Mac. Этот блок намеренно
закрыт до проверки. Он меняет только параметры нового процесса:

```powershell
$receiverMcpReviewed = $false # true только после проверки MCP/Tandem
$receiverDisableKeys = @() # mcp_servers.NAME.enabled / plugins.PLUGIN.mcp_servers.NAME.enabled
if (!$receiverMcpReviewed) { throw 'Сначала проверьте MCP namespace' }
$disabledOverrides = @()
foreach ($key in $receiverDisableKeys) {
  if ($key -cnotmatch '\A(?:mcp_servers\.[A-Za-z0-9_-]+\.enabled|plugins\.[A-Za-z0-9_-]+\.mcp_servers\.[A-Za-z0-9_-]+\.enabled)\z' -or (($key -split '\.')[-2] -ceq 'accord_test')) { throw 'Только точные чужие MCP enabled keys' }
  $disabledOverrides += @('-c', "$key=false")
}
$bridge = 'C:\ABSOLUTE\agent-board.exe'
$workspace = 'C:\ABSOLUTE\Test_internet'
$ownToken = 'C:\ABSOLUTE\receiver.token'
$bridgeJson = ConvertTo-Json -Compress -InputObject $bridge
$argsJson = ConvertTo-Json -Compress -InputObject @('mcp','--server','https://ACCORD_ORIGIN','--project','p_PROJECT_ID')
$mcp = 'mcp_servers.accord_test={command=' + $bridgeJson + ',args=' + $argsJson + ',env_vars=["AGENT_BOARD_TOKEN"],enabled=true}'
$env:AGENT_BOARD_TOKEN = (Get-Content -Raw $ownToken).Trim()
try { & codex @disabledOverrides -c $mcp --cd $workspace --sandbox read-only }
finally { Remove-Item Env:\AGENT_BOARD_TOKEN -ErrorAction SilentlyContinue }
```

Не печатайте token и не сохраняйте его в profile/JSON. В новом клиенте поручите
**только координацию**: `describe` с role=`coordinator`, своей `machine`, новыми
`run_id/request_id`; затем `workflow_submit` с реальными `policy_id/policy_version`,
новым `request_id` и `text`. Сохраните workflow ID; код параллельно с managed командой не пишите.
Один bridge — одна сессия, session подставляется мостом. Выбор проекта в браузере
не меняет project ID этого процесса.

Чтение: `accord://workflows?limit=50`, `accord://workflow?workflow_id=ID`,
`accord://workflow-report?workflow_id=ID`. `workflow_get` — resource/read, не tool.
Управление: `workflow_command` с текущими `workflow_id/version`, `request_id`,
`kind=pause|clarify|resume|cancel`, при необходимости `text`; не вымышленный `stop`.
При неизвестном исходе сохраняйте request_id и сверяйте результат; новый ID может
создать другой workflow. На человеческий вопрос отвечайте в UI.

## Второй участник и границы проверки

Сначала проверьте его собственный вход/проект/describe, затем отдельно отправку
требования и разрешённое управление; выполнение на его ПК требует своего worker.
Для взаимного управления каждый владелец подтверждает consent на своей машине.
**Native Windows, native Claude и физический второй ПК — NOT-RUN** до проверки
на этих машинах. Mac QA/сборка `.exe`/исходный config не доказывают такую установку.
Оператор отдельно фиксирует реальные версии, installation/sample PASS/FAIL/NOT-RUN.

Подробнее: [автоматизация](automation-guide.md), [worker](automation-worker-setup.md),
[WORK224](work/WORK-224-test-internet-onboarding.md), [client231V2](work/WORK-231-V2-client-guide-hardening.md).
CLI-флаги приведены для принятого пакета231V2; совместимость других версий заново
здесь не проверялась. Его источники: [Codex CLI](https://learn.chatgpt.com/docs/developer-commands),
[Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli), [Claude CLI](https://code.claude.com/docs/en/cli-reference).
