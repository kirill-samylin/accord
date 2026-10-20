# WORK-148 — backend координации

20.10.2026. TASK011. Автор backend, ветка `codex/coordination`, начало343b93c.
Область: internal/core/store/projects/transport/bridge/web, cmd/agent-board,
авторские unit/integration и существующие acceptance schema expectations.
Канонические docs, frontend, infra и новые process/E2E не изменяет.

Статус: IN PROGRESS. Реализация по docs/coordination.md после WORK147 APPROVE
и явного ответа пользователя «Да, этот контур». Checkpoint не означает QA PASS.

Контракт: pinned profile assignment, routing по session snapshot и current
profile; existing live leases не меняются. Atomic CheckCreate захватывает target
version/SHA/generation и successful latest review prerequisite для QA. Explain
использует authoritative gates, current provenance отдельно от successful outcome.
Schema4 additive coordination_metadata, старые JSON rows DDL не переписывает.

## Реализовано

- Core TaskAssignment/Assign/CheckCreate/Coordination DTO и event-time structured
  assignment previous/next/reason, check target/SHA/generation provenance.
- Store atomic assigned Create и Assign с expected task version; прежний Next
  проверяет pinned session snapshot/current profile и сериализует claims между
  DB connections. Edit/archive блокируют новые claims, не живой holder.
  SessionView/Claim Queue персональная; общая queue отражает current availability.
- Общий task constructor используется CheckCreate: human/coordinator, capture
  done developer target Version/SHA/generation, QA только после successful latest
  current reviewer и blocks/approved в той же транзакции.
- Отдельный Coordination reader с authoritative blocked/successful/latest gates,
  current provenance отдельно successful outcome, bounded deterministic200
  collections, latest/current checks первыми, unique walk budget/truncated.
- Human admin assignment/link/cancel/restore незахваченных задач. Чужой live lease
  не обходится. Nonzero stale Transition.Generation не становится admin fallback.
- Schema4 additive coordination_metadata exact singleton(1,1), failclosed повреждение,
  published preflight полных2/3/4; existing-only migration и capabilities согласованы.
- Scoped REST/CLI assign/check_create/coordination, strict unknown/duplicate/null/query
  rejection, MCP coordination resource и bridge SID injection; tools остаются12.
- Browser task detail Coordination/active metadata profiles/can_coordinate и
 4durable actions с исходными session/payload/epoch, recovery и unknown retry.
  Generic create принимает profile pair атомарно. Browser human не исполняет agents.

## Авторская проверка

PASS: первый полный `make test race vet`, включая все Go packages и существующую
process acceptance. Авторские новые unit/integration покрывают concurrent claims
на2DB connections, profile edit/archive/live lease, versions/replay/audit, atomic
QA/latest gates/reopen race/negative current, bounded deterministic explanations,
3→4 exact row preservation/snapshot/damaged4, REST/MCP12/resources/bridge session,
project ACL/revoke→regrant/archive, browser committed-lost-response retry/draft/
dependencies/check create. После последних узких metadata/bounded fixes выполняется
повтор полного Go/race/vet; окончательный результат будет записан ниже.

Existing schema fixtures адаптированы корректным удалением новой metadata до
искусственного downmark; существующий reset profiles3 fixture действительно
приводится к3. Existing acceptance изменены только два schema expectations3→4
и текст assertion, новых process/E2E здесь нет. Catalog resource count3→4.
Все новые production файлы и author tests≤300строк. Dependencies не менялись.
Process/browser/VPS QA отдельным тестировщиком и production delivery — NOT-RUN
этим автором; не выдаются за PASS source checks.

## Финальный авторский checkpoint

PASS: повтор полного `make test race vet`, включая tests/acceptance; private log
`.local/task011/backend-final-checks.log`. Последняя reader-only поправка
сохраняет обе required-role причины одного target, сортирует stable top201 и
отличает отрицательный current от stale; после неё PASS store unit и store race
(24.8s), CLI unit и targeted vet. CLI author test безопасных routes/payload добавлен.
Private race log `.local/task011/backend-last-store-race.log`. `git diff --check` PASS.

Учтены ранние reviewer150 findings: no unbounded duplicate recursive explanation,
truncated при фактическом unique cutoff, deterministic bounded closure subset и
две role reasons; запрет stale-generation admin fallback; native mutation query
rejection. Root уточнение exact coordination metadata singleton выполнено в Store
и published preflight с damaged extra-row regression. Source готов к независимому
APPROVE; этот WORK не заменяет approval или отдельную процессную/браузерную QA.
