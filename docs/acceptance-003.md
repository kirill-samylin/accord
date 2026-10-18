# TASK-003 — независимая процессная приёмка

Набор **003.1**, tester WORK-028,18.10.2026. Код реализации принят отдельно:
[review-003](review-003.md) APPROVE и [security recheck](security-003-recheck.md)
APPROVE. Тестировщик не менял application code/infra scripts.

## Тест-кейсы

Воспроизведение: `make check` из корня с Go и Python3. Для исключения Go test
cache этот прогон использовал ignored wrapper, добавляющий `-count=1 -json`
только к `go test`; vet/build получают обычные аргументы. Каталоги synthetic
SQLite и tokenfile создаются0700/0600; очищаются только собственные subprocess.

| ID | Действия | Ожидаемый результат |
| --- | --- | --- |
| P003-01 | Запустить настоящие CLI server и stdio MCP child; describe/next; не обращаться к bridge до прохождения серверным HTTP Date более2 начальных taskTTL; закрыть stdin; дождаться expiry; взять задачу другим CLI | Background pulse удерживает прежнее generation; EOF останавливает child; естественное expiry, новый generation+1; stale comment отклонён |
| P003-02 | Через bridge взять и release задачу; повторить прежний next; дождаться исходногоTTL; остановить child и перезапустить CLI server; повторить сохранённый next через CLI | Исторический response идентичен; released lease не возобновлена, events/attempts не дублируются; история сохраняется после SIGTERM/restart |
| P003-03a | Между настоящими CLI server/bridge поставить HTTP proxy fixture, вернуть503 на вызов comment; восстановить proxy; выполнить get и пять мутаций | LEASE_UNCERTAIN необратим; диагностический get работает; мутации не доходят до upstream; комментарий отсутствует, generation не растёт, lease истекает |
| P003-03b | Отозвать synthetic bearer локальной CLI при живом bridge и удерживаемой задаче | Независимый pulse обнаруживает revoke; stderr сообщает quarantine; пять мутаций заблокированы без HTTP; lease истекает без reacquire |
| P003-04 | CLI backup при живом server и непустом WAL; проверить SHA/schema/privatefile/no-overwrite; отозвать token после backup; остановить source; проверить task/events/replay в отдельной копии; остановить копию, maintenance reset; создать новые tokens | Snapshot содержит task/lease/history/comment/events/idempotency; reset отзывает оба старых credentials, закрывает sessions/lease с аудитом; свежий token не использует старую session; новый claim увеличивает fencing |
| I003-01 | Persistent Python fault tests install/rollback/recovery: backup/start/readiness failure, schema1/unknown/migration/unversioned DDL, первый install, fallback failure, оба guard, DB symlink, WAL | Совместимый fallback запускает прежний executable и сохраняет записи candidate; несовместимость/неудача оставляет stopped+guard; DB не откатывается; оба entrypoint используют один lifecycle |
| I003-02 | Реальные private file/symlink/SQLite/fsync с injected directory fsync failure; mock service commands | File затем directory fsync до первого stop; directory error не вызывает stop и не меняет существующую DB; inspection guard также durable; pointer switch атомарен |

Новые Go файлы: bridge_process (helpers), bridge_lifecycle, bridge_fault,
backup_process. Пять прежних process cases сохранены без изменений. В Python
перенесены17 ранее ignored regression cases WORK025 плюс две независимые
проверки сохранения candidate writes и настоящего symlink switch.

## Фактический прогон

**PASS** на этом Mac, darwin/arm64, Go1.27.2; полный `make check` завершился0:
format/vet, свежий Go test всех пакетов,19 Python tests, свежий Go race всех
пакетов и build. Acceptance package: **9 top-level tests**, включая две
fault subcases; нормальный запуск15.089s, race-run16.032s. Все новые кейсы
P003-01–04 и I003-01–02 PASS; все прежние process cases PASS.

Первый targeted run выявил неверный expected code в новом тесте: свежий token
для чужой восстановленной session возвращает UNAUTHORIZED, как требует
sessionFor; ожидание NO_SESSION исправлено. Product code не менялся. После
этого targeted backup case и полный свежий набор PASS.

Все75 implementation files cmd/internal/infra совпали с обоими независимыми
review/security manifests. Единственная правка уже рассмотренного build
wiring — Makefile: infra-test добавлен в check. Новые тесты и эта wiring
дельта требуют следующего независимого review; этот протокол его не заменяет.

## Точная версия и доказательства

- База966dda6 + неизменённая принятая implementation дельта TASK-003.
- Manifest12 test/wiring files: SHA256
  `5b8e0c83ec9def05e5e00216146d5ecb4480eef39aa3a1189f4669a19dfb922b`.
- Полный make-check log: SHA256
  `d6854b3a864dde75b716ed7ae0867777f1e1af44379ae6c256585de9e276c344`.
- JSON результаты/измерения: SHA256
  `040aa3ee4301bab35fc7cdc15d8c01a0ca70e489fc9754fed57727b9f9bf5110`.
- Private ignored evidence: `.local/evidence/acceptance-003/`, в том числе
  tests.sha256, review-consistency.json, go-uncached, make-check.log, results.json.

## Границы

Process tests используют обычный executable из TestMain. `go test -race`
проверяет Go test packages; дочерний executable отдельно с race не собран.
HTTP503 proxy — явный test double сетевого отказа, не удалённый TLS/VPS.
Python19 — реальные synthetic filesystem/SQLite, но systemctl/runuser/readiness
mocked; это не Linux/systemd runtime и не моделирование отключения питания.
Восстановленная копия ненадолго запускается на private loopback до reset
только для проверки точности snapshot; эксплуатационная процедура требует
reset до предоставления доступа восстановленным данным.

NOT-RUN этим tester: live VPS/TLS/backup timer/ports/sandbox, реальный remote
Codex/Claude, второй физический ПК, Windows runtime/ACL, offsite и powerloss.
Эти границы не являются PASS полной V1. Независимое review новых tests pending.
