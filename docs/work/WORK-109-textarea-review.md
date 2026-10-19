# WORK-109 — независимое delta review исправления textarea

- Родитель TASK-005 / REQ-005; роль reviewer, исполнитель `/root/control_delta109`.
- Статус: выполнено. Вердикт **approve** узкой дельты WORK-108; **R106-01 закрыт**.
- Проверенная версия: `a30746c06ae9379c0fca09ac1757330f0f69ac42`, база
  `46fffb94f73130ddf5f654f3f11b5453123380d8`.
- Прочитаны AGENTS Accord, общий процесс/parallel-work, профиль reviewer и правила
  структуры Давинчи read-only, соответствующие части control-web-contract,
  WORK-106 и WORK-108. Автор исправления — другой агент. Использована унаследованная
  модель, равная модели автора; именованные Claude модели здесь не закреплены.
- Tandem OFF по прямому исключению AGENTS Accord. Запись только в этот WORK и
  `/private/tmp/accord-control-099/delta-review/`; исходники, Git, сервер и Давинчи
  не изменялись. Чужой WORK-110 сохранён.

## Границы и freeze

Независимо сверены все **213/213** source/test записи manifest
`/private/tmp/accord-control-099/source-freeze-fixed.json`; mismatches=[].
SHA-256 manifest: `7afa42654df715f1df3ddb7b33cf18facf43f86bf3d0589261b779639c0ad4ad`.
Evidence: `delta-review/freeze-verification.json`.

В source diff от базы только `internal/webui/templates/{task,create,layout}.html`
и новый `internal/webui/textarea_test.go`. Изменения общих документов координатора
не входят в этот source approve; полное предыдущее ревью не повторялось.
Сохранён diff: `delta-review/reviewed-source.diff`.

## Решение по R106-01

Сравнение каждой пары файлов подтвердило: все production изменения — ровно
удаление одного LF непосредственно перед пятью закрывающими `</textarea>`:
editor answer, release reason, readonly stale answer, create description,
recovery field template. Атрибуты, условия, формы и выражения не менялись.
Evidence: `delta-review/delta-inspection.json`.

Служебный LF непосредственно после открывающего тега оставлен корректно:
официальный [WHATWG HTML, textarea tree construction](https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inbody)
предписывает проигнорировать ровно один следующий U+000A. Таким образом он
поглощается, а намеренные ведущие LF данных остаются; завершающих добавочных LF
больше нет. Пустое значение остаётся пустым, 16384 UTF-8 байта не растут до16385.

Новый regression проверяет полные значения и число полей, а не подстроки:
пустое, обычное, русский multiline, HTML/entity-looking данные, один/несколько
ведущих LF, завершающие LF, только whitespace, окружающие пробелы/табуляции,
ровно16384 байта. Охвачены оба editor, readonly stale, create и три recovery поля.
Тестовый reader нормализует HTML newlines, однократно декодирует entities и
снимает только один initial LF. Это целевой unit reader, не полный HTML parser
и не доказательство browser DOM. Произвольные CR/NUL byte-exact не заявляются.

Production не добавляет trim или иной нормализации. `draft`/`answer`/`savedDraft`
возвращают строки; прежнее contextual `html/template` escaping сохранено.
Raw `template.HTML`, новые секреты, зависимости и ослабление CSP не добавлены;
security middleware, CSRF/action IDs, fencing и JavaScript вне дельты неизменны.
Новых production дублей/абстракций нет; UI-kit Давинчи N/A для отдельного Go SSR
Accord. Затронутые файлы 258/90/187/85 строк, исключений к300 строкам не нужно.

## Фактические проверки

PASS на Go1.27.2 darwin/arm64:

```text
.local/toolchains/go/bin/go test ./internal/webui -run '^(TestTextareaDraftValuesExact|TestAllPagesEscapeUntrustedContent|TestOwnedAnswerAndStaleDraft)$' -count=1 -json
```

- Package PASS, 0.398s, exit0. Новый точный regression —11/11 cases;
  escape regression —6/6 pages; owned editor/stale readonly regression PASS.
- PASS `git diff --check 46fffb94f73130ddf5f654f3f11b5453123380d8..HEAD`.
- JSON evidence: `delta-review/targeted-tests.jsonl`, `targeted-summary.json`.
- NOT-RUN: полный make/race, Node suite, новые process/browser E2E, actual visual,
  live/HTTPS/deploy, нагрузка и physical second PC. Они не подтверждаются этим
  approve и не требуются для повторного узкого source review.

Блокирующих замечаний в дельте нет. В совокупности с source/security выводами
WORK-106 этот approve снимает R106-01 и разрешает переход к отдельной QA107
и актуальному независимому дизайн-ревью. Общая TASK-005 не объявляется завершённой.

## Журнал

- 2026-10-19T22:39:31+03:00 started — фактический date, чтение scope/профиля.
- 2026-10-19T22:40:12+03:00 verified — freeze213/213, targeted unit/regressions PASS.
- 2026-10-19T22:40:25+03:00 reviewed — подтверждено ровно5 удалений LF и отсутствие
  иных production изменений; approve передан координатору после записи WORK.
