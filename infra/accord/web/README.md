# Эксплуатация веб-Accord

Включение опционально: `serve --web-origin HTTPS_ORIGIN --web-data PRIVATE_DIR`.
Параметры можно задать через `ACCORD_WEB_ORIGIN` и `ACCORD_WEB_DATA`. Для обычного
localhost-тестирования разрешён только явный literal-loopback HTTP origin;
в production — HTTPS и защищённые cookie. Без web-origin прежний native сервер
сохраняет поведение. Board schema2, web.db schema1; D/E migrations не включены.

За nginx используйте фиксированный loopback Host как раньше. Рендер публичного
конфига: `python3 infra/accord/public/render_public.py --ip PUBLIC_IPV4 --phase tls --web`.
Адрес берётся из защищённой конфигурации; реальные параметры не коммитятся.
У web upload отдельный предел52494336байт и120с, остальные native запросы64КиБ.
Origin и Authorization сохраняются буквально, включая пустой Origin. Новое поле
X-Accord-Client-IP nginx всегда перезаписывает remote_addr; только при этом
разрешён `--web-trusted-proxy` на loopback listener. Иначе безопасный общий предел
входа5попыток/минуту. Cookie не заменяет bearer на native API.

Сначала backup существующей БД и private-копия старого бинарного файла/конфига,
затем nginx syntax + systemd verify, остановка единственного writer и атомарное
переключение релиза. При ошибке вернуть старый бинарный файл/конфиг и запустить
сервис. Для rollback БД не восстанавливается: это потеряло бы новые ответы.
Новые web-data остаются private и не удаляются. Проверяются `/readyz`, вход,
native auth/Origin,12MCP tools, resources и фактический файл.

## Резервное копирование и восстановление

Установите reviewed `backup.py` как `/opt/accord/admin/web-backup.py`, root:root0644.
Каталог `/var/backups/accord-web` — root:root0700. Units из этого каталога
устанавливаются в `/etc/systemd/system`; после проверки запуска включается
`accord-web-backup.timer`. Старый hourly board backup остаётся отдельным.

Новый daily backup на короткое время останавливает Accord под общим maintenance
lock, копирует согласованные board.db + web.db + immutable blobs/staging,
проверяет SQLite integrity/schema, metadata references и SHA256 manifest, fsync
и атомарно публикует снимок. Затем служба запускается даже при ошибке копирования.
Хранятся7успешных daily копий. Запас диска после копирования≥512МиБ. Последняя
хорошая копия не удаляется до публикации новой. Эти копии находятся на том же VPS;
внешнее резервирование отдельно не настроено.

Для восстановления сначала остановить writer и сохранить текущий state. Проверить
snapshot через `validate_snapshot` и скопировать его в новый offline staging,
никогда поверх работающего экземпляра. Применить `prepare_restored_auth` к staging:
удаляются browser/preauth/actions. Восстановленная board.db проходит существующий
`maintenance reset` с отзывом старых токенов/сессий и проверкой восстановления;
выпуск новых токенов локальным оператором обязателен. Все state dirs0700,
files0600, owneraccord. Проверить loopback fixture: вопросы/история/версии/hashes,
затем атомарно переключить каталог и запускать сеть. Не публиковать сырые backup
или manifest с реальными данными в репозитории.

Объём1ГиБ дизайнов может потребовать примерно8ГиБ с семью self-contained snapshots
плюс board/backups/temp и системный резерв. При нехватке места upload/backup
отклоняется с ошибкой; работа с уже сохранёнными задачами должна продолжаться.

SQLite compatibility: offline source имеет закрытый/checkpointed WAL; непустой
WAL отклоняется вместо игнорирования committed frames. Источник читается как
immutable read-only, чтобы root backup не создавал sidecars пользователя accord.
Копия нормализуется в journal_mode=DELETE с проверкой header1/1. После закрытия
удаляются только собственные regular0600 sidecars нового target (WAL лишь пустой),
поэтому строгий manifest не зависит от сохранения stale SHM конкретной SQLite.
Живые source sidecars не удаляются. Проверки повторяются на настоящем Go fixture.
