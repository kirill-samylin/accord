# WORK-053 — performance analysis и implementation contract

- Исполнитель/instance: `/root/perfarch053`, архитектор, свежий агент.
- Родитель TASK-003; координатор единственный редактор общей карточки.
- Статус: исследование завершено, контракт на приёмке координатора.
- Версия C: `1e9025cd6e3122f0adf6165e09845202736d004a`; source не менялся.
- Область записи: этот WORK, `docs/performance-contract-053.md`, ignored
  `.local/evidence/performance-053/**`; чужие checkouts/WORK не менялись.
- Прочитаны repo AGENTS, requirements, architecture, service/security/config
  contracts, runtime049/WORK049, acceptance042, D store/config snapshot path;
  профиль архитектора и общий процесс/parallel-work Давинчи прочитаны read-only.
- Tandem OFF по пользовательскому указанию repo. Named Claude models здесь
  не закреплены; runtime профиля не предполагается. Новые агенты не создавались.
- UI/design review N/A. Выпуск/ВPS/deploy/Git/отправка сообщений вне внутренних
  collaboration tools не выполнялись. Production tests не добавлялись.

## Результат

[Контракт](../performance-contract-053.md) содержит доказанный bottleneck,
cache/connection/candidate ownership, dirty persistence, schema4 projection,
регрессии и полный порядок реализации/повторной приёмки.

1. Реальный C Store benchmark SQL fixture2000tasks/20sessions/2041events,
   GOMAXPROCS1: get≈24.6ms, tasks≈25.6ms, events≈29.2ms, stats≈29.6ms,
   heartbeat≈32.3ms. saveBoard≈17.0ms, loadBoard≈5.51ms, facts/enrich≈1.16ms,
   whole-board JSON clone≈6.26ms. Профиль подтверждает repeated SQL prepare.
2. Real C annual scan private fixture200000events: filtered events522.6ms,
   stats514.2ms,≈219–222MB allocation/operation. Это не RSS и не runtime proof.
3. D full config-template Parse≈5.69ms/3.92MB, Decode≈80µs: повтор Parse на
   каждой D transaction требует version/hash-keyed immutable reuse. D runtime
   не профилировался и сюда не интегрировался.
4. Pinned connection data_version modernc probe PASS: owncommit2→2,
   external commit2→3 после writerlock. `_txlock=immediate` поддержка проверена
   по pinned primary source; отдельный probe лог — txlock.log.
   Process/error/reconnect случаи NOT-RUN.
5. Private cost-only sketches: cached narrow operations45–376µs;
   indexed200k event query≈32µs с EXPLAIN index SEARCH;
  20owners×2 SQL prefix seeks≈388–389µs. Эти sketches намеренно неполные,
   не production implementation и не семантический/latency PASS.

Главная причина — full-board load/save + JSON isolation в serial transaction,
а не отдельно Stats или proxy. Delta UPSERT один не достигает target;
annual scans также требуют projection/indexes. Это вывод из source и локального
профиля, не утверждение измеренного VPS CPU распределения.

## Передача и обязательные остатки

- Root синхронизирует schema contracts: D3, performance4, E5. Координатор
  разрешил такое предложение внутренним сообщением; сам contract sync здесь
  не выполнен и требуется до автора migrations. Единственный migration owner.
- D052 defects/review/QA сохраняются независимо; performance начинается
  от frozen combined source с полным read/config/auth snapshot contract D.
- Warm read/bounded heartbeat — промежуточный этап. Финальный fix включает
  обычные mutations/COW/dirty persistence/affected graph, не benchmark-only tuning.
- Новая schema4: verified prebackup, atomic streaming backfill, exact prefixes,
  event indexes, dirty/cursor guards и rollback/restore/reset. Chronology fallback
  сохраняет исходную семантику; clock regressions не считаются fast-path PASS.
- Independent code/security review затем tester acceptance обязательны;
  автор этого исследования не выдаёт независимый approve будущему коду.
- VPS049 FAIL≈10025ms сохраняется. Повтор20/2000/20 и annual200k, ordinary API,
  target resource limits, TLS/effective limits остаются NOT-RUN для будущего fix.
- Admission100rps/burst200/max64 не ослабляется. После ускорения old unpaced
  harness может получить429; exact rerun и отдельно reviewed paced method
  публикуются раздельно, запрещено скрыто менять acceptance/fixture.

## Фактические проверки и воспроизведение

Локальный Go binary `.local/toolchains/go/bin/go` сообщил Go1.27.2 darwin/arm64;
заданный в поручении путь с basename go1.27.2 отсутствовал, использован этот
проверенный pinned runtime. GOTOOLCHAIN=local; build/mod/tmp caches — текущего
checkout, как Makefile. Dependencies не обновлялись. GOMAXPROCS=1 ограничивает
Go parallelism; это не эквивалент quota1vCPU/512MiB или VPS/TLS окружению.

Go overlays добавляли только private *_test.go к main Store/config packages.
Команды/аргументы записаны в `run-metadata.json`; baseline2 repeats500ms,
annual1x, narrow/projection2×500ms, CPU profile3s. Все диагностические запуски
завершились PASS в смысле исполнения benchmark/probe. Acceptance PASS не присвоен.
Полный make check не повторялся: production source/tests не менялись.

Private evidence содержит fixture/overlays, baseline/narrow/config/annual/
projection logs, matching final CPU profile+binary и pprof summaries,
source/dependency manifests и evidence SHA256. Синтетические данные только,
файлы0600/каталог0700; raw049 не переписывался. Исторический первый profile
и его text summary оставлены отдельно; binary для него был перезаписан новым
benchmark build, поэтому воспроизводимый matching pair — `cpu-final.pprof`
и `profile-store.test`, а не старый cpu.pprof и store.test.

## Журнал

- 2026-10-18T19:20:46+03:00 created: карточка WORK записана после исследования;
  ранний точный started timestamp не фиксировался и не выдумывается.
  Исполнитель `/root/perfarch053`; статус исследование→на приёмке.

- 2026-10-18T19:24:03+03:00 completed: контракт/профили переданы координатору;64 production Go/mod files
  совпали с HEAD, private probe/benchmarks завершились, собственные документы
  прошли whitespace check. Independent review и исправление runtime не выполнены.
  Состояние: на приёмке; следующий владелец — координатор и новый исполнитель.
