# TASK-009 — мультипроектность Accord

20.10.2026. Прямое поручение: полноценные проекты, подключение репозиториев,
переключение, изоляция данных и прав; текущий проект и данные сохранить.
База `64e534b`, установленный TASK008 release `490bf55`.
Ветка `codex/multi-project`. Статус: source/tests APPROVE; local+Linux runtime PASS; VPS остановлен с durable boot guard; исправляется effective-launch P1.
Контракт [multi-project.md](multi-project.md), независимое направление APPROVE136.

## Результат для пользователя

Проект — отдельная область задач, сессий, очередей, заметок, событий, решений,
дизайна, конфигурации и доступа. Репозиторий привязывается к проекту безопасной
Git-ссылкой; проекты можно создавать, открывать, архивировать и восстанавливать.
Переключатель показывает только разрешённые проекты. Управляющий назначает,
меняет и отзывает права конкретных участников по стабильным идентификаторам.
Одинаковые имена и номера T-1 в разных проектах не объединяют данные и права.

Текущие board.db/web/blobs, токены, сессии, leases, generations, evidence и
durable actions сохраняются; legacy адреса остаются закреплены за default.
Новая выбранная вкладка не меняет смысл уже открытой формы, фонового запроса,
выданного action ID или native/MCP клиента другой вкладки.

Ссылка на репозиторий означает сохранённую связь. Удалённый clone/проверка Git
доступа и выполнение кода принадлежат последующему worker/execution этапу;
интерфейс не выдаёт сохранение URL за проверенные credentials или clone.
OAuth/GitHub-вход не подменяет ранее выбранную авторизацию отдельными токенами.

## Контракт и обязательные границы

- Go остаётся владельцем auth/domain/storage, Next — клиентом; зависимости не
  обновляются без причины. Registry и изолированные SQLite/web stores —
  исходное архитектурное направление; точный контракт после WORK135/136.
- Stable opaque project ID, явный scope в адресах; имя/URL/path не являются ACL.
- Credential identity, project membership и board principal различаются.
  Старые owner/machine строки не объединяются в аккаунт и не дают admin сами.
  Новый B-only credential не получает доступа к default.
- Проверки expiry/revoke/членства/роли выполняются на сервере до данных, replay
  и записи. Учитываются viewer, last owner, архив, текущие browser/native сессии.
- Смена проекта изолирует cache, drafts, filters/cursors, commands, polling,
  files и retries. Поздний ответ A не меняет B; неизвестный результат сохраняет
  исходный запрос A. Авторизация не заменяется скрытыми кнопками.
- Транзакции сохраняют fencing, leases, independent SHA/generation gates,
  CSRF/Origin, idempotency и ограничения файлов. HTTP upload/stream не держит
  глобальную write-блокировку registry120s.
- Crash-consistent provisioning/publish/recovery; не обещать общей атомарности
  нескольких SQLite WAL файлов. Backup охватывает registry и все проекты.
  Downgrade на старый binary не должен обходить новые ACL; данные не откатывают
  из snapshot поверх новых пользовательских записей.
- Реальные данные/токены только в ignored runtime или на VPS. Tandem не трогать.

## Владение и порядок

Root ведёт эту TASK, канонические requirements/API/ADR, Git, интеграцию,
операционную поставку и передачу следующего этапа. WORK135 сначала проектирует
backend/migration, WORK136 независимо критикует и проверяет, WORK137 готовит
Next UX/cache/API. До принятого контракта продуктовый код не меняется.

После согласования root явно распределяет непересекающиеся области реализации.
Unit/integration пишет разработчик. Новые process/E2E пишет отдельный tester
после независимого APPROVE исходников; новые тесты также проходят review.

Выданы GO: WORK135 — Go backend/CLI/bridge/авторские Go unit/integration;
WORK137 — `web/src/**` и клиентские unit; root — `infra/accord/**`, связанные
infra unit, канонические docs/Git/integration/операционные действия.
WORK136 проверяет независимо и не меняет source. Общая TASK — только root.
WORK138 — root infra/backup/restore/оператор и авторские unit. WORK139 — отдельный
tester; пока read-only план, новые process/E2E только после product APPROVE136.

Уточнения принятого контракта: self-service create-own-project без globaladmin,
явные начальные квоты, lazy materialization только прежнего legacy права;
global cookie expiry ограничена старой cookie/token, tombstone раньше fallback
удаления; deep link требует явного POST для отсутствующего project binding.
Revoke→regrant использует access_epoch и не оживляет старые session/action IDs.
Downgrade сохраняет возможное human чтение, запрещая writes и cached replay.
Describe namespace при epoch>0 использует приватный HMAC, не предсказуемый hash.

## Приёмка

1. Два и более реальных synthetic проекта с репозиториями; CRUD связей,
   переключение, reload/back/deep links, две вкладки, сохранность default.
2. Owner/editor/viewer и A-only/B-only/shared identities; revoke/downgrade
   немедленно закрывает старые сессии/actions. Последний owner защищён.
3. Отрицательные UI/REST/MCP/resources/file тесты на чужие IDs, keys, sessions,
   generations, actions, CSRF, cursors и assets; без утечки данных/существования.
4. Durable retry, races, cancel/late response, archive/restart и crash recovery.
5. Полный backup/restore/migration rehearsal на synthetic fixture; safe rollback
   либо явный fail-closed downgrade. Один Go writer, старые данные не стираются.
6. Независимые source/tests/evidence review и фактический Go/Next/Chrome QA.
   Перед VPS — отдельные проверенные manifests, snapshot и migration guards.
7. Фактическая версия, проверки PASS/FAIL/NOT-RUN, ограничения и clean checkpoint.

## Разрешённая очередь следующих чатов

Пользователь прямо поручил после TASK009 создавать новый чат и выполнять дальше:

1. Постоянные профили агентов.
2. Координация.
3. Управляемый запуск.

Новый чат создаётся после фактического завершения текущего этапа, получает
проверенный checkpoint, owner/границы/приёмку, ссылки на канонические документы
и эту прямую авторизацию последовательной передачи. Незавершённый TASK009
не переименовывается в следующий этап. Детальные контракты профилей/execution
формируются в своих этапах до реализации; текущая задача их не имитирует.

## Фактические проверки

Product source и авторские unit/integration — независимый APPROVE136.
Frontend: format/typecheck,53 unit и Next production build PASS. Go: fmt/vet,
все tests, race и build PASS. Root infra:31 новых/115 всего unit PASS;
mock orchestration не считается Linux runtime. Полный `make check` EXIT0.

Актуальный Go amendment после `42f02c7`:215 файлов, aggregate SHA256
`20b896df6eaee22e39410b2ca54eec35a5e9ffe1f720afb38145ee0904662cf5`.
Он не допускает пересоздание утраченного registry authority или пустого web
metadata опубликованного проекта: preflight идёт до Store/web constructors,
данные и blobs остаются на месте. Проверенный macOS binary SHA256
`3946080cacf567a8dc2456cfe1f240e05d42846f4cac72993766c4b77cabd396`.
Root повторный fmt/vet/allGo/115infra/race/build EXIT0.

Отдельный QA139 actual run3:25/25 PASS. Реальный старый binary490bf55 создал
synthetic web1/tasks/cookie/live lease/blob; новый сохранил строки и bytes при
web2 migration. Проверены A-only/B-only/shared/viewer, scoped REST/MCP/files,
revoke/regrant epochs, concurrent durable replay, прерванный upload, архив,
restart, startup authority-loss отказ, full snapshot/NEW offline restore/reset,
pending quarantine и запрет старого binary с marker/без marker.
Chrome проверил две вкладки, cache/drafts/back/deep explicit open, поздний ответ,
реально committed lost response и точный retry,1MiB upload через Next/SHA,
viewer UI, mobile/light/dark;6 screenshots/315 responses, page errors0.
Final evidence private `.local/task009-qa/run-1792500343500/results.json`, SHA256
`9452466964fdbb014480f503db0de0165757434cdeaf6a381e78690f9c18adc1`.
Chrome155.0.8059.85/Node24.19.0/Go1.27.2; дополнительный process-only16/16
PASS. Final test freeze15 code files+README aggregate
`e9df458b0733cc430b1b021d7d3cf5748f1d822d3826cb08c35a31a3b8307590`;
после full run только platform-aware metadata text, не сценарии. Owned ports
3100/3101 FREE, дочерние Go/Next/MCP/Chrome завершены.
Next QA BuildID `_BtTvE7AHru0IDFWVC3t0`. Final test/evidence APPROVE136;
стабильный контраст dark buttons подтверждён, screenshot transition исправлен.

## Поставка на VPS — текущий checkpoint

Clean release `5454eb5c98ac088e5ead3daaaad637423112ddad`; Go source215 не менялся.
Linux/amd64 binary SHA256
`6950b220cd60b9abf3d295f114c60512ae40334480302c8fcf742ab47d61e0a8`,
Go1.27.2/buildvcs clean; Next BuildID `YYrHpN-gOyr560HLMZaUM`, upstream7892,
manifest SHA256 `9f3e3e3d3e00cf713b6aa7d58cbb04f1287cfd7950e5e04a880e25fe6554124f`.
Immutable transfer:47 Go/infra и1256 Next files verified, active не менялся.

На VPS точный reviewed process-only script:16/16 PASS, Linux x64/Node24.19.0,
actual old490bf55/new5454 binaries, generated private fixture. HTTP3100/3101
после cleanup без listeners; plain bind initially встретил только TCP TIME_WAIT,
ss и SO_REUSEADDR bind подтвердили свободные порты. Production paths не трогал.
Linux hardened Node/sharp/nginx stage PASS, owned7894, NRestarts0, peak≈47MB;
transient unit удалён, старые active pointers сохранены до apply.

Прежний apply завершился без ошибки, но effective-mode acceptance далее FAIL: до cutover создан format1 snapshot текущих данных, затем
Go/Next остановлены, permanent capability/backup guards установлены, явно
назначен выбранный существующий default owner, registry включён, web1→2.
Обе active pointers и реальные `/proc/MainPID/exe` hashes соответствуют release;
Node cwd — новый immutable bundle. Owned7892/7893 PASS, NRestarts0.
Сравнение с непосредственно предшествующим verified snapshot: все старые строки
9 board tables, design rows и blob bytes сохранены, физические default paths
не менялись. На VPS опубликован прежний единственный проект; synthetic A/B
не добавлялись в реальные данные. Private evidence `.local/task009/deploy/`.

Состояние оператора `pending-https-smoke`. Первый generic production inspection
подтвердил HTTPS/root/unauthorized/private-metrics и успешный вход, но сам harness
скрыл исходную причину последующей ошибки и завершился unhandled cleanup wait.
Этот run **FAIL**, итогового report нет; business mutations не выполнялись,
headless Chrome не остался. QA139 исправляет только harness diagnostics/wait
ownership; новый review обязателен до повтора. Продуктовые Go/Next frozen.
Полный HTTPS11screens/native/MCP inspection и finalize с actual format2 backup
остаются обязательными до закрытия. Физический второй ПК NOT-RUN;
сохранённая Git-ссылка не означает clone или проверенный удалённый доступ.

### Effective-launch P1 и безопасное исправление

Второй HTTPS run корректно зафиксировал FAIL LOGIN_REDIRECT; expected scoped
redirect отсутствовал. Root установил фактическую причину: `web.conf` сортируется
после `90-projects.conf` и возвращает прежний ExecStart без `--projects-data`.
Binary5454 был новым, но HTTP работал в legacy mode. Прежние PID/hash/readiness
проверки НЕ доказывают ACL activation; production multiproject PASS отменён.
Registry bootstrap существовал, но новые проекты в production не создавались.

Root вызвал reviewed fail_closed под admin+maintenance locks: Go/Next остановлены,
phase blocked-roll-forward. Затем durable START_REQUIRES_INSPECTION0600+fsync
блокирует существующий cold-boot guard; реальные DB не менялись и не откатывались.
Исправляются только root infra и связанные unit: zz override, exact effective
ExecStart/ExecStartPre executable+argv, backup ExecStart, actual owned /proc argv
и binary hash; permanent guard проверяет effective launch перед каждым стартом.
Owned marker сохраняется во время recovery/install и снимается только после
всех effective/capability проверок под locks, перед guarded startup; неизвестный
manual marker не удаляется. Независимый review обязателен до resume.

5454 immutable app release не переписывается: новые reviewed operators получат
отдельный immutable release. Однократный audited offline journal amendment
меняет только helpers+audit, сохраняет revision/binary/bundle/preserved/owner;
original journal O_EXCL0600+fsync, guards, stopped MainPID0/portsfree обязательны.
После resume необходим повтор полного HTTPS/native/MCP inspection и finalize
format2 backup. Go/Next product source не меняется.
