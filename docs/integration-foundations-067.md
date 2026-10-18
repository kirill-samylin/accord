# Offline E config и portability — WORK-067

18.10.2026. Реализация pure validation поверх approved E foundations059ae;
не runtime-ready E. Требования — [integration contract](integration-contract.md)
§2/4/5, [config contract](config-contract.md) и [review046](review-046-integration-libs.md).
Результат и exact freeze — [WORK067](work/WORK-067-integration-config.md).

## Repository trust

Config version1 получает optional `github.repository_allowlist`. Это массив
до16 уникальных identities `github.com/<owner>/<repo>`; каждый непустой элемент
ASCII, ≤256bytes, без scheme, `.git`, query, fragment или port. Config требует
host prefix; более короткая provider-форма owner/repo здесь запрещена.
Сравнение и snapshot используют lower-case owner/repo; различие регистра
не позволяет дважды включить один repository. Валидатор переиспользует pure
`github.CanonicalRepository`, не создаёт второй provider regex.

При непустом `github.events` нужен непустой explicit allowlist1..16, иначе
CONFIG_INVALID даже с Options.GitHubEvents=true. Отсутствующий allowlist и []
допустимы при пустом events и означают отсутствие доверия. `project.repository`
сохранён как прежние D metadata; его значение никогда не разрешает webhook.
`require_approval_for_merge` также не выполняет merge и не создаёт approve.

Эта additive форма не меняет version1, но прежний binary отвергнет новое поле:
сначала обновляют binary, затем применяют соответствующий config. JSON Schema
проверяет форму/лимиты, Parse дополнительно проверяет canonical duplicates,
byte bounds, `.git` и зависимость events → allowlist.

## Исходящие endpoints

Config вызывает existing `notifications.ValidateURL` без DNS: HTTPS,
canonical lower-case ASCII DNS hostname, без IP literal, credentials, query
или fragment; только implicit port либо443. Hostname syntax validation offline
не доказывает public destination IP. Runtime sender по прежней библиотечной
политике проверяет все resolved IP перед каждой попыткой. Env proxy, redirects
и private destination override от этого изменения не появляются.

Snapshot удаляет explicit :443 и записывает пустой root path как `/`, поэтому
`https://example.invalid`, `https://example.invalid/` и
`https://example.invalid:443/` имеют одну endpoint identity при одном secret_env.
Identity не зависит от subscriptions: повтор endpoint с другими events тоже
CONFIG_INVALID. Разный secret_env NAME либо другой path — другая identity.
Разрешены только `task_created_for_human`, `escalated`, `lease_expired`,
`epic_done`; unknown и duplicate events отвергаются, не игнорируются.
Максимум16 endpoints, максимум4 уникальных events на endpoint.

Точный алгоритм будущего outbox endpoint_id: SHA256 от UTF-8 bytes
`canonicalURL + NUL + secretEnvName`, результат lower-case hex64.
В delimiter участвует нулевой байт, не текстовая escape-последовательность.
Сначала ValidateURL удаляет default port, затем пустой URL.Path становится `/`;
url.String() даёт canonicalURL. В hash участвует только имя env, никогда значение.
Сейчас helper private в internal/config/integrations.go: пока outbox отсутствует,
нет второго публичного API без потребителя. Runtime owner должен выделить/вызвать
единый helper при wiring, а не завести расходящийся алгоритм или policy.

## Offline использование и capability

`config.Parse(data, Options{})` и обычная offline `config validate FILE`
не читают env, не разрешают DNS, не делают network/DB IO и не раскрывают source
values в diagnostics. Env fields остаются именами. Snapshot.Decode и
CanonicalJSON возвращают независимые copies, включая новые nested slices.
Старые embedded/distributable templates не менялись и не включают интеграции.

GitHubEvents/Notifications flags по умолчанию false; непустые events/endpoints
по-прежнему дают CONFIG_UNSUPPORTED без реализованной runtime capability.
Включать эти flags вправе будущий runtime, который реализовал соответствующий
приём/доставку. Установка flags сама по себе не включает auth.github/OAuth.
Config parsing с flags=true — только pure capability-aware validation,
не проверка работоспособности интеграции. При реальном enabled runtime нужен
отдельный startup/readiness check секрета≥32bytes, env-name validation его
не заменяет. Никаких реальных credentials или provider settings здесь нет.

## Portability identity и оставшиеся обязанности

Общий локальный validIdentity используется Parse/Encode; validRef — data и
reference validation. Идентификатор непустой, ≤256bytes, без NUL; это относится
также ко всем typed refs task.epic/target, note.epic, history.origin_ref,
link.from/to. Некорректная ссылка не передаётся authoritative mapping callback.
Encode отвергает NUL до записи record; ExportFile при таком record не публикует
finalfile и удаляет staging. Stream Encode может уже записать manifest либо
предыдущие корректные records; atomic publication обеспечивает ExportFile.

Исправление REV046-02/P3 проверено авторскими regression; независимая приёмка
ещё нужна. REV046-01 остаётся
обязательством будущего SQL runtime: Fact.Hash — hash immutable provider fact,
он намеренно не включает task binding. PR pointer version должен учитывать
Fact.Hash + normalized Match(task_key,reason) + Binding. Equal-time T1→T2 и
matched→missing/unknown/ambiguous обязаны дать order_conflict, а не same.
Unit проверки config/interchange не закрывают эту persistence regression.

SQL/schema5, gates, durable replay/import apply, outbox claims/CAS/recovery,
HTTP/CLI export/import wiring, реальный ingress/provider и DNS/delivery readiness
не реализуются здесь. Runtime E начинается после independent approve+QA
performance/schema4; эта preparation не меняет порядок допуска.
