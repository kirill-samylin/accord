# WORK279 — независимый review восстановления Test_internet

24.10.2026. Исполнитель: отдельный reviewer, read-only runtime/server.
Source: `d2af9c882683da7067056351f72218b03b141659`.
Область записи: только эта карточка. Source/runtime/config/сервер reviewer не менял.

## Предварительный verdict — APPROVE ограниченного плана

План: проверить текущий native executable, отсутствие живых/неопределённых
attempts и pending receipts; штатно остановить только Test_internet LaunchAgent;
сделать private backup; вызвать существующий `worker consent` с прежним config;
запустить прежний LaunchAgent; проверить новый ограниченный read-only managed run.
Это повторное явное локальное согласие, без отключения security checks.

Независимое selective чтение private config/journal подтвердило:

- config mode0600;
- `mr_69cf2ea3e4eec251d2b368cc53e047d0`: failed/generation1,
  diagnostic `prepare_pins`, native session отсутствует;
- pending0, local accepted/launching/running/stopping/needs_recovery runs0;
- provider pin `1000011-54cd664` → current `1000011-5592e74`;
- workspace pin и current совпадают: `1000011-55cc618`;
- automation consent version1.

Токены/host secret не выводились и в карточку не включались.
Подпись/current provider version и факт отсутствия OS-процессов остаются
обязательными проверками координатора перед mutation: journal сам по себе не
является OS process proof.

## Обоснование и ограничения

`execution.go:launch` проверяет pins до durable launch intent, workspace lock,
bridge и native spawn. Именно `prepare_pins` в данном failed attempt означает
отказ до native запуска. Ошибка является ожидаемой защитой при замене inode
бинарника, включая обновление приложения с прежним абсолютным путём.

`paths.go` запрещает symlink и сетевые пути, сравнивает exact path + физическую
identity. Нельзя редактировать journal pins вручную, удалять state или повторно
enroll вместо consent. `consent.go` заново публикует прежний allowlist через
owner-authorized host update и затем сохраняет новые pins; host secret и
согласованные controllers/workspaces/providers сохраняются.

Отдельному CLI consent нужен exclusive state lock (`journal.go`), поэтому
работающий worker необходимо штатно остановить. После положительной idle-проверки
это безопасно. Restart создаёт новый worker incarnation/epoch; это ожидаемо,
не resume прежнего native conversation. Отсутствие pending исключает превращение
старого terminal failed receipt в reconciliation recovered_stopped.

Automation version повышать не требуется, если `automationHostConsent` остаётся
полностью прежним: provider inode не входит в этот public snapshot, workspace
identity не менялась. `managed_hosts.go` требует повышения только при изменении
самого automation snapshot. Host version штатно увеличивается. Менять expiry,
budgets, grants, profiles, presets или policy этим ремонтом не разрешено.
Если consent запросит повышение automation version, следует остановиться и
выяснить расхождение, а не автоматически расширять/переиздавать согласие.

Исходный run следует сохранить failed как честный terminal итог. Server
`ManagedCommand` запрещает resume failed и требует exact native session;
`ManagedStart` допускает новый run, если workspace не занят/reserved. `failed`
не входит в `managedActive`/UI busy. Не нужно отправлять fabricated stop или
`recovered_stopped`, менять server DB/state, либо обещать продолжение разговора,
которого не было. Старый экран останется историей ошибки; новая попытка создаёт
новый ID с тем же согласованным профилем.

## Итоговая проверка

После consent/restart независимый selective readback — PASS:

- current config byte-exact с private backup и SHA256 из preflight;
  mode0600/UID501/GID20, backup root0700/config0600;
- все прежние local run records и receipt records сохранены полностью;
  host ID/secret прежние (secret сравнен только как boolean, не выводился);
- оба path pins совпадают с текущими OS identities, pending0;
  worker incarnation штатно изменилась;
- `server-before.private.json` и `server-after-consent.private.json`:
  host version1→2, worker epoch4→5; automation snapshot полностью равен,
  controllers/workspaces/providers и owner access epoch прежние, active;
- все prior server run records сохранены полностью, включая failed исходный.

Независимая проверка финального smoke/evidence WORK280 — PASS:

- новый ordinary run `mr_f36f593ffcbe7f143bf1706c157a315b`, generation1;
  snapshots показывают running и completed с одним native UUID
  `01a16e3d-629b-7261-b6e0-3c720baf71ff`;
- start applied, last_error отсутствует, output `PIN_REPAIR_LAUNCH_OK`;
  completed receipt относится к тем же run/command/dispatch/generation/UUID;
- native read evidence: AGENTS.md и README.md fulfilled/exit0;
  process audit: после exit descendants0/owned groups0/observed live PIDs0;
- live local journal completed/same UUID/pending0; Test_internet HEAD/status
  полностью равны checkout baseline; automation snapshot прежний/epoch5,
  прежняя server history целиком сохранена;
- exact установленный executable из LaunchAgent plist имеет SHA256
  `f6291d57963620554da09f50f8908ec4b0fe735cc6e04d23aba160016780dadb`.

Review выявил ошибочную атрибуцию другого SHA256 в промежуточном тексте WORK280;
QA исправил её до завершения, финальная карточка согласована с exact executable.
Runtime менять не потребовалось.

Итог: APPROVE ремонта локального consent/pins и одного fresh native launch.
Проверка нового run не подменяет повтор исходного пользовательского задания
или полный workflow. Исходный failed run без UUID корректно сохранён.
Same-UUID resume этого failed run, Windows, Claude execution и второй физический
ПК — NOT-RUN. Reviewer не выполнял server/runtime mutations и новых native запусков.
