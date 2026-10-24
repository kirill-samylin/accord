# WORK238 — независимый APPROVE фактического negative/fix loop F6V11

24.10.2026. Ops validation QA. **APPROVE / PASS** для exact synthetic negative
fix loop на одном Mac с настоящим Codex; `gate_scope=negative_fix_loop`.
Никакого изменения продукта/runtime, recovery, повторов, моделей, CLI приложения/
Git, helpers, HTTP, браузеров, global config или provider credentials не было.

Private0600 report `.local/task014/evidence238/review.private.json` SHA
`a30ea2029731c13d19d42a9d293026aea875d9a0455d3d184368a275534495d0`.
Read index192 bounded files SHA
`2f754e70db5dfdee3026aacb58d30e07a49d892b2d4eced620c934a7c6a75d2f`.
Исходный runtime `pipeline210-f6-1792798761726100000`, actual results SHA
`1ff6815c122b09d54246c5c444bb49bab239b518190f2f0ac23dca05db994dfe`.

## Версия

Все81 source members проверены против manifest
`724a404760d5f936614086168b90007d0ebdd93f8709302c836162c4673ee252`.
Exact source202 receipt
`34cc1fb7046e70838f4d83830c793524d5654b87a97e673be5eecb49bc8afac8`
и root GO связаны с этим runtime/source. Server/QA outer — a32db229…,
worker — be95daed…/SHA9755ed0f…. Старые foundation receipts A209/C216/N218
сохраняют первоначальный a32 scope, а не переименованы в новые worker proofs.

## Фактический цикл

Workflow `wf_ec9c18f094f81ee85534071191038a3b` completed, control_epoch1.
Один materialized plan/одна developer task, native started6/invocations6,
fix_total1/fix_per_task T-3:1; policy limits точно совпали с approved plan.

| Этап | Native UUID | Задача / поколение |
|---|---|---|
| Аналитик | `01a16b22-4041-7a10-a0dc-1ef1b5fdbea3` | T-2 / 1 |
| D1 | `01a16b23-35d4-74f3-8582-8f969f56659c` | T-3 / 1 |
| R1 | `01a16b24-6b1c-79b2-b1f2-e4fd69ca231e` | T-4 / 1 |
| D2 | `01a16b25-b66e-77c1-afb9-7e4a3eabc768` | T-3 / 2 |
| R2 | `01a16b26-d44f-7b43-8e16-2d160ff67edd` | T-5 / 1 |
| QA | `01a16b28-2427-7f82-a922-54f94e3a4e39` | T-6 / 1 |

Все шесть native/board sessions и managed runs различны, attempt_no1,
continuation0. У D1/D2 managed generation1 относится к разным runs; generation1→2
относится к повторно открытой T-3. Эти измерения не смешивались. Каждый start
command applied/не forced, каждый task attempt completed с отдельной сессией.

Первый commit A `49f38bbdbdff721ec1e6149d0a973aaabc10dcbe` действительно содержит
только разрешённую synthetic ошибку `return abs(a) + b`. Отдельный pre-reopen
initial-developer snapshot сохраняет оригинальный честный run_summary с
SYNTHETIC NEGATIVE ATTEMPT/failed/not production success — он не восстановлен
из позднего mutable T-3. Настоящий check завершился exit1, passed1/failed1;
JUnit сохраняет точную ошибку `add(-3, 0)`: `3 != -3`.

R1 независимо завершил review **changes_requested** на A/generation1 с настоящими
findings. Эти findings присутствуют в workflow messages; delivered_messages D2
увеличен относительно D1. Сервер открыл ту же T-3 generation2/returns_count1.
Новый commit B `5506d9d2b5b266776d2a0861ee28972b50259775` имеет parent A и содержит
точно `return a + b`. Новый D2 check — exit0/passed2. Новый R2 **approve** и новый
tester **passed** относятся к точному B/developer generation2; actors независимы
от обеих developer sessions.

Три genuine JUnit archives сверены с worker check reports, SHA-named artifacts,
tuple step/run/task/generation/UUID/epoch/preset и реальным Git manifest:

- D1: `64be131d37e833505a5ddd154b3f9d852efa77752f36b4fbd47d0b00d319a347`, exit1, 1/1.
- D2: `9ce499bfa16969de34bcf559691ade5b5cc43d40af2f3b5d24a4d1490065e6fb`, exit0, 2/0.
- QA: `638141027a6e2cf861620372d5395313c0fb83e979cd80707b712709d836a919`, exit0, 2/0.

Везде два frozen meaningful testcase, errors/skipped0, mandatory
outcome_bytes_truncated — именно wire boolean false. Ошибка A сохранена как
отрицательное доказательство, а не объявлена успешной приёмкой.

## Источник, терминальные данные и cleanup

Offline loose Git objects без Git CLI доказывают exact chain seed→A→B, ровно два
commits, только `src/math_ops.py` в каждом diff и точные разрешённые строки.
Все остальные frozen bytes/modes неизменны, checkout совпадает с B tree,
собственный hooksPath пуст, remotes/replace/shallow/grafts отсутствуют. Historical
A/B manifests совпадают с соответствующими exit/check observations.

Пять закрытых БД прочитаны через immutable/query_only после отсутствия companions;
SHA/identity и отсутствие companions проверены до/после. Workflow/счётчики/план/
messages/grant с точной owning wireProfile редактирующей проекцией совпадают с API.
Все девять attestations совпадают целиком. Epic done, leases/questions/reservations0,
шесть managed runs completed/шесть start commands applied. Success-only policy
revoked, host disabled, четыре profiles archived, legacy token revoked подтверждены
самими закрытыми DB, а не только итоговым report.

Durable journal SHA
`363c647179db7aca533f22dfc68c3748f455536d1a9a34dd1d8f0b0adf0379a5`:
шесть runs/pending0, у каждого native_exited/exit_durable/managed_ack/
lock_released/exit_ack=true, один matching completed receipt, forced/unreportable
отсутствуют. Отдельные числовые native OS exits не архивированы и не выдуманы.

49 generic actual supervisor proofs — proven/success=true, exit0, forced=false;
Next — отдельный expected130/operation_success=false с bound nonce/readiness/
launch/stop intent/source hashes. UI/MCP exit0/signalnull, все requests/streams/
joins/cleanup steps положительные, manualfalse. WORK219 по архивным raw bytes и
TOML подтвердил только126-byte own-workspace trust insertion; другие parsed данные
и companion metadata неизменны. Нынешняя глобальная конфигурация не читалась.

Final UI PNG `cc90fbdd6d2f53f202d5ade01b1c76b7b4965ba49986ae65a9f71c76221310fb`,
160848bytes/1440×1100: визуально «Завершено», 6/6 запусков,6 стартов,1 исправление.
Это viewport, все шесть карточек целиком не видны; полный результат доказан raw
API/БД/журналом/Git/checks, а не выдуман по изображению.

## Ограничения

APPROVE только negative_fix_loop этого Mac/Codex synthetic запуска. Controls/
recovery/H1/B2, Windows/Claude/физический второй ПК, production и установка не
закрываются им. Полные625MiB binaries/libraries заново не читались: exact81-source
manifest и bounded192 raw/source files сверены, recorded approved source-after
guards сохранены явно. Исторические F6V8/V9/V10 и прочие FAIL не переписаны.
Ни formal gate receipt, ни stage/apply GO проверяющий не создавал.
