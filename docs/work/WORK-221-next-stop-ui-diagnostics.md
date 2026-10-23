# WORK221 — known Next stop и ограниченная UI диагностика QA

23.10.2026, root. Prospective CONTRACT APPROVE202; новый source/runtime NOT-RUN.
Основание: actual203V7 FAIL d11eeb49… на ui_team TimeoutError2 до native0.
Причина точного UI await пока UNPROVEN. Modes/bytes PASS;18supervisorcalls actual0,
Next actual130/proven/!forced/successfalse, browser0. StopIntent не записан:
исторический FAIL/manualtrue/configNOTOBSERVED не пересматривается.

## Два разных факта

Generic ownedSuccess, все одноразовые команды, агенты, worker, Go servers и
проверки остаются strict actual integer0 +proofsuccess +wrapper0 +known/drained/
!forced/noerrors. Ни 130, ни иной nonzero не становятся operation_success:true.

Только для собственного QA Next server разрешён отдельный **known closure**
KNOWN_EXPECTED_NEXT_STOP_130. Его executable SHA/Node, exact server.js argv/cwd,
Next manifest/build и копия pinned start-server.js связаны с source descriptor.
Настоящие Ready banner и HTTP readiness200 сохраняются до остановки. Pinned Next
код обрабатывает SIGINT через exit130; generic Unix supervisor Stop посылает SIGINT.

До EOF на собственном ещё live child handle, без прежнего proof/exit/signal,
сохранить durable private StopIntent (fsync file+directory), bind к nonce/command/
readiness/источнику/монотонному времени. После — wrapper0, nonce-bound proven,
!forced, typed exit130, successfalse, diagnostic absent, streams drained/noerror.
Сохранить actual130 и operation_success:false; причинность одного timing не
объявлять доказанной. Этот факт означает подтверждённое завершение, пригодное
для cleanup/config snapshot, а не успех бизнес-операции.

Pre-stop130, отсутствующий intent/readiness/binding, startup failure, другой
executable/argv/cwd/role,143/иной nonzero, forced/unknown/error/late writer — FAIL.
Нельзя сделать общий acceptedExitCodes или ослабить ownedSuccess. Старый UI FAIL
не превращается в PASS. Existing owned PGID/fencing и stop implementation не меняются.

## Диагностика следующего UI trial

До каждого await в login/open/team/role operations записывать finite enum stage.
При сбое, пока свой браузер ещё принадлежит harness: до5s общего остатка текущего
cleanup, без нового browser/model/HTTP business вызова, сохранить private0600
stage +sanitized own-source stack frames (file/line/column, без raw message),
finite DOM roles/labels/counts и viewport screenshot ≤5MiB. Все form controls и
potential raw-value blocks маскируются; никаких values/HTML/cookies/token/body/
headers/auth state. Network evidence — только route category/status/method,
без query/body/secret headers. Ограничить число и длину labels/records.

Capture before browser closure, bounded/joined, без поздних writers после seal;
невозможность capture означает NOT_OBSERVED, не успех. Диагностика не продлевает
setup180/workflow900/cleanup60 и не заменяет процессные proofs. Не запускать новую
модель ради диагностики; следующий full trial получает отдельный exact root GO.

## Source и review

gui_connection_qa owns новый203V8 и собственную WORK203, все старые freezes/claims/
evidence immutable. Pure tests проверяют positive exactNext closure и все negative
bindings/exit/intent/readiness cases; no actual server/CLI/model/settings в unit.
После source freeze независимый202 full delta review +pure cases, затем отдельный
root GO. Future F6/C2/H1/B2 adapters наследуют только exact reviewed helper; нынешние
их runtime gates закрыты. Product code, global settings, GUI/Test и production
не меняются этим QA контрактом. User scope — довести рабочий цикл и управление.
