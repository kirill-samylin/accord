# TASK-008 — публикация Next.js клиента на существующем VPS

Дата20.10.2026. Пользователь явно поручил выкатить новую версию на VPS,
чтобы открыть новый интерфейс в работающем приложении. База `7d046d5`,
проверенный product source TASK007 `5cc0a88`. Статус: **публикация завершена**.
На VPS установлен `490bf55e59b16c32817cb2ad4b9f7da7e1df13d9`.
Это приёмка данного развёртывания, не завершение полной V1/Paperclip parity.

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
  `web/next.config.ts` standalone packaging; затем root явно расширил область
  на отключение prefetch у28 Next Link в11 UI файлах; своя WORK, private scratch.
- WORK131 — независимый reviewer: только своя WORK, чтение source/operators/
  manifest и evidence. Runtime/SSH не изменяет.
- WORK132 — отдельный tester: после product/operator APPROVE новые process/E2E,
  read-only live browser/API smoke (вход/выход своей сессии допустим), своя WORK.
  Не создавать fake задачи/дизайн на рабочей доске, не отзывать реальные токены.
- После прерывания: WORK133 — независимая сверка продолжения и closure;
  WORK134 — отдельный tester, запуск неизменного approved browser script.

## Приёмка

1. Проверенная immutable Linux Go+Next+Node поставка, systemd/nginx checks.
2. Backup/rollback path и независимое APPROVE до active switch.
3. Реальные HTTPS login, новый `/app`, основные страницы/themes/mobile,
   Cookie/CSP/static assets, native401 без bearer и authenticated read.
4. После restart Go/Node/readiness, backup/timers и старые маршруты сохранены.
5. Не заявлять физический второй ПК/fullV1/fullPaperclip parity по этому deploy.

## Результат

Установлены новые Go binary и Next standalone под отдельным системным
пользователем. `/` направляет на `/app`, старый `/web/` и native REST/MCP
сохранены. Серверные данные/токены не переносились локально и не сбрасывались.
После итогового backup journal имеет `phase=complete`, оба сервиса и nginx
active/running, все3 backup/renew timers active/waiting. Публичные HTTPS после
finalize: app200, readyz200, native без bearer401. Частная маршрутизация и
protected config hashes совпадают с inventory; Tandem не подключался.

| Проверка | Фактический результат |
| --- | --- |
| Source, новые unit/process/E2E и immutable artifacts | Independent APPROVE131/133 |
| Формат, TypeScript, клиентские unit, fresh standalone build | PASS42/42, Go upstream7892 |
| Полные infra unit | PASS84/84; nginx/native/upload/Origin guards и rollback |
| Linux staging Go + Node/native libraries/nonce/systemd/nginx | PASS; isolated synthetic Go fixture удалена |
| Production HTTPS/native/browser | PASS8/8, production-04, Chrome155/Node24 |
| No-prefetch и переходы |0 speculative requests при viewport/hover; явный click JSON200 |
|10 экранов ×2 темы ×1440/390 |40 PNG, hashes проверены root/reviewer, фактический visual PASS134 |
| Network/runtime |114 app/web responses200,11 assets,429/page/CSP/static errors0 |
| Вход/выход | Existing token, secure cookie, own logout→401; прочие board mutations0 |
| Backup/rollback | Реальные rollback01/02 PASS; согласованные snapshots до apply и после smoke |
| После finalize | Go/Node/nginx0 restarts, readiness200, private routes/env/timers сохранены |

Финальные Linux Go SHA256
`f1fc49d74ecc44da5d0bb522ad6f2f5a0624e59e70ba9104290339cc0670009f`,
Next1250-file manifest
`9fec4a1852fd60677ab61e7892ac05b406b6d5b254225907b1d268d553dacdb4`,
BuildID `FXxaQO59Zj0_bmIx40KNi`. HTTPS report SHA256
`641ce01b2596811f88da96903eee223fdb870b8729dbe9840fab692500d46c1c`.
Final backup manifest SHA256
`cc182b68ee33a017cf0a53178c0be741a2c157714d9c4c6723dc3e3ede70db59`;
snapshot хранится только на VPS, directory0700; backup service завершился success0.
Next после smoke около55МБ, peak57.5МБ, без перезапусков.

Evidence: ignored `.local/task008/{release-proof-03,linux-stage-proof-03,
linux-go-proof-03,https-04-root-proof,final-private,post-finalize-https}.json`,
`finalize-03.log`; browser `.local/task008-qa/production-04/`.
Подробности: [WORK133](work/WORK-133-next-vps-resume-review.md),
[WORK134](work/WORK-134-next-vps-resume-qa.md).

Изменённые области: `infra/accord/next/**`, tracked-only packaging в
`infra/accord/build-release.sh`, `web/next.config.ts`,28 Link props в `web/src/`,
`tests/infra/test_next*.py`, `tests/integration/next_vps_*`, эта TASK, next-web и
WORK130–134. Домен Go/SQLite/auth/leases не изменён этим deployment этапом.
Git локальный, `codex/next-vps`; push/merge не выполнялись.

Физический второй ПК/телефон, внешние Codex/Claude Code и production
task/upload/lease mutations — NOT-RUN в этом read-only smoke. Synthetic mutation
acceptance остаётся TASK007. Secondary widget в overview PNG был снят в loading;
отдельное завершение каждого такого widget этим screenshot gate не доказывается.
Блокирующих visual/runtime findings нет. Не заявляется полная Paperclip parity.

## История кандидатов и исправлений

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

Локальный TASK007 PASS не использовался как замена production запуска.
Секреты, БД и пользовательские снимки остаются только в игнорируемом runtime
или на VPS. Исторические FAIL ниже сохраняются независимо от финального PASS.

Первый candidate `80486fc`: source/tests/artifact review APPROVE, Linux
Go/Node stage PASS, backup/apply PASS. Реальный HTTPS browser smoke **FAIL**:
семь static/font HTTP429 помешали загрузке страницы входа. До board mutations
тест не дошёл. Выполнен реальный rollback: исходные pointer/config hashes
восстановлены, прежние Go/nginx здоровы, новый Node unit/pointer удалён;
БД из backup не восстанавливалась. Отчёт FAIL сохранён в `.local/task008-qa`.
Исправление: отдельный ограниченный bucket50r/s burst100 только для Next static,
fonts/license; прежние API/web/login лимиты и connection32 сохраняются.
После этого потребовались новые review и повторный HTTPS smoke.

Candidate02 `1518248`: отдельная asset zone устранила static429. Linux stage и
повторный apply PASS; production02/03 дошли до успешного входа, защищённой cookie,
native authenticated read и MCP12, но **FAIL** при переходах. Diagnostic03 показал
nginx429 на app/RSC и `/web/tasks` после viewport prefetch и частых full-page
переходов теста. Это не утверждение о сбое каждого обычного входа: повторные
переходы усилили нагрузку. До finalize выполнен rollback без восстановления БД.

Финальная узкая UI дельта отключила автоматическую предзагрузку всех28 Next Link;
разделы открываются явным нажатием. Лимиты native/web/connection не повышаются.
Авторский format/typecheck/42unit/production build, independent source/test
review и реальный HTTPS regression без prefetch burst — PASS на490bf55.
