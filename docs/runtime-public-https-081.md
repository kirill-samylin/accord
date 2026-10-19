# Публичный HTTPS Accord — фактическая проверка 19.10.2026

Пользователь поручил обычный HTTPS и отдельные токены без Tailscale на клиентах.
Live остаётся C2/schema2, binary1e9025cd6e3122f0adf6165e09845202736d004a.
Публичный origin передаётся в private connection.json; в Git не публикуется.

## Развёртывание и автоматизация

- Новый nginx слушает только публичный IPv4:80/443. Backend7892 остаётся
  loopback-only; внешний authority проверяется до fixed loopback proxyHost.
- HTTP80 обслуживает только ACME challenge; API по HTTP не проксируется.
- Публично доверенный Let's Encrypt IP-сертификат, TLS1.2/1.3, обычная проверка
  цепочки и IP. Никаких self-signed/insecure и стороннего wildcardDNS.
- Certbot5.8.0 в изолированном /opt/accord/acme; hourly enabled/active
  accord-public-renew.timer. Normal renew и dry-run/run-deploy-hooks прошли
  в том же production sandbox с exit0. Непрерывный checkedreload также
  выполняется при no-op renewal; remaining-life24h guard видим через systemd.
- Staging отделён от production; dry-run не заменил production cert,
  временный runtime override удалён. Certprivatekey root0600.
- Raw access/error requests не журналируются; auth/Origin/Host checks
  сохранены. Limits:64KiB body, perIP20r/s+burst40/32conns и существующие
  backend100r/s+64inflight. Nginx429 получает Retry-After, без дубликата upstream.

## Проверки

| Проверка | Результат |
| --- | --- |
| ACME staging/production, SANIP/chain/expiry/keymode | PASS |
| Реальные nginx-t staging/current, reload | PASS |
| Normal и dry-run renewal с hook в strict unit, timer | PASS |
| QA081 внешние TLS/auth/MCP/Host/Origin/size/rate/HTTP |21 PASS |
| QA081 предпосылка «TailscaleStopped» | FAIL: клиент оказался Running; не скрыт |
| QA082 focused при фактическом TailscaleStopped |6 PASS,0FAIL |
| Восстановление исходного Running и prefs после082 | PASS |
| Приватные маршруты/Tandem container и health | unchanged,200 |
| Бинарник/release Accord и loopback readiness | unchanged,200 |
| Физический второй ПК/Windows/Claude runtime | NOT-RUN |

QA081 действительно отправил пустой Origin (не удалил его):403.
Без токена/неверный токен:401; оба отдельных действующих токена:200.
Pc-2 stdio initialize/catalog12 проверен без board-session pc-2.
Mac synthetic E-5 создан/get/events и отменён/getverified; чужие задачи не менялись.
Hostwrong/loopback/absoluteURI:connectionclosed; body65537:413; metricspublic404;
HTTPAPI404; bounded80staticACMErequests gave53×404/27×429, Retry-After1.
QA082 при Stopped повторил defaultPKI/IPverify, ready200, pc2HTTPinitialize,
настоящий stdio initialize/catalog12 без доски. После проверки восстановлены
исходное состояние Tailscale и неизменные prefs; VPS Tailscale не выключался.

## Пойманные и устранённые ошибки

- Initial unit verify встретил deliberate nginx runtime mask после установки;
  после nginx-t маска снята, повторная проверка dependency/start PASS.
- HTTP nginx1.24 отверг stream-only ssl_handshake_timeout; до reload/open443
  параметр удалён. Handshake ограничен HTTP client_header_timeout10s.
- Review079 исправил точечный доступ nginx-t к PID в strict sandbox и
  Retry-After для собственных nginx429; операторская версия прошла runtime.
  Замороженный первоначальный author/exampleTLS не является готовой версией.

## Версии и границы

Backend SHA25691ac8396cda97ee0577be6074eb9523f8ab8ce8beb7579cc52ec9684116ff1a2.
Mac client SHA256c58286c9e08708052a6f5f375fb44ca115be2b784b5250f86de297897385cc1e.
Reviewed template/unit — infra/accord/public из окончательной operator-версии.
Code/security review079 и финальное review083 тестов/документов отдельны от автора.
Named Claude model binding недоступен; все агенты с унаследованной моделью.

Это закрывает публичный HTTPS ingress текущего bearer-сервиса. D4/performance,
полная V1, offsite backup, SLA и физический2PC не объявлены завершёнными.
Исходные токены не перевыпускались. Рабочий public-токен сохраняет прежние
полномочия: отдельная выдача не добавляет RBAC/scopes сверх liveC2.
Полные private evidence с hostname/IP остаются в игнорируемой .local/evidence,
в Git входят только этот отчёт и обезличенные инфраструктурные файлы.
