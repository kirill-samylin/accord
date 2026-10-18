# Производительность и schema4

Реализация WORK057 следует [контракту053](performance-contract-053.md).
Независимая проверка063 выявила открытые блокеры: Create пишет shared done-epic
мимо COW, local HandoffEpic возвращает alias nested поля. **REQUEST CHANGES**;
исправления назначает root отдельному свежему исполнителю. Ниже описан механизм
и его текущие исключения, не готовый к выпуску runtime.
Это не подтверждение p95/RSS на VPS. Текущий target runtime,
независимые review/security и QA фиксируются отдельно в TASK003.

## Транзакции и кэш

Store удерживает одно writer connection из pool4. Каждый connection получает
busy_timeout5s/cache_size8MiB из DSN; writer BeginTx использует проверенный
modernc1.60.1 `_txlock=immediate`. Контекстный gate сериализует локальные
транзакции, SQLite — разные процессы. Clock/auth/config читаются после lock.
Spare connections обслуживают Authenticate, Ready, Config/History и local lists.
Admin/mint/revoke/scopes/config/handoff/recovery идут через тот же writer gate.
Backup и offline DatabaseSchema используют собственные handles.

После lock PRAGMA data_version читается на retained writer. Внешний commit
инвалидирует snapshot; cold reload читает JSON tasks/sessions/notes и строит
компактные индексы. Собственный commit публикует candidate после подтверждения,
не маркируя старый state post-lock token другого connection. Любая SQL/commit/
clone ошибка и panic сбрасывают поколение; следующий вызов начинает cold reload.
No-op reads не делают dummy UPDATE и не меняют внешний data_version.
Close прекращает admission, дожидается владельца и закрывает retained connection
перед pool; ожидание gate учитывает отмену. Отдельного фонового worker нет.

Current config читается SQL version/hash/canonical bytes в каждой транзакции.
Parse/compiled schema переиспользуются только для точного совпадения текущей
редакции; Decode выдаёт transaction-private Workflow maps. Cache bounded одной
редакцией, новый connection/data_version reload не наследует доверие старой.
Неканонические/повреждённые bytes/hash — ошибка до изменения данных.
Токен и role scopes читаются SQL при новых операциях и durable replay;
replay не требует живой lease и ничего не продлевает.

Candidate содержит отдельные maps, общие immutable entities и dirty journal.
Первый editTask/editSession/editNote делает полный JSON detach одного entity.
В Create ветка done-epic пока пишет через readTaskFor; reviewer063 подтвердил
изменение committed RAM и расхождение с SQL даже после rejected candidate.
Это открытый blocker, который должен закрыть следующий fixer.
Новый объект регистрируется в journal. Metadata и role generations сохраняются
только при изменении; UPSERT готовится один раз на изменённую таблицу.
Expiry имеет отдельную границу candidate, поэтому доменный отказ не сохраняет
частичные изменения/renewal/replay. Generic transact/mutate DTO detach до commit: aliases
RawMessage/attempts/lease/maps/slices не получают доступа к snapshot. Local HandoffEpic обходит этот generic результат:
это открытый blocker063, а не подтверждённый detached outward path.

Индексы numeric order, role tasks, ownership, active lease areas, title trigrams,
epic children, verification и reverse dependencies обновляются dirty delta
после commit. Transaction overlay учитывает новые/редактируемые keys.
Metric facts пересчитываются для affected closure через старые и новые связи;
role/config changes инвалидируют gates целиком. Memo gate predicates живёт
только в одной transaction и сбрасывается перед изменением/emit. Никаких новых
ограничений на граф, историю попыток, description или количество групп нет.

## Журнал и статистика

Events сохраняются исходным raw JSON. Schema4 expression indexes соответствуют
query expressions task_key/type/cursor. [SQLite INDEXED BY](https://www.sqlite.org/lang_indexedby.html)
закрепляет соответствующий task/type/composite index: без этого planner выбирал
type-only и читал annual type stream, проверяя task JSON. Требование включает
fail-closed при исчезновении индекса; это план schema4, не необязательный hint.
Каждый filter stream читает LIMIT+1;
merge держит максимум две страницы, duplicate types исключаются. SQL predecessor
lookup сохраняет highest-scanned NextCursor на gaps и unmatched tail.
HighWater включает материализованную expiry этой же транзакции.

Projection_control хранит reducer version, built cursor, dirty и compatibility.
INSERT/UPDATE/DELETE triggers отмечают raw history dirty; даже UPDATE старой
строки с прежним max(cursor) требует synchronous rebuild перед Stats.
События/projection/replay/entities коммитятся вместе. Нельзя временно отвечать
старой projection или запускать её async rebuild.

Stats_prefix хранит exact cumulative values как decimal strings; SQLite REAL
не используется. UTC key — пара Unix seconds/nanosecond, включая year1/zero;
равные timestamps coalesced. Разность prefixes `<to` и `<from` проверяется на
int64 при сборке DTO. Group presence учитывает zero-metric events.
Blocked_prefix хранит exact nanosecond integrals и active counts, blocked_open —
start facts. Группы span берутся с его начала, границы клипуются обоими краями,
open intervals заканчиваются min(to,now). CurrentQueue всегда берётся из
post-expiry board с current config/gates.

Backward event timestamps, repeated blocked start, отрицательные raw tokens,
слишком большой закрытый span и несовместимый порядок переводят историю на
старый streaming reducer. Открытый legacy span вне Time.Duration range или
now раньше последнего события также используют oracle для данного запроса.
Это correctness fallback, **не performance PASS**. В annual diagnostic
compatibility=false; время монотонно, legacy-long open spans отсутствуют.

Seconds DTO остаются float64. Проверка differential допускает10ns абсолютной
погрешности плюс1e-12 относительно величины seconds DTO при суммировании;
timestamps не округляются. Integer/token/count поля сравниваются точно.

## Миграция и эксплуатация

Перед schema1/2/3→4 сохраняется существующий private verified snapshot с checksum.
DDL/indexes/streaming backfill/reducer coverage и differential ranges/user_version4
фиксируются одной write transaction. Backfill держит максимум256 decoded events;
обычный schema4 restart raw annual history не сканирует. Проверки включают
DDL conflict, cancellation, SQLite storage-full rollback и сохранность exact
raw event/replay bytes. Ошибка оставляет исходную schema и verified snapshot.
Backup/restore/recovery принимают schema4; более старый binary обязан отказать.
Downgrade — только offline restore прежнего snapshot + revoke/reset, с потерей
последующих изменений. E получает отдельную migration5.

Локальный synthetic annual diagnostic выявил migration29.1s, поэтому прежнее
installer ожидание≈15s недостаточно. Install для исходной schema1/2/3 использует
отдельный readiness cap300s; `--migration-ready-timeout 1..900` — явный bounded
operator override. Обычный schema4 restart сохраняет cap15s. Failed/restarting
systemd service завершает ожидание сразу; timeout проходит через прежние
schema-aware inspection/rollback guards, DB автоматически не восстанавливается.
Service CPU50%/MemoryMax256MiB не менялись. До live upgrade target VPS migration
измеряется изолированно;300s не являются обещанием длительности на VPS.

Annual synthetic DB≈173MB требует отдельной проверки backup retention/space.
Retention/byte budget — отдельная DevOps delta координатора; WORK057 не меняет
backup.py/backup service и не объявляет VPS дисковую ёмкость достаточной.

## Авторские доказательства

Первый diagnostic M1Pro/darwin-arm64, Go1.27.2, GOMAXPROCS1,2000tasks/200000events:
backfill+prebackup+validation29.108s; schema4 Open4.445ms; cold/warm get20.871ms/
0.441ms; tasks20 0.450ms, cold filtered events20 22.275ms, stats20owners2.749ms.
Create/update/next/comment/heartbeat/link/note/release/transition0.751–2.334ms
за один вызов. Это SQL synthetic fixture + реальные Store API, не HTTP400/p95.
Начальная версия диагностического seed держала200k generated events; её RSS
не используется для оценки migration/runtime. Финальный streaming fixture,
manifest, actual checks и binary hashes записываются в WORK057.

Независимые review/security, process QA, target VPS latency/RSS/start, TLS,
physical2PC, реальные providers: **NOT-RUN автором**. Старый049 latency FAIL
не снимается локальными числами. Большой graph fan-out, много групп/максимальные
payloads и совместная нагрузка Tandem остаются измеряемыми рисками.

Финальный author diagnostic после закрепления filter indexes: migration29.270s,
schema4 Open4.248ms, cold/warm get21.889ms/0.390ms, tasks20 0.467ms,
filtered events20 0.487ms, stats20owners2.487ms; обычные мутации0.811–2.215ms.
Peak RSS всего streaming test process (seed+migration+API)79,904,768bytes
(76.20MiB); это macOS diagnostic, не target VPS RSS. Annual raw fixture SHA256
`4ef88b11725e7a53c3a8b522666568afd7b4c43e52b57d46be14534cdc3e096b`.
Локальные числа не снимают R063-01/02 и не означают correctness approve.
