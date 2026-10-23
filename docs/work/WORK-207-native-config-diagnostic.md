# WORK207 — одноразовая закрытая диагностика config RPC

23.10.2026, root. CONTRACT DRAFT / source prep pending review202.
WORK206V3: настоящий app-server запущен один раз, permission proof не получен;
owned exit доказан, model0, config/pins неизменны. Диалог скрыт продуктовым helper
как задумано. Нельзя предположением выбрать legacy/profile/protocol причину.

## QA-only диагностический контур

Product8c не меняется. Отдельный reviewed private QA broker запускается через
тот же candidate `worker supervise` generic mode. Broker и native app-server
остаются в одной owned process group; broker не выходит до остановки/reap своего
child. Parent lifeline и nonce proof candidate остаются прежними. Никакого
произвольного RPC/pass-through, model/threads/auth/tools/config writes.

Ровно один app-server с теми же native executable/environment/cwd и permission/
disabled-MCP overrides, что в206. Broker сам последовательно отправляет только
initialize → response → initialized → config/read(cwd,includeLayers) → response
→ configRequirements/read → response → stdin EOF. Проверяет exact IDs/errors;
server requests не обслуживает. Неизвестные notifications можно лишь сохранить
private и пропустить в пределах frame budget; ни одной ответной команды на них.
RPC errors/unknown reply IDs/malformed/oversize/timeout — fail/stop, без retry.

Native stderr также допускается только в отдельный exclusive0600 private file,
не более256КиБ; превышение даёт controlled fail/stop. Ни stderr, ни raw exception
строки broker не выводятся в stdout/stderr/report. Root читает через safe
projection известных error codes/types, не печатает содержимое. Это позволяет
отличить startup/protocol причины даже при отсутствии JSONL responses.

Для диагностики **raw bounded frames** записываются только в новый own private
0700 HOME root, exclusive0600 files. Максимум2МиБ/frame,4МиБ total,35 frames,
8КиБ исходящих fixed requests. Raw config может содержать secrets: никогда не
stdout/stderr/provider transcript/server/Git. Обычный вывод broker содержит
только фиксированные stages/counts/types и собственные exit facts. Чтение raw
root выполняет отдельно с минимальной safe projection (field names/types,
сравнение известного expected profile, без значений секретов/URL/аргументов).

Контекст только fixture: seed repo/свой .git, никакого GUI/Test repo/production;
те же бинарные/source/config before/after pins и natural owned exit proof.
Исходный native user config только читается/сохраняется private, не меняется и
не восстанавливается. Broker не меняет defaults, managed constraints или права
ради PASS. Этот capture не является принятым permission proof или native turn.

Общий execution budget90s +30s cleanup reserve, signal→finally/atomic FAIL,
exact live owned handles и source/binary checks before/after. Native RPC budget
20s + reserve7s не уменьшается. Не создавать отдельных child sessions/groups;
при partial spawn/unknown — manual FAIL, без ложного отсутствия процессов.
Собственный broker может завершиться успешно только после known native exit0,
drained output и неизменности inputs; policy/protocol mismatch всё равно
отмечается отдельно, не превращается в grant.
Broker exit не подменяет native exit: отдельно сохраняются actual Popen wait
status и stdout/stderr EOF/drain facts child, отдельно nonce proof/exit broker
и его supervisor. Неизвестный native status не считается natural0.

## Ownership / порядок

Только separate QA author198/206: private `.local/task014/qa207/`, этот WORK
дальше ведёт QA; root TASK/contract и202 review остаются раздельными. Сначала
202 CONTRACT APPROVE, затем QA sources,202 SOURCE APPROVE и exact root GO для
ONE config invocation. Компиляция новых Go helpers не нужна; approved candidate
supervise охватывает broker с его native child. Если надежное владение group
и parent loss не доказуемы, probe остаётся NOT-RUN до исправления источников.

После observed safe facts — узкое product исправление при необходимости,
source review/units/new candidate и повтор обычного206 proof по отдельному GO.
Actual commit/OS-negative/full pipeline всё ещё обязательны. Linux199V4-P2
PASS7 — отдельная storage приёмка, пока pending independent evidence review.

## QA source preparation after CONTRACT APPROVE202

QA автор: `.local/task014/qa207/` содержит run.py, broker.py, protocol.py,
raw_capture.py, pins.json, README. Каждый новый source≤300 строк. Только source:
compile/process/native/config RPC/SSH/GUI/production NOT-RUN. Нужны SOURCE
APPROVE202 и отдельный exact root GO, контракт не является разрешением execution.

Импортируются byte-exact pinned206V3 owned.py8c83d2b7… (generic framing fix),
budget.py375d004a… (90+30/signals/provisionalFAIL), fixture/inspection/config_snapshot;
206 sources и прежние FAIL artifacts не меняются. Candidate8c43df9/8726daa,
native9394a943, existing helper0d85bf8 и Python3.14bd349815 pinned. Fresh HOME
directory использует qa206-* prefix только из-за guard уже approved helper;
report WORK207 и новый каталог, старый runtime не переиспользуется.

Parent сохраняет current product/source/config before/after pins и exact raw
before/after config bytes0600. После обычного owned setup/identity и MCP5+5s
passes candidate generic Supervise запускает HOME copy broker modules. Native
Popen без new session/group, samePGID проверяется. Parent не передаёт произвольные
RPC. Broker фиксирует собственные response/notification counters, native actual
Popen wait code и stdout/stderr EOF/drain отдельно от broker/supervisor exit.

Raw files exclusive0600 с caps2MiB/frame/4MiB/35frames/stderr256KiB; unknown
notifications сохраняются private и пропускаются, server requests никогда не
получают ответ. Фиксированный dialogue20s + cleanup7s. Если native остаётся жив
или reader не drained, broker пишет FAIL/manual и quarantines без новых RPC,
не выходит добровольно с живым child; parent30s закрывает lifeline и candidate
останавливает ту же owned PGID. Root разрешил этот source-only quarantine plan.
Known final group exit не выдаётся за неизвестный native natural0.

Обычный broker stdout содержит только safe fixed enums/counters/exit facts.
CAPTURED означает сохранённые3responses + actual native0/drains, **не** принятый
permission proof. Любая ошибка/неоднозначность остаётся FAIL. Далее root читает
raw только отдельной минимальной safe projection. Source manifest фиксируется
до202 review; никакого runtime результата на момент подготовки нет.

## Actual single207 capture — FAIL, raw retained privately

После SOURCE APPROVE202 exact manifest6c5881ea… / pins5cf70ff4… root дал один
execution GO, он использован. Candidate8c43df9/8726daa/native9394/helper0d85/
Pythonbd349 unchanged. Начало17:11:52.948Z, окончание17:11:55.340Z (2.392s).
MCP5/5disabled, один broker и один actual app-server, models0. Повторов нет.

Broker same_owned_pgid=true. Ответ initialize получен (response_count1);
dialogue_stage=config_read, failure_code=unexpected_frame_shape. Два raw frames,
422 stdout bytes и183 stderr bytes сохранены только private0600. QA raw values
не читал и не выводил; root читает отдельной минимальной safe projection.
Не установлено, что именно в форме второго frame, до этого readback нельзя
называть missing profile, legacy setting или точную protocol cause.

Actual native Popen wait known exit0 после stdin EOF/cleanup; stdout/stderr EOF
и drain подтверждены. Флаг natural_native_exit0=false здесь означает, что normal
dialogue path не завершился, **не** неизвестный Popen status. Broker exit1,
candidate nonce exit proven=true/success=false/forced=false/childexit1, supervisor
exit0/drained. Unknown/manual inspection нет. Parent сохранил broker-safe.json
в results.json несмотря на generic Call assertion failure.

Все binary/source/config pins неизменны; private before/after config bytes
одинаковы. Config/global/GUI/production/Testproject не менялись, restore не было.
Последующие success-only fixture assertions NOT-RUN после FAIL. Это не принятый
permission proof и не native development/commit или OS-enforcement результат.

Private HOME qa206-1792775512947736000: results.json SHA
b95deddcb1f0ba56fd96d1bd2f9ec5b77e692671e5970b2079e1589d472b510c;
broker-safe.json SHA2d4b106ae6668404f62e3d0507bb60515e861900dcb4a0c5cab47cc8c7d445c1.
Executed source copies207 и reused206 сохранены, прежние206 FAIL не затронуты.
До отдельного source/GO никаких следующих native вызовов нет.

## Root safe projection / V2 parser source-only correction

Root прочитал только safe projection original207: initialize frame valid; frame2
notification remoteControl/status/changed со стандартными method/params и
emittedAtMs integer. Неизвестных extra keys0. Установленная официальная схема
ServerNotification.json:8809 задаёт optional emittedAtMs type=integer/format=int64;
SHA8e719512bc26bb3871fcf7b378b79abf912d82787609c0ad418b71f179d0883f.

По root SOURCE ONLY GO новая qa207-v2 допускает metadata только notifications,
type(stamp) is int (bool rejected), signed64 bounds; replies/server requests
не допускают это поле. Такое же правило после EOF. Unknown notifications
по-прежнему только private capture/ignore в пределах budget, без ответов/новых
RPC. Единственный executable delta — protocol.py; остальные source/pins кроме
нового protocol hash byte-exact. Schema reference сохранён для независимого review.
Старый FAIL и raw evidence сохранены. Product8c/config/permissions не менялись;
V2 НЕ запускался, нужны202 SOURCE APPROVE и отдельный exact root GO.

## Actual207V2 — CAPTURED, not a permission proof

После FINAL SOURCE APPROVE202 manifest672247aa… / pinsae849b7e… и отдельного
root GO выполнен **один** новый frozen run. Candidate8c43df9/8726daa, native9394,
helper0d85 и Pythonbd349 прежние.17:15:43.054Z→17:15:45.184Z (2.129s).
**CAPTURED**:3responses initialize/config-read/requirements +1notification,
4frames, outgoing349bytes. Raw stdout75791bytes, stderr0 — только exclusive600
private files. QA raw values не читал и не выводил; root анализирует safe projection.

Actual native Popen wait known/natural exit0, samePGIDtrue, оба потока EOF/drained,
capture_error=null. Broker exit0 и отдельный candidate nonce proof proven/success/
exit0/!forced, supervisor exit0/drained. Unknown/manual inspection нет. MCP5/5
disabled, fixture HEAD/readme/config/hooks/clean state сохранны, все binary/source
pins и native config before/after bytes неизменны. Models0, retries0, restore0.

Результат доказывает закрытый diagnostic capture/lifecycle, **не** accepted
permission proof, native commit, OS enforcement или full workflow. Прежние207FAIL
и206FAIL сохранены отдельно. До нового exact GO больше native вызовов нет.

Private HOME qa206-1792775743054756000: results.json SHA
cddd2f6555a723661899b693887869440a99e15da1fdda6ef2973fa7aff8abe7;
broker-safe.json SHAd4898d881dfe31e0985d4cbb811fa431a554a4ab4481cee35a5383361494ef19.
Executed source207V2 и reused206 копии сохранены; raw файлы доступны root только
для отдельно разрешённой минимальной safe projection.
