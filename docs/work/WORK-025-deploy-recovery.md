# WORK-025 — безопасное восстановление после ошибки обновления

- Роль: разработчик; исполнитель infrafix003, TASK-003.
- Статус: реализация завершена; передано на независимое повторное ревью.
- Вход: AGENTS, TASK-003, service/security contract, WORK-019, deployment,
  профиль developer и общий процесс sibling-проекта прочитаны.
- Границы записи: install.py, rollback.py, новый recovery.py, restore.py,
  accord.service, узкое уточнение deployment.md, эта WORK и private evidence.
  Расширения restore/unit/docs явно поручил координатор по security findings.
- Live VPS, Tandem, данные пользователей и Git не изменялись.

## Исправления

SEC003-01: install и rollback используют общий activation/recovery lifecycle.
Ошибка backup до переключения, исключение systemctl start либо отсутствие
readiness приводят к явной остановке неготового процесса. Прежний executable
автоматически возвращается и проверяется только при неизменной известной
schema1. Ошибка операции сохраняется даже при успешном возврате старого
сервиса; DB snapshot автоматически не восстанавливается.

Перед переключением DB читается stdlib SQLite в mode=ro с учётом WAL:
фиксируются PRAGMA user_version и SQLite schema fingerprint. Изменившаяся,
повреждённая, неизвестная схема и небезопасные private files закрывают
автоматический restart. Совместимость произвольных версий не предполагается;
изменения persisted data/source format должны повышать user_version.

При несовместимости current остаётся на новом executable, сервис остановлен.
При неуспешном возврате остаётся выбранный прежний executable, также stopped.
Первая неудачная установка не включает автозапуск и остаётся stopped.
Во всех этих случаях root создаёт private START_REQUIRES_INSPECTION, fsync
file и state directory; unit запрещает запуск до явной проверки оператором.
Guard автоматически не удаляется. RESTORE_REQUIRES_RESET остаётся отдельным
запретом, не изменяется и не подменяется inspection guard.

SEC003-02: restore fsync каталога сразу после создания/fsync boot guard и до
первой stop-команды или смены DB; ошибка directory fsync не допускает stop/DB
изменений. Это обеспечивает durable directory entry guard до восстановления.

REV003-01: одинаковые switch/healthy в install/rollback вынесены в recovery.py.
Helper включён в ADMIN_FILES, build-release уже копирует все infra *.py.

## Проверки и передача

PASS: AST четырёх Python-файлов; git diff --check;17 локальных unit сценариев
с mock subprocess и private synthetic SQLite. Покрыты backup failure, start
exception, readiness failure, schema migration/unknown schema/unversioned DDL,
первый запуск, успешное обновление, неудача возврата, оба guard, symlink DB,
WAL user_version, fsync ordering обоих guard и injected restore directory
fsync failure. Assertions проверяют current, число start/stop, неизменность
данных и сохранение мигрированных данных без скрытого DB rollback.

Команда: `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s
.local/evidence/infrafix003 -p 'test_*.py' -v`.

Это synthetic unit evidence, не проверка systemd/VPS, reboot/power-loss или
реального обновления. Runtime/E2E выполняет отдельный tester после approve.
Все изменённые source-файлы меньше300 строк.

Private evidence: `.local/evidence/infrafix003/`.
Manifest6 SHA256: `2476dfd5b72589538cdecdf6f108de7efc287b8befe5df97a5bbc0b5c8929f5c`.
Test source SHA256:
`f6dd8b2f54204093db45a194adaa03df50cd9b97ce0c76d79f65d780e3aed8c3`,
`0036f3a7fb9d48f7d2d582081fec629b003490c3f4f689fbed7cee1fa63d8415`.
Log SHA256: `38b91d0a972d92cbfc707a8ad28b381fe6adcefee98107b265e92a58637a526f`.

Следующий исполнитель: независимые review003/securityrecheck003; переданы
конкретные source hashes и команда повторения unit evidence. Самостоятельный
approve не выставлялся; live этап до их решения не разрешён.
