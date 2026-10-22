# WORK168 — независимая process/browser приёмка TASK013

22.10.2026. Отдельный тестировщик; координатор root. База
`d9f9eba6fdecb131e36f49907d421a2df43cccbd`, ветка `codex/managed-agents`.
Область после открытия gate: `tests/process/task013/`,
`tests/browser/task013/`, игнорируемая `.local/task013-qa/` и эта карточка.
Общую TASK, product source, GUI, production и Tandem тестировщик не изменяет.

## Gate

Прочитаны AGENTS, TASK013, managed contract/worker guide и WORK163–167.
Backend/worker ещё проходят independent review165. До явного сообщения
координатора **GATE OPEN** новые process/E2E tests не пишутся и не запускаются.
Подготовлены только матрица и чтение существующих fixture/tooling.

## Матрица после SOURCE APPROVE

- Изолированный loopback HTTP server/schema5 + Next и реальные worker процессы.
  Два synthetic principals, два hosts/workspaces на одном Mac: взаимный local
  consent, отсутствие control без grant, viewer/foreign project denial.
- Запуск native-like deterministic fixture: process-tree ownership, server
  accepted/applied/completed различия, bounded output и exact UUID continuation.
  Stop, resume, clarify на natural boundary/после stopped, ordered mailbox.
- Exact start/command/report retry без повторного side effect, stale version,
  leased worker epoch/no takeover и терминальная сверка.
- Worker SIGKILL с descendants: независимый lifetime guard; durable global
  workspace intent и restart needs_recovery без нового native launch.
- Unix containment отдельно проверяет принадлежащую PGID и detached/setsid
  fixture: обычная owned group прекращается, detached/unrelated process не
  затрагивается и очищается только собственным fixture handle. Полное произвольное
  дерево после ухода из PGID не гарантируется; workspace lock — кооперативное
  согласование worker, а не OS запрет на любые другие редакторы.
- Потеря control HTTP: monotonic deadman прекращает owned tree; offline не
  выдаётся за stopped до подтверждения. Local consent revoke и owner revoke.
- Заполненная ordinary mailbox/history оставляет отдельный reserve stop;
  неизвестные/подменённые scope и replay после revoke не создают запуск.
- Настоящий Chrome/Next: list/detail, start/stop/resume/clarify, delivery labels,
  conflict draft/retry, readonly controls, desktop/mobile/light/dark screenshots,
  отсутствие page errors/asset failures/horizontal overflow.
- Synthetic capable schema4→5 update всех проектов: настоящий old/new binary,
  all-project snapshot, installed guard before DDL, row/ACL/profile/dedup history
  preservation, capability floor; затем actual offline full restore и auth-reset.
  Restored hosts disabled, старые proofs не действуют, pending не auto-launch.
  Lifecycle systemd/Next boundaries existing updater fixture симулирует явно;
  SQLite/Go migration/backup/restore/auth-reset выполняются реально.
- Отдельно настоящий native Mac Codex в synthetic read-only checkout с прежними
  local approvals. Никаких bypass/autoapprove флагов. Native denial фиксируется
  явно, fixture не заменяет этот результат. Claude на Mac отсутствует.

Новые тесты и evidence передаются reviewer165. Изучен существующий
`tests/process/task012-update` и переиспользуемый task011 adapter. Новая
schema4→5/managed reset проверка будет в собственной task013 области. Production
updater/systemd/HTTPS сохраняется отдельным ограничением.

## Статус

**GATE OPEN** получен от root после independent SOURCE APPROVE165 реализации и
author tests. Созданы отдельные process/browser suites; source review165 идёт.
Candidate binary копируется в новый private run root и проверяется до/после;
тесты не исполняют меняющийся `bin/agent-board` под уже работающими worker.

Первые фактические наблюдения (промежуточные, не финальный PASS):
- Первый process run остановился на ошибочном имени operation в новом harness;
  `managed_hosts` исправлен на фактический `managed_list`.
- Второй run выявил product FAIL native version preflight даже для process
  fixture с exit0. Независимый diagnostic: Darwin WNOWAIT errno0; leader PGID
  сохранён, state5 zombie; сигнал пустой zombie group возвращает EPERM. Передано
  root/164/165, source fix и повторная проверка обязательны.
- Actual capable updater/schema4→5/full restore: первый PASS, 3 top-level checks,
  `.local/task013-qa/update-1792700794102780000/results.json`. Реальные old/new Go,
  семь DB/три проекта, old rows/ACL/profiles сохранены, floor до DDL отвергает old
  writer; actual reset отключает hosts/стирает proofs/отменяет dispatched commands,
  runs needs_recovery, memberships epochs+1, boot guard сохранён. Systemd/Next
  lifecycle в этом отдельном suite — явные simulation boundaries. Финальный
  повтор после candidate fixes запишет source/binary drift pins.

Windows runtime/native Claude и второй физический ПК **NOT-RUN**: cross-build и
две установки на одном Mac их не заменяют. Secrets/реальные origins/пользовательские
данные в tests/reports не записываются; локальные credentials только в private
ignored fixture файлах.

## Промежуточная фактическая приёмка

- Checkpoint `0b05610`, immutable candidate v3: process suite **FAIL**,
  `.local/task013-qa/run-1792701762262/results.json`, binary/source unchanged.
  До сбоя8 PASS: schema/floors, взаимный control/ACL, lost accepted+started exact
  retries с одним запуском, output redaction/chunks, mailbox/exact UUID resume,
  natural-turn clarification, forced owned PGID stop с nonce spoof rejection и
  нетронутыми detached/отдельным unrelated процессами, actual stdio pinned profile
  после edit/archive, host-proof denial. Offline group прекратила запись до30s,
  однако result стал needs_recovery из-за неопределённого ownership proof. Этот
  обязательный сбой не скрывается успешными частичными проверками. Автор164
  диагностирует private fixed stage codes после повторного source review.
- Standalone private helper diagnostic20/20: lifeline EOF, genuine nonce final
  proof proven=true, interrupted success=false, helper exit0. Не заменяет
  проваленный полный worker/offline сценарий.
- Actual capable migration/restore v3 **PASS3**,
  `.local/task013-qa/update-1792701762378298000/results.json`, old/new binary и
  test/operator source drift pins успешны; runtime границы описаны выше.
- Actual Next/Chrome v3 **PASS5**,8 screenshots,
  `.local/task013-qa/run-1792702249063/results.json`, source/binary unchanged;
  browser155.0.8059.86, Next build `W8qRjfzNWW2Khybb_CVtI`. Реальные start unknown
  response→exact retry, HTTP409→explicit newer version+draft, stop/resume,
  viewer/project isolation, light/dark desktop1440/mobile390. Нет page errors,
  failed static assets, router prefetch или horizontal overflow. Просмотрены
  реальные mobile light и desktop dark PNG. Первые два browser FAIL относились
  к слишком строгим новым test locators; selectors исправлены без product edits.
- Real installed Codex CLI `0.163.0-alpha.11.1` v3 **FAIL** до первого thread ID,
  `.local/task013-qa/run-1792702309939/native-result.json`. Exact invocation
  diagnostic с пустым stdin, без model call выявил `invalid transport` при
  enabled-only overrides discovered MCP connectors. Full valid disabled stdio
  entries разбираются CLI корректно (`No prompt provided`), source fix передан
  root/165. Native auth/MCP/next/resume пока не прошли; approvals не ослаблялись.

Отдельно обнаружено: новый attempt создаёт свежий board bridge, прежняя task
lease может удерживаться до TTL. Root/165 согласовали narrow authoritative
handoff только после proven terminal worker report; обычный stdio EOF и unknown
остаются без implicit release. После independent source approve тесты проверят
реальный next той же задачи новым session ID/большим task generation. Один
provider UUID без этой проверки не объявляется рабочим продолжением задачи.

## Проверка frozen v5, 23.10.2026

Independent165 SOURCE APPROVE получен для product fixes и всей новой QA source.
Проверен immutable candidate `3ea1f8b16ca7bb81d6a2639f67b9f2d78fdc2f1a`, SHA256
`7d2864cc488d573463c69f9670100c989a21bf5915c4f8319977592d57a864ca`.
Каждый suite копирует его в новый private runtime. Source/binary pins после
нижеследующих завершённых проверок совпадают. Production не изменён.

- Полный HTTP/worker process suite **PASS16**:
  `.local/task013-qa/run-1792704142678/results.json`. Помимо прежней матрицы
  проверен actual stdio terminal handoff: ordinary EOF сохраняет lease, proven
  stop освобождает её; тот же task немедленно получает новый board session и
  большую fencing generation после exact provider UUID resume. Две owned PGID
  под одним worker принудительно прекращены параллельно до30s при потере сети;
  обе после reconnect получили proven stopped. Одиночный offline, SIGKILL,
  restart/global intent и reserved stop также PASS. Private ownership unknown
  в этом запуске не возник. Из-за прежней intermittent ошибки выполнены ещё
  два независимых новых runtime повтора: **PASS16** в
  `.local/task013-qa/run-1792704425213/results.json` и **PASS16** в
  `.local/task013-qa/run-1792704614479/results.json`. Во всех трёх runtime
  binary/source unchanged; оба offline сценария proven, unknown не повторился.
  Это три фактических PASS с диагностикой, а не доказательство невозможности
  прежнего непостоянного сбоя. После завершения собственные3100/3101/3102 свободны.
- Actual Next/Chrome **PASS5**,8 screenshots:
  `.local/task013-qa/run-1792704399407/results.json`. Проверены exact retry после
  потерянного ответа, explicit409 conflict adoption с сохранением draft,
  stop/resume/message, viewer/foreign-project isolation, desktop/mobile и обе
  темы. Нет page errors, failed static assets или horizontal overflow.
  Визуально просмотрены actual detail390dark и list1440light: читаемы, overflow
  отсутствует. Process fixture в browser suite не выдаётся за native provider.
- Capable all-project upgrade/full restore/auth-reset **PASS3**:
  `.local/task013-qa/update-1792704408397229000/results.json`. Семь DB/три проекта,
  исходные rows/ACL/profiles/history сохранены; guard поставлен до DDL;
  restored hosts disabled/proofs erased/pending cancelled, auth epochs+1,
  immutable snapshot/live business history и boot guard сохранены. Old/new
  binary и source pins unchanged. Systemd/Next lifecycle симулируется явно.
- Настоящий Codex CLI `0.163.0-alpha.11.1` **FAIL**:
  `.local/task013-qa/run-1792704338811/native-result.json`. После actual valid
  isolated MCP configuration создан реальный conversation UUID, но native
  `accord_managed.describe` отклонён политикой CLI: `MCP tool call requires
  approval, but approval policy is never`. Ответ CLI честно сообщил отказ;
  synthetic DB подтверждает managed sessions0, task attempts0, task ready/no
  lease. README marker не прочитан, Next и native resume не проверены. Run
  completed означает окончание отказа и не засчитывается как native PASS.
  Никакие approval bypass/autoapprove/skip-Git или sandbox escalation не
  применялись. Синтетический checkout создан как обычный Git repo.

Предыдущий v4 общий process FAIL сохранён:
`.local/task013-qa/run-1792703868918/results.json`. Это новая fixture ошибочно
читала expired session без `include_closed:true`; tiny fix и явная проверка
наличия historical row независимо approved165 и включены в v5. Product logic
не менялась. Actual whole-map installed native MCP config-only probe PASS
сохранён отдельно в
`.local/task013-qa/run-1792702309939/mcp-isolation-wholemap-source-proof.json`:
единственный enabled connector — exact attempt-bound managed stdio, остальные
discovered connectors disabled. Это подтверждает config parsing/isolation,
но не заменяет проваленную native MCP acceptance.

Общий TASK013 **не завершён**: обязательный native scenario FAIL; Windows/native
Claude и второй физический ПК **NOT-RUN**. Production/schema5 capable delivery
в VPS этим локальным QA не выполнялась.

Reviewer165 независимо одобрил финальное v5 evidence: три PASS16, реальные
private binary hashes и source inventory без drift, cleanup failures отсутствуют;
browser5/8 и updater3 с тем же pin, native отказ/нулевые managed sessions и task
attempts подтверждены. Причина прежнего flake остаётся недоказанной.

Следующая отдельная product delta предусматривает default-off локальный выбор
native MCP tools. QA source подготовлена только для fixed opt-in
`ACCORD_TASK013_NATIVE_ALLOW_DESCRIBE_NEXT=1` → `codex: [describe,next]` внутри
нового synthetic native fixture; unset/прочие значения grant не создают.
Произвольный список из environment не принимается. README требует явное согласие
человека до применения и отмечает write-смысл `next`. На момент подготовки
ответа на вопрос о разрешении нет: actual native повтор с grant **NOT-RUN**,
v5 FAIL сохранён. Node syntax и diff checks PASS; source независимо approved165
и заморожена перед candidate-v6. Source approval не является human permission
на actual native execution с grant.

## Frozen v6: default-off повтор

Candidate `798e689efc38761c5b4ec377e59b627edbe04d54`, SHA256
`4177b6f4eefa5d27584e2e1b5ca56a73a1dcfb47014d603bf40275daa9b13334`.
Root сообщил полный makecheck PASS и Next BUILD READY. QA выполняется без
native opt-in, пока нет явного согласия человека и root GO. Первый завершённый
suite — capable update/restore/auth-reset **PASS3**:
`.local/task013-qa/update-1792705220314847000/results.json`, old/new binaries
unchanged и source unchanged. Полный process main **PASS16**:
`.local/task013-qa/run-1792705219055/results.json`, binary/source unchanged.
Оба offline proof сценария, SIGKILL/lifeline, handoff и reserved stop PASS;
ownership unknown не возник. У четырёх actual synthetic worker configs поле
`native_mcp_approved_tools` отсутствует: default-off сохранён.

Actual Chrome/Next v6 **PASS5**,8 screenshots:
`.local/task013-qa/run-1792705429405/results.json`, тот же binary/source pin,
drift отсутствует. Mobile detail390dark просмотрен, читаемость/overflow без
замечаний. Собственные services завершены, LISTEN на3100/3101/3102 отсутствуют;
повторный bind с SO_REUSEADDR успешен. Native повтор с новым grant остаётся
**NOT-RUN**, пока human permission и root GO не получены. Прежний v5 native
**FAIL** остаётся отдельным фактом; default-off process/browser его не заменяют.
