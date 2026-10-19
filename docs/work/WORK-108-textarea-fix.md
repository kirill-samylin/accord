# WORK-108 — точное восстановление textarea после R106-01

- Родитель: TASK-005 / REQ-005; роль developer, исполнитель `/root/control_draftfix108`.
- Статус: выполнено в границах реализации; независимое delta review ожидается.
- База по передаче координатора: `46fffb94f73130ddf5f654f3f11b5453123380d8`,
  рабочая копия `.local/web-control-099`; причина — единственный P2 R106-01
  из [WORK-106](WORK-106-control-code-security-review.md).
- Прочитаны AGENTS Accord, общий процесс и профиль developer Давинчи read-only,
  parallel-work, контракт control-web и TASK-005. Модели именованных Claude ролей
  здесь не закреплены. Tandem отключён по AGENTS Accord; Давинчи не изменялся.
- Владение передано UI104 через координатора: только три templates, новый
  `internal/webui/textarea_test.go` и этот WORK. Production вне границ frozen.

## Изменение

Во всех пяти затронутых местах сохранён один служебный LF после открывающего
`textarea`, а закрывающий тег поставлен сразу после template expression:
answer, release reason, readonly сохранённый answer, create description и
readonly поля recovery. Других изменений HTML, атрибутов, форм, CSRF/action IDs,
actions, условий показа, CSS или JavaScript нет. Экранирование `html/template`
сохранено; raw HTML, новые зависимости и новые продуктовые правила не вводились.

Причина сохранения начального служебного LF: HTML tree builder игнорирует ровно
один первый U+000A после textarea start tag. Поэтому служебный LF поглощается,
а собственный начальный LF пользовательского текста остаётся. Завершающий LF
парсер не поглощает; он удалён из шаблонного форматирования. Основание проверено
по официальному [WHATWG HTML, in body / textarea](https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inbody).
Это узкое исправление восстановления прежних форм, не изменение их контракта.

## Фактические проверки

- RED до правки templates:
  `.local/toolchains/go/bin/go test ./internal/webui -run '^TestTextareaDraftValuesExact/plain$' -count=1`
  — FAIL: answer и reason получали `exact retained draft\n` вместо
  `exact retained draft` (0.893s).
- PASS после правки:
  `.local/toolchains/go/bin/go test ./internal/webui -count=1` — 0.447s.
- Новый unit regression сравнивает полные значения и список textarea через
  `reflect.DeepEqual`, а не наличие подстроки. Проверены пустое/обычное значение,
  многострочный русский текст, HTML-looking текст с закрывающим textarea/script
  и entity-looking строкой, один/несколько начальных LF, завершающие LF,
  только whitespace, пробелы/табуляции вокруг текста и ровно16384 UTF-8 байта.
  Проверяются editor answer/reason, readonly stale answer, create description
  и три recovery поля. Пустые readonly/recovery поля по прежним условиям скрыты.
- Unit reader извлекает реальный SSR output и моделирует относящиеся к случаю
  RCDATA/entity/newline правила HTML; это не полноценный HTML parser и не browser
  evidence. Совпадение ведущих LF подтверждено этими unit cases и правилом
  спецификации. Побайтовое сохранение произвольных CR/NUL не заявляется:
  браузер нормализует входной HTML согласно собственным parsing rules.
- NOT-RUN: новые process/browser E2E, actual visual, полный make check/race,
  deploy/HTTPS/load/physical second PC; это независимые этапы. Node6 не повторялся,
  поскольку JavaScript не менялся. Git/remote/release не выполнялись.

## Проверенная версия и передача

SHA-256 изменённых source/test файлов после PASS:

| Файл | SHA-256 |
| --- | --- |
| `internal/webui/templates/task.html` | `f7c8666e7c0cafb6950e4c4fa8a990d72fa39b3c8a1a8a1d8c41ec183891fe86` |
| `internal/webui/templates/create.html` | `24337a549b0672f9ce0d8d5f19e389677a25e78414b5cf89f0a822d4348f6c30` |
| `internal/webui/templates/layout.html` | `1ddb2d404ddbc4880ffdb56dbb1ca9202e0e44bf6fb5dbb0a3ccfed55f91328d` |
| `internal/webui/textarea_test.go` | `fff96ecf71da215c39b02699a64962f72855c1f06ce1245608026438fc60110e` |

Следующий исполнитель — новый независимый delta reviewer по назначению root;
после актуального approve координатор открывает process/browser QA107. Автор
не выдаёт себе approve и не объявляет TASK-005 завершённой.

## Журнал

- 2026-10-19T22:37:10+03:00 started — фактический date при чтении scope/контракта;
  узкая область и отсутствие других authors подтверждены передачей root.
- 2026-10-19T22:38:01+03:00 verified — фактический date после red/green проверки;
  unit webui PASS, SHA source/test сняты.
- 2026-10-19T22:38:30+03:00 completed — фактическое время записи WORK; реализация передана
  координатору для независимого delta review.
