# WORK-022 — резервирование и локальное администрирование

- TASK: [TASK-003](../TASK-003-working-service.md).
- Роль: разработчик; профиль `davinci-gui-/docs/agents/roles/developer.md`.
- Статус: на независимом ревью.
- База: `966dda6`; отдельные Claude-модели ролей в текущей среде не закреплены.
- Область: `internal/core/admin.go`, `internal/store/auth.go`, новые
  `backup.go`, `maintenance.go`, `admin_sessions.go` и admin/backup tests;
  `cmd/agent-board/admin.go`, `token.go`, admin tests; узкая подготовка
  private0700 fixture в существующем `client_test.go` согласована координатором.

## Реализация

Локальные `token list` и `sessions list` показывают только заявленные поля
идентичности/срока; hashes и bearer не сериализуются. `token create`, `revoke`
и `revoke-all --reason` записывают audit в той же SQLite-транзакции; повторный
revoke не создаёт повторного события. Изменение credentials откатывается,
если audit невозможно сохранить. Агентский API не получает admin endpoint.
Create остаётся явным bootstrap; остальные admin команды используют existing
private file, SQLite `mode=rw`, schema1. Опечатка пути не создаёт новую БД.

`backup --db SOURCE --output NEWFILE` использует параметризованный SQLite
`VACUUM INTO` при работающем источнике/WAL. Источник — существующий regular0600,
без URI/symlink и writable чужими parent; output parent0700. В том же каталоге
создаётся private stagingO_EXCL0600; проходят schema1 и полный `integrity_check`,
file fsync/SHA256. Затем `os.Link(stage,final)` атомарно публикует snapshot
без overwrite, directory fsync, удаление stage и повторный directory fsync.
Если hardlink недоступен, процедура отказывает; final не бывает частичным.
При ошибке удаляются только пути, всё ещё указывающие на собственный inode.
В stdout только SHA/time/schema metadata. Этот протокол согласован координатором
вместо предварительного создания finalO_EXCL, видимого до заполнения.
Владельцу каталога и процессам того же UID доверяем; защиты от злонамеренного
root/того же UID эта процедура не обещает.

`maintenance reset --db RESTORED --reason restore` — **только offline** после
остановки всех старых серверов. Одна транзакция отзывает все восстановленные
credentials, истекает все sessions, закрывает все незавершённые attempts
с reasonrestore и снимает leases с переводом задач ready. История,
role/task generations, idempotency не удаляются/не снижаются. Новый token
не может использовать старую session; следующий claim увеличивает generation.
Повторный запуск не возобновляет старые identity. CLI сам не останавливает
сервер и не доказывает его остановку: это предусловие infra/оператора.

## Проверки

- `make test`: PASS на промежуточной реализации, включая прежние process tests.
- Финальный `go test -race -timeout 60s ./internal/store ./cmd/agent-board`:
  PASS,store6.323s/cmd1.558s,18.10.2026; отчёт зафиксирован UTC14:35:01.
  Включает новую атомарную
  публикацию и реальное прерывание synthetic64MiB snapshot после появления
  private stage: final до проверки отсутствует, после cancel нет обоих файлов.
- Промежуточный `make check` остановлен по конкретным собственным PID:
  параллельно создаваемый bridge test имел зависающую timeoutfixture.
  Автор bridge подтвердил и исправил fixture. Полный итоговый `make check`
  принадлежит координатору после freeze параллельных изменений.
- Unit/integration: online WAL и concurrent writer, соответствие token/audit
  в snapshot; integrity/schema/checksum/private modes; snapshot restart;
  token из snapshot после последующего revoke на source и recovery отклонён;
  старые session/generation не используются новым token; fencing растёт;
  rollback credentials+session+lease при отказе audit; missing/URI/symlink/
  existing/unsafe permissions/sidecar paths отклонены без исправления прав;
  failed VACUUM убирает собственный partial output.
- Новые process/E2E здесь не добавлялись; независимое ревью ещё не выполнено.
- Live VPS, физический второй ПК, Windows runtime: NOT-RUN этим исполнителем.

## Следующая передача

Независимым reviewer/security проверить весь admin diff, протокол публикации
snapshot и предусловие offline recovery; после approve отдельный tester
проверяет backup/restore/recovery процессами и эксплуатационными скриптами.
Официальное поведение `VACUUM INTO` проверено координатором по
[SQLite](https://sqlite.org/lang_vacuum.html): согласованный snapshot,
существующий пустой target допустим; fsync результата выполняет приложение.

## Проверенная версия admin slice

Незакоммиченная дельта к966dda6; shared `main.go`/`types.go`/`sessions.go`
принадлежат WORK-021 и в этом manifest не заявляются как авторские файлы.
SHA256:

```text
4789d71da2790514fcd72016216df6d1d40b1f107d871ce9a1e3434cdf62f0cb internal/core/admin.go
4957c4652b0c0308220ff8294601fcf0a511793ba129199866ae78fc7e4961e1 internal/store/auth.go
f600d45fcb36c7d554db132ca08ef5b2e425bd07ef85a3b59e3a6534b3b6f15d internal/store/maintenance.go
51c7c36b765e8d2c6364ec6d5ead49fb8d2e622141ed7fa6b928c1a7582ed284 internal/store/admin_sessions.go
208ff1f411c1018be05aec835c85ffa3c0d7061fbf8f18479a441d99e76cbb1a internal/store/backup.go
b8b1d4d728c6457e5841795141b8b05cbd9a6dcd65e4b2412fc83ce59fd94926 internal/store/admin_test.go
3392595cd4159bad9eac8ead7af9e3a5cac003cd26941dbb5266cd14e9a6e3c2 internal/store/backup_test.go
2496a9b895e37e6e37acfcb3b7ef162e298db90acb0ea986a82b5c716e93bb70 internal/store/backup_unix_test.go
a72703b8157846952835b5e8ee60d5fde484a154414d4c7fd8af37284a4fb19b cmd/agent-board/admin.go
e17c1a193d44e67b480b70cf97fd9fa602ec21b2b755aabfe9ab5330357d9551 cmd/agent-board/admin_test.go
cdb5c610bf26f85b05c04fb2448556e345ec056caf9f3dd0ce3b5506579b1e01 cmd/agent-board/token.go
274d1a4efc15e2a610d3c49bca91790a66736b2ad962bfc17ebe2f0ba1369bdf cmd/agent-board/client_test.go
```

## Журнал

- 2026-10-18T14:33:47Z — SHA manifest подготовлен; финальные atomicpublish/
  interruption tests завершены последующим запуском. Hashes перед передачей
  перепроверяются; независимого approve разработчик себе не выставляет.
- 2026-10-18T14:35:01Z — финальные race результаты и manifest перепроверены;
  `git diff --check` PASS, передача reviewer/security.
