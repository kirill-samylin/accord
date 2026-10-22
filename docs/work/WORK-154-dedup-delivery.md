# WORK154 — поставка дедупликации и «Команды»

22.10.2026. Координатор; область записи: TASK012, этот WORK и private
`.local/task012/deploy/`. Product исправляет отдельный автор156; source/tests
проверяет153; новые process/browser проверки пишет155 после SOURCE APPROVE.

## Полномочие и checkpoint

Пользователь: «залил новые изменения. залей, обнови стенд и дай ссылку».
Это разрешает push проверенной `codex/dedup` и безопасное обновление прежнего
Accord VPS. Main/merge/PR не требуются. Управляемый запуск этой поставкой не
реализуется; прежние вопросы о worker не относятся к дедупликации.

Исходный source `4ec5ccaeec929136d4aa900869d54b1f8a6c614a`, parent `cebcbe5`.
Read-only VPS preflight подтвердил Go/Next release
`e53c18c9643b9ee086141d0f227ab485f4f88cac`, Go SHA256
`a03f64d5ff008b7f63023117fb7b1c15c33e5617ce7ac08ffc6440610315269d`,
Next BuildID `HHViYVPwDuQx7uj4Uk1YC`, active services/timers, NRestarts0,
inspection/restore markers отсутствуют. Это проверено заново; прежний snapshot
не используется как доказательство текущих данных.

## Проверки и порядок

Root `make check` на source4ec5cca **PASS**: Go fmt/vet/all/race/build,
embedded web, infra155, Next72/tests/types/format и настоящий Next production
build. Прежний NOT-RUN production build закрыт локальной сборкой на Node24.19.
Лог private `.local/task012/deploy/make-check.log`.

Первичный secret-pattern scan66changed files: совпадений0. Это ограниченный
pattern scan, не доказательство отсутствия всех возможных пользовательских
данных. Случайный generated Next cache перед push удаляет автор156 и добавляет
ignore. Credentials/реальные VPS данные не включаются в отчёты или Git.

Новую persistent duplicate semantics до поставки защищает собственный capable
floor; board schema4/coordination/ACL сохраняются. Независимый SOURCE APPROVE,
авторские regression tests, отдельный QA и review новых tests обязательны.

Поставка: clean immutable Go Linux/Next bundle → exact hashes → inactive
transfer → actual hardened stage → full all-project snapshot и штатный
`infra/accord/projects/upgrade.py apply` → независимый readonly HTTPS →
finalize/full backup → actual owned runtime/preservation → final review.
Initial TASK009 apply, DB rollback, bootstrap/reset и старый Go запрещены.
Remote Python только `-B`; unknown recovery artifacts не удаляются.

`deploy-task012.command` оставлен как чужой untracked файл; не исполняется.
Его комментарий об автоматическом rollback не соответствует capable updater.

## Статус

IN PROGRESS. WORK153 **SOURCE + AUTHOR TESTS APPROVE**:69files manifest SHA256
`7b55514bb25f43a13aec24c4d71664d89f771ee272cd55613f347e0eec756290`.
Root независимо повторил формулу и получил тот же hash. Final root `make check`
**PASS** после всех fixes: Go/all/race/vet/build, embedded16, infra157,
Next72/types/format/production build. Лог `make-check-final-source.log`.
Tester155 теперь разрешены новые process/browser tests. Release/push/production
update/final review пока не завершены; результаты дополняются по факту.
