# TASK-003 — независимый security recheck

Вердикт по проверенной native service поставке: **APPROVE**. SEC003-01 и
SEC003-02 закрыты проверкой исправленного кода и отдельными fault tests.
Новых блокирующих замечаний в указанной области не обнаружено. Это разрешает
следующий этап изолированной process/E2E приёмки и затем проверяемое
развёртывание; не заменяет результаты Linux/VPS и не означает завершение V1.

Проверяющий `securityrecheck003` не участвовал в реализации. Профиль security
и общий процесс прочитаны; именованные модели Claude в этой среде не закреплены.
База `966dda6`, незакоммиченная замороженная дельта TASK-003 после WORK-025.
Scope: native application, stdio bridge, локальное администрирование и infra.

## Границы доверия и сохранённые меры

Доверены root/оператор, UID accord, владелец клиентского токена. Недоверены
HTTP/MCP JSON, browser Origin, proxy headers и task text. Проверены bearer
проверки перед replay и в транзакциях, привязка session к principal/machine,
монотонность generations, secret redaction и отсутствие удалённых admin
endpoints. Раздельные токены дают identity/revoke; scopes/RBAC/OAuth не заявлены.

Native Origin policy отвергает любой присутствующий Origin до auth, в том
числе GET/null/пустой/multiple header. CrossOriginProtection и SDK-защита
сохранены. Admission использует фиксированный bucket и bounded semaphore,
metrics требуют bearer и не содержат labels с credentials или данными задач.
Общий клиент запрещает remote HTTP, URL credentials, redirect и off-origin
передачу bearer; ответы ограничены с явным overflow error.

Bridge сохраняет единственную собственную сессию, сериализует heartbeat и
изменения lease. Revoke, network timeout, authoritative lease loss и некорректный
ответ ведут в quarantine без повторного получения задачи; диагностические
get/events разрешены. Replay не продлевает локальное владение. EOF прекращает
пульс. Инструкция явно требует прекратить edits при потере lease: bridge
не блокирует самостоятельные shell/file tools физически.

Snapshot — проверенный SQLite VACUUM INTO в private staging, integrity/schema,
fsync, атомарная публикация hardlink без overwrite. Offline recovery в одной
транзакции отзывает восстановленные credentials, закрывает sessions/attempts/
leases, сохраняя fencing counters и аудит. Остановку всех writers обеспечивает
операционная процедура, сама команда reset этого не доказывает.

## Закрытие возвращённых замечаний

**SEC003-01 — PASS.** `recovery.activate` обрабатывает исключения backup,
переключения, start и readiness единым failure path. Перед восстановлением
отменяет возможный restartjob через stop. Прежний executable запускается лишь
при неизменных известной schema1 и полном sqlite_schema. Изменившаяся версия
или даже unversioned DDL, неизвестная совместимость, первый неудачный install
и неудачный fallback оставляют stopped состояние с durable
`START_REQUIRES_INSPECTION`. Unit и повторная установка учитывают этот guard.
БД автоматически не откатывается. В независимом тесте запись, сделанная
failed candidate, сохранилась после восстановления предыдущего executable.
Restore guard остаётся отдельным; код не снимает ни один guard автоматически.

**SEC003-02 — PASS.** Restore синхронизирует marker file и затем каталог STATE
до первой остановки служб и до любого копирования/замены БД. Отказ directory
fsync не вызывает stop и не меняет исходную БД. Это проверка порядка вызовов
и fail-closed обработки; отключение питания на реальном VPS не выполнялось.

## Собственные проверки

На Mac, реальные изолированные SQLite/HTTP/MCP fixtures:

- `go test -race -count=1 -timeout 90s ./internal/client ./internal/bridge
  ./internal/store ./internal/transport ./cmd/agent-board` — PASS: соответственно
  1.528s / 13.057s / 6.456s / 2.702s / 1.720s. Среди проверок malicious Origin,
  redirect/off-origin credentials, revoked snapshot/recovery, interrupted
  snapshot, >2TTL heartbeat, network/revoke quarantine и protocol-only stdout.
- Независимый `infra_recheck.py` — **10/10 PASS** после WORK-025 freeze:
  backup/start/readiness failures, schema version и unversioned DDL guard,
  первый install, preexisting guard, atomic pointer switch, обе marker/order
  проверки. SQLite, marker, fsync и symlink — реальные private tempfile;
  systemctl/runuser/readiness заменены test doubles. Это **не Linux runtime**.
- Compile всех Python helpers без исполнения top-level, `git diff --check` — PASS.

Логи и самостоятельный probe в ignored `.local/evidence/security-003-recheck/`.
Probe SHA256 `2e56b3760a228d9c9f3068cf00edfc92ce862a8ec6ef0fdcb69f9b1f594cb2c1`;
race log `a6eaaf76e74c2dc375bfb325f91ea3339b8b8a062fc43a0c8f03107e6f9a6312`;
infra log `8b98647c54a20ed600ef3f10b6ae9126cf1677ac175b7b3d0aa564b6a73a99fc`.
Постоянные process/E2E tests следующего этапа должен добавить отдельный tester
с последующим независимым ревью; ignored probe не является их заменой.

## Проверенная версия и остаточные границы

Manifest **85 файлов** (весь cmd/internal Go source+tests, infra, dependencies,
AGENTS/README, requirements/architecture/API и service/security/deployment
contracts) сохранён как `.local/evidence/security-003-recheck/reviewed.sha256`.
SHA256 manifest: `d87f40b05ab72b51bf370a843e0a787f7ac3a18762614fcd1b9da62dc60d4c61`.
Динамические TASK/WORK и само заключение в manifest не входят. Изменения
проверенного кода требуют нового адресного review; approve не переносится
автоматически на будущие правки.

Live privateTLS/proxy Host, MainPID/ownership/systemd sandbox, фактические
permissions/ports/timer/backuprestore и сохранность Tandem443 этим проверяющим
**NOT-RUN**. Они обязательны при приёмке развёртывания SEC01–12. Физический
второй ПК, Claude runtime, offsite, GitHub OAuth и полная V1 не зачтены.
Копии только на одном VPS не защищают от потери VPS; закрытые каталоги
не защищают данные от root или доверенного того же UID.
