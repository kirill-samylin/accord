# WORK-159 — подключение «Давинчи 2.0» к Accord

22.10.2026. Координатор. Поручение: найти локальный GUI, подключить общий проект,
подготовить второго участника и проверить работу. Пользователь подтвердил
`/Users/shagi/Desktop/Project/davinci-gui-`, второй человек — Windows + Claude.
Инструкция рассчитана на Claude Code; Claude Desktop chat — отдельная среда.
Следующее уточнение разрешило убрать неиспользуемый Tandem из GUI.

## Области и независимость

Автор158: общая инструкция и точечные GUI AGENTS/CLAUDE/parallel-work плюс
`docs/agents/accord.md`. Reviewer160: независимый source/config/ops/test review.
Tester161: новые process/browser проверки после source approve. Root: private
операции, MCP settings, package, эта карточка. Прежние dirty изменения GUI и
untracked `deploy-task012.command` в Accord сохранены; соседний продуктовый код
не менялся. Remote push/merge/release не выполняются.

До работы прочитаны requirements/architecture, multi-project, profiles,
coordination и TASK012. Действующий серверный release —
`64ed870b749289b545d29cd2ec7813c6266de54d`, board schema4 с полным capability
floor registry/ACL/profiles/coordination/deduplication. Управляемый запуск в эту
задачу не входит.

## Выполненное подключение

- Новый общий проект «Давинчи 2.0», ссылка на действительный Git remote GUI,
  branch `main` по локальному `origin/HEAD`. Repo association не делает clone
  и не выдаёт Git-доступ.
- Два новых catalog-only credentials: Mac — owner GUI, коллега — editor GUI.
  Прежние credentials/ACL не заменялись; default доступ новым не выдавался.
- Offline issuance прошла source approve. Текущий capable binary, effective
  launch, admin/maintenance locks, owned startup hold, full verified format2
  snapshot до выдачи, сравнение прежних registry и всех board/web rows — PASS.
  Только две additive identity/credential записи, без изменения прежних строк.
  После проверки writer восстановлен; binary/release/guards не заменялись.
- Проект и member grant созданы через штатные web actions с CSRF и issued
  action IDs. Private cookies/payload сохранялись для exact retry; чужие
  projects/ACL не изменялись.
- GUI `.codex/config.toml` использует private launcher и точный project scope.
  Shared `.mcp.json` содержит только `${ACCORD_*}`/`${AGENT_BOARD_TOKEN}` для
  Claude Code. Private credential/connection/client — вне checkout в
  `~/.config/accord-gui`; права каталога0700 и файлов0600, binary0700.
- Неиспользуемый Tandem удалён из активных MCP/AGENTS/CLAUDE/parallel-work.
  Его untracked connector перемещён с рабочего пути в ignored private backup,
  старый guide помечен архивным. `.local/tandem/ssh` используется Accord admin
  transport и не удаляется; серверный старый сервис/данные не сносились.

## Проверки и пакеты

- PASS: независимый SOURCE APPROVE158/160; actual local config/privacy/hash
  review; diff whitespace обоих проектов.
- PASS: actual Codex CLI прочитал проектную настройку `accord_gui`, обнаружил
  12tools и8resource templates; Tandem отсутствует.
- NOT-RUN/BLOCKED CLIENT APPROVAL: тот же CLI не выполнил `describe`: локальная
  policy `never` запрещает требующий подтверждения MCP-вызов. После этого
  resource reads вернули NO_SESSION. Это не проверка успешной agent session;
  политика не обходилась. CLI exit0 не трактуется как business PASS.
- PASS: Windows/amd64 client cross-build из clean exact source64ed, Go1.27.2,
  `vcs.modified=false`; SHA256
  `00a1a61df1ca3c723151fdcb570c2cae2a9ac84259f3facbeea3c0cb7659b612`.
  Actual Windows/Claude runtime — NOT-RUN.
- Mac client: проверенный TASK012 candidate4742e71, SHA256
  `f17b4bb8dfa06dbdd9168687beb79c2fa4ce9b33579f6e58cca86932738bbfb5`.
  Diff4742→64ed содержит только docs/tests; cmd/internal product code идентичен.
- Независимые actual HTTPS/bridge/browser проверки161 и review новых тестов
  выполняются; результаты будут дописаны после получения evidence.

## Границы управления

Доступны задачи, назначения, комментарии/уточнения, сессии, lease ownership и
экран «Команда». Освобождение/повторный claim задачи не означает остановку или
возобновление OS-процесса. Отзыв credential ограничивает дальнейшие серверные
операции, но не способен физически остановить локальные правки. Удалённый
stop/start/resume Codex/Claude, push указаний работающему агенту и managed
workers — NOT IMPLEMENTED. Физический второй ПК — NOT-RUN; полный V1 не заявлен.

## Независимая проверка сохранности и пакет

Reviewer160 независимо проверил live сервер read-only после подключения:
snapshot manifest и capability marker совпадают; прежние registry_meta,
identities, credentials, projects, repositories, memberships сохранены exact
subsets. Строго ожидаемые additions:0/2/2/1/1/2; новые роли owner/editor
относятся только к одному GUI. Exact64ed, effective launch/backup, owned Go
PID/executable/argv и активный Next — PASS.

Public Windows x64 пакет создан в ignored `.local/gui-connection/`:
`Accord-GUI-Windows-x64-2026-10-22.zip`,8файлов, SHA256
`54852249e7b3e57a035dae7c5d57d7e8717014f86fc12aa2071e87ad16033bdb`.
Manifest roundtrip и поиск точных private origin/обоих credential значений во
всех распакованных entries — PASS. Внутри binary/source/checksums, MCP template,
CLAUDE supplement, инструкция и интерактивный PowerShell launcher. Launcher
прошёл SOURCE APPROVE160, UTF-8 BOM для PowerShell5.1; native execution NOT-RUN.
Индивидуальные access файлы коллеги находятся отдельно в private
`.local/gui-connection/colleague/`; никакой отправки третьим лицам не было.

## Фактическая функциональная QA

Отдельный tester161 после SOURCE APPROVE160 выполнил live process/Next UI
проверки: **29/29 PASS**. Три самостоятельных stdio процесса, разные серверные
сессии и два индивидуальных credentials на одном Mac. Проверены MCP12 и
profiles/coordination/team/similar resources, TLS/scoped ACL/default denial,
точный claim и отказ второму исполнителю (authoritative task:null), фоновый
heartbeat, фактический holder в REST/browser, release → ready → изменение
собственной задачи через owner browser → reclaim generation2 с уточнением →
done. Тестовые epic/task завершены, GUI-код не менялся. Evidence содержит
действительный существующий GUI commit с явной пометкой administrative smoke,
без фиктивного результата разработки/независимого review.

Actual Chrome отрисовал live/done экран «Команда» и карточку задачи —4снимка
с privacy masks. Page errors/assets/unexpected mutations отсутствуют; browser
logout прошёл. Root отдельно просмотрел live-team screenshot. Физический
Windows/Claude и actual Codex business invocation этим не подменяются.

Public пакет готов; для этого Mac подготовлены private quick-start и scoped
ярлык проекта. При старом browser login нужен новый token file владельца GUI
из `~/.config/accord-gui/gui.token`; прежний вход сам не меняется.

После QA штатный all-project backup повторён: format2 snapshot проверен,
новый GUI включён, capable guards/effective launch/owned runtime — PASS.
Reviewer160 дал FINAL SOURCE+QA REVIEW APPROVE; blocking findings0. Полный
source/evidence и ограничения в WORK160/161. Серверный release не менялся.

Статус: LOCAL CONNECTION + NATIVE/WEB QA PASS; Windows/Claude/second-PC
acceptance остаётся NOT-RUN, actual Codex session call BLOCKED CLIENT APPROVAL.
Реальные origin, identities, tokens/cookies и сырые production данные находятся
только в ignored private evidence, не в этой карточке или Git.
