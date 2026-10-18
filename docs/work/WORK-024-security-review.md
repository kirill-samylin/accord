# WORK-024 — независимая безопасность TASK-003

- Роль: security; исполнитель `securityreview003`, отдельный от авторов.
- Статус: завершена возвратом; вердикт REQUEST CHANGES.
- Границы записи: эта WORK, `docs/security-003.md`, собственное private evidence.
- Основание: AGENTS, TASK-003, service/security contracts; общий процесс и
  security-профиль sibling прочитаны. Tandem отключён по решению пользователя.
- Модели Claude-профилей здесь не закреплены; используется та же доступная
  модель, что у авторов. Production-код reviewer не изменяет.

## Проверено

Статически: native Origin на GET/POST, fixed admission/metrics, shared client
origin/tokenfile/redirect policy; stdio session injection, heartbeat, quarantine,
EOF; локальные admin/token audit и atomic snapshot/recovery; systemd, root-owned
releases, выдача токенов, Tailscale8443/изоляция443, restore boot guard.

Независимый запуск на Mac: `go test -race -count=1 ./internal/transport
./internal/store ./internal/bridge ./internal/client ./cmd/agent-board` — PASS:
2.601s / 6.278s / 12.436s / 1.235s / 1.562s соответственно. Это tests с
synthetic локальными fixtures; live VPS и физический второй ПК не затронуты.

## Возврат

SEC003-01: installer/rollback обрабатывают только отрицательную readiness;
ошибка `systemctl start` проходит мимо возврата current. Ошибка backup после
stop оставляет исправную прежнюю службу остановленной. Координатор принял
возврат на отдельного автора infra fixes. Точные условия — security-003.

SEC003-02: restore синхронизирует guard file, но не его directory entry до
изменения состояния. Требуется fsync STATE перед stop/replace и проверка
отказа; иначе reboot fail-closed не доказан. Передан координатору отдельно.

Bridge/admin ещё финализируют свои файлы. Итоговый approve запрещён до
freeze авторов, независимой проверки infra fix и повторного exact manifest.
Промежуточный private manifest сохраняет состояние первичного чтения, а не
подменяет окончательную проверенную версию.

## Журнал

- 2026-10-18: started — выполнено чтение контрактов и новых поверхностей.
- 2026-10-18: returned — SEC003-01 передан координатору; ожидается исправление.
- 2026-10-18: completed — SEC003-02 также принят координатором; область fix
  расширена на restore.py. Этот отчёт сохраняет первичные findings; после
  окончательного freeze нужен новый независимый security recheck.
