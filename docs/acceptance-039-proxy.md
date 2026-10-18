# WORK-039 — приёмка native proxy Host

18.10.2026, tester `/root/proxyqa003`. **PASS** для описанного native delta;
новые тесты передаются независимому reviewer. Изолированная рабочая копия
`.local/worker-source-003`, commit `dc9c74cd692e287173def9fb5743dbf18ffdef21`,
parent `99e722c44f83b1808120bba1fabb04cf2f68a5b5`. Начальный snapshot чистый;
все164 существовавших tracked файла остались побайтно неизменными после прогона.
Актуальное independent code/security APPROVE — [WORK-036](review-036-proxy-config.md).
Основной C WIP не собирался, не проверялся и не изменялся тестировщиком.

## Тест-кейсы, набор версии1

В native worktree добавлен только
`tests/acceptance/proxy_process_test.go` (284 строки). Использованы прежние
TestMain/board cleanup/token/MCP helpers; production source не изменён.
Данные и токены синтетические, каждый server запускается на ephemeral loopback
порту с отдельной private SQLite; процессы завершаются через owned cleanup.

| ID | Шаги для воспроизведения | Ожидаемое поведение |
| --- | --- | --- |
| PX01 | Запустить настоящий CLI с `--external-host accord.synthetic.example:8443`; локальный TLS reverse proxy передаёт этот Host серверу через socket. Official SDK выполняет initialize, tools/list, describe, create, get. | Initialize и11 инструментов доступны; session identity сохранена, get возвращает созданный epic. Противоречащие X-Forwarded-Host/Proto в proxy не мешают точному разрешённому Host. |
| PX02 | К тому же CLI подключить SDK напрямую по loopback, с отдельным HTTP transport. | Initialize успешен: внешний адрес не ломает локальный доступ. |
| PX03 | Направить HTTP на actual socket с чужим Host, другим/отсутствующим портом, включая `/mcp`, `/api/v1/describe`, `/metrics`, `/healthz`. Вставить разрешённое имя в X-Forwarded-Host и Forwarded. |403 до auth; forwarded metadata не разрешает чужой authority. |
| PX04 | Правильный Host: `/mcp` без bearer; `/mcp` с bearer и present empty Origin; `/metrics` с/без bearer; `/healthz`. | Соответственно401,403,200/401,200. |
| PX05 | Отдельные CLI процессы получают URL/wildcard/portless flag либо env с неканоническим портом. | Быстрый ненулевой exit, без timeout, без startup stdout и без создания БД. |
| PX06 | Запустить CLI только с валидной ACCORD_EXTERNAL_HOST; затем с invalid env, перекрытым валидным явным flag. | Правильный внешний Host возвращает health200 в обоих случаях. |
| PX07 | CLI без external host; отдельный настоящий stdio CLI worker делает describe/events; затем внешний Host на `/mcp`. | Developer session и события доступны, stdout содержит только MCP и не содержит bearer; внешний Host403 без настройки. |

## Протокол прогона

Окружение: macOS/darwin arm64, Go1.27.2. CLI действительно собран TestMain и
запущен отдельным процессом. TLS proxy — локальный `httptest.NewTLSServer` с
проверкой его сертификата штатным client, backend — настоящий отдельный CLI
через TCP. Это не тест реального Tailscale и не Codex. Никаких замен store,
clock, HTTP handler или authentication в этих процессных сценариях нет.

| Прогон | Команда из native worktree | Результат |
| --- | --- | --- |
| Normal | `go test -json ./tests/acceptance -run '^TestProxyProcess' -count=1` | PASS,4 верхних теста и17 subtests,1.637s |
| Race harness | `go test -race -json ./tests/acceptance -run '^TestProxyProcess' -count=1` | PASS, те же4+17,2.743s |
| Static | `go vet ./tests/acceptance`; `gofmt -l tests/acceptance/proxy_process_test.go`; `git diff --check` | PASS |

`-race` инструментирует Go test harness и связанные с ним пакеты. Дочерний CLI
в обоих прогонах обычный, без `-race`; одинаковый binary SHA256
`d2816118a41dc72948c9788f5856753a3340c090bb9f80c3b37f7ec5432b67a8`.
Это отдельная tester-сборка, не Linux/Mac release artifact координатора.
Полный make check заново не повторялся: автор native delta зафиксировал полный
PASS, WORK-036 воспроизвёл затронутые проверки; здесь проверен новый process delta.

Первые незачётные прогоны обнаружили два дефекта самой fixture: describe machine
не совпадала с token machine; общий protocol helper требовал минимум3 frames,
а тест успевал только initialize/describe. Исправлены синтетическая machine и
добавлен полезный events read. Production source не менялся. Исходные fail-логи
сохранены отдельно; финальные normal/race JSONL не содержат failed tests.

## Доказательства

Все пути ниже относительно native worktree `.local/worker-source-003`;
сырые доказательства остаются ignored. Итоговый JSON содержит версию, список
всех пройденных тестов, время, binary/log hashes и ограничения.

- `.local/evidence/proxy-acceptance-003/initial-snapshot.json` — clean HEAD/parent.
- `initial-source.sha256`:164 tracked файла, manifest SHA256
  `934a94fc8c3bc825ba7c924d2a3aade780bb6fecdd1c2d8a4e65cb6749db143b`.
- `initial-parent.diff` SHA256
  `215fe50ba100a5b0ae2f4489586ffe4558953d1a886f044b023249501c5617a7`.
- `normal.jsonl` SHA256
  `a71436ba884fb52d6cf4b449e7fb3cad33a379c85f7bda1d76ac68b69ecaee59`.
- `race.jsonl` SHA256
  `88c2587931dafaf0a4f6dfe038561d91439a44baa64556087267240b6e4bb4fd`.
- Test source SHA256
  `58b00ee32775350b3d11b189805f24e7c431b908f0aabaf61207f70d4a568ee5`.
- Result manifest7 SHA256
  `9a4d09652ad3b4ebd7c71cae06885c4c9e4df53ad5c0b91deb6e8687bf27890a`.

## Отдельный runtime результат и границы

Координатор сообщил и сохранил в основной копии sanitized
`.local/evidence/codex003/summary.json` и
`.local/evidence/runtime-server-003/proxy-update.json`; эти два metadata файла
прочитаны tester, но live/model повторно не запускались. В summary указан тот
же commit, remote real Codex10 успешных calls, synthetic E1/T2 cancelled,
отзыв test token подтверждёнHTTP401 и локальный plaintext удалён; log SHA256
`72b3bed5884cb7b64b46a49d23bdf61f5e952562d1707845f151e20df3984160`.
Runtime metadata: HTTPS ready200/MCP initialize200, Tailscale routes и metadata
Tandem unchanged. Это результат координатора, отделённый от SDK приёмки tester.

NOT-RUN здесь: Windows/другая архитектура, второй физический ПК, Claude Code,
полный V1/config/C WIP, live deployment/restore. Новые tests требуют независимого
review перед интеграцией; перенос в основную ветку и дальнейший приёмочный gate
выполняет координатор. Секреты, внешние private адреса и реальные данные в
тесты/отчёт не включены.
