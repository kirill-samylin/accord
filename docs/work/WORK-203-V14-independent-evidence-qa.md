# WORK203 V14 — независимая проверка фактического полного цикла

24.10.2026. Ops validation QA. **APPROVE / PASS** для отдельного synthetic
запуска на Mac с настоящим Codex. Исходный результат, runtime, БД и настройки
не менялись. Новых агентов, моделей, CLI приложения/Git, helpers, HTTP, браузеров
или повторов проверки не запускалось.

Private report `.local/task014/evidence203-v14/review.json`:
`9a29b227867a8b9cdbc5a29d5348be22585af7508a06666cee0ce3a20433f4d8`.
Индекс 143 ограниченных чтений `read-index.json`:
`7a7ce80092d5a2689e1de9ddffccbbee1424b3514a3b720c2e76052709f0feb6`.
Отчёты private0600, каталог0700, секретов в проекциях нет.

## Версия и разрешённый запуск

Runtime `pipeline203-1792796395902673000`; actual results SHA
`ebce126f892d9d6165ea56b735a824620243a37dca2abcb91734f28d96b5261a`.
Source manifest `19436f9bf1df8951db5d5ccb5dea9f06136442bb8a852688c61b07a9e8a4c08d`
целиком перепроверен. Exact source202 receipt, root GO и его input hash связаны
с этим manifest/runtime и separate worker revision/SHA.

Server и QA outer supervisor — `a32db22973d5b43d4fb227a89337bf5b82a4969f`.
Worker — `be95daed7153f4abeb72012286cc748d0596b479`, SHA
`9755ed0fb0de869751635ba4e5adf1a4d742b0a202e6659e871d337fceed27ff`.
Исторические foundation receipts сохраняют исходную a32-версию; они не
переименованы в доказательства нового worker. Его build NOT_RUN_AT_BUILD также
не переписан: ниже приведены отдельные факты нового runtime.

## Фактический результат

Workflow `wf_935fd898c7a6a39c993b896ff3307d5c` completed: один materialized plan,
одна задача разработки, четыре подтверждённых запуска, fix0/continuation0.
Все native UUID, managed runs и board sessions различны; pinned profile,
step/task/run generation, epoch1 и qualified evidence сверены.

| Роль | Native UUID | Managed run |
|---|---|---|
| Аналитик | `01a16afe-b4da-7a41-ae64-96972b9865f8` | `mr_97f37c9903fb916367ff17f7ea48ce5e` |
| Разработчик | `01a16aff-c1bb-7ba1-aa81-d18c94b8818a` | `mr_906d2be9300c59b422cc15ff9fb725e6` |
| Ревьюер | `01a16b00-ef34-79e3-9337-43dbfd508beb` | `mr_1038d20b008a70f19de57f58ccde7377` |
| Тестировщик | `01a16b02-1041-7181-ae4c-825bf0e10ef3` | `mr_3961b713de4ddb2361f98b6ed3c0066c` |

Настоящий developer commit `ad76196d9e0f7644826c23cd5102614f52860a41` имеет ровно
одного parent seed и меняет только `src/math_ops.py`. Loose Git objects, HEAD,
полный checkout tree, неизменные frozen bytes/modes, пустой собственный hooks
каталог и отсутствие remotes/replace/shallow/grafts проверены без Git CLI.
Manifest `9d8bbc529805148cf94e118b81655b779aea6db6a0207a0f48370308202c5dc8`
совпадает с D/R/T exit observations и обоими checks. Reviewer `approve`, tester
`passed` относятся к этому SHA и developer generation1; их actors независимы
от разработчика.

Два настоящих `python-unit` check: каждый exit0, passed2/failed0/skipped0.
Их отдельные JUnit bytes/корневые счётчики и оба meaningful testcase сверены:

- Developer: `805faccf4c5d4cc658bde28ff80b5ba542ed39db4ace5cf9f59142a93e3b28a8`.
- Tester: `d9ec8feb8a06cf7bf0e564f16df06bd922e28556c81bf3371e4fc53734bb2aca`.

Actual UI intake и real stdio MCP replay дают один workflow с тем же intake
hash/text. В закрытой БД один `workflow_submitted`; он связан с настоящей browser
human session через web action/session/browser backlink. Scripted coordinator
MCP и scripted API human — отдельные actors. Web-relative stored redirect и
project-prefixed UI redirect сверены в их собственных пространствах маршрутов.

## Терминальное состояние и очистка

Raw worker journal SHA
`90aadb2d50ca2e50c5393bd54f6f162854719fd105ff9aabe551199179c634cc`, pending0.
У всех четырёх native_exited/exit_durable/managed_ack/lock_released/exit_ack=true,
один completed receipt с правильным UUID, без forced/unreportable. Saved
`journal.private.json` сериализован иначе, SHA `5e0ef661…`; его parsed object
полностью совпадает с durable state. Хеши разных сериализаций не подменялись.
Отдельные числовые native OS exit codes не архивированы; они не выдуманы.

Пять закрытых DB читались только после отсутствия всех companions, через
immutable/query_only. SHA/identity и отсутствие companions проверены до/после.
Durable workflow/API поля совпадают, включая grant после точной owning
`wireProfile` редактуры instructions/description; все шесть attestations совпадают
целиком. Epic done, leases/questions/reservations0, все четыре managed runs
completed, четыре start commands applied. После успешного цикла подтверждены
policy revoked, host disabled, четыре profiles archived и legacy token revoked.

35 generic supervisor proofs: proven/success=true, exit0, forced=false. Отдельный
Next stop — ожидаемый130, operation_success=false; nonce, launch binding,
readiness200, durable stop intent и source hashes связаны. Approved runtime
closure guards подтверждены recorded result; UI/MCP actual exit0/signalnull,
requests/streams fulfilled/drained, все steps/joins положительные, manualfalse.
Внешнее завершение процессов проверялось отдельно от native ACK/data terminal.

WORK219 независимо проверен по архивным before/after bytes и TOML: только 123-byte
own-workspace trusted insertion; другие parsed данные и companion metadata
requirements/managed_config неизменны. Классификация `QA_TRUST_ADDITION_ALLOWED`.
Текущие global config/provider credentials не читались.

Final UI PNG `163cc1c9e01e56ecdcda454918eaa0bee0743985cd88514f9c6c3967ad9beab4`,
144905 bytes, 1440×1100: визуально «Завершено», 4/4 запуска и 4 подтверждённых старта.
Это viewport: нижняя карточка тестировщика не помещается целиком; её результат
подтверждён actual API/БД/attestation/JUnit, а не выдуман по изображению.

## Границы APPROVE

Только один Mac, Codex и synthetic fixture. Windows/Claude/второй физический ПК,
production rollout и остальные негативные/control сценарии этим PASS не закрыты.
Полные 625 MiB binaries/libraries повторно не перечитывались: новый exact source
manifest перепроверен, approved actual source-after guards сохранены явно.
Исторические 203V11/V13, F6V8 и C2V10 FAIL остаются неизменными. Независимая
проверка исходного кода в этой карточке не заменяет отдельный SOURCE_APPROVE202.
