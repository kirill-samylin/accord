# GitHub facts foundation

Ingress вызывает Verify(headers,raw,secret) до Decode. Secret≥32bytes; raw≤1MiB,
application/json, absent/identity Content-Encoding; duplicate signature/event/
delivery headers запрещены, включая неканонизированные key casing. HMAC exact
raw bytes; payload/signature/secret не входят в ошибки. Ingress сам ограничивает
чтение raw body до allocation, preauth rate/timeouts и disabled404.

Decode допускает unknown provider fields, но duplicate JSON keys/invalid UTF8/
depth>32 запрещены. Разрешены PR opened/synchronize/closed, review submitted,
check_suite completed; ping без board effect, другие signed types/actions ignored.
Repository exact allowlist; base repo обязан совпадать; fork head не доверенный
источник. URL конструируется из allowed repo/number, payload URL не используется.
Facts содержат только whitelist полей, int64 IDs и exact SHA40/64; полнота unknown.
Review fact не становится core Review/approve/AuthorSession и не меняет head.

Policy — config snapshot + compiled Go regexp + map upper-case matched key→
существующий canonical key и current known bindings. Matcher ограничен input1MiB
и1024 matches; сверх лимита консервативно ambiguous, а не удачный префикс.
Check suite использует только exact repo/PR/SHA bindings и branch keys.
Непривязанные facts сохраняются caller-ом даже для missing/unknown/ambiguous key.
Поздний PR может потребовать reattachment старых CI facts: это store owner.

ClassifyReplay не хранит dedup; store ищет ID **и** event+bodyhash в одной write tx.
CompareVersions только классифицирует source-time/hash; разные equal/missing times
дают order_conflict. Store сохраняет все immutable facts, ведёт conflict pointer,
историзирует заменённый binding и не использует arrival/GUID/max suite ID.
Применять comparator можно только к версиям одной semantic identity (repo/kind/
object_id/SHA/app_id); current PR pointer отдельно сравнивается across SHA.
Ни HTTP GitHub API, ни storage/core gates библиотека не вызывает.
Commit uncertain/error не считается успешной webhook receipt: HTTP2xx выдаёт
только интеграционный слой после durable commit. Здесь transport endpoint нет.
