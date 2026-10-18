# WORK-046 — независимое review/security E foundations

Вердикт: **APPROVE ограниченного библиотечного среза WORK-044**. Блокирующих
дефектов в согласованных границах не обнаружено. Это не approve runtime E,
не подтверждение persistence/import apply/outbox и не готовность V1.

Проверяющий: отдельный от автора агент `/root/integrationreview046`, роли
reviewer + security; профили и общий процесс `davinci-gui-/docs/agents` прочитаны.
Именованные модели Claude здесь не закреплены; использована та же доступная
наследуемая модель, что у автора. Tandem OFF по прямому указанию пользователя
и AGENTS текущего сервиса. Source/tests не изменялись; Git/live/send не выполнялись.

## Проверенная версия и границы

Checkout `/Users/shagi/Desktop/Project/agent-board/.local/integration-libs-003`,
ветка `codex/integration-libs`, HEAD/base
`2dfd86bf1627775a8fdd87c650d9a3ada4cae17f`. Изменения незакоммиченные.
Все **19 файлов** frozen manifest WORK-044 проверены SHA256 и совпали;
сама WORK-044 — двадцатый переданный файл, не включённый в собственный manifest.
Источник manifest: `docs/work/WORK-044-integration-libraries.md` в этом checkout.

Прочитаны все исходники, README, тесты и JSON Schema в
`internal/integrations/github`, `internal/notifications`, `internal/interchange`,
`docs/service-contract.md` §E и `docs/integration-contract.md`. Декомпозиция Go
файлов соответствует ориентиру 300 строк. JSON Schema 625 строк — декларативная
схема с повторяемой структурой свойств; деление ради размера не требуется.
Новых зависимостей и обходов типизации нет; новый блокирующий дубль логики
не обнаружен. Общий JSON checker переиспользуется тремя пакетами.

Осознанно отсутствуют schema4/store/core gates, config wiring, HTTP/CLI handlers,
outbox claims/CAS/recovery, durable dedup/import upsert и snapshot acquisition.
Это следующие задания, а не фиктивно завершённая часть WORK-044.

## Проверенные свойства и модель угроз

Недоверенные границы: подписанное содержимое GitHub всё равно не является
серверным approve; recipient/DNS/HTTP response недоверенные; JSONL и его история
не создают серверные sessions, leases или доверенное завершение.

- GitHub: HMAC-SHA256 точных raw bytes проверяется до JSON decode, сравнение
  constant-time; max1MiB, singleton headers включая mixed casing, encoding/media
  policy, UTF8/depth/duplicate keys. Provider unknown fields допускаются, но в
  facts выходит whitelist. Exact repo/base allowlist, безопасно построенный PR
  URL, SHA40/64, int64 IDs; task matching ограничен и консервативен при ambiguity.
  Review не меняет PR binding и не создаёт core review. Replay classifier не
  обещает durable dedup; comparator не использует arrival/GUID/max suite ID.
- Outgoing: HTTPS443 canonical hostname, без URL credentials/query/fragment;
  все DNS ответы проверяются до dial, mixed/private/meta/mapped/special-use
  отвергаются. Dial закреплён за проверенным IP; исходные Host/SNI и обычная TLS
  chain validation сохранены. Каждая попытка заново разрешает DNS; redirect,
  env proxy, cookies, decompression и connection reuse отключены. Контекст/total
  timeout5s, connect/TLS/headers2s, headers16KiB и discard4KiB заданы явно.
  Подписываются exact stored bytes с ID/ts; retry/dead классификация и шесть
  total attempts соответствуют контракту. Delivered означает только HTTP2xx.
- JSONL: required/exact-case/unknown/duplicate/null checks, int64 без float64,
  limits bytes/line/records/depth/diagnostics; typed references, basic DAG,
  source-bound existing mapping callback. Raw file SHA и canonical record SHA
  различаются по назначению. History/Evidence — недоверенные переносимые DTO;
  active authority/config secrets не имеют допустимых полей. Parse с nil
  validator явно описан как syntax/schema/references/DAG-only. Будущий store
  обязан повторить policy/config/lease/reference validation под writer lock.
- Export принимает один уже полученный sanitized snapshot; 0600 staged file,
  fsync, atomic hardlink no-overwrite. Ошибка до публикации не оставляет partial
  destination; post-publication error явно не обещает rollback. Секрет, вручную
  вписанный в свободный текст, библиотека распознать не может — это честно указано.

Поддерживаемые API/security assumptions сверены с официальными источниками:
[GitHub validation](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries),
[Go1.27.2 HTTP Transport](https://pkg.go.dev/net/http#Transport),
[IANA IPv4](https://www.iana.org/assignments/iana-ipv4-special-registry),
[IANA IPv6](https://www.iana.org/assignments/iana-ipv6-special-registry).
Консервативная IP policy покрывает опубликованные там special-use диапазоны,
включая globally reachable special allocations; актуальность будущих allocation
изменений не гарантируется навсегда и уже отмечена README.

## Блокирующие замечания

Нет в границах pure libraries. Отсутствующий runtime не объявляется проверенным.

## P3 и обязательства интегратора

### REV046-01 — Fact.Hash не является PR binding-pointer hash

Место: `internal/integrations/github/decode.go:83-87,150-155`,
`internal/integrations/github/order.go:16-18`, README GitHub.

Два PR наблюдения с одинаковыми repo/number/SHA/state/updated_at, но разными
task matches T-1 и T-2 дают разные `Delivery.Binding`, однако одинаковый
`Fact.Hash`. `CompareVersions` этих **fact hashes** корректно возвращает `same`:
сам Fact DTO одинаков. Если будущий store применит именно этот hash для
binding pointer, он потеряет требуемый equal-time conflict; matched→missing/
ambiguous/unknown имеет ту же опасность.

Это **не blocker foundation**: Binding и Match выдаются отдельно, сравнение
принимает произвольный hash, store пока отсутствует. Нельзя приписывать пакету
ещё не написанную неправильную интеграцию. Координатор 18.10.2026 подтвердил
обязательство E persistence: отдельный normalized PR-pointer hash/compare из
Fact.Hash + Match(task_key/reason) + Binding; equal-time regression для T1→T2 и
matched→missing/ambiguous/unknown обязателен до приёмки runtime E. Fact.Hash
остаётся хешем immutable provider fact; это различие следует явно сохранить
в README/API-документации следующего задания.

### REV046-02 — Encode допускает NUL в external_id

Место: `internal/interchange/export.go:32`, сравнить с
`internal/interchange/parse.go:69` и JSON Schema external_id.

`Record.ExternalID = "bad\x00id"` с корректным task data проходит Encode;
последующий Parse экспортированного файла возвращает invalid document.
Нужна общая проверка identity (непустой, bytes≤256, без NUL) для parser/exporter
и regression, что ExportFile не публикует такую запись. Сейчас snapshot callback
доверенный, stable IDs получает от server keys; достижимого пути с внешним NUL
в production ещё нет, поэтому P3, без блокировки foundation. Координатор
согласовал исправление тем же владельцем модуля в следующем задании до выпуска E.

## Фактические проверки

Go `/Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go`,
go1.27.2 darwin/arm64; module go1.26.1. Независимые прогоны:

```text
go test -count=1 -race ./internal/interchange ./internal/notifications ./internal/integrations/github
PASS interchange 1.694s; notifications 1.705s; github 1.580s
go vet ./internal/interchange ./internal/notifications ./internal/integrations/github
PASS (exit0)
```

Существующие unit/integration сценарии проверяют raw/Unicode mutation,
headers multiplicity, allowlist/fork/key matching, precision/replay/ordering,
TLS Host/SNI/chain, DNS mixed/rebinding, env proxy/redirect, retry policy,
JSONL malformed tail/duplicates/authority/case/required/null/precision/limits,
references/cycles/policy callback и atomic no-overwrite export.

Дополнительно выполнены два временных synthetic Go overlay tests — без записи
в source tree. Они намеренно ожидают более сильное поведение и оба FAIL:

```text
TestReview046EqualTimeBindingConflict:
  bindings=T-1/T-2 hashesEqual=true order=same
  distinct equal-time bindings collapsed
TestReview046ExportRejectNULIdentity:
  Encode error=<nil> Parse error=invalid interchange document
  Encode published a forbidden identity
```

Первый FAIL — подтверждённый контрпример **неверного использования API** для
будущего binding pointer, а не падение согласованной семантики Fact comparator.
Второй — подтверждение P3. Эти результаты не маскируются PASS основного набора.
Для воспроизведения первого достаточно Decode двух существующих fixturePR
с T-1/T-2 и одним SHA/time, затем сравнить их Fact.Hash; второго — Parse обычной
fixture, заменить Record.ExternalID на строку с NUL, Encode и снова Parse.

NOT-RUN: полный main check, новые process/E2E, schema4/storage/runtime E,
outbox durability/crash/CAS, import authorization/upsert/requeue/concurrency,
реальные ingress/GitHub/API/provider messages/browser/OAuth/второй физический ПК,
Linux/Windows runtime, deploy/release. Ни одно из них не представлено как PASS.

Следующий владелец: координатор интегрирует frozen foundation и назначает
store/runtime E после D; автор закрывает P3 export и явно закрепляет отдельную
семантику binding versions. Runtime E требует собственного code/security review
и последующих независимых process/E2E, включая перечисленные обязательства.
