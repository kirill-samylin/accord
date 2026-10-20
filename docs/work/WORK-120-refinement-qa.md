# WORK-120 — независимая browser/visual QA TASK-006

- Роль: отдельный тестировщик `/root/browser_qa`, модель наследована от автора.
- Дата: 20.10.2026. Область записи: `tests/web/refinement*.cjs`, эта WORK;
  приватный runtime только `.local/task006-qa/`.
- Новые tests написаны только после независимого119 APPROVE и явного GO root.
- Статус: **PASS функциональная QA и независимый visual APPROVE** на финальной
  версии7ca7c9e; независимое review самих tests оформляет WORK121.

## Версия и воспроизведение

Основной повтор до anchor дельты: `08742039daf00c5f420a770686a6673670567fa1`,
Go1.27.2 darwin/arm64. Последняя принятая версия —
`7ca7c9ec613856db104a69a602f3990a88c9f1c7` (см. финальную дельту ниже).
Binary `bin/agent-board` SHA256
`ce1893254cbea30caccadbb8cfa680c684158ca2a9723628863fc0b0a4b0d77f`.
Source manifest `.local/task006-freeze/source-final.json` SHA256
`02c2512e73a0de00802f61e9cb776c26cf20bbcc7f7533300a2e57b4cbf27af8`;
самостоятельно сверены 204/204 хешей, mismatches=[]; reported vcs.modified=true
не скрывается. Новые4test файла заморожены в
`.local/task006-qa/test-freeze.json`, SHA256
`30468fcae591722010a42fef9c20ea779208b9fb3040755b502af74228f4ac66`.

Реальные Go server/native stdio MCP/Chrome155.0.8059.82/Playwright, Node24.19.0;
synthetic токены/данные, literal loopback. HTTP/domain/store не подменяются.
Clock ускоряет browser timers, Go clock/state настоящие. SIGSTOP/SIGCONT
применён только к собственному synthetic server. Cleanup собственных процессов
успешен, static_failures=[]; runtime/private токены не входят в Git.

Запуск: экспортировать `ACCORD_QA_BINARY`, уникальный private `ACCORD_QA_DIR`,
`ACCORD_QA_SOURCE_MANIFEST`, `ACCORD_QA_PLAYWRIGHT`, `ACCORD_QA_CHROME` согласно
существующему `tests/web/README.md`; выполнить `node tests/web/refinement.cjs`.
Четыре новых файла используют существующие harness/fixture, каждый <100 строк.

## Фактические проверки

| Прогон | Версия | Результат и область |
| --- | --- | --- |
| `refinement-final-detail/refinement-results.json` |0874203|**PASS7/7**, final frozen test source,39screenshots |
| `control-final/control-results.json` |0874203|**PASS5/5**, affected `layout-poll-only` |
| `browser-core-final/results.json` |0874203|**PASS4/4**, только `browser-core` |
| `control-01/control-results.json` |a2a48ad|**PASS10/10**, полный control, исторический source до CSS preview fix |
| `refinement-01/refinement-results.json` |a2a48ad|**FAIL** ошибочного ожидания нового теста; не принимается за fullPASS |
| `refinement-02/refinement-results.json` |a2a48ad|**PASS7/7**, исторический source; последующий визуальный root finding оставался |

Initial FAIL требовал release reason в Activity. Domain сохраняет его в Attempt,
обычный released Event не имеет Reason; SSR ведёт себя так же. Исправлен только
тест: XSS literal проверяется на настоящем claimed title, activity сравнивается
с реальным списком event cursors из dashboard DTO. Product contract не менялся ради PASS.

Новые7checks: command dialog keyboard/input/стрелки/Escape/focus return,
Cyrillic server search и literal HTML, +create переход, mobile320/390 и backdrop,
JS-off390 GET search; agent/human/all + roles сохраняются в реальных формах;
total открывает done/cancelled; timeline10 последних в точном DESC и полный
журнал по task_key. Dashboard по умолчанию отправляет overview, opt-in заменяет
его dashboard в том же цикле (max_inflight=1), native release/claim меняет
ownership/counters/events; keyboard focus не теряется. Offline и настоящий
server timeout сохраняют последний good DOM и дату, recovery работает;
native lease expires_at/generation неизменны. Настоящий revoke401 терминален,
следующих polls0. Отдельные unit schema-invalid проверки — работа авторов,
новый E2E не подменяет ответ malformed fixture.

Существующие affected checks:12routes1440/390, drawer focus trap/Escape, theme
persistence/Inter; offline/reconnect/inflight/timeout/retention/no-renew/draft,
настоящие headed hidden45s0requests/visible resume200, token revoke401 и GoTTL409,
protected explicitcontinue. Browser-core: real file login сJS/безJS, UIcreate,
native12tools/claim, XSS, human claim/read-no-renew/explicit heartbeat,
whitespace400 draft/fresh action/correct answer/history. Старый binary в этом
scope не запускается: required metadata указывает текущий binary; compatibility
rollback/full14 journeys этим прогоном не заявлены.

Приватная сводка `.local/task006-qa/acceptance-summary.json` содержит точные
SHA каждого принятого JSON и версии. SHA сводки до следующей CSS дельты:
`8b4db8b550789d873a6f34e457af03172ffe744ec09bb328fe73d77ae7365916`.

## Независимый visual осмотр

Capture matrix39: overview/agents/agent detail/task/create form ×1440/768/390 ×
light/dark;6дополнительных viewport captures form, commands320/390, empty390.
Inter fonts ready, DPR1/scale1, document scrollWidth<=viewport во всей matrix.
На overview20events сохранены в DOM; отдельный labelled keyboard region
имеет clientHeight<=520desktop/<=min440,65svh mobile и scrollHeight>clientHeight.
Реальный PageDown прокручивает region; ссылка полной активности открывает
полный журнал. Root finding гигантского blank agents panel подтверждён по
старому screenshot; в0874203 панели выровнены по началу и overview bounded.

Pixels независимо осмотрены через view_image: overview1440light/768dark/390light;
agents1440light/768dark/390light; task1440dark/768light/390dark;
form viewport768light/390dark; command320/390dark; empty390dark; отдельно
agent detail1440light/768dark/390light. Длинный claimed title обрезается в roster
с полным title attribute, в detail переносится; sidebar240 на768 сохраняет
ширину, mobile drawer/команды помещаются. Табличные/пустые/form состояния
читаемы в обеих темах. Long full journal agent detail остаётся обычной страницей
с pagination; bounded preview относится к dashboard. FullPage capture формы
после scroll-to-field показывает sticky header посередине изображения — для
оценки использованы дополнительные viewport captures.

Сравнение — перенос current-default Paperclip visual language по WORK102/110/114
и pinned reference5739a53; pixel-perfect соответствие и полный набор Paperclip
не заявляются. Первый visual verdict ожидал final anchor CSS delta, найденную независимым
reviewer121: `#create` заголовок скрывался sticky topbar. После исправления
ниже — **APPROVE** в пределах TASK006; не blanket оценка всех возможных данных.

## Финальная anchor дельта и закрытие

После явного независимого121 source APPROVE повторён весь **замороженный**
`refinement.cjs`: `.local/task006-qa/refinement-anchor/refinement-results.json`
**PASS7/7**,39captures. Test-freeze30468 остаётся точным; новых code edits нет.
Последний binary SHA256
`45605611fffeef33b971616c44ba862812da6a68ad18b96004c757ba81d0f4df`;
manifest `.local/task006-freeze/source-anchor.json` SHA256
`f99cde8360ba72280d0359c46e5a3d2c3421ca169423aae932d135e96a606088`.
Снова самостоятельно сверены204/204source hashes, mismatches=[].

Независимый ручной browser audit до заполнения формы:
`.local/task006-qa/anchor-final-fixed/anchor-results.json`, **PASS4/4**.
Через настоящую команду «Создать задачу» (безJS — sidebar link) открыта
`/web/tasks#create`; header/heading измерены реальным Chrome DOM:

| Viewport | JS | Heading top | Sticky header bottom |
| --- | --- | --- | --- |
|1440×900|да|147.19px|56px|
|768×900|да|145.19px|56px|
|390×900|да|145.09px|56px|
|390×900|нет|145.09px|81px|

Все4viewport PNG независимо осмотрены через view_image: заголовок/описание/поля
доступны, верх формы не закрыт. Последний overview1440light также осмотрен
повторно; bounded panels сохранены. При первом runtime audit selector выбрал
sidebar link под открытым modal: modal правильно перехватил pointer, utility
получила timeout. Этот **FAIL стенда** сохранён в `anchor-final/`; исправлен
только private utility selector на точный dialog/sidebar, новый каталог
`anchor-final-fixed/` содержит настоящий PASS. Frozen repo tests не менялись.

Финальная сводка `.local/task006-qa/acceptance-summary-final.json` SHA256
`482802cc1127f92961d1e0bae2e70118203248b70398bbc416b11ec7e7be581b`.
Она отделяет final7ca7+4 проверки от087control5/browser-core4 и a2a fullcontrol10;
старые reports не выданы за повтор на последнем binary. Последняя7ca дельта
меняет только CSS `#create` scroll margin, другие scoped passes сохраняются
как regression evidence на указанных версиях. Runtime cleanup успешен.

Изменённые QA файлы: `refinement.cjs`, `refinement-features.cjs`,
`refinement-dashboard.cjs`, `refinement-visual.cjs` в `tests/web/` и эта WORK.
Product/source/общая TASK этой ролью не менялись. Рекомендуемый пользователю
скриншот: `.local/task006-qa/refinement-anchor/overview-1440-light.png`;
мобильные commands — `commands-390.png`, форма —
`.local/task006-qa/anchor-final-fixed/anchor-390-js.png`.

## Границы

Production/HTTPS deployment/Tandem/remote, физическая вторая машина, полный
accessibility/contrast audit и полный legacy14/upload/rollback — **NOT-RUN**
этим этапом. Два native процесса на одномMac не равны двум физическим машинам.
Git push/merge/release не выполнялись. Новые test sources проходят отдельное
независимое121 review; QA автора тестов не подменяет этот gate.
