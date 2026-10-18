# Review076 — process tests072

**APPROVE** test/documentation delta commit
`f94c95d19ab7516511e32533f99b1c6430c705b3` относительно
`c1ef9ae2e8176064e1e652a9b1d480bf7b038943`. Блокирующих замечаний нет.
Это независимое ревью новых тестов; оно не является разрешением выпуска D4
и не закрывает annual/VPS acceptance или весь TASK-003.

Ревьюер `/root/review076` не автор реализации или тестов072. Прочитаны
Accord AGENTS, общий процесс, parallel-work, профиль reviewer и правила
декомпозиции. Named Claude model bindings недоступны; используется inherited
модель. Production source, соседний проект, сервер, токены и Git не изменялись.

## Версия и достоверность

- Проверен весь diff12 файлов, окружающие fixtures и строгий CAS guard.
  SHA256 всех12 файлов совпадают с `.local/work072/delivery-manifest.json`.
- Независимо пересчитаны182 tracked source/build/dependency файла:
  все совпали с freeze072; aggregate
  `75de79a54889210e0739111de35f660284385d684f0159f32b52c5607a19501b`.
  Commit072 не меняет production code.
- Пересчитан полный лог тестировщика:
  `5a8a33838dddaf598a11dbb22f13a3caf3ef5f3cc95641a697f2399ad932913f`.
  Проверены recorded command/result, exit0,121.056s, оба37-case acceptance
  stages, Python106, race/build и открыто указанные skip/warnings.
- `git diff --check c1ef9ae HEAD`: PASS. Полный check повторно не запускался.

## Содержательность assertions

`expectedCurrentSchema=4` задаётся тестом явно и не выводится из константы
реализации. Native C2/D3 fixtures создаются историческими исполняемыми файлами;
schema1 использует сохранённый canonical SQL fixture. Положительный D3 path
не имитирует предыдущую версию одной заменой PRAGMA. Старые raw JSON/replay
guards, snapshot checksum/modes, scopes и reset/fencing assertions сохранены.

P072-01 сравнивает точные значения восьми исходных таблиц до runtime Sweep,
проверяет backfill/indexes/triggers, неизменность SQL при restart и точный
отказ старого D3 без изменения DB bytes. P072-02 создаёт реальный DDL conflict
и требует неизменности полного логического SQL state, schema3 и отсутствия
частично созданных объектов. SQL helper сортирует row encodings, сохраняя
raw string cells; `any` здесь нужен для динамической формы SQL Scan,
а не для обхода валидации продуктового контракта.

P072-03 использует настоящий REST, естественное истечение SID, exact replay,
отдельный offline CLI scopes/config change и тот же live PID. Отказ после
изменения текущих полномочий и после restart должен быть FORBIDDEN_ROLE403,
без durable SQL writes и без renewal. P072-04 проверяет обе стадии ошибки
Create, live cache, SQL и restart, сохранение request_id, однократный reopen
каждого родителя, точный replay и доступность нового child для claim.

CAS тест не ослаблен: два CLI с одним expected version дают ровно одного
winner, второй обязан получить CONFIG_CONFLICT; дополнительно проверяются
active version/hash, history и audit delta. SQLITE_BUSY не принимается.
Backup/reset drill проверяет реальный WAL snapshot, credentials/SID revoke,
историю, notes/config и последующую generation. Python recovery по-прежнему
явно mocked; schema3→4 запрещает небезопасный возврат прежнего executable,
unchanged4 допускает проверенное восстановление.

## Независимый выборочный replay

2026-10-18, darwin/arm64, Go1.27.2, `GOTOOLCHAIN=local`, pinned root
GOCACHE/GOMODCACHE/GOTMPDIR. Выполнено `go test ./tests/acceptance -count=1 -v`
с `-run` ровно следующих трёх top-level tests:

| Test | Результат |
| --- | --- |
| TestPerformanceNativeD3MigrationProjectionRestartAndOldRefusalProcess | PASS |
| TestPerformanceSchema3DDLFailureRollsBackAllTablesProcess | PASS |
| TestPerformanceReplayUsesCurrentSQLScopesAfterExpiryProcess | PASS обе restriction |

Exit0, пакет5.851s. Historical D3 SHA256
`3075235e87ff260b41d7e44bec7cf5af927fc0ffacb018d553a3fc4a934373ac`.
Текущий independently built ordinary child SHA256
`86eb0046ea5da7a277e1e0bc7c182f3e0cce37f147c299fdf282fa66e30e1378`.
Его hash не подменяет historical072 child hash: replay выполнен уже на
committed f94c95d, тогда как072 строился до commit; source freeze проверен
отдельно. Полный набор и reopen/CAS/backup результаты приняты по frozen
логу тестировщика и проверке assertions, а не названы собственным повтором.

## Структура, рекомендации и пределы

Новые Go-файлы≤133 строк, общий SQL/native helper вынесен, новых одинаковых
реализаций не обнаружено. `test_recovery.py` вырос300→339 физических строк,
но содержит299 непустых строк без комментариев, то есть остаётся внутри
согласованного max-lines ориентира. Перенос существующей RestoreOrderingTests
в отдельный модуль возможен при следующем изменении recovery, не требуется
для этой приёмки. UI/API/dependency changes отсутствуют; UI-kit N/A.

Три ignored SQLite destructor ResourceWarning в полном072 логе открыто
зафиксированы; изменённый recovery targeted был clean. Рекомендуется отдельный
cleanup прежних fixtures `test_activation_atomic.py`, без заявления clean
warnings для всего106-case набора. Это не маскировка падения072 и не основание
повторять полный check без изменения.

Annual method/draft,200k target, Linux/VPS/resources, physical second PC,
actual Claude/Codex, OAuth/UI и crash/power-loss **NOT-RUN076**. Ordinary
child остаётся без `-race`, даже под race harness. Известный R070B01 старого
`backup_process*.py` вне delta072;074 отдельно проверил исправление073.
076 не переносит approve на ещё не слитые комбинации и не объявляет эту
методику включённой в make check072.
