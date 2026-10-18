# WORK-028 — независимая процессная приёмка TASK-003

- Роль: tester, исполнитель e2e003; код реализации не писал.
- Статус: тесты завершены, передано на независимое review после APPROVE
  review-003 и security-003-recheck.
- Входы: AGENTS/TASK-003, service/security contracts, WORK021–027;
  профиль tester и общий процесс соседнего проекта прочитаны.
- Запись: новые tests/acceptance/*_test.go, tests/infra/**, Makefile wiring,
  docs/acceptance-003.md и эта WORK. Product/infra scripts не меняются.
- Все данные synthetic, каталоги private; VPS/Tandem не используются.
- Именованные Claude-модели здесь не закреплены; используется модель родителя.

## План проверки

1. Реальные CLI server и stdio MCP child: heartbeat >2TTL, EOF и новый fencing;
   сетевой отказ/отзыв токена → quarantine без мутации/повторного захвата;
   исторический replay/restart без duplicate effects/hidden task renewal.
2. CLI online snapshot при живом WAL; точная история/replay в отдельной копии;
   offline reset, отказ старым credentials/session, новый fencing generation.
3. Постоянные Python fault regressions install/rollback/recovery/restore:
   явно mocked systemctl/runuser/readiness, настоящий private SQLite/fsync.
4. Полный make check после additions, manifest review/source consistency;
   затем отдельный reviewer новых тестов. Process binary обычный build;
   race проверяет Go suite, не автоматически дочерний executable.

Physical2PC/Claude/удалённый Codex/Linux systemd/powerloss здесь NOT-RUN.

## Результат

Четыре новых top-level process tests (network/revocation — два subcases),
19 persistent Python infra tests, `make infra-test` включён в check. Старые
пять process tests не изменены. Никаких source/infra исправлений не потребовалось.
Первый targeted run исправил только ошибочное ожидание NO_SESSION на чужой
session: фактический договор UNAUTHORIZED. Последующий targeted backup и
полный свежий make check PASS; acceptance15.089s/race16.032s, Python19/19.
Implementation75 файлов соответствует обоим review manifests без дельт.

[Протокол и воспроизводимые кейсы](../acceptance-003.md). Private evidence:
`.local/evidence/acceptance-003/`. Test manifest12 SHA256
`5b8e0c83ec9def05e5e00216146d5ecb4480eef39aa3a1189f4669a19dfb922b`;
make-check log `d6854b3a864dde75b716ed7ae0867777f1e1af44379ae6c256585de9e276c344`;
JSON results `040aa3ee4301bab35fc7cdc15d8c01a0ca70e489fc9754fed57727b9f9bf5110`.
Все новые файлы до300 строк. Самостоятельное independent approve не выставлено;
следующий шаг — отдельный reviewer tests/Makefile, затем Linux/live acceptance.
