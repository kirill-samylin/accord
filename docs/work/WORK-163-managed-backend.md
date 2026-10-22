# WORK-163 — backend управления native агентами

22.10.2026. Автор backend; координатор root. Source contract после независимого
APPROVE WORK165 — `docs/managed-agents.md`. База `d9f9eba6fdecb131e36f49907d421a2df43cccbd`,
ветка `codex/managed-agents`, рабочие изменения без отдельного commit.

Реализованы core DTO, project-local schema5 hosts/runs/commands+metadata,
transactional isolated mutations/idempotency и fenced poll/report. Host owner и
локальные controllers независимы от project/task/profile role. Worker proof256bit
генерируется локально, сервер хранит только hash. Current project ACL/owner epoch/
host grant проверяются до native и browser cached replay; controller command issuer
проверяется повторно до dispatch/accepted/start. Human management session отдельно
от per-run bridge; provider credentials серверу не передаются.

Start/stop/resume/clarify сохраняются до dispatch. Stop обгоняет running command;
applied только после worker receipt и process/provider evidence, forced отмечается.
Clarification during running ждёт natural turn boundary; новое уточнение в stopped/
completed ждёт explicit resume. Resume объединяет прежний mailbox sequence и
продолжает exact native session. Lease timeout не означает stopped, новый worker
не получает automatic takeover при живом/неоднозначном run. Recovery старой
generation не меняет новый resumed run. Restore отключает host proofs/epochs,
pending commands и automatic resume, сохраняя историю.

Native routes `/api/v1/p/{id}/managed/**`, CLI mapping и browser `/web/p/{id}/managed`
с action_id/CSRF/epoch/recovery интегрированы. Browser не имеет enrollment/controller
edits. Конфликт команды возвращает свежий authoritative run/version/capability.
PageData содержит current project-local actor control ID и server observed_at;
полные логи/provider reasoning не публикуются, bounded normalized tail64KiB.

Изменённые области: `internal/core/managed.go`, `internal/store/managed*.go`,
store/persistence/migrations/schema validation/maintenance, projects managed service/
storage preflight, transport managed operations/routes, web managed handlers/
projection/actions, CLI client mapping; старые fixtures корректно формируют schemas1–4.
Schema migration не чинит заявленную повреждённую5 через CREATE IF NOT EXISTS.

Авторские unit/integration: store double poll/retry/proof/no double start, stop
priority/forced/exact resume, queued clarification/recovery fencing, controller
revoke before cached replay/dispatch, restore; registry foreign project/viewer/
owner epoch; реальный browser HTTP conflict/projection/CSRF/unknown fields и cached
disable; schema4→5 source bytes/private snapshot и damaged5. Native strict parser
проверяет unknown/null/duplicate/query/shell fields. Целевые package store/projects/
transport/web/cmd PASS после исправления fixture; итоговый прогон обновляется ниже.

Фактическая проверка выполнялась bundled Go в private `.local/tests-managed`
TMPDIR: стандартный macOS temporary parent не удовлетворяет существующим private
path guards, защиту не ослабляли. Новых зависимостей нет.

После source review исправлены: dispatched revoke больше не сообщает stopped
без proof (durable owner safety stop); failed resume запрещён; exact terminal
report replay отделён от transient lease/generation checks; CreatorEpoch и current
issuer credential проверяются через registry→legacy authority→selected board,
включая atomic credential set без nested default DB. Revoke/regrant не оживляет
старые queued commands. Author regressions дополнены этими сценариями и
nondefault legacy owner/issuer path; целевые5packages PASS.

Добавлены последние security regressions: accepted/applied issuer revoke→owner
safety-stop; отдельный reserve stop при заполненных16mailbox/256history; cached
known409 после освобождения workspace не запускает скрытую mutation (pure
ManagedPermission guard). Metadata list и encoded detail tail bounded≤1MiB client
response:≤100commands/256KiB commandJSON, commands_truncated; полная history durable.
Final author package run store/projects/transport/web/cmd PASS; git diff --check PASS.

Независимый implementation review: PENDING. Новые process/E2E пишет отдельный
tester после APPROVE; native Mac/Windows/второй ПК и production: NOT-RUN этим автором.
GUI/Tandem/infra/frontend/секреты не изменялись. Исторический `transport/http.go`
превышает300строк до этой задачи; дополнительные routes сохраняют его единую
HTTP perimeter/dispatch ответственность. Новые implementation файлы≤300строк.

Final recovery follow-up: recovered_stopped закрывает peer accepted/execution
commands текущего generation, сохраняет undispatched clarification mailbox.
Old accepted generation не блокирует/не исполняется после resume. Новый регрессионный
сценарий start→stop accepted→recover start→resume generation2 с сохранённым
уточнением PASS; целевые5packages повторно PASS и diff check PASS.

Containment boundary уточнена по независимому review: Unix stop подтверждает
managed CLI и owned PGID; setsid/double-fork может выйти из группы. Полное дерево
и отсутствие всех writers не заявляются. Windows contract — owned Job subtree;
реальный Windows остаётся NOT-RUN. Workspace mutex кооперативен, фоновым сервисам
нужна локальная проверка. Отдельному QA обязателен setsid/unrelated probe с честным
разделением owned scope PASS и full containment NOT-RUN. Implementation не менялась.

Profile continuity follow-up на базе65e1ba0: независимый design APPROVE WORK165.
Managed Describe narrow binding проверяет current host owner/epoch, живую worker
lease, run generation/state и matching accepted/applied execution, затем берёт
immutable run.Profile. Обычный Describe по-прежнему current active; task assignment
и прочие gates не ослаблены. Guard действует перед managed Describe replay тоже.
Per-attempt private bridge config immutable/O_EXCL/fsync, путь attempt-N; bridge
сам инжектирует run/generation и скрывает эти поля модели. Host proof в child не
передаётся. Разрешённая root область расширена на bridge/cmdworker/worker config.
Авторские store/worker binding regressions и in-process actual MCP archive snapshot
interop PASS; registry owner epoch/foreign-owner test и полный7packages прогон
фиксируются ниже. Это не real provider/Windows/process QA.

Final profile follow-up author run: store/projects/bridge/worker/transport/web/cmd
PASS, включая registry epoch/foreign host owner и actual in-process MCP binding;
git diff --check PASS. Независимый source re-approve и отдельный process QA
остаются обязательными до поставки нового candidate.

Root Forced recovery follow-up: recovered_stopped сохраняет Forced=true даже
при следующем proof без Forced; managed_recovery_test дополнен assert true→false
не понижает историю принудительной остановки. Focused store recovery/managed
Describe PASS, diff check PASS. Source проверяется WORK165 перед rebuild.
