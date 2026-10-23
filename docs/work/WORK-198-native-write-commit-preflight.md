# WORK198 — native Codex write/local commit permission preflight

23.10.2026. Отдельный QA; root координатор, reviewer169/189. Root разрешил
authoring после scoped193 SOURCE APPROVE97; runtime **NOT-RUN** до отдельного
review этого harness и exact root GO. Никакого pipeline/production/GUI запуска.

## Цель и frozen candidate

Проверить установленный реальный Codex: одна правка synthetic `message.txt`
и один локальный Git commit с обычными developer193 permissions. Это узкий
preflight, не автоматический workflow и не worker enrollment.

Clean candidate3e6ac81ee81909e3c33a2c86e593f4781357878d:
`.local/task014/releases/3e6ac81ee81909e3c33a2c86e593f4781357878d/mac/agent-board`,
SHA7207d53679dbc0e91132ec3bc206f70e860516d3cc818aab093a2ce023d84dae.
Runtime binary будет exclusive0700/fsync скопирован в fresh private HOME/bin
и проверен SHA до/после, без запуска свежего supervisor из Desktop. Нет warmup,
изменений установленного GUI worker, xattr/privacy/auth/global native settings.

Native argv соответствует providerArgs193 developer: sandbox_mode=workspace-write,
exec/--json/stdin prompt. Нет model override, approval-policy override, full access,
ignore-user-config или skip-repository flags. Native config/auth остаются обычными
пользовательскими; effective CODEX_HOME config hash проверяется до/после.
[Official OpenAI Sandbox docs](https://learn.chatgpt.com/docs/sandboxing)
подтверждают, что sandbox распространяется и на запущенные Git-команды;
разрешение actual commit проверяется runtime, не выводится из успешной записи файла.

## MCP и owned lifecycle

До модели выполняются два config-only owned5s вызова: discovery mcp list JSON и
inspection с точными ephemeral overrides. Все discovered names, включая уже
существующий accord_managed, выключаются; новый managed server не создаётся.
Использован approved193 transport-complete whole-TOML-map pattern с quoted keys
внутри table: disabled stdio/inert false и HTTP/loopback placeholder. Dotted quoted
CLI paths и пустой map как замена disable inventory не используются.
Имена/env/original URLs/argv не публикуются; raw inventory только private.
Любой enabled/extra/missing connector либо config parse/ownership failure запрещает
native model invocation. С Tandem доской/другими MCP сервисами не взаимодействует.

Actual candidate `worker supervise` — только private owned-exit wrapper прямого
Codex exec, без сервера/MCP bridge/pipeline. Отдельный stdin lifeline, nonce-bound
positive exit proof, actual native exit code и drained output обязательны.
Nonblocking handshake и bounded180s native turn не допускают зависания до main.
EOF инициирует существующий owned cleanup; numeric PID/group signals из Python
не используются. Unknown exit/manual-inspection остаётся FAIL без второго attempt.
Unix PGID не объявляется arbitrary detached-tree containment.

## Fixture и предметный результат

Fresh HOME root0700, только synthetic checkout/message.txt, local Git identity,
empty owned template/hooks, no remotes, fsmonitor/untrackedcache disabled только
в local fixture config. Parent создаёт один исходный commit BEFORE native start.
Native должен заменить точные bytes `QA198 before\n` → `QA198 after\n`, сделать
ровно один child commit и сообщить actual HEAD marker. После native только
read-only verification: parent/commit count, exact tree/path/mode/bytes, clean
worktree/index, refs/config/hooks/remotes unchanged. Parent не коммитит результат
за модель и не исправляет partial write/permission refusal.

PASS требует actual native UUID/turn.completed, отсутствия error/MCP calls,
положительного owned exit0 и независимо проверенного commit. CLI exit0/текст
агента сами по себе не PASS. Private bounded4MiB provider log/1MiB line cap,
readonly discovery1MiB cap, no raw credential/provider output в stdout/report.
На FAIL сохраняются безопасные стадии и partial-write/commit observations только
после proven exit; retry/расширение permissions не выполняются.

## Source-ready

Private `.local/task014/qa198/` (runtime ещё не создан):

- run.py145 строк — `bbb0a77cc8ab0e813f5a09457c5e74e8d6765a7ba14363011323ac2112d7ff43`;
- fixture.py61 строк — `4ea01a1c28c03111c4430e14d2eef9f92d21c718d7219a1dd08e0321ee096068`;
- native_stream.py83 строки — `26dfb31c454c717d6735d1677a89e434643c913cf2270239228f17c5fa755fbb`;
- owned_call.py91 строка — `b970e314886c3135f174ca6d1d95fb2057d045293d795ee80850790c7386b837`;
- mcp_isolation.py35 строк — `8b04cda122fdfdca271a9345844d074e8779be7a8f8985bacc65c1f50a03cdfe`;
- source-pins.json — `eaebde9f368c40f7f6b63cdbcf6fc4fb14a21af1842a6c922ba249a48f0c248c`.

Manifest содержит97 exact193 files fingerprint78c3ac53629e191653aa773b9f370856c20923c8470ecdab7491efb2ea96a468,
дополнительные provider/process/discovery source pins, native/git executable hashes
и сам harness. Current source pins и AST PASS; fixture/Git/MCP/native/model
**NOT-RUN**. WORK197 role plan исправлен: независимость через different native
UUID/RunID/SessionID/contributor provenance, один owner principal допустим;
два site principals — отдельная intake/control проверка.

Foundation195/production196 complete сохраняются; Windows/Claude/physicalsecondPC,
полный pipeline/negativefix/UI/Linuxrestore остаются отдельной QA.

## Первый actual harness run — FAIL до Git/native/MCP

169 SOURCE APPROVE и отдельный exact root GO получены на указанные fingerprints.
Единственный запуск остановился на fixture_setup: проверка всех inherited GIT_*
variables отказала из-за **GIT_PAGER**. Значение не выводилось. Это слишком
широкий test-only environment guard; native permission defect не доказан.

`native_attempts=0`, owned_calls пуст, Git/MCP/config/model invocations0.
Созданы только пустые owned checkout/empty-hooks directories в fresh HOME root;
native/commit/write результат **NOT-RUN**. Input pins и effective native user
config unchanged; no ownership uncertainty, поскольку child не создавался.
Код теста после failure не менялся, автоматического retry нет.

Private root `/Users/shagi/.config/accord-gui/diagnostics/qa198-1792769934859163000/`:
`results.json` SHA1c87b4885133a00cc691f2d5fe6672ac1b5ddc4491acb40936e0474b56585297,
failure-private.json, safe-failure-proof.json и executed-harness copies.
Предложена узкая test-only поправка: разрешить только presentation GIT_PAGER
с fixed --no-pager у parent fixture Git, остальные GIT_* overrides по-прежнему
отклонять. На момент первого failure поправка не применялась; дальнейший reopen ниже.

## Узкий test-only reopen — SOURCE READY, повтор NOT-RUN

Root разрешил только allow GIT_PAGER + fixed parent Git --no-pager. Создана
отдельная private source directory `.local/task014/qa198-v2/`; исходные198 source,
source-pins и первый execution report не перезаписаны. Две изменённые строки
fixture.py разрешают единственную presentation variable и запрещают pager у
parent fixture commands. Все остальные inherited GIT_* по-прежнему отвергаются
до child; native permission/sandbox/MCP isolation и one-attempt budget прежние.

New fixture.py SHA12ebec7b5ab5a7149950cc252f6c7d621f89078b58313c6510d2c0aa908dcfe5;
new source-pins.json SHAc7beeba47a91d25b623f9ba029eb45ff4c4f05e2a2d6c03caf72326283252346.
Runner SHA bbb0a77c и три остальных module fingerprints не менялись; candidate7207,
native/git/source193 pins прежние. AST/current source pins PASS; новый execution
NOT-RUN до169 narrow source reapproval и отдельного root GO.

## V2 actual native — write PASS / commit FAIL, overall FAIL

169 narrow SOURCE APPROVE и отдельный exact root GO получены. Единственный
v2 execution: discovery0.656s, verify0.062s; все5 discovered connectors disabled,
новый managed server не создавался. Actual installed Codex выполнил один turn
за27.546s, real UUID/turn.completed, no MCP calls. Все3 owned supervisor calls
имеют proven success/exit0, output drained, no forced termination/ownership
unknown. Exact `QA198 after\n` записано в message.txt — write **PASS**.

Commit **FAIL**: HEAD остался исходным, parent за модель commit не выполнял.
Observed tool executions: pwd0, git status0, rg1, printf0. **Нет observed git
add/commit invocation или соответствующего OS error/exitcode.** Final agent text
сообщает `.git/index.lock / Operation not permitted`; это semantic refusal,
а не доказанный фактический Git write permission denial. Exit0 не принят как
результат задачи. Повторной модели/обхода/изменения permissions не было.

Отдельно harness comparison обнаружил изменение hash effective native user
config во время run. QA не записывает этот файл; root подтвердил отсутствие
своих concurrent settings edits. Причина неизвестна, config не восстанавливался.
До-run hash оставался только в памяти harness, поэтому exact changed fields
независимо восстановить по этому artifact нельзя; ограничение сохранено.
Drift даёт отдельный FAIL даже независимо от отсутствующего commit.

Code/source/candidate/HOME binary hashes unchanged. Private HOME root
`/Users/shagi/.config/accord-gui/diagnostics/qa198-1792770153191955000/`:

- results.json SHA4b34c6852a53b479e25fac6058096b8a316928fc08ea637c007cfc6206cc52ff;
- input-provenance, executed-harness exactv2 copies, bounded private provider log;
- safe-native-proof.json с whitelist command/error facts и config drift limitation.

Fixture с partial write сохранён для review; своих активных processes/unknown
ownership нет по positive supervisor proofs. Global config не purged/restored.
Любой новый native запуск требует нового design/source approval/root GO.
TASK014 pipeline gate остаётся открытым; эта проверка не является его PASS.
