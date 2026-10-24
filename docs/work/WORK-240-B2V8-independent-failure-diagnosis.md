# WORK240 — независимая диагностика FAIL B2V8

24.10.2026. Ops validation QA. **FAIL сохранён**, `gate_pass=false`. Только
bounded чтение собственных закрытых артефактов/source/БД. Не было новых native,
CLI приложения/Git, helpers, OS process queries, HTTP/browser/config/provider/
token действий, recovery, повторов или изменения данных.

Private0600 report `.local/task014/evidence240/review.private.json`:
`6fe4e6ae3cb954da86d063e8bdf552db5baebc8bea80b6c4da04cb88ada371c6`.
Read index149 файлов: `692554078ef4ab2d3d8cd166458fd753e8f93b9a29eda4c5f78e9f3f267bf3eb`.
Runtime `pipeline210-b2-1792799410583409000`, actual results SHA
`194534e11ecf9ca833d12b3d65255b90107f03fa4dbe99736923279480ec4dd7`.
Все94 source members manifest
`05498724cf09c7a71989861b316f4f5155fd93c407821d0bbb894fc9cb4fe14c`
перепроверены; source202/root GO и servera32/workerbe95 bindings сохранены.

## Точное место отказа

Первый несоблюдённый predicate — `observer.cjs:27`:
`assert.equal(out.stage,'OBSERVED')`. Записанный **до assertion** файл
`resume-argv-booleans.private.json` SHA
`e5ef0dcacef6ebc0dca854d20fbe7a043d4da7d955ab94359894f7adb823153e`
содержит `{stage:NOT_OBSERVED,resume_argv_exact:false,ownership_proven:false}`.
Actual observer log SHA
`e25aa91c861587a17aec8f12da610408bae2308f26d7b8a1813d6a1f99b588bf`
подтверждает тот же payload и owned supervisor proven/success/exit0/notforced.
Успешное завершение observer процесса **не доказывает** факт наблюдения argv.

Внутренняя причина NOT_OBSERVED не архивирована. Reviewed C observer объединяет
в этот результат несколько разных ветвей: пределы времени/числа запросов,
недоступность/гонки process metadata, несовпадение path/argv/identity и прочие
отказы. Нельзя выбрать одну из этих причин по имеющемуся payload. Никакого нового
diagnostic binary или расширения probe проверяющий не создавал.

## Что действительно произошло до этого

Два различных principals — local owner и catalog editor. Actual pregrant start
возвращает403/FORBIDDEN_ROLE без run. Только supported local owner consent
меняет controllers с owner на owner+remote. Поздний authority snapshot сохраняет
active policy/consent/host lease, действующих human sessions, owner/editor access
и can_control обоих; их registry memberships остаются active. Ни исторический
last_error, ни старый delivery_unknown не приняты за доказательство текущей потери
прав.

Remote UI pause принят. Actual преждевременный resume получил409/CONFLICT с
точным «Сначала подтвердите остановку», а не случайную stale-version ошибку.
Gen1 proven stop сохранён отдельно: native stopped, все пять ACK/release flags,
pending0. Start row сохранил исторический `delivery_unknown`; prior live-applied
witness и matching tuple квалифицируют stop без переименования этого статуса.

После gen1 остановки remote UI действительно отправил clarify и resume; CSRF
отличается от owner. Дошли до `resumed-373`: тот же managed run и native UUID
`01a16b2c-9967-73c2-b0e4-0b48787b56a2`, generation2/attempt2/continuation1,
два invocation/start, delivered_messages1. Actual вопрос содержит переданный
canary и «B2 resumed delivery», options `[Cancel trial]`; workflow awaiting_human,
epoch4, question1. Функциональные resume и доставка уточнения **не заменяют**
обязательное независимое OS-наблюдение exact resume argv.

## Гипотеза старого worker anchor

Source `coordinator.cjs:73–81` сначала завершает before-grant worker и применяет
local consent, затем присваивает новый after-grant Owned и только после этого
создаёт anchor. `observer.resume` перед вызовом проверяет тот же owned PID.
Actual anchor helper stdout совпадает с сохранённым tuple: PID41651/PPID41559,
UID501, birth1792799416.923092. Birth позже закрытия before-grant log и позже
local-grant receiptmtime; anchor file записан после birth. Оба worker incarnation
имеют positive owned exit0/notforced. Сохранённые доказательства не подтверждают
гипотезу устаревшего pregrant anchor. Это не раскрывает внутреннюю NOT_OBSERVED
ветвь observer и не является новым OS PID query.

## Финальное состояние и ограничения

Gen2 full stopped proof, actual old credential fence request, cached remote
revocation403 и финальная terminal cancel проверка **не достигнуты**.
После error-cleanup закрытая project DB: workflow stopping v13/epoch5;
reserved_runs0, но reservation row1/question1/lease1. Gen2 run stopping,
stop command queued. Durable journal pending1: native_exited/exit_durable=true,
managed_ack/lock_released/exit_ack=false. Полная native/business очистка
**NOT_PROVEN**, несмотря на исходные outer cleanup.complete=true/manualfalse.

Owner/remote UI и MCP exit0, raw generic supervisor proofs proven/success0/
notforced; Next отдельно known130/operation_success=false, все recorded joins
положительны. Эти факты относятся к внешним ресурсам. Project/registry DB открыты
только при отсутствии companions, immutable/query_only; SHA/identity до/после
совпали. Секретные колонки не запрашивались. Архивная config219 classification
сохранена как recorded evidence; новый полный её аудит не выполнялся, текущий
global config не читался.

Только synthetic same-Mac/two-principal контур. Физический второй ПК/Windows/
Claude/production не проверены. Старые отчёты, доказательства и текущий FAIL не
переписаны; mandatory argv guard, ACL и CAS не ослаблялись.
