# WORK267 — полный disabled stdio override для известного cua_repl

Автор gui_connection_qa,24.10.2026. SOURCE READY; только новый private client267/v1
и эта карточка. WORK265 candidate/actual history и установленное не изменены.
QA261/native workflow/control этим WORK не выполняются.

Root actual265 первый parser failure сохранил stdout0/stderr115, но numeric exit
не записал: для него exit UNKNOWN. Отдельный root parser-repeat result подтверждает
FAIL_PARSER/exit1/timedoutfalse/stdio_drainedtrue/stdout0/stderr115/models0 и global
config unchanged. Private stderr прочитан/hash-проверен, без публикации полного
текста/значений. namespace cua_repl подтверждён; старый FAIL не стал PASS.

Source internal/worker/codex_config.go уже объясняет transport контракт: injected
stdio, отсутствующий в user config, не получает transport от enabled-only leaf;
полный disabled table сохраняет валидную schema без копирования секретных argv.
Новый кандидат использует тот же известный подход, без fallback/retry.

Из actual installed600 скопированы client/launcher/Claude. В client изменены только
два поля: cua_repl leaf убран из codex_disable_mcp_keys (прежние два сохранены),
добавлен codex_disable_stdio_servers=['cua_repl']. Новый launcher принимает ровно
этот список/имя и создаёт один process-only override с enabled=false, command=
ранее проверенный установленный worker и args=[]. Нет arbitrary names, URL transport,
новых tool grants/allowlists/model/approval/sandbox/глобальных настроек.
Worker263 dfcef4b/f629 pins, project/workspace/token paths и Claude JSON неизменны.
Оба readiness flags false, inventory NOT_PROVEN; установленное byteexact до/после.

Freeze `.local/task014/client267/v1/source-pins.json` SHA
`639eb59f68b7c17e853d27ad4a8e4d5632deedd1d9c146c1acbaa555fa5cefd2`,6members.
Python AST2 PASS. Новый pure test выбирает только builder function AST, проверяет
TOML/argv/known-server rejection/falseflags; NOT-RUN до независимого source review.
Автор не запускал CLI/launcher/main/provider/model/MCP/network/browser/native,
не читал token contents/global provider config/worker config/state.

Следующие отдельные root действия после SOURCE APPROVE202: reviewed replacement,
bounded actual262 JSON inventory recheck, затем отдельный activation review.
Этот кандидат не включает флаги и не создаёт новый submit текущего workflow.
