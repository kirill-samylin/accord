# WORK-155 — независимый process/browser QA TASK012

22.10.2026. Исполнитель: отдельный tester155. Область записи:
`tests/process/task012/`, `tests/browser/task012/`, этот WORK.
Каноническая TASK и product source не изменяются. Новые process/E2E созданы
только после независимого SOURCE APPROVE153 для `69256c419a7a5529e7890633304e54c819b358c0`.

## Контур

- Реальные Go process/SQLite, отдельные синтетические owner/editor/viewer,
  два проекта, CLI/REST и stdio MCP12. Проверки warn/replay, strict concurrent
  create, force/reason, duplicate/restore, чужая аренда, epic related dismiss,
  ACL/project boundaries, недоступные pinned revisions, off и nonrenewal.
- Реальный Next production build и Chrome: «Команда», обзор, список/карточка,
  duplicate/different actions, strict create и recovery. Desktop1440/mobile390,
  light/dark; ошибки загрузки/React и горизонтальное переполнение.
- Production readonly script — продолжение TASK011: team/native/web/MCP,
  similar по существующей задаче, пустой create dialog без submit. Browser
  mutations разрешены только own login/logout/explicit project switch.
  Task digest и session expiry сравниваются до/после; own auth cleanup и
  закрытие browser обязательны. SOURCE APPROVE153 script SHA256
  `400ce6a9ce3c227e4089505684356eae429965e26c2d4c41c3be33ce5d8a41ff`;
  shared support SHA256
  `f3c2313e4b169e266f5bb3945ebc82049a892fd3bf80cd16a4c312eda07265cf`.
- Updater4→4 — отдельный tester157; этот WORK не приписывает его результаты.

Каждый запуск пишет новый private0700 `.local/task012-qa/run-*`; токены0600,
значения токенов и реальные пользовательские данные в Git не попадают.
Harness сохраняет binary/source/build provenance и source inventory; все
owned Go/Next/MCP/browser закрываются в finally. Занятые3100/3101 отклоняются
до HTTP, чужие процессы не завершаются. Source не меняется ради теста.

## История фактических запусков

Начальный clean candidate: `69256c419a7a5529e7890633304e54c819b358c0`, macOS
SHA256 `889bf7705c31b2ba732b0038d52dbf363de1c8734685a671d3f9e43b82d8ab76`.

1. `run-1792693168799` — **FAIL test-only**: `profile_archive` fixture
   передал reason вместо обязательного `state:"archived"`. Исправлен запрос,
   product source не изменялся. Во время ранней подготовки дописывался readonly
   test source; это не является финальным unchanged-source PASS.
2. `run-1792693202518`, `run-1792693217122`, `run-1792693223830` —
   **FAIL product**: stdio MCP после describe/tools12 возвращает
   `-32602 Resource not found` для `accord://team`. Независимо найдено, что
   `internal/bridge/resources.go` mirror allowlist не включает team/similar.
   Root/reviewer уведомлены; author156 исправляет и проходит повторное review.
   Последовательные run уточняли диагностику и собственную MCP session fixture;
   исходный product FAIL не скрывается новым PASS.
3. `run-1792693257539` — **частичные PASS**, затем **FAIL test-only**:
   карточка корректно показывает обе заметки (duplicate и forced override),
   locator `.duplicate-banner` был неоднозначен. Уточнён по тексту
   «Эта задача отмечена дублем»; ни одна заметка не удалялась.

Последний запуск фактически подтвердил Go capabilities; warn+idempotency;
live warnings+foreign lease сохранение; duplicate/restore; epic dismiss;
viewer/project ACL; edited/archived pinned assignment exclusion; strict
concurrent single winner409. Реальный Chrome подтвердил: concurrent late409
без совпадения в preview → consent/force; UTF8 reason400 сохраняет видимое
поле/значение; failed preview → authoritative409 с сохранением draft;
actual commit/lost response → frozen fields+byte-identical retry+single create;
strict preview блокирует создание и диалог видим при390.

4. `run-1792693626921` — **PASS22/22**, первый полный frozen run после fix156:
   `4742e71c96d41aed17243e04ab3e5171f9d5da1f`, binary SHA256
   `f17b4bb8dfa06dbdd9168687beb79c2fa4ce9b33579f6e58cca86932738bbfb5`,
   report SHA256 `4814637e01a61b47cd6a9e0fe32a1f3b67d14d3fa283e5923eb7a98d0bf75996`.
   `source_unchanged:true`,17screens, Chrome155.0.8059.86,482responses.
   После него узко усилен process epic restore: actual unlink related,
   два цикла duplicate→open, удаление duplicate relation, сохранение другой
   related и возврат пары. Это отдельный test-source delta, product не менялся.
5. `run-1792693746818` — **окончательный PASS22/22**, тот же clean candidate,
   actual Next BuildID `7cLF6QiHKLNWtrKA9ckFg`, готовый build upstream3101,
   Node24.19.0. Источник tests прошёл повторный SOURCE APPROVE153,8file manifest
   SHA256 `788b6d59bd393641496f4674eb3c6955475a98113cee710f402687e0868b03b3`.
   Report SHA256 `eac721307dc8afc0f5ca875668cbb55b47b4ab1c8d48cce8bcb8055aae238b82`.
   `source_unchanged:true`;10actual process checks и12actual Chrome checks.
   Новый MCP stdio реально вернул team и similar с точными task keys для
   Unicode+spaces и literal`C++` (не только наличие resource content).
   Снимки не содержат реальных данных:17synthetic screenshots;4представителя
   первого полного run просмотрены отдельно (team mobile dark, overview desktop,
   duplicate detail mobile, strict dialog mobile), layout без overflow.

Проверено обновление открытой «Команды» после native claim/release на настоящем
10s polling без navigation; browser offline останавливает GET, online
возобновляет; B scope не показывает A people/work; чтения не продлевают browser
session. Visibility input моделировался DOM getter/event в headless Chrome;
timers/hook/GET фактические, OS tab lifecycle **NOT-RUN**. Это ограничение
сохранено непосредственно в machine-readable check.

Production readonly **NOT-RUN** в этом WORK на данном checkpoint; его root
оператор запускает отдельно после применения. Linux exact release process и
updater принадлежат root/tester157, здесь не приписываются. Два физических ПК
**NOT-RUN**, реальные Claude/Codex execution workers **NOT-RUN**, полный V1
**NOT-RUN**. Mobile — Chrome viewport. Все owned процессы финального run закрыты.

## Запуск

Product: Node24 `tests/process/task012/run.cjs` с
`ACCORD_TASK012_BINARY`, `ACCORD_TASK012_REVISION`; Linux добавляет
`ACCORD_TASK012_BROWSER=0`. Browser использует готовый actual Next build в
`web/.local/next` upstream3101, установленный Chrome/Playwright без установки
зависимостей. Нельзя конкурентно пересобирать Next во время browser QA.

Production readonly: Node24 `tests/browser/task012/production-readonly.cjs`,
только после отдельного approval точного script digest; существующие private
client/token files передаются через `ACCORD_QA_CLIENT_FILE/TOKEN_FILE`, новый
private `ACCORD_QA_DIR`, exact `ACCORD_QA_REVISION`, Chrome/Playwright paths.
Production бизнес-мутации не тестируются.

Файлы не превышают300 строк. Production readonly harness сохраняет ранее
reviewed TASK011 safety/auth/cleanup контур и добавляет TASK012 reads;
последующие изменения скрипта требуют повторного review.
