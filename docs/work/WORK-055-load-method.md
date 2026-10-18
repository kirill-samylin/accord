# WORK-055 — метод offered-rate catalog load

Родитель: TASK-003. Исполнитель: `/root/loadmethod055`, developer/unit tester.
Статус: выполнено автором; независимая проверка ожидается.
Этап: реализация метода, не оптимизация production.
Вход: main `ad41f6f`; approved C `1e9025cd6e3122f0adf6165e09845202736d004a`,
APPROVE048 и исторический опыт049. Tandem OFF по прямому поручению.
Профили developer/tester, общий процесс/parallel-work прочитаны; закреплённые
модели Claude в этой среде не действуют. Независимый approve себе не выдаётся.

## Границы

Только `tests/acceptance/catalog_load.py`, необходимые `catalog_*.py` helpers,
`tests/infra/test_catalog_load_pacing.py`, `docs/load-method-055.md`, эта WORK,
ignored `.local/evidence/load-method-055/`. Production/Go, чужие WORK/TASK,
Git-запись, VPS/live и sibling запрещены. Approved cgroup helper не изменяется.

Сохраняются legacy unpaced режим, default20 sessions/2000 tasks/400 measured,
equal mix, target p95≤50ms/RSS≤128MiB/startup≤1s и admission100/burst200/64.
Opt-in offered-rate не маскирует429; любой не200/ошибка даёт FAIL без retry.
Годовой200k fixture и дополнительный API mix вне этого среза.

## Журнал

- 2026-10-18T19:19:49+03:00 — started; входы и область сверены.
- 2026-10-18T19:25:38+03:00 — completed; unit31 PASS, tiny C smoke PASS метода;
  передача независимому reviewer, target/performance PASS не заявлен.

## Результат и решения

Добавлен opt-in offered-rate1..100;0/отсутствие сохраняет unpaced сценарий.
Дозируются describe/seed/warmup, measured waves сохраняют20 concurrent-ready
workers и400 samples. Monotonic/no catch-up, barriers с deadline и abort/wakeup;
429/любая ошибка дают FAIL без retry/исключения slow samples. Native policy и
thresholds не изменены. Explicit persisted-defaults решает TTL argv mismatch,
без capability guessing/unsafeDB/config patch. D source читался, D не запускался.

Декомпозиция: `catalog_load.py` orchestration/report/CLI187 строк;
`catalog_runtime.py` HTTP/owned child125; `catalog_pacing.py` pacing/accounting118;
`catalog_workload.py` equal-mix threads55; unit suite288. Это границы разных
ответственностей, смысловые дубли не добавлялись; cgroup helper не менялся.
Подробный контракт, команды и caveats: `docs/load-method-055.md`.

## Проверки и evidence

- PASS31: `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/infra -p 'test_catalog_load*.py' -v`;
  17 новых synthetic cases +14 существующих cgroup; final unit.log0.135s.
- PASS tiny method smoke approved C:2 sessions/10 tasks/10 requests each,
  offered100/persisted-defaults;39 attempts/39 responses HTTP200;
 20 measured,4×5. Final Mac arm64 elapsed0.263243s/75.976rps;
  p954.575ms, RSS24.938MiB, populated17.091ms. Это не target performance PASS.
- `git diff --check` PASS. Final snapshot находится в ignored evidence:
  `freeze.sha256` фиксирует7 файлов включая эту WORK; `method.diff` включает
  tracked delta и все новые scoped files; `manifest.json` фиксирует base,
  binary/evidence/diff hashes. Self hash manifest в него не включается.
- Целевой Linux/VPS20/2000, D process, annual200k events, дополнительные
  mutations, Go/full suite, live/deployment, TLS, второй физический ПК NOT-RUN.
  Полного API/performance/V1 PASS нет. Новых product-process/E2E нет.

## Передача

Статус реализации: выполнено, передано на независимое review; общая приёмка
не завершена. Следующий исполнитель — отдельный reviewer exact freeze055.
После approve — координатор согласует product source и target run с053/054.
Модель как у автора должна оставаться у reviewer; named Claude binding здесь
недоступен. Ни Git/push, ни VPS/live, ни чужие TASK/WORK/sibling не изменялись.
