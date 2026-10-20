# TASK011 independent process and Chrome QA

Written by tester152 after independent source APPROVE150. Only synthetic private
new `.local/task011-qa/run-*` fixtures are used. The harness rejects occupied
ports3100/3101, owns the Go/Next/MCP processes and cleans them up. No production
requests, secrets or project data are used. Tokens remain private0600 files.

Requires Node24.19.0, Python3.12+, reviewed Go candidate and trusted schema3
baseline executable. Chrome mode also requires the completed Next production
build in `web/.local/next` with upstream3101, existing Playwright and Chrome.
Dependencies are not installed. Root must not rebuild Next while Chrome runs.

```sh
ACCORD_TASK011_BINARY=/absolute/reviewed/agent-board \
ACCORD_TASK011_REVISION=REVIEWED_SOURCE_REVISION \
ACCORD_TASK011_SOURCE_DIRTY=clean-reviewed-candidate \
/absolute/node24 tests/process/task011/run.cjs
```

Use `ACCORD_TASK011_BROWSER=0` for actual Linux process-only runs. Linux baseline:

```sh
ACCORD_TASK011_OLD_BINARY=/absolute/cbf2020/agent-board \
ACCORD_TASK011_OLD_REVISION=cbf202086431335d4657c722a0f75328a09dc4ea \
ACCORD_TASK011_OLD_SHA256=66f949aa6022ade9cf4881c1a8cd65201b35d94c6635db942fcd6f73d42546aa \
ACCORD_TASK011_BROWSER=0 \
ACCORD_TASK011_BINARY=/absolute/reviewed/agent-board \
ACCORD_TASK011_REVISION=REVIEWED_SOURCE_REVISION \
ACCORD_TASK011_SOURCE_DIRTY=clean-reviewed-release \
/absolute/node24 tests/process/task011/run.cjs
```

macOS default baseline proof is `.local/task011/legacy/proof.json`, verified
343b93c clean executable. Optional overrides: `ACCORD_TASK011_PYTHON`,
`ACCORD_TASK011_PLAYWRIGHT`, `ACCORD_TASK011_CHROME`, `ACCORD_TASK011_QA_DIR`.
QA_DIR must be a NEW child of `.local/task011-qa`, with safe owner/private ancestry.

`results.json` records binary and baseline hashes/provenance, test and product
source hashes, unchanged-source verification, checks and limitations. Failed run
reports are retained. SQL-row hashes independently compare all7DB old rows and
blobs through actual3→4. Full snapshots/restores3+4, profile revisions/session
snapshots/livelease/cookie, routing/races/replay/strict input, native REST/CLI/MCP,
review→QA/SHA/generation/dependencies/ACL are exercised. Chrome adds actual
commit-lostresponse exact retry,409 drafts, explicit lifecycle/dependency/check
forms, two tabs and late A→B, viewer/logout and16theme/viewport screenshots.
The coordination dialog keyboard smoke uses actual Enter/autofocus/full Tab and
Shift+Tab trap/Escape with exact focus return and no business writes. OS native
headless picker Arrow selection is N/A after an independent non-product repro;
functional profile selection is covered by real browser select controls.

Updater orchestration: `tests/process/task011-update`. Its real Go/SQLite/crash
boundaries and simulated systemd/Next/UID boundaries are stated separately.
Production readonly HTTPS is a later root-run script with separate credentials.
One host is not a physical second PC. Workers/providers and native Claude/Codex
are not tested; full V1 is not claimed.
