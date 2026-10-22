# TASK013 process, browser and native QA

Independent tester WORK168 after SOURCE APPROVE165. All state is new private
`.local/task013-qa/`; no production/GUI/Tandem credentials or paths are read.
The process suite refuses occupied loopback ports3100/3101/3102. Candidate binary
is copied into its private run before execution; source and binary drift fails.

```sh
ACCORD_TASK013_REVISION=EXACT_REVIEWED_REVISION \
  /absolute/node24 tests/process/task013/run.cjs
```

Default: real Go HTTP/server/worker/supervisor processes + deterministic native-like
Python3.11+ fixture (stdlib TOML parsing), actual Next/Chrome browser. Fixture is explicitly not Codex or
Claude. Requires an existing production Next build that targets loopback3101.
`ACCORD_TASK013_BROWSER=0` runs process checks without browser. Chrome/Playwright
paths can be set using `ACCORD_TASK013_CHROME`/`ACCORD_TASK013_PLAYWRIGHT`.
The fixture Python script must be executable. Native APIs/permissions are never
weakened to make these tests pass.

`ACCORD_TASK013_NATIVE=1` adds a separate real installed Mac Codex test at the end:
provider session ID, actual managed stdio MCP Describe/profile, actual task next,
and exact conversation continuation with a new board session and a greater task
fencing generation for the same task. A real synthetic Git checkout is created;
Git trust checks are preserved. Terminal worker handoff is checked through task
attempt history, while ordinary stdio EOF retains its lease. It preserves the existing local CODEX_HOME
and approvals; no bypass flags or automatic approval. Native evidence is separate
`native-result.json`. Native denial is FAIL, not a fixture PASS. Windows/native
Claude and the second physical PC remain NOT-RUN.

```sh
PYTHONDONTWRITEBYTECODE=1 ACCORD_TASK013_REVISION=EXACT_REVIEWED_REVISION \
  python3 -B tests/process/task013/update.py
```

Updater test pins the prior TASK012 binary to its existing private proof, copies
old/new binaries into a private run, seeds three projects/seven DB using the old
writer, applies the actual capable updater and full offline restore/auth-reset.
SQLite/Go/snapshot/restore operations are real. Systemd/Next lifecycle/ingress
boundaries reuse the explicitly simulated TASK011 environment adapter; they are
not hardened-host acceptance. No live service or root-owned paths are modified.

Process containment: Unix assertions cover the owned process group. A detached
setsid child and a separately owned unrelated process deliberately remain alive
through stop and are cleaned only by their fixture lifetimes/handles. This is
not proof of arbitrary descendant containment. Windows Job runtime is separate.
Reports include hashes and truthful limitations. Failed reports remain intact;
use a NEW run directory for each rerun. New tests/evidence require review165.
