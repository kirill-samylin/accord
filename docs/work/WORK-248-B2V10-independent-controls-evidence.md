# WORK248 — независимая приёмка фактического B2V10

Вердикт **APPROVE / PASS** в узком контуре управления двумя участниками на одном физическом Mac. Отдельный тестировщик `ops_validation_qa` проверил завершённый runtime только чтением; новые native/helper/browser/HTTP процессы, провайдеры и изменения настроек не запускались.

Actual runtime: `pipeline210-b2-1792801588013379000`, root session67741 exit0. `results.json` SHA256 `09e0c64aad4b74c615132bea1edefbc7694e12118b64e5e3b153812a649b7b4d`. Source manifest `b2cc221cbdf1494722c106909732215e9aa0dc83d7e409ab4f313df3d308b6fa` — 99 файлов, все текущие хеши совпадают. Source APPROVE202 receipt SHA256 `031b5dfa9efc65a243d697b6c78f56574530e6fe6268c858921c5dde1824a12c`; root GO связан с тем же runtime/source/worker.

Server и QA outer: `a32db22973d5b43d4fb227a89337bf5b82a4969f`. Worker: `be95daed7153f4abeb72012286cc748d0596b479`, SHA256 `9755ed0fb0de869751635ba4e5adf1a4d742b0a202e6659e871d337fceed27ff`; сохранённые отдельные provenance/source202 совпадают. Исторические foundations не переименовываются в новую версию.

## Фактические подтверждения

- Отдельному remote editor без локального разрешения отказано настоящим 403/FORBIDDEN_ROLE, запусков нет. Затем только local owner consent изменил controllers 1→2. Четыре реальные UI-команды pause/clarify/resume/cancel имеют durable web actions и связаны через board sessions именно с remote human, отличным от owner.
- Два настоящих native turn одной conversation с двумя разными board session/lease generation 1/2. Обе остановки подтверждены matching run/gen/dispatch/UUID receipts, `run_exit` observations и пятью native-exit/durable/ACK/lock флагами. Исторические start/resume `delivery_unknown` не переименованы в applied; stop — applied.
- Exact OS resume observer V5: OBSERVED, ownership/same conversation/exact argv true, 1 poll/7685 queries; бинарник совпадает с pinned source202. Pre-stop resume отвергнут точным 409 «Сначала подтвердите остановку». Старая ранее действовавшая capability отвергнута после resume как 401/UNAUTHORIZED.
- Уточнение доставлено один раз и присутствует в настоящем вопросе агента; итоговая отмена сохранила тот же вопрос, отменённый вместе с human task. UI submit связан с owner human, настоящий stdio MCP replay вернул тот же workflow/intake hash.
- Owner revoke изменил controllers 2→1 и host version +1 без изменения automation/runs. Cached remote command получила публичный 403/FORBIDDEN_ROLE. Её durable web action сохраняет error, не успешный redirect. Немедленные API readbacks и exact no-mutation assertions source101–103 фактически пройдены до итогового check6/PASS; отдельные raw-пакеты этих двух readbacks **не архивированы**.
- Независимая закрытая DB corroboration: workflow точно совпадает с сохранённым cancelled snapshot после штатной profile redaction; managed run, все 4 commands и 2 attestations совпадают с сохранёнными stopped/final snapshots. Leases/open questions/reservations/journal pending — 0; workflow cancelled/epoch5, invocations/starts2. В cleanup policy revoked/version2, host disabled, 4 профиля archived, 2 catalog credentials и legacy credential revoked.
- 45 generic exits подтверждены как 0 без forced; Next имеет известный proven130, не operation success. Оба Chrome завершились 0 без signal/page errors, requests fulfilled; MCP 0/drained/no protocol failure. Все 47 cleanup steps и 52 final joins положительны, manual inspection false.
- Архивные before/after config bytes и metadata точно равны, shared219 `STRICT_UNCHANGED`. Текущие глобальные настройки, секреты и provider config не читались. Текущие source pins совпадают; runtime after-pins положительны, большие candidate binaries повторно не перечитывались.

## Особый режим закрытых DB

До первого SQLite connect обнаружены сохранённые WAL0/SHM32768. Root отдельно разрешил только readonly аудит этих **закрытых собственных синтетических** DB после положительного owned closure. Пять DB открыты `mode=ro&immutable=1` / `query_only`, все соединения закрыты. Для всех 11 main/WAL/SHM файлов до/после совпали dev/ino/uid/gid/mode/size/SHA256, набор файлов неизменен; WAL строго 0, SHM строго 32768. Companion-файлы сохранены, checkpoint/delete/repair/write не выполнялись. Это разрешение не распространяется на live или production DB.

## Замороженные артефакты и границы

Private report `.local/task014/evidence248/review.private.json`: SHA256 `745c4526e1c2851845ade18130f1d152c589f0ee34a87b590ac9219f53560ff9`.

Общий read index `.local/task014/evidence248/read-index.private.json`: SHA256 `dea063522d0b6496aa566f14988f489d1c72a8ad8608a81663b37d63ee41b98e`, 198 записей. Частные source/closure и business/native доказательства связаны хешами внутри итогового отчёта. Каталог 0700, файлы 0600, секретные поля в отчёт не переносились.

Remote cancelled screenshot визуально проверен: отмена, 2/2 запусков, аналитик attempt2 со статусом stopped, доставленный canary-вопрос. Хеши/размеры четырёх PNG 1440×1100 сохранены в source/closure proof.

Это controls-only проверка одного Mac/двух участников с настоящим Codex, а не полный business pipeline и не два физических ПК. Windows/Claude/физический второй ПК — NOT-RUN. Числовые native OS exits отдельно не архивированы; durable native/ACK/lock и OS argv доказательства указаны точно. Прежние B2V8/V9 FAIL и неизвестная внутренняя причина V8 observer остаются без изменений. Формальные production gates/GO пишет только координатор.
