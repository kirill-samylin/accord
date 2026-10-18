# Accord — контракт завершения сервиса

Дата: 18.10.2026. Родитель: [TASK-003](TASK-003-working-service.md).
База обследования: `966dda6`; этот документ описывает реализацию следующих
срезов, а не утверждает их готовность. Источники: [ТЗ 1.0](source/TZ-agent-board-service-v1.0.md),
[требования 1.1](requirements.md), [API](api.md), [roadmap](roadmap.md).
При конфликте исходника действуют исправления требований 1.1.

## Решения и границы

Пользователь поручил доделать рабочий проект. Уточнения: сначала отдельные
токены вместо GitHub OAuth; второй физический ПК будет доступен позже.
Это откладывает внешние проверки, не отменяет остальных требований V1.
Tandem отключён от процесса; существующий сервис и его данные сохраняются.

Ближайший эксплуатируемый результат: общий сервер Accord с bearer-доступом,
автозапуском, резервированием и безопасным клиентом для долгих операций.
Его допустимое название — «рабочий выпуск с токенами». Полная V1 требует
оставшихся строк карты ниже. Два процесса на Mac не закрывают physical2PC.
Функциональные страницы из исходника определены; их визуальная реализация
ждёт макетов либо явного разрешения пользователя на простой SSR-интерфейс.

Не меняются: fencing, немедленная expiry, транзакционная идемпотентность,
положительные exact-SHA gates, независимая server-session provenance,
аппаратное ограничение одной задачи на сессию и запрет снять чужую аренду.
Режим `off` конфигурации не отключает авторизацию, fencing или эти инварианты.
Сервис не исполняет код и не способен остановить локальные записи агента.

## Карта полного объёма

Статус «есть» означает найденную реализацию на базе, не новый PASS TASK-003.

| Источник | Есть | Остаток и зависимость |
| --- | --- | --- |
| §1–3, один экземпляр/проект | Go/SQLite, отдельный проект, MIT | EN README + ключевые RU документы |
| §4, участники/токены | owner/machine, hash, expiry/revoke, sessions | list/audit/scopes контракты; GitHub/OAuth отложены пользователем |
| §5, эпики/задачи/попытки | основные поля, история, lease generation | category/complexity/external_id, tokens/returns, branch update |
| §5.4, связи | blocks/contract_of/spawned_by/related, cycles | duplicate_of, связи эпиков |
| §5.7, заметки | отсутствуют | CRUD, limits/curation, role/epic retrieval |
| §6, состояния/evidence | ready/claim/done/cancel/reopen/gates | created stage, configurable roles/schema, applicability decision |
| §7, аренды/singleton | транзакции/TTL/replay/fencing, architect | автоматический heartbeat, sweep/metrics, config singleton roles |
| §7.3, owner | поле owner | off/warn/block/shared_roles/handoff |
| §8, очередь | atomic role-filtered next, stable order | waiting counts, human eligibility, epic priority/ownership |
| §9.1, старт | describe/next/events | rules/notes/my_tasks/queue в describe |
| §9.2, architect handoff | create/link/release/done/new next | реальный remote/два физических ПК |
| §9.3–4, human/design-owner | human evidence CLI/API | category eligibility, design-owner, web, escalation |
| §9.5–6, returns/expiry | reopen/attempt expiry | returns/token aggregates, persisted branch before death |
| §9.7–8, duplicates/experience | отсутствуют | trigram detection, duplicate_of, notes |
| §9.9, эпик двух машин | локальные process сценарии | полный remote эпик, Claude+Codex, физический ПК позже |
| §10, конфигурация | только TTL flags | YAML/schema/validate/apply/history, roles/rules/areas |
| §11.1–2, API/MCP | 11 tools, REST/CLI, real Codex local | 12-й note, read resources/prompts, enriched summaries |
| §11.3–4, REST/resources | tasks/session/events | lists/sessions/notes/stats/config/import/export; aliases |
| §11.5, OAuth MCP | отсутствует | отдельно после bearer; metadata/PKCE/audience, не имитация OAuth |
| §12, веб | отсутствует | «Моя очередь», «Состояние», токены/config/audit/stats; UI approval |
| §13, аудит/статистика | монотонные core events | richer typed payload, token/config/import audit, metrics of work |
| §14, перенос | отсутствует | schema JSONL, external_id upsert/conflicts, export, SQLite backup |
| §15.1, поставка | Mac binary, cross-build Linux/Windows | container/systemd, private HTTPS, dev mode, upgrade/rollback |
| §15.2–3, эксплуатация | timeout/body limits, private DB | backup drill/scheduler, JSON logs/metrics/rate limit/secret scan |
| §15.4, проверки | unit/integration/race/process/reviews | remote/client/heartbeat, crash/backup/load and target OS runtime |
| §15.5–16, документы/template | RU pilot/API/architecture | complete config/manual, commented davinci.yaml, EN docs |
| §17–18, выпуск/AC1–12 | локальная часть AC2/4/6 | все прочие критерии по карте; AC1 physical2PC открыт |
| §19, решения | Go/SQLite, MIT, coordinator non-singleton | planner не вводить; tokens-first; UI approval; физический ПК позже |

Webhooks/notifications исходника (§3/11/15/18): пока отсутствуют целиком,
контракт ниже. Наличие CLI токенов не закрывает human browser identity/OAuth.

## Срез A — stdio MCP bridge и долгие операции

`agent-board mcp --server <origin> --token-file <private-file>` запускает
stdio MCP, используя официальный SDK; stdout содержит только протокол,
диагностика — stderr без токена. Remote origin только HTTPS, redirects
запрещены; прежняя literal-loopback HTTP политика сохраняется. Токен читается
из private file или существующего env; секрет не передаётся аргументом.
Один bridge соответствует одному запуску агента и одной board-сессии.
Общая клиентская библиотека HTTP переиспользуется CLI и bridge.

Список 11 tools/существующие input DTO сохраняется; `describe` запоминает
session_id/role_generation, `next` — task_key/generation, успешные terminal
transition/release очищают владение задачей. Параллельные bridge-вызовы и
таймер сериализуются для согласованного снимка владения. Чужая session_id
отвергается локально. Повтор describe не оставляет тайно продлеваемую старую
сессию; для смены роли нужен новый процесс. GET/replay не считаются heartbeat.

SessionView расширяется additive-полями `session_ttl_seconds`,
`task_ttl_seconds`, `heartbeat_every_seconds`; сервер возвращает их из
активной конфигурации. Default 900/900/300 секунд; для тестового TTL1s
использовать дробные секунды либо duration в наносекундах, не округлять до0.
Интервал bridge ≤ min(TTL)/3, `--heartbeat-interval` только уменьшает его.
Каждый tick — новый request_id, текущие task+generation+role_generation;
один неопределённый запрос можно повторить только с тем же request_id.

Ошибка сети/timeout/неразборчивый ответ немедленно включает quarantine:
изменяющие tool-вызовы возвращают понятный `LEASE_UNCERTAIN`, уведомление
MCP/logging и stderr требуют остановить локальные изменения. Read get/events
не продлевают и не восстанавливают владение. LEASE_LOST/401 окончательны для
этого процесса. Перезапуск открывает новую сессию; прежняя аренда доживает
TTL. Автоматически получать новое generation/новую задачу и продолжать
работу нельзя. EOF/SIGTERM прекращают таймер, не освобождают задачу заранее.

Обязательные проверки: ожидание >2TTL с bridge сохраняет lease; kill bridge
даёт expiry и новый claim; timeout quarantine не допускает mutation;
stale/revoked token и concurrent transition/tick не оживляют lease;
реальный Codex → удалённый HTTPS Accord, отдельно Claude при доступности.

## Срез B — управляемый сервер и данные

Один процесс на VPS слушает loopback, private HTTPS proxy/Tailscale закрывает
внешний транспорт. Новый private порт/endpoint не вытесняет Tandem443.
Системный пользователь без login, DB/sidecars0600, data dir0700, binary RO,
systemd restart+hardening. Без общедоступного порта, Funnel и token в URL.
Health/ready показывают только состояние, без данных board.

`token list --db` возвращает `{id,owner,machine,expires_at,revoked}` без secret/
hash. `sessions list --db` — `{session_id,owner,machine,role,run_id,expires_at}`
без секретов. Create/revoke пишут атомарные audit events; повтор revoke
идемпотентен, revoke не снимает чужие leases до срока. Администрирование
через local CLI/SSH; агентский bearer не получает remote mint/revoke право.
Для Mac и второго ПК отдельные токены; обмен приватным каналом. Сам тестовый
токен/данные отделяются от рабочих и отзываются после тестов.

`backup --db PATH --output NEW` делает согласованную SQLite-копию с online
backup либо `VACUUM INTO`, private staging O_EXCL0600, parent0700, без cp live WAL.
Проверяет integrity_check, schema version, публикует metadata SHA/time/version
без секретов. Архив содержит credential hashes и всё содержимое board:
права/хранение такие же, как DB. Копия проходит full integrity_check и fsync
файла/каталога; final появляется только после проверки через атомарный hardlink
без overwrite, затем staging удаляется. Если hardlink недоступен — отказ,
не переход к частичной публикации. Restore только в новый закрытый
каталог при остановленном сервере; integrity/schema check до запуска.
Snapshot может вернуть уже отозванные credentials: перед доступом обязательны
`token revoke-all --db --reason restore` и `maintenance reset --db --reason
restore`. Первая транзакционно отзывает все токены+audit, вторая закрывает
сессии/попытки, освобождает leases без снижения generations+audit; только
локальный admin/offline, после гарантированной остановки старого экземпляра.
Выдаются новые токены. No automatic restart; сначала оператор проверяет
изолированный restore. Два сервера одного проекта одновременно не включаются.

Скрипты install/update/rollback/backup проверяют ОС/архитектуру, checksum,
создают pre-upgrade backup, не перезаписывают secrets/config. Rollback бинаря
допустим только при совместимой schema; иначе restore backup с предупреждением
о потере последующих записей. Таймер backup+retention и drill обязательны;
offsite backup остаётся явным риском, если владелец не предоставил место.

## Срез C — списки, заметки, статистика

`GET /api/v1/tasks`, `/epics`, `/sessions`: authenticated, limit1..200,
фильтры state/role/epic/owner; numeric stable key order и opaque next_cursor,
`high_water` фиксирует границу списка, повторный проход начинается заново.
Чтение не продлевает lease. Списки по умолчанию исключают done/cancelled;
явный state/include_closed открывает историю. Не возвращать весь журнал.

`note` — 12-й MCP tool; `{action:create|delete,request_id,scope_type:role|epic,
scope_id,text,source_task_key}` либо `{action:delete,key,reason}`. Note:
`key,scope_type,scope_id,text,source_task_key,author_principal,author_session,
created_at,deleted_at?`. UTF8≤2048bytes, источник существует; role/epic существуют.
Автор создаёт только для своей роли/эпика доступной задачи; delete автор
либо can_curate_notes. Soft delete+event, не терять provenance. Лимиты из
исходника30/24000bytes на роль,20/16000 на эпик; проверяются транзакционно.
`LIMIT_EXCEEDED` содержит старейшие ключи без автоудаления.
Хранение: notes таблица, sequence не сбрасывается; GET /notes и MCP resource.
Describe additive notes/rules/my_tasks; next additive epic_notes/queue waiting.

Attempt получает `tokens_spent?` (целое≥0, missing отличается от0), task
`returns_count` и `category/complexity/external_id`. Comment принимает
`tokens_spent_delta?` и `branch?`, мутация только своей живой попытки;
transition final total не суммируется с delta второй раз: total≥sum(delta),
итог попытки=total, иначе sum(delta), missing если ничего не передано.
Reopen увеличивает returns_count ровно один раз. Source event сохраняет
typed payload для first_done/rework/role/owner/epic/tokens/old+new state.
Старые события без этих данных отмечаются incomplete, история не выдумывается.

`GET /stats?from=RFC3339&to=RFC3339&group_by=role|owner|epic` — полуоткрытый
интервал [from,to), UTC; result totals+groups+incomplete_data. Метрики:
first completions отдельно от rework, epic completions, current queue,
first_claim→first_done wall duration, blocked duration, returns, tokens,
missing_tokens_share, expired leases/rule warnings. Не модельное время/рейтинг.

## Срез D — конфигурация и все правила

`config validate FILE` offline: schema/unknown fields/role refs/TTL/limits/
evidence/reopen/areas/tool budget. `config apply --db FILE` локальный admin:
одна транзакция config_history(hash,canonical_json,raw_yaml,actor,time,version)
+active version+event. Невалидный файл ничего не меняет. CAS `expected_version`
предотвращает потерю конкурентного изменения. GET /config/history read-only.
Секреты задаются env names, значения не попадают в config/history.

Roles: kind agent|human, singleton, create/cancel/curate, evidence JSON Schema,
independent_from/reopen. Нельзя удалить роль/изменить singleton при живой
сессии, ужесточить схему evidence существующей leased task или TTL на ходу;
apply конфликтует409 с точным списком, остановка сессий не выполняется молча.
Восстановление имеет монотонные config_version/schema migrations. Никаких
произвольных scripts/expressions, schema local-only refs и bounded depth.

Created stage — `create.ready:false`, default true для совместимости;
created→ready проверяет обязательные поля. Update только title/description/
priority/areas/category/complexity; role/evidence/lease не через patch.
Epic-owner off/warn/block, shared_roles; default warn шаблона. Handoff отдельная
admin/coordinator мутация c reason+expected_version, не передача task lease.
Human eligibility из category/complexity+owner config, проверяется при claim
и done; design-owner отдельная human-role. Planner singleton не включается.

Area canonical relative paths, aliases→один root, запрет traversal/двусмысленности;
parent/child пересекаются, сервер не читает клиентские FS. overlap warn default,
single_editor_role по template. Duplicate normalize(title) casefold+whitespace,
char trigrams similarity Dice=2|A∩B|/(|A|+|B|), threshold0.8, same epic+open;
warn возвращает similar keys; block force требует reason+право coordinator.
duplicate_of cancel с reason и target, не удаляет историю и не снимает live lease.
Связи эпиков используют тот же DAG/gate evaluator; contract-delivery default done.

## Срез E — интеграции, перенос и UI

GitHub incoming: HMAC-SHA256 raw body до parsing, bounded body, delivery_id
UNIQUE, project repository allowlist. Типы PR/check_suite/review; mapping key
из branch/body regex; SHA обязателен. Хранить deliveries/unmatched и current
facts by SHA, delivery replay не повторяет события. Out-of-order не затирает
новые факты; completeness unknown после gap, manual redelivery без API calls.
PR URL/CI из webhook не подменяют review/tester server-session gate.

Outgoing: durable outbox в транзакции события, delivery_id, body+timestamp HMAC,
bounded HTTP timeout, retry/backoff до заданного лимита+dead-letter audit;
SSRF policy запрещает loopback/private/meta endpoints по умолчанию, redirects
off. Human unanswered escalation48h один раз на цикл с повторным уведомлением.

JSONL versioned records epic/task/link/note, external_id UNIQUE, checksum/
source/import event. Offline validate/dry-run перед atomic apply; reject active
leases/conflicting external mapping; повтор того же файла идемпотентен.
Export snapshot включает все сущности/историю, secret tokens/session auth не
переносятся; backup SQLite отдельный. Мигратор Давинчи — внешний script,
реальные данные не переносить без отдельного поручения.

Web2pages+service views: списки/пусто/loading/error, pending submit prevents
double claim, expiry/read-only fallback, human claim/answer/reopen/comment;
state/lease/warnings/dependencies; own token management; config/history/audit/
stats. Browser использует HttpOnly Secure SameSite cookie+CSRF и human identity,
не хранит bearer в localStorage/URL. Token bootstrap browser auth требует
отдельного scope/rolehuman, не превращает любой agent bearer в администратора.
GitHub OAuth/MCP OAuth после пользовательского tokens-first этапа.

## Критерии приёмки и порядок

A и B независимы; C после их контрактов; D миграции под одним владельцем;
E backend после D, UI после отдельного разрешения, OAuth после credentials.
Каждый срез: unit/integration → независимые code/security review → отдельные
process/E2E и их review → актуальная общая сборка/race. Изменение после approve
требует повторной проверки дельты. Нельзя публиковать документацию как PASS.

Рабочий bearer выпуск: deployed version/health, separate tokens/revoke,
real remote Codex, >2TTL heartbeat/kill/reclaim, backup/restore drill,
upgrade/restart/rollback, no shared Tandem resources, operator+2PC instructions.
Полная V1: вся карта, AC1–12 исходника с исправлениями1.1, load20sessions/
1000tasks p95≤50ms и стенд1vCPU/512MiB; RSS≤128MiB/start≤1s отдельно измерены.
Linux/macOS amd64/arm64 и Windowsamd64 build не заменяет runtime checks.
Физический2PC и Claude при недоступности остаются NOT-RUN с владельцем,
OAuth и UI имеют явные условия; незакрытое требование не исчезает из отчёта.
