# WORK188 — Standard/Interactive configuration-only comparison

23.10.2026. Отдельный QA; root координатор, reviewer169. Source preparation
по root поручению и169 PLAN APPROVE. Runtime **NOT-RUN** до независимого
SOURCE APPROVE нового runner и отдельного exact root GO.

## Основание и границы

WORK184 failure evidence APPROVE: Background job фактически spawn5, но helper
не начал Go test за100s; collection incomplete, manual-inspection flag остаётся.
WORK188 не меняет исторические184/177 reports и не повторяет Background.
Нужны fresh Standard и Interactive one-shot labels, по3 discovery/verification
samples actual Go helper, без модели, MCP tools/call, постоянного worker или
изменений GUI/production. Config-only inert `/usr/bin/false` binding сохранён.

Private root `.local/task013-live/qa/qa188-1792761973034483000/` содержит тот же
compiled helper `d48cacdef69e547d815080bb28ee84c743d7090f592b25bb0e0e52ec27593182`
и private Go test `653b37345971fcbccaba58425bcac2167dc0ecf9b2da3152f9de575e6b0459b5`.
363 owning source files pin к committed
`62e354525b83a2b671ab37ede938ffe7e838e24b`; текущие automation edits не используются.
Native5s deadline, bounded read, Wait/PGID и ownership fail-fast не меняются.

## Новая фиксация процессов

`run_classes.py`202 строки; новый `process_inventory.py`99 строк, только private
QA sources. Каждый poll читает top-level PID exact fresh launchctl label и
safe OS census: PID/PPID/PGID/UID/comm/starttime. Сохраняются только label-bound
helper и наблюдаемые потомки; полный census, args и environment не сохраняются.
Разрешён ожидаемый переход xpcproxy→exact helper. Unexpected executable, UID,
смена identity или отсутствующий PID во время census дают FAIL; numeric PID
signals отсутствуют.

Family identities отслеживаются и после выхода родителя, если были замечены
раньше. Census выполняется непосредственно перед bootout; postbootout до5s
наблюдает отсутствие известных identities и exact unique helper path.
Own label bootout выполняется и при ошибке prebootout census. Cleanup не
подменяет uncertain outcome: outer100s timeout/ownership suffix/missing report
останавливают все последующие contexts, finally сохраняет FAIL/manual flag.

`ps lstart` имеет секундную точность и не является kernel birth nonce. Census
доказывает только состояние ранее наблюдённых identities, не отсутствие всех
когда-либо существовавших потомков. Не выдаём sampled family за containment
произвольного дерева. Exact helper entry должен наблюдаться хотя бы раз,
иначе exit/bootout не дают PASS.

Сохранены прежние8 input pins и добавлен pin inventory module. GUI3316 baseline,
persistent service top-level running, native/config/plist/compiled/source/runner
hashes проверяются независимо; finally публикует report даже при cleanup FAIL.
Inherited operational plist fields прежние: EnvironmentVariables,
WorkingDirectory, ExitTimeOut15, ThrottleInterval; ProcessType отличается
Standard/Interactive. Лейбл/аргументы/логи/one-shot lifecycle собственные.

## Source-ready evidence

- `run_classes.py`: `66f0acde1d89dcc4e2a67ba00e28c87b714baf19b7c7911d712305d7cb34b351`;
- `process_inventory.py`: `a4c7e6e8937a0040327bf01fe78f252ffa61999a098cbcc7a3125c5dc0d750e0`.

AST parse обоих sources и363 source pins PASS. Helper/process execution
**NOT-RUN**. Не создавались LaunchAgents, не запускался native CLI/model,
не менялись persistent config/plist/worker и GUI. Runtime gate ожидает169 и root.

## Actual execution — FAIL, collection incomplete

Получены169 SOURCE APPROVE exact runner/inventory/helper hashes и отдельный
root GO. Единственный запуск 23.10.2026 начал requested Standard job
в13:29:24 UTC. На первом census exact fresh-label PID45238 с exact helper
comm/starttime показал UID0/PGID1; ownership assertion немедленно остановил
выполнение. Через41ms prebootout snapshot того же PID/comm/starttime показал
UID501/PGID45238. Это возможное переходное состояние launchd startup, но
причина не доказана; запрет на неоднозначную ownership не ослаблялся.

Runtime spawn type Standard ещё не был записан: исключение возникло раньше
фиксации этого поля. Samples/native configuration outcomes не получены.
**Interactive NOT-RUN**, повторов нет. Native model/MCP tools/call0.

Cleanup independently PASS: prebootout tracked helper сохранён, после own
label bootout exact helper и наблюдённая family отсутствуют; own labels absent.
Результат остаётся FAIL/manual-inspection из-за первоначального UID mismatch.
Numeric PID signals не применялись. Все9 input pins/source363 и GUI3316 baseline
unchanged, persistent worker top-level running. История184 не изменялась.

Private `comparison.json` в том же fresh QA root:
SHA256 `cc2b04fc6db6f51ffa7c58d1f2bc0c3ba003ce0e100688f172a34f1a78ed0eef`.
`result=FAIL`, `collection_complete=false`, `collection_failure=AssertionError`,
`requires_manual_inspection=true`. Safe process snapshots сохранены в
`standard/process-inventory-private.json`. До нового решения reviewer/root
дальнейшие process comparisons не выполняются.
