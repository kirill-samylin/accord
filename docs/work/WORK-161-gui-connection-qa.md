# WORK-161 — независимая QA подключения GUI к Accord

22.10.2026. Отдельный tester после source approve WORK-160.
Область записи: эта карточка и игнорируемая private `.local/gui-connection/qa/`
(каталог 0700). Исходники продукта/GUI, credentials, конфигурацию и инструкции
tester не меняет. Новые process/E2E tests проходят отдельное source review
до live исполнения. Тестовые данные явно помечены
`TEST GUI Accord connection 2026-10-22`.

## План и границы

Только новый GUI project, own epic/task/note; production settings и чужие
задачи не меняются. Три самостоятельных stdio bridge процесса: coordinator с
Mac owner credential, developer и competing developer с отдельным credential
коллеги/editor. Один физический macOS host; это не Windows/Claude acceptance
и не два физических ПК. MCP12, profiles/coordination/team/similar resources,
atomic claim, auto-heartbeat, scoped REST, browser JSON team/task, ACL/default
denial, cooperative release/correction/reclaim и completion evidence.

Release/reclaim — добровольное управление арендой. OS stop/resume, запуск
модели и отправка новых инструкций в работающий агент продуктом не реализованы.
Серверный task update становится данными следующего get/claim, сам агент
должен прочитать изменения; это не prompt injection в процесс.

Реальный Codex CLI: root сообщил discovery PASS (12 tools/8 templates,
Tandem отсутствует). Describe не выполнен: локальная approval policy CLI
`never` отклонила требующий approval MCP call. Это не server FAIL; actual
agent business path **NOT-RUN / BLOCKED BY CLIENT APPROVAL**. Политику tester
не обходит. Самостоятельный native bridge smoke не считается actual Claude.

## Review и проверенная версия

Независимый reviewer WORK-160 дал **SOURCE APPROVE** перед live исполнением:
private `qa/live_smoke.py` SHA256 `6863610b…`, `qa/ui_smoke.cjs`
SHA256 `f953bd0f…`. Закрыты замечания о строгом результате competing claim
(успешный ответ с явным `task:null`, а не любой отказ) и о masking имён,
machine labels и session links на screenshots. Syntax checks обоих sources
**PASS**. После исполнения source hashes не изменились.

Production service: существующий TASK012 release
`64ed870b749289b545d29cd2ec7813c6266de54d` (issuance/root evidence), без
обновления сервиса. Mac client — установленный capable binary, эквивалентный
TASK012 product source согласно WORK-160. GUI checkout HEAD при smoke:
`d2d16e95e2659f94dde41fa7bd3f3d576d6a505b`. SHA использован только как
идентификатор существующей локальной версии; тест не менял код GUI и не
объявляет его реализацию прошедшей review/QA.

## Фактические результаты

**PASS — 29 проверок** реального HTTPS service через три stdio bridge процесса
и отдельные owner/editor browser sessions. Созданы только собственные явно
помеченные E-1/T-2/N-1 в GUI project; задача и эпик завершены, живых task leases
после сценария нет. Note сохраняет ограниченный administrative smoke result.
Все bridge процессы закрыты; browser logout **PASS**.

- MCP initialize/12 tools; три различные server-generated sessions, два
  credentials (owner и editor). Profiles, coordination, team, similar resources
  прочитаны; similar нашёл собственную тестовую задачу.
- Editor developer получил T-2. Конкурирующая отдельная developer session
  получила успешный `next` с явным `task:null` и без lease. Automatic heartbeat
  действительно увеличил expiry живой аренды.
- MCP team, scoped REST team, owner/editor browser JSON team показали текущего
  holder. Оба credentials видели только GUI в каталоге; default alias был
  недоступен. Editor не получил member-management action; owner получил.
- При живой чужой аренде owner/editor browser не предлагал update. Developer
  добровольно сделал release; задача стала ready. Owner-requester через
  настоящий browser update исправил description незахваченной задачи с fresh
  CSRF/action и сохранённым private pending payload. Следующий claim получил
  generation 2 и увидел исправление. Note был виден coordinator. Done evidence
  доступно после completion.
- **Rendered Next UI PASS**: desktop Chrome, scoped «Команда» и detail T-2
  во время живого claim и после done. В первом состоянии был один busy task
  link; после done — ни одного. В завершённой task detail видна исправленная
  description. Page errors / failed assets / unexpected browser business
  mutations — 0. Четыре masked screenshots визуально просмотрены tester.

Private evidence: `.local/gui-connection/qa/live-proof.json`,
`ui-live-proof.json`, `ui-done-proof.json`, screenshots `live-team.png`,
`live-task.png`, `done-team.png`, `done-task.png`. Screenshots снимают только
main content, без адресной строки; имена/machines/person chips/session links
замаскированы. Реальный origin, credentials, cookies и raw user data в эту
карточку не перенесены.

## Ограничения и итог

Native connection + server coordination + browser наблюдение/коррекция
собственной незахваченной задачи — **PASS**. Task release/reclaim проверен
как добровольный контур клиента. Принудительная остановка/продолжение OS
процесса, запуск модели и доставка нового prompt агенту — **N/A: текущий
продукт этого не реализует**, проверкой не подменялись.

Actual Codex CLI business path после discovery — **NOT-RUN / BLOCKED BY CLIENT
APPROVAL** (см. выше). Actual Windows + Claude, второй физический ПК и реальная
совместная разработка — **NOT-RUN**. Общий V1 PASS не заявляется.
