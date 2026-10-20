# WORK-151 — координация: operations и интеграция

20.10.2026. Root. Scope: infra/accord/projects, tests/infra, канонические docs,
Git/release/VPS integration. Backend148 и frontend149 имеют отдельное владение;
review150 независим. Контракт принят по WORK147 в docs/coordination.md.

## План/границы

Schema4/coordination1 floor до DDL; full snapshots2/3/4 и исторические1/2/3
migration copies/checksums; additive logical preservation всех старых tables,
runtime/systemd guards прежние. Initial TASK009 apply не повторять, DB rollback
запрещён. Authority marker UID/mode/bytes не менять. Реальные данные не скачивать.

Статус: реализация оператора IN PROGRESS, source review/QA/Linux/VPS NOT-RUN.
Исходная production версия cbf2020 только передана; до поставки нужна проверка.

Объём: guard требует coordination1 и schema4; layout/validation принимает полные
2/3/4; migration history включает strict schema3 files; upgrade_storage проверяет4.
Новые infra unit для snapshots/metadata/history и отказа прежнему schema3 binary.

## Промежуточная проверка

Root targeted storage14/guard7 и fullinfra155 PASS; independent150 повторил155
и подтвердил slice APPROVE. SQLite ResourceWarnings прежних fixtures не FAIL.
Новые7coordination unit используют реальные SQLite snapshots/history/copy; native
maintenance-reset только mocked в offline orchestration unit, это не process PASS.

Readonly production preflight PASS: actual currentcbf2020 Go/Next pointers, journal
complete, installed schema3 guard/runtime/effective config, validated current full
backup, markers0, services/timer active. Чтение не изменяло pointers/данные/guards.
Private baseline `.local/task011/deploy/before-inventory-private.json`; реальные
DB/credentials не копировались. Schema4/release/поставка пока NOT-RUN.

Для actual3→4 QA собран clean macOS schema3 baseline из343b93c в отдельном
managed coordination-baseline checkout. Go VCS revision/modified=false и
capabilities проверены, SHA4e11d2569120dada3df15061bcfc5193aacb7931a048110f27242f0b9419e085.
Executable/proof сохранены private `.local/task011/legacy`; временный checkout
архивирован после сборки. Это baseline fixture, не новая production поставка.

## Source freeze

WORK150 общий SOURCE APPROVE; root independently matches74files manifest
5cd77400d904fe72783b3beabda63b1fa3a4137ddfc914e651a3347548cc672a.
Полный root make check PASS (Go/all/race/vet/build, frontend, infra155,
Next67/types/format/build), log `.local/task011/full-check-source.log`.
Tester152 теперь разрешено писать/запускать новые process/E2E. Source сохраняется
до QA findings; релиз/production пока NOT-RUN.
