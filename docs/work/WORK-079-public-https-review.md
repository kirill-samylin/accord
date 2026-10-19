# WORK-079: независимое security/infra ревью публичного HTTPS Accord

Статус: в работе. Роль: security; отдельный от автора агент, та же доступная модель; именованное закрепление моделей в этой среде недоступно.

## Объём и границы

Пользователь явно разрешил публичный HTTPS и индивидуальные токены без Tailscale на клиентах. Проверяется новый proxy/ACME слой при неизменных Accord executable live1e9025c/schema2 и существующем Tandem. Запись только review/; live, ключи, токены, source и авторские файлы не меняются.

Активы: credentials, board/SQLite, приватный Tandem, TLS key, доступность VPS. Недоверенные данные: HTTP authority/Origin/body/forwarded headers, неаутентифицированная интернет-нагрузка. TLS terminates в root-managed nginx; bearer и native Origin policy проверяет существующий loopback backend. Authorization не берётся из forwarded headers. Публичный адрес больше не является секретом/сетевым доступом: bearer — действующая граница авторизации.

## Предварительные замечания

1. Renewal systemd ProtectSystem=strict требует проверить write access nginx -t к логам и другим runtime путям; передано координатору до freeze.
2. Установка nginx не должна автоматически временно открывать wildcard80. Координатор подтвердил runtime mask ДО apt и первый start только после bound-IP конфигурации; итоговое доказательство ещё требуется.

## Проверки и источники

Прочитаны Accord AGENTS, deployment, security-contract, API; общий процесс и профиль security соседнего проекта readonly. Официальные первичные источники: https://letsencrypt.org/2026/03/11/shorter-certs-certbot (Certbot>=5.4 webroot/IP + reload hook), https://letsencrypt.org/2026/01/15/6day-and-ip-general-availability (160 часов), https://nginx.org/en/docs/switches.html (-t открывает referenced files).

Freeze/final verdict и runtime ещё ожидаются; live PASS не выдан.

## Журнал

- 2026-10-19T17:11:31.937984+00:00 — начат независимый обзор; замечания переданы координатору.

- 2026-10-19T17:14:53.750916+00:00 — frozen8/8 verified, independentstaticprobesPASS; proxy/TLSbootstrapAPPROVE, renewalR079-01передан. Reportreview-079.md.

- 2026-10-19T17:15:30.456306+00:00 — operatordeltaPIDRW+Requires/staticbootstrapreviewAPPROVE, R079-01closedSTATIC;runtimepending.

- 2026-10-19T17:17:31.425789+00:00 — contractcontrolmapping: R079-02 edge429RetryAfterпередан, не блокируетbootstrap/TLSissuance, требуетсякитоговомуcontractPASS.

- 2026-10-19T17:19:00.932401+00:00 — R079-02operatorheaderstaticPASS, certificatesissuanceevidencePASS; runtimeenable/renew/externalguardspending.

- 2026-10-19T17:19:48.794639+00:00 — activate.pyи080docsstaticAPPROVE;actualrenew/externalguardsevidenceожидаются.

- 2026-10-19T17:23:01.124546+00:00 — FINAL: implementation+renewalAPPROVE, actualactivationevidenceпроверено;079завершён,081/rootexternalacceptanceотдельно.
