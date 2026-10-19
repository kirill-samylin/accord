# Независимое финальное ревью 083 — публичный HTTPS Accord

**APPROVE** новых проверок 081/082 и интегрированной клиентской документации.
Блокирующих замечаний нет. Проверяющий отдельный от авторов, модель
унаследована; закрепление Claude-моделей ролей здесь недоступно.

Scope: пользователь поручил обычный HTTPS с индивидуальными токенами без
Tailscale на клиентах. Production infra/security имеет отдельное APPROVE079;
083 проверяет новые тестовые оракулы, результаты и точность передачи клиентам.
Исходники Go, module files, Git и сервер reviewer не менял, credentials не читал.

## Проверенная версия

База репозитория `8b6eff14f2de1290e701950fcd3bb7db14a79539` + незакоммиченный
финальный diff. SHA всех 24 проверенных репозиторных файлов — локальный
`integrated-files.json`; SHA проверок/evidence — `reviewed-files.json`.
Карточка TASK после этого может дополниться координатором отчётом о закрытии.

- QA081 check.py: `0b802c564a8d4f82feb1b1e02a5946a40caf9d5dd2c7ee9559e720462b941f0c`.
- QA082 scriptnovpn.py: `a9167f5c891ddedc7e1cf513d830105121382b996a48922cd8b49d19f4518c86`.
- Effective renderer: `25095ba31f2f1349be45cbc0eba34049d2bc6923e351d262bf334fe5edf55ffd`.
- Effective renew unit: `07f84ea01d04d741f3aabceb07b5c2cac701d913ad3898c148e9ee1f7d161b27`.
- public-https.md: `6b2033e16b74052308ad08ff91e21c7733670cdf6557a1869792642b02cbf97b`.
- second-pc.md: `0ad0f002417f7e074435d2433455dd74c38c34fb557b0285f2f31c166ace83fa`.
- receiver-agent.md: `57b1a5eecb9ccdd3beb9da8d9d6b90e4512020ee4c560e2e7a54b7d563955920`.

`git diff 1e9025c -- '*.go' go.mod go.sum` пуст: исходники и module bytes
соответствуют живому C2. Бинарник на сервере остаётся SHA
`91ac8396cda97ee0577be6074eb9523f8ab8ce8beb7579cc52ec9684116ff1a2`.
Шесть infra/public файлов побайтно совпадают с окончательными effective
артефактами 079; два примера повторно получены из renderer с RFC documentation
IP, не содержат stream-only ssl_handshake_timeout и не являются deploy input.

## Проверки и результаты

Независимо выполнено ровно три изолированных probes, **3 PASS**:

1. QA081 через настоящий http.client encoding действительно отправляет
   `Origin: `; отсутствующий заголовок различается. Тест не теряет empty Origin.
2. Missing-token запрос не содержит Authorization, wrong-token содержит
   отдельный synthetic Bearer и идёт на POST /mcp. Это не одинаковые оракулы.
3. QA082 при BackendState Running фиксирует FAIL и выходит до сети/чтения
   токена. Stopped нельзя имитировать отсутствующим результатом.

Probes используют mock socket и synthetic значения, не сеть, файл токена или
доску. Полный живой E2E reviewer повторно не запускал по правилам проекта.

Сверены фактические результаты тестировщиков:

- QA081: **21 PASS + 1 FAIL** предпосылки TailscaleStopped при Running.
  FAIL сохранён и честно отражён в runtime-документе. Оба отдельных токена
  дали initialize200; missing/wrong401; stdio pc-2 дал точный каталог12.
  Pc-2 board-session не создавалась. Проверен только новый Mac epic E-5:
  get/events и отмена/get cancelled, чужие задачи не менялись.
  Origin foreign/null/empty403, Host/absolute authority refusal, spoofguard,
  65537bytes413, metrics404, HTTPAPI404. Ровно80 static ACME запросов:
  53×404/27×429, у ограниченных Retry-After1. Это bounded smoke, не SLA.
- QA082: **6 PASS/0 FAIL** с реальным BackendState Stopped, обычной системной
  проверкой цепочки/IP, readiness200, pc-2 HTTP initialize, native stdio
  initialize и tools/list12. Никаких tools/call/describe или board mutations.
- Restoration evidence: Running→Stopped→Running, exit0, prefs hash до/после
  равны, server Tailscale не менялся. Первый FAIL не переписан вторым прогоном.
- Серверное evidence подтверждает исходный binary release, readiness и
  неизменные старые приватные маршруты/Tandem. Полный серверный dump содержит
  actual address и остаётся private; в Git переносится обезличенный отчёт.

## Документация и границы

Новые client docs и receiver-agent указывают ordinary HTTPS, прежний токен,
новый origin из отдельно переданного JSON и обязательное изменение --server.
JSON не объявляется автоматически читаемым клиентом. Есть POSIX0600/0700,
Windows ACL/environment без literal secret, TLS без insecure, готовые binaries,
без требования Go, Tandem или нового сервера. Старые entry points имеют
заметный баннер на новые инструкции. Windows GUI environment не обещан.

Конкретный origin/токены не появились в проверенных public docs/infra;
root выполняет дополнительную секретную проверку Git без публикации значений.
Финальный клиентский ZIP в момент review ещё собирается: **reviewer NOT-RUN**
его CRC/nested binaries/forbidden-file scan. Эти проверки выполняет координатор
до передачи. Применение этого approve к непроверенному изменённому source
не допускается.

Физический второй ПК, Windows/Claude runtime, полный V1, D4/E5 и SLA —
**NOT-RUN/вне данного среза**, не объявлены готовыми.

Неблокирующая рекомендация для будущего повторного применения QA081:
обернуть cancellation в дополнительный finally для гарантированного b.close
при ошибке cleanup, а результат FAIL возвращать ненулевым exit status.
Текущий зачётный cleanup успешен и результат принят по JSON, а не exit alone.
Файлы тестов 140/108 строк, дублирование ограничено самостоятельными
одноразовыми сценариями с разными bounded scopes; общего нового framework
для этого этапа не требуется. UI не затронут.

Побочный эффект read-only проверки renderer: importlib создал ignored
`infra/accord/public/__pycache__`; координатор уведомлён удалить до упаковки.
Продуктовые/конфигурационные bytes reviewer не менял.

Следующий шаг — координатор проверяет итоговый клиентский пакет и секреты,
передаёт обновлённый connection.json отдельно и закрывает именно публичный
HTTPS срез. Реальный второй ПК подтверждает своё подключение самостоятельно.
