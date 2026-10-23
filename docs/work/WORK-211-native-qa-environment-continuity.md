# WORK211 — уточнение окружения между QA209A/C/B

23.10.2026, root. CONTRACT APPROVE202 получен на следующее уточнение.
Это не product code, не изменение прав и не runtime GO.

QA209A actual PASS на clean a32db229/e9fcade0. Independent EVIDENCE APPROVE202:
results `5af7cd0dc460ada45923e8ab01e1a1342a39eec00eb431b98cd2ea6b798c1368`,
safe proof `11929b17c35c95850a008682807a688c51856aef8a75acebbb8bbb06e516d83c`,
emitted product permission digest
`49ed13b85c8c280fb9a686df07c549356aff2f25de6ca0f19a6f45e5550c20d9`.
Native executable/Git identity связаны,22 owned calls natural0/!forced/drained,
fixture/config/source/binary pins unchanged, model0. Это configured grants
proof; OS enforcement, native commit и pipeline ещё не проверены.

A использовал одно in-memory окружение для всех calls, но не сохранял private
env bytes/hash. Поэтому byte-exact equality исторического A с будущими C/B
**не доказана**. Не восстанавливать её предположением или текущим os.environ.

## Согласованное продолжение

1. C создаёт новый private0600 canonical environment snapshot/hash до child
   calls. Raw env может содержать secrets: только private HOME, без values в
   stdout/report/Git. Native calls C используют именно сохранённый snapshot.
2. В том же сохранённом A fixture/spec, с прежними exact candidate/native/Git
   pins выполняется **один дополнительный normal product permission proof**.
   Его actual digest/executable/Git identity обязаны совпасть с A. Никакого
   диагностического broker, реконструкции digest из flags или fallback.
3. После PASS — один direct OS probe из approved PLAN209. C остаётся models0
   и90s+30s; два отдельных native calls сохраняют полные20s+7s reserves.
   Не запускать очередной child при недостатке полного call budget.
4. Нового MCP discovery нет: exact A disabled-map overrides сохраняются;
   product effective-map validation заблокирует unknown/enabled connector drift.
   User config/source/binary/fixture before-after checks остаются обязательны.
5. B читает **точно frozen C env file** (canonical bytes/hash, ownership/private
   file/source pins before-after), а не заново текущий environment. B по плану
   делает свой normal preinspection и требует equality A=C=B permission tuple,
   затем максимум ONE model turn/180s. Остальные B240+30 bounds неизменны.
6. C failure/unknown/drift блокирует B; no hidden retries/более широкие права.
   Все raw snapshots/proofs/executed sources сохраняются private. Повторного
   A env equality claim нет. Superseded A/206/207 evidence не переписывается.

Separate QA209 author получает source-only GO для C после этого contract approval;
нужны independent SOURCE review точных files/pins и отдельный root GO на ONE C.
B executable sources также отдельно review до своего bounded root GO. Source203
и production/GUI/Test не входят в область этой поправки. Полная приёмка209
требует A+C+B; A PASS не разрешает преждевременно запускать pipeline203.
