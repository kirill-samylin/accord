# WORK246 — независимая диагностика B2V9

Статус: **FAIL сохранён**, причина классифицирована. Исполнитель — отдельный тестировщик `ops_validation_qa`; продукт и сценарий не изменялись, повторного запуска не было.

Проверен завершённый runtime `pipeline210-b2-1792801102978184000`: исходный `results.json` SHA256 `5387edbd653dd2a39d893172fa0b6f1d29384c085a97455757ba8983daf8e16f`. Server/outer — `a32db22973d5b43d4fb227a89337bf5b82a4969f`, worker — `be95daed7153f4abeb72012286cc748d0596b479`, SHA256 `9755ed0fb0de869751635ba4e5adf1a4d742b0a202e6659e871d337fceed27ff`. Все члены source manifest `df368afe4a5220ad75aee8c1a64c92237e80df0d527ed4fe02823fd2fc5f5a06` проверены по сохранённым хешам.

## Точная причина

`remote-control.cjs:100` ожидает внутренний текст «Нет локального разрешения владельца». Фактический сохранённый ответ — HTTP 403, `FORBIDDEN_ROLE`, публичный текст «Недостаточно прав для этого действия». Предыдущие проверки статуса и кода на строке 99 проходят. Контракт `internal/web/http.go:202–217` намеренно отображает этот код в публичное сообщение; `automation_action.go:67–87` и `browser_json.go:119–153` сохраняют его в JSON. Это неверное ожидание QA, а не ослабление авторизации. В `automation_control.go:71–105` проверка контроллера выполняется до версии/CAS: отказ 403 не подменён конфликтом устаревшей формы.

## Подтверждённое состояние

- Два turn одного настоящего native UUID, две разные board session и поколения 1/2. Обе остановки имеют совпадающие durable receipts, все пять exit/ACK/lock флагов и точные `run_exit` observations. Исторические start/resume остаются `delivery_unknown`, без переименования в applied; соответствующие stop — applied.
- Observer V5: `OBSERVED`, точный resume argv, доказанное владение, та же conversation, один poll / 7685 queries. Старая capability отвергнута 401/UNAUTHORIZED. Причина прежнего B2V8 NOT_OBSERVED остаётся UNKNOWN.
- Локальный revoke: версия host увеличена на 1, controllers 2→1; before/after automation и runs совпадают. Политика остаётся active с двумя control principals, отдельное локальное разрешение remote уже отсутствует.
- Закрытая board DB прочитана только `mode=ro&immutable=1` / `query_only` после проверки отсутствия всех companion-файлов. До/после совпали SHA256 и filesystem identity, companions не появились. Workflow совпадает с сохранённым `cancelled-1` после точной штатной redaction профилей; managed run полностью совпадает с `stopped-gen2`, все 4 commands и 2 attestations совпадают. Workflow cancelled/version14/epoch5; три задачи cancelled, leases/open questions/reservations — 0, journal pending — 0.
- 40 generic supervisor exits доказаны как 0 без forced; Next имеет известный доказанный exit130, не operation success. MCP закрыт отдельным подтверждением, его лог не содержит generic supervisor exit. Owner UI, remote UI, MCP и остальные joins подтверждены исходным cleanup; manual inspection false. Числовые native OS exits отдельно не архивированы, их не выдумываем.
- Исходный config comparison — `STRICT_UNCHANGED`; текущие глобальные настройки не читались. Сохранённые after-pins/source-after положительны; большие binary повторно не хешировались.

Прямые API readbacks **после** отказа не выполнялись: assertion100 находится до них. Отдельная проверка закрытого durable состояния подтверждает отсутствие business mutation; она не переименовывается в исполнение пропущенных API assertions. B2V9 не получает gate PASS. Проверка двух физических ПК, Windows и Claude — NOT-RUN.

## Замороженные артефакты

Private report: `.local/task014/evidence246/review.private.json`, SHA256 `26441b7da82ec93fb2527a8016105a6d5fdf2c2d7754c9f0f995b68756fdac03`.

Read index 160 ограниченных файлов: `.local/task014/evidence246/read-index.private.json`, SHA256 `fd11b94c904abbfab29fb8dc2a9c900c4f474804dd9ee2bb6ac64ca588893436`. Каталог 0700, файлы 0600; секретные поля не перенесены в отчёт. Исходные runtime/reports/DB неизменны. Дальнейшая исправленная версия требует отдельного source review и root GO.
