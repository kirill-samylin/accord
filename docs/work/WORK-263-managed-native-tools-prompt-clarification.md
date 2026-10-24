# WORK263 — уточнение MCP namespace и встроенных native tools

Автор `ops_validation_qa`, 24.10.2026. Изменены только
`internal/worker/execution.go` и focused regression
`internal/worker/role_check_prompt_test.go`; общий TASK/профили/config/worker/
permissions и сервер a32 автором не менялись.

Первый настоящий analyst turn Test остановлен на human question без чтения
локальных файлов. Проверен только exact UUID transcript с совпадающим session id
и Test cwd; read-only sandbox/approval policy подтверждены его turn metadata.
Было7 code-mode calls: MCP/catalogue/resources/question, ни одной попытки native
file/shell/Git. Scoped terminal receipt подтверждает5 literal-true ACK/exit/lock
флагов, checks0 и pending0. Live identifiers, transcript path/hash и минимальные
проекции находятся только в private ignored diagnosis.

В native prompt присутствовала строка «Use only the accord_managed MCP connector
for this run». Model statement «нет reader» **не доказывает отсутствия builtin
tools**: его catalogue query фильтровал только MCP/resource/read-list имена,
исчерпывающий каталог native tools не наблюдался. Actual source chain настраивает
MCP map и role sandbox; отдельного builtin shell/file disable в нём нет.
Ранее принятые реальные native read-only Git/file сценарии подтверждают наличие
такого режима, но не подменяют полный каталог именно этого turn. Причина поэтому
классифицирована как **plausible prompt ambiguity**, а не доказанная пропажа tools.

Новая формулировка ограничивает только MCP namespace. Available native project
file/shell tools остаются под прежним local sandbox, saved role и task lease;
новых tools/permissions она не выдаёт. Analyst/reviewer явно могут read-only
project inspection/Git, не edit/commit/check. Developer может только прежние
assigned edits/local commit. Tester сохраняет reviewed source/HEAD и отдельный
artifact consent. Verification commands нельзя выполнять напрямую через shell:
developer/tester используют только существующий workflow_run_check и ждут actual
completed evidence; analyst/reviewer это право не получают. Отсутствующий/denied
tool требует blocker/вопроса, не выдуманного evidence.

По последующему явному поручению root выполнены targeted3 worker unit functions,
включая4role prompt cases, actual role-check refusal и manual/continuation request
preservation: **PASS**, process exit0/2.712s. Только synthetic unit tests; ни native
provider, ни настоящий worker/model не запускались. Dependencies/network отключены
GOTOOLCHAINlocal/GOPROXYoff/GOSUMDBoff. Repo pinned gofmt no diff и git diff-check
PASS. Первичный gofmt отсутствовал в PATH (127), затем использован точный repo
toolchain; первоначальный source-only снимок сохранён, не переписан.

Code freeze `.local/task014/worker263-prompt/source-pins.json`:
`6fb901baa5723c8101c30abd81fab76f1c0056202e71d66fafd97bd3ccfe777d`,4members.
Execution SHA256 `03dabe62c33a49f4960656e4843eee617227b080c3692ca4cfb0788f1aa2ffa8`;
test SHA256 `0d26476c8a02ec5c2fb1ded90f334740d3e7e29dfc32390901920620056bc40a`.
Targeted unit proof `.local/task014/worker263-prompt-tests/targeted-go-proof.private.json`:
`a3baeffe638032e8940fcaf6758d231143fd7d7afe8356678a1f1894b9d953ef`.
Log SHA256 `131043ff9b148c46941932f68d8b164a3dc771d8a3c5c7fe71bcdf325f4491d6`.

Independent202 source review обязателен до root build/install. Actual patched
worker/same-UUID resume/native validation **NOT-RUN** автором; это следующий
операционный этап root. Sandbox/approval/allowlists/check ACL/lease/generation/
SHA gates не менялись, Windows/Claude/физический второй ПК не проверялись.
