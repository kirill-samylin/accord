# WORK-094 — интеграция и выпуск веба Accord

- Исполнитель: /root, координатор/интегратор; модель наследуется.
- Источник: прямое поручение19.10.2026 о рабочем вебе для людей и агентов.
- База: f063380. Работа в отдельном codex/web-service; Tandem OFF.
- Статус: исходники реализованы; независимые проверки и QA/выпуск в работе.
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
отказывает до файлового IO. На Linux/macOS гарантии ownership/nofollow сохранены.
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
