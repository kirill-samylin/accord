# SEC-086: условия безопасности веб-Accord до реализации

19.10.2026. TASK-004; независимый security-профиль, не автор кода.
Исходная проверенная база: `f06338089bec3b8c7f17f541c463131f5c98f96c`.
Вердикт проектирования: **допустимо реализовывать с условиями ниже**.
Это не security approve будущего кода или развёртывания. Реализация и runtime
проверки пока NOT-RUN. Пользователь разрешил простой рабочий веб и выпуск;
OAuth отложен. Отдельных продуктовых блокеров не выявлено.

## Активы, входы, доверие

Активы: bearer, browser-session, задачи/аренды/generation, ответы/история,
содержимое и provenance дизайна, board.db/schema2, новая web.db/schema1,
backups и доступность существующего VPS/Tandem. Недоверенные входы: анонимный
HTTP, Origin/Host/forwarded headers, cookie/form/query, multipart/file metadata,
сами файлы и текст агентов/людей. TLS/nginx и OS identity сервиса доверены
в своих явно настроенных границах. Same UID/root не изолирован от данных.

Действующий token сохраняет текущие pilot-права owner/machine. Наличие
human browser-session не является новой OAuth identity, отдельной ролью
администратора или доказательством человека. Содержимое дизайна/ответа не
является разрешением выполнить код, скачать URL или управлять агентом.

## Условия, обязательные для approve реализации

### SEC-W01: общий HTTP-периметр — P1 при обходе

Host validation, глобальные admission/concurrency bounds, panic recovery,
no-store/nosniff оборачивают web, native, login и файлы ДО dispatch. Нельзя
разместить web handler перед существующей защитой NewHandlerWithOptions.
Native `/api`, `/mcp`, `/metrics` сохраняют отказ403 для любого присутствующего
Origin, включая пустой/null/malformed, и действующий bearer-контракт. Новые
agent file REST endpoints также native: bearer обязателен, browser cookie
не служит заменой. Health/readiness не раскрывают данные. Новых MCP tools нет.

WebOrigin — один явно сконфигурированный canonical HTTPS origin, без userinfo,
path/query/fragment; IP допустим для существующего публичного адреса. Неверный
конфиг закрывает запуск. Не выводить WebOrigin из Origin/Host/Forwarded.
Существующий ExternalHost принимает DNS:port, IP в нём отклоняется: не обходить
эту защиту. При сохранённом nginx upstream Host=loopback web Origin сверяется
с WebOrigin отдельно; trusted origin для web CrossOriginProtection задаётся
явно. Нельзя добавлять browser origin в native exception или CORS wildcard.

### SEC-W02: browser identity и отзыв — P1 при обходе

Вход принимает bearer один раз в body: текст либо один локальный tokenfile
не больше4096 байт. Ровно один непустой источник; общий body/multipart overhead
ограничен, токен не возвращается в HTML даже при ошибке. После проверки
выдаётся новый криптослучайный session secret минимум256 бит; сервер хранит
его SHA256 и token ID, expiry и необходимые native session IDs, но не bearer.
Cookie предпочтительно `__Host-accord-session; Path=/; Secure; HttpOnly;
SameSite=Strict`, без Domain. Новая login session не переиспользует preauth ID.
Предлагаемый технический срок:8ч absolute, ограниченный expiry токена.

На КАЖДОЕ чтение/мутацию/metadata/preview/download/replay вызывается новое
Store.AuthenticateID: по ID заново получить owner/machine/expiry при revoked=0.
Нельзя доверять сохранённым owner/machine в cookie/web.db или ListTokens cache.
В f063380 такого public метода ещё нет. Principal создаётся только сервером;
ID сам по себе не credential и метод не получает публичного auth endpoint.
Мутации дополнительно сохраняют authenticatePrincipal ВНУТРИ существующей
board-транзакции до idempotent replay. Отзыв, завершившийся до linearization
мутации, запрещает её. Уже начатый download не обязан отменяться задним числом.

Logout — защищённый POST, удаляет текущую browser-session/прекращает heartbeat,
чистит cookie; не отзывает общий token и не снимает чужую/live lease. Запрещены
bearer в URL/localStorage/sessionStorage/HTML/JS/logs/errors; access logs не
содержат request body, Authorization, Cookie, tokenfile или ответ целиком.
Storage errors дают недоступность, а не маскируются как неверный bearer.

### SEC-W03: login/CSRF/XSS — P1 для чужой мутации/скрипта

GET не меняет состояние и не продлевает board lease. Все browser mutations,
включая login/logout/heartbeat/upload, требуют CSRF token, привязанный к
preauth или auth session, и exact Origin=WebOrigin. Missing/null/duplicate/
malformed/чужой Origin отклоняются; обычная browser GET navigation остаётся
доступна. Preauth state ограничен по сроку/числу и не содержит bearer.
SameSite и стандартный CrossOriginProtection дополняют, а не заменяют явный
CSRF контроль. Для multipart проверка cookie/origin/CSRF выполняется ДО
записи file bytes; CSRF в header либо в отдельном первом bounded поле,
иначе ранний отказ, не полный ParseMultipartForm на50MiB.

Login: максимум5 неудачных попыток/мин/IP,429+Retry-After. Клиентский IP
принимается лишь из отдельного заголовка X-Accord-Client-IP, который nginx
ВСЕГДА перезаписывает своим remote_addr; backend только literal loopback,
trusted proxy link явно настроен и peer проверен. Не доверять XFF/Forwarded
или произвольному заголовку локального запроса. Если этот proxy-контракт
нельзя доказать — безопасный один глобальный bucket5/мин вместо IP buckets,
с честным ограничением общей доступности входа. Bounded bucket count/TTL,
параллельные запросы резервируют попытки атомарно до auth, успешные можно
не учитывать; проверка-before/count-after без резерва обходит предел.
Общий существующий admission остаётся перед дорогой обработкой.

html/template autoescape для всех user/agent текстов и имён. Не приводить
недоверенное к template.HTML/JS/URL; без raw Markdown HTML. CSP для UI минимум
`default-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'none';
form-action 'self'`; script-src без unsafe-inline/eval, небольшие локальные JS
файлы; Referrer-Policy same-origin для browser HTML (no-referrer обнуляет Origin у native form POST); native endpoints сохраняют no-referrer. Формы/ошибки не показывают токен повторно.

### SEC-W04: арендная модель и история — P1 при обходе

Browser actor связан с проверенным token ID и server-generated native human
session. Session ID/role/principal не принимаются из формы как полномочие.
Claim использует существующий атомарный адресный Next; answer/release/
heartbeat идут через существующие Store methods с actor, request_id и
проверенной task generation. Раздельные браузеры не должны разделять одну
server session, иначе чужая попытка станет своей. Stale/expired request не
перезапускается автоматически с новой generation/новым claim. UI показывает
конфликт, сохраняет несекретный draft в текущей странице. Чтение не heartbeat.

Нынешний Attempt не сохраняет Evidence.Answer; следующий answer перезапишет
текущий Evidence. Исторический `human_answered` event обязан включать answer,
task/epic, token ID/owner/machine, server session/generation и server time,
сохранённые В ТОЙ ЖЕ board-транзакции, что успешный Transition и replay.
Нельзя dual-write answer между board.db и web.db. Изменение только web-пути
недостаточно: native human completion тоже должно сохранять историю. Событие
immutable, exact replay не создаёт дубль. Старый binary не переписывает
существующий raw events JSON; это надо проверить rollback test, не предполагать.

### SEC-W05: файлы/квоты/ресурсы — P1 при чтении/исполнении/DoS

50MiB включительно на file; ограниченный overhead request, количество полей,
длина полей/filename и один file part. Streaming MultipartReader + ограничение
body; никакого io.ReadAll50MiB/неограниченного ParseMultipartForm. Квота1GiB
считает accepted+pending+staging, включая параллельные запросы. Перед приёмом
атомарно зарезервировать объём, при неизвестной длине — полный50MiB; запас
диска минимум512MiB с учётом всех зарезервированных записей. Ограничить upload
concurrency отдельно (предлагается2), timeout и время обработки. Не отключать
native30s/64KiB глобально; bounded upload route deadline/nginx cap согласовать.
Квота включает производные thumbnail, а cleanup освобождает резерв однократно.

Data/blob/staging directories0700, DB/WAL/SHM/blob0600, umask0077 до IO; unsafe
existing path отклоняется. Имена хранения server random IDs, не filename,
relative URL или путь клиента. Create O_EXCL + no-follow, parent containment,
regular-file checks на чтении/записи; проверка prefix строкой недостаточна.
Допустим descriptor-rooted os.Root плюс запрет symlink, поскольку os.Root сам
может следовать внутренним symlink. Даже authenticated URL не открывает
произвольный path; не использовать directory FileServer для blobroot.

MIME/extension/declared size недоверенные. ZIP только opaque, не распаковывать
и не запускать. HTML/SVG/PDF/ZIP только attachment, безопасный сформированный
Content-Disposition без CRLF, nosniff, application/octet-stream и sandbox CSP;
никаких iframe/object/static serving. Не исполнять design HTML ради preview.
PNG/JPEG preview: проверить actual format + DecodeConfig + bounds до полного
Decode. Предлагаемый bound: width/height<=8192 и area<=16M pixels, preview
concurrency1; полноценное декодирование выявляет corruption. Thumbnail лучше
перекодировать в проверенный raster. Ошибка preview не должна приводить к
inline выдаче активного файла. WebP в принятом ADR-W4 доступен только как attachment/octet-stream без preview;
новая зависимость для decoder не добавляется. Если позже включается preview,
ему потребуется настоящий поддерживаемый decoder с теми же bounds.

У каждого upload стабильный request ID, scope=principal+operation и canonical
metadata+content SHA256. Повтор с другим содержимым — conflict; одинаковый не
создаёт вторую версию/объём. Immutable версии, task/epic links проверяются
сервером. Durable pending -> write bounded temp -> fsync -> atomic publish ->
metadata visible; нет видимой successful metadata без готового файла. Crash
между любыми шагами даёт recovery cleanup/reconcile; никогда не публиковать
восстановленный orphan как успех по догадке. Download/metadata/preview требуют
текущей авторизации, URI без credentials. MCP resource возвращает metadata/
authorized URL, без чтения произвольного файла и без нового tool.

### SEC-W06: выпуск, backup и restore — P1 при открытии старого доступа

Отдельная web.db/schema1 избегает конфликта D4/E/schema4; board остаётсяschema2.
Backups новых данных требуют согласованного snapshot board+web+immutable blobs
после остановки ВСЕХ writers либо доказанного equivalent barrier. Просто три
живых независимых copy не гарантируют согласованности. У backup private modes,
checksums, bounded retention; restore drill в отдельном каталоге.

После restore до сети: удалить browser/preauth sessions и выполнить существующую
recovery policy отзыва всех восстановленных tokens/native sessions/leases с
сохранением монотонной generation. Это исключает resurrection токена, отозванного
после backup. Обычный restart, напротив, сохраняет действительные browser-session
до expiry, но вновь проверяет token. Rollback binary сохраняет web files/DB
и новые raw answer events; old native binary их не использует. Readiness/health,
совместимость MCP12 и прежние маршруты Tandem проверяются после переключения.

## Минимальные доказательства для независимого security review

1. Auth: cookie flags/no Domain; rotation/fixation; logout/restart/expiry/revoke
   на page, replay, metadata, preview и download; подставленный tokenID/session
   не авторизует; revoke внутри board transaction также блокирует replay.
2. CSRF: login/logout/heartbeat/upload и обычная форма с отсутствующим,
   неправильным/чужим token, null/empty/duplicate Origin не меняют состояние;
   корректный HTTPS origin через настоящий proxy работает; плохой Host отвергнут
   до auth/файлов; любой Origin на native по-прежнему403, MCP12 regression.
3. Login limiter:6-я ошибка429, параллельный burst не обходит5, spoof XFF/
   X-Accord-Client-IP не создаёт unlimited keys; recovery после окна, bounded map.
4. XSS: script/HTML в title/answer/filename — текст; upload HTML/SVG/PDF только
   attachment; no bearer в response HTML, URL, browser storage и логах.
5. Файлы: exact50MiB accepted/+1 rejected, общий лимит с двумя pending,
   недостаток диска, oversized multipart поля/число частей, cancelled upload,
   traversal/absolute/CRLF/dangling+regular symlink; за пределами root нет IO.
6. Preview: MIME spoof, malformed/truncated image, огромные dimensions/pixels,
   WebP actual decode; предел памяти/concurrency. ZIP не распаковывается.
7. Идемпотентность/recovery: replay upload/answer, payload mismatch, два claims,
   stale generation, два цикла ответа с сохранением обоих; fault между publish/
   metadata и restart не создаёт дубль/висячий успех/неограниченный staging.
8. Backup restore в изоляции: content SHA/версии/links/ответы сохранены,
   browser sessions и старые revoked-after-backup tokens больше не работают;
   rollback old binary не теряет неизвестные поля raw events.

Не требуются как новые блокеры: OAuth provider, покупка virus scanner,
распаковщик, изоляция от root или полный enterprise RBAC. Download-only файл
остаётся недоверенным для получателя; это не обещание его безвредности.

## Проверенные источники

Прочитаны AGENTS.md отдельного agent-board (Tandem OFF), TASK-004,
security-contract.md и actual C2 auth.go/http.go/http_security.go/http_host.go,
transitions.go/evidence.go/mutation.go/events.go/persistence.go/core types,
WEB-01..07/WEB-AC аналитика084. Профиль security и общий процесс Давинчи
прочитаны только как инструкции роли, исходники Давинчи не изменяются.
Официальная документация проверена19.10.2026:
[CrossOriginProtection](https://pkg.go.dev/net/http#CrossOriginProtection),
[autoescape html/template](https://pkg.go.dev/html/template),
[filesystem os.Root](https://pkg.go.dev/os#Root),
[image DecodeConfig](https://pkg.go.dev/image#DecodeConfig).

Фактически выполнено: read-only source/document review и сверка версии HEAD.
Тесты, adversarial probes, SSH/live/proxy/token checks: NOT-RUN; это условия
для последующего review, а не подтверждённые уязвимости ещё не написанного веба.
