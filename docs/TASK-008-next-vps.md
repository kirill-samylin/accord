# TASK-008 — публикация Next.js клиента на существующем VPS

Дата20.10.2026. Пользователь явно поручил выкатить новую версию на VPS,
чтобы открыть новый интерфейс в работающем приложении. База `7d046d5`,
проверенный product source TASK007 `5cc0a88`. Статус: подготовка Linux release.

## Контракт

Существующий HTTPS origin и токены сохраняются. Новая страница `/app` становится
основным входом с `/`, native REST/MCP и `/web/**` остаются в Go. Nginx направляет
`/app`, `/_next/static`, `/fonts` и license assets в отдельный loopback Node service;
загрузки `/web/**` идут прямо в Go с прежними лимитами/Origin/CSRF. Next runtime
не получает ни БД, ни токены. Бизнес-схема/аренды/идемпотентность не меняются.

До переключения: актуальный read-only inventory, фиксированный release manifest,
проверка Linux runtime, nginx syntax/systemd verify, независимый review и backup
на самом VPS. Не копировать реальные БД/секреты в локальные artifacts или Git.
Один Go writer; atomic symlink/config switch. При сбое вернуть прежние binary,
nginx и service units; БД не восстанавливать поверх новых пользовательских данных.
Сохранить прежние backup/renew timers и частные маршруты. Tandem не подключать.

## Владение

- root — SSH inventory/apply, release/Git/TASK/docs, подтверждение HTTPS результата.
- WORK130 — deployment developer: `infra/accord/next/**`, связанные unit tests,
  `web/next.config.ts` только standalone packaging; своя WORK, private scratch.
- WORK131 — независимый reviewer: только своя WORK, чтение source/operators/
  manifest и evidence. Runtime/SSH не изменяет.
- WORK132 — отдельный tester: после product/operator APPROVE новые process/E2E,
  read-only live browser/API smoke (вход/выход своей сессии допустим), своя WORK.
  Не создавать fake задачи/дизайн на рабочей доске, не отзывать реальные токены.

## Приёмка

1. Проверенная immutable Linux Go+Next+Node поставка, systemd/nginx checks.
2. Backup/rollback path и независимое APPROVE до active switch.
3. Реальные HTTPS login, новый `/app`, основные страницы/themes/mobile,
   Cookie/CSP/static assets, native401 без bearer и authenticated read.
4. После restart Go/Node/readiness, backup/timers и старые маршруты сохранены.
5. Не заявлять физический второй ПК/fullV1/fullPaperclip parity по этому deploy.

## Результат

Read-only inventory подтвердил прежнюю установленную версию `ac6c965`,
Go127.0.0.1:7892, здоровые nginx/Go и backup/renew timers. VPS Linux x86_64,
около1ГиБ RAM; Node отсутствовал. Новый runtime — отдельный `accord-next`
без доступа к state/credentials, loopback7893; staging7894. Сборка выполняется
локально, Linux runtime проверяется на VPS до переключения.

Node24.19.0 Linux x64 получен с nodejs.org и проверен по официальному
SHASUMS256: tar.xz `14b342e71204f811bde6153be8e04b62aef63c236fef92b55f9c83154b409647`,
binary `bc17c508ffeed0ec622934f9b7fa72f8e78da65350e63c3eceb56fa688aa5e12`.
Next bundle содержит static/public, Node LICENSE и Linux optional modules,
проверенные по существующему npm lock; зависимости проекта не обновляются.

Клиентские format/typecheck и42unit — PASS. Остальные статусы будут заполнены
по фактическому review/stage/deploy/HTTPS smoke; локальный TASK007 PASS не
является доказательством production запуска. Секреты, БД и пользовательские
снимки остаются только в игнорируемом runtime или на VPS.

Первый candidate `80486fc`: source/tests/artifact review APPROVE, Linux
Go/Node stage PASS, backup/apply PASS. Реальный HTTPS browser smoke **FAIL**:
семь static/font HTTP429 помешали загрузке страницы входа. До board mutations
тест не дошёл. Выполнен реальный rollback: исходные pointer/config hashes
восстановлены, прежние Go/nginx здоровы, новый Node unit/pointer удалён;
БД из backup не восстанавливалась. Отчёт FAIL сохранён в `.local/task008-qa`.
Исправление: отдельный ограниченный bucket50r/s burst100 только для Next static,
fonts/license; прежние API/web/login лимиты и connection32 сохраняются.
Окончательный вывод требует нового независимого review и повторного HTTPS smoke.
