# RUN-001 — приёмка локального среза 0A

Сводка зафиксирована 18.10.2026 в 13:10:13+03:00. Стенд: macOS arm64,
локальный Go 1.27.2, SQLite и настоящий HTTP/MCP/CLI без подмены хранилища.
Это приёмка локального среза, не полной V1 или двух физических машин.

## Результат

- PASS: `make check` — форматирование, `go vet ./...`, `go test ./...`,
  `go test -race ./...`, сборка `bin/agent-board`.
- PASS: `go mod verify` с GOMODCACHE из `.local/toolchains/cache/mod`:
  `all modules verified`.
- PASS: независимое [code review исправлений](review-recheck.md),
  [security review](security-recheck.md) и [ревью process-тестов](acceptance-review.md).
- PASS: [приёмка отдельного тестировщика](acceptance.md) — 5/5 process-тестов,
  без FAIL/SKIP. Независимый reviewer повторил 3/3 сценария.
- PASS: после общего check повторно сверены 46/46 файлов code manifest и
  4/4 файлов test manifest; исходники и тесты не менялись после approve.
- PASS: `git diff --check`; локальные Markdown-ссылки существуют, Go-файлов
  длиннее 300 строк нет. Поиск признаков токенов/приватных ключей и приватного
  адреса Tandem среди исходников и документов совпадений не дал. Это узкая проверка,
  а не доказательство отсутствия любых секретов или всех уязвимостей.

## Проверенная версия и доказательства

| Артефакт | SHA-256 |
| --- | --- |
| Code manifest, 46 файлов | `9cc3437a3f55c5a46859d8610997fb7e14ceb0a0cbcfc3fcd3864d2b03d006b4` |
| Test manifest, 4 файла | `4e24eacc793aaa24ea44e1faf1efd621139b97f47e61419d91b6b40a4949a6f8` |
| Итоговый make-check.log | `ea79b686dc57d1bef8069de22d77f1d54355364e41052b4c5bd335eb50bf9550` |
| Собранный bin/agent-board | `af7dacd5b6ee86d5bf4712985de793e90055d4f61f9db5c3104dc1ab28e1aa2d` |

Локальные журналы: `.local/evidence/final/make-check.log`,
`source-manifest-check.log`, `tests-manifest-check.log`; отдельные evidence
исполнителей — соседние каталоги `.local/evidence/`. Сырые данные и бинарники
в Git не включаются; исходники, человекочитаемые отчёты и контрольные суммы — включаются.

## Граница доказательств

Проверены настоящие отдельные CLI/server процессы на одном Mac, конкурентный
захват, запрос архитектору, TTL/fencing, retry, фильтрованный журнал,
graceful SIGTERM/restart и отзыв токена. «Потеря ответа» — явное отбрасывание
полного ответа тестовым клиентом; это не сетевая авария или kill в commit.

NOT-RUN: реальные Claude Code/Codex, два физических ПК, Linux/Windows,
удалённый TLS/proxy, OAuth, аварийный отказ диска/процесса во время commit,
нагрузочные p95/RSS/start цели и отдельный Govulncheck.
N/A: дизайн-ревью/UI и качество inference — UI и генерации моделей в срезе нет.
Развёртывание, push, PR, merge и выпуск не выполнялись.

Следующий этап — 0B из [roadmap](roadmap.md): реальные клиенты и две машины.
Переход к полной V1 не происходит автоматически из локального approve.
