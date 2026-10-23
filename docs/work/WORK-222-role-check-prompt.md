# WORK222 — инструкции запуска проверок по роли

24.10.2026; author automation_worker_impl, coordinator root, independent review202.
SOURCE READY / FROZEN, review и native QA ещё NOT-RUN.

## Причина и границы

По переданному root результату actual203V11 аналитик и разработчик завершили
работу, а ревьюер попытался выполнить `workflow_run_check` и получил отказ.
Исходный `Engine.prompt` предписывал этот инструмент всем automation-ролям,
тогда как `automationCheckRequest` разрешает новый запуск только developer/tester.
Независимо от runtime-отчёта это противоречие подтверждено чтением обоих путей.
Перед изменением прочитаны automation/managed-agents contract и TASK014.

Изменены только `internal/worker/execution.go`, новый
`internal/worker/role_check_prompt_test.go` и эта карточка.
ACL, lease/IPC/attestation, native permissions, sandbox и approval gates сохранены.
Другие рабочие изменения, конфигурация и диагностические артефакты не менялись.

## Реализация

- Developer/tester получают требование выполнить обязательные разрешённые preset ID,
  дождаться completed и прочитать реальный результат. Queued/running, failed,
  NOT-RUN и N/A не являются PASSED. Developer сохраняет prerequisite clean committed HEAD.
- Analyst получает workflow_plan под собственной арендой и workflow_question;
  планирование не требует запуска проверки.
- Reviewer получает read-only review точного bound target SHA и diff с доступными
  qualified evidence, честный approve/changes_requested/blocked и запрет собственного
  запуска preset. При необходимости человеческого уточнения остаётся workflow_question.
- Общие правила describe/next, lease loss, immutable profile, source/HEAD preservation,
  consented artifacts и test_design/NOT-RUN сохранены. Ручные managed run и текст
  continuation остаются прежними.

## Проверки

PASS: targeted Go race, 1.531s, четыре test function:

```text
TestAutomationPromptMatchesCheckRoleAuthority
TestAutomationPromptDoesNotGrantAnalystOrReviewerChecks
TestRoleCheckPromptPreservesManualAndContinuationRequests
TestLocalCheckReplayNeverRespawnsAndCompletionWaitsDurableServerACK
```

Новая typed-role matrix проверяет все четыре роли и обязательность настоящих
результатов у developer/tester. Отдельные mock/file units подтверждают отказ
полностью bound analyst/reviewer запросам без создания execution intent.
Существующий положительный тест проверяет реальный local-check executor с fake
process, TAP parsing, idempotent replay и durable server ACK; новых process/E2E нет.
PASS: gofmt и scoped git diff --check. Файлы 256/72 строки, исключения не нужны.

Native/CLI/model/config RPC, build, commit, push, production и physical Windows/second PC:
NOT-RUN автором. Этот patch ещё не доказывает успех полного native pipeline.

## Source pin

Base HEAD: `f5956d892eb8c86db3c7113198098bb11cd6999e` (root docs-only HEAD).
Product base по root — a32; новая worker revision ещё не выпущена.

```text
371912fdf76cdc56be2522d8a196e8acc071f881de76cc463ab49267a50169cd  internal/worker/execution.go
e3d35e626d4001657c469adc4ee95f7867a9badd7a78f7b2739c2168a57fc289  internal/worker/role_check_prompt_test.go
```

Следующий шаг: независимое source/batch review202, затем только отдельный root GO
на выпуск patched worker и native QA по WORK223. Серверные пути не изменены.
