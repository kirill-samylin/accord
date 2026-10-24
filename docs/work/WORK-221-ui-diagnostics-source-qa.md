# WORK221 — отдельный helper диагностики UI, source checkpoint

23.10.2026, ops_validation_qa. Scope: только два private файла V8; контракт
`WORK-221-next-stop-ui-diagnostics.md`, остальные V8 файлы и общий freeze принадлежат
gui_connection_qa. Production, конфигурация и исторические FAIL не менялись.

`captureFailure(permit,c,ui,error)` возвращает конечные факты OBSERVED/NOT_OBSERVED.
Общий лимит — остаток текущего cleanup permit, максимум 5 секунд; screenshot и
DOM read запускаются только в существующей странице. Незавершённые promises
сразу anchored и отслеживаются в `c.uiDiagnosticReads`; интегратор закрывает
свой браузер и bounded join всех этих reads до положительного closure/config proof.
Поздний ответ не пишет файлы: обработка результата после общего deadline не
продолжается, а каждая запись дополнительно проверяет active/deadline/permit.

Артефакты `ui-failure.private.json` и `ui-failure.png` пишутся исключительно WX,
0600, fsync file+directory, в проверенный собственный 0700 runtime root. Screenshot
только viewport PNG, buffer до открытия файла, максимум 5 MiB; маскируются все
controls/contenteditable/pre/code и дополнительные формы/блоки потенциальных
значений. DOM возвращает только семь ограниченных счётчиков и восемь booleans
сравнения с фиксированными известными labels. Текстовые узлы controls/raw blocks
не читаются; никакие произвольные текстовые значения не сохраняются. Stack
содержит только allowlisted owning CJS basename/line/column, максимум 16 записей.
Helper не читает credential, HTML, cookies, body, headers или raw error message.

Source hashes:

- `ui-diagnostics.cjs`: `e3a8fce0c4e021054183d3c933d8b53b66838c15736dead695cdfc6a24ee8059` (83 строки).
- `ui-diagnostics.test.cjs`: `c6fd48aa4fc820ea86fb5026543f34a8be6c741c4d65b401230d444d2284622e` (29 строк).

Pinned Node 24.19.0 syntax-only `--check` обоих файлов: PASS. Три authored pure
группы покрывают projection stack, строгую finite DOM projection и отсутствие
записи expired/sealed/oversized/late buffer. Тесты не исполнялись: NOT-RUN до
независимого source review202. Импорта helper, браузера, HTTP, model/native run,
изменения настроек и общего manifest этим исполнителем не было. SOURCE_READY
передан root/gui_connection_qa; whole-V8 freeze и review принадлежат интегратору.

До whole-freeze интегратор запросил локальную проверку origin страницы и явный
source binding. Добавлены `page_origin_verified` (без URL/query) и validated
64hex `source_manifest_sha256`; чужой origin не допускает DOM/screenshot и даёт
NOT_OBSERVED. Явный `truncated:boolean` сообщает count/node/text cap, без raw
значений. Финальные hashes заменяют предыдущий component checkpoint:

- helper: `86063c20963580bb394cce2604ee5f3d97d3f0873fa2590aad778b34b7d2f5fc` (86 строк).
- pure tests: `3279448014290e2fa0966818d7cde0678bdebe7c6c7feaae2d476264e999e46b` (31 строк).

Повторный syntax-only PASS обоих файлов; новые pure cases не запускались.
Финальный checkpoint передан интегратору, дальнейшие изменения закрыты до review.
