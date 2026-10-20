# Next UI: поставка на существующий Accord VPS

> После активации TASK009/010/011 этот исторический TASK008 оператор и rollback
> не применяются. Следующие capable обновления — только
> [projects/upgrade.py](../projects/README.md), с all-project snapshot и roll-forward.

Это узкий оператор TASK008, не первоначальный установщик. Требует root/Linux,
существующие Go7892, nginx public include и reviewed web backup service. До
stage/apply нужны независимый review и реальные значения из приватного inventory.
Скрипты не получают production токены и не читают содержимое БД.

## Сборка

`web/next.config.ts` включает standalone. Из `web/` собрать production с
`ACCORD_GO_ORIGIN=http://127.0.0.1:7892` на проверенной Node24.19.0. Сборка на
маленьком VPS не требуется. Упаковка принимает Linux-x64 Node из официального
архива Node24.19.0, заранее сверенного с официальным SHASUMS256.txt. SHA256
извлечённого bin/node передаётся отдельно; Node LICENSE обязателен рядом с bin/.

```sh
python3 infra/accord/next/package.py --revision FULL_COMMIT \
  --node .local/task008/node/linux/bin/node --node-sha256 VERIFIED_BINARY_SHA \
  --output .local/task008/releases/FULL_COMMIT/next
```

Поставляются standalone server, `.local/next/static`, public/fonts/licenses и
Node LICENSE. Darwin sharp/libvips исключаются; Linux-x64 sharp/libvips берутся
по точным версиям/URL/SRI SHA512 существующего lockfile. Зависимости/lock не
меняются. Любой оставшийся native member должен быть ELF64 x86_64. Полный
manifest связывает каждый файл, BuildID, source revision и Node checksum.
Фактическая Linux загрузка Node/sharp проверяется в stage, переносимость не
выводится только из macOS сборки.

## Stage и переключение

Перенести bundle, reviewed operator directory и Linux Go binary в приватный
root-owned каталог VPS. Передать независимо сверенный hash manifest и Go binary:

```sh
python3 OPERATOR/operate.py stage --revision FULL_COMMIT \
  --bundle UPLOADED_BUNDLE --manifest-sha256 MANIFEST_SHA \
  --go-binary UPLOADED_GO --go-sha256 GO_SHA \
  --expected-nginx-sha256 INVENTORY_NGINX_SHA --port 7893 --stage-port 7894
python3 OPERATOR/operate.py apply --revision FULL_COMMIT
```

Stage проверяет immutable дерево, сохраняет старые nginx/unit/отсутствие unit,
проверяет кандидаты nginx/systemd и Linux runtime в отдельном временном hardened
unit7894. Публичных Node портов нет. У сервиса отдельный UID без групп Accord,
закрыт доступ к данным/backup/env; память ограничена320MiB, heap192MiB.

Apply сначала вызывает существующий согласованный board+web backup. Затем под
maintenance lock останавливает единственного Go writer, атомарно меняет Go и
Next pointers, запускает/проверяет Go и Node, проверяет/перезагружает nginx.
`/web/**` и exact upload идут напрямую в Go с прежними ограничениями и Origin;
Next обслуживает только `/app`, assets и license. `/` перенаправляет на `/app`.
`/_next/image` не публикуется: текущий UI не использует image optimizer.
Native, ACME, private routes, прежние env/drop-in/timers не переписываются.

После **независимого внешнего HTTPS/native/browser smoke** оператор вызывает:

```sh
python3 OPERATOR/operate.py finalize --revision FULL_COMMIT
```

Finalize делает новый согласованный snapshot и повторяет readiness/guards.
`pending-https-smoke` не является завершённой публикацией. Smoke не подменяется
локальным ответом Next; unit tests не заменяют Linux/HTTPS проверки.

## Возврат

```sh
python3 OPERATOR/operate.py rollback --revision FULL_COMMIT
```

Старые pointers, nginx, Node unit/enable-state восстанавливаются независимо;
ошибка одного файла не пропускает запуск прежнего Go writer. При неуспехе apply
или finalize выполняется автоматический rollback. SIGTERM/INT/HUP запускают
восстановление, повторные сигналы игнорируются во время recovery. SIGKILL/потеря
питания требуют ручного rollback по durable журналу. БД никогда не копируется
обратно: новые пользовательские записи сохраняются. Старые релизы и
root-only журнал `/var/backups/accord-next/FULL_COMMIT` не удаляются. Новый
системный UID после rollback допустимо оставить; он не имеет доступа к данным.
