# WORK-049 — C live deployment и изолированная нагрузка

- Исполнитель: `/root/live049`, DevOps; отдельный от реализации и review.
- Статус: выполнение завершено, на приёмке координатора; latency criterion FAIL, 18.10.2026.
- Вход: C source/Host APPROVE041, QA042, test review045; harness047 APPROVE048.
- Версия поставки: `1e9025cd6e3122f0adf6165e09845202736d004a`, schema2, только C.
- Запись: этот WORK, `docs/runtime-catalog-049.md`, ignored `.local/evidence/runtime-catalog-049/**`.
- Общая TASK/source/предыдущее evidence не изменяются; commit/push не выполняются.
- Профиль DevOps, общий процесс и parallel-work прочитаны. Именованные Claude-модели здесь не закреплены.
- Tandem board OFF по указанию пользователя; существующий Tandem/server443 сохраняется.
- Внешний второй ПК NOT-RUN; нагрузка не использует существующую БД/API.

## Результат

- Только approved immutable C `1e9025cd6e3122f0adf6165e09845202736d004a`
  установлен вместо dc9c74c. Binary SHA256
  `91ac8396cda97ee0577be6074eb9523f8ab8ce8beb7579cc52ec9684116ff1a2`.
- SOURCE/13 release checksums локально и на VPS PASS;108 frozen source
  files совпали с QA042. D не включена, production source не менялся.
- Живая schema1→2; schema1 pre-upgrade/migration copies и schema2 snapshot
  checksum/integrity/private permissions PASS. Existing external-host config,
  old token rows, working Mac secret/client bytes и E-1/T-2 cancelled сохранены.
- TLS readiness/MCP12 tools, scoped synthetic tasks/notes/provenance/stats,
  Host/Origin/auth, cleanup/revoke401, systemd/timer/hardening PASS.
- Exact selected Tandem metadata и все Serve routes до/после полностью
  совпали; Tandem local/HTTPS443200, Funnel OFF. После нагрузки/cleanup
  повторена полная сверка. Доска/claims не использовались.
- Целевой isolated experience20sessions/2000tasks/20requests выполнен полностью,
  400 HTTP200. p50=3463.879мс,p95=10025.109мс,p99=12798.825мс,
  max=17163.349мс. **Latency50мс FAIL≈200.5×**. Seed237.626с,
  measured91.800с. Exit0/success не объявлен performance PASS.
- Sampled peak server RSS35.723MiB≤128 PASS; populated startup28.268мс≤1000
  PASS; empty startup16.805мс. Это наблюдённые thresholds, не полный V1.
- Runtime unit CPUQuota1s/MemoryMax536870912, MainPID87884,
  childPID91493/PPid87884 и тот же cgroup в harness independently bound;
  leaf cpu.max100000/100000,memory.max536870912. Host1logicalCPU,
  MemTotal979160KiB; shared Tandem и observer noise. Ancestor effective limits
  **unavailable**; resource-environment PASS не присвоен.
- Длительный SSH monitor разорвался, in-memory samples потеряны; benchmark
  продолжился под systemd. Persisted short snapshots сохранили identity/limits,
  cumulative OOM counters0 и Tandem200. Непрерывный timeline не заявляется.
- Own transient unit/helpers/temp DB/tokens очищены; private result0600 на
  VPS сохранён. Working DB не заменялась/не откатывалась. Auto rollback после
  миграции не запускался; соответствующий failure guard на live не инъецировался.
- Changed tracked files: `docs/runtime-catalog-049.md`, этот WORK.
  Private helpers/evidence только `.local/evidence/runtime-catalog-049/**`.
  Shared TASK/rootREADME/source, старое evidence, Git этим исполнителем не менялись.
- Отчёт: [runtime-catalog-049](../runtime-catalog-049.md). Полный C source suite
  не повторялся:108 files exact-match QA042; здесь проверена установка/runtime/load.

## Передача

Координатору: принять операционный результат и фактический latency FAIL;
создать отдельную performance WORK с независимым review и тем же повторяемым
20/2000/20 сценарием. Размеры опыта и thresholds не уменьшались. Корень
производительности здесь не диагностировался и source не оптимизировался.
Второй физический ПК/Claude Code/new actual Codex C smoke/reboot/power-loss
NOT-RUN; OAuth/UI/D/E runtime/full V1 не закрыты. UI/design review N/A.

## Журнал

- 2026-10-18T18:57:44+03:00 started: инструкции и immutable release найдены; начинаются baseline audit и проверка hashes. Исполнитель `/root/live049`.

- 2026-10-18T19:03:40+03:00 live_checks_completed: approved immutable C установлен; schema1→2, migration/schema1 copies и post-install/schema2 snapshot integrity/checksum PASS; TLS12MCP/scoped catalog-note-stats PASS, synthetic cleanup/revoke401 PASS. Frozen108 source hashes совпали. Load20/2000/20 выполняется отдельно.

- 2026-10-18T19:08:19+03:00 completed: deploy/live checks PASS; full20/2000/20 experience finished, latency threshold FAIL, RSS/startup observed thresholds PASS. Own unit/helpers/temp credentials cleaned; final Tandem/allServe/TLS443 audit PASS. Handoff координатору для приёмки/отдельного performance WORK.
