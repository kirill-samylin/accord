# Релизы и выкатка на стенд

Процесс от 25.10.2026. Он не отменяет capable-оператор
[`infra/accord/projects/upgrade.py`](../infra/accord/projects/README.md): все
переключения на сервере по-прежнему делает он, с полным backup всех проектов и
roll-forward. Новое здесь — автоматические проверки, воспроизводимая сборка по
тегу и один драйвер выкатки, который используют и оператор с ноутбука, и
GitHub Actions.

## Версии и ветки

- Рабочие ветки `codex/…` → pull request в `main` → merge после зелёного CI.
- Релиз — тег SemVer `vMAJOR.MINOR.PATCH` на проверенном коммите (`v0.1.0` —
  первый управляемый релиз, 25.10.2026). До полного V1 major остаётся `0`.
- Сервер идентифицирует релиз **полным SHA коммита** (`SOURCE`, журнал
  `/var/backups/accord-projects/<sha>`). GitHub Release хранит пару
  `tag ↔ REVISION`; один коммит нельзя выпустить повторно под тем же SHA, если
  его журнал на сервере уже есть.
- Откат данных запрещён. «Откат» — это новый релиз (следующий тег) с нужным
  кодом, выкатанный тем же путём вперёд.

## CI (`.github/workflows/ci.yml`)

Запускается на каждый push и pull request, три независимых job:

| job   | что выполняет                                   | эквивалент локально |
|-------|-------------------------------------------------|---------------------|
| go    | gofmt, vet, unit/integration тесты, race, build | `make fmt-check vet test race build` |
| infra | Python unittest по `tests/infra`                | `make infra-test`   |
| web   | prettier, tsc, node --test, `next build`        | `make web-install web-check` |

Версии закреплены теми же числами, что и в [development.md](development.md):
Go 1.27.2, Node 24.19.0, Python 3.12.

## Сборка релиза (`.github/workflows/release.yml`)

Push тега `v*` собирает из этого коммита и публикует GitHub Release:

- `accord-<tag>-server-linux-amd64.tar.gz` — вывод `infra/accord/build-release.sh`:
  Linux Go binary, `infra/`, `SOURCE`, `SHA256SUMS`;
- `accord-<tag>-next-linux-x64.tar.gz` — вывод `infra/accord/build-next.sh`:
  standalone Next, Linux Node 24.19.0 (архив сверен с закреплённым SHA-256),
  Linux sharp, `manifest.json`, `SOURCE.json`;
- `agent-board-<os>-<arch>` — клиентские binaries для worker/MCP;
- `REVISION` и `SHA256SUMS` для всех assets.

Локально то же самое делает `make release` (из чистого закоммиченного дерева,
Node 24.19.0 в `NEXT_NODE`): результат в `.local/releases/<sha>` и
`.local/releases/<sha>-next`.

## Выкатка (`.github/workflows/deploy.yml`, `infra/accord/rollout.py`)

Драйвер `infra/accord/rollout.py` ходит на сервер только по SSH и ведёт фазы:

```
upload   — новые каталоги /opt/accord/releases/<sha> и /opt/accord/next/releases/<sha>,
           сверка каждого файла по SHA256SUMS / manifest.json; active не меняется
stage    — upgrade.py stage с immutable hash'ами и текущим SHA nginx-конфига
apply    — upgrade.py apply: full snapshot, переключение pointers, миграция
smoke    — внешний HTTPS: /readyz и /app/login (200, с TLS-проверкой)
finalize — upgrade.py finalize: новый полный backup, журнал complete
prune    — удаление старых каталогов релизов (не backup'ов данных)
```

Фазы идемпотентны по журналу: повторный запуск продолжает с записанного
состояния (`staged`, `pending-https-smoke`, `complete`). Прерванный apply
(`activation-started`, `blocked-roll-forward`) продолжается только явным
`--resume` после осмотра журнала; автоматических retry/rollback нет.

Ручной запуск с ноутбука (ключ и known_hosts — приватные файлы оператора):

```sh
python3 infra/accord/rollout.py all \
  --target root@СЕРВЕР --identity /path/to/key --known-hosts /path/to/known_hosts \
  --go-release .local/releases/SHA --next-bundle .local/releases/SHA-next \
  --public-url https://СЕРВЕР --receipts .local/rollout/SHA --prune --keep 1
```

`rollout.py status` печатает инвентарь сервера (pointers, релизы, журналы,
SHA nginx, свободное место) без изменений.

В GitHub Actions workflow **Deploy** запускается вручную (`workflow_dispatch`)
с тегом и фазой, в environment `production`; одновременно идёт не больше одной
выкатки. Квитанции фаз сохраняются как artifact. Для него нужны:

| имя | тип | значение |
|-----|-----|----------|
| `ACCORD_VPS_TARGET` | variable | `root@host` (или отдельный deploy-пользователь с sudo-less правами root на `/opt/accord`, `/var/lib/accord`, systemctl) |
| `ACCORD_PUBLIC_URL` | variable | `https://host` стенда |
| `ACCORD_VPS_SSH_KEY` | secret | приватный ключ **только для выкатки** (отдельный, не личный) |
| `ACCORD_VPS_KNOWN_HOSTS` | secret | строка `known_hosts` сервера |

Environment `production` стоит защитить required reviewers: тогда каждый
запуск Deploy ждёт подтверждения человека.

## Старые копии на сервере

Каждый релиз оставляет каталоги в `/opt/accord/releases` и
`/opt/accord/next/releases` (~200 МБ на релиз). Они не нужны capable-updater'у
(возврат назад запрещён), поэтому фаза `prune` оставляет активный релиз,
выкатываемый и `--keep N` предыдущих (по умолчанию 1), остальные удаляет.
Снимки данных `/var/backups/accord-web` (семь последних, ротация в
`web/backup.py`), журналы `/var/backups/accord-projects/<sha>` и
`/var/backups/accord-next` не трогаются никогда.

## Что делать при ошибке

1. Прочитать stderr драйвера и журнал `/var/backups/accord-projects/<sha>/state.json`.
2. `staged` / `pending-https-smoke` — исправить причину и повторить ту же фазу.
3. `activation-started` / `blocked-roll-forward` — осмотреть сервер, затем
   `rollout.py apply --resume …` (вызывает `upgrade.py resume`).
4. Нужен другой код — новый тег и новый релиз вперёд; старые binary/БД не возвращаются.
