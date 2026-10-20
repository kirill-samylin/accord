# Мультипроектный Accord: операции

Это оператор обновления существующего TASK008 VPS, а не новый установщик.
Go7892/Next7893/nginx/accord-web-backup уже должны работать. Обязательны
независимый source review, synthetic migration/restore rehearsal и Linux stage.
`docs/multi-project.md` задаёт registry, ACL и неизменные default paths.

## Следующие capable релизы после TASK009

Для уже активированного registry использовать **`upgrade.py`**, а не initial
`operate.py apply`. Аргументы stage те же immutable Go/Next hashes/revision/
nginx hash, без owner-identity. Команды stage/apply/resume/finalize сохраняют
отдельный journal kind=capable-update. Никакого bootstrap-owner повторно.

Перед schema4 writer останавливается под owned inspection marker; создаётся
полный format2 snapshot registry/default/всех проектов. Новый permanent guard
требует `coordination:1,agent_profiles:1,board_schema:[4]` вместе с registry/ACL. Старый TASK009
binary отвергается до запуска. Offline additive migration проверяет прежние
строки всех таблиц и immutable файлы против snapshot до открытия listener.
Исторические migration copies schema1/2/3-64hex.db с checksum сохраняются.
Ошибка означает stop+marker+roll-forward; старые БД назад не накладываются.
После независимой HTTPS приёмки finalize запускает новый полный backup и
повторно сверяет effective systemd argv/guard/backup и owned /proc binary SHA.

Ниже сохранена процедура **первого** TASK009 cutover для истории. Повторять её
для новых релизов запрещено.

## Сборка и stage

Собрать clean reviewed commit прежними `build-release.sh` и Next `package.py`.
Поставить Linux Go, Node/Next bundle и полный checksummed infra каталог
в новый root-owned private staging. Не выполнять build на малом VPS.

```sh
python3 OPERATOR/projects/operate.py stage --revision FULL_COMMIT \
  --bundle NEXT_BUNDLE --manifest-sha256 VERIFIED_NEXT_MANIFEST_SHA \
  --go-binary GO_BINARY --go-sha256 VERIFIED_GO_SHA \
  --expected-nginx-sha256 PRIVATE_INVENTORY_SHA \
  --owner-identity SELECTED_EXISTING_TOKEN_ID
```

Stage проверяет existing TASK008 unit, immutable manifests, Go capabilities,
Linux Node/sharp в hardened transient unit7894, nginx syntax; сохраняет private
журнал `/var/backups/accord-projects/FULL_COMMIT`. Active pointers/data не меняет.
Scoped MCP и upload добавляются узким преобразованием nginx. Прежние native,
Host/Origin/body ограничения и отдельный asset rate budget сохраняются.

## Активация

```sh
python3 OPERATOR/projects/operate.py apply --revision FULL_COMMIT
```

Перед остановкой writer выполняется прежний согласованный board+web backup.
Под существующим maintenance lock writer останавливается; ставятся root-owned
capability helper, постоянный systemd drop-in с явным `--projects-data`, полное
daily backup и защищённый legacy restore. Прежние boot guards сохраняются.
Go/Next pointers переключаются атомарно по отдельности; БД назад не копируются.
Registry мигрирует текущие данные на месте, web1→2 сохраняет browser state.

Под остановленным writer оператор явно назначает default owner через локальный
CLI по выбранному на stage ID. Повтор того же owner при resume идемпотентен;
иной уже существующий owner даёт отказ. После запуска проверяются сохранность
прежних logical records и вход существующим токеном.
Выбирать owner по стабильному существующему token ID; имена/первый вошедший
не являются разрешением. Этот шаг и независимые внешние HTTPS/native/browser
проверки завершаются до finalize:

```sh
python3 OPERATOR/projects/operate.py finalize --revision FULL_COMMIT
```

Finalize требует прежние окружение/units/timers, постоянные project guards,
активные capable pointers и новый формат2 backup всех проектов. Статус
`pending-https-smoke` не означает законченное обновление.

## Проверка фактического режима запуска

Capability binary и HTTP200 сами по себе не доказывают включённую изоляцию.
Собственный `zz-accord-projects.conf` должен действовать после прежнего web.conf.
Оператор проверяет resolved systemd ExecStart/ExecStartPre: точные executable и
argv, обязательный projects-data и guard `--unit accord.service`; отдельно —
полный project backup ExecStart. Permanent ExecStartPre повторяет config check
при каждом старте. После owned readiness сверяются actual /proc/MainPID argv,
ELF SHA и стабильность PID; после backup restart проверки повторяются.
Неизвестный own override не перезаписывается, прежний web.conf сохраняется.

До install/cutover ставится fsynced owned START_REQUIRES_INSPECTION; он блокирует
cold boot через существующий base guard во всём recovery окне. Снять его можно
только после capability/effective config проверки и offline bootstrap перед
защищённым запуском. Unknown/manual marker не удаляется. При ошибке marker
возвращается перед stop; journal остаётся blocked-roll-forward.

## Восстановление после ошибки

После начала активации старый Go не умеет ACL и не допускается. Ошибка apply
останавливает Go/Next, сохраняет все записи и выставляет `blocked-roll-forward`.
Исправить причину, проверить журнал и повторить **тот же проверенный capable**
релиз:

```sh
python3 OPERATOR/projects/operate.py resume --revision FULL_COMMIT
```

Resume не вызывает прежний backup и не восстанавливает БД. Обновление другим
capable релизом требует нового review/migration плана. Удалять защитный drop-in,
отключать projects-data или вызывать старый TASK008 auto-rollback запрещено.
SIGKILL/потеря питания оставляют `activation-started`; после ручной проверки
durable журнала resume принимает это состояние и повторно останавливает оба
процесса перед тем же roll-forward. Это не общая атомарность нескольких SQLite.

Если обычное создание проекта прервалось и исходный browser action утрачен,
остановить единственный writer и явно отвергнуть **выбранный provisioning**:

```sh
runuser -u accord -- CAPABLE_GO_BINARY projects reject-provisioning \
  --projects-data /var/lib/accord/projects-data --db /var/lib/accord/board.db \
  --web-data /var/lib/accord/web --project PENDING_PROJECT_ID \
  --reason provisioning-recovery
```

Private bytes переходят в quarantine, permanent ID tombstone запрещает прежнему
create action возродить этот ID; quota освобождается. Active/default отклоняются.
Повтор завершает тот же отказ. Опубликованные данные, credentials и leases не
сбрасываются. Проверить новый полный backup и только затем вернуть writer.

## Полная резервная копия

Daily `accord-web-backup.service` использует `admin/projects/backup.py`, общий
maintenance lock и остановку единственного writer. Формат2 включает registry,
default, все published projects и unpublished provisioning recovery sets;
immutable blobs/staging, schemas, checksums, private modes, fsync и atomic
publication проверяются до retention7. Неизвестные файлы, orphan каталоги,
неразрешённые SQLite journals и недостаток ≥512MiB reserve дают отказ.

Snapshot остаётся на VPS; внешнее хранилище этим этапом не настроено. Старый
hourly board backup допустим как дополнительная копия, не полное восстановление.
Старый single-board restore отказывается изменять multi-project state.
Старые format1 snapshots остаются для истории и не удаляются format2 retention.
Restore разрешён только в новый offline staging с отзывом всех credentials,
browser/native sessions/leases до любого listener; исходный snapshot неизменен.

```sh
python3 OPERATOR/projects/restore.py --snapshot FORMAT2_SNAPSHOT \
  --output NEW_PRIVATE_OFFLINE_STAGING --binary CAPABLE_GO_BINARY
```

`projects maintenance-reset` перепривязывает default paths к новому staging,
отзывает credentials, web/native sessions/actions/leases во всех проектах;
unpublished provisioning переходит в quarantine с permanent reject tombstone.
Boot guard остаётся и после auth reset. Новые credentials, проверка логических
данных и isolated runtime обязательны до снятия guard/публикации.
Этот helper не заменяет production paths и не запускает сеть.
Фактическая процедура и synthetic evidence фиксируются в WORK138/TASK009.

TASK011: schema4 содержит обязательную coordination_metadata(1,1). Snapshot/restore
принимает полные board2/3/4 и сохраняет исторические migration snapshots schema1/2/3
побайтно с checksum. Старые capable schema3 binary новым guard отвергаются до DDL.
