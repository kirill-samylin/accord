# WORK-044 — E: чистые integration foundations

Статус: выполнено, передано на независимое code/security review. Не E целиком.
Родитель: TASK-003; назначение координатора `/root`, исполнитель
`/root/integrationlibs044` (developer); профиль
`davinci-gui-/docs/agents/roles/developer.md`. Именованные модели Claude здесь
не закреплены; использована наследуемая доступная модель. Tandem OFF по
прямому указанию пользователя/AGENTS; отдельный task_claim не применяется.

Рабочая копия: `.local/integration-libs-003`; ветка `codex/integration-libs`,
base `2dfd86bf1627775a8fdd87c650d9a3ada4cae17f`. Commit/push не выполнялись.
Область записи: только новые internal/integrations/github, internal/notifications,
internal/interchange и эта WORK. Store/core/config/CLI/transport/dependencies,
main и D/schema3 другого автора не изменены.

## Реализовано

- GitHub: raw-body HMAC-SHA256, singleton headers/case-duplicates, JSON media,
  identity encoding, bounds, UTF8/depth/duplicate JSON keys; signed ping/ignored
  events; supported PR/review/suite required fields, repo/base allowlist, exact
  SHA40/64 и безопасно построенный URL. Неизвестные provider fields не копируются.
  Нормализованные UTC facts, int64 IDs, stable hash; task-key extraction/
  missing_key/unknown_task/ambiguous_key, suite exact repo/PR/SHA bindings;
  replay classifier по ID и event/body; source-time ordering/conflict classifier.
- Notifications: whitelist payload≤32KiB с stable ID/cursor/ts; подпись exact
  stored bytes, safe request, HTTPS443 canonical URL. Все DNS answers проверяются,
  pin validated IP, private/meta/mapped/mixed/special-use отказ; SNI/Host исходные,
  обычный TLS chain validation; без env proxy/cookies/redirect/decompression/
  connection reuse. Total5s, connect/TLS/headers2s, headers16KiB/discard4KiB.
  Typed delivered/retry/dead и чистая six-total-attempt/jitter/Retry-After policy.
- JSONL: streaming input16MiB/10000records/256KiBline/depth32, ≤100 diagnostics,
  strict exact case/required fields/nulls, duplicates, UTF8, int64 precision,
  no float conversion; exact file SHA и canonical per-record SHA; typed DTO/schema,
  references/basic DAG и authoritative existing-mapping callback; policy validator
  interface, syntax-only при nil; Plan/Result DTO без SQL/apply implementation.
  Stream encoder и0600 fsync+atomic-hardlink no-overwrite file publication.
- Exact APIs/ownership/ограничения описаны README каждого пакета; JSON Schema
  v1 сохранена внутри interchange. Координатор принял DTO конкретизацию:
  task.data.type=task, свободная роль для последующей config validation, epic
  отдельный kind, типизированные ссылки, history.origin_ref внутри data.

## Фактические проверки

Go `go1.27.2 darwin/arm64`, module go1.26.1; зависимости не добавлены/не обновлены.
Команды использовали `/Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go`.

PASS: `go test -race ./internal/interchange ./internal/notifications ./internal/integrations/github`.
PASS: `go vet` того же targeted набора. PASS: gofmt (все Go файлы ≤300 строк).
Unit/integration synthetic: Unicode/raw mutation/header multiplicity/content
policy; allowlist/fork/key/ID precision/replay/order/missing action; matcher cap;
TLS SNI/Host/chain и повторный DNS, mixed/meta/reserved/mapped IP/redirect/env proxy,
retry statuses/cap; JSONL duplicate/malformed last line/precision/negative tokens/
foreign authority/case alias/required/null/reference/cycle/diagnostics and byte
caps; policy callback; no-overwrite0600 и no-partial publish при snapshot error.
Новые process/E2E не создавались: они принадлежат tester после approve.

Официальные источники прочитаны для выбранных API/security policy:
[Go HTTP Transport](https://pkg.go.dev/net/http#Transport),
[GitHub webhook signature](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries),
[IANA IPv4](https://www.iana.org/assignments/iana-ipv4-special-registry),
[IANA IPv6](https://www.iana.org/assignments/iana-ipv6-special-registry).
IANA special-use отвергаются целиком, включая отдельные globally reachable
special allocations; IPv6 консервативно только2000::/3 кроме special-use.

## Ограничения и следующий владелец

NOT-RUN: независимое review/security, runtime/store/schema4/CLI/HTTP E,
transactional dedup, reattachment/history pointer/conflict resolution,
completion gates, outbox claim/CAS/crash/retry persistence/escalation,
authoritative import upsert/trust/requeue/lease/config concurrency и snapshot
export integration. Никакого durable success stub: Delivered только HTTP2xx,
Result.Applied библиотека не устанавливает. DB owner после D выполняет эту часть.
Schema в schema/ устанавливает интегратор из согласованной пакетной версии.

NOT-RUN: реальные GitHub ingress/provider messages/UI/OAuth/второй физический ПК.
Все endpoints тестов synthetic/local через trusted injected resolver/dialer;
production IP policy тестами не отключается. Test CA доступна только same-package
тесту; production TLS roots штатные. Public ingress/реальные сообщения не разрешены.

Source callback ParseWithReferences caller обязан привязать к одному source;
локальные existing links/roles/evidence/owners/areas/leases повторно проверяются
store под writer lock. Offline syntax-valid не означает authoritative plan/apply.
Exporter получает уже согласованный single snapshot; mixed live pages/credential
redaction из произвольного пользовательского текста он определить не может.
Config values исключены DTO, history хранит только config_version и foreign
metadata. Все imported evidence/attempt/session labels недоверенные, никаких
current server sessions/leases/claimed author/trust поля не импортируется.
ExportFile может оставить полный destination при post-publication fsync error;
ошибка не означает rollback. Filesystem hardlinks обязательны, fallback нет.

## Журнал

| Время Europe/Moscow | Событие | Автор/исполнитель | Статус | Основание |
| --- | --- | --- | --- | --- |
| 2026-10-18T18:42:03+03:00 | created/assigned/started (фиксация карточки) | /root → /root/integrationlibs044 | в работе | Реализация назначена сообщением координатора; раннее точное время не восстанавливается |
| 2026-10-18T18:42:03+03:00 | completed/handed_off | /root/integrationlibs044 → /root | на проверке | targeted race/vet PASS; frozen manifest ниже |

## Frozen manifest

SHA256 каждого переданного файла; сама WORK не хеширует себя. Правки после
manifest требуют обновления manifest и повторного затронутого review.

| Файл | SHA256 |
| --- | --- |
| `internal/integrations/github/README.md` | `6d5f09cc67cda642b02b68445c0b7b9b9a70fc7724f5b9f1c05234e02c2c81ac` |
| `internal/integrations/github/decode.go` | `51c18b729df622f91a56670e34dfb7c2e35d090a20340ab033f9e43210f23677` |
| `internal/integrations/github/github_test.go` | `fe04a134d48af4d8dd98a6883010a79976f415ed777b01045f3e3935275782d9` |
| `internal/integrations/github/order.go` | `b5a0f0361e98c942992ca630a274f37bc482749dbbfc497d8b6a73040dfa2335` |
| `internal/integrations/github/types.go` | `04d50410fa867450923ddba5474ec18aba938d846fa2c6e1f40c1626b3c0926b` |
| `internal/integrations/github/verify.go` | `ddcc52956ec7504ef8796a2fc182777ccbe877b3fdf9718ae2ef411cd293159c` |
| `internal/notifications/README.md` | `04dfd5ef1757863613f1cf9b069f74be186371e85a1a8debc4f8f79381f36bac` |
| `internal/notifications/message.go` | `79989137f11d6c6de752d2f835a878004e87b91611622b5b56c25650dc426b72` |
| `internal/notifications/notifications_test.go` | `8713c8ebd7ec5af695274e066b968d96c440164bd3c1562e5197ef70d3818403` |
| `internal/notifications/policy.go` | `43456ce46126b5e7fd25fd11b17f674763997f531527dc4cfcee48971dff7b4f` |
| `internal/notifications/sender.go` | `62456191870b0c64031be7b1d090dc1e964f677aac3af56eed770e42fb27f67f` |
| `internal/interchange/README.md` | `19a57e8f85a84536478234498b99787f7b9eeae4455c2c424c231222d3786ccb` |
| `internal/interchange/agent-board-jsonl-v1.json` | `f7c3764b4e8d0be815b3e51431a5f767a456d98b6dabfecfe1dc8b1f1ac8b65c` |
| `internal/interchange/export.go` | `1e1e5eeb7a2f89f592e09efcfc7ca3fa77485a49cad27ff67c61ec78e5786a63` |
| `internal/interchange/json.go` | `3cc8c90c419cbb2bc161caea8ec82bc5800ab5a49511c04f292d99baa9233a00` |
| `internal/interchange/parse.go` | `d7405badc296ae555179986eadcdf2d7f6f4975ca44bdc658fbab586af1ab644` |
| `internal/interchange/parse_test.go` | `881d2557f04f9275e7519ee2a876ea0683bb9aa8e0bba8bd1d7fe3b6f157737e` |
| `internal/interchange/types.go` | `bed4ab46275435499880919595ffd46a13c6846c68f8cdf6077af5deb174744f` |
| `internal/interchange/validate.go` | `049926ad93866e1d534dedb1059bad5fdfeaf7eb3c0e16a7b8e458cd25962945` |
