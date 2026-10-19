# WORK-078 — Public HTTPS infrastructure

Дата:19.10.2026. Роль DevOps, отдельный исполнитель; профиль и общий процесс
прочитаны read-only. Named Claude role model в этой среде недоступна,
использована унаследованная доступная модель без override.
Статус: ГОТОВО К НЕЗАВИСИМОМУ REVIEW; НЕ РАЗВЁРНУТО исполнителем.

Прямой запрос пользователя: обычный HTTPS Accord с индивидуальными токенами,
без Tailscale на клиентах. Разрешённая область записи только
`/private/tmp/accord-public-078/author/`. Repo read-only; зафиксированный
текущий main HEAD8b6eff14f2de1290e701950fcd3bb7db14a79539, действующий
серверный binary1e9025c/schema2 не меняется данной поставкой.

Реализовано: IP-bound nginx bootstrapHTTP и TLS→fixedloopback proxy;
strict publicHost/absoluteURIhost guards,64KiBbody, boundedperIPlimits,
Origin/Authorization pass-through, spoofheadersdrop, bufferedIO/retryoff,
metricsprivate, publicaccess/errorlogoff. Certbot5.8 path/shortlivedlineage,
hourlytimer+expiryguard+nginxcheckedreload; hourlyExecStartPost повторяет
reload после успешного/no-op renew, устраняя потерю reload после прошлого сбоя. READMEprocedure/rollback/QA.
Все code/artifactfiles<300строк. Никаких токенов, ключей, действительных
частных адресов или данных в файлы не помещено. Домен/IP не выдуман:
renderer только получает подтверждённый публичный IP от координатора.

PASS локально: Pythoncompile; sh -n hook;10invalidIPclasses rejection,
canonicalIPaccept; bootstrapбезproxy/TLS, fixedloopback64KiB/phases guards.
Условия неприменимости: macOSбезnginx/systemd, поэтому реальныйnginx -t,
systemd verify/sandbox, ACMEissuance/renewal, HTTPS/MCP/auth/Origin/rate и
сохранностьTandem/слушающихадресов НЕ ЗАПУСКАЛИСЬ исполнителем.
Reviewer/координатор выполняет эти проверки на VPS, не выдавая шаблон заPASS.

Наблюдения: в исходникеMaxBodyBytes=64<<10,1MiBотносится кclientresponse;
подтверждено координатору. Нельзя proxy_set_headerOrigin$http_origin:
может удалить emptyOrigin; фактическийpass-through требует negativeprocesscheck.
ProtectSystemstrict renewunit оставляет /var/log/nginx writable дляnginx -t.
BindingтольконазначенныйпубличныйIPv4 не конкурирует сTailscaleадресом.
Внешниеportsоткрываютсякоординатором послеreview; executorSSH/liveнеиспользовал.

Передача: независимый reviewer проверяет manifest и код/безопасность;
координатор развёртывает проверенный набор и фиксирует реальные QA/
состояние служб/фактическийIPorigin в privateconnection.json.
ПолнаяV1/targetload/physical2PC этим изменением не закрываются.
