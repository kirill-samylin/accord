# WORK296 — `worker setup`: подключение компьютера без ручного JSON

Исполнитель developer, ветка `codex/worker-setup` от `codex/native-preflight`
(6062e0d). **UNIT PASS / make PASS / web-check PASS / реальный enroll/consent
на Mac NOT-RUN**. Поручение пользователя 26.10.2026: сделать команду
`agent-board worker setup --config PRIVATE_JSON`, которая проводит владельца по
подключению компьютера и провайдеров (включая gennadiy) без правки JSON
руками, затем сама выполняет enroll/consent и перезапускает LaunchAgent.

## Что было

Подключение нового провайдера — ручной путь: остановить LaunchAgent, заменить
бинарь, дописать `executables`/`providers`/режим прав/
`native_mcp_approved_tools`/`models` в `worker.json`, выполнить `worker
consent`, снова запустить службу. `Config.Validate` принимает только
абсолютные пути к обычным файлам; для gennadiy настоящий Mach-O лежит в
npm-пакете `gennadiy-darwin-arm64`, а `~/.gennadiy/bin/gennadiy` и
`/opt/homebrew/bin/gennadiy` — шимы.

## Реализованный объём

`cmd/agent-board`:

- `worker_setup.go` (новый): операция `worker setup --config PRIVATE_JSON`.
  Единственный аргумент — путь к config; stdin обязан быть терминалом, ответы
  из pipe/аргументов не принимаются. `worker.go` диспетчеризует `setup`,
  `main.go` — строка в `--help`.

`internal/worker` (все файлы ≤300 строк):

- `setup_discover.go`: `setupProbe` ищет CLI в `PATH` и типичных местах
  (бандл ChatGPT, `~/.codex/bin`, `~/.claude/local`, `~/.local/bin`,
  `~/.volta/bin`, `~/.nvm/versions/node/*/bin`, Homebrew, `~/.gennadiy`).
  Кандидат канонизируется (`EvalSymlinks`), скрипты (`#!`, `.cjs`, `.js`,
  `.cmd`, `.bat`) отклоняются; для gennadiy npm-launcher
  `node_modules/gennadiy/bin/gennadiy.cjs` разрешается в
  `node_modules/gennadiy/node_modules/gennadiy-<os>-<arch>/bin/gennadiy`.
  Bash-шим `~/.gennadiy/bin` сам ничего не содержит и пропускается. Версия —
  один bounded `--version` через `readNativeWithEnv` (5с, `providerEnv`,
  для gennadiy плюс `OPENCODE_DISABLE_*`). `PATH`/`HOME`/системные каталоги
  и reader версии инъектируются для тестов.
- `setup_prompt.go`: line-oriented prompter (`ask`/`confirm`/`choose`),
  EOF до ответа — ошибка, а не согласие по умолчанию.
- `setup_connection.go`: шаг 1 для нового config — путь к токену (проверка
  `ReadToken`, содержимое не печатается), сервер (`https://` или loopback),
  проект, machine, имя компьютера, `state_dir` (по умолчанию `<dir>/state`).
  Несекретные значения предлагаются из `connection.json` рядом с токеном.
- `setup_dialogue.go`: шаги 2–3 — выбор executable на провайдера (найденные,
  «оставить текущий», свой абсолютный путь через тот же `resolve`), рабочие
  папки (существующие сохраняются, провайдеры переспрашиваются; новые —
  канонический путь, alias из имени папки), итог и подтверждение записи.
- `setup_options.go`: шаги 4–6 — режимы прав с пояснением последствий,
  наборы `native_mcp_approved_tools` («только чтение» `describe,next,get,
  events`; «работа по задаче» плюс `update,comment,note,transition,release,
  link`; «ничего»; «оставить текущий»), `models` через запятую.
- `setup_write.go`: `readSetupConfig` (существующий private файл читается без
  `Validate`, чтобы отредактировать config с устаревшим бинарём; испорченный
  файл не перезаписывается), `writeSetupConfig` (`Validate` →
  `MarshalIndent` → `.next` `O_EXCL` 0600 → fsync → rename → fsync dir;
  новый каталог `0700`), `replaceExecutable` (копия текущего бинаря в
  `.next` 0700 → rename).
- `setup_service.go`: LaunchAgent `com.accord.gui.worker` из
  `~/Library/LaunchAgents` (только darwin), `plistProgram` извлекает
  `ProgramArguments[0]`, `launchctl bootout gui/UID/label` и
  `launchctl bootstrap gui/UID plist`.
- `setup.go`: оркестрация `Setup`: read → discover → dialogue → `Validate` →
  preflight `probeProviders` (`--version`/`--help`) → write → остановка
  службы (по подтверждению; без неё enroll/consent невозможны из-за lock
  state) → опциональное обновление бинаря (по подтверждению, только если
  файл другой) → `Engine.Enrolled()` ? `Consent` : `Enroll` → запуск службы
  снова даже при ошибке consent. `Engine.Enrolled()` — новый метод.

Документация: раздел «Быстрое подключение» в `docs/managed-worker-guide.md`,
абзац в AGENTS.md, эта карточка.

## Границы и допущения

- `controllers` команда не меняет: код второго участника добавляется вручную,
  как раньше. `automation` тоже сохраняется как есть; если выбранный набор
  инструментов противоречит automation-ролям, `Validate` отклоняет config и
  файл не пишется.
- Consent не выполняет `probeProviders` в самом worker, поэтому setup делает
  preflight до записи. Pins нового executable обновляет сам `sendConsent`.
- Команда не устанавливает LaunchAgent там, где его нет: тогда печатается
  напоминание про `worker run`.
- Windows/Linux: обнаружение и запись работают, служб нет (`findWorkerService`
  возвращает nil); реальный прогон там NOT-RUN.
- Дубликаты одного CLI в разных префиксах (например gennadiy 1.18.10 в nvm и
  1.4.5 в Homebrew) показываются оба; выбор за владельцем.

## Фактические проверки

- Unit (`go test ./internal/worker -run 'TestSetup|TestPlist'`) — PASS:
  обнаружение на фиктивном дереве (бинарь codex, скрипт claude отклонён,
  npm-launcher gennadiy → платформенный бинарь, bash-шим пропущен, дедупликация
  symlink-кандидата), диалог со скриптованным stdin (новый config из подсказок
  `connection.json`, редактирование существующего с сохранением
  server/project/token_file/state_dir/controllers и текущих tools, повторные
  вопросы при плохих ответах, отмена, EOF), оркестрация (файл 0600 и каталог
  0700, `LoadConfig` читает результат, порядок `bootout` → connect →
  `bootstrap`, обновление бинаря только по подтверждению, перезапуск службы
  при ошибке consent с подсказкой `worker consent`, нетронутый файл при отмене
  и при провале preflight, отказ для файла 0644), разбор plist.
- Локальный `ACCORD_TEST_SETUP_DISCOVER=1 go test -run TestSetupDiscoverInstalled`
  на этом Mac — PASS: найден codex 0.164.0-alpha.11.1 из бандла ChatGPT,
  gennadiy 1.18.10 (nvm) и 1.4.5 (Homebrew) как настоящие
  `gennadiy-darwin-arm64/bin/gennadiy`, claude не установлен. Только чтение.
- `make NEXT_NODE=<node 24.19.0> web-check` (prettier, tsc, 131 web-тестов,
  `next build`) — PASS; веб не менялся.
- Реальный бинарь через псевдотерминал (Python `pty`, scratch-токен и
  scratch-папка вне репозитория): все шесть шагов диалога, реальное
  обнаружение (codex из бандла ChatGPT, два gennadiy, claude не найден),
  итог и отмена на вопросе о записи — файл не создан, exit 1 с
  «setup cancelled». Без TTY команда отказывает; лишние флаги отклоняются.
- `make fmt-check vet test race build infra-test` (toolchain Go1.27.2 из
  `.local/toolchains`) — PASS дважды: до и после добавления ожидания
  освобождения state lock (`openWhenReleased`, до30с после `bootout`).
- NOT-RUN: интерактивный `worker setup` на реальном `worker.json` этого Mac
  (остановка LaunchAgent, consent с gennadiy) — по указанию пользователя
  локальный worker не менялся; реальный managed запуск gennadiy; Windows.
