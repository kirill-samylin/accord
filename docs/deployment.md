# Общий сервер Accord: установка и эксплуатация

Контракт TASK-003: один процесс и одна БД на Linux/VPS, отдельные токены
каждой машины. Существующий Tandem, его контейнер, данные и HTTPS443
сохраняются. Инструкции описывают процедуру; фактический результат развёртывания
фиксируется отдельно в TASK/WORK. Второй физический ПК пока недоступен.

## Размещение и доступ

| Ресурс | Назначение |
| --- | --- |
| `/opt/accord/releases/<full Git SHA>/agent-board` | Root-owned неизменяемый исполняемый файл и SHA256 |
| `/opt/accord/current` | Атомарно переключаемая ссылка на установленную версию |
| `/opt/accord/admin` | Проверенные root-owned эксплуатационные скрипты |
| `/var/lib/accord` | БД, журналы, private-файлы; пользователь accord, 0700 |
| `/var/backups/accord` | Согласованные SQLite snapshots и SHA256, каталог 0700, файлы 0600 |
| `127.0.0.1:7892` | HTTP только внутри VPS |
| Частный Tailscale HTTPS8443 | HTTPS → loopback7892, без Funnel |

Пользователь `accord` системный, без login shell. Он может менять только своё
состояние/копии, но не binary, symlink, helpers или systemd units. Веб-входа
через GitHub пока нет по выбору пользователя: доступ — по отдельным bearer
токенам машин плюс членство в частной сети. Это не OAuth и не административная
роль: владелец SSH/root управляет установкой и выпуском/отзывом токенов.

Service имеет umask0077, строгую файловую изоляцию, запрет привилегий и доступ
только к loopback. Лимиты 256MiB и50%CPU выбраны для совместного VPS1CPU/1GB;
это ограничение ресурсов, не доказательство целевой нагрузки исходного ТЗ.
При необходимости root меняет лимиты через `systemctl edit accord.service`,
фиксирует изменение и повторяет нагрузочные проверки.

## Сборка и установка

Поддержанный сервер: Ubuntu24.04 amd64, systemd255, Python3 stdlib, Tailscale
с уже включёнными HTTPS/MagicDNS. Отдельный sqlite3, Docker и Go на VPS
для Accord не требуются. Native Linux runtime подтверждается только после
фактической установки; cross-build сам по себе этого не доказывает.

После review, security и проверок закоммитьте конечное дерево. Скрипт
отказывается собирать незакоммиченные изменения и требует Go1.27.2:

```sh
sh infra/accord/build-release.sh
```

Получится `.local/releases/<SHA>/` с бинарником Linux/amd64, исходной ревизией,
скриптами и `SHA256SUMS`. Каталог приватный; credentials туда не включаются.
Перенесите его по SSH/SCP с закреплённым ключом сервера в root-owned каталог
на VPS. Не отключайте StrictHostKeyChecking; значения частных адресов/ключей
не должны попадать в публичную документацию. На сервере под root:

```sh
cd /root/accord-release-<SHA>
sha256sum --check SHA256SUMS
python3 infra/install.py \
  --binary ./agent-board \
  --sha256 <SHA256_БИНАРНИКА_ИЗ_ПРОВЕРЕННОГО_SHA256SUMS> \
  --revision <ПОЛНЫЙ_GIT_SHA> \
  --assets ./infra
systemctl status accord.service --no-pager
systemctl list-timers accord-backup.timer --no-pager
curl --fail --max-time 5 http://127.0.0.1:7892/readyz
```

Installer проверяет hash до изменения, приватные каталоги и системную учётную
запись; отличающиеся существующие права/владельцы отклоняются без исправления
чужого состояния. Версия с тем же SHA не перезаписывается другим binary.
Установка включает автозапуск Accord и hourly backup timer, делает первый
snapshot. До этого шага проверяются свободные порты7892/8443 и состояние
Tandem; его команды запуска/остановки installer не вызывает.

Отдельно после локальной readiness под root:

```sh
python3 /opt/accord/admin/publish.py
```

Это добавляет только HTTPS8443. Скрипт проверяет, что порт не принадлежит
другому сервису, Funnel выключен, а все прочие маршруты до/после совпадают.
Нельзя использовать `tailscale serve reset` или заменять всю конфигурацию.
С Mac проверяется `https://<ЧАСТНОЕ_DNS_ИМЯ_VPS>:8443/readyz` с обычной
проверкой TLS. Health без токена не доказывает авторизованный MCP-доступ:
отдельно нужны401 без токена и успешный synthetic сценарий своим токеном.

## Отдельные токены и отзыв

Один токен на машину/участника, по умолчанию30дней. Выпуск на VPS под root:

```sh
python3 /opt/accord/admin/token.py --owner kirill --machine mac-main --days 30
python3 /opt/accord/admin/token.py --owner colleague --machine second-pc --days 30
```

Wrapper вызывает штатный CLI под accord и выводит только путь нового
private-файла в `/var/lib/accord/issued/`. Секрет не выводится и не передаётся
в аргументах. Перенесите нужный файл по SSH/SCP в приватный0700каталог клиента,
проверьте0600, подключите его через поддерживаемую конфигурацию клиента.
Отдельную копию для второго ПК выдавайте при подключении; не передавайте ему
токен первого. После подтверждённой доставки удалите серверный исходный
secret-файл. Удаление файла не отзывает credential в БД.

Управление credential выполняется штатным CLI `token list --db ...`, `token
revoke --db ... --id ...` и `token revoke-all --db ... --reason ...` под accord.
Актуальный статус проверки команд — TASK-003. При утере/истечении токена root
выпускает новый и отзывает прежний.
Не редактируйте таблицы SQLite вручную, не храните bearer в unit/env-файлах
с правами0644, Git, сообщениях, логах или history.

## Обновление и откат executable

Повторите сборку/перенос/проверку SHA и installer для новой ревизии. Перед
переключением installer останавливает только Accord и делает snapshot через
его прежний executable. Затем переключает current и проверяет readiness.
Ошибка backup до переключения или ошибка запуска/readiness после него
завершает обновление с ненулевым кодом. Скрипт останавливает неготовый процесс
и автоматически возвращает прежний executable только при проверенно
неизменной известной схеме1; проверяет readiness прежней версии. При этом БД
не заменяется, даже если старый процесс успешно восстановлен.

Если схема изменилась, неизвестна или восстановленный executable неготов,
сервис остаётся остановленным. При несовместимости current сохраняет новый
executable, прежний не запускается. Создаётся durable private boot guard
`/var/lib/accord/START_REQUIRES_INSPECTION`; он запрещает systemd запуск и
после перезагрузки. Первая неудачная установка также остаётся остановленной
с этим guard. Оператор должен проверить схему и совместимость установленного
executable, устранить причину сбоя, затем явно удалить только этот guard и
запустить Accord с проверкой readiness. Скрипты guard автоматически не удаляют.
Это не restore: `RESTORE_REQUIRES_RESET` остаётся независимым запретом до
полного credential/lease reset; удалять его вместо inspection guard нельзя.

Откат на уже установленный, совместимый со схемой executable:

```sh
python3 /opt/accord/admin/rollback.py --revision <ПОЛНЫЙ_ПРЕЖНИЙ_GIT_SHA>
```

Проверяется сохранённый SHA256, делается snapshot, меняется только binary.
Неподдержанная схема должна вызвать отказ приложения, а не переписывание БД.
Если старому коду нужна старая схема, требуется явное восстановление snapshot
ниже. Оно теряет изменения после точки копии и требует перевыпуска credentials.
Самый первый выпуск не имеет установленной предыдущей версии для binary rollback.

## Автоматические копии

`accord-backup.timer` запускает `backup.py` каждый час с jitter до120секунд.
Persistent timer догоняет пропущенный запуск после простоя. Wrapper вызывает
`agent-board backup`, который обязан создавать consistent online snapshot,
проверять SQLite integrity, сохранять0600и синхронизировать файл на диск.
Обычное копирование живого `board.db` без WAL недопустимо.

К каждому завершённому snapshot записывается SHA256; после успешной копии
остаются168последних завершённых пар. Это около7дней при hourly расписании,
а не обещание7календарных дней при ручных дополнительных запусках. Незавершённые
файлы не считаются валидной копией и автоматически не удаляются. Ручной запуск:

```sh
systemctl start accord-backup.service
systemctl status accord-backup.service --no-pager
journalctl -u accord-backup.service --since today --no-pager
```

Неудачный запуск остаётся failed в systemd/journal. Внешняя доставка уведомлений
и offsite-хранилище не настроены. Копии на том же VPS защищают от ошибки данных,
но не от потери VPS/диска. До подтверждённого offsite администратор должен
периодически забирать закрытую пару DB/SHA256 по SSH на другой диск; она содержит
историю, token hashes и idempotency, поэтому публиковать её нельзя. Приложение
не пишет содержимое задач или секреты в backup journal.

## Восстановление

Сначала read-only проверка выбранного snapshot:

```sh
python3 /opt/accord/admin/restore.py \
  --backup /var/backups/accord/board-<TIMESTAMP>-<UUID>.db --check-only
```

Она проверяет regular0600файлы, владельца, SHA256 и `PRAGMA integrity_check`.
Следующая команда изменяет только Accord и требует осознанного выбора точки
восстановления; выполнение не является частью обычного обновления:

```sh
python3 /opt/accord/admin/publish.py --disable
python3 /opt/accord/admin/restore.py \
  --backup /var/backups/accord/board-<TIMESTAMP>-<UUID>.db
```

Restore ставит boot guard, останавливает timer/backup/Accord, делает копию
прежней базы, сохраняет прежние DB/WAL/SHMв отдельном0700каталоге, атомарно
ставит проверенную копию и оставляет процесс **остановленным**. Guard
`/var/lib/accord/RESTORE_REQUIRES_RESET` блокирует в том числе автозапуск после
перезагрузки. При ошибке не снимайте его ради зелёного статуса.

Snapshot возвращает token/session/lease состояние на прошлый момент и может
вернуть ранее отозванные tokens. Перед снятием guard обязательно выполните
штатный offline reset под accord: он атомарно отзывает все credentials,
инвалидирует восстановленные сессии/аренды и оставляет audit event.

```sh
runuser -u accord -- /opt/accord/current/agent-board maintenance reset \
  --db /var/lib/accord/board.db --reason restore
```

Проверьте успешный результат reset прежде, чем снимать guard. Выпустите новые
токены; не восстанавливайте доступ прежним secret-файлом.

Только после credential reset/lease проверки root снимает guard, запускает
Accord, проверяет readiness и401со старым токеном, выдаёт новые токены и
возвращает публикацию и timer:

```sh
rm /var/lib/accord/RESTORE_REQUIRES_RESET
systemctl start accord.service
curl --fail --max-time 5 http://127.0.0.1:7892/readyz
systemctl start accord-backup.timer
python3 /opt/accord/admin/publish.py
```

Restore drill выполняется **на изолированной копии**, отдельном каталоге/порту
и synthetic данных. Проверяется реальное сохранение задач, event cursor,
идемпотентности, отзыва токенов и lease fencing; затем token reset и новый
сеанс. Работающий общий сервер ради drill не откатывается. Хеши backup,
результат integrity и runtime-доказательства записываются без credential/data.

## Диагностика

```sh
systemctl status accord.service accord-backup.timer --no-pager
journalctl -u accord.service --since today --no-pager
systemctl show accord.service -p User -p MemoryCurrent -p MemoryMax -p CPUQuotaPerSecUSec
```

Остановка Accord: `systemctl stop accord.service`. Остановка его privateHTTPS:
`python3 /opt/accord/admin/publish.py --disable`. Оба действия не должны менять
Tandem. Не удаляйте БД/копии/issuedcredentials при диагностике; не применяйте
`chmod -R` или `chown -R` к чужим каталогам. После обновления проверяйте
Tandem health и неизменность HTTPS443 отдельно.

Основания выбора: [Tailscale Serve](https://tailscale.com/docs/reference/tailscale-cli/serve),
[systemd execution sandbox](https://www.freedesktop.org/software/systemd/man/255/systemd.exec.html).
