# WORK-036 — независимое ревью proxy Host и offline config

18.10.2026. Роли: reviewer + security; автор этого заключения не участвовал
в реализации WORK-035/033. Общий процесс, профили reviewer/security и AGENTS
прочитаны; именованные модели Claude здесь не закреплены, используется та же
доступная унаследованная модель. Tandem и live-системы не затронуты.

## Native proxy — APPROVE

Основа `99e722c44f83b1808120bba1fabb04cf2f68a5b5`, изолированная
`.local/worker-source-003`, ветка `codex/native-proxy`, frozen дельта WORK-035.
15 файлов source/tests/instructions проверены по manifest
`.local/worker-source-003/.local/evidence/proxy-host-003/manifest.sha256`:
`78684c73e660c511142a6b644b9ba1adc2fe868554391835eba1c8f0b8413cc5`.
Все 15 checksum совпали. WORK-035 не входит в этот source manifest.

Блокирующих замечаний нет. SDK v1.8.0 документирует и реализует
`DisableLocalhostProtection` именно для автоматической проверки loopback
listener. Внешний фильтр проверяет Host на всех путях до аутентификации:
canonical localhost/loopback либо ровно одна явно настроенная DNS:port authority.
Отсутствующий/неканонический/чужой Host, другой порт, conflict absolute-form и
дубли в вручную созданном Header отвергаются. X-Forwarded-Host/Proto не участвуют
в принятии решения. Неверная настройка fail closed; без настройки SDK guard
остаётся включённым. Поиск/вызов SDK handler остаётся за bearer и explicit
native Origin guard; лимиты, время ожидания и fencing не изменены.

Активы: bearer, задачи/история, loopback API и root-managed systemd deployment.
Недоверенные входы: Host/Origin/forwarded headers и запросы tailnet-клиента.
Доверенная граница — root задаёт точную authority, TLS завершает Tailscale Serve.
Клиент не может добавить разрешённое имя через свои заголовки. Сам Host не
является аутентификацией. Прямой bind по-прежнему только literal loopback.

Env-файл: строгая одна managed строка, root:root0600/каталог0700, symlink и
чужой формат не перезаписываются; exclusive private staging, file fsync,
atomic replace и directory fsync предшествуют activation. Omission/idempotent
повтор сохраняют настройку; systemd получает EnvironmentFile. Build копирует
новый helper, installer устанавливает его в root-owned admin. Ошибка fsync
прерывает установку; изменённая env-настройка не откатывается вместе с binary.
Этот предел и отсутствие remote MCP proof у старого binary rollback описаны.
Секреты/реальная private authority в diff не добавлены.

Фактические независимые проверки:

- PASS: targeted `go test -race ./internal/transport ./cmd/agent-board
  -run 'Test(ExternalHost|ExactHost|HostPolicy|MCPAdvertised|ServeRejects|Native|Origin)'
  -count=1`: transport1.378s, cmd1.298s.
- PASS: Python `test_host_config.py`, 5/5, 0.008s; реальные files/fsync,
  root ownership/directory policy — явно обозначенные test doubles.
- PASS: реальный loopback HTTP listener + official SDK initialize: configured
  Host200, неверные Host/port403, отсутствие bearer401, present Origin403.
- PASS: frozen source checksum и `git diff --check`; затронутые кодовые файлы
  до300 строк, новых необоснованных дублей нет.
- Полный make check не повторялся: автор WORK-035 зафиксировал полный PASS;
  ревью воспроизвело затронутые проверки, а не весь process suite.

NOT-RUN этим ревью: реальный Tailscale HTTPS после fix, systemd EnvironmentFile
на Linux, actual Codex и новые process/E2E. APPROVE относится к коду/безопасности
дельты; следующим этапом tester выполняет приёмку, координатор проверяет
конкретную собранную ревизию на VPS и настоящем клиенте.

## Offline configuration parser — APPROVE

MAIN рабочая копия Accord: только frozen `internal/config/**`, schema/templates,
go.mod/go.sum и WORK-033; C store/transport WIP в область этого заключения
не входит. Manifest `.local/evidence/config-parser-003/manifest.sha256`:
`2ba42fc38fd702073bbaa68c3f16b3fda46f864cc1d27f2746e989db2b5ac3cc`.
Все 21 checksum совпали. Новая dependency только `go.yaml.in/yaml/v3 v3.0.5`;
остальные pinned зависимости не изменены.

Блокирующих замечаний нет. Проверены strict AST до decoding, UTF8/byte/depth/node
лимиты, один документ, duplicate/nonstring keys, anchors/aliases/tags/merge;
shape неизвестных полей и semantic role/user/tool refs; TTL bounds, unsupported
capabilities/safety flags; независимость snapshot/default copies и canonical
roundtrip. Schema/template mirrors совпадают embedded assets. Собственных IO,
network/remote schema loader, env expansion или shell нет. Отдельный disposable
shape view использует float64, но canonical config/evidence numeric constraints
сохраняют json.Number и сравниваются exact big.Rat с ограниченным exponent.

Restricted evidence schema проверяет vocabulary, local refs, cycles, depth,
node/property caps и Go regexp; URI/date-time assertions выполняются при
validation. Builtin types/byte limits/required floor не отключаются mode;
missing extra evidence при warn/off остаётся Complete=false. Серверная
provenance/current SHA/generation/positive gate отделена и явно не объявлена
реализованной этим parser. Tool budget честно помечен как conservative UTF8
bytes, schema bytes выдаются отдельно. Path/alias resolution не обращается к ФС.

Фактические независимые проверки:

- PASS: `go test -race ./internal/config -count=1`, 6.097s.
- PASS: `go vet ./internal/config`.
- PASS: отдельный reviewer probe в ignored `.local/evidence/review-036/`:
  escaped local ref с exact >2^53 bound, огромный exponent, неправильный тип
  array item, exact numeric const; все четыре обхода отвергнуты.
- PASS: отдельные deep12000 input/alias/tag/duplicate probes отвергнуты без
  panic; полный документ оставался внутри byte cap.
- PASS: checksum21/21, owned diff whitespace; все Go файлы ≤233 строки.

Рекомендация P3, не блокер: `internal/config/schema_helpers.go`, `resolveRef`
заменяет ~1/~0, но не отклоняет прочие ~escape. Можно дополнить строгой RFC6901
проверкой: ref `#/$defs/a~2b` должен считаться malformed даже при literal key
`a~2b`; сейчас такая ссылка локальна и применяет найденную схему, поэтому
разрешённую область и constraints не обходит. Регрессия на invalid tilde escape
улучшит строгость диагностики.

NOT-RUN: schema3 migration/config CAS/history/live reload, scoped role grants,
реальный bridge с custom role и process apply conflicts. Их здесь нет по
согласованной границе WORK-033; parser approve не означает runtime config PASS.
Следующий интегратор использует docs/config-contract.md и отдельное ревью
изменённой store/CLI/bridge версии перед live apply.
