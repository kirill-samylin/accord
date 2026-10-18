# Независимое ревью тестов TASK-003

Вердикт: **APPROVE** для новых process/infra tests, Makefile wiring и
проверенной клиентской инструкции. WORK-030,18.10.2026. Проверяющий не писал
реализацию или тесты; approval относится к этому срезу, не к ещё не
выполненной приёмке VPS или полной V1.

## Область и вывод

Прочитаны четыре новых Go-файла process acceptance, окружающие helpers и
пять прежних сценариев, три постоянных Python-файла, Makefile, протокол
WORK-028 и его JSON/log/manifests. Утверждения проверены относительно service/
security contracts и независимо принятых implementation review. Блокирующих
дефектов в тестах не найдено. Необоснованных подавлений ошибок, секретов в
выводе или ложного PASS не обнаружено.

- Heartbeat проверяется настоящим отдельным stdio CLI через OS pipes и MCP
  SDK: авторитетный HTTP Date проходит более двух исходных taskTTL при
  бездействующей модели; сохраняются generation и одна попытка. EOF требует
  чистого выхода child, после естественного expiry successor получает +1,
  старый comment отвергается. Health polling сам аренду не продлевает.
- Replay после release проверяет идентичный historical DTO, отсутствие новой
  lease/attempt/events, затем ту же историю после настоящего server restart.
- Network503 — явно обозначенный proxy test double между настоящими
  CLI-процессами. После восстановления get разрешён, пять изменяющих операций
  остаются в quarantine и не достигают upstream. Revoke обнаруживает фоновый
  pulse; итоговая БД не содержит комментария или нового generation, lease
  истекает. Это не claim о реальном сетевом timeout/VPS.
- Snapshot создаётся CLI при непустом живом WAL: проверяются содержимое
  task/history/comments/events, replay, checksum/schema,0600 и no-overwrite.
  После offline reset старые credentials и session не работают, история
  сохранена, successor увеличивает generation; восстановленные токены не
  считаются рабочими credentials. Временный private-loopback запуск копии
  до reset явно отделён от эксплуатационной процедуры.
-19 Python cases используют реальные private SQLite/files/symlinks/fsync,
  но mocked service/readiness/runuser. Есть негативные guards для неизвестной
  схемы/DDL, отказов backup/start/fallback и directory fsync; candidate data
  сохраняются при возврате прежнего executable. Это полезные постоянные
  регрессии с узкими структурными проверками общего activation lifecycle.
- Таймауты ограничены; cleanup завершает только собственные PIDs. Stdout
  bridge проверяется как JSON-RPC, synthetic bearer отсутствует в stdout/
  stderr. Новые source/test файлы укладываются в300 строк.

Makefile включает infra-test в check с передачей ненулевого exit. Wrapper
зачётного прогона WORK-028 использует exec и добавляет count=1/json только к
Go tests, не скрывая exit. JSON не содержит fail; hashes полного log/results/
12-файлового test manifest совпали с протоколом. Application/infra native
source совпадает с review/security manifests; ожидаемые последующие дельты —
Makefile и отдельно просмотренное уточнение README о Python3.

## Инструкции

В docs/client-setup.md корректно разделены один stdio process/одна board-session
и direct HTTP с самостоятельным heartbeat разных агентов. Bridge не обещает
автоматический lifecycle внутренних субагентов и физическую блокировку edits.
Windows env fallback использует существующий ReadToken путь, не приравнивает
POSIX mode к ACL и явно остаётся NOT-RUN. Токен не вставляется в TOML/argv.
command/args/env_vars/required/bearer_token_env_var сверены с
[официальной OpenAI документацией](https://learn.chatgpt.com/docs/extend/mcp?surface=cli).
README и development уточнены координатором по замечанию ревью: Python3
stdlib unittest/sqlite3 нужен для check, pip не требуется, обычный build от
Python не зависит; mocked infra не назван реальным Linux runtime.

## Независимые проверки и версия

На Mac darwin/arm64, Go1.27.2 повторены три top-level process tests:
HeartbeatAndEOF, FaultQuarantine(network/revocation), OnlineSnapshotAndOfflineRecovery
с count=1/timeout90s — **PASS**, package9.628s. Полный E2E повторно не запускался
по профилю reviewer; зачётный свежий полный test/race/check — WORK-028.
`make infra-test` — **19/19 PASS**,0.037s; `git diff --check` — PASS.

Manifest16 (все acceptance Go, три infra Python, Makefile, client-setup,
README/development и acceptance protocol) сохранён в ignored
`.local/evidence/test-review-003/reviewed.sha256`, SHA256:
`65f141a9dcc5115c5eee1f9f41f15288a6430438eaba9f71c44e5653247d102f`.
Process log SHA256 `7d2fca703f2a4b40c7ad7cf191994dafec8d46a3b9850019762fc9e4545aa7fd`;
infra log `5c3c78e8c74990215fdfbbfd9c5bf8fc2e2fbd7f2cb1076e44aed64f55910e66`.
Проверен также исходный test manifest WORK-028:
`5b8e0c83ec9def05e5e00216146d5ecb4480eef39aa3a1189f4669a19dfb922b`.

Live TLS/systemd/backup timer/restore, реальный remote Codex/Claude, Windows
runtime/ACL, физический второй ПК и отключение питания этим reviewer NOT-RUN.
Следующий шаг — проверяемая native deployment acceptance; этот approve её
не заменяет. Изменение тестов или инструкции требует адресного recheck.
