# WORK-023 — защита native HTTP и эксплуатационные метрики

- Родитель: [TASK-003](../TASK-003-working-service.md).
- Роль: разработчик; исполнитель `transport003`.
- Статус: выполнено; передано на независимые code/security review.
- Основание: [security-contract](../security-contract.md), SEC-07–09;
  [service-contract](../service-contract.md), общий bearer-сервис.
- База: `966dda6`; проверенная дельта идентифицирована SHA256 ниже.
- Именованные Claude-модели в текущей среде недоступны; профиль передан путём,
  закрепление моделей не действует. Самопроверка не является approve.
- Область: только `internal/transport/http*.go` и эта WORK. CLI, SDK MCP,
  хранилище, credentials и конфигурация VPS не менялись.

## Результат

`NewHandler` сохранён; `NewHandlerWithOptions` позволяет задавать общий rate,
burst и concurrency. Defaults: 100 requests/s, burst200, max64 одновременно.
Неположительные/нечисловые rate и неположительные остальные опции используют
defaults; отключения защиты нет. Один token bucket и semaphore имеют размер,
заданный конфигурацией; нет map по bearer, IP или forwarded headers. Лимитер
стоит до аутентификации/SQLite, также ограничивает health/ready. Rate429
содержит целый `Retry-After` с округлением вверх; saturation503 — `Retry-After:1`.
Отклонённые запросы не резервируют будущие токены; panic освобождает concurrency.
Это ограничение частного сервиса, не обещание DDoS-защиты.

Для `/api`, `/mcp`, `/metrics` и их подмаршрутов любой присутствующий Origin
отвергается403 до аутентификации, включая GET, пустой/null/malformed, несколько
значений и совпадение с Host. Forwarded headers не дают доверия. Native запрос
без Origin работает; existing CrossOriginProtection и MCP SDK localhost-защита
сохранены. Будущие web-маршруты не получают native Origin policy автоматически.

Ответы сохраняют no-store, дополнены nosniff. Native endpoints дополнительно
дают Referrer-Policy:no-referrer и CSP `default-src 'none'; frame-ancestors 'none'`.
Эти headers выдаются также на отказы Origin, bearer и admission. Тела/ошибки
ограничиваются прежними правилами, panic/token не выводятся. CORS не открыт.

`GET /metrics` требует действующий bearer и не требует board session. Семь
фиксированных unlabelled series: uptime, total requests, rate-limited,
concurrency-rejected, active requests, runtime heap/sys bytes. Active включает
текущий scrape. Нет token/owner/session/task/path labels. Query/body отклоняются;
неверный метод405. Go runtime Sys **не RSS** и так не называется. Измерение RSS
процесса на целевой ОС остаётся отдельной эксплуатационной проверкой.

## Проверки

На доступном Mac с Go1.27.2 darwin/arm64, локальными изолированными SQLite
fixtures; живой VPS и Tandem не нагружались:

- `go test -count=1 ./internal/transport` — PASS,0.962s на первой полной дельте.
- После добавления revoke/recovery regression: `go test -race -count=1
  ./internal/transport` — PASS,3.008s, включая все старые HTTP/MCP tests.
- `go vet ./internal/transport` — PASS.
- gofmt — PASS; все изменённые Go файлы ≤269 строк.

Новые проверки: presence-Origin на native GET/POST с различными значениями;
пустой/noncanonical/multiple header; forwarded-host не помогает; будущий web
не заблокирован native policy; auth не вызывается на Origin/rate/concurrency
отказах; fixed bucket восстанавливается без накопленных reservations;
panic cleanup/redaction; heartbeat после реального429 и refill; bearer metrics
без session, revoke, отсутствие labels/идентификаторов/secret, query/method,
security headers и минимальные public health/ready.

Не запускалось: удалённая нагрузка и TLS proxy/client acceptance. Новые
process/E2E — отдельный tester после независимого approve.

## SHA256 проверенных исходников

```text
92c546f3db43ebe9c35113c576eb641d2c5aace11e9bb40bcc43eb181c744262  internal/transport/http.go
a1c041cd791f2f8e561a84229c2c7e0dc39229c53439e76d965c2dd2081be11f  internal/transport/http_security.go
665257e2623b14da2db31bb9b591e1682079924101df73daecda6df25e57679e  internal/transport/http_metrics.go
f662789e3939921e22b31840faa19d57c031a429d1e2662829a88a863cc1ba88  internal/transport/http_security_test.go
5269affcc23066fc2a454f4850d50e03cb323fc8f4f5dadc00bd2d89bddb978a  internal/transport/http_metrics_test.go
4330694e0c5746e6b67428505082b772834ca4531e7be927fe84b96e61a814c7  internal/transport/http_recovery_test.go
```

## Журнал

- 2026-10-18T17:32:19+03:00 completed: `transport003` передал реализацию и
  проверки координатору. Независимый approve и live rollout ещё не выполнены.
