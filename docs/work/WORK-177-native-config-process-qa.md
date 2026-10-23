# WORK177 — actual Go configuration-only native диагностика

23.10.2026. Отдельный QA; root координатор, author WORK176, independent reviewer
WORK169. Production service/config, GUI source, profile/controller grants и
models не изменяются. WORK172 native GUI preparation FAIL остаётся исходным
непройденным результатом; WORK180 third participant login PASS его не заменяет.

## Основание и gate

Автор176 добавил только приватные allowlisted preparation stages с ownership
suffix, сохранив прежнее public error. Root передал SOURCE/units APPROVE169
и разрешил QA process probe после review самого нового probe. До этого gate
созданы только private source copy/Go test/runner и compiled test binary;
actual process probe **NOT-RUN**.

Existing Python config-only чтения в interactive/LaunchAgent context ранее
PASS примерно33/31ms; actual worker preparation отказал примерно через5.19s.
Это не устанавливает причину. Нужно пройти настоящий Go readNative/startProcess
с Wait/PGID proof и исходными5s timer/1MiB limit в том же GUI cwd, не менять guards.

## Новый private probe

`.local/task013-live/qa/qa177-1792759978487633000/` содержит363 owning source files,
точно совпавших с current approved176 tree, и отдельный private `_test.go`.
Новая production CLI/RPC/doctor surface не добавляется. Go test binary собран
без исполнения тестов; обычные owning sources не правились.

- `TestQA177ActualConfigOnly` вызывает actual `readNative` и
  `verifyCodexMCP`, с production default `startProcess`, без fake-process seam.
- Только установленный native Codex `mcp list --json` и generated map + тот же
  `mcp list`; никаких `exec`, `resume` или MCP tools/call. Verification bridge
  намеренно inert `/usr/bin/false`: никакая попытка её запуска не даёт доступа.
- Три bounded samples в interactive context и три под отдельной temporary
  LaunchAgent. Native working directory — настоящий GUI checkout. Temporary
  plist копирует operational environment/WorkingDirectory действующей службы,
  но имеет собственные label/program arguments/logs/one-shot lifecycle.
- Только result, fixed diagnostic stage/ownership suffix, elapsed milliseconds,
  byte count/output SHA/connector count. Raw stdout/stderr/env, credential paths
  и private origins не попадают в отчёт; test logs остаются private.
- Source/testbinary/nativeCLI/persistent config/plist/runner pins до/после,
  GUI3316file baseline до/после, exact top-level launchd state. Production
  worker не заменяется и не останавливается. Удаляется только owned temporary
  label. На outer timeout — FAIL/manual inspection; generic Codex pkill или
  неподтверждённые PID kills запрещены.

Source fingerprints перед review:

- runner `064200409f7a6656a89eca37b13468b8751420111094ff3f02dcda8127d07b08`;
- private test `a9ac82ed703064d4cd46ba58b802a64c86295b4dc8ff08efd56eb6c21e846cdd`;
- compiled test `4262a15f11e134950f939c99586fb4eccf53b98f7f7586ec8b0cd08b2c78f71c`;
- source manifest `2a8138fb20de388e92ed42090b5e44f7f9cd5dd42e78af2d05b0f5080ed2bfb2`.

Build/gofmt/Python syntax и363 source pins **PASS**. Source передан169;
execution до его approve не выполняется. Любой native model/UI retry требует
нового отдельного root GO после фактического диагноза.

## Финальный source review и actual результат

Reviewer169 потребовал независимые cleanup/verification steps с always-report,
full363 source/manifest postchecks и честный manual-inspection status на
timeout/nonzero/missing output. Helper bootout не считается proof завершения
отдельной native PGID. PASS требует observed phase PASS, exact interactive
exit0 и top-level launchd last exit code0, без manual/timeout uncertainty.
После этих дельт final runner SHA256
`79576e21715237e029b095dd3f5e6ada69d6f9997ca572c6238eaa8dec8023ab`.
Go test/probe/source manifest pins прежние. Final SOURCE APPROVE169 и отдельный
root GO получены перед одним bounded запуском.

Actual config-only comparison **PASS**:
`.local/task013-live/qa/qa177-1792759978487633000/comparison.json`, SHA256
`98ae562b9460851970ab157d5490c5b84f0473bf3fe5010ddcb9e59f02a598ba`.

- Interactive:3 discovery+3 verification PASS,31–56ms на phase.
- Temporary LaunchAgent:3 discovery+3 verification PASS,31–45ms на phase.
- Оба test process exit0, fixed diagnostic stages/ownership uncertainty не
  возникли; native model/MCP tool calls0.
- Temporary owned label удалён; GUI3316file baseline,363 source files, manifest,
  probe/runner/native CLI/persistent config/plist pins unchanged. Исходный
  persistent LaunchAgent продолжает running. Timeout/cleanup failures нет.

Это проверка настоящего Go readNative/startProcess, но с own diagnostic binary/
label и inert verification binding. Она не равна полной production worker
preparation/native launch. Первичная5.19s preparation ошибка **не воспроизведена**
и причина **не установлена**; WORK172 native GUI FAIL не закрыт этим PASS.
Полные safe reports передаются169 для независимого evidence review.
Reviewer169 дал final evidence **APPROVE**, независимо сверив raw contexts,
sources/pins, отсутствие temporary label и running исходного worker. Все12
config phases PASS подтверждены; причина исходного отказа остаётся неизвестной.
