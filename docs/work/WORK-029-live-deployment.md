# WORK-029 — фактическая установка Accord

- Роль: DevOps; отдельный исполнитель live-поставки.
- Статус: завершён ограниченный live шаг; HTTP готов, remote MCP заблокирован.
- База: immutable release 99e722c44f83b1808120bba1fabb04cf2f68a5b5.
- Вход: review-003 APPROVE, security-003-recheck APPROVE,
  acceptance-003 PASS, acceptance-003-review APPROVE.
- Область записи: private `.local/`, этот WORK и `docs/runtime-server-003.md`.
- Общая TASK и production source не изменяются; публикация Git не выполняется.
- Именованная модель Claude недоступна; модель среды наследуется.
- Разрешены установка отдельного Accord, HTTPS8443, отдельный токен Mac,
  systemd/backup/runtime проверки; Tandem/443/его данные сохраняются.
- Внешние ограничения: второй физический ПК позже, Claude runtime не проверен.

## Результат

- Native immutable поставка99e722c реально установлена и опубликована
  по частному HTTPS8443. Старый Tandem/443/контейнер не изменён.
- Linux systemd/permissions/hardening/loopback/TLS/401/auth metrics/Origin
  policy/ephemeral revoke/реальный online backup и timer: PASS.
- Изолированный native Linux restore/reset с реальным systemd boot guard:
  PASS; metadata2tasks/2sessions/5idem/cursor6 сохранена, generation1→2,
  old bearer/SID401. Только fixture paths/unit names переадресованы.
- Mac working token отдельно, private0600; remote plaintext удалён.
- Для actual Codex smoke координатору выдан отдельный synthetic token;
  отзыв будет выполнен по завершении его прогона.
- Отчёт: `docs/runtime-server-003.md`, private evidence в `.local/evidence/runtime-server-003/`.
- Source/общая TASK не менялись, push/commit не выполнялись.
- Ограничения: physical secondPC/Claude/reboot/powerloss/live previous-release
  rollback/offsite не проверены или не настроены; полный V1 не объявляется PASS.
- Итог: native HTTP/эксплуатационные проверки PASS; **remote MCP FAIL**.
  Координатор при настоящем Codex smoke обнаружил default SDK DNS rebinding
  отказ по внешнему Host Tailscale Serve. Нужен отдельный безопасный fix
  exact-host allowlist/outer guard → review → retest → повторная поставка.
- Текущий HTTP service оставлен работать. Synthetic Codex token оставлен
  только для retest; координатор отвечает за последующий отзыв.
- Полный native MCP release этой WORK не принят.
