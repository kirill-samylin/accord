# WORK-152 — независимая QA координации

20.10.2026. TASK011. Tester152 не является автором product source.
Владение: `tests/process/task011/**`, `tests/process/task011-update/**`,
`tests/browser/task011/**` и этот WORK. Каноническую TASK ведёт root.

## Текущий статус

LOCAL QA PASS24/24, включая обязательный keyboard slice; независимый review150 новых тестов впереди/в работе; новые тесты проходят независимый review150. Production delivery/HTTPS пока NOT-RUN. Product binding:
74 files manifest SHA256 `5cd77400d904fe72783b3beabda63b1fa3a4137ddfc914e651a3347548cc672a`.
Root make check PASS. Candidate clean source `f2edac94238c2b8fc267c37d9048187902bdf849`,
macOS binary `.local/task011/candidate/agent-board`, SHA256
`daf342c2bb7d714af7d4aceea6910de31eb9e6ddc0751b926ac667284e0bf8ac`.
Actual process/Chrome написаны после этого APPROVE и выполнены; итоговые evidence ниже.
Next completed build находится в `web/.local/next`, upstream3101; root сохранял
сборку до завершения local QA; теперь primary build освобождён для release7892. Ранее предположенный `.next` не является distDir.

## Исторический план до APPROVE

READ-ONLY PLAN. Прочитаны coordination/TASK011 и прежние TASK010 process,
SQLite inspection/snapshot/restore, browser forms/isolation и updater harness.
Общий source APPROVE150 ещё не передан. Новые process/E2E не написаны и не
запускались. Этот план не является тестовым PASS.

## Provenance и ограничения

Использовать только новую private synthetic fixture в ignored `.local/`;
никаких production записей/credentials. Harness владеет child processes и
отказывается использовать занятые порты. Данные/токены private0600/каталоги0700;
в отчёте только hashes, counts, synthetic labels. До/после запусков сравнить
hashes тестов и product/operator source, сохранить SHA executable и provenance.

Baseline macOS: clean schema3 revision343b93c71297c58c972f73fed4d1d4ca3a349740,
`.local/task011/legacy/agent-board`, SHA256
`4e11d2569120dada3df15061bcfc5193aacb7931a048110f27242f0b9419e085`.
Linux baseline production cbf202086431335d4657c722a0f75328a09dc4ea,
SHA256 `66f949aa6022ade9cf4881c1a8cd65201b35d94c6635db942fcd6f73d42546aa`.
Baseline must be verified from actual bytes and trusted proof, not inferred
from current branch. New binary/source revision/status supplied by root.

Planned env: `ACCORD_TASK011_BINARY`, `OLD_BINARY`, `OLD_SHA256`, `OLD_REVISION`,
`REVISION`, `SOURCE_DIRTY`, `BROWSER` under the same prefix. Outputs:
`.local/task011-qa/run-*`; updater `ACCORD_UPDATE_*`, `.local/task011-update-qa`.
Node24.19.0/Go1.27.2 fixed; existing Playwright/Chrome, no dependency updates.

One host is not a physical second PC. Loopback process/browser does not prove
production HTTPS or native Claude/Codex. Updater harness has real Go/SQLite/
crash-resume; systemd/Next/UID adapters are explicit simulations. Production
owned runtime/stage/apply/backup and HTTPS are separate root-run evidence.

## Planned process coverage after source APPROVE

1. Actual old schema3 seed default+two projects, registry/default/all7DB,
   credentials/memberships/repositories; profiles/revisions/bound+unbound sessions,
   live lease/attempt, native replay, existing browser cookie, note/history/blob.
   Stop old process and compare independent SQL-row hashes through offline3→4
   migration/repeat. Snapshot3 retained; no old rows silently modified.
2. Restart new actual binary: cookie/session/profile snapshot/lease generation/
   heartbeat/download preserved. Missing coordination metadata fails closed
   before blob cleanup; schema3 binary/floor denied after4.
3. Profile assignment exact ID/version/role, same-name isolation, native/MCP
   next eligibility and session queue counts; unassigned role pool remains usable.
   Two actual concurrent matching sessions yield one lease. Wrong/unbound profile
   does not claim assigned work; task key targeted next cannot bypass routing.
4. Assignment expected version409, explicit clear, foreign-project IDs, partial
   pair/unknown/null fields, role/human/epic mismatch; replay preserves exact
   response/audit and conflicting payload rejected. Edit/archive pauses new
   claims, refresh explicit; existing live holder heartbeat/completion/snapshot
   unchanged. Reassign cannot steal live work or reset generation/history.
5. Human/coordinator dependency creation/removal, cycle, positive gate and
   requester/live-consumer generation. Detail reasons agree with actual next.
   Human cannot complete/release another agent's lease.
6. Developer exact SHA→explicit reviewer→approve→atomic QA with approved link→
   passed→epic close. Negative/pending/not_run, sameSHA target reopen/newgeneration,
   newer verification, cross-epic obligations and unlink cannot bypass gate.
   Stale target version/concurrent change rejects check_create atomically.
7. Viewer/archive/ACL epochs and action/session/project scoping, logout and
   own-native MCP12/resources. Actual old/new full format2 restore preserves
   history/profiles/revisions/assignments/audit and performs documented auth reset.

## Planned Chrome coverage

- Actual create-assigned and detail assignment forms, profile/version selection,
  reasons/dependency/target links, explicit review→QA buttons and transition UI.
- Concurrent409 keeps user draft and expected version until explicit refresh;
  actual server commit followed by lost response retries identical action/body
  without duplicate mutations/audit. Bound operation retains source project.
- Viewer has reads with no mutation controls; project A→B while A response is
  delayed, independent tabs, cache/draft/action isolation. Keyboard/focus.
- Desktop1440/mobile390 × light/dark screenshots of assigned detail, blocker,
  review/QA/current-stale state; horizontal overflow/page/CSP/asset failures and
  private prefetch checked. Screenshots visually inspected before report.

## Planned updater coverage

Seed actual schema3 default+two projects/profiles/history and private snapshot;
separate relocation adapters for host-only boundaries. Actual3→4 apply and final
backup, startup failure retained hold+resume, abrupt exit after Go pointer,
manual-marker/helper drift/known temp/unknown temp failclosed preservation.
Independent SQL-row oracle verifies7DB old rows, new coordination metadata,
profile/session/history, new snapshot4 with original snapshot3 retained; exact
old capability denied. Full restore4 is separately exercised by process suite.

After source APPROVE: write tests → review new tests → execute/report fixes;
any product finding returns to author/reviewer. Tests never weaken product gates.
All required items remain NOT-RUN until actual evidence exists.

## История запусков

Первый process-only run `.local/task011-qa/run-1792527958017`: FAIL после3PASS.
Причина — ошибка нового теста: конкурентный прямой HTTP был направлен на
`/api/v1/p/{id}/next`, а существующий контракт — `/tasks/next`. Product не менялся;
исправлен только test URL, старый report сохранён. Actual updater первый run
`.local/task011-update-qa/run-1792527959164440000`: PASS8/8, включая реальные
SQLite3→4/crash/resume; host-only boundaries остаются declared simulations.

Повтор process-only `.local/task011-qa/run-1792527983414/results.json`: PASS13/13,
`source_unchanged:true`, actual candidate SHA/provenance зафиксированы. Проверены
реальные3→4/7DB/profile rows+revisions, full restore3+4, bound/unbound sessions,
живой lease/cookie/blob; routing/reassign/replay/strict REST/MCP12; review→QA/
negative/latest/cross-epic/sameSHA newgeneration; human/architect handoff;
browserJSON/actions/ACL epochs. Product findings не выявлены этим запуском.

Все новые тестовые scopes переданы независимому reviewer150. Full process+Chrome
запущен после этого PASS, product source и Next build зафиксированы; итог впереди.

Первый fullChrome run `.local/task011-qa/run-1792528021144`: FAIL после native11
и Chrome3PASS. Ошибка test locator `getByLabel('Действие', exact:true)` для
вложенного label/select; actual screenshot показал открытую штатную форму.
Исправлен только тест на select с известными вариантами. Screenshot/FAIL сохранены;
никакого product change или подавления ошибки сервера. Требуется полный повтор.

FullChrome repeat2 `.local/task011-qa/run-1792528115021`: FAIL после native11 и
Chrome4PASS (добавление/удаление dependencies и cancel/restore также прошли).
Ошибка теста: после закрытия create-check dialog немедленно прочитан старый URL
до окончания asynchronous Next navigation. Добавлено ожидание фактического
перехода на новый task URL перед native Get. Product не менялся, FAIL сохранён.

Chrome repeat3 `.local/task011-qa/run-1792528150784`: FAIL после native11 и
Chrome6PASS, включая explicit browser review→nativeapprove→browserQA→nativepassed→
epicclose и atomic create-assigned. Late-response/two-tabs сценарий дошёл до
project switch menu, но catalog загрузка показала error. Добавлена фиксация
HTTP status и safe diagnostic labels для определения причины; пока это
не отнесено к product или тесту. Native/source не менялись.

Reviewer150 замечания к тестам: actual MCP create с assignment вместо Native
setup при заявленном MCPcreate coverage и проверка auth reset всех board token/
browser rows после restore4. Оба усиления внесены до следующего полного запуска.
Ограничение updater: simulated prior helper set уже содержит current guard bytes;
его PASS не доказывает реальную смену guard3→4, нужен отдельный actual deployment.

## Первый итог local QA до keyboard extension

Два последовательных full actual process+Chrome PASS23/23:
`.local/task011-qa/run-1792528244009/results.json` и финальный
`.local/task011-qa/run-1792528339624/results.json`. Финальный report SHA256
`8b1bc9bdbc96418ffc9e3ac73c5d06e3227730fdea7ddad7b8a723948d828dbb`.
`source_unchanged:true`, candidate sourcef2edac9, binarySHA daf342c2…bf8ac,
Next BuildID `p8Oy9g77thoUV1qX6mfsx`, actual upstream3101, Chrome155.0.8059.86.
В финальном browser run284responses, page/asset failures0, privateprefetch0,
ownlogout PASS. Локальные forms проверены через реальные browser controls; отдельный keyboard-only
проход и полная accessibility audit — NOT-RUN. Keyboard command palette входит
в подготовленный production readonly script, его run впереди.

Финальные16screens (assignment/blocker/checks/epic ×1440/390 ×light/dark)
визуально проверены через4contact sheets. Горизонтального document overflow нет;
назначение/статусы/причины/проверки читаются; mobile390 — viewport одного Chrome,
не физический телефон. Screenshot hashes отдельно `screenshots-manifest.json`
в том же private run. Полный перечень проверок — `checks` финального report.

Actual updater report `.local/task011-update-qa/run-1792527959164440000/results.json`
PASS8/8, SHA256 `9c64704dc7f7606deb17c005d8c044b9514225450c1e78301d88d9e5f21fb61e`.
Тестовые/operator source неизменны во время прогона. Реальны Go/SQLite3→4,
all7DB independent old-row preservation, profile revisions/bound snapshots/
lease-history, crash/resume и full snapshots3+4. Systemd/Next/UID и prior-helper
installation — simulations; реальная смена root helper/guard3→4 остаётся root
stage/apply барьером. Полный restore4 с authreset проверен process-suite отдельно.

Final source18files manifest `.local/task011/qa-tests-manifest.json`, SHA256
`c46a1abb24aa6302c3a89da32e192907bb62dca763eb2bdfeb353284b92e34df`.
Все новые files переданы reviewer150; product/infra source tester не менял.
Ранние FAIL оставлены выше. Причина одиночного catalog-load error run8150784
не установлена; после добавления точной фиксации catalog HTTP/body два полных
последовательных запуска прошли. Не объявляется исправлением product.

## Production readonly preparation

`tests/browser/task011/production-readonly.cjs` подготовлен из approved010.
SHA256 `5cb2b9b29a8ca6ac2bae87b6a45ff8b71497f7208dfb2a25de5c5206b0828395`.
Те же private env `ACCORD_QA_*`, root запускает после reviewed safe apply.
Разрешены только собственные login/projectopen/logout, native+browser/MCP reads.
Никаких task/profile/lease/ACL writes, synthetic records в production не создаёт.
Проверяет existing tasks coordination/native resource/tab, before/after tasks
hash, CSP/assets/prefetch/cookie/logout и desktop/mobile/themes. При пустом
existing native task collection detail получает явный N/A, данные не создаются
ради PASS. Script hashes+unchanged фиксируются, tokens/data не печатаются.
До отдельного review/run production HTTPS — NOT-RUN. Физический второй ПК,
реальные Claude/Codex, workers/providers, полный V1 — NOT-RUN.

## Дополнительный keyboard slice перед final freeze

По прямому root поручению добавлен отдельный `browser/task011/keyboard.cjs`:
focus trigger→Enter, autofocus inside assign dialog, полный круг Tab/ShiftTab,
реальное ArrowDown/ArrowUp profile select с восстановлением, Escape close и
возврат фокуса к исходному trigger. Проверяет отсутствие business POST и прежние
Task.Version/assignment. Product/зависимости не меняются. Новый full run требуется
для актуального frozen test manifest; прежние23PASS остаются историей, не новым
keyboard PASS. Полная axe/accessibility audit по-прежнему NOT-RUN.

Первый keyboard fullrun `.local/task011-qa/run-1792528526759`: FAIL после
native11PASS. Enter opening/autofocus/Tab/ShiftTab прошли, но test предполагал,
что ArrowDown закрытого native select на macOS сразу меняет выбор. Он открывает
popup на текущем элементе, и Enter оставляет очередь роли. Исправлен только
keyboard sequence: Space открывает popup, затем ArrowDown/Up и Enter выбирают.
Product не менялся; FAIL и actual screenshot сохранены. Полный повтор обязателен.

Keyboard repeat `.local/task011-qa/run-1792528566983`: FAIL после native11 на том
же дополнительном native picker Arrow subcase. Reviewer150 независимо повторил
оба sequence на минимальном actual Chrome select **без React и Accord** и получил
ту же пустую selection. Это ограничение native headless picker automation, не
подтверждённый product defect. Root явно принял узкий обязательный scope без OS
popup: Enter open/autofocus/Tab+ShiftTab trap/Escape exact focus return/POST0 и
unchanged task. Только дополнительный native picker keyboardchange — N/A;
функциональная selection проверена actual forms через native browser selectOption.
Тест удаляет только этот избыточный subcase; product не меняется. Full repeat
запущен заново, прошлыеFAIL сохранены. Полный a11y audit не выполнялся.

## Финальная local QA после keyboard extension

Full actual process+Chrome `.local/task011-qa/run-1792528643872/results.json`:
**PASS24/24**, `source_unchanged:true`. Report SHA256
`c29a27320c1c6b49c066aa3a4682cd26b3df4bda8a07110fbe5bbd0b3e46387a`.
Product candidate f2edac94238c2b8fc267c37d9048187902bdf849 и binary daf342…bf8ac
не менялись. Actual Next BuildID `p8Oy9g77thoUV1qX6mfsx`, upstream3101.
Chrome155.0.8059.86;284responses; page/asset errors0, prefetch0, ownlogoutPASS.
Сохранены и визуально проверены финальные16screens; hashes рядом в
`screenshots-manifest.json`, contact sheets являются вспомогательными копиями.

Обязательный keyboard PASS: focused trigger→Enter открывает именно новый
assign dialog; autofocus внутри; полный цикл Tab и ShiftTab удерживает focus;
Escape закрывает и возвращает **тому же** trigger. BusinessPOST0,
Task.Version/assignment unchanged. Native OS picker arrow selection — N/A по
независимому minimalChrome воспроизведению и явному root disposition выше;
functional profile selection фактически прошла в forms. Полная a11y auditNOT-RUN.

Финальный19file test manifest `.local/task011/qa-tests-manifest.json` SHA256
`983d757a7668080929652e167f985a9a874b4e15da7fefed25a7ff033ee4bdae`.
Production readonly script SHA5cb2b9…28395 не менялся. Updater8/8 предыдущий
report и его источники/tests/operator hash не менялись; лишний повтор не нужен.
Все изменения этой добавки только browser011/README/WORK152; source product и
infra не менялись. Primary Next build освобождён root для immutable release.
Независимый reviewer150 получает final19files+report; actual Linux exact release,
production stage/apply/HTTPS/backup ещё остаются отдельными root барьерами.
