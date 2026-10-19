# WORK-103 — control web backend

- Родитель TASK-005 / REQ-005; разработчик `/root/control_backend103`, экземпляр текущего запуска.
- База a6501d8; область `internal/core/control*.go`, `internal/store/control*.go`, shared catalog helper, `internal/web/**` и эта WORK. UI104 единственный автор `internal/webui/**`.
- Прочитаны профиль developer, общий процесс, parallel-work Давинчи read-only, AGENTS Accord, frozen §8/9 control-web-contract, WORK100/101. Tandem OFF согласно AGENTS Accord; модели Claude недоступны, модель наследована.
- Статус: completed; передано на независимое ревью, не approve. DTO согласованы сообщением root без изменений. Требования источника: фактические session/lease/token/task facts, отсутствие process claims; native12/schema2/web1/auth/Origin/fencing не меняются.

## Решения

- Полные overview counts и batch token validity в одной store transaction; native Sessions остаётся TTL-only.
- Catalog task pagination выделена в общий helper без смены native predicates. ControlTasks применяет literal lower-case Go search по всему набору до paging, cursor binds query.
- DESC RecentEvents читает SQL-фильтры до LIMIT+1, materialized expiry включается той же transaction. Browser projection не включает TokenID событий.
- Новые read pages используют frozen Page; JSON overview отдельным путём не выдаёт form action IDs.

## Проверки

Выполняются targeted unit/integration. Полные makecheck/race, новые process/browser E2E, deployment не запускались и принадлежат последующим ролям/этапу после approve.

## Журнал

- 2026-10-19T22:22:43+03:00 started — фактическое date при чтении профиля; ownership и frozen DTO подтверждены root.

## Реализовано и файлы

- `internal/core/control.go`, `control_validation.go`: frozen readers DTO, browser filter vocabulary; никаких native tools.
- `internal/store/control_{overview,sessions,tasks,activity}.go`: точные counts, batch current credential/session/lease projections, SQL DESC keyset, whole-project literal search.
- `internal/store/catalog.go`: единственный общий helper task filters/pagination для native и browser; прежние native filter/cursor semantics сохранены.
- `internal/web/contracts.go`, `http.go`, `pages.go`, `control_{data,pages,queries}.go`: новые SSR маршруты и default overview, legacy routes/forms сохранены; только JSON overview poll, query validation и retained Draft. JSON ошибки `{code,error}`, stale native session `LEASE_LOST`409, auth401; cached expired web board session также требует explicit continue.
- `internal/store/control_test.go`, `control_counts_test.go`, `internal/web/control_test.go`: синтетические unit/integration fixture regressions. Все новые файлы меньше300 строк.

## Фактическая проверка frozen backend

- PASS: `.local/toolchains/go/bin/go test ./internal/store ./internal/web -count=1` — store2.229s, web1.750s. Это targeted packages, не полный makecheck/race/process/browser/live suite.
- PASS: `git diff --check -- internal/core internal/store internal/web`.
- PASS regressions: >100 task/session projection и search hit после100; русский mixed case/title/description/key; counts empty/state/blocked/closed; revoked credential при живом TTL и native TTL-only compatibility; credential expiry; current lease generation; exact session lookup после100, kind/human и filter-bound opaque paging; recent SQL filters доLIMIT, strict DESC/new event insertion, materialized expiry; repeat reads не изменяют lease/generation/session/events; JSON reads не создают action IDs и не раскрывают token/CSRF; query invalid/unknown/duplicate400; SSR routes и actual settings/stats DTO; прежние web regressions также прошли.
- NOT-RUN: independent code/security/design approve, новые process/browser E2E, build/full suite, deployment. Последующие роли проверяют итоговую интеграцию UI104; author сам себе approve не выдаёт.
- Контракт и пользовательские инструкции/TC ведёт root/аналитик по общему scope; точные backend сценарии для QA перечислены выше и в контракте §7. Новые process/browser E2E здесь намеренно не создаются согласно процессу.

## Handoff

Baseline `a6501d8` + рабочие файлы; Git commit запрещён координатором. Frozen source manifest: `/private/tmp/accord-control-099/backend-103-sha256.txt`, SHA256 `e55fb67d263cd916342eec299a92d7b65d3bcf228e4d88baf1cb620720c78bcc`. UI104 независимо редактирует `internal/webui/**`, его изменения не включены в backend freeze. Привязка manifest охватывает16 core/store/web source/test files.

- 2026-10-19T22:28:34+03:00 completed — фактическое date после PASS store/web; frozen backend готов к независимому ревью, проверяющему переданы пути/manifest. Исходные ошибки compilation/test fixture устранены до текущего PASS, фиктивных успешных результатов нет.
