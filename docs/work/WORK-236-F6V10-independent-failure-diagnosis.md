# WORK236 — точная причина FAIL F6V10

24.10.2026. Ops validation QA, bounded read-only inspection. **FAIL сохранён**,
`gate_pass=false`; новых процессов/native/моделей, helpers, CLI приложения/Git,
браузеров, HTTP, recovery, исправлений или повторов не выполнялось.

Private0600 report `.local/task014/evidence236/review.private.json`:
`fa579acfb8435a95676e8c62dbc4ce4fc7f19fdb073370dfca34de5a6bda4cf3`.
Read index12 файлов: `d330cfb08389831f314d767c864c97ddb63e2e1937af15351da87658ff3749c3`.
В `own-native-projected-packets.private.json` сохранены только own call IDs/
code hashes, точные ошибки и bounded synthetic final reply; секретов нет.

## Подтверждённая цепочка

Actual runtime `pipeline210-f6-1792797958472115000`, results SHA
`8d0a07bee8b06e4db3f912898b0c71b29cb6520f21bbfa0829bd5fda4ada6bf6`.
Analyst, D1 и reviewer1 завершились; reviewer1 действительно вынес
`changes_requested` на отрицательный commit
`697102aae024a0b47d7d66cd6493f8d96a67a1db`, target generation1. D1 check настоящий:
exit1, passed1/failed1. Сервер разрешил одну fix-попытку того же T-3, task
generation2, новый managed run generation1. Эти два поколения относятся к разным
объектам и **не обязаны совпадать**.

Own D2 UUID `01a16b19-c9c9-7dc0-a53b-7357ba3a264b`; единственный exact native log
214888bytes, SHA `c58471fabacd1f64c1cd760174180b99a420676ccb532bafa1d0c4057a8d1f19`.
session_meta ID и cwd совпали с этим QA checkout; другие sessions не читались.
Сохранённый native code рассматривался только как текст, не исполнялся.

1. D2 сделал describe и next именно назначенной T-3.
2. Затем в batch ошибочно запросил `accord_managed.get` **T-4**, задачу ревьюера.
   Actual response: `FORBIDDEN_ROLE`, «Разрешена только назначенная workflow
   задача». Это ожидаемая защита `automation_capability.go:179–188`: разрешены
   назначенная задача и, для собственного review, её target; произвольная T-4
   сюда не относится.
3. Native соблюл stop-on-refusal: исходник не изменял, новый commit/check/transition
   done не создавал. Genuine final reply прямо признаёт ошибочный запрос T-4.
   HEAD остался отрицательным A; exit observations source_clean=true.
4. Попытка отправить `workflow_question` с task_generation2 и workflow version1
   получила actual `CONFLICT`, «Workflow изменился». Это workflow CAS
   (`automation_questions.go:15–16`), не ошибка пары managed/task generations.
   Вопрос в БД не создан.
5. D2 native turn завершился, но T-3 не был завершён. Gate
   `automation_advance.go:230` возвращает evidence_missing: «Native turn не
   завершил назначенную задачу». Последний observation содержит workflow blocked,
   reason EVIDENCE_MISSING. Это объясняет FAIL, а не свидетельствует об успешном
   исправлении или ложной ошибке проверки.

Reviewer2 и tester не достигнуты. Полный отрицательный fix loop не закрыт.
Ослабление ACL/CAS не требуется и не предлагалось. Необходимость изменения
prompt/передачи reviewer findings требует отдельного авторского source review;
эта диагностика не объявляет отсутствие доступного контекста продуктовым багом.

## Завершение и границы

Во всех четырёх durable journal runs native_exited/exit_durable/managed_ack/
lock_released/exit_ack=true, pending0. Managed completed здесь означает завершение
процесса, а не успешную разработку. Отдельные числовые native OS exit codes не
архивированы и не выдуманы.

После штатного failure cancel закрытая project DB: workflow cancelled v23/epoch2,
reserved_runs/reservations/questions/leases0; четыре managed runs completed,
четыре start commands applied, T-3 cancelled generation2. Companion files отсутствуют
до/после immutable/query_only чтения; SHA/identity сохранены. Это отличается от
незавершённой terminal cleanup прежнего F6V8, который остаётся отдельным FAIL.

Original outer cleanup.complete=true/manualfalse и все recorded drains positive,
UI/MCP exit0. В этой узкой диагностике raw lifecycle proofs всех generic children
и архивный WORK219 повторно не аудировались; recorded config classification
сохранена отдельно. Текущая global config/providers/tokens не читались, никаких
секретных колонок БД не запрашивалось. Scope — synthetic Mac/Codex; production,
Windows/Claude/второй физический ПК не проверялись.
