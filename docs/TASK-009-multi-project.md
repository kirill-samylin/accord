# TASK-009 — мультипроектность Accord

20.10.2026. Прямое поручение: полноценные проекты, подключение репозиториев,
переключение, изоляция данных и прав; текущий проект и данные сохранить.
База `64e534b`, установленный TASK008 release `490bf55`.
Ветка `codex/multi-project`. Статус: SOURCE APPROVE, local runtime PASS; final test review и VPS впереди.
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
Next QA BuildID `_BtTvE7AHru0IDFWVC3t0`. Final test/evidence review идёт;
стабильный контраст dark buttons подтверждён, screenshot transition исправлен.

VPS TASK009 пока NOT-RUN; только read-only inventory490bf55, Go/Next200,
board2/web1, projects-data отсутствует. Linux release/stage, фактическая
миграция production, внешний HTTPS/native/browser smoke и полный backup
остаются обязательными до закрытия. Физический второй ПК NOT-RUN;
сохранённая Git-ссылка не означает clone или проверенный удалённый доступ.
