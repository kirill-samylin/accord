# WORK266 — план узкой capable поставки WORK264

Автор: automation_infra. **PLAN READY, исполнение NOT-RUN.** Собственная область:
эта карточка и private `.local/task014/ops266/`. Canonical source/docs/старые ops212,
сервер/liveDB/GUI/Mac worker и рабочий Test workflow не менялись.

Проверены source-only canonical upgrade/staging/preservation/backup guards,
прежние operator212 и final independent evidence258. Baseline уже **a32/schema6,
complete**, не прежний273/schema5. Last final-audit cfdacbe2… и acceptance258
подтверждают4boards/9runtimeDB/9historicalDB на момент предыдущего выпуска;
свежая рабочая Test history должна войти в новую полную копию.

WORK264 получил SOURCE APPROVE202 receipt b47eb60d…/manifest cfcf4d71…,
committed candidate **4ae529272e1542e8473902de26d68349af266d1d**. Исправление:
просроченный interrupted step становится стабильным blocked/STEP_TIMEOUT;
deadline, budgets/ACL/lease/SHA gates не расширены; одна существующая UI подсказка.
Новые Linux/Next artifact hashes/BUILD_ID ещё не известны: сборка не выполнялась.

Полный компактный operational checklist и точные canonical build/package/
stage/apply/finalize entrypoints — private `.local/task014/ops266/PLAN.md`.
Нового updater/framework/scripts не создано. Root выполняет позже после source
review/build provenance и завершения/quiescence текущего R→T sample; никаких
самовольных stop/cancel для освобождения выпуска.

Ключевые delta checks:

- clean committed4ae Go+Next build, same exact Source/VCS/bundle/member hashes;
  Next нужен для одной изменённой строки; Mac worker не переустанавливать;
- новую маленькую task-local привязку ops212 отдельно review: baselinea32,
  schema6 live inventory/active workflows, новыйcandidate, source264 approval;
  прошлые полные native gates не retagging на новуюrevision;
- UID999 WAL-aware readonly inventory/closed connections, UNKNOWN-before-SSH,
  existing locks/owned process proofs и полный backup всех проектов сохраняются;
- capable schema6→schema6 без DDL: exact all-table/history/authority preservation,
  pre-copy тоже schema6, **не ожидать ещё4schema5 migration copies**;
- independent HTTPS smoke exactnewrelease → canonical finalize → ownedruntime,
  journalcomplete, новый full format2 schema6 backup и independent final review.

Проверка подготовки: источники/receipts read-only и локальные manifest/hash/diff
checks. Actual build/deploy/browser/native/new process tests: **NOT-RUN**.
План не заявляет production исправление или завершение общего TASK014.

Private plan manifest SHA256:
`b82ab1fd7c7bdfb526db84d5606362efc385c3dcd6ff0118201b736786bda570`.
136-line plan; refs/hash и whitespace/fence checks PASS. Все private files0600,
каталог0700. Никаких scripts/runtime actions в этом WORK не создавалось/не запускалось.
