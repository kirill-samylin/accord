# WORK-060 — дополнительные author unit/integration performance

Статус: выполнено (author tests; независимое ревью отдельно). Исполнитель `/root/perftests060`, роль разработчик,
экземпляр этого запуска; parent WORK057/TASK003, contract053 §§2–4,7.
Вход HEAD `30abe1674e972ea28272dacdb18392070840e0ed`, ветка
`codex/performance`, production WORK057 изменяется параллельно.
Именованные Claude модели недоступны; закрепление модели роли не действует.
Tandem OFF по прямому поручению; claims не выполнялись.

## Границы и цель

Запись только `internal/store/{cow,authority,fault}_performance_extra_test.go`,
этот WORK и ignored `.local/work060/`. Production, schema migrations,
существующие tests, Davinci, общая TASK, Git/push/live не меняются.
Это авторские регрессии до review, не независимое approve и не process QA.
Проверить returned graph detachment, same-DB authority reload и атомарность
candidate при SQLite failure/cancellation/Close через Store API и durable DB.
Прочитаны AGENTS, полный contract053, config/security contracts, WORK057,
review054, developer profile и процесс/parallel-work Davinci.
Новые продуктовые правила и permissions не вводятся.

## Журнал

- started 2026-10-18T19:45:05+03:00: `/root/perftests060`, ready → в работе;
  назначены непересекающиеся файлы, WORK057 подтвердил compile readiness и
  разрешил targeted/race своих tests; полный check/annual параллельно не запускать.

## Результат

11 top-level black-box unit/integration tests, 5 fault subcases. Только публичные
Store DTO/API, injected Clock и SQL private fixture; assertions на private cache
структуру, принудительной мутации production state и subprocess/E2E нет.

- `cow_performance_extra_test.go` (279 строк): poisoning полного возвращённого
  графа task/claim/session/note/events/config/history и повторное чтение/replay.
  Присутствующие maps, slices, RawMessage bytes, leases, attempts/timestamps,
  evidence_complete, comments, links, bound review, similarity и warning areas
  действительно заполнены fixture. Rules/notes в Describe, role deadline в
  Heartbeat/Sessions и soft-delete deadline Note проверяются отдельно.
  После poisoning durable logical dump и следующая Store transaction неизменны;
  completion evidence/attempt/replay сохраняют integer 9007199254740993 точно.
- `authority_performance_extra_test.go` (214 строк): A warm → B update → A update
  сохраняет новую title/version; внешний singleton/session виден сразу; новый
  note limit применяется следующей операцией A. Released task generation не
  оживает от старого exact replay. External scopes и revoke применяются перед
  следующим вызовом/replay. Expired Describe и Create replay остаются точными
  без renewal при сохранённом scope; после его удаления оба FORBIDDEN_ROLE.
  Подмена canonical config при прежнем hash даёт CONFIG_INTERNAL для read/write;
  DB не меняется, восстановление valid bytes возвращает рабочий Store.
- `fault_performance_extra_test.go` (237 строк): synthetic SQLite trigger abort
  на idempotency/tasks/sessions/events/projection_control. Все durable таблицы,
  включая lease/session/audit/replay/projection, совпадают с pre-call dump;
  следующий cached read прежний, retry того же request успешен ровно один раз,
  повторный replay не меняет DB. Cancelled admission не резервирует replay.
  Injected public Clock устанавливает owned transaction после BeginTx; 16
  cancelled waiters завершаются, concurrent Close ждёт owner и освобождает pool,
  canceled owner не публикует успех, второй Store затем успешно пишет.

Существующие author tests уже покрывают обычную lease/replay/expiry semantics,
config CAS и migration/oracle cases; они не скопированы. Дополнительный real
SQLite uncertain commit wrapper принадлежит WORK057 (`transaction_fault_test.go`),
здесь не дублирован и не приписан WORK060.

## Фактические проверки

Pinned executable MAIN `.local/toolchains/go/bin/go`: go1.27.2 darwin/arm64,
`GOTOOLCHAIN=local`, shared GOCACHE/GOMODCACHE в `.local/toolchains/cache/{build,mod}`,
GOTMPDIR `.local/toolchains/tmp`. Первичный временный собственный tmp допустим,
финальные команды использовали shared tmp. Фактические команды из рабочей копии:

```text
go test ./internal/store -run '^TestPerformanceExtra' -count=1 -v
PASS: 11 top-level + 5 fault subcases, 1.027s; .local/work060/targeted.log

go test -race ./internal/store -run '^TestPerformanceExtra' -count=3 -v
PASS: 48 executions including subcases, 25.069s; .local/work060/race.log

go test ./internal/store -run '^TestPerformanceExtraExpiredReplayUsesCurrentScopeAuthority$' -count=1 -v
PASS: 0.513s; .local/work060/scope-delta.log
```

Последняя команда относится к финальной test delta + Create replay (14 строк),
согласованной WORK057 после первого freeze. Targeted all/race выше относятся
к предыдущей версии authority test (Describe branch); неизменные cow/fault —
финальные. WORK057 получил новый hash и выполняет final namespace/race на новой
версии. Полный cmd/internal check, vet/build, process QA, target VPS/load,
независимое approve и отдельные legacy OpenExisting1/2/3 tests здесь NOT-RUN:
full check принадлежит WORK057, process/E2E tester после approve. Существующие
migration/admin compatibility tests сохранены. Production FAIL не обнаружены.

Ранние незачётные попытки: fixture Area.Aliases=nil не проходил config schema;
исправлен synthetic input на explicit []. Trigger stats_prefix не достигался:
фиксированные часы общего fixture раньше реального migration event, поэтому
projection переходила в compatibility path. Замена на fault публикации
projection_control проверяет rollback независимо от этого path; bounded prefix
workloads проверяет WORK057. Cancelled SQL owner может вернуть context.Canceled
или sql.ErrTxDone из-за asynchronous database/sql rollback; оба — отказ,
при этом неизменность DB и освобождение ресурсов проверены. Эти test assumptions
не объявлены production defects. Неопределённый commit здесь не инжектировался.

## Версия и передача

HEAD `30abe1674e972ea28272dacdb18392070840e0ed`; tracked WIP diff сохранён как
`.local/work060/observed-tracked.diff`, SHA256
`3bcadbb6fb1eb5de3231b96b75b3b77b5f3e9c18b1366081e2ad51c07f1e0187`.
Before/after race manifests сохраняют hashes всех Go/Python inputs:
`.local/work060/source-before-race.json` / `source-after-race.json`.
Aggregate before `e5cd176bae7515d039ebd5c77474c41b5e077f2e94d065c3cb852ed9c5d79f9e`,
after `eb06337b886a5e498e85b1691e444ae823f521def60c47c0500e0e840b0413ad`.
Во время race WORK057 менял migrations_schema4_test.go,
projection_validation.go, stats_performance_test.go. Это observed WIP, не
final production freeze, точный compiled hash в переходный момент неизвестен.
Координатор и WORK057 уведомлены; final assembled checks связывает WORK057.

Финальный freeze собственных Go-файлов:

| Файл | SHA256 |
| --- | --- |
| cow_performance_extra_test.go | 390877870c68c8231bbcf2deee87d2b0a92e4f995a5cc02a589b6b75084c00c2 |
| authority_performance_extra_test.go | 69af162a3e7480292a7330d318727998876537483abbdd03a29763a1a442c876 |
| fault_performance_extra_test.go | 07299d15211ff780d5a81d15e3e2fcf7272bf7d315ad5121f39a1a968e9ad9fc |

Все новые Go-файлы <300 строк; gofmt применён. Это author delivery до review,
не независимый approve. Следующий исполнитель: WORK057 включает все три файла
в final manifest/checks, root назначает независимые review/security и затем QA.

## Продолжение журнала

- completed 2026-10-18T19:55:32+03:00: `/root/perftests060`, в работе → выполнено;
  переданы 3 frozen test files и hashes WORK057, targeted/race фактически PASS
  на указанных версиях, final narrow Create replay delta PASS; source не правил.
  Запись в Go-файлы остановлена, приёмка/independent review не присвоены.
