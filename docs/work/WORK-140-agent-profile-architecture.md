# WORK-140 — независимый архитектурный разбор постоянных профилей

20.10.2026. Исполнитель: отдельный архитектурный reviewer TASK-010.
Область записи: только этот WORK. Product source, общая TASK, runtime и VPS не
изменялись. Рассмотрены AGENTS, requirements, architecture, API, multi-project,
control-web-architecture, TASK009 и реальные Store/session/project/web/backup
цепочки. Это архитектурное заключение; source review и фактическая приёмка впереди.

## Вывод

Канонический `docs/agent-profiles.md` сверен после создания координатором.
Предложенный ограниченный контракт пригоден для реализации при условиях ниже.
Самостоятельная project-local сущность, immutable revisions,
явный native Describe по ID+version и существующий Next UI дают фактические
профили без обещания worker execution. Архитектурных блокеров продукта после
фиксации этих условий нет. Поставка остаётся отдельным обязательным барьером:
TASK009 initial apply повторно использовать нельзя.

## Рекомендуемый контракт

- `AgentProfile`: immutable случайный `ap_<32hex>`; project определяется Store,
  не browser supplied path. Названия допускают совпадения и не связывают identities.
- `name` — непустой, до120 Unicode codepoints; `description` — до4KiB UTF-8;
  `instructions` — plain text до16KiB UTF-8. Не HTML, не executable template.
- `role` — coordinator/analyst/architect/developer/reviewer/tester; human исключён.
  Профиль не получает права credential, токен, аренду или singleton generation.
- `provider` — unconfigured/openai/anthropic/local/custom; `model` — до200
  безопасных символов. Это предпочтение, сохраняемое и читаемое как данные.
  Значение custom не разрешает URL, executable или передачу arbitrary settings.
- Запрещены поля baseURL, env, command, filesystem cwd, credentials/secret refs,
  arbitrary JSON, tool permissions, budgets и launch configuration. Их принятие
  потребовало бы следующего execution/security-контракта. Сами instructions не
  являются secret storage: UI должен сообщать, что данные видят участники проекта.
- `state=active|archived`, monotonically increasing `version>=1`, timestamps и
  server-owned provenance. Создание, редактирование, архив и восстановление
  фиксируют неизменную revision и audit в той же транзакции с durable response.
- CRUD здесь означает create/read/update и soft-delete=archive. Hard-delete не
  нужен: он разрушал бы исторические связи. Архив сохраняет данные/сессии/историю.
- Не более128 профилей на проект, включая архив. Нужен также явный предел роста
  revisions, например1024 на профиль: при достижении409, без скрытого удаления.
  Иначе quota128 не ограничивает неограниченную историю instructions.
- Список всех до128 записей допустим без cursor. History read должен быть bounded
  и адресовать profile ID+version; нельзя возвращать всю потенциальную историю
  с полными инструкциями в каждом list/poll ответе.

Owner/editor активного проекта управляют профилями; viewer только читает.
Архивный проект сохраняет чтение, запрещает профильные мутации и новые profile
sessions. Роль human browser session может управлять профилем при project write
ACL; редактирование профиля не даёт этой сессии native role профиля.

Raw credential/local principal ID не следует добавлять в browser DTO:
`internal/web/browser_page.go` специально исключает token IDs. Допустимы
server-generated actor session ID и человекочитаемые owner/machine подписи;
внутренний principal provenance при необходимости хранится отдельно от public view.

## Явное использование и сохранение старых сессий

`Describe` сохраняет обязательные request_id, role, run_id, machine и добавляет
пару `profile_id`+`profile_version`. Без пары остаётся прежний unbound сценарий.
Нельзя автоматически искать профиль по role, owner, run_id, machine или имени.
Partial pair и version<=0 —400; unknown/чужой ID —404; устаревшая версия либо
archived profile при NEW Describe —409. Role должен строго совпасть с revision.

Это условие обязательно: нынешний `projects.service.Describe` проверяет viewer/
project-state по `in.Role` до вызова Store. Если принять role=human, а затем
молча заменить его ролью профиля, существующий ACL можно обойти. Сохранение
обязательной роли и строгого match безопаснее расширения двухступенчатого lookup.

Profile snapshot (либо неизменная revision reference с явной public projection)
записывается атомарно с новой server-generated сессией. Он включает реально
выбранные version/role/instructions/provider/model; ответ Describe отдаёт эти
данные клиенту. Update/archive/restore профиля не переписывает уже открытые
сессии, attempts, evidence, leases и роль. Предпочтительно full snapshot в
SessionView плюс revisions для истории, чтобы heartbeat/catalog не зависели от
неизвестного текущего state профиля.

Текущий access guard применяется до любого replay. Exact replay ранее успешного
Describe возвращает прежнюю сессию/snapshot и после edit/archive профиля: это
восстановление ответа, не создание нового запуска. Replay не продлевает TTL.
Смена version с прежним request_id даёт idempotency conflict. Новый request_id
подчиняется текущим profile state/version и singleton правилам. Revoke→regrant
сохраняет TASK009 session epochs и HMAC Describe namespace.

Несколько credentials могут явно открыть сессии одного профиля; профиль не
singleton identity. Только native architect role остаётся singleton. Независимое
review по-прежнему определяют session/SHA/generation, не profile ID. Один профиль
может участвовать в разных независимых сессиях; это не доказательство разных
людей, моделей или процессов. Старые сессии сохраняются unbound без backfill.

## Хранение и wire boundaries

Рекомендуется board schema3: profiles и immutable profile revisions внутри
каждого project board.db. Они используют существующий auth→registry guard→Store
transaction/idempotency путь. Profile mutation не получает обход общего ACL
через новый service interface, direct REST, MCP resource, CLI или browser action.
Project-local opaque ID не заменяет membership. Read не открывает/не renew сессию.

Schema2→3 — транзакционное добавление таблиц и PRAGMA user_version без переписи
старых rows. Проверять исходную schema2/required tables до DDL. Для уже schema3
отсутствие profile таблицы/колонки/metadata — failclosed, без CREATE IF NOT EXISTS
repair. Unknown future schema — отказ. Проверки запуска до default constructors и
lazy managed runtime должны принимать валидные2/3 и проверять3 полностью.
Fresh provisioning создаёт3 до publish-active. Registry1/web2 не нужно менять.

Нельзя случайно добавить `board.db.migrations` в published project folder, который
сейчас имеет строгую allowlist: полный snapshot после этого будет отвергнут.
Координатор принял сохранение durable per-board migration snapshots с точным
именем `schema{1,2}-<64hex>.db` и `.sha256`: существующий Store identifier()
генерирует32 random bytes, то есть64hex. Root расширяет только соответствующие
strict allowlists, private-mode/checksum validation, copy/manifest/restore и
synthetic fixtures. Snapshots не удаляются. Такое размещение принимается при
полном сохранении исторических migration sets. Full all-project snapshot до
production migration всё равно обязателен; локальный файл его не заменяет.

REST read/get/revision/mutation и CLI можно добавить отдельными DTO. В MCP
предпочтителен read resource для profiles и расширение существующего describe;
CRUD не должен молча нарушать текущий стабильный лимит12 tools. Если добавляется
новый tool, это требует явной корректировки принятого контракта, а не случайного
увеличения. Unknown/duplicate fields и bounded request остаются обязательными.

Browser mutations требуют того же Origin/CSRF/server-issued action_id, exact body,
original session/project path и recovery после uncertain response. Cached success
проверяет current write ACL и epoch. Profile mutation должна попасть в список
recoverActions до создания новой browser board session; иначе lost response
останется невосстановленным или повторится с новым ID.

## UI

В разделе агентов нужны различимые «Профили» и «Сессии». Профили читаются из
нового storage, имеют add/edit/archive/restore, страницу инструкций/настроек и
версию. Действующие/истёкшие/отозванные session statuses не становятся states
профиля. Session detail показывает исходный profile ID/version и snapshot либо
«Без профиля». Имена не используются для связи.

Карточка профиля не показывает «Запущен» по факту его существования или живой
board session. Provider/model подписаны сохранённым предпочтением; фактический
процесс/подключение/расходы/кнопка запуска отсутствуют до следующих контрактов.
Допустим пример native Describe с ID/version/role без bearer; это использование
профиля в API, а не интеграция с настоящим Claude Code/Codex.

QueryClient keys, placeholders, drafts, form mount keys, pending/retry state и
captured mutation URL включают проект и profile ID. Ответ версии1 не должен
перезаписать dirty draft версии2. На conflict пользователь видит сохранённый
draft и новое состояние; новый action берётся только после известного исхода.
Viewer/archived project не получают рабочие write forms, прямой POST всё равно
отклоняет сервер. Полные instructions не нужны в общих lists/command palette.

## Блокеры безопасной поставки

1. Отдельный capable updater вместо TASK009 initial apply. Требуются полный
   snapshot registry/default/всех published/pending/quarantine, сохранение ACL,
   FS modes, owned inspection marker и roll-forward без data rollback.
2. Capability/independent guard должны проверять поддержку profiles/schema3,
   а также прежние registry/ACL и resolved systemd executable/argv/backup. Старый
   TASK009 binary сам отвергает3 в preflight/migrate, но предварительный guard
   обязан отклонять его до начала запуска над мигрированным состоянием.
3. Snapshot copy/validate/restore принимают и полностью проверяют board3; текущие
   layout.py/validate.py жёстко ожидают2. После3 старый default-only backup и
   schema2-only helper запрещены. All-project restore только в новое staging;
   maintenance reset сохраняет profiles/revisions, отзывает sessions/credentials.
4. Неизменные source/bundle/Go/Next hashes, synthetic migration/update/recovery
   rehearsal и actual Linux evidence до production. При failed cutover сохранять
   owned marker, останавливать writers, не заменять production DB fixture.

Проверять уже установленный zz override и реальные /proc PID/argv/ELF SHA. Любой
manual/unknown inspection marker остаётся на месте. Повтор bootstrap-owner и
повтор одноразового TASK009 helper amendment не относятся к новому обновлению.

## Проверки после реализации

Авторские unit/integration: immutable revisions/version conflicts/idempotent
create/edit/archive/restore; field bounds; mismatch role/human bypass; missing/
foreign IDs; archive new bind vs exact replay; restart; old session snapshots;
same-name nonmerge; profile edits не меняют role/singleton/leases/gates; registry
epoch revoke→regrant/downgrade/cached replay; published missing3 tables failclosed;
schema2 preservation; backup/restore3 с сохранением всех профилей и старых данных.

После независимого source APPROVE отдельный tester пишет process/browser:
два проекта и совпадающие имена; viewer direct POST; две вкладки; dirty draft;
late A response; действительно committed lost response/exact retry; archived
project/profile; narrow390/wide1440/light/dark; profile session и unbound legacy
session; synthetic Linux updater success/interrupt/resume/old binary denial.
Новые тесты проходят независимый review. Production HTTPS и actual full backup
обязательны отдельно от local mocks. Physical second PC остаётся NOT-RUN.

## Первичные источники и статус

Проверено20.10.2026: [SQLite transactions](https://www.sqlite.org/lang_transaction.html)
подтверждает один write transaction одновременно и отсутствие вложенного BEGIN;
[PRAGMA](https://www.sqlite.org/pragma.html#pragma_user_version) описывает
application-owned user_version; [ALTER TABLE](https://www.sqlite.org/lang_altertable.html)
задаёт ограничения schema changes. Новые зависимости/SDK не требуются.

## Dispositions канонического контракта

Приняты: owner/editor CRUD, immutable revisions и cap1024, public provenance через
session ID, native strict role match, provider/model только данные, soft archive,
schema3 и новый capability floor. История максимум1024 metadata-only entries и
explicit full revision read соответствуют bounded read требованию.

Остаётся уточнение wire до source review: текущий API ограничивает MCP12 tools;
перечень новых native operations в каноне сам по себе не разрешает13-й tool.
Рекомендуемый выбор — read resources profiles/profile, расширенный Describe,
CRUD в web/REST/CLI. Альтернатива — явно согласованное изменение прежнего лимита
в requirements/API. Это не блокирует остальные авторские области, но должно
быть решено до приёмки transport implementation.

Архитектурный review: APPROVE выбранной модели и перечисленных инвариантов;
MCP wire clarification остаётся открытым disposition. Implementation/source/
process/browser/VPS: NOT-RUN этим исполнителем. Профили не объявляются готовыми
по данному архитектурному заключению.
