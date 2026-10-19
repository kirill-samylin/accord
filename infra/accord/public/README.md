# Публичный HTTPS Accord — артефакты 078

Объём: новый nginx на публичном IPv4:80/443 → существующий
127.0.0.1:7892; отдельные существующие bearer токены. Код/БД Accord,
Tandem и Tailscale-маршруты не изменяются. Клиентам Tailscale не требуется.
Результаты развёртывания и независимого review — docs/runtime-public-https-081.md
и docs/review-079-public-https.md в корне репозитория.
Именованные Claude-модели ролей в этой среде недоступны; профиль DevOps прочитан.

## Условия и файлы

Публичный IPv4 должен принадлежать локальному интерфейсу VPS. Порты 80/443
этого адреса свободны и доступны из интернета. Отдельный listen на адресе,
не 0.0.0.0/[::], сохраняет существующие Tailscale443/8443.
Для NAT-сервера данный renderer без отдельного решения не используется.
Не выключать Tailscale/Tandem и не изменять их маршруты.

- `render_public.py`: stdout nginx include внутри http{}; фазы bootstrap/tls.
  CLI принимает только канонический публичный unicast IPv4, без путей/портов.
- `public-tls-reload.sh`: проверка nginx -t, затем только reload nginx.
- `accord-public-renew.service` и `.timer`: hourly certbot renew с deploy hook,
  повторяющийся checked reload даже при no-op renewal, dependency nginx
  и точечный доступ к /run/nginx.pid для nginx -t; проверка остатка
  сертификата минимум24ч и видимый failed-status.
- `example-*.conf`: обезличенные образцы; они не являются конфигурацией VPS.

Nginx/Certbot ставятся до выдачи сертификата. Certbot 5.8.0 в отдельном
root-owned venv `/opt/accord/acme`; не обновлять системный Python.
Версия выбрана после проверки доступности координатором. Не использовать
дистрибутивный старый Certbot, не поддерживающий IP+webroot.

## Порядок установки под root, после review

1. Сохранить исходные nginx/firewall/units и список слушающих адресов.
   Исключить запуск нового пакета с default wildcard80 listener: на время
   установки маскировать только новый nginx, убрать только его штатный
   default-site, затем установить собственный проверенный include.
   Существующий nginx с неизвестными сайтами требует отдельной сверки,
   не автоматического удаления чужих конфигураций.
2. Создать root-owned `/var/www/accord-acme/.well-known/acme-challenge`
   с правами0755, без чужих файлов/симлинков. Это только публичные ACME
   challenge-файлы. Root-private ACME state/key directories не публикуются.
3. Отрендерить bootstrap через `--ip <PUBLIC_IPV4> --phase bootstrap`,
   установить `/etc/nginx/conf.d/accord-public.conf` root0644 и проверить
   `nginx -t`. Запустить nginx. Проверить HTTPchallenge снаружи; любые
   другие HTTPпути дают404, чужой Host разрывает соединение.
4. Сначала выполнить staging issuance с **отдельными** config/work/log dirs
   в root-private staging каталоге, чтобы untrusted staging chain не попал
   в production `/etc/letsencrypt`. Шаблон production команды:

```sh
/opt/accord/acme/bin/certbot certonly --non-interactive --agree-tos \
  --register-unsafely-without-email --required-profile shortlived \
  --webroot --webroot-path /var/www/accord-acme \
  --ip-address <PUBLIC_IPV4> --cert-name accord-public-ip
```

   `--register-unsafely-without-email` означает отсутствие писем CA;
   основная автоматизация/состояние сервиса обязательны. Если есть
   согласованный адрес оператора, заменить флаг на `--email <ADDRESS>`.
   Публичный IP и выданный сертификат публичны в CT; токены туда не входят.
   Проверить SAN содержит ровно IP, issuer trusted production, приватный
   ключ root0600, parents не доступны посторонним.
5. Отрендерить фазуtls и проверить временный конфиг с существующими
   production-сертификатами до reload. Атомарно заменить только собственный
   include; `nginx -t` и reload. Не менять `ACCORD_EXTERNAL_HOST`: nginx
   сам проверяет входную authority и отправляет фиксированный loopbackHost.
6. Установить hook root0755 и units root0644. Выполнить
   `systemd-analyze verify` для обоих units, daemon-reload, enable --now timer.
   Запустить renewservice вручную; проверить именно serviceExit0 под sandbox
   (включая nginx -t доступ к /var/log/nginx), не только certbot вне unit.
   Для проверки deploy hook при staging dry-run использовать
   `renew --dry-run --run-deploy-hooks` с явным hook и тем же production
   cert-name; staging сертификат не становится live. Проверить таймер.
7. Финальные внешние TLS/MCP проверки ниже. Обновить private connection.json
   на `https://<PUBLIC_IPV4>` без токена; на каждом клиенте меняется только
   serverorigin, tokenfile сохраняется прежним. Передать обновлённый файл
   отдельно от публичной Git-инструкции.

## Защита и проверки

TLS1.2/1.3; ACME HTTP80 не делает redirect и не проксирует bearer.
На HTTPS только /api/v1*, /mcp, /readyz, /healthz; /metrics остаётся приватным.
Body64KiB соответствует действующему MaxBodyBytes, а не clientresponse1MiB.
Public nginx limiter:20r/s burst40 и32concurrent per IP; fixed-size1MiB zones.
Backend100r/s burst200/64concurrent сохраняется, не обещает DDoS-защиту.
Сетевой клиентский IP не берётся из X-Forwarded-For. Входные Forwarded,
X-Forwarded-{For,Host,Proto,Port}, X-Real-IP в upstream удаляются.
Authorization и Origin проходят обычным pass-through, в том числе пустой
Origin: **не добавлять** `proxy_set_header Origin $http_origin`, иначе nginx
может удалить пустой заголовок и ослабить действующий backend Origin guard.

Request/response buffering и cache отключены; unsafe mutation не повторяется
nginx автоматически. StreamableHTTP/json-response поддержан без SSEбуфера.
Длинный proxyreadtimeout не отменяет собственные timeout/lease guard Accord.
Public accesslog отключён: атакующий поток не заполняет диск;
URL/identity/headers/body не журналируются.
Nginx errorlog для этих public servers отключён, потому что его штатные
строки могут включать requestURI: диагностика через private backendmetrics,
`nginx -t`, systemd status и независимые health/TLSchecks. Приватные ключи,
токены и серверная БД не попадают в артефакты/Git. Токен даёт прежние
полномочия пилота, отдельный token не добавляет scopes/RBAC.

Обязательные фактические проверки координатора/тестировщика:

- real nginx -t обеих фаз; listener только publicIPv4:80/443;
- externalTLS без insecure, SANIP, полная trustedchain, дата/expiry;
- readyz200; без bearer /mcp401; неверный bearer401; рабочие отдельные
  Mac иpc-2 bearer доступны каждый со своей machineidentity;
- Hostчужой, IPсчужимпортом, loopbackHost → отказ на nginx до backend;
- Originчужой/null/**пустой** →403 и не открывает сессию;
- spoofForwarded не меняет origin/host/authполитику;
- body>64KiB →413 до backend, HTTPAPI404, /metrics404 публично;
- реальный native stdioMCP initialize/toolslist12 через публичный origin;
- состояние Tandem/privateHTTPS до/после не изменилось;
- renewservice sandboxExit0, dryrun+hook, timerавтостарт и expiryguard;
- targetedlimited stand вместо нагрузочного удара liveобщегоVPS.

Это не доказательство физического подключения второго ПК: его оператор
подтверждает локальный MCP после передачи connection.json+своего токена.

## Откат

Удалить/отключить только `accord-public.conf`, проверить nginx -t и reload
либо остановить только новый nginx, если он исключительно для Accord.
Остановить/disable только `accord-public-renew.timer`; ключи оставить закрытыми.
Вернуть прежний clientorigin при необходимости. Не менять binary/БД Accord,
не отзывать рабочие токены автоматически, не выключать Tandem/Tailscale.
Откат publicproxy не является restore и не требует записи SQLite.

## Первичные источники, проверены 19.10.2026

- [Let's Encrypt: Certbot IP certificates](https://letsencrypt.org/2026/03/11/shorter-certs-certbot):
  IP через webroot поддержан Certbot≥5.4, профиль shortlived; production
  выдача отличается от staging, webserver reload настраивается отдельно.
- [Certbot user guide](https://eff-certbot.readthedocs.io/en/stable/using.html):
  renew/dry-run/deploy-hooks и выбор lineage.
- [Nginx proxy](https://nginx.org/en/docs/http/ngx_http_proxy_module.html):
  buffering, HTTP1.1, управление headers и отказ от upstream retries.
- [Nginx rate limit](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html)
  и [TLS](https://nginx.org/en/docs/http/ngx_http_ssl_module.html).

## Файлы основного nginx

`nginx.conf` подключает только собственный Accord include и не открывает
wildcard listeners. Применять только к новому выделенному nginx после
сохранения прежнего файла; чужие сайты не перезаписывать. `nginx-resource.conf`
ограничивает новый nginx256соединениями/однимworker и systemd CPU25%/Memory128M.
HTTP handshake ограничен client_header_timeout10s; ssl_handshake_timeout
относится к stream и в HTTP-конфиг не добавляется. Nginx429 получает
Retry-After1; существующий backendRetry-After сохраняется без дублирования.
