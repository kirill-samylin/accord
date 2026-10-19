# WORK-094 — интеграция и выпуск веба Accord

- Исполнитель: /root, координатор/интегратор; модель наследуется.
- Источник: прямое поручение19.10.2026 о рабочем вебе для людей и агентов.
- База: f063380. Работа в отдельном codex/web-service; Tandem OFF.
- Статус: выполнено. Независимые review/QA и фактический production выпуск PASS.
- Владение: cmd wiring, transport extra/MCP, bridge resources, core/store history
  и questions, infra web backup/optionalnginx, интеграционные unit tests и docs.
  После передачи авторов также исправлены пагинация UI, OShelpers и browser header.

## Реализация

Один процесс SSR + native с общими Host/admission/recovery пределами, отдельная
web.db/schema1, board schema2. Строгий native Origin deny и12MCP tools сохранены.
Защищённые browser cookies не авторизуют native routes. Web-data и origin opt-in
через flags/env. MCP design resources metadata и stdio proxy с worker-bound session.
Resource reads дизайна не запускают heartbeat. Старые native workflows совместимы.

Human ответ записывает immutable human_answered с текстом и provenance в той же
board transaction, включая native human completion и exact replay. Новое чтение
ListDecisions newest-first с epicfilter не меняет native ascending Events.
ListQuestions — native queue order с ограниченной страницей и creation boundary;
чтение не claim/heartbeat. Legacy fallback честно ограничен фактически сохранённым
текущим ответом. Исторические неизвестные тексты/авторы не выдумываются.

Uploads/SSR реализованы отдельными авторами087/088/089. Root закрыл найденную090
пагинацию101-го эпика и заметил portability: Unix owner/nofollow/freedisk helpers
в platform files, Windows native client компилируется, unsupported web-server OS
отказывает до обслуживания файлов. На Linux/macOS гарантии ownership/nofollow сохранены.
Фактический UI-review092 поймал browser form Origin:null из-за no-referrer.
Browser policy изменена на same-origin: Origin/CSRF остаются строгими, внешнему
сайту Referrer не отправляется. Native design API сохраняет no-referrer.
Источник поведения: WHATWG Fetch §append-a-request-origin-header.

Новый offline daily snapshot board+web+blobs под общим maintenance lock с короткой
остановкой единственного writer, manifest/integrity/hash/reference verification,
7копий,512МиБ diskfloor. TERM/INT превращаются в исключение для finally restart.
Root-private snapshots/директории; строгая проверка дерева без symlinks/лишних
путей. Restore удаляет browser/preauth/actions и требует existing native recovery
с отзывом восстановленных credentials. Предыдущий hourly board backup сохраняется.
Optional nginx --web сохраняет native64КиБ и plaintext404, увеличивает только
browser upload52494336байт/120с, перезаписывает доверенныйclientIP на loopback link.

## Фактические проверки до QA

- make check: PASS fmt,vet,Go packages и existingprocess acceptance,74infra,
  race и build. Private log /private/tmp/accord-web-084/make-check.log.
- После portability/header: targeted tests и Windowsamd64 build; итоговая
  допроверка/5platform builds фиксируется в завершении этой WORK.
- source scan354files:0 совпадений с реальными machine tokens, endpoint/SSHkey.
  Никаких секретов или реальных адресов в Git не добавлено.
- Независимое code090:approve baseline с четырьмя исправленными findings;
  OS/header delta дополнительно проходит независимый code/security091.
- Самопроверка не является independent approve. Live/browser/restore/rollback
  в этом разделе не отмечаются PASS до фактического выполнения.

## Ограничения

Не включает невыпущенные D4/E/schema4/5, полнуюV1/SLA или повторную нагрузочную
приёмку. WebP/PDF/SVG/HTML/ZIP только защищённое attachment, preview PNG/JPEG.
Native clients Linux/macOS/Windows; файловый web-server поддержан Linux/macOS.
Физический второй ПК — сообщение пользователя «вроде подключился», не нашPASS.
Offsite backup отдельно не настроен. Gitmain не сливается без поручения.

## Проверенные дополнения и подготовка

- WORK092 actual Chromium desktop/mobile: APPROVE после исправления формы входа
  и whitespace ответа; root добавил pre-bind400/русское поле, safe DOM textContent
  и aria-invalid. Независимая дельта WORK093: APPROVE, границы16KiB/4KiB,
  отсутствие binding/изменения lease при validation и corrected retryPASS.
- Production source зафиксирован6245829; отдельная clean worktree собрала
  checksummed Linux release и5native clients: darwinarm64/amd64,linuxamd64/arm64,
  windowsamd64. ZIPCRC/hashPASS, actualmachine secrets/endpoint scan0.
- Artifact directory вне Git содержит готовые clients и отдельный адрес; token
  в него не входит. Primary working branch ещё не переключалась.
- DevOps096stage-only: release manifestlocal/remote,nginx-t и systemdverifyPASS;
  активная версия пока прежняя. Reviewer097 дополнительно проверяет packaging
  assets и deploy/rollback operator, включая fault injection config restore.
- QA095 выполняет separate realserver/browser/nativeworker/backup/rollback/
  byteboundary/slowdeadline. Выпуск и итоговый статус ниже только после фактов.

## Возврат QA: переносимость offline snapshot

Actual Go fixture и /usr/binPythonSQLite3.54 выявили retained WAL/SHM в destination,
не проявившиеся на initial syntheticDELETE DB. Первый narrowerfix удалил WAL,
но оставил SHM; обе failed попытки сохранены отдельно, не объявленыPASS.
Root окончательно нормализует offline target DELETE, проверяет header1/1,
удаляет только newly-created closed ownedregular0600 sidecars (WAL лишь0),
fsyncdir. SourceimmutableRO не создаёт root-owned source sidecars; nonempty
sourceWAL отклоняется без потери committed frames. 2новых regression tests:
реальный WAL→repeatedvalidate/records/noextras и отказuncheckpointedWAL.
7backupunits PASS на Python3.14 и systemPython3.9/SQLite3.54. Независимый097
re-review и095actualrestore обязательны; предыдущий staged624helperнеактивируется.
Go/web/client source624 неизменён, новый релиз обновит только infra helper/docs.

## Фактический guarded выпуск: возврат и root entry

Первый apply остановлен до мутаций строгим manifest: Mac tar добавил31AppleDouble
sidecar. Проверены magic00051607, regular paths ._*; extras перемещены в private
quarantine неактивного релиза; все26checksummedfiles неизменны. Следующий apply
автоматически откатился из-за root303/web/→401 для гостя. РеальныйrollbackPASS:
oldnativeC2ready, точныеTandem/fullServe/server.env unchanged, newweb files preserved.
Root исправил только guest entry: rootGET/HEAD→login, validcookie→home, storageerror503.
Independent098 APPROVE +4targetedtests; остальныеnative/webmutations неизменны.
Для нового clean release используется явный Python tar manifest (без Macxattrs),
проверяется точное множество файлов до apply. Новый actualHTTPSsmoke обязателен.


## Итоговый выпуск и закрытие TASK-004

Production release `d9aa4c745a4c2038ba22bdc7d92f89dded63ad7f`, Linux binary SHA256
`414996d7d0c802e7260beecbf632f7cac8b93c96dfb2e09c97244411fc718ff5`.
Чистая сборка, sealed Python USTAR transfer27files, strict manifest, nginx syntax,
systemd verify PASS. Mac AppleDouble больше не переносились.

Повторный apply выявил отдельную operational race: сразу после nginx reload
readyz200, но root404 от прежних workers. Diagnostic trace сохранён; auto rollback
снова полный, без ошибок. Operator получил bounded public readiness:20attempts
с timeout5s на запрос и интервалом0.5s, постоянный отказ по-прежнему ведёт к
rollback. Independent097 exact reverse-diff +6mocked regressions APPROVE.
Новый frozen operator SHA256
`3c934e54af980dccad37878dcc980a939f9e66e532287199e64664830aefb697`,
неизменный driver `b709039116c969b8f774532b69bdd3e1ecddefe801321f23cd20f5ef58f5651d`.
Remote operator hash сверён до actual apply. Все прежние неудачи сохранены
отдельными evidence; они не засчитаны за PASS успешного выпуска.

**Actual guarded apply и finalize PASS.** Trusted public HTTPS, реальный Chrome,
нормальный file-token form login303 и Secure/HttpOnly/Strict cookie, четыре
страницы200, logout303. Нативные проверки: оба существующих machine tokens,
12tools, design templates/metadata, unauth401, emptyOrigin403 и отсутствие native
доступа по одной cookie. Один служебный1px PNG сохранён в проектной библиотеке;
браузерный upload→native attachment download побайтно совпал, SHA256 совпал.
Всего29 production smoke checks PASS, raw secret/cookie в evidence не записаны.

Согласованный daily snapshot реально выполнен и повторно validated; snapshot и
web tree приватны и принадлежат правильным пользователям. После backup restart
service/HTTPS/readiness healthy, установленный binary/manifest соответствует
указанному release. Старые hourly backup/certificate timers и новый daily timer
активны. Полная Serve конфигурация и Tandem Id/Image/StartedAt/RestartCount/Mounts
точно равны первоначальному pre-web baseline; server.env bytehash неизменён.

Итоговые источники evidence вне Git:
- `/private/tmp/accord-web-084/live-smoke-private.json`:29 PASS;
- `/private/tmp/accord-web-084/post-finalize-private.json`:PASS;
- `/private/tmp/accord-web-084/deploy/apply-result-private.json` и
  `finalize-result-private.json`:PASS;
- `/private/tmp/accord-web-084/make-check-final.log`:полныйPASS;
- QA095/097 immutable manifests и результаты, final source scan373files/5ZIP0.

Финальная Git фиксация меняет только документы относительно production d9aa;
Go/native/files/infra исходники уже frozen. Primary codex/pilot получает ff-only;
локальный установленный mac client пересобирается из принятых исходников.
Публикуются только проверенные рабочие ветки, не merge main. Инструкция для
браузера `docs/web-guide.md`; готовые native clients передаются отдельно безтокенов.
Физический второй ПК NOT-RUN, обе existing credentials проверены с одного Mac.
Завершён requested web scope; общие ограничения V1/D4/E/SLA/offsite выше сохраняются.

После snapshot restart повторён authenticated native download того же служебного PNG: HTTP200 и исходный SHA256 PASS (`post-restart-file.json`).
