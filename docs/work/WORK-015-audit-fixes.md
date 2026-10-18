# WORK-015 — исправления аудита перед публикацией

- Родитель: TASK-002. Роль: разработчик, агент `auditfix002`.
- Статус: на независимой проверке; исходная дельта заморожена 18.10.2026.
- Вход: AGENTS, TASK-002, требования/архитектура/API, docs/audit-002.md;
  профиль developer и общий процесс Давинчи прочитаны.
- Область записи: transport MCP+новая compatibility regression; store
  tasks/dependencies+новая audit regression+POSIX fixtures; CLI client/tests;
  эта WORK. Контрактные документы/общая TASK принадлежат координатору.
- Tandem отключён по указанию пользователя, соседний проект не изменялся.
  Git commit/push не выполнялись. Именованные Claude-модели в этой среде
  не закреплены; результат требует отдельного reviewer, это не self-approve.

## Исправленный результат

A2-01: разрешены только две версии штатного SDK: основная `2026-07-28` и
совместимая `2025-11-25`. Код собственного handshake не добавлялся. API
сверен с закреплённым официальным Go SDK v1.8.0: server.go:185–200 объясняет
SupportedProtocolVersions и legacy handshake, client.go:249–254 — опцию
ClientSessionOptions.ProtocolVersion. Новые тесты проверяют реальный HTTP
initialize → initialized → tools/list → tools/call legacy-протокола и
использование созданной board-session в REST. Дополнительный SDK-клиент
принудительно использует legacy и делает list/call. Прежний тест latest
SDK-протокола также проходит. Проверка установленного Codex — у координатора;
эти integration tests не выдаются за проверку реального агента.

A2-02: Next выдаёт только задачи эпика в состоянии open. Закрытый/отменённый
или отсутствующий родитель не даёт новый claim. Отмена не переписывает
состояния/попытки детей. Регрессия проверяет общий next, фильтр epic и точный
ключ задачи; после явного восстановления эпика в open первый claim снова
разрешён с generation=1 и сохранением истории.

A2-03: явно переданный ненулевой generation в Update/Link всегда требует
действующую собственную аренду. Истёкшая аренда не переключает такой запрос
на права requester/coordinator. Правило одинаково для link и unlink.
Административный запрос generation=0 к незахваченной задаче сохраняет прежние
права. Регрессии: holder=requester для developer и coordinator × update/link/
unlink; после материализации expiry и отклонённой stale-мутации неизменны
задача, session expiry, events и idempotency. Затем административная операция
проходит с тем же request_id: ошибка не расходует ключ и не продлевает сессию.

A2-04: CLI читает ответ с ограничением 1 МиБ + 1 байт и до любого stdout
отклоняет превышение. Ошибка явно сообщает неопределённость владения и
требует остановить локальные изменения, восстановить связь и проверить
аренду. Regression responses — валидный JSON размером limit−1/limit/limit+1;
первые два возвращаются полностью, третий даёт ошибку с пустым stdout.
Лимит остаётся ограничением пилота; пагинация всей истории задачи не добавлена.

UMASK: fixtures, которым намеренно нужны public0644, теперь явно Chmod0644,
не полагаясь на унаследованный umask. Production file security не ослаблена,
глобальный umask тестовыми исправлениями не меняется.

## Фактические проверки

PASS на авторской дельте, macOS arm64, локальный Go1.27.2:

- `go test ./internal/store ./internal/transport ./cmd/agent-board -count=1`
  при umask022: 0.484s / 0.880s / 0.264s.
- Та же проверка при umask077: 0.586s / 0.822s / 0.275s.
- `go test -race ./internal/store ./internal/transport ./cmd/agent-board -count=1`:
  4.781s / 2.270s / 1.554s.
- `make build`: PASS. Все изменённые Go-файлы форматированы gofmt и <300 строк.

Во всех запусках использованы GOTOOLCHAIN=local и игнорируемые локальные
GOCACHE/GOMODCACHE/GOTMPDIR в .local/toolchains; зависимости не обновлялись.
Полный make check, process/E2E, реальный Codex/Claude, VPS, два физических ПК
и Linux/Windows этим исполнителем NOT-RUN: владельцы координатор/тестировщик.
Следующий шаг — независимое recheck конкретной дельты, затем общая приёмка.

## Frozen SHA-256 manifest

Это идентичность восьми исходных файлов автора, не manifest всей TASK и не
заключение независимого reviewer.

```text
1cd74dd4941dd918f1722e8a506897f5924eaaecd7cc8f9f6fffb146385f803c  cmd/agent-board/client.go
cd54ee3ded558ccbccf8fd15b710dc1470243dfe25d6f9b21548feabbab6e92f  cmd/agent-board/client_test.go
0ff52c26c31f7fec8aff9cf18ec363c22cc42f8d5d92f4d68b1ae7ad1408b7e8  internal/store/audit_regression_test.go
9ec5bd101c0e3edb5f82e97f78bd1c4e7794eb129d94252e9b7ec23bab65c961  internal/store/dependencies.go
f9005b3d96f2c53d942b359202f7f41526da6759b681e1191dbcb0d4393be977  internal/store/files_unix_test.go
f88258a12fb0b996109b78769b4374a881dba04cc8327df40813735a3490814e  internal/store/tasks.go
337fb83db655810022dc056d03efb8d34fcbc52f16acb34acb5385095612fcb1  internal/transport/mcp.go
c551e24b4580eeda0c368374c5aa5f55b24af2da0338cbae3d8f9e0214a8339d  internal/transport/mcp_compatibility_test.go
```
