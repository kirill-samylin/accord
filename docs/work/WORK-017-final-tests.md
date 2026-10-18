# WORK-017 — финальный ретест TASK-002

- Родитель: TASK-002. Роль: независимый tester, агент `finaltest002`.
- Статус: выполнено, PASS. Основание: APPROVE WORK-016 / audit-002-recheck.
- Область записи: эта WORK, docs/runtime-002-final.md,
  .local/evidence/final002. Исходники, тесты и зависимости не меняются.
- Проверяется frozen manifest43 SHA-256
  `699678fd6d9912d554ea10e4b663db5de84d8eda48d4c9652a54ec6f81343432`.
- Общий процесс, профиль tester, AGENTS/TASK-002/runtime-002/recheck прочитаны.
  Tandem не используется, соседний проект не меняется. Именованные модели
  Claude не закреплены в этой среде; тестировщик отдельный от автора.

## План

Полный make check при umask077, включая все пять process-тестов и race;
каждый go test получает `-count=1 -v` через локальную обёртку Go, прочие
команды передаются без изменения. Дополнительно file-security fixtures при
umask022, go mod verify, существующий govulncheck и Linux/amd64 CGO0 build.
Сравнить source manifest до/после, сохранить exit codes, SHA и версии.
Реальный Codex повторно не запускается; источник — docs/codex-002.md.

## Журнал

- 2026-10-18T16:10:01+03:00 started: source manifest43 сверён с WORK-016.
- 2026-10-18T16:10:23+03:00 make check077 PASS, все5 process и race,
  count1; format/vet/build PASS. Fixtures022, modverify и govulncheck PASS.
- 2026-10-18T16:11:07+03:00 completed: Linux/Windows amd64 CGO0 build PASS,
  итоговый source manifest совпадает с исходным. Это compile-only.

## Результат и передача

[Финальный протокол](../runtime-002-final.md) содержит все exit codes,
версии, времена, SHA и границы вывода. Полный local-набор пройден;
исторический umask077 FAIL закрыт фактическим ретестом. Исходники/тесты/
зависимости не менялись. Итоговый macOS binary и два process-прогона имеют
SHA-256 `b7abf6b5925b83a5cbba984b47d967735a5d3583bd2253ea8fe206a051234eee`.

NOT-RUN: physical2PC, Claude, удалённый Accord/TLS, Linux/Windows runtime,
нагрузка V1/fault injection. Codex smoke принадлежит координатору; повторно
не запускался. Acceptance дочерний сервер собирается без race; инструментирован
Go test/harness и unit/integration. Полная V1 этим PASS не объявляется.
Следующий владелец — координатор: сверить итоговые доказательства, commit/push
проверенной ветки в пределах пользовательского поручения. Git-действий,
Tandem/сервера и глобальных настроек этот tester не менял.
