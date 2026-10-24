# WORK203 V13 — независимое actual evidence review

**Final verdict: FAIL / QA_HARNESS_LIST_LIMIT_CONTRACT_MISMATCH.** Actual business
completion подтверждён закрытыми данными, но исходный harness FAIL не заменяется
PASS и downstream runtime gate остаётся закрыт. Подробности после подготовки.

## Историческая подготовка

24.10.2026, ops_validation_qa, отдельный от автора harness/product reviewer.
Статус подготовки: **NOT-RUN; финальное закрытие ещё не подтверждено root**.
Runtime `pipeline203-1792794432339921000` не читался и не перечислялся. Чтение
итоговых files/journals/closed DB и final verdict — только после root sessionexit.

Прочитан frozen203V13 source contract: manifest
`2ce92a5c4a6c2567664ee14148adff03b4b9460e5015436be8dde62286005fc5`,
source202 receipt6647b5b4…. Assertions byte-exact approvedV11; добавлены явные
dual-version provenance guards. Server/QA outer a32/e9fc, workerbe95/9755,
worker provenanceV3d98/source20280bf и internal self-exec нового worker должны
быть явно различены; старые foundations не объявляются тестом нового worker.

После закрытия планируется только bounded archived evidence inspection и own
closed DB read-only без WAL omission/aux writes. Никаких CLI/Git/product/helper
imports/native/browser/model/HTTP/global native settings. Offline loose Git
objects допустимы как чтение собственного synthetic checkout; snapshots только
уже сохранённые private. Исторические FAIL и actual runtime не изменяются.

Обязательные независимо сверяемые факты:4ordered UUID/run/board sessions и
канонические managed outcomes/run_exit/ACK,1plan/1task/1developer commit, тот же
target SHA/generation у независимого reviewer approve и tester passed,2checks с
реальными report hashes/JUnit2tests/zero failures-errors-skips, zero reservations/
leases/questions/pending, actual UI+realstdio same-action replay, cleanup token/
policy/host/profile readback, strict owned/browser/MCP closure и alljoins, exact
Next-known130 не operation success, archived shared219 strict addition либо
unchanged, source/tool/Next/copy preservation и completed UI PNG metadata.
Числовые native OS exit codes нельзя выдумывать из managed phase; они будут
обозначены отдельно как observed либо not separately archived. Source-approved
guards переиспользуются как контракт, helper functions не исполняются заново.

Private gate-ready evidence report и read-index будут созданы в fresh
`.local/task014/evidence203-v13/` только после разрешённого закрытого чтения.
Они свяжут final results SHA, source/worker/receipts и relevant raw hashes;
independent PASS/FAIL не заменяет исходный harness result. Scope ограничен одним
synthetic Mac/Codex; Windows/Claude/физический второй ПК/GUI/production NOT-RUN.

## Независимая readonly диагностика после закрытия

Root подтвердил session93672 exited1. Final results SHA
`430ac29c0896fa35f7d1f81cbdb40cb8d106a02191499570845f5ec0dd100b14`
verified: FAIL/terminal_provenance/AssertionError. Source51files, source2026647,
GO/input/worker bindings verified. Closed own project DB без WAL/SHM/journal,
opened mode=ro&immutable=1, SHA до/после
`08fb9e49069ea860200cc7d0370f14e1e71cea3ae1c4134703c26315ca23c78e`.
Actual runtime/source/snapshots не менялись; дополнительных CLI/Git/helper/
module/native/model/config/browser/HTTP действий не было.

**Причина:** complete() assertions.cjs:16–26 совпадают с observation68, включая
plan.prerequisites=[]; terminal-tasks/runs ещё не сохранены. Следующий вызов
assertions.cjs:27 → api.cjs:43 `/automation/list` передаёт limit100. Actual
Store.WorkflowList в internal/store/automation_list.go:35 допускает максимум50;
поэтому QA API.request assertion api.cjs:19 требует невозможный200. Это mismatch
QA caller с существующим list contract; report pagination100 — отдельный API и
здесь не меняется. Failed HTTP status/body не архивированы, фактический packet
не выдумывается; источник классификации — deterministic frozen call/guard и
порядок сохранённых evidence. Пустой diagnostic stack объясним: api/assertions
не входят в allowlisted owning frames, не является доказательством отсутствия сбоя.

До свежего expensive trial проверены все последующие data predicates вручную,
без исполнения original assertion helper:

- Workflow completed/version25/epoch1, native4/started4,1materialized plan/1task,
  questions/reservations/leases/pending0. Epic и4role tasks done.
- Четыре разные UUID/run/board sessions; каждый managed completed, один applied
  start command/no forced, все5terminal ACK flags true, completed receipts exact,
  task generation/profile snapshot/run_exit/source tuples совпадают. Отдельные
  numeric native OS exit codes не архивированы и не объявляются нулями.
- Reviewer approve, tester passed; target commit
  `88f9210468dd3da55ea7abc127ea93ac667993fb`, target generation1 и evidence verified
  SHA совпадают; board actors distinct от developer, findings имеются. Все4start
  commands applied, delivery_unknown здесь отсутствует.
- Offline loose Git objects доказывают один commit поверх seed, только
  src/math_ops.py, current source matches HEAD; остальные frozen bytes/modes
  сохранены. Manifest9d8bbc52… совпадает с postdeveloper exit/check observations.
  Developer permission proof связан с actual .git dev/inode, не старым baseline.
- Два check attestations developer/tester с точными role/run/gen/task/UUID/epoch,
  preset1/digestcf16b09d…, target SHA, exit0/two tests/fail0/skip0. Оба actual
  archived reports проходят даже original frozen JUnit regex/attrs/end guard;
  hashes e14b34a1… и d00e5bfe…. Final JUnit соответствует tester check.
- Последующие API DTO expectations проверены source: output_truncated и
  commands_truncated — nonomitempty bool, не missingfalse. Read-only DB facts
  соответствуют остальным последующим terminal predicates. Реальные terminal
  API calls и final UI assertions не происходили из-за более раннего FAIL.

Actual UI+realstdio MCP same-action replay подтверждены cached request/одним
workflow event, canonical coordinator session и тем же intake hash. Cleanup
complete/manualfalse:23generic supervisor typed0/success/!forced, Next known130
successfalse с bound live intent/readiness/source/nonce, browser/MCP actual0,
close FULFILLED, alljoins/diagnostic pending0. Archived config raw+metadata
подтверждают exact123byte own-workspace trust addition и unchanged прочие данные,
QA_TRUST_ADDITION_ALLOWED; live config не читался. Context after-pins сообщает
source/tool/Next/worker original+copy/provenance unchanged; строгие approved
guard facts переиспользованы, helpers повторно не запускались. Раздельные server/
outera32 и workerbe95/9755/provV3d98/source20280bf сохранены в private report.

Final completed UI PNG и post-success credential/policy/host/profile revoke
ветка **NOT-RUN**, поскольку harness уже failed. Сохранён failure viewport PNG,
1440×1100/135345bytes, SHA15c073fc…; он не переименован в final UI assertion PASS.
Полный625MiB preservation reread здесь не повторялся: root запросил конечный
critical checkpoint и освобождение слота, source51files/raw hashes привязаны.

Fresh private artifacts0600:

- `.local/task014/evidence203-v13/review.json` SHA
  `71ac00d38dbb4f508bfc626978897ef30e8b59ca9bf985b4d73d6fd8af436b87`.
- `read-index.json` SHA
  `c7c1fc4a6f2ea117707fd456741eed73f43880800e4385fa96aa3ecbfad17677`
  (120bounded source/archive files).

Минимальная следующая QA correction: list-only limit50, сохранить report100 и
все business/tuple predicates; finite route/status и owning api/assertions frame
помогут диагностике без raw body/auth. Автор этого review ничего не исправлял и
прогон не повторял. Overall FAIL/gatefalse сохранены; independent business facts
не дают основания ретроспективно ставить runtime PASS. Один synthetic Mac/Codex;
Windows/Claude/второй физический ПК/GUI/production и прочие сценарии NOT-RUN.
