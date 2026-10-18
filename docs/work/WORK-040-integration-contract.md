# WORK-040 — исполнимый контракт интеграций E

- Родитель: [TASK-003](../TASK-003-working-service.md).
- Роль: архитектор, integration_contract003; статус: выполнено, ожидает accepted.
- Запись: только эта WORK и [integration-contract](../integration-contract.md).
- Входы: AGENTS, requirements1.1, original§11.7–14, service-contract E,
  config-contract C/D, actual config schema/types, store transactions/gates/
  evidence/migrations/persistence. C/schema2 ещё меняется, D/schema3 ожидается.
- Общий процесс/parallel-work/design-first/architect profile прочитаны в
  соседнем проекте read-only; модели Claude к профилям здесь не прикреплены,
  используется унаследованная доступная модель без изменения настроек.
- Tandem отключён пользователем; подключений/board tasks не создавал.

## Результат

Определены schema4 под одним владельцем после D, incoming HMAC/dedup/raw body
caps/repository allowlist/facts per SHA/out-of-order/conflicts. Новое additive
config поле github.repository_allowlist обязательно при enabled events;
сейчас в actual schema его нет. GitHub reviews не подменяют core provenance.

Определены durable transactional outbox, stable delivery/body/signature,
at-least-once semantics, initial+5 retries, SSRF/DNS pinning/no proxy/redirect,
состояния отказов, explicit human unanswered cycles/escalation и monitoring API.

JSONL получает schema, source namespace/external mappings для всех kinds,
offline validate/dry-run/atomic apply/replay, safe metadata-only upsert,
полный export и отдельный SQLite recovery. Импорт done/evidence/history имеет
imported_unknown trust и не удовлетворяет ни один gate, включая completed;
восстановление server-session авторства через JSONL запрещено.

UI не реализован и не объявлен согласованным: описан только необходимый
tokenfirst bootstrap→cookie/CSRF security boundary для следующего review.
Public ingress не разрешён; частный Tailscale8443 недоступен GitHub provider.
OAuth отложен по выбору пользователя. Bulk import notification summary и
консервативные import state conflicts явно выделены для проверки координатором,
не названы новым пользовательским согласованием.

## Проверки и ограничения

- PASS: read-only trace actual config/store/gates и соответствия source/current REQ.
- PASS: официальные GitHub signature/delivery/payload docs и Go/OWASP SSRF сверены.
- PASS: git diff --no-index --check для каждого нового owned файла без замечаний
  (exit1 означает новые файлы); UTF8/trailing whitespace/local links проверены.
- N/A: приложение/runtime/E2E для документальной задачи не изменялись/не создавались.
- NOT-RUN: реализация E, её independent review/security/process acceptance,
  реальный GitHub/канал уведомлений/browser/физический2PC.
- Контракт превышает ориентир300 строк: таблицы storage/API и проверки доверия
  оставлены в одном исполнимом документе; нет production source file >300.

База чтения: HEAD `99e722c44f83b1808120bba1fabb04cf2f68a5b5` с незакоммиченной
реализацией C/schema2 и offline config; версия их реализации не объявлена approve.

18.10.2026 started → contract prepared → передан координатору. Следующий шаг:
его принятие, независимый contract/security review и назначение одного E store
owner после D; source/общая TASK/Git/live/соседний проект не изменялись.

## Уточнение координатора после передачи

Консервативные технические defaults summary audit и отказа от overwrite
state/evidence приняты для реализации; это не новое согласование пользователя.
Добавлен отдельный локальный admin import requeue с CAS/reason/audit только для
imported_unknown: устраняет тупик с evidence_ref, сохраняет исходную history,
не выдаёт доверия/аренды и не применим к обычным принятым результатам.
Исходный SHA выше относится к версии архитектора; эта дельта контракта подлежит
независимому review вместе с E. UI/public ingress/recipient остаются открытыми.
