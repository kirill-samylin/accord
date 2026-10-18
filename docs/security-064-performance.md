# WORK-064 — security review performance / schema4

**REQUEST CHANGES.** Два уже открытых блокера063 независимо воспроизведены;
новых блокирующих security-находок по этому срезу не подтверждено.
Это не security approve поставки, live/deploy или функциональная приёмка.

## Версия и границы

Source: `/Users/shagi/Desktop/Project/agent-board/.local/performance-003`.
Проверенная production/test версия `b9fdc155a1d0a02bf6000c3654e006a778be23cd`,
base `30abe1674e972ea28272dacdb18392070840e0ed`. Root в ходе проверки добавил
только docs063 commit `cb5388b8abd3c152d546e20bae7fe8e4625d9b79`;
production/test source остался точным b9.
Manifest057 SHA256 `67c8d3d8b7e336214007140796b6fd14d6c1cc92b6eba78560b142c6e5ae69ce`.
До/после сверены все48 source hashes; mismatch0, aggregate
`42d32f7826481346d8fc16fede29bb04c674708560b36edbe8e517c61272c2d6`.
Author list содержит51файл с документацией; reviewer docs не source.
Доказательства и hashes: `.local/work064/results.json`, `source-before.json`,
`source-after.json`. Авторские PASS не заменяли собственные прогоны ниже.

Роль `/root/security064` отдельна от автора. Прочитаны Accord AGENTS,
security/config/performance053/runtime contracts, WORK057/060, D review054,
процесс/parallel-work и профиль security Давинчи. Named Claude role bindings
недоступны; унаследована та же доступная модель, что у автора. Tandem OFF
по прямому поручению. Production и tests checkout, Давинчи, общая TASK,
Git-операции/SSH/live этим reviewer не выполнялись. Новые process/E2E и дополнительные агенты не запускались.
Запись ограничена этим отчётом, собственной WORK и ignored evidence.

## Модель угроз

Активы — token/scopes, session/role/task generations и TTL, текущие правила,
entity state, exact-SHA gates, audit/replay, migration snapshot и его provenance.
Недоверены HTTP/MCP inputs и переданные caller DTO. Trusted local admin и тот же
OS UID имеют прямые права к данным; это не tenant isolation. Другой Store/rawSQL
commit не может быть проигнорирован как локальный cache hit. Cache — только
ускорение: SQL authority, writer lock, current config и fencing сохраняются.

Главные новые риски: разделяемый entity pointer после fork, пропущенный dirty
journal, outward nested aliases, принятие stale cache после внешней записи или
ошибки commit, частичная публикация entities/audit/projection/replay и изменение
provenance/полномочий при schema4 migration.

## Блокеры

### R063-01 · P1 · Rejected Create меняет committed RAM без SQL/audit

`internal/store/tasks.go:38` получает общий parent pointer через `readTaskFor`,
затем `tasks.go:72–75` напрямую меняет `State`/`Version` без `editTask`.
Условия: авторизованный caller с create task, завершённый parent epic.
Успешный child Create оставляет SQL epic `done/version2`, RAM — `open/version3`.
Если последующая area validation отклоняет candidate (`../outside`), следующий
Get также показывает `open/version3`, хотя операция отвергнута.

Security impact — нарушение целостности authoritative state и атомарности отказа:
решения eligibility/gates и видимое состояние могут зависеть от процесса/cache;
отказ не должен менять состояние, обходя durable mutation и audit. Не установлен
новый bypass token/role или доказанный positive exact-SHA gate exploit; этого
ограничения достаточно для точности оценки, оно не снижает blocker063.

Нужно пройти write barrier/reopenEpic до изменения parent, учесть verifier reopen
того же epic и проверить success→SQL/reload/restart, rejected candidate→RAM/SQL/
audit/replay unchanged. Оба исходных probes063 повторены независимо: **FAIL**.

### R063-02 · P1 · Local HandoffEpic отдаёт alias к retained state

`internal/store/handoff.go:46,52` захватывает shallow Task во внешнем `out` и
возвращает его после adminTransaction; generic detach в
`internal/store/transaction_snapshot.go:102` копирует только внутренний `struct{}`
из `internal/store/maintenance.go:49–52`. Outward Task не проходит этот boundary.
Условия: локальный административный вызов HandoffEpic, nested поле результата.
Изменение `*out.FirstDoneAt` после возврата меняет следующий Get с2026 на2025.

Security impact — caller может изменять retained entity вне writer gate,
валидации, CAS/audit и durable commit; nested Areas/Links также должны быть
изолированы. Это дефект integrity и потенциальная race при concurrent caller.
Удалённый bearer exploit этим тестом не доказан: точка входа local-admin,
обычный bearer Handoff идёт через generic mutate. Severity P1 и blocker063
сохранены, чужой независимый finding не закрыт самопроверкой.

Нужен generic transaction result Task либо явный полный detach внутри callback
с обработкой ошибки **до commit**. Проверять весь nested graph и durable state;
post-commit detach с возможной ошибкой недостаточен. Probe063 повторён: **FAIL**.

## Проверенные механизмы без дополнительных blockers

- `transaction_snapshot.go:61–83`: immediate writer lock предшествует data_version,
  current config, clock и transaction auth. Retained connection задаёт поколение
  data_version; replacement начинает cold. Publish cache только после Commit;
  SQL/clone/commit error или panic сбрасывает cache/connection. Внешний task,
  singleton, config/scopes/revoke виден следующей transaction.
- `mutation.go:38–78`, `auth.go:107–124`, `runtime_roles.go:63–91`: replay повторно
  читает действительность токена и текущий role scope из SQL; current role,
  human allowlist/membership берутся из active canonical config. Expired SID
  может получить точный исторический DTO по контракту204, без renewal. Такой DTO
  не становится живой lease: следующий stale generation mutation запрещён.
  Token expiry equality/revoke запрещает и replay. Проверки выполнены ниже.
- `transaction_snapshot.go:125–150`: cache reuse требует совпадения version,
  SHA и exact canonical bytes; иной valid config парсится, mismatch fails closed.
  Собственный probe меняет canonical bytes+SHA rawSQL при прежней version и
  подтверждает новый note cap. Fault retained writer не выдаёт cached success;
  retry заново читает rawSQL task change.
- `snapshot_copy.go`, `mutation.go`, `persistence.go`: edit barriers копируют
  RawMessage/nested graphs; candidate отделён от expiry snapshot, event/replay/
  projection/entities сохраняются одной transaction. Чтение всех direct entity
  writers подтвердило два исключения выше; новые не обнаружены. Partial SQL
  faults/отмена и оба исхода неоднозначного Commit повторены независимо.
- `events_indexed.go:31–49,94–110`: task/type/limit — bound parameters,
  INDEXED BY выбирается только из трёх внутренних constants. В persistence,
  projection table names и migration trigger operation names также константные.
  SQL interpolation внешнего имени/фильтра в этой дельте не обнаружена.
- Schema4 сохраняет raw events/replay bytes, exact integer prefixes и UTC sec/ns;
  dirty INSERT/UPDATE/DELETE triggers требуют sync rebuild, compatibility histories
  идут через retained reducer. Migration4 DDL/backfill/validation/user_version
  находятся в одной transaction. Это не новое право импортировать audit facts.
  Schema1/2 legacy scopes остаются точным прежним списком, без wildcard/new roles.
- Private backup/path guards и recovery token/session reset сохранены; raw live
  DB при WAL не копируется. Schema4 принимают backup/restore, newer schema отклоняется.
  Recovery/TTL/fencing не ослаблены ради производительности. Verified backup и
  schema3 rollback при DDL/cancel/storageFULL пройдены, byte preservation проверено.
- Pool4 реально проверен: retained writer и3 spare имеют busy_timeout5000 и
  cache_size−8192. No-op read не меняет внешний data_version. Typed BUSY retry
  ограничен admission/startup; бизнесоперации и Commit не повторяются вслепую.
- Infra timeout delta: prior schema1/2/3 допускает cap300s/явный1..900s override,
  schema4 сохраняет обычные15s. Ошибка/restart systemd завершает ожидание;
  schema-change rollback закрывается inspection guard.5unit checks PASS.
- Admission100rps/burst200/max64 и transport security не менялись (static diff).
  Новых buckets по недоверенному bearer и обхода admission нет. HTTP/TLS/load
  validation в этой WORK не запускалась.

## Фактические проверки

Pinned Go1.27.2 darwin/arm64, GOTOOLCHAIN=local; shared build/mod/tmp paths MAIN.
Все команды/exit/log/overlay/probe hashes — `.local/work064/results.json`.
Новые security probes работают через ignored Go overlay, исходные test файлы
не менялись. Три probes063 использованы с сохранением их авторства и повторены
самостоятельно; это не выдаётся за новые найденные064 дефекты.

| Проверка | Факт |
| --- | --- |
| Targeted store authority/TTL/scopes/replay/config/fault/migration/immediate/commit +3own probes | PASS18 top-level,9subtests;1.623s package |
| Race expired-scope replay + both real SQLite Commit outcomes | PASS2 top-level,2subtests;2.732s package |
| Race собственные token-expiry/rawSQL-config/reconnect/all-pool probes | PASS3 top-level;2.398s package |
| Schema1/2 exact scopes/history + online WAL backup/recovery + unsafe path guards | PASS5 top-level;0.583s package |
| Readiness timeout/schema-aware rollback checks | PASS5unit;0.017s |
| Независимый повтор известных probes063 | FAIL3 /2blockers;0.303s package |
| Source48 hashes до/после | PASS, mismatch0 |

Полный cmd/internal/race/vet/build и annual200k diagnostic здесь **NOT-RUN**:
полный author прогон принадлежит WORK057; он не заменяет security review.
Новые product process/E2E до source approve не создавались/не запускались.
QA056-01 concurrent CLI BUSY_RECOVERY closure этой WORK не присвоено:
source retry прочитан, независимый process retest остаётся этапом tester.
Target VPS/API p95/RSS/startup/TLS, live deploy,2physicalPC, реальные providers,
recovery на настоящем сервере — **NOT-RUN**. Static Python checks не systemd PASS.
Source063/064 blockers не снимаются локальными performance числами057;
original049 p95≈10025ms **FAIL** остаётся, C live1e9025c/schema2 не менялся.
Backup retention/byte budget зарезервирован отдельной WORK062 и сюда не входит.

## Передача

Новых рекомендаций, требующих отдельной implementation delta, нет. Не расширять
fixes за пределы двух blockers; добавить regression coverage для их boundary.
Source freeze b9/aggregate42d32f сохранён. Fixer работает в другой копии по root;
эта WORK source не исправляет и исправленную будущую версию не одобряет.
После исправления требуется независимый exact-source recheck обоих blockers,
потом разрешённый root этап process/runtime QA. Security этап на этой версии
завершён вердиктом REQUEST CHANGES; TASK003/WORK057 не объявлены принятыми.
