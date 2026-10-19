# Независимое security/infra ревью 079

Дата: 19.10.2026. Роль — security, агент отдельный от автора. Использована та же доступная модель; закрепление моделей именованных ролей в этой среде недоступно. Прочитаны Accord AGENTS, deployment, security-contract, API, общий процесс и профиль security. Reviewer не менял source, Git, SSH и сервер, не читал значения токенов или ключей.

## Вердикт

**APPROVE реализации публичного HTTPS и проверенной автоматизации продления TLS на VPS.** Бинарник Accord `1e9025c` и схема 2 не меняются. Это не окончательный зачёт публичного MCP, авторизации или сохранности Tandem: эти фактические проверки принадлежат тестировщику 081 и координатору.

Проверенная поставка: авторский manifest SHA256 `45a114a2f67bef68e656dad44ee6eca30c9807108815c3185efa32e9af9397e5` (8 файлов, все хеши совпадают) и отдельно проверенные изменения оператора. Точные хеши всех файлов и использованных доказательств — `reviewed-files.json`.

Effective renderer: `operator/render_public.py`, SHA256 `25095ba31f2f1349be45cbc0eba34049d2bc6923e351d262bf334fe5edf55ffd`.

Effective renewal unit: `operator/accord-public-renew.service`, SHA256 `07f84ea01d04d741f3aabceb07b5c2cac701d913ad3898c148e9ee1f7d161b27`.

При публикации нужно использовать эти версии оператора, его nginx.conf/resource override и авторские hook/timer. Старый авторский пример TLS содержит неподдерживаемую директиву: примеры следует заново получить из окончательного renderer.

## Находки и исправления

1. **R079-01, P1, закрыто.** При `ProtectSystem=strict` команда `nginx -t` не могла открыть `/run/nginx.pid` с O_RDWR. Unit оператора разрешает запись ровно этого файла и требует активный nginx через `Requires=nginx.service`. Предварительно также разрешены необходимые каталоги логов и добавлен checked reload после каждого успешного/no-op renew: единичный сбой deploy hook больше не оставляет старый served certificate до следующей ротации. Обычный renewal и staging dry-run с deploy hooks в том же sandbox действительно завершились с кодом 0.
2. **R079-02, P2, закрыто статически.** Nginx выдавал собственный 429 без обязательного `Retry-After`. Map статуса и upstream-заголовка добавляет `1` только для 429, у которого ещё нет upstream Retry-After. Оба server block используют этот заголовок; неошибочные ответы его не получают, ответ backend не дублируется. Статические проверки обеих фаз прошли. Фактический ответ 429 проверяет 081.
3. **RUNTIME078-01, закрыто.** Реальный staged `nginx -t` поймал `ssl_handshake_timeout` в HTTP context: директива относится к stream. До reload ошибочная конфигурация не дошла. Reviewer пропустил это при статическом обзоре; initial static approve не был подтверждением работоспособности конфигурации. Оператор удалил директиву; HTTP SSL handshake остаётся ограничен `client_header_timeout 10s`. Окончательные staged/current `nginx -t` прошли.
4. Первоначальная проверка systemd unit остановилась из-за намеренной маски nginx на время установки. Resume проверил хеши установленных файлов, владельцев и неактивность nginx; снял маску только после `nginx -t`. Повторные verify/start прошли.

## Модель угроз и достаточность мер

Защищаются bearer credentials, данные и backups Accord, TLS key, существующий Tandem и ресурсы общего VPS. Недоверены authority, Origin, body, forwarded headers и неавторизованная интернет-нагрузка. Root-managed nginx завершает TLS, а существующий loopback backend проверяет bearer и native Origin policy. Индивидуальные токены дают отдельные identity и отзыв; scopes/RBAC этим изменением не добавляются.

Nginx слушает только подтверждённый публичный IPv4:80/443, без wildcard или IPv6 listener. Stock sites и другие include не подключены. Runtime mask установлен до apt, первая проверка конфигурации выполнена до запуска. Tailscale 443/8443 и Tandem не меняются этой поставкой; итоговое сравнение делает координатор.

Принимается точная IP authority и соответствующий absolute URI host; upstream Host всегда `127.0.0.1:7892`. Входные Forwarded/X-Forwarded/X-Real-IP удаляются и не дают полномочий. Authorization и даже пустой Origin проходят обычным pass-through; существующий backend отвергает любой присутствующий Origin на native endpoints. CORS, OAuth и публичного выпуска токенов нет.

HTTP80 обслуживает только строгий ACME challenge path, остальное возвращает 404 без redirect и proxy. Публичный metrics закрыт. Сохраняются body 64 KiB, конечные header/read/idle timeouts, per-IP 20 запросов/с с burst40 и 32 соединениями, а также backend 100 запросов/с, burst200 и 64 concurrent requests. Nginx ограничен одним worker, 256 соединениями, 25% CPU и 128 MiB. Это не обещание DDoS-защиты или целевого SLA.

Cache, buffering и автоматические upstream retries отключены. Public access logs и request-rich error logs отключены также на верхнем уровне: URL, заголовки и тела запросов не записываются в nginx request logs.

Используется production IP certificate Let's Encrypt на 160 часов; Certbot 5.8.0 установлен в отдельный root-owned venv. ACME state приватен, ключ root0600. Hourly persistent timer, проверка остатка минимум 24 часа и checked reload автоматизированы. Клиентам не нужны Tailscale, отдельный домен, новый CA или отключение проверки TLS. Сам IP certificate и запись CT публичны по природе; токены, БД и SSH-ключи туда не входят.

## Проверки и доказательства

Независимо выполнены статические проверки: author 8/8 SHA; 11 некорректных IP (loopback/private/CGNAT/multicast/порт/URL/newline/альтернативная запись/IPv6); canonical IP, точный bind, отсутствие proxy в bootstrap, фиксированный upstream, отсутствие переписывания Origin и HTTP redirect, отключение request logs. После изменений оператора повторены проверки обеих фаз на Retry-After, header timeout и отсутствие неподдерживаемой SSL-директивы.

Первый проверяющий oracle ожидал уже удалённый log_format и завершился IndexError. Проверка исправлена под актуальный `access_log off`, production source не менялся; повтор прошёл.

Прочитаны фактические доказательства координатора:

- `bootstrap.json`: nginx-t, unit verify и старт — exit0.
- `certificates.json`: staging/production issuance, SAN IP, trusted chain и checkend — exit0; приватный ключ root0600.
- `activation.json`: окончательные staged/current nginx-t, reload, normal renew и dry-run с deploy hooks в том же unit sandbox — exit0. Production certificate остался прежним; временный override удалён. Timer enabled/active, следующее срабатывание назначено.

`activate.py` меняет во временном override только ExecStart; hardening остаётся тем же. Очистка выполняется в finally, timer включается после успешных проверок. SHA256 activation evidence: `bcbd31f78e4c6a6bf170e38e45bef2e9b2e60057535f3d0132f8babefbc1b9a4`.

**NOT-RUN reviewer:** публичные auth/Origin/spoof headers/body/429/MCP, тест физического второго ПК, Windows, сравнение Tandem/listeners/executable. Публичный acceptance выполняет 081, сохранность служб — координатор. Наличие шаблона или TLS certificate не подменяет эти проверки.

Неблокирующая рекомендация: fsync родительского каталога после последнего staged→active rename конфигурации. При обычной записи file/parent fsync уже есть.

## Документы 080

`public-https.md`, `second-pc.md`, `WORK-080.md` — **APPROVE** по зафиксированным хешам. Нет реальных IP, токенов или ключей. Старый порядок с Tailscale явно заменён. Указано, что клиент сам JSON не читает: адрес нужно перенести в `--server`. Есть POSIX 0700/0600 и Windows ACL с временным окружением без literal token/TOML secret; тест ограничен своим новым epic и его отменой. Физический второй ПК и Windows не объявляются проверенными.

При интеграции координатор обновляет старые точки входа и пакет, чтобы пользователь больше не начинал со старого шага Tailscale.

## Первичные источники

- https://letsencrypt.org/2026/03/11/shorter-certs-certbot — Certbot >=5.4, IP/webroot и deploy hook.
- https://letsencrypt.org/2026/01/15/6day-and-ip-general-availability — срок 160 часов.
- https://nginx.org/en/docs/switches.html — nginx-t открывает файлы конфигурации.
- https://github.com/nginx/nginx/blob/release-1.24.0/src/core/ngx_cycle.c#L296 и #L966 — PID O_RDWR в test mode.
- https://github.com/nginx/nginx/blob/release-1.24.0/src/http/ngx_http_request.c#L330 и #L632 — HTTP SSL handshake использует client_header_timeout.

## Границы результата

Кража bearer даёт прежние полномочия токена до expiry/revoke. Новых OAuth/RBAC, offsite backup, DB migration или исходников сервера в 078 нет. Full V1, D4, ежегодный объём, E5 и два физических ПК остаются отдельными этапами. В отчёт не перенесены значения credentials и частные параметры подключения.

Следующий шаг: принять 081 и сравнение служб координатора, обновить private connection.json/пакет/документацию и сообщить фактический результат без заявления о готовности полной V1.
