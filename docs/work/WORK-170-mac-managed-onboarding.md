# WORK-170 — подключение Mac к управлению Давинчи 2.0

Автор: managed_worker. Область: только локальная private подготовка worker,
его user LaunchAgent и эта карточка. Production обновляет координатор;
GUI checkout, старый GUI bridge и Tandem не изменяются.

## Новое поручение и границы

Пользователь явно поручил обновить production Accord и подключить свой Mac для
запуска и управления агентами Давинчи 2.0 из панели. Это отдельное основание для
локального scoped подключения; прежнее согласие на synthetic describe/next не
переносится. Координатор подтвердил необходимый для этой задачи режим:
Codex workspace-write для одного GUI checkout и следующие 10 инструментов
только временного `accord_managed` connector:

`describe`, `next`, `get`, `events`, `update`, `comment`, `note`, `transition`,
`release`, `link`.

Создание новых задач и ручной heartbeat не разрешены модели. Внутреннее
поддержание уже выданной аренды bridge сохраняется. Global approval policy,
sandbox bypass, native auth и пользовательские Codex настройки не меняются.
Новая реальная проверка — чтение GUI без продуктовых правок; workspace-write
подготавливает требуемую последующую работу, но не разрешает QA менять GUI.

## Проверка существующего состояния

- Существующие GUI connection/token находятся в приватном каталоге владельца.
  Token не копировался и не публиковался. Фактический origin остаётся только
  в private config, без переноса в эту карточку.
- Старый установленный GUI bridge понимает schema4, сохранён на прежнем месте.
  Worker установлен отдельно в `~/.config/accord-gui/managed/`.
- Два действующих project-local principal ID сопоставлены с registry identity
  metadata каждого участника по фактическому server membership export.
  Единственный additional controller — второй участник; owner управляет своим
  host автоматически. Registry identity ID не используется вместо principal ID.
- Native CLI: Codex `0.163.0-alpha.11.1`, абсолютный canonical путь из текущего
  приложения. GUI directory и executable без symlink components, владелец —
  текущая macOS учётная запись. Filesystem identity GUI установлена.
- На момент prelaunch проверки нет workspace lock для этого checkout и нет
  процесса agent-board. Уже открытые Codex процессы не подключаются и не убиваются.
- GUI ветка содержит прежние незакоммиченные изменения. Перед проверкой сохранён
  private hash baseline 3316 tracked/untracked файлов, исключая Git ignored.
  Проверка baseline не является доказательством отсутствия внешних писателей.

## Подготовленные файлы

Private runtime вне Git:

- `~/.config/accord-gui/managed/agent-board` — новый отдельный Mac binary;
- `~/.config/accord-gui/managed/worker.json` — scoped config, additional controller,
  один workspace alias `gui`, Codex executable и exact10 permissions;
- `~/.config/accord-gui/managed/state/` — пока пустой private state;
- `~/.config/accord-gui/managed/logs/` — private stdout/stderr файлы;
- `~/.config/accord-gui/managed/com.accord.gui.worker.plist` — пока НЕ загруженный
  LaunchAgent template.

Каталоги `0700`, config/plist/logs `0600`, executable `0700`, владелец текущий
пользователь. Старые token, connection, GUI MCP launcher/binary сохранены.
Исходные staged configs, proofs и GUI baseline лежат только в игнорируемой
`.local/task013-mac-onboarding/`; их полное содержимое не предназначено для Git.

LaunchAgent `com.accord.gui.worker` использует абсолютные paths к binary/config,
обычную user session, explicit HOME/CODEX_HOME/PATH с установленными node/go/git,
без token/credential environment. `RunAtLoad` и `KeepAlive` поддержат отдельный
worker после включения; `ThrottleInterval=30`, `ExitTimeOut=15`. SIGTERM штатно
останавливает принадлежащую worker CLI process group; аварийная неоднозначность
не обнуляет journal/checkout intent и не выдаёт автоматический takeover.
Unix detached descendants остаются известной границей TASK013.

## Версия и проверки до включения

Чистый release worktree: `273ccbb15c99dfac91c37f34d2926abf01ca866f`.
Mac build: Go1.27.2, CGO_ENABLED=0, darwin/arm64, trimpath, buildvcs=true.
`go version -m` подтверждает точный revision и `vcs.modified=false`;
clean status проверен до и после build. Binary SHA256:
`e3d4caf6ac13621702ba2ef84155b399d46fe14546b0969611fb7176050e2d3a`.

PASS: capabilities board_schema5/managed_agents1 плюс прежние floors;
plist parse/lint; private paths/owners/modes; точное metadata→principal mapping;
сохранение старого GUI bridge hash; сохранение GUI baseline.
Проверки read-only/local preparation, без enrollment, provider conversation,
запуска worker или загрузки LaunchAgent.

NOT-RUN до production capable apply, независимого WORK169 review и root GO:
identity на обновлённом сервере, enrollment, launchd bootstrap, реальный native
read-only GUI smoke из панели, stop/continue/clarification. Их выполнит отдельный
QA после approve; эта карточка не объявляет подключение работающим до результата.
Windows Claude и второй физический ПК этим Mac подключением не проверяются.

## Фактическое включение после approve/GO

После production capable apply координатор дал явный GO на enrollment и
user LaunchAgent. Независимый WORK169 Mac PRELAUNCH APPROVE получен до запуска.

PASS: live `worker identity` совпал с ожидаемым project-local owner principal из
private membership export. Дополнительный controller — точный active editor
второго участника. Выполнен один enrollment; состояние и host proof сохранены
приватно. Ни secret, ни token не копировались в эту карточку.

LaunchAgent установлен в user `Library/LaunchAgents` и загружен через launchd
GUI domain текущего пользователя. Фактический process принадлежит этому
пользователю и использует отдельный clean273 binary; первая стабильная instance,
без restart loop. Фоновая служба работает независимо от открытого терминала.
Private stderr пуст на момент проверки.

PASS: live HTTPS management read показывает active host, свежую worker lease,
совпадение owner principal, can_control, exact additional controller, provider
Codex и workspace alias gui. Локальный journal без native runs/pending reports.
Сразу после включения весь baseline3316 GUI файлов совпал с подготовленным:
файлы GUI подключением не изменены.

Private operational evidence:
`.local/task013-mac-onboarding/live-identity-private.json`,
`live-enrollment-private.json`, `live-launchagent-proof.json`,
`live-health-private.json`.

Реальный native GUI smoke, управление через браузер и проверку stop/continue/
clarification выполняет отдельный QA WORK172. На момент этой карточки native
provider для GUI автором не запускался. Windows/второй физический ПК остаются
NOT-RUN; фоновая служба Mac сама их не подключает.

## Первый live native запуск: FAIL до conversation

Независимый QA WORK172 выполнил один запуск из интерфейса. Run завершился
ошибкой подготовки isolated native MCP connection до создания provider session;
задача осталась ready без lease/attempt. Это не успешный native GUI запуск.
Worker остался online; новых запусков автор не выполнял.

Read-only диагностика: environment фактического worker прочитан через собственный
process environment API без вывода credential values. В exact GUI cwd с этим
окружением config-only Codex MCP discovery завершился exit0, valid JSON для
шести stdio connectors, stderr пуст. Config-only повтор с точными whole-map
10-tool overrides и immutable bridge file неудавшейся попытки тоже exit0:
managed command/args совпадают, все остальные connectors disabled; private
bridge contents/права соответствуют config. Эти пробы не вызывают модель или
MCP tools и не меняют provider/global settings.

Private proofs: `.local/task013-mac-onboarding/diagnostic/discovery-proof.json`
и `verification-proof.json`. Native readNative ownership/Wait остаётся гипотезой,
не установленной причиной; generic prepare error пока скрывает конкретный stage.
Сигнальные/ownership проверки не ослаблялись. Source/config/service не менялись.
По новому порядку пользователя диагностика native временно отложена до публикации
репозитория; worker оставлен idle running, новых native starts нет.
