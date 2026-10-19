# WORK-095 — независимый process/browser QA TASK-004

- Роль: tester, исполнитель `/root/webqa095`; автор реализации — другие агенты.
- Статус: **QA PASS; передано на независимое ревью WORK097**. Финальный browser/process набор14/14 PASS, дополнительные real netHTTP границы2/2 PASS. Исторические отказы ниже сохранены и замещены финальным прогоном.
- База: `6245829b5db7c27534be421565dab5cddddf2573`, freeze production после WORK090/091/093 APPROVE; independent UI092 APPROVE отдельно.
- Прочитаны Accord AGENTS, TASK004/web requirements/API, tester profile и общий/parallel процесс Давинчи read-only. Tandem OFF по явному указанию Accord. Модели именованных Claude-ролей здесь не закреплены; модель наследуется, автор/tester/reviewer разные агенты.
- Запись только `tests/web/**`, этот WORK и `/private/tmp/accord-web-084/qa`. Production source, Git, live сервисы и Давинчи не менялись.

## Сценарии и промежуточные результаты

Человекочитаемые повторяемые кейсы отделены от результатов: `tests/web/README.md`.
Тесты — реальные server/netHTTP/native MCP OS pipes/Chrome. HTTP mock отсутствует;
fixtures и токены только synthetic, raw credentials не входят в JSON/отчёт.

Промежуточный полный browser run `/private/tmp/accord-web-084/qa/run-095-complete/results.json`:
login file/plainPOST,8desktop/mobilepages,humancreate→native12tools/describe/next,
explicitheartbeat/read-no-renew,whitespace400nearfield→correctwithoutreload→answer,
race303/409/stale409draft,4uploads/3scopes/versions1+2/native resources+RESTSHA,
auth/Origin/CSRF/foreignSID,UI50MiBpreflight,31srealstream,restartcookie/history/tasks/files — PASS.
Далее `backup.snapshot` на остановленнойfixture отказал; remaining restore/rollback
сценарии этим прогоном NOT-RUN. Полный этап не закрыт.

Отдельный `/private/tmp/accord-web-084/qa/upload-final/upload-results.json`:
exact52428800bytes303/+1byte413 PASS; неполныйZIP terminated byserver через120007ms,
ECONNRESET, ready200 PASS. Отдельная escalated postcheck после остановки:
pending/deadline rows0, reserved0, staging0. Последняя добавлена в финальный test source
и будет повторена с ним; предыдущий Node process уже загрузил старую версию suite.

## Возврат и ограничения

System `/usr/bin/python3` Python3.9 / SQLite3.54.0: escalated реальныйsnapshot
воспроизводимо отказывает (сначала src.backup: unableopenDB; после source probe —
validate_snapshot: unexpected snapshot file). Independent097 воспроизвёл WAL/SHM
sidecars в destination pending. Базы source600/каталоги700, integrityok.
BundledPython3.12.14/SQLite3.53.1: тотже snapshot тойжеfixture PASS. Различиеruntime
не выдаётся за productionPASS; root определяет исправление portability,097re-review.

Существующие Go tests лично: TestBrowserCookieRotationRevokeAndLogout и
TestOriginConfigurationFailsClosed PASS (web0.767s). Это config/handler тесты
Secure/HttpOnly/SameSite, не browserTLS. PhysicalsecondPC, liveTLSupgrade NOT-RUN
этим tester. Developer задача только создана/захвачена, developer delivery не заявлен.

## Журнал

- 2026-10-19: started — approve090/093 проверены, подготовлен независимыйнабор.
- 2026-10-19: testfixture returned — повреждённыйbase64PNG заменён validChromium1pxPNG;
  Playwright запретbuffer>50Mb устранён filepathfixture. App не менялась.
- 2026-10-19: returned — offline backup runtime portability передан root/097;
  remaining сценарии ожидают принятойдельты и финального прогона.


## Финальная приёмка tester — PASS

После независимого097 approve окончательного backup helper выполнен один clean
полный browser/process прогон после адресного исправления тестового CLI reason.
Финальные тестовые входы не меняют продуктовые правила: maintenance reset принимает
буквально `restore`; старыйC2 API сравнивается по известным ему event identity полям,
а **полные исходные bytes** ответа/event/schema сверяются отдельно по rawSQLitehash.
`state-hash.py` читает только остановленнуюБД через immutableRO, без sidecars.

- **PASS14/14:** `/private/tmp/accord-web-084/qa/run-095-final-pass/results.json`,
  finished2026-10-19T18:38:16.231Z. Source каждого test файла совпал с captured manifest
  на момент запуска. Настоящие Go server PID39690, stdio worker PID39931, Chrome155.0.8059.82,
  Playwright1.62.1, Nodev24.19.0, Go1.27.2/darwinarm64, Python3.9/SQLite3.54.0.
- **PASS:** полный human→agent/native question→browser400correction→answer/history;
  exactclaim race303/409, stale409draft, explicitheartbeat/read-no-renew и correctedrelease400→303.
- **PASS:** реальные browseruploads3scopes/versions1+2, native resources + REST SHA,
  safeJPEGpreview отPNG; HTML/SVG/ZIP attachment/octet-stream/nosniff/sandbox;
  anonymousbrowserfile denied, nativeauth/Origin/CSRF/foreignSID rejected.
- **PASS:** UI50MiB+1 preflight не отправилPOST; ZIPstream31027ms→303, превышающий global30s.
- **PASS:** новый server process сохранил browsercookie без отзыва, tasks/history,
  metadata и точные bytes пятиassets. Stoppedfixture snapshot/manifest8files/integrity/schema2+1,
  offlinecopy, browserauth reset и native reset; oldbearer/oldcookie отвергнуты,
  freshsynthetic token login восстановил history и все5fileSHA.
- **PASS:** фактически запущенный прежнийC2 `f063380` на новойboard schema2,
  затем новыйserver. Полный raw answer+human_answered event+schema hash
  `e701faedd145ee2631a5ea3bbc8cf97d39369d2f2b864100287982ef20bc9be0`
  совпал доold, послеold и послеnew; всёweb дерево побайтово неизменно приoldrollback.
  СтарыйtypedAPI проецирует только прежниеEventData поля; потеряunknown полей в его
  JSON не засчитана за потерюdurabledata, сохранность полногоevent доказана rawhash.
- **PASS2/2 separate netHTTP:** exact52428800bytes303/+1byte413;
  неполныйZIP закрыт самимserver через120007ms, ECONNRESET, ready200.
  Тот timing получен до assertion-only дельты tests на **томжеGo binary**;
  inputs/deadline path не менялись. Полный120s не повторялся по указаниюкоординатора.
  Новые SQL/staging postconditions независимо выполнены послеtiming: pending/deadline rows0,
  reserved0, staging0. Записаны отдельно в `upload-final/postcheck.json`; не выданы
  за исполнение поздней версии всего120s test source.
- **PASS existing handler tests:** Secure/HttpOnly/SameSite/rotation/revoke/logout и
  fail-closedorigin; web0.767s. BrowserTLS остаётся вне HTTPfixture.

### Версия и контрольные суммы

Go runtime source — `6245829b5db7c27534be421565dab5cddddf2573` + dirty documentation/test state;
Go source после сборки не менялся. Прежний C2 buildinfo — `f06338089bec3b8c7f17f541c463131f5c98f96c`, vcs.modified=true. Собран tester по поручению root из основного checkout `/Users/shagi/Desktop/Project/agent-board` командой `.local/toolchains/go/bin/go build -o <private QA binary> ./cmd/agent-board`.
Инфраструктурная дельтаroot не вGo binary: одобренный helperSHA
`0dc601d2c7acab4e53180d77e8bbd60cea869195c566ac3a605f8eba9d7d9b51`.
Source manifest151files SHA `ef88a506d37470998d4e5be520c0d984cb856f2d53ea0b0da18da7c3dc3b0053`.

| Артефакт | SHA256 |
| --- | --- |
| accord-new binary |2303c50db3235a306313e0a568fdcc01a7f2cf7f46723d13dc7250b1c1a2447a|
| accord-old binary |33b4810202feb6ccbef9133d2842a863409f9b560efd05235d450120bf2f88b1|
| tests/web/browser.cjs |d2d8995cbe4c090593bb69a657884a82cbde43a1f810883db6c2bccb5a171fe1|
| tests/web/harness.cjs |13dd4286e8c6b848a87f7c04d7d7330575b70662f663c96f24bec6658e6a1c96|
| tests/web/lifecycle.py |bfcd6851d30939b7d74424c473597c766008df70e29c7d295a6e2e25ae09e363|
| tests/web/state-hash.py |9bb16317b4a91de6618bc6b720a0716dd8aac2c54b06b91dd494e2f73bd632b5|
| tests/web/upload-process.cjs |02c08138e587a87594abb896d43272c6367eb009f8230cef2c5337eedab8c99f|
| finalbrowser results.json |7bf1ea5fc7778be300e2b554bb701958b79f4d1b3c849b326d8b5bf2c8331ec2|
| earlier120s upload-results.json |22e05a4558e757e30c4b847ab03248e8b0f26cadb7108a392b6657440b19196e|
| supplemental postcheck.json |63973bd9cb3e133763063a036b289de1a058ae08c2120f0bd8ee794fb2b1a6c4|

Полный набор SHA и путиprivate evidence —
`/private/tmp/accord-web-084/qa/run-095-final-pass/evidence-manifest.json`.
Новые файлы: `tests/web/README.md`, `browser.cjs`, `harness.cjs`, `lifecycle.py`,
`state-hash.py`, `upload-process.cjs` и этотWORK. Каждыйcodefile менее300строк.
Инструкция запуска и case/expected отделены от результатов в tests/web/README.md;
внешниеtoolpaths задаются толькоenv, секреты в source/reports отсутствуют.

### Оставшиеся границы

- **NOT-RUN tester095:** production HTTPS/VPS/live apply, физический второйПК,
  Windows/Linuxnativebinary и полныйdeveloper delivery/evidence/review cycle.
- UIfunctional verdict — независимыйWORK092; exactpixelapprove N/A безмакета.
- Проверка нового test source — отдельныйreviewer097; PASS tester не заменяетегоapprove.
- Исторические FAIL reports и failure screenshots в приватномQAroot сохранены;
  original backup defect исправилroot, тестировщик приложение не менял.

## Финальный журнал

- 2026-10-19T18:34Z — helper correction independently097APPROVE; новый cleanrun.
- 2026-10-19T18:36Z — test-only CLI reason mistake исправлена, targeted real lifecycle PASS;
  oldtypedprojection assertion согласована097 без ослабленияrawDBchecks.
- 2026-10-19T18:38:16Z — completed tester: full14/14 PASS; actualold/new rollback,
  restoredfreshlogin/history/files завершены. Test source SHA сверены послепрогона.
- 2026-10-19T18:39Z — handed_off reviewer097: заморожены6testfiles+WORK,
  private evidence manifest передан; root принимает общийэтап и выпускаетотдельно.


## Независимая сверка происхождения старого C2

Reviewer097 отдельно выполнил read-only сверку исходников основного checkout:
`git diff --name-only f063380 -- cmd internal go.mod go.sum` пуст;
также diff `1e902..f063380` для тех же областей пуст. Таким образом, исходники
фактически запущенного old QA binary соответствуют согласованному pre-web C2,
несмотря на dirty состояние checkout из-за документации. Это подтверждение
эквивалентности исходников; macOS QA binary не назван точной копией production
Linux binary. Проверку Git выполнил097, а не tester095.
