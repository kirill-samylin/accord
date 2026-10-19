# WORK-115 — независимое ревью control QA и выпуска

- Родитель TASK-005 / REQ-005; reviewer `/root/control_testreview115`, отдельный от авторов/tester107.
- Статус: **выполнено — APPROVE** final E2E source, clean release packaging и frozen deployment operator/stage. Блокирующих замечаний нет.
- Область записи: только этот WORK и private `/private/tmp/accord-control-099/review115/`. Source/tests/infra/Git/SSH/live не изменяются.
- Прочитаны reviewer profile/общий процесс/parallel-work/code-structure Давинчи read-only и AGENTS самостоятельного Accord. Tandem OFF по прямому указанию пользователя; успешный claim не заявляется. Именованные модели Claude не закреплены, унаследованная модель равна авторской.
- Production review112 и visual114 самостоятельны; этот reviewer проверяет новые browser/process tests и release packaging, а не заменяет их заключения.

## Начальная проверка

Проверены6новых `tests/web/control*.cjs`, diff `browser.cjs`/`harness.cjs` и README. Assertions используют реальные Go/native/Chrome, synthetic token revoke, strict400 cursor/filter, реальный401/409 и real SIGSTOP/timeout/recovery. HTTP/domain responses и document.visibilityState не подменяются. Clock ускоряет JS timers; отдельный409 deliberately достигается с clientDate60s behind при настоящем GoTTL2s. В реальной visibility проверке нужны actual hidden/visible, иначе FAIL. Полный14 набор reviewer не повторяет; после freeze воспроизводит2bounded scopes.

Все новые файлы40–108строк, изменённые browser229/harness96; общей fixture и harness достаточно, новых существенных дублей/необоснованных абстракций/unsafe typed casts нет. UI-kit Давинчи к отдельному Go SSR сервису N/A. Static request failures сохраняются в evidence; отсюда не делается общий network PASS без чтения итогового JSON.

Private remote operator совпал с frozen SHA256 `3c934e54af980dccad37878dcc980a939f9e66e532287199e64664830aefb697`. Driver относительно прошлого проверенного `/private/tmp/accord-web-084/deploy/driver.py` отличается только BASE evidence directory. Алгоритм не менялся: manifest/backup/maintenance lock/stop writer/atomic current/readiness/pending smoke/finalize/rollback без восстановления БД. Driver `transfer-stage` содержит прежний platform tar; для текущего выпуска координатор использует отдельный exact USTAR transfer, его доказательство ещё ожидается. Apply/finalize этим reviewer не запускаются.

## Ограничения и передача

Итоговый approve ограничен проверенными source/tests/package/stage; actual apply/finalize и HTTPS smoke остаются у координатора. Физический второй ПК, production HTTPS smoke, actual deploy/rollback — NOT-RUN reviewer. Loopback на одной машине этого не подтверждает.

## Проверка frozen evidence и bounded subsets

Самостоятельно сверены214source hashes `source-freeze-batch.json` SHA256 `451e6888353900b1ba8016bdf2830c185b961b892d2749bb6fc83d10bae50a1c`, mismatches=[]; QA binary `9efa45893de16e369d3a12bd21add9ca9793f891993f229a5af0539250f25a27`.12test files final manifest SHA256 `f02f0e3965e1b2793f115c920d90ae2b808a24eb9ad0aec1efd5ff5b488824c4` совпали. Summary evidence SHA256 `7c526154a63492e9ab86537146a286109653f4fd2aa6460f0c1d0ecfde08482e`:14legacy PASS,5data PASS из сохранённого overallFAIL на последующем selector,5targeted layout/poll PASS. Все accepted entries действительно присутствуют как PASS в исходных JSON; hashes совпали. Browser/harness/lifecycle/state-hash в полном14run совпадают с текущими source.

Reviewer independently запустил data-only:5PASS, источниктотже, server/native/Chrome закрыты; `/private/tmp/accord-control-099/review115/data/control-results.json`. Негативные400 filter-bound cursor и XSS/literal поисковые assertions сохранены; реальный credential revoke уменьшил effective sessions/lease без изменения workflow task.

**Возврат R115-01, test reliability:** poll-only прошёл первую realGo transport/offline/timeout/retain/no-renew группу, затем actual visibility helper упал `clock.pauseAt: Cannot fast-forward to the past`. Date.now брался отдельным browser roundtrip, а clock продолжал идти до pauseAt. Это race тестового стенда, не подтверждённый product defect; исходный FAIL сохранён `/review115/poll/control-results.json`. Tester107 получил узкую доработку4clock sites; reviewer source не правит. Новый freeze/retest ещё обязательны.

## Packaging и stage

Самостоятельно проверен clean detached release `ac6c965f3391822b803114efde2a7dcb757ed38c`: git clean,214application files exactQA source; SOURCE/SHA256SUMS строгий набор27files без symlink/extras. Linuxamd64 CGO0 Go binary содержит VCS revisionac6 и modified=false. Binary SHA256 `1db9fee0874c3709b3688bd2a31c4e6b7ac28525760f0a4b91de4102fdd4a9ce`; manifest SHA256 `4f79b2a9c4daad39008f138d41837b57d4c1e157e24005cc0022ba349136ab2f`. Это отдельный Linux binary с чистойVCS stamp, не Mac QA byteidentity.

Private `transfer-exact.py` проверяет manifest и путь, создаёт Python USTAR с27явными regular files, mode755 binary/644others иmtime0; не включает metadata/macOS AppleDouble. Remote post-transfer proof27files/hash совпадает. Реальныйtransfer/stage выполнил root; reviewer не делалSSH. Stage proof SHA256 `315d0209755fda29291ce3aca8ad38e4d339b033bb21d893da4b35fba706210c`: real nginx-t/systemd-analyzeverify=true, correctrevision; server.env hash совпадает с preupgrade; полный Serve и исходные Tandem Id/Image/StartedAt/RestartCount/Mounts совпадают сoriginal pre-web baseline. Stage live_changed=false. Post-finalize script прочитан: committed/readiness/snapshot strict permissions/timers/inventory/env guarded; actual run принадлежитroot и ещё NOT-RUN.

Private live smoke найденный stale login exact name исправленroot на текущуюпубличнуюподпись. Smoke не создаёт fake task/design, не отзывает real token; проверяет сохранённыйassetSHA, actual browser/native auth/read/control/mobile/themes/poll иlogout. Локальныйmac.token отсутствует, root выбрал существующий privatepc2token; runtime утверждение будет только credential-на-этомMac, без claims macaccess/physical2PC. Настоящий HTTPS smoke здесь не выполнялся.

## Закрытие R115-01 и итог

Tester107 вынес один `pausedClock(page, anchor)` в общую fixture: install фиксированного anchor−1hour, pauseAt(anchor) ДО загрузки timedUI. Четыреsites заменены; для login401/409/actualvisibility открывается свежаяblankpage. Moving browserDate roundtrip убран. Client409 всё ещё60sbehind, реальный GoTTL2s истекает по wallclock; HTTP/visibility/lease/draft/terminal assertions не ослаблены. Только3testfiles отличаются от предыдущегоfreeze, productsource214 неизменён. Все три перечитаны; новый helper переиспользуется, files<300строк.

**PASS reviewer affectedpoll-only4/4**, exact final clockfixed source, реальный Go/native/Chrome: offline/reconnect/one-inflight/timeout-retention/recovery/no-renew/no-draftrewrite; actual headedhidden45s0requests/foreground1Go200; real401/409 terminalstop/no-more-requests/protectedcontinue. Собственные processes закрыты штатно, static failures=[], cleanupfailure отсутствует. Сохранён исходный reviewer FAIL и separate fixedPASS, результат не перезаписан. Reviewer data-only5PASS сохраняется: новая Clockhelper в data-only не вызывается, exerciseddata/fixture successpath неизменён.

Final12test source manifest `qa/test-source-clock-fixed.json` SHA256 `3f9e0e2835fb68626328cd0425cc3cdfb0906483e35c4ce6a9e75dde97f55f8a` самостоятельно сверён. Сводка `qa/acceptance-evidence-clock-fixed.json` SHA256 `f85892406334575cf418c9d88e81a94c505583fe3dab7a245381b6d3355bb217` пересчитана:14legacy+5data+1layout+4newpoll=24actual acceptedchecks, соответствуют исходнымreports/hashes. OverallFAIL старого selector прогона сохранён и не объявлен полнымPASS. Finalappac6/QAmanifest451e/binary9efa совпадают.

| Reviewer evidence | SHA256 |
| --- | --- |
| data-only5PASS | `2c1e2399d3bf50d42349b52a8a46d699b7b7e27141b1542cd7f321f7ec04fb07` |
| originalpollFAIL race | `7795ec69d3155d81a80f686153a63f4d67a4eca69875bae36b8e43c2d4fc85f3` |
| fixedpoll-only4PASS | `87ce8d2a1770f9b0483ec101fbcdb8cac0409ab5d316704709564e1ddeee4d36` |

**APPROVE** позволяет root начать guarded apply clean releaseac6c965 с проверенными operator/driver и explicitUSTARtransfer, затем выполнить actual authenticated HTTPS browser/native smoke и finalize либо rollback. Apply/finalize/rollback/SSH этот reviewer не исполнял. LinuxbinaryVCSstamp/strictmanifest проверены статически, liveHTTPS и postrestart backup являются обязательным следующим этапом, этим approve не засчитаны. Физический второй ПК и same-owner browser concurrency NOT-RUN; реальный agentprocess launch вscopeне входит. Макетная pixelperfectclone Paperclip не заявляется.

- 2026-10-19 — reviewed: testsource/24QA evidence/214appsource/cleanrelease/stage parity проверены независимо; R115-01 закрыт targetedrepeat; APPROVE передан root. Source/tests/infra не редактировались reviewer, написан только собственный WORK.
