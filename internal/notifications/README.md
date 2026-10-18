# Signed bounded notification attempts

Encode принимает явный Payload с stable delivery_id/event_cursor/ts и только
четырьмя согласованными events. Description/answer/evidence отсутствуют в DTO.
Task URL сейчас запрещён: trusted browser origin ещё не настроен.
Body≤32KiB; NewRequest подписывает exact stored body текущим secret≥32bytes,
X-Agent-Board-Delivery совпадает с ID в body. Retry не изменяет raw bytes/ts/ID.

NewSender(nil,nil) использует system DNS/net.Dialer и отдельный transport на
каждую попытку. HTTPS443 ASCII canonical hostname без credentials/query/fragment;
DNS перед connect, все ответы проверяются, mixed/private/meta/reserved/mapped
отклоняются. Диапазоны IANA special-use запрещены целиком, включая отдельные
special-use globally reachable allocations; IPv6 только global2000::/3 кроме
special-use. DNS-IP пинуется; TLS ServerName+Host исходные, chain validation
включена. Env proxy/cookies/redirect/decompression/keep-alive отключены.
Timeout5s total, connect/TLS/response headers≤2s, headers≤16KiB, discard≤4KiB.
Политика адресов требует пересмотра при новых IANA allocations.

Result.Delivered означает **получен HTTP2xx**, не durable outbox ack. Retry для
network/408/429/5xx; Dead для policy/invalid-message/3xx/остальных status.
Возвращаются только классы/status/RetryAfter без URL/body/error strings.
RetryDelay — initial+5 retries (attempt1..5 дают 1/2/4/8/16m), jitter0..20%;
Retry-After может увеличить delay до1h. Caller обязан persisted next_at/jitter,
claim_generation CAS, recovery/expiry/dead audit и ограничение2 concurrency/20batch.
Library не делает SQL, worker loop или подписок на board events.

Synthetic tests инжектируют resolver/dialer: public address проходит production
policy и только затем test dialer направляет соединение на local TLS fixture.
Test CA известна только тесту; SNI/Host/chain проверяются обычным TLS transport.
Реальных provider messages нет. Инъекции — доверенный внутренний API;
конфигурация/агенты не должны иметь доступа к замене resolver/dialer.
