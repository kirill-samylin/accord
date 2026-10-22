# WORK-165 — независимое ревью управляемых агентов

22.10.2026. Reviewer TASK013; область записи только этот WORK. Реализацию и
общую TASK ведут отдельные исполнители/координатор. Прочитаны AGENTS,
requirements/architecture, multi-project, agent-profiles, coordination, TASK012,
managed-agents и существующие store/transport/projects/browser hooks.

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
