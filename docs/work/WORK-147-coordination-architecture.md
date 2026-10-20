# WORK-147 — независимая архитектура координации

20.10.2026. TASK011, независимый архитектор. Ветка `codex/coordination`,
исследован исходный checkpoint `343b93c71297c58c972f73fed4d1d4ca3a349740`.
Область записи: только этот WORK. Production, Tandem и workers не запускались.
Статус: архитектурная рекомендация; не source APPROVE и не runtime PASS.

## Основания и фактическое состояние

Прочитаны AGENTS, requirements, architecture, API, control-web-architecture,
multi-project, agent-profiles, TASK009/010 и релевантные разделы неизменного ТЗ.
Проверены core types/control/profiles/catalog; store tasks/dependencies/gates/
transitions/profiles/mutation/persistence/migrations/schema validation; transport
operations/MCP; browser control pages/actions/profiles; Next task create/edit/
detail/properties. Новые внешние API и зависимости не предлагаются.

- REQ06/09/12/14–24 уже реализуют pull-координацию: роль, атомарный `next`,
  аренда/fencing, зависимости, попытки, точный commit/generation, append-only
  события и durable replay. Второй scheduler создаст расходящиеся правила.
- `Task`/`CreateInput`/`Next` не содержат назначения профилю. Любая сессия той же
  роли вправе забрать задачу; сохранённые profiles пока не маршрутизируют работу.
- Next browser реально создаёт задачи/эпики, редактирует текст и работает с
  human-вопросами. Зависимости показаны, но browser не создаёт/удаляет их;
  agent lifecycle не имеет полноценного координаторского управления.
- `blocked` — bool без объяснения. Owner текст задачи не является profile ID,
  credential или доказательством владения; активную работу определяет lease.
- Старый binary load/save целиком сериализует Task: неизвестные новые JSON поля
  будут потеряны при следующей записи. Даже JSON-only расширение требует floor.

## Рекомендуемый минимально полноценный объём

1. Явное назначение существующей задачи project-local профилю или общей очереди
   роли. Опциональное такое же назначение при create, чтобы между create и assign
   другая сессия не успела захватить работу.
2. Один серверный predicate пригодности к `next` с объяснением причин: состояние,
   эпик, dependencies, review target, профиль, совпадение session snapshot и роль.
   Browser, REST/CLI и MCP используют те же правила; read-view ничего не claims.
3. Browser управление назначением, входящими зависимостями, отменой/возвратом
   ready задачи и эпиком. Для done→ready обязательны прежние reason/evidence_ref.
   Создание следующей review/test задачи — явная форма с target SHA, затем
   связь reviewer→tester с approved. Никакого автоназначения review каждому commit.
4. Показать профиль/версию отдельно от текущего lease-holder, блокирующие задачи
   и конкретные причины недоступности, target SHA/generation и актуальность
   проверок. Переход по blocker/target открывает задачу в том же проекте.
5. Проектный обзор координации — bounded/paginated read model или расширение
   существующего списка с деталями. Полный граф всех проектов не загружать.

Это завершённый ручной контур подготовки, передачи и проверки работы; не
планировщик, provider adapter, process controller, файловый lock или Paperclip
feature parity. Native developer/reviewer/tester по-прежнему сами `describe/next`.
Browser не открывает за человека agent-role session и не завершает чужую работу.

## Назначение: варианты и рекомендация

| Вариант | Польза | Цена/риск |
| --- | --- | --- |
| Только stable ID + роль | Изменение инструкций не оставляет pending задачу ждать переназначения | Старый snapshot-сотрудник забирает работу после изменения инструкций; выбор реальной версии неявен |
| Stable ID + pinned revision | Проверяемый выбранный набор инструкций, точный match сессии | После edit/archive pending задачи требуют явного refresh/reassign; старая revision не открывается новой describe |
| Session ID | Конкретный lease-independent адресат | Связывает план с TTL и машиной; ломает восстановление, профиль превращается в identity |

Рекомендую pinned revision, которую координатор явно видит и может обновить.
Это существенное продуктовое допущение: до реализации root должен обозначить
его пользователю и учесть полученное уточнение. Session ID не использовать.

Предлагаемый `Task.assignment` — optional `{profile_id, profile_version,
profile_name, role}`; имя и роль сохранены при назначении, не берутся из текущего
профиля для исторического отображения. Инструкции не дублируются в каждой задаче:
immutable profile revision остаётся источником. Оригинальный Task.Role неизменен.

При назначении профиль существует в том же проекте, active, с точной текущей
version и совпадающей task.role. Epic/human задачам профиль не назначается.
Отсутствие assignment сохраняет прежнюю общую очередь роли, включая старые tasks.
Ролевой общий пул доступен bound и unbound сессиям. Assigned task доступна только
bound сессии с точным ID/version/role, а current profile всё ещё active и той же
версии/роли. Current edit/archive блокирует **новые** claims до явного решения;
existing lease/heartbeat/release/completion сохраняются без переписывания snapshot.

Переназначение/снятие: только ready/created, unleased, generation=0; expected
Task.Version обязателен, exact mismatch→409. Live lease нельзя обойти даже
координатору. Done/cancelled сохраняют назначение как историю; сначала обычный
разрешённый возврат в ready, затем reassignment. Архив/изменение не сбрасывают
attempts, evidence, task generation и audit. Требуется явное reason до4КиБ.

Assignment не даёт ACL, singleton или reviewer independence. Несколько
сессий/машин с одним profile могут конкурировать, но только одна получает lease.
Одноимённые profiles не считаются одинаковыми. Новая версия с другой ролью
требует другой подходящий профиль или отдельную новую задачу, не смену Task.Role.

## API и полномочия

Предлагаемый native `assign` (REST/CLI-only): request_id, task_key, version,
profile_id/profile_version (оба отсутствуют для общей очереди), reason.
Явный clear лучше отдельного mode, чтобы неполный pair не стал clear.
`create` получает optional pair; вся проверка и создание одной транзакцией.
MCP budget12 сохраняется: create/next уже покрывают создание+claim, assign можно
либо оставить REST/CLI/browser, либо позже включить в существующий update с
жёстким отдельным validation. Не добавлять 13-й tool молча.

Read `coordination`/task detail: авторизованный actor, paginated task projection,
fixed creation boundary/filters и bounded blockers. Рекомендуемые reason codes:
`epic_not_open`, `dependency_unsatisfied`, `target_not_done`, `target_sha_changed`,
`target_generation_changed`, `assignment_archived`, `assignment_changed`,
`profile_mismatch`, `role_mismatch`, `lease_held`, `task_not_ready`.
Actor-specific reasons отдельно от общих gates: human browser видит routing
статус, но не получает бессмысленное «роль human не developer» как блок проекта.
`blocked` прежней семантики dependency gate не смешивать с assignment eligibility.

Нельзя поддерживать независимую копию gate logic в read model: общий predicate
или proof-producing evaluator. Empty next и queue counters должны использовать
одинаковую assignment пригодность. Историческая общая role queue в stats может
оставаться общей, если DTO/подпись явно различают её от session claimable queue.

ACL: owner/editor активного проекта могут подготовить план через живую human
или coordinator session; viewer только read. В single-board режиме явно
перечислить те же role gates. Native developer может создать дочерний запрос
существующими правами; расширение assign не должно дать снять чужую аренду.
Сохраняются current auth+ACL+access_epoch до replay, scoped opaque IDs, archived
project read-only, strict unknown/duplicate/null input, прежний64КиБ body limit.

Browser actions: issued project/epoch-bound capability, CSRF, durable payload,
исходный project redirect, retry **той же** action при unknown result. Для нового
version conflict draft сохраняется; обновлённый объект принимается только явно.
Pending recovery должен знать каждый новый operation. Поздний ответ A не меняет
project B, draft или selection; current task expected version не перезаписывать
поллингом в открытой форме. Viewer не получает mutation capabilities.

Dependencies/lifecycle используют existing Link/Transition и их правила,
не прямую запись state/links из browser. Координаторская форма не обходит
requester ограничения: требуется явное разрешённое human-role правило для
административных операций owner/editor с живой human session. Это узкое
расширение role gates, не универсальное право имитировать holder.

## Миграция и эксплуатация

Рекомендую board schema4 + capability `coordination:1`, registry1/web2 прежние.
Schema bump необходим даже без новых таблиц, чтобы schema3 binary отказал до
любого чтения/перезаписи Task JSON. Marker/capability floor ставится до DDL
через reviewed capable updater. Оператор должен поддержать источники2/3→4 и
all-project preservation, snapshots/restore schema4 и все исторические schema1/2/3
migration copies; schema3 snapshot до DDL обязателен. Не переиспользовать initial
TASK009 apply и не откатывать БД. Migration не назначает profiles старым задачам
или сессиям и не редактирует old JSON rows без необходимости.

Source runtime preflight, offline migrate-existing, backup validation,
provisioning/restore, guard, capabilities, upgrade stage/apply/resume/finalize и
unit fixtures обновляются согласованно. Authority marker остаётся service-UID,
regular0600; helper root-owned policy не применяется к нему. Незнакомые recovery
artifacts не удалять. Production full snapshot до apply и итоговый snapshot после.

## Приёмка и порядок

До реализации root фиксирует контракт и явные принятые/отложенные решения.
Авторы backend/frontend отдельно с непересекающимися файлами и WORK;
независимый reviewer читает source+unit/integration+operator. Только после
source APPROVE отдельный tester пишет новые process/браузер тесты; сами тесты
также независимо review. Затем exact Linux release/synthetic updater rehearsal,
production backup/update/readonly HTTPS, итоговый clean checkpoint.

Обязательные сценарии:

- Two same-role profiles: assigned A не получают B/unbound; unassigned доступны
  всем подходящим; два конкурентных A дают ровно один claim. Restart сохраняет.
- Exact revision, partial pair/null/unknown, role mismatch, foreign-project ID,
  stale Task.Version, replay и payload conflict; clear не удаляет историю.
- Profile edit/role change/archive блокируют новый claim; existing lease живёт;
  refresh/reassign явно восстанавливает очередь. Snapshot остаётся immutable.
- Зависимость developer→architect/human; release, ответ/contract, новый claim.
  Cycle, live consumer lease и stale generation остаются failclosed.
- Developer SHA→reviewer approve→tester passed→epic close; negative/not_run,
  target reopen sameSHA/newgeneration, superseding check, cross-epic obligation
  не обходятся UI, unlink или assignment. Review создаётся явно с точным SHA.
- Browser create-assigned атомарен; assign conflict сохраняет draft; committed
  lost-response retry не дублирует события. Viewer/epoch/archive/logout/project
  A→B/two tabs не обходят guards; keyboard/mobile focus и blocker navigation.
- Default+2projects migration3→4 сохраняет tokens/memberships/cookies/sessions/
  snapshots/leases/task JSON/notes/events/idempotency/blobs/исторические backups;
  schema3 binary отказывает; crash/resume и full restore4 actual process PASS.
- Production actual Go/Next SHA/BuildID, effective owned runtime, guards, helpers,
  all DB preservation, final journal/backup и HTTPS подтверждаются отдельно.

Физический второй ПК, native Claude/Codex integration, provider support и
managed worker execution остаются NOT-RUN/следующие этапы. Сейчас тесты не
запускались: этот WORK фиксирует архитектурный source-анализ, не QA реализации.

## Дополнение: атомарное явное создание QA

После уточнения root рекомендую расширить **существующий** CreateInput optional
`prerequisite_key`, не вводить новый scheduler или отдельный verification service.
Поле допускается только при создании task роли tester; source — reviewer в том
же project, с тем же developer TargetKey и TargetCommit. Target developer должен
быть done с данным SHA в момент операции (для этого нового сокращённого пути),
а обе проверки связываются с его текущим generation. Source может ещё ждать
approve: созданный tester тогда уже безопасно blocked. Cancelled или stale
reviewer для нового такого цикла отвергается; нужны новый reviewer/цикл.

Create + incoming `{from: prerequisite_key, to: newTester, type: blocks,
condition: approved}` + event + durable response фиксируются одной транзакцией.
Reject не оставляет ни task, ни link, ни event. Это закрывает race между двумя
browser запросами create и link, когда tester мог бы забрать задачу до review.
Прежний Create без поля остаётся совместимым и не получает неявной политики
«любой tester всегда после reviewer». Другие consumers используют обычный Link.
Новый цикл пользователя — явные кнопки «Создать ревью»/«Создать QA» с видимым
точным target SHA; автоматические цепочки по событию не добавляются.

Использовать existing Create validation/gates и constructor link, а не вызывать
Store.Link вторым transaction изнутри Create. Actor не обходит никакие чужие
lease права: новая незахваченная задача и edge появляются вместе. Regression:
concurrent tester next против create с prerequisite не выдаёт tester до current
approve; next после superseding negative reviewer также закрыт через existing
activeVerification. MCP остаётся12tools, Create schema расширяется совместимо.

## Принятое уточнение root: `check_create`

Disposition: **ARCHITECTURE APPROVE** для узкого варианта root ниже. Он заменяет
рекомендацию `prerequisite_key` в предыдущем дополнении; оба варианта одновременно
не реализовывать. Source/test/QA approval остаются отдельными будущими барьерами.

Новая REST/CLI/browser mutation `check_create`: `{request_id,target_key,
target_version,role,profile_id?,profile_version?}`, только human/coordinator
при project write ACL. Role только reviewer/tester. Target — done developer с
valid current SHA и точным expected Task.Version; backend сам фиксирует SHA,
generation и target provenance. Assignment pair — optional, с теми же строгими
active/current/version/role правилами. Generic Create остаётся совместимым.

Для tester backend выбирает latest применимый reviewer и требует **successful**
по существующему gate evaluator, а не просто `state=done/outcome=approve`:
current SHA/generation, activeVerification и recursive dependencies обязательны.
В той же транзакции создаётся tester и `reviewer blocks tester: approved`.
Если reviewer отсутствует/pending/negative/stale/superseded/blocked, вся операция
отклоняется без частичных строк/событий. Reviewer создаётся без implicit QA.

Рекомендация для минимального wire: title/description не принимать. Backend
генерирует понятный заголовок «Ревью T-N» / «QA T-N» и описание со ссылкой на
целевую задачу, её названием, SHA и generation. Критерии исходной developer
задачи читаются через Get; пользователь может добавить detail существующим
разрешённым metadata update. Не копировать автоматически инструкции профиля,
secrets, comments или mutable evidence в новые тексты. Так user явно создаёт
проверку выбранного результата, не конструирует второй шаблонизатор.

Полный ответ — созданная Task с capture SHA/gen/assignment/links. Повтор имеет
тот же request/session scoped durable result даже после изменения target или
профиля, если current auth/ACL позволяют replay; новый request со старым
version получает409. Unknown payload сохраняется для exact retry. Browser
кнопки доступны только на фактически done developer task, QA дополнительно
отображает конкретного требуемого reviewer; факт доступности перепроверяет store.
MCP tool budget12 сохраняется: новый check_create только REST/CLI/browser,
существующие generic Create/next остаются доступны native MCP.

Обязательные дополнительные тесты: race target reopen/version update против
check_create, два одновременных reviewer create с latest semantics, exact replay
после target mutation, latest reviewer negative/pending/stale/cross-epic,
concurrent next не видит tester без approved edge, assignment validation rollback,
viewer/foreign project/archived project/epoch denial. Title не выдаёт human
оператору права holder: созданную задачу исполняет native role session.
