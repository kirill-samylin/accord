# WORK-019 — эксплуатационная поставка Accord

- Роль: DevOps; исполнитель deployment003; TASK-003.
- Статус: на независимой проверке; live-изменения до review/security не выполняются.
- Границы: `infra/accord/**`, `docs/deployment.md`, эта WORK, private evidence.
- Профиль devops и общий процесс sibling-проекта прочитаны. Именованные
  Claude-модели в этой среде не закреплены; используется доступная модель автора.

## Реализация

Native Linux distribution из чистого reviewed Git tree с закреплённым
Go1.27.2; binary+assets+SOURCE+SHA256SUMS. Root-owned immutable releases и
atomic current; dedicated accord/nologin, private state/backups, systemd
hardening, resource bounds, readiness/startup, explicit binary rollback.
Private Tailscale8443 добавляется отдельным скриптом с before/after сравнением
всех чужих маршрутов; reset/Funnel/изменение443 не используются.

Штатный appbackup запускается hourlytimer, SHA256+fsync, retention168пар;
копия живого DBfile не используется. Restore проверяет checksum/integrity,
останавливает только Accord/его backup, сохраняет прежний DB+journals,
атомарно ставит snapshot, оставляет boot guard и остановленный сервис до
resetcredentials/leases. Выдача token отдельному owner/machine через0600файл,
без секрета в stdout/argv. Offsite и внешний alert transport не настроены.

## Текущее evidence

- READ-ONLY VPS PASS: x86_64, systemd255, Python3 sqlite3.45.1, Tailscale1.104.1
  имеет --https/--bg, свободны7892/8443; Serve содержит только443 и0Funnelentries.
- Никакие stop/start/install/Serve config mutations при этой проверке не вызывались.
- PASS: AST/shell syntax, `git diff --check`, helper checks для изоляции443/8443,
  чужого proxy target, integrity/checksum, отказа public0644/symlink, retention2из3
  и0600checksums. Appbackup в wrapper-тесте mocked: это не runtime-доказательство
  Go backup, SQLite concurrency, systemd или VPSdeployment.
- Build script fail-closed на незакоммиченном дереве; полноценная reproducible
  сборка возможна после финального commit. Все файлы меньше300строк.
- Независимые security/code review и Linuxruntime остаются следующими этапами.
- App `backup`, `token list`, `token revoke-all` и maintenance reset реализует
  отдельный исполнитель admin003; согласованы flags/output. Reset атомарно
  отзывает все credentials и инвалидирует restored сессии/аренды, offline.
- Физический второй ПК NOT-RUN по ответу пользователя «смогу позже».

## Ограничения/передача

Чужой Tandem не используется и не изменяется. Single shared VPS остаётся
общей точкой отказа; локальные snapshots не заменяют offsitebackup. CPU50%и
256MiB — эксплуатационный выбор для shared1GB, нагрузка ещё не измерена.
Автоматический app rollback не восстанавливает DBмолча; неготовность оставляет
процесс остановленным с явной ошибкой. Live deploy разрешает координатор
только после независимого approve конкретных файлов.
