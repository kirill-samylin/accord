# WORK-112 — независимое code/security delta review WORK-111/113

- Родитель TASK-005 / REQ-005; исполнитель `/root/control_batchreview112`, reviewer + security delta.
- Статус: выполнено. Вердикт **APPROVE** узкой дельты WORK-111/113; блокирующих замечаний нет.
- Проверенная версия: `ac6c965f3391822b803114efde2a7dcb757ed38c`; база
  `a30746c06ae9379c0fca09ac1757330f0f69ac42` (approve WORK-109 исправления R106-01).
- Прочитаны AGENTS Accord, общий процесс/parallel-work, профили reviewer/security
  Давинчи read-only, правила структуры, применимые части control-web-contract,
  WORK-106/109/110/111/113. Полное старое приложение повторно не аудировалось.
- Независимый от автора агент; модель унаследована и равна модели автора.
  Именованные модели Claude в этой среде не закреплены.
- Tandem OFF по прямому исключению AGENTS Accord. Запись только в этот WORK и
  `/private/tmp/accord-control-099/batch-review/`. Source/Git/SSH/deploy/Давинчи
  не изменялись; параллельные QA-файлы сохранены.

## Проверенная дельта и provenance

Production diff содержит только `internal/core/control_validation.go`,
`internal/web/control_queries.go`, `internal/webui/assets/{cards,shell}.css`,
`internal/webui/templates/layout.html`; тест — `internal/web/control_epic_test.go`.
Документы координатора в commit не являются предметом source approve.
Сохранён diff: `batch-review/reviewed-source.diff`.

Самостоятельная сверка manifest `/private/tmp/accord-control-099/source-freeze-batch.json`:
**214/214** source/test записей совпали, mismatches=[]; revision и HEAD совпали.
SHA-256 manifest: `451e6888353900b1ba8016bdf2830c185b961b892d2749bb6fc83d10bae50a1c`.
SHA-256 `/private/tmp/accord-control-099/accord-batch-final`:
`9efa45893de16e369d3a12bd21add9ca9793f891993f229a5af0539250f25a27`.
Evidence: `batch-review/freeze-verification.json`. Этот review не запускал бинарь
для actual UI и не выдаёт его SHA за проверку визуального результата.

## Решение по коду и безопасности

- `ValidControlTaskKey` сохраняет прежнюю T-only семантику через общий private
  helper. `ValidControlEpicKey` допускает E-only; обе проверки требуют положительное
  int64 и точный canonical decimal roundtrip. Zero, отрицательные, leading zero,
  плюс, whitespace, неверный prefix и overflow не проходят; MaxInt64 допустим.
  Разделение `epic_key`/`task_key` в query не ослабляет unknown/duplicate parameter
  rejection и не меняет разрешённые ключи маршрутов.
- Новый regression проходит настоящие HTTP handlers, login и store fixture:
  пять действующих/legacy маршрутов с E-N возвращают200 и сохраняют EpicKey;
  неверные epic значения дают400. Activity по T-N проходит, E-N как task_key
  отвергается. Это проверка поведения browser perimeter, не копия валидатора.
  Сохранённый автором `epic-before-fix.log` подтверждает400 до исправления;
  `batch-fix-tests.log` — affected packages PASS, отдельно от собственного запуска.
- Угрозы этой дельты: подмена namespace query и обход input validation. Разделение
  префиксов их устраняет. Auth/session/CSRF/actions/leases/DB schema/native
  protocol и список12 native operations в source diff не меняются. Новых raw HTML,
  URL/resource imports, сетевых адресов, секретов или зависимостей нет.
- Ширина `min(240px, 100vw)` находится только внутри max-width767 media и
  `html[data-enhanced=true] .sidebar`; no-JS width100% и desktop rules сохранены.
  Badge state selectors получают такую же specificity, как generic badge base,
  и выигрывают по порядку. Цвета остаются прежними, правила локальны `.badge`;
  `.status-dot` clauses не меняются. Actual geometry/colors — отдельный WORK-114.
- В пустом `.submit-status` удалён только LF text node: существующий `:empty`
  может скрыть рамку, role/aria-live остаются. Формы, textarea values, actions и
  JavaScript не меняются. Контекстное escaping и точность draft проверены ниже.
- Новых production дублей/ненужных абстракций нет; общий canonical helper
  переиспользует одинаковую семантику. UI-kit Давинчи N/A для отдельного Go SSR
  Accord. Размеры затронутых файлов35/154/50/185/269/186 строк, исключение к300
  не требуется. Новые поддерживаемые API/зависимости не выбирались.

## Фактические проверки

PASS на Go1.27.2 darwin/arm64:

```text
.local/toolchains/go/bin/go test ./internal/web -run '^(TestControlEpicFiltersPreserveExistingRoutes|TestBrowserOriginCSRFAndDraftBoundaries|TestControlUnknownAndInvalidQueriesRetainDraft)$' -count=1 -json
.local/toolchains/go/bin/go test ./internal/webui -run '^(TestStaticOnlyKnownAssets|TestControlAssetsHaveExplicitTrustedMediaTypes|TestAllPagesEscapeUntrustedContent|TestTextareaDraftValuesExact)$' -count=1 -json
git diff --check a30746c..ac6c965f3391822b803114efde2a7dcb757ed38c
```

- Web:3/3 targeted tests, package PASS0.679s, exit0. Origin/CSRF/replay/draft
  boundaries и unknown/invalid query retention сохранены.
- Webui:4/4 targeted top-level tests, package PASS0.715s, exit0; escape6/6 pages,
  textarea11/11 cases, explicit static media types и known-assets restrictions.
- `git diff --check` PASS. JSON evidence: `batch-review/web-tests.jsonl`,
  `webui-tests.jsonl`, `targeted-summary.json`.
- NOT-RUN: полный make/race, полный Node/process/browser E2E, actual visual,
  live HTTPS/deploy/load/native runtime/physical second PC. По разделению роли
  они выполняются координатором/QA107/дизайн-ревью114 и не подтверждаются здесь.

APPROVE разрешает продолжить финальную QA107 на этой версии. Source/security
заключения WORK-106 и исправление textarea WORK-109 не заменяются повторным
полным аудитом; новых блокирующих рисков в проверенной дельте не найдено.
Закрытие visual UI110-01/02 требует actual retest114; TASK-005 не объявляется
завершённой этим документом. Дополнительных обязательных source исправлений нет.

## Журнал

- 2026-10-19T22:50:06+03:00 started — фактическое время, профили/process/scope.
- 2026-10-19T22:51:07+03:00 verified — freeze214/214 и targeted checks PASS.
- 2026-10-19T22:51:47+03:00 reviewed — source/security delta APPROVE передан
  координатору; следующий шаг actual114 и финальная QA107.
