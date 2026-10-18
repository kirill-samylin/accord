# Интеграции и перенос — контракт E

18.10.2026; WORK-040 → [TASK-003](TASK-003-working-service.md).
Статус: технический план для реализации и независимого review, не runtime PASS.
Приоритет: [requirements1.1](requirements.md), [service-contract E](service-contract.md),
[config C/D](config-contract.md), затем исходник §11.7–14. OAuth отложен выбором
пользователя; визуальный UI ждёт ответа о макетах/простом SSR. Этот документ
не разрешает публичный ingress, импорт реального проекта или настройку каналов.

## 1. Владение и миграция

E начинается после интеграции D/schema3 и performance/schema4 по
[контракту053](performance-contract-053.md). Performance schema4 принадлежит
WORK057; E store-исполнитель будет владеть schema5, core DTO и gate evaluator.
Другой исполнитель не создаёт параллельную migration5.
Общий механизм D: private pre-migration backup, write-lock, повторная проверка
версии, все DDL/backfill/user_version в одной транзакции. Более новая схема
отвергается до DDL; binary rollback не откатывает данные. Backup/restore,
maintenance reset и installer schema compatibility обновляются вместе.

Минимальные таблицы (время UTC; счётчики int64, не float64):

| Таблица | Ключ и существенные поля |
| --- | --- |
| github_deliveries | delivery_id PK, event, body_sha256, config_version, received_at, outcome, reason; UNIQUE(event,body_sha256) через отдельную identity-таблицу при нескольких delivery ID |
| github_facts | repository, kind, object_id, sha, source_time?, fact_hash, data, delivery_id; UNIQUE(repository,kind,object_id,sha,fact_hash); immutable normalized versions, не весь payload |
| github_task_bindings | repository, pr_number, task_key, source_time?, status; UNIQUE(repository,pr_number,task_key) |
| notification_outbox | delivery_id PK, event_cursor, endpoint_id, config_version, url, secret_env, body, state, attempt, next_at, locked_until, claim_generation, last_code; UNIQUE(event_cursor,endpoint_id) |
| question_cycles | task_key PK, cycle, waiting_since, escalated_at? |
| import_runs | source, file_sha256, actor, config_version, applied_at, result; PK(source,file_sha256) |
| external_entities | source, kind, external_id, local_key, record_hash; PK(source,kind,external_id), UNIQUE(kind,local_key) |
| imported_history | source, kind, external_id, record_hash, data; immutable foreign history, не server sessions/events |

Существующий tasks JSON получает `completion_trust: local|imported_unknown`;
миграция ставит local для существующих результатов с сохранением прежней
истории. Только серверная completion-транзакция выставляет local новым
результатам. JSONL/ordinary patch не принимают это поле как полномочие.
Не экспортировать/импортировать credential hashes, active sessions, idempotency
keys, outbox jobs, секреты или действующие leases. Restore SQLite — отдельная
операция с reset; JSONL не является средством восстановления полномочий.

## 2. Конфигурация и интерфейсы

Сейчас `internal/config/types.go` и embedded schema не содержат repository
allowlist. В E добавить optional `github.repository_allowlist`: unique array
до16 строк `github.com/<owner>/<repo>`, ASCII owner/repo, ≤256bytes на строку,
без scheme/.git/query/fragment/port; lower-case canonical comparison. При
непустом github.events allowlist обязателен и содержит1..16 элементов;
отсутствие/пустой массив → CONFIG_INVALID. При disabled events отсутствие/[]
допустимы и означают отсутствие доверенных репозиториев, никогда wildcard.
Это additive язык config version1; старые binary
не понимают новое поле, поэтому config применяют только после upgrade.
`project.repository` уже существует в D как необязательная строка метаданных
(`internal/config/types.go`, обе JSON Schema). Это не allowlist и не полномочие:
сохранить обратную совместимость поля, не выводить из него доверие к webhook.
Единственный источник доверия — explicit `github.repository_allowlist`.
`require_approval_for_merge` только отражается,
не включает merge и не делает GitHub review серверным approve.

Secrets по существующим env-name полям; значения private env-file, не YAML.
Offline validate не читает env/сеть. Runtime при enabled-интеграции проверяет
наличие непустого секрета ≥32bytes; отсутствие → readiness false/startup error,
не fallback без подписи. Config apply проверяет runtime capability; изменение
allowlist/URL не пересылает старые pending jobs на новый адрес автоматически.
Webhook endpoint_id = SHA256 canonical(url + secret_env); duplicate endpoint
запрещён. Удаление endpoint отменяет его pending jobs с audit; в config apply
это атомарно. Ротация значения env требует restart; завершённые подписи не
хранятся, каждый retry подписывается действующим секретом.

| Интерфейс | Контракт |
| --- | --- |
| POST /webhooks/github | provider HMAC; отдельная политика от agent bearer, без cookie и CORS; disabled→404 |
| GET /api/v1/integrations | authenticated native read: configured/disabled, unmatched counts, completeness, outbox pending/dead counts; без secret/полного URL query |
| GET /api/v1/integrations/github/deliveries | limit1..200/cursor, status/reason/key; без исходного PR body |
| GET /api/v1/integrations/notifications | paginated delivery status/attempt/next_at/last_code; без body/secret |
| POST /api/v1/import?dry_run=true\|false | native human-scope owner, X-Session-ID и X-Request-ID; application/x-ndjson, X-Import-Source; body exact bytes |
| GET /api/v1/export | authenticated human-scope owner; snapshot NDJSON, Content-Disposition attachment, no-store |
| CLI import validate FILE | offline syntax/schema/references; без DB/network/мутаций |
| CLI import --file FILE --source NAME --dry-run | remote plan против текущей board, без lease renewal |
| CLI import --file FILE --source NAME | тот же authoritative apply; обычный actor, не admin bypass |
| CLI export --out NEWFILE | private0600 no-overwrite staging+fsync; SQLite copy отдельно существующим backup |

Import/export доступны только явно разрешённому human scope и configured local
owner; coordinator-agent без human scope не получает import. Права на создание
роли/curation/cancel и owner/area ограничения D применяются дополнительно.
Не добавлять 13-й MCP tool: E доступен REST/CLI, get содержит GitHub facts.

## 3. GitHub: проверка и replay

Принимать JSON identity encoding, max1MiB raw body; больше →413, compressed
body→415. Это намеренно меньше лимита GitHub25MB, превышение видно отправителю;
отсутствующее событие не выдумывается. Ограниченный preauth limiter и timeout
действуют до парсинга. HMAC-SHA256 exact raw bytes, hex64 `sha256=`, constant-time
comparison; отсутствующая/дублированная/невалидная signature→401. Сначала
проверка подписи, затем bounded JSON decode, запрет duplicate object keys;
неизвестные поля GitHub допускаются для совместимости, но никогда не копируются
в evidence как произвольные полномочия. Не логировать payload/signature/secret.
Проверить единственные X-GitHub-Delivery (GUID), X-GitHub-Event и JSON action.

Разрешены pull_request opened/synchronize/closed, check_suite completed,
pull_request_review submitted; signed ping→200 без board effect. Другие
подписанные типы/actions→202 ignored с ограниченной receipt-записью; broken
required fields→422. repository.full_name сопоставляется с exact allowlist;
неразрешённый repo→403 без facts. PR base repo должен совпадать с envelope repo,
fork head repo допускается как источник кода, не источник allowlist доверия.
URL PR берётся только как `https://github.com/<allowed-owner>/<allowed-repo>/pull/<number>`
с согласованным number; SHA только40/64hex. Никогда не загружать payload URLs.

Одна write-транзакция: current config snapshot → delivery dedup → normalized
facts/bindings → webhook_received или unmatched_webhook → commit →2xx ≤10s.
SQLite error/commit uncertain→503, не success до durability. Same delivery ID+
same event/bodyhash возвращает прежний outcome без новых events/outbox; ID с
другим payload→409 WEBHOOK_REPLAY_CONFLICT. Подпись GitHub не связывает delivery
header с body: повтор тех же event+body под новым GUID также не повторяет
business events. Заголовки сами по себе не удостоверяют sender/actor.

## 4. GitHub: сопоставление, порядок и доверие

Task key ищется configured Go regexp в head branch и PR body; matcher bounded
1MiB input, результаты нормализуются к существующим ключам. Нет ключей, ключ
отсутствует или более одного различного task key → unmatched с причиной
missing_key|unknown_task|ambiguous_key; не прикреплять факт ко всем задачам.
Review использует вложенный PR; check_suite — известные PR numbers/bindings
того repo и head_sha, либо head_branch key; неоднозначность → unmatched.
Facts сохраняются даже когда binding неизвестен. Более поздний PR webhook
может связать ранее unmatched CI факты exact same repo/PR/SHA, с отдельным
audit; это не повтор исходного business event. Не делать GitHub API вызовов.

PR state/head/binding отсортированы по payload pull_request.updated_at; новый
однозначный key делает старый binding historical, не оставляет две current tasks.
Меньшее время
не заменяет current pointer; равное время+различное содержимое → order_conflict,
current unknown, оба факта сохранены. Arrival time и GUID не являются порядком.
Hash immutable Fact не включает task binding. Для PR current pointer обязателен
отдельный normalized hash: Fact.Hash + Match(task_key,reason) + Binding.
Сравнивать только Fact.Hash нельзя: одинаковый SHA/state/source_time при смене
T-1→T-2 либо matched→missing/unknown/ambiguous должен дать order_conflict,
а не Same. Persistence хранит эти pointer версии и выполняет отдельный compare;
это обязательная integration regression WORK046, не реализованная библиотекой
SQL-операция. Export encoder также обязан отвергать NUL identifiers до записи.
Review хранится по review.id+commit_id, выводится submitted_at/state/user;
CI по suite.id+head_sha+app.id, status/conclusion и source timestamp если он
есть. Если payload не даёт сравнимого времени, несколько конфликтующих facts
не схлопываются в «последний доставленный». Результат unknown/conflict.
Не считать максимальный suite.id доказанным порядком разных прогонов.

Task получает read-only `github` projection: pr/pr_state/head_commit/merged_at,
ci facts/reviews, observed_at, completeness=unknown, conflict reasons. Факты
другого SHA видны только как история, не «зелёный CI текущего commit».
Нет configured required-checks set: показываются наблюдённые suites, общий
ci_status=unknown при неполноте; отдельный success не доказывает весь CI.
При exact current SHA можно показать observed failure/success по каждому suite.
GitHub reviews не записываются в Task.Review/AuthorSession/attempts; они не
открывают approved/passed gates. Completion с обязательным pr при включённой
интеграции требует observed PR для allowed repo и exact evidence.commit SHA,
без конфликта binding/head; PR-URL string недостаточен. Закрытый/merged PR —
факт, не автоматическая task_done/epic_done и не разрешение сервиса на merge.

Без API невозможно доказать отсутствие пропусков: completeness всегда unknown,
а не healthy=true по одному webhook. Дубликаты подавляются, но redelivery
пропущенного события разрешена; оператор повторяет доставку в GitHub UI.
Публичный endpoint и реальные доставки проверяются отдельно от signed fixtures.
См. официальные [подписи](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries),
[delivery ID/redelivery](https://docs.github.com/en/webhooks/using-webhooks/best-practices-for-using-webhooks),
[типы payload](https://docs.github.com/en/webhooks/webhook-events-and-payloads).

## 5. Исходящие уведомления и эскалация

Только configured events: task_created_for_human, escalated, lease_expired,
epic_done. Snapshot event + eligible recipients + outbox INSERT одной board
транзакцией. Нет HTTP внутри writer lock. Не настроен endpoint → только event.
Payload исходника дополняется stable delivery_id/event_cursor; ts/event/task/epic
сохраняются. Body ≤32KiB; human description/answer/evidence не отправляются.
task.url опционален: без отдельно настроенного trusted browser origin опустить,
не выводить URL из Host/X-Forwarded-Host. Не помещать bearer/capability в URL.
Idempotent replay операции и duplicate cycle не создают вторую outbox запись.

Worker один, batch≤20, concurrency≤2; короткая transaction claim due rows с
locked_until=now+15s, сеть после commit, результат CAS по claim_generation.
Claim увеличивает attempt/generation; устаревший worker не может ack новый claim.
Recovery после
crash снимает только просроченный worker-lock. Exactly-once доставки в сеть
невозможны: после полученного2xx и crash до ack будет повтор с тем же ID/body.
Получатель дедуплицирует X-Agent-Board-Delivery. Подпись исходника
X-Agent-Board-Signature: sha256=HMAC(secret, raw body), включая неизменяемый ts
и ID внутри body. Body не меняется при retry, подпись проверяется constant-time.

HTTPS only, без credentials/fragment/query, hostname ASCII canonical, port443;
нет proxy из env, cookies, auth headers, redirects, automatic decompression.
Dedicated Transport.DialContext разрешает DNS перед КАЖДОЙ попыткой, проверяет
все A/AAAA, отклоняет mixed public/private, затем dial только проверенного IP;
TLS ServerName и Host остаются исходным hostname, обычная chain validation.
Не использовать transport с неограниченным cached connection/re-resolution:
DisableKeepAlives для этой малой очереди исключает обход повторной проверки.
Запретить loopback/private/linklocal/CGNAT/Tailscale/multicast/unspecified,
IPv4-mapped variants и остальные IANA special-use/reserved ranges. Одного
IsGlobalUnicast недостаточно: private адреса тоже могут быть global-unicast.
Обойти private policy настройкой hostname/redirect невозможно; в V1 нет
private-destination override. Local synthetic tests inject resolver/dialer,
production SSRF policy тестом не отключается. Политика соответствует
[OWASP SSRF](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html),
[Go Transport](https://pkg.go.dev/net/http#Transport) и
[netip адресам](https://pkg.go.dev/net/netip#Addr.IsGlobalUnicast).

Total timeout5s (DNS+dial+TLS+response), response discard≤4KiB, headers bounded.
Любой2xx → delivered. Network/408/429/5xx → initial + до5 retries, интервалы
1m/2m/4m/8m/16m с persisted jitter0..20%; Retry-After может увеличить задержку
до1h. Остальные4xx/3xx и SSRF rejection → terminal dead, без redirect.
После exhaustion → dead + notification_failed audit; status хранит код/класс,
не response body/секретный URL. Outbox disk error → rollback исходной мутации,
не потерянное уведомление; worker error не отменяет уже завершённую задачу.

Human cycle начинается при первом ready, explicit reopen либо restore cancelled;
created draft не эскалируется. waiting_since не сбрасывается claim/release/expiry.
Граница now≥waiting_since+questions.escalate_after; ready/in_progress и ответа
нет → один escalated event+outbox+escalated_at той же транзакцией. Done/cancel
закрывает цикл. Sweeper/restart/два DB connection не дублируют эскалацию.
Изменение escalate_after применяется к текущему циклу со следующей sweep;
не повторяет уже отправленный escalated. Это default48h, не 48h от последнего HB.

## 6. JSONL: формат, валидация, импорт

Поставить `schema/agent-board-jsonl-v1.json`; unknown fields/kinds/version,
duplicate JSON keys/IDs, UTF8 errors, negative tokens, auth/active-session/lease
поля →422 с line/path/code, ≤100 diagnostics. Raw input≤16MiB/10000 records,
line≤256KiB, nesting≤32; без архивов/URL refs/выполнения markdown. Request_id
и source1..128 ASCII `[A-Za-z0-9._-]` из headers, fileSHA256 от exact raw bytes.
Лимиты больше native1MiB только на отдельном import handler, не глобально.

Первая строка manifest `{"format":"accord-jsonl","version":1,"source":"..."}`;
source обязан совпасть с X-Import-Source.
Каждая epic/task/link/note строка имеет kind, external_id, data; references
`{kind,external_id}` в том же source, разрешаются независимо от порядка строк.
Server keys не переносимы: export формирует stable external_id из существующего
mapping либо `accord:<instance-id>:<kind>:<key>`. Ссылки на чужой source запрещены;
target уже существующего source допускается только через authoritative mapping.
Link/note также external_id, не только task: иначе upsert создаёт дубли.
Task.ExternalID сохраняет уникальность C (per kind, ≤256bytes); совпавший ID
чужого source либо manual task без mapping → conflict, не автоматический захват.
`kind:history` содержит external_id, origin_ref и whitelisted data с timestamps/
comments/attempt/evidence snapshots
как untrusted data; чужие session IDs — только namespaced origin_session_label,
не рабочие SessionID. Export сохраняет эту историю без восстановления sessions.

Offline validate проверяет schema/references/DAG; dry-run ещё роли, права, areas,
limits, uniqueness и текущий board без мутаций/lease renewal/expiry persistence.
Ответ plan: fileSHA, config_version, create/update/unchanged/conflicts, максимум
200 записей с continuation; никаких «dry-run PASS» при усечённой проверке.
Apply заново проверяет ВСЁ после writer lock/current config, не доверяет плану.
Concurrent claim/config apply между dry-run/apply →409 IMPORT_CONFLICT при
новом конфликте, не частичная загрузка. Ошибка любой строки откатывает весь
файл, mappings/history/events/outbox. Expiry может материализоваться отдельным
core правилом, но импортные изменения при отказе отсутствуют.

Upsert identity=(source,kind,external_id), unique local mapping. Existing:
metadata-only fields title/description/priority/areas/category/complexity;
note text только с curation правом; link только при обычном разрешении D.
Нельзя менять role/epic/owner/state/provenance/evidence/history существующего
local task импортом; несовпадение→conflict. Любая затронутая live lease или
живой consumer link→409 даже владельцу lease. Не обходить fencing импортом.
Разрешённый imported history append дедуплицируется canonical record_hash;
изменённая запись сохраняется отдельной historical revision, не заменой аудита.

New imported in_progress → ready без lease с source_state в history; done
сохраняется как done+completion_trust=imported_unknown, cancelled сохраняется;
готовые/draft задачи проходят обычные required-fields/rules. Imported done
НЕ удовлетворяет completed/approved/passed dependency, target-review claim или
epic close, даже если в файле SHA совпал и author_session выглядит настоящим.
Imported epic done показывается source_state=done, local epic остаётся open до
обычной серверной приёмки. Для доверенного результата нужен новый явный цикл
ready→claim→completion с серверными attempts; import apply может только создать
такой ready task, не выдать ему доверие. Исторический done не auto-reopen:
для доверенного результата обычный reopen по-прежнему требует reason/evidence_ref.
Для недоверенного imported done нужен отдельный локальный admin путь:
`import requeue --db DB --task KEY --expected-version N --reason TEXT`.
Он применим только к completion_trust=imported_unknown и при отсутствии lease;
CAS, reason и audit обязательны. История/evidence импорта остаются в history,
текущие evidence/author/attempt binding очищаются, state=ready, доверие не
повышается. Аренду выдаёт только новый next; generation не заимствуется из
файла. Это проверка перенесённой истории, не дефект/rework уже принятой работы;
returns_count не увеличивается. Для imported verifier нужен новый актуальный
target binding после реального выполнения target; старый approve не переносится.
Этот privilege не доступен агентскому bearer, не отменяет живую чужую аренду
и не применяется к результатам, полученным текущими серверными attempts.

Same source+fileSHA: текущая auth проверяется, возвращается сохранённый result,
без продления session/новых событий; X-Request-ID conflict сохраняет обычную
семантику. Изменённый файл повторно upsert; unchanged записи ничего не меняют.
Изменённое state/evidence в новом файле существующей task → conflict, не overwrite.
Успех: один import event(source,fileSHA,counts,actor)+import_run в той же tx.
Import не рассылает тысячи human_created; один summary event, human queue
сразу доступна; обычные последующие ready/reopen уведомляются. Этот deliberate
bulk-notification policy требует явной фиксации реализации/review.

## 7. Экспорт, браузер и внешние условия

Export читает одну SQLite read snapshot и фиксирует high_water/config_version,
не смешивает страницы живого board. NDJSON включает сущности, links/notes,
attempts/comments/source times и events как history records; нет токенов,
session auth, секретов config, outbox bodies. Config history — sanitized snapshot,
не исполнимое auto-apply; неизвестные historical роли сохраняются в history,
active task роли обязаны существовать. Streaming bounded memory, abort/error
не публикует partial CLI file. JSONL не обязан быть ≤16MiB для полного export:
если превышен import cap, CLI сообщает необходимость разделить на валидные
references-closed пакеты; полный disaster recovery использует SQLite backup.

Будущий UI: после разрешения внешнего вида human-scoped одноразовый bootstrap
token, выданный local admin, вводится POST формой по HTTPS; не URL/localStorage.
Обмен на отдельный opaque HttpOnly+Secure+SameSite=Strict cookie, rotate on login,
серверный hash/expiry/revoke, CSRF token+exact Origin для mutation и login.
Agent bearer не становится browser admin; login cookie не принимается /api/mcp
native routes. Human board session сохраняет fixed role/provenance и fencing;
кнопка не превращается в передачу аренды. Own token mint ограничен собственным
owner/разрешёнными role scopes, не admin-wide mint. Этот auth flow ещё требует
отдельного UI/security review; сейчас не создаётся ни страница, ни браузерный вход.

Внешние prerequisites: GitHub admin создаёт repo webhook/secret и подписки;
доступный GitHub public HTTPS ingress с valid cert и только webhook route
требует отдельного разрешения владельца. Private Tailscale8443 не подходит;
Funnel/публичный API/изменение Tandem443 не включать. Для уведомлений нужны
выбранный получатель/secret, согласие на отправку title/eligible и проверка
реального канала. Реальные credentials/данные не приложены. OAuth, UI approval,
физический2PC остаются внешними открытыми условиями, fixtures их не закрывают.

## 8. Обязательные проверки и передача

Unit/integration: подписи Unicode/raw-byte mutation/missing/duplicate headers;
allowlist/fork/conflicting key; delivery replay+changedGUID/body conflict;
reverse PR/CI/review order/equal-time conflict/SHA change/missing fields;
fact approval не открывает core gates. Никакого GitHub API вызова.
Outbox crash before send/after2xx before ack, six-attempt limit, revoke config,
DNS mixed/rebinding/IPv4mapped/meta/redirect/proxy env; no network under DB lock;
human escalation ровно на границе/restart/reopen/leaseexpiry/concurrency.
JSONL malformed last line/duplicates/cycles/active lease/new config -> zero
import writes; same-file replay after restart; changed-file upsert; foreign
session/approve/attempt injection не открывает даже completed gate; history
не улучшает stats как новое выполнение. Export+import stable mappings/history.
Migration4→5 crash/backup/reset/oldbinary rejection, race current config apply.
После code+security approve отдельный tester пишет process/E2E и проходит review.
Реальный GitHub/outgoing provider/browser/второйПК — NOT-RUN до условий выше.
