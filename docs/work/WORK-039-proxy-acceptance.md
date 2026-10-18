# WORK-039 — приёмка внешнего Host на настоящем CLI-процессе

- Родитель: [TASK-003](../TASK-003-working-service.md).
- Роль: tester; исполнитель `/root/proxyqa003`.
- Статус: выполнено; этап: передача новых process/E2E на независимое ревью.
- Вход: native `dc9c74cd692e287173def9fb5743dbf18ffdef21`, parent
  `99e722c44f83b1808120bba1fabb04cf2f68a5b5`, clean snapshot в изолированной
  `.local/worker-source-003`; основной C WIP не используется.
- Scope записи: новый `tests/acceptance/proxy_process_test.go` только в native
  worktree, эта WORK и `docs/acceptance-039-proxy.md` в основной рабочей копии,
  ignored evidence. Производственный код, Tandem, сервер, commit/push не входят.
- AGENTS Accord, общий процесс, parallel-work и tester profile прочитаны.
  Именованные модели Claude в этой среде не закреплены; унаследована модель
  родителя. Независимое code/security APPROVE проверено в WORK-036.

## План

Набор версии1: настоящий CLI-сервер, private SQLite/token, TLS reverse proxy
с явным Host, official SDK initialize/list/describe/create/get; отдельный
actual-socket HTTP matrix: чужой Host/порт, отсутствующий bearer, present
Origin и spoofed forwarded headers. CLI startup fail-closed и env fallback;
loopback остаётся совместим. TLS proxy синтетический локальный, SDK не Codex.
Targeted normal и race-harness прогоны; child binary обычный, не race build.

## Доказательства

`.local/worker-source-003/.local/evidence/proxy-acceptance-003/`:
initial snapshot clean, 164 tracked файла; source manifest SHA256
`934a94fc8c3bc825ba7c924d2a3aade780bb6fecdd1c2d8a4e65cb6749db143b`;
parent diff SHA256 `215fe50ba100a5b0ae2f4489586ffe4558953d1a886f044b023249501c5617a7`.
Результат: **PASS**, normal1.637s/race-harness2.743s,4 top-level и17 subtests
в каждом; vet/gofmt/diff-check PASS. CLI child ordinary binary SHA256
`d2816118a41dc72948c9788f5856753a3340c090bb9f80c3b37f7ec5432b67a8`.
Исходные164 tracked файла остались неизменными. Новый test source SHA256
`58b00ee32775350b3d11b189805f24e7c431b908f0aabaf61207f70d4a568ee5`;
result manifest7 `9a4d09652ad3b4ebd7c71cae06885c4c9e4df53ad5c0b91deb6e8687bf27890a`.
Полный [протокол и воспроизводимые кейсы](../acceptance-039-proxy.md).
Live Tailscale/Codex — отдельный результат координатора, metadata прочитаны;
тестировщик не запускал модель/серверные изменения. Два начальных fixture fail
сохранены и объяснены, production source не исправлялся.

## Журнал

| Время Europe/Moscow | Событие | Статус | Основание |
| --- | --- | --- | --- |
| 2026-10-18T18:18:34+03:00 | started | в работе | Clean native snapshot, актуальное independent APPROVE; scope подтверждён |

| 2026-10-18T18:22:24+03:00 | completed / handed_off | выполнено | Native process delta PASS, source unchanged; новые tests переданы координатору для reviewer |
