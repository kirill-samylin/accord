# WORK-165 — независимое ревью управляемых агентов

22.10.2026. Reviewer TASK013; область записи только этот WORK. Реализацию и
общую TASK ведут отдельные исполнители/координатор. Прочитаны AGENTS,
requirements/architecture, multi-project, agent-profiles, coordination, TASK012,
managed-agents и существующие store/transport/projects/browser hooks.
Файл превышает300строк как последовательный audit trail независимых review,
runtime findings и повторных approvals; прежние verdicts сохраняются для сверки.

## Предварительный контракт

Первый verdict: **REQUEST CHANGES**, четыре блокирующих уточнения перед wire
implementation, переданы архитектору и координатору:

1. Enrollment и worker authentication должны иметь конкретный путь через current
   registry guards: human credential + отдельное host proof допустимы, но worker
   proof сам по себе не даёт task/catalog CRUD. Исключение one-time выдачи секрета
   только private CLI output, не browser/server view. Local project credential
   для bridge — отдельный секрет с собственными native permissions.
2. Worker epoch/poll instance nonce и право захвата poll ownership требуют явного
   правила. Истечение heartbeat не позволяет запускать второй процесс поверх
   неизвестного первого; скопированный credential не доказывает новый host.
3. До dispatch queued start можно отменить. После durable dispatch, даже пока
   accepted report не пришёл, cancellation не доказывает отсутствие процесса.
   Stop остаётся ожидающим worker ACK. Нужны monotonic phases/terminal replay;
   accepted start никогда не переходит обратно в dispatch по timeout.
4. Controller IDs должны иметь project/identity namespace. Browser host update
   не расширяет локальное согласие; новые grants вступают в силу только после
   подтверждения локального allowlist. Отзыв проверяется перед replay/poll/report.

Высокоуровневые границы согласованы для последующей реализации: outbound-only
worker, project isolation, immutable profile snapshot, отдельная board lease,
локальные workspace/executable/preset allowlists, exact provider resume, OS-owned
trees, ambiguous crash без auto relaunch, schema5/managed_agents floor, all-project
backup и roll-forward. Это не source APPROVE реализации.

Повторное чтение обновлённого контракта (273 строки): **SOURCE CONTRACT APPROVE**.
Архитектор закрыл все четыре замечания: local generated secret с server hash,
existing owner bearer/management session+host proof; native-only consent updates;
project-local principal controllers; startup nonce/epoch bind без timeout takeover;
durable dispatch и stop-after-dispatch; monotonic report phases/terminal replay.
При реализации poll обязан отдавать ожидающий stop даже при активном accepted
start; stop не ждёт завершения provider turn. Clarify ждёт границы turn согласно
контракту. Approval относится к контракту и не заменяет source review кода.

## Integration mapping

- Core DTO и transport.Service используют общий store; новые native routes
  наследуют bounded strict JSON, Host/Origin/admission и current auth.
- Project service каждый human action и worker poll/report оборачивает current
  registry membership/access_epoch guard. Одного initial router read guard мало.
- Browser managed operations входят и в runAction, и в recoverActions; issued
  action_id/CSRF/current write ACL сохраняются также для cached result/replay.
  Redirects, DTO и query keys сохраняют исходный project scope.
- Host heartbeat и OS state независимы от board read session/presence. UI не
  называет queued/accepted command исполненной; provider exit0 не task completion.
- Host/worker auth не заменяет bridge Describe/Next, task generation и SHA gates.

Проверки на этом шаге: read-only source review PASS; unit/integration/process/
browser/native provider/Windows/второй физический ПК NOT-RUN. Новые process/E2E
тесты отдельного tester начинаются только после независимого SOURCE APPROVE и
также проходят отдельное review. Секреты/runtime/VPS/GUI/Tandem не изменялись.

## Частичное source review готовых областей166/167

Baseline `d9f9eba6fdecb131e36f49907d421a2df43cccbd` плюс незакоммиченные TASK013
изменения. Прочитаны web managed components/lib/tests и scoped integrations,
root capability/CLI integration, infra guard/layout/migration_files/validate/
upgrade_storage/upgrade и новые/изменённые infra tests. React skill
vercel-react-best-practices применён к отмене запросов/scoped caches/параллелизму.

Область infra167: **SOURCE APPROVE в пределах готового diff**. Новый5 floor
сохраняет все прежние floors, schema4 snapshots с checksum включены в allowlist,
metadata coordination проверяется и на5, managed tables/metadata обязательны.
Logical hashes прежних таблиц и immutable historical bytes сохраняются. Это не
approval ещё разрабатываемого Go migration/auth reset и не actual updater PASS.
Новые семь snapshot unit cases и guard regressions также APPROVE.

Web166/integration: **REQUEST CHANGES**:

- P2 `web/src/lib/managed-agents.ts:hostOnline`: browser Date.now сравнивается
  с server absolute timestamps и требует now>=last_seen. Браузер с часами на2с
  позади сервера видит свежий host offline и не может start/resume. Нужен server
  observed_at/выведенный server freshness либо response clock offset; локальные
  несинхронные часы не должны быть жёстким control gate. Тест future+1s сейчас
  закрепляет ошибку; требуется skew regression с реальной семантикой freshness.
- P1 backend integration `internal/web/managed.go:managedError`: командный409
  всегда формирует managed list без managed_run. useAction сохраняет прежний
  stale detail/version, поэтому предусмотренная UI обработка version conflict
  не получает актуальную работу. Требуется scoped current run detail для ошибки
  command и actual HTTP regression; новые capabilities без нового version мало.

Оба замечания переданы координатору. Остальные прочитанные UI guards сохраняют
project scope, frozen uncertain payload, pinned profile и отдельный host consent.
Фактический независимый повтор: infra164/164 PASS, Node26 web85/85 PASS,
TypeScript PASS. Первый запуск через системный старый Node не поддержал
strip-types; повтор правильным Node26 успешен, это не product FAIL.
Фактический backend/worker code review, новые process/browser/native CLI/Windows,
updater runtime и production пока NOT-RUN/PENDING. Full SOURCE APPROVE не выдан.

## Повторное review — сервер и worker

Web166 clock-skew fix: server `observed_at` используется вместо browser clock во
всех control gates; новые tests проверяют ±сутки/±2с и настоящую server freshness.
Version-conflict guard принимает только действительно новую detail того же run;
backend managedError теперь возвращает scoped current detail. **Web SOURCE
APPROVE**, включая новые три regression tests; независимый Node26 повтор88/88 PASS.
Фактический HTTP409 regression прочитан, выполнение после завершения backend fixes.

Backend163 initial full review — REQUEST CHANGES, переданы автору:

- P1 queued command с durable dispatch при отзыве инициатора нельзя отменять с
  run=stopped: native start уже мог произойти до потерянного accepted response.
  Нужны recovery/stopping и owned stop, без освобождения workspace.
- P1 resume failed разрешался сервером вопреки stopped/completed контракту и UI.
- P1 перед cached report replay проверялись transient lease/generation: потерянный
  accepted/completed response после30с не восстанавливался. Current auth/proof/
  owner epoch проверяются до replay; phase/fencing/lease — для новой мутации.
- P1 current issuer authorization в registry учитывала только active membership,
  без credential expiry/revoke и сохранённой issuer epoch. Queued start мог
  пережить revoke/regrant. Нужны current credential+write ACL+matching issuer epoch.
  Согласован порядок lock registry→legacy default credential guard→selected board;
  default selected board проверяет native credentials без вложенного того же DB.
- P2 Go schema5 managed metadata COUNT/MAX принимал единственную повреждённую
  строку id2/version1. Требуется точная id1/version1, как coordination и infra
  validator, и regression повреждённой объявленной5 без auto repair.

Worker164 initial source review — REQUEST CHANGES, переданы автору:

- P1 incoming command issuer не пересекался с current local controllers+enrolled
  owner; одного server grants недостаточно для local host consent.
- P1 reply до8КиБ отправлялся одним output entry, сервер разрешает4КиБ: completed
  report застревал с400. Нужны bounded UTF8 chunks и known-secret/ANSI/control/URL
  credential redaction до журнала и передачи.
- P1 Windows проверял exit только primary process, затем закрывал Job и заявлял
  complete/stopped без проверки descendants. Unix group SIGKILL errors игнорировались.
  Нужен доказанный owned tree exit либо fail-closed needs_recovery без free checkout.
- P1 Config os.Stat следовал symlink; UNC/reparse/canonical root и повторная
  identity проверка перед spawn не обеспечивали заявленный local path контракт.
- P2 backend/UI принимали unconfigured profile provider, worker отвергал его.
- P2 journal save мог записать больше8МиБ, тогда restart reader не читал собственный
  файл. Нужна согласованная byte quota без потери ambiguous state.
- P1 Unix native containment при SIGKILL/crash worker отсутствовал: Setpgid сам
  по себе не прекращает дочерний process group при смерти родителя. Watchdog и
  reaper — goroutines того же worker; native provider и MCP bridge могли продолжить
  запись/heartbeat без control deadline. Нужен независимый parent-lifetime
  supervisor/deathguard с owned group/pipe EOF и bounded termination. Durable
  needs_recovery запрещает новый launch, но не останавливает старый процесс.

Проверены первичные Windows API: [Job Objects](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects)
и [Microsoft о завершении Job](https://devblogs.microsoft.com/oldnewthing/20130405-00/?p=4743).
Job object обычно не сигнализируется при завершении всех процессов; primary wait
не является доказательством всех потомков. Native Windows runtime NOT-RUN.

Инструкция `docs/managed-worker-guide.md` прочитана: CLI/config/взаимный local
consent/exact native resume/необязательный автозапуск соответствуют выбранному
scope; окончательная сверка с path/job fixes ещё нужна. Полный Go повтор во время
редактирования163 остановился на временно несовпадающих сигнатурах, не считается
проверкой готового diff. Новые process/E2E тесты ещё не разрешены SOURCE APPROVE.

Итог текущего review turn: **REQUEST CHANGES** для полной реализации. Авторы
163/164 исправляют замечания, финальное чтение готовых fixes и повтор Go/race
ещё PENDING. Scoped approvals web/infra не разрешают process/E2E приёмку всего
контура. Проверяющий изменил только этот WORK, production/GUI/секреты не трогал.

## Финальное независимое SOURCE APPROVE

Повторно прочитаны fixes163/164 и связанные unit/integration regressions.
**SOURCE APPROVE** для backend, worker, web, infra и новых авторских тестов в
обозначенной ниже границе; это разрешает отдельному tester писать/выполнять новые
process/E2E и не является runtime/production acceptance.

Закрыты все перечисленные implementation findings. В частности, current issuer
credential/CreatorEpoch и host-owner epoch проверяются под правильным lock order;
accepted/applied revoke вызывает safety stop без fake stopped; существующий stop
переиспользуется, ему зарезервировано место сверх обычной command quota. Current
generation recovery закрывает прежние accepted/dispatched peers, сохраняет
undispatched clarification и позволяет exact native resume. Cached browser error
проверяет только разрешения и не выполняет отвергнутую мутацию после изменения
состояния. Encoded detail tail ограничен256КиБ/latest100 с честным truncated;
run list не дублирует prompt/output/full profile instructions. Schema5 metadata
требует именно единственную строку id1/version1.

Worker повторно проверяет local owner/controller consent до launch, сохраняет
path pins/canonical identity и checkout intent, блокирует links/reparse/network,
нормализует и редактирует known-secret output до журнала и отправляет≤4КиБ UTF8
chunks. Journal quota согласована с reader. Report4xx и stop errors fail closed.
Unix independent helper следит за приватной parent lifeline, WNOWAIT pin и
nonce-bound final proof; missing/invalid proof сохраняет needs_recovery. Windows
Job lifecycle проверяет owned job active count перед подтверждением exit.

**Заявленная Unix граница** — managed CLI и принадлежащий PGID. Она не включает
процессы, отделившиеся через setsid/double-fork/new PGID, и не доказывает отсутствия
всех writers. Checkout mutex кооперативен. Windows Job — отдельная более сильная
граница, реальный Windows NOT-RUN. Обновлённые managed-agents/guide/TASK явно
сохраняют это ограничение. Проверены первичные [POSIX kill](https://pubs.opengroup.org/onlinepubs/009604499/functions/kill.html)
и [создание session/process group](https://www.gnu.org/software/libc/manual/2.26/html_node/Process-Group-Functions.html).
QA обязательно проверяет escaped setsid child и не выдаёт его за whole-tree PASS.

Независимые фактические проверки: target Go store/projects/transport/web/cmd PASS;
полный race store/projects/transport/web PASS; worker/cmd race PASS; после последней
recovery delta повтор managed race четырёх пакетов PASS. Web88/88 PASS и финальный
TypeScript PASS. Infra164/164 PASS ранее в неизменённой reviewed области. Две
константы существующих acceptance schema4→5 отдельно APPROVE: checksum/bytes/
preservation assertions не ослаблены, новых process сценариев root не писал.
Root tiny UI CommandsTruncated notice и нейтральный forced-stop текст APPROVE.

База `d9f9eba6fdecb131e36f49907d421a2df43cccbd` + текущие uncommitted changes.
Binding113 изменённых/новых product/test файлов в cmd/internal/infra/web/src/tests:
SHA256 `a9c838550b6db5c24db4f6a690cdb091e942fcd09cf610318d32cf902b388fe3`
(sorted `path sha256` lines с завершающим newline; docs из fingerprint исключены).
После binding автор164 уточнил только комментарий Unix reap: проверяется
отсутствие live members в owned group, без обещания всех descendants. Этот
comment-only delta отдельно APPROVE, поведение не менялось.
Reviewer менял только WORK165. Полный root make check ещё выполняется; отдельные
child/crash/browser/native CLI/updater tests и их review, физический Windows,
второй ПК и production остаются NOT-RUN/PENDING. Source approval их не заменяет.

## Повторное review после checkpoint65e1ba0 и первых process наблюдений

Checkpoint `65e1ba0f2b8ccdc9fcd920c31c4af93c64997c89` прошёл root make check.
Отдельная QA обнаружила реальный Darwin preflight FAIL: zombie-only pinned PGID
возвращал EPERM на SIGKILL. Новый reap сначала проверяет census+WNOWAIT leader;
при live0 exit уже доказан, при live>0 ошибки сигнала по-прежнему fail closed.
Этот узкий runtime fix **SOURCE APPROVE**, фактический повтор QA ещё PENDING.

Закрыт поздний profile-continuity дефект: каждый native turn открывает свежий
Describe, а pinned revision после edit/archive больше не current. Managed
Describe теперь требует owner/current epoch, live worker lease, exact run/gen,
starting/running и accepted/applied execution с matching worker/epoch; возвращает
только серверный immutable run snapshot. Проверка действует до cached replay.
Ordinary Describe/current-profile и assignment/next/lease gates не ослаблены.
Bridge скрывает binding от model schema и фиксирует его в приватных immutable
attempt-N файлах, поэтому старый bridge не получает binding новой generation.
Store/registry/MCP interoperability/immutable-file regressions прочитаны.

Закрыт offline reconnect дефект: poll после истёкшей lease отвергал worker раньше
доставки его локального terminal proof. До нового poll при expired deadline,
no-active и pending теперь отправляется только durable recovery. Выбирается
current generation; locally-proven terminal даёт recovered_stopped, uncertain —
delivery_unknown. Retries сохраняют request/payload в одной worker incarnation;
новая incarnation меняет request ID. Recovery не Ack/apply accepted launch,
не продлевает worker authority и не запускает код; transient network остаётся
waiting, authoritative4xx fail-stop. Отдельное обновление human session не равно
новой worker lease. Три unit HTTP regressions прочитаны, heartbeat5s проверен.

**SOURCE APPROVE** всех этих late product deltas. Независимый полный Go race:
store/projects/bridge/worker/transport/web/cmd — **PASS7/7**; diff check PASS.
Binding36 изменённых/новых product и QA файлов поверх65e1ba0, sorted path+SHA256:
`8221470c8b415452585a573c5cb450f3928f8615b5e9c0f58e4ed429505b2f29`.

Отдельные новые QA tests также **SOURCE APPROVE**: deterministic native-like
process harness, lost accepted/started responses, redacted multibyte output,
owned PGID/forged proof/detached/реальный unrelated heartbeat, crash+checkout
intent, consent, offline, protocol-only16/256 quotas с257th reserved stop;
actual stdio bridge после profile edit/archive; Chrome flows/screenshots;
реальный old/new updater и offline restore. Частные binary copies SHA проверяются
до/после, source inventory фиксирован; cleanup ждёт принадлежащие процессы,
native report сохраняется даже при cleanup FAIL. Native Codex test отдельно
проверяет реальный session UUID, managed MCP Describe, task lease и exact resume,
без обхода approvals. Он не подменяется fake-provider PASS.

Первые updater evidence3 checks прочитаны: actual schema4→5 всех7DB/3projects,
old-row preservation/floor-before-DDL и full restore/auth reset PASS; lifecycle
systemd/Next симулирован явно. Финальный repeat с immutable binary/source pins и
новой reviewed сборкой ещё PENDING. Process/browser/native фактическая приёмка
после late fixes, Windows/Claude/второй физический ПК и production — NOT-RUN или
PENDING. Это разрешение на отдельную QA, а не завершение TASK013.

Последующая однострочная backend delta `recovered_stopped` сохраняет
`Command.Forced = old || incoming`; прочитана вместе с regression true→false
без downgrade и дальнейшим resume. **SOURCE APPROVE**; author focused PASS,
независимый полный race выше был до этой tiny delta. Binding36 выше её не включает.

QA v2 на checkpoint615ddf0 дала9 промежуточных PASS, включая actual stdio
archive-snapshot и offline deadman/reconnect, затем FAIL на новом worker после
локального consent revoke: прежняя30s host lease ещё ограждала old incarnation.
Это корректный server fence; weakening/takeover не одобряются. Предложенное
initial-only bounded acquisition ожидание35s концептуально APPROVE при local
noactive/no-unknown/no-pending и без lease renewal/launch. Source fix ещё PENDING.
Весь v2 run не PASS: есть failure и source drift; требуется final repeat.
Небольшой отдельный browser/native bootstrap wrapper и перевод enroll/consent
на async fixture child (proxy остаётся отзывчивым) прочитаны: test SOURCE APPROVE.

Initial-acquisition runner и7-case HTTP regression после реализации прочитаны:
**SOURCE APPROVE**. До первого успешного poll разрешён только bounded monotonic
35s retry409 с новым request ID, при noactive/no-pending и исключительно terminal
local runs. После первого успешного poll409 сразу fatal;401/403 также сразу fatal.
Ни deadline, ни epoch, ни launch не создаются ожиданием. Старый live/ambiguous
владелец не теряет server fence. Независимый focused race worker initial acquisition
+ expired recovery/request-incarnation tests **PASS**. Late test SIGINT_IGN с
actual applied forced-stop assertion прочитан и APPROVE; detached/unrelated
границы сохранены. Новая private candidate и финальная QA evidence ещё PENDING.

Следующая QA выявила функциональный gap: fresh bridge при native resume создаёт
новую board session, но lease/singleton прежней managed session ждут TTL. Узкий
terminal handoff design APPROVE до реализации: при authenticated positive-exit
completed/stopped/failed/recovered_stopped atomically закрываются только server-
bound same managed run/current generation/host-owner sessions и их exact task
leases с finishAttempt/released audit. Task fencing не сбрасывается; новый Next
проходит обычные gates и увеличивает generation. Ordinary EOF, unknown и чужие
sessions/leases не меняются. Author source/tests ещё PENDING; native acceptance
должна доказать новый task claim после resume, одного conversation UUID мало.

QA v3 на immutable0b05610/source unchanged дала8 промежуточных runtime PASS
(включая forced stop), но offline stop стал needs_recovery/ownership uncertain.
Общий FAIL сохранён; причина ещё расследуется. Unknown не является stopped и не
может освободить board leases новым handoff. Финальный QA gate требует review
будущих fixes и повтор; предыдущий source approval не является acceptance.

## Diagnostic/handoff source re-approve после0b05610

Прочитаны managed_terminal.go и два новых author regression файла: все4
positive-exit phases release exact attempt+expire exact managed sessions;
ordinary/foreign/othergen/otherRun/done сохраняются, mismatch rollback атомарен,
old writes fenced, fresh singleton/Next немедленно получают новые generations,
historical terminal replay не освобождает новый claim. **SOURCE APPROVE**.

Прочитаны private allowlisted diagnostic codes, nonce-bound helper field,
generic Error()/errors.Is, сохранение stage только в local journal; Receipt/UI
его не получают. PGID/intent fail-closed поведение не ослаблено. Concurrent
stopBatch устраняет последовательные grace delays между owned groups; barrier
regression требует входа всех Stop до ожидания и сохраняет каждую uncertainty.
**SOURCE APPROVE** diagnostics/concurrent stop, но причина intermittent Unix
failure ещё UNKNOWN: это разрешение диагностической QA, не исправленный runtime.
Независимый focused race store/worker новых regressions PASS; cmd compilation
PASS, matching test cases N/A. Diff check PASS. Binding15 Go files поверх
`0b05610167e27d3bf556e59960c0d9f8b6ae1bd4`:
`2339bc9db04d83f1a12c7b41930a3c6a74213654b3fadfa1ec5d4fe89676e733`.

Actual browser v3 evidence прочитана: immutable SHA737793070b90a1fe…,
revision0b05610, source/binary unchanged true,5checks/8screens PASS. Просмотрены
четыре pixels light/dark + desktop/mobile + list/detail; blockers не выявлены.
Tiny test selector changes anchored combo names/#managed-message APPROVE,
поведенческие assertions сохранены. Это actual Chrome/Next с fake provider;
native Codex/Windows/второй ПК не подтверждает. Отдельный native v3 startup FAIL
на disabled MCP transport остаётся открытым до source fix/review и actual rerun.

## Native adapter follow-up: transport scope approved, concurrency P1 открыт

Полные matching stdio/HTTP disabled tables, отсутствие raw original credential
URLs в argv, reserved HTTP-name rejection и обязательная config-only проверка
единственного enabled managed connector/exactself+attemptargs прочитаны вместе
с5 unit regressions. Auth/sandbox/approval/rules настройки не обходятся.
Но **REQUEST CHANGES** для combined adapter: Ack→apply→launch держит глобальный
e.mu во время двух blocking native inspections. Watchdog/StopAll других активных
workspace не может взять mutex; если prior report backlog съел lease budget,
inspection может удерживать живые процессы послеdeadline. Последняя проверка
перед новым spawn этого не исправляет. Нужны blocking prep вне mutex и повторная
проверка authority/state/pins после reacquire; barrier regression должен доказать
Stop старого active до завершения blocked inspection и отсутствие нового spawn.
Root/164 уведомлены, fix/source re-review PENDING.

Новые native.cjs/bridge.cjs handoff tests SOURCE APPROVE: marker не содержится в
prompt и читается из synthetic git checkout; initial ended attempt подтверждает
actual Next, proven terminal handoff; resume требует same provider UUID, нового
board session и большего fencing same task. Отдельный actual stdio сценарий
проверяет ordinary EOF retains lease, затем proven stop release и immediate
reclaim после profile archive. Выполнение новой native версии ещё NOT-RUN.

Root docs TASK013 matrix/WORK167 local-vs-server deadline/guide manual2PC
приёмка APPROVE после уточнений: возвращается лишь незавершённая задача, новый
claim не гарантирован вне обычных gates. Исправленный guide не выдаёт будущую
Windows/двухПК проверку за сделанную. Дополнительно независимо прочитан final
updater v3 `update-1792701762378298000/results.json`: PASS3, revision0b05610,
SHA737793070b90a1fed579ef090a39389be45e884a30b9360d4cf9188e4c07905d,
source_unchanged/binaries_unchanged true. Это тот же binary, что browser v3;
evidence APPROVE в заявленных границах реального Go/SQLite/snapshot/auth-reset
и симулированного systemd/Next. Первый updater6a52 больше не единственный proof.

Combined adapter/preparation **SOURCE APPROVE** после fixes: blocking inspections
идут по deep immutable config/run snapshots вне mutex, затем повторно проверяются
closed/fatal, stop epoch, exact run/state/gen/session, deadline, controller/config
и pins. Barrier regression доказал Stop другого active при blocked verification
и отсутствие deferred spawn; snapshot alias test прочитан. Независимый focused
race PASS, затем полный worker/cmd race PASS. Quoted CLI path probe обнаружил
отдельный functional FAIL: literal quote names оставляли originals enabled.
Второй inspect это корректно отвергал. Финальный source использует один whole
`mcp_servers={quoted TOML keys...}` override, transport-aware full tables и ту же
strict exact-managed/all-others-disabled проверку. Последний full worker/cmd race
и diff check PASS; binding24 Go files поверх0b05610:
`e924cb9c999c1b2636d3570c96ed9c82047b1a172f600d10d24704edaba6a06e`.
QA разрешены exact new-shape config-only probe и truthful fixture delta с её
отдельным review. Actual native и intermittent Unix unknown пока не закрыты.

Новая QA fixture delta SOURCE APPROVE: provider действительно разбирает leading
-c/TOML whole table, имеет injected stdio+HTTP inventory, требует полного exclusive
transport и логирует resulting entries. Main проверяет sole managed exact binary/
attempt args, all others disabled и output truncation label. Новый concurrency
сценарий запускает две owned группы в одном worker, SIGINT игнорируется; обе
должны прекратить heartbeat до30s и после связи получить forced terminal proof.
Failure summary содержит только private state/closed stage codes. JS/Python
syntax независимо PASS; actual новый process/native run ещё PENDING.
Exact wholemap config-only native proof прочитан: PASS6 entries, один enabled
managed/exact args и все остальные disabled. Этот probe не запускал model и не
проверял permission/task flow. Guide4 строки про отдельные working copies и Git
APPROVE: workspace alias не обещает синхронизацию файлов между компьютерами.

V4 main остановлен на fixture TypeError: корректно закрытая managed session
отсутствует в sessions default active-only list. Предыдущий source review это
упустил. Tiny test fix bridge/native `include_closed:true` + явный old-row assert
сверен с настоящим Sessions/ListInput: SOURCE APPROVE, поведенческие assertions
не ослаблены. Actual updater v4 evidence PASS3:
`update-1792703868984271000/results.json`, revision85fbc50e59a22ecac572a8261ebe6f685d6e2a9c,
binaryfa303a77cc88636ee8cc4c5b11af31fab5324c87c587687d6e61ffc74a0000c0,
source/binaries unchanged true. Final main/native/OS-stress повтор ещё PENDING.

V5 main actual evidence **APPROVE: PASS16**:
`run-1792704142678/results.json`, revision3ea1f8b16ca7bb81d6a2639f67b9f2d78fdc2f1a,
SHA7d2864cc488d573463c69f9670100c989a21bf5915c4f8319977592d57a864ca,
source/binary unchanged true. Private binary hash независимо пересчитан.
Дополнительно прочитаны actual SQLite/local journals: same-task stdio handoff
ready/noLease,2 закрытые managed_stopped attempts с generation1→2/разными board
sessions; один worker завершил обе parallel группы, forced отражён во всех3
соответствующих commands. Crash остаётся needs_recovery, сохраняет local intent;
stopping у quota-only synthetic run не выдаётся за процесс. Это deterministic
provider + actual workers/HTTP/SQLite/stdio, не native Codex/Windows. Одного PASS
недостаточно назвать прежнюю intermittent Unix unknown причину исправленной;
root/QA планируют повтор и отдельные native/browser/updater на том же candidate.

V5 browser `run-1792704399407/results.json` **PASS5/8screens**, updater
`update-1792704408397229000/results.json` **PASS3**: тот же3ea1f8b/7d2864…,
source/binary unchanged true, evidence независимо прочитана; два latest pixels
desktop-light/mobile-dark просмотрены без blockers. Границы fake provider и
systemd/Next simulation сохранены.

V5 actual native Codex0.163.0-alpha.11.1 **FAIL**:
`run-1792704338811/native-result.json`. Transport startup проходит и native turn
завершается, но README marker не получен. QA зафиксировала native MCP approval
denial при policy never; SQL независимо подтверждает0 managed board sessions,
0 task attempts/leases. Native completed не означает functional acceptance.
Silent broad approval/sandbox/rules bypass не одобрен. Root исследует только
default-off narrow local per-tool consent proposal; без human ответа применять
его нельзя. Это не закрывает existing native FAIL и потребует source/QA review.

## Финальное evidence review v5

**EVIDENCE APPROVE в заявленной границе**: main suite трижды PASS16 на одной
версии3ea1f8b16ca7bb81d6a2639f67b9f2d78fdc2f1a/SHA7d2864…:
`run-1792704142678`, `run-1792704425213`, `run-1792704614479`/results.json.
Независимо пересчитаны все3 private binary SHA; все stored source inventories
сравнены с actual files — drift0. Source/binary unchanged true, cleanup failures
отсутствуют. В каждом local journal обе SAME-worker parallel группы stopped,
private ownership diagnostic отсутствует; отдельно ожидаемый crash остаётся
needs_recovery. Browser5/8screens и updater3 выше на той же версии подтверждены.
Три прохода подтверждают повторяемый успех этих сценариев, но не устанавливают
причину прежнего intermittent unknown и не отменяют исторический v3 FAIL.

Actual native workflow остаётся FAIL на approvals; Windows/Claude/второй ПК и
production NOT-RUN. TASK013 не complete. Новая default-off local optin до
реализации требует official API proof/source review; её применение к реальному
native требует отдельного явного human consent. Existing v5 native FAIL этим
не переименовывается в PASS. Проверяющий менял только WORK165.

## Default-off native permission option source review

Новая `native_mcp_approved_tools` реализация/tests/docs **SOURCE APPROVE**.
Только локальный config, configured codex/claude, exact unique известные12 имён;
unset/null/empty не создают override. Codex получает только fixed managed
connector selected enabled_tools, default prompt и явные selected approve/
nonselected prompt. Claude — exact managed allow names и deny остальных известных
bridge tools. Global approval_policy/sandbox/auth/builtin permissions не меняются;
новая map глубоко копируется перед unlock в immutable preparation snapshot.
Браузер, commands и profiles это поле не задают. Author unit прочитаны;
независимый полный worker/cmd race **PASS** (5.1s/3.1s), diff check PASS.

Применена инструкция OpenAI Docs, independently fetched official
[Codex MCP](https://learn.chatgpt.com/docs/extend/mcp): per-tool approval_mode и
approve поддержаны. Также fetched official [Claude CLI](https://code.claude.com/docs/en/cli-reference)
и [permission names](https://code.claude.com/docs/en/permissions) для exact rules.
Disabled synthetic installed parser evidence прочитан: approve accepted,
invalid enum rejected; get показывает выбранные enabled_tools, list их скрывает,
effective approval policy оба не показывают. Runtime introspection не заявляется.
Docs native-agent-permissions/guide/contract read, границы source APPROVE.
Рекомендовано уточнение про остальные model MCP calls: существующий автоматический
heartbeat bridge при живой сессии не становится отдельным новым native grant.

Это source approval безопасного выключенного по умолчанию механизма, не согласие
на его применение. После предыдущего actual denial реальные model/MCP вызовы
с grant до явного human ответа не авторизованы. Root готовит конкретный scope
только describe+next для synthetic read-only QA. Existing v5 native FAIL остаётся
в отчёте; новый native result, Windows и второй ПК ещё не PASS.

QA follow-up native flag/README **SOURCE APPROVE**: только exact
`ACCORD_TASK013_NATIVE_ALLOW_DESCRIBE_NEXT==='1'` создаёт fixed local Codex
describe/next grant; любое другое значение оставляет поле absent. Список tools
из environment не принимается, result фиксирует scope. README отдельно требует
human consent, называет next write и ограничивает synthetic read-only checkout.
Node syntax/diff check PASS. Это не является human consent: actual execution с
flag не разрешена, пока человек не ответил. Guide/permission doc уточняют, что
автоматический bridge heartbeat существующих leases сохраняется отдельно от
model MCP tool grants. Новая v6 full root check PASS сообщена root; actual
runtime новых permission настроек пока NOT-RUN/PENDING, старый native FAIL сохранён.

## Default-off candidate v6: final runtime evidence

**EVIDENCE APPROVE** revision798e689efc38761c5b4ec377e59b627edbe04d54 /
Mac SHA4177b6f4eefa5d27584e2e1b5ca56a73a1dcfb47014d603bf40275daa9b13334:
main `run-1792705219055` PASS16, browser `run-1792705429405` PASS5/8screens,
updater `update-1792705220314847000` PASS3. Три actual private binary SHA независимо
пересчитаны, stored source inventories сравнены с current files — drift0;
source/binary unchanged true, cleanup failures нет. Latest mobile-light list и
desktop-dark detail pixels просмотрены без blockers. Во всех4 main config новый
grant field отсутствует; обе parallel группы stopped/diagnostic0, ожидаемый crash
сохраняет needs_recovery. Native execution с grant v6 NOT-RUN: human ответа нет,
historical v5 native FAIL сохранён. Windows/Claude/второй ПК/production NOT-RUN.
Source и default-off runtime подтверждены, TASK013 complete не объявлен.

## Curated delivery review

**PACKAGE CONTENT APPROVE**, не production/native acceptance. Прочитаны
manifest-798e689.json, SOURCE/README, оба JSON templates и latest TASK status.
Оба ZIP CRC PASS, actual SHA/bytes совпадают с manifest:
Mac `b2b2b018a4c093d9a84969d5afebf3fce91a1270f222ee3831c799fca4672d12`,
Windows `ba4084f93e0604a61a3760fcabdb77aae0c2ec8f4176f55e1d57aa45d91dd019`.
Все10 entries каждого ZIP совпадают с curated files byte-for-byte;9 CHECKSUMS
пересчитаны для каждого набора. Нет absolute/traversal archive paths, все relative
Markdown links разрешаются. Binary headers независимо подтверждают Mac ARM64 и
Windows PE x64; SHA совпадают с SOURCE/manifest, Mac — с actual v6 QA binary.
Windows binary SHA7059b925b8ea9e7dffca8105c3eaf251ba13082d80917008111be9a21aa2efb0.

Templates JSON parse PASS: placeholders origin/project/token/path/machine,
controllers empty, native grant отсутствует; decoded Windows separators ровно
по одному backslash. Known private origin/path/project/token-pattern scan
curated text files чист. SOURCE честно сохраняет candidate/incomplete,
v5 native FAIL, v6 grant NOT-RUN/human pending, Windows/второй ПК/production
NOT-RUN, Unix PGID boundary и недоказанную исходную flake cause.
Latest TASK table/changed areas/read-only connection guide APPROVE. Человеческого
ответа на grant нет; новый native запуск не выполнен и не одобрен этим review.
Reviewer изменял только WORK165; общий TASK, implementation и delivery files — root.

## После явного разрешения человека на Mac synthetic native test

Root передал прямой human ответ «Разрешить этот тест»: только describe/next,
synthetic read-only README и exact session resume. Один authorised actual run
`run-1792705928065/native-result.json` на прежнем798e/4177… overall **FAIL**:
marker отсутствует. SQL независимо подтверждает1 managed board session,
1 task attempt gen1 с managed_completed reason и ready/noLease после proven exit.
Output говорит об only describe/next и отсутствии file-reading operation, а не
policy never. Native grant работает в этом partial scope, resume ещё NOT-RUN.

Найден fixture prompt conflict: QA profile «Use only accord_managed» запрещал
модели builtin file reading, нужное для README. Узкий one-phrase QA test diff
явно ограничивает accord_managed координацией и разрешает прежнее builtin
read-only чтение README. **SOURCE APPROVE**, independent Node/diff checks PASS.
Никакие native rights/sandbox/MCP grant/product source не расширяются. Имеющийся
human scope уже включает это чтение; root разрешает один новый private attempt
с тем же binary и отдельным test revision/hash. Прежний partial FAIL сохранён;
новый actual result и resume ещё PENDING. Windows/production вне разрешения.

Новый authorised actual native result **EVIDENCE APPROVE: PASS4**:
`run-1792706107727/native-result.json` и wrapper results PASS. Настоящий Codex
CLI0.163.0-alpha.11.1, product798e689/4177…; QA test commit
cc4b17539b38902d43d09731e6ec28d9ff1f8af1, native.cjs SHA
0da1396e7bc87552dcdf3f3c9d21cbe69b24eb2936b42ba3e5ee738bd606646f.
Private binary SHA пересчитан, stored inventory/current files drift0,
source/binary unchanged true. Config содержит только codex describe/next grant,
sandbox read-only, provenance соответствует human scope. Git status synthetic
checkout независимо проверен — чист, неотслеживаемых/изменённых файлов нет.

Actual SQLite:2 managed board sessions разныхID для generations1/2; одна и та же
task имеет2 ended managed_completed attempts с fencing1→2 и разными sessions,
ready/noLease. Start/resume execution commands applied. Оба private completed
receipts gens1/2 содержат один идентичный provider UUID, совпадающий с final run.
Random marker из README и QA_NATIVE_RESUME присутствуют в native output.
Подтверждены настоящие Describe/Next/read-only marker/точное продолжение с новым
task claim и обоими terminal handoffs. Historical denial/fixture FAIL не
переписаны. Этот PASS ограничен Mac synthetic read-only scope; реальный Windows,
второй физический ПК и production не проверены. Новые R2 delivery docs ещё ждут
отдельного content review; product binary остаётся прежним.

R2 final curated delivery/docs **CONTENT APPROVE**: actual ZIP SHA
Mac84dedcf1529aec47e79a6784c3113d6d763c27eae5cef2a79d2eaf67f557f02d,
Windowsce7779fadfdde57eacb4b2b0c719a54ec1ee52f89b2f8521c7f323a34c7bf8da
и bytes совпадают с manifest-798e689-r2.json. CRC,9 checksums/10 entries каждого
архива, byte equality curated folders, безопасные relative paths/Markdown links
и known-private pattern scan независимо PASS. Product binaries прежние4177/7059;
templates сохраняют placeholders/controllers[]/grant absent, Windows separators
декодируются по одному. Оба immutable R1 ZIP SHA повторно совпали с прежним manifest.
SOURCE/TASK/WORK167 точно отделяют product798e от QA-onlycc4, ограниченный Mac
native PASS4 после human scope от historical FAIL и Windows/двухПК/production
NOT-RUN. Ранее неизвестная flake cause не переименована в исправленную. Candidate
и TASK остаются incomplete. Reviewer менял только WORK165; дополнительных native
запусков или разрешений/production действий этот verdict не добавляет.
