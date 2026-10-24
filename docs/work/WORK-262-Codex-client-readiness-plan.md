# WORK262 — bounded Codex CLI readiness, SOURCE ONLY

`ops_validation_qa`, 24.10.2026. Подготовлен только private план
`.local/task014/client262-readiness-plan/`: exact command argument arrays,
expected output predicates, privacy/lifecycle bounds, README и input pins.
Нового runner/framework нет. Никаких CLI/model/MCP bridge/runtime запусков.

Первым изучен actual `internal/worker/codex_config.go`, затем discovery parser.
Поддерживаемая проверка — pinned Codex `mcp list --json` из Test cwd, сначала
baseline catalogue, затем exact231V2 process-only disable2keys и accord_test
stdio table. Worker source использует своё reserved name; клиентский план
сохраняет именно approved standalone accord_test и точные command/args.

Оба шага закрыты отдельным root GO. Per-child5s, final owned wait/drain10s,
stdout/stderr cap1MiB, raw только0600 под0700. Raw MCP JSON может содержать
env/header/URL значения: публично только finite status/counts/binding predicates.
Token file не читался/не хешировался; argv содержит только ссылку на него.

PASS требует exactly one enabled accord_test со stdio и exact worker+args,
explicit enabled boolean, bounded unique entries, отсутствия других enabled
connectors. Disabled entries могут оставаться в JSON. If enabled=false-only
создаёт invalid plugin table — STOP/report, без молчаливого transport/full-map
fallback, дополнительных disable keys или global/profile/plugin изменений.

Installed231V2 client.json сохранился byteexact: launch_ready=false и namespace
flag=false. Launcher не запускался. Pinned Codex SHA взят из ранее принятого
tool pin и root fresh preflight declaration;249MB binary повторно не читался,
перед actual GO root подтверждает exact pin. JSON/TOML static PASS.

Manifest3members SHA256
`b0a3a2c0efe6d45c937f070f39d04dc4650558b94036b6615e74d8fd176aff77`.
Все private files600 под700. Actual CLI readiness NOT-RUN до root GO; это не
live handshake/native/workflow acceptance и не registration текущего desktop чата.
Claude/Windows/физический второй ПК NOT-RUN. План не блокирует отдельный уже
разрешённый пользователем Python workflow через UI.
