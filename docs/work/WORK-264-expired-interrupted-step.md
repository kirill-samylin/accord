# WORK264 — истёкший срок прерванного этапа

Автор: ops_validation_qa. Статус: SOURCE READY / developer unit PASS; независимое review и production-проверка ожидаются.

## Фактическая причина

В ограниченном сохранённом снимке реального цикла текущий этап `interrupted` имел истёкший deadline, тогда как общий workflow deadline ещё действовал. `workflowContinueTurn` возвращал общий `CONFLICT`; scheduler для queued workflow сохранял queued и увеличивал версию при каждом повторе. Поэтому сообщение о занятой папке не объясняло причину, а optimistic version мешала человеческому управлению. Исторический текст ошибки native run не используется как доказательство этой причины. Снимок и его SHA находятся только в приватных references.

## Изменение

- `internal/store/automation_advance.go`: отдельный `STEP_TIMEOUT` для истёкшего native continuation и незавершённого interrupted этапа, включая отсутствие созданной native conversation. Обычный обработчик ошибок переводит workflow в blocked; следующие ticks больше не меняют workflow. Результат уже завершённой задачи по-прежнему проверяется прежними evidence/SHA gates.
- `internal/store/automation_step_timeout_test.go`: durable human question/stop/answer после истечения срока, недиспетчеризованный interrupted этап, точная граница deadline, стабильные последующие ticks, отсутствие новых команд/изменений задач/расхода budget и путь human pause → explicit replan. Новый этап расходует обычный invocation, общий deadline и потолок запуска сохраняются.
- `web/src/lib/automation-reasons.ts`: только существующая строка `step_timeout`, с действительными названиями кнопок «Пауза» и «Пересмотреть план». React-логика не изменена.

Сроки, делегация, ACL, sandbox и бюджеты не расширены. Общие transient workspace conflicts сохраняют прежнюю последовательность запуска.

## Проверка

Targeted 6 unit tests: PASS, 0.543s, включая два новых test functions и две interrupted подветки. Полный `internal/store`: PASS, 7.473s. Pinned gofmt и diff whitespace: PASS. Первая авторская итерация выявила ошибку сравнения raw внутреннего grant с redacted WorkflowGet; тест исправлен на сравнение одинаковых представлений, прямые Changed/version assertions сохранены.

Приватный frozen manifest: `.local/task014/store264-step-timeout/source-pins.json`, SHA256 `cfcf4d715d20ccf9dff7e085b4ec5d92d7139edce5d77630a8466ba9b339e998`. Копии исходников, pinned references и наблюдённые unit результаты сохранены отдельно; это не архив сырых subprocess потоков.

Production deployment, server control, повторный native запуск и browser QA: NOT-RUN автором. Реальный предыдущий queued результат не переименован в PASS. Поставка требует независимого review и отдельного разрешённого capable обновления с сохранением данных.
