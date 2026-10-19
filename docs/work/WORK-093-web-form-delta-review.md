# WORK-093 — независимое ревью валидации answer/release

- Роль: reviewer, исполнитель `/root/webdelta093`; автор изменения — другой агент `/root`.
- Статус: **APPROVE**. Открытых блокирующих замечаний нет.
- Родитель: TASK-004; зависимости — baseline code090/security091 APPROVE.
- Проверенная база: `f06338089bec3b8c7f17f541c463131f5c98f96c`, незакоммиченная frozen дельта в `.local/web-service-084`; точные SHA256 ниже.
- Прочитаны AGENTS Accord, процесс, reviewer profile, parallel-work и code-structure Давинчи. Именованные Claude-модели не применяются; модель наследуется и совпадает с моделью автора. Автор и reviewer независимы. Tandem OFF по прямому поручению; Давинчи не изменялся.
- Единственная запись в repository — этот WORK; репродукции только `/private/tmp/accord-review-093`. Git, deployment, реальные сервисы и пользовательские данные не изменялись.

## Проверенный объём

Узкий re-review `internal/web/mutations.go` и нового `form_validation_test.go`, затем согласованная координатором дельта `internal/webui/assets/app.js` и `templates/task.html` для ошибок возле полей. Остальная реализация повторно не аттестуется: её основание — WORK-090/091.

Whitespace-only answer теперь отклоняется как VALIDATION/400 до bindAction/runAction. Пределы включительные: answer 16384 байта, release reason 4096 байт; TrimSpace проверяет непустоту, исходный текст сохраняется. Невалидная форма не связывает native request, не меняет generation/срок lease; сервер сохраняет Draft, русскую field error и CanEdit при действующем собственном владении, выдаёт новый operation-bound action ID.

Новый unit-тест проверяет наблюдаемое исправление: whitespace → 400/сохранённые Draft+редактор → исправленный ответ303. Дополнительная независимая репродукция проверила ранний отказ до связывания и обе границы, включая release.

В JS 400/422 копирует fresh action ID и текст ошибки в постоянный элемент возле соответствующего поля через textContent; DOM/HTML из ответа не вставляется. hidden и aria-invalid обновляются, textarea связана с ошибкой aria-describedby. Черновик остаётся в исходном textarea, submit снова включается. 401/403/409 сохраняют блокировку с явным переходом; network/503 сохраняют прежний ID и возможность exact retry. Дельта не меняет Origin/CSRF/current-auth guards, generation fencing, bindAction payload equality, recovery/unknown-outcome ветки.

Переиспользование/декомпозиция проверены: новых существенных дублей или обходов типов нет; 247/32/198/106 строк соответственно, все четыре файла менее300 строк. UI-kit Давинчи N/A для самостоятельного Go SSR сервиса Accord. Две короткие проверки пределов локальны к форме и не требуют отдельной абстракции.

## Фактические проверки

Лично выполнены на toolchain Go из `.local/toolchains/go/bin/go`, GOTOOLCHAIN=local; существующие build/mod/tmp каталоги Accord:

- **PASS:** `go test ./internal/web -run 'TestWhitespaceAnswerIsCorrectableWithoutLosingLease|TestExactQuestionRaceAnswerCrashReplayAndFencing|TestExplicitHeartbeatReleaseAndStaleDraft' -count=1 -v` — три теста, web0.357s. Подтверждены исправление формы, exact replay после сбоя/expiry без повторного решения, payload conflict, stale generation и release/heartbeat regression.
- **PASS:** `go test -overlay=/private/tmp/accord-review-093/overlay.json ./internal/web -run TestReview093ValidationBeforeBindAndBoundaries -count=1 -v` — четыре subcase, web0.400s. Whitespace и превышение предела для answer/release дают400; original web_actions payload/result остаются пустыми, lease generation/expiry сохраняются; новая исправленная форма с ровно16384/4096 ASCII-байт завершается303.
- **PASS после финальной UI-дельты:** `go test ./internal/webui -run 'TestAllPagesEscapeUntrustedContent|TestUploadCSRFFirstAndFileLast|TestPaginationPreservesFilters' -count=1 -v` — webui0.329s, включая шесть escape-сценариев страниц.
- **PASS:** `node --check internal/webui/assets/app.js` на финальном SHA.
- Статическое чтение финального JS подтверждает safe textContent, fresh ID на400, сохранение ID для неизвестного результата и неизменную блокировку409. Это не browser-runtime доказательство.

## Версия и ограничения

```text
fdc4e56e82095c8880d0839073f2586e35d9b8e3ab7e8063d799c04757c6225b  internal/web/mutations.go
262fc0ce75905ae155eacf44273d10506321bcdd2dfc691e21f062f99880fc87  internal/web/form_validation_test.go
93d63bbf0dba76f9321a533f658715e1f4f7be50fd59cbcdab4f7946d0bd4a05  internal/webui/assets/app.js
108c5b90e3838d3cac6c6784622f51252535f4bcbfb5c28f073b96a7b66e9b02  internal/webui/templates/task.html
```

NOT-RUN этим reviewer: браузер/near-field accessibility runtime, новый process E2E, полный Go/race/vet, live deployment, rollback/restore. Эти результаты не подменены unit/static проверками. Отдельный UI092 и tester095 подтверждают свои критерии на финальном freeze; code approve позволяет начать QA095. Остальные ограничения090/091 сохраняются.

## Журнал и передача

- 2026-10-19: started — получена узкая дельта answer/release, прочитаны контракты роли и окружающий код.
- 2026-10-19: checked — три existing/new unit и четыре independent overlay subcase PASS.
- 2026-10-19: scope updated — координатор передал две UI-дельты near-field errors; source перечитан, template subset и JS syntax PASS.
- 2026-10-19: completed — **APPROVE** указанного manifest, передано координатору и QA095. Следующее изменение этих файлов требует затронутого re-review.
