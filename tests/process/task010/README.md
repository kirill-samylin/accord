# TASK010 independent process/Chrome QA

Runs only synthetic data in a NEW owned private `.local/task010-qa/run-*`.
The harness refuses occupied ports3100/3101, starts owned child processes, and
cleans them up. It never sends requests to production. Sources stay unchanged
through a run; the report records exact test file hashes. New tests were written
by tester145 after independent source APPROVE143.

Prerequisites: reviewed Go binary, exact5454 baseline binary, Python3,
Node24.19.0; browser mode also needs the existing production Next build at
Go origin3101 and installed Playwright/Chrome. No dependencies are installed.
The baseline must match trusted source revision and executable SHA: local
`.local/task010/legacy/proof.json`, or explicit operator-supplied variables.

```sh
ACCORD_TASK010_BINARY=/absolute/reviewed/agent-board \
ACCORD_TASK010_OLD_BINARY=/absolute/baseline5454/agent-board \
/absolute/node24 tests/process/task010/run.cjs
```

Linux process-only mode (no Next or Chrome required):

```sh
ACCORD_TASK010_BROWSER=0 \
ACCORD_TASK010_BINARY=/absolute/reviewed/agent-board \
ACCORD_TASK010_OLD_BINARY=/absolute/baseline5454/agent-board \
ACCORD_TASK010_OLD_REVISION=5454eb5c98ac088e5ead3daaaad637423112ddad \
ACCORD_TASK010_OLD_SHA256=OPERATOR_VERIFIED_BASELINE_BINARY_SHA256 \
ACCORD_TASK010_REVISION=OPERATOR_VERIFIED_NEW_SOURCE_REVISION \
ACCORD_TASK010_SOURCE_DIRTY=clean-reviewed-release \
/absolute/node24 tests/process/task010/run.cjs
```

Additional optional paths: `ACCORD_TASK010_PYTHON`, `ACCORD_TASK010_PLAYWRIGHT`,
`ACCORD_TASK010_CHROME`. Never pass credentials through environment arguments.
All tokens are synthetic, created into private files under the fixture. Reports
contain hashes/counts/expected synthetic error labels; no bearer values.

Coverage: actual schema2→3 across default and two projects, exact old rows and
blobs, durable migration repeat, old/new full format2 restore/reset with history,
legacy cookie/session/lease/unbound preservation; profiles versions/replay/
archive/snapshots/same-name isolation; native and browser ACL/epochs; actual
stdio MCP12tools/resources; failclosed missing profile authority without blob
cleanup or table repair. Chrome adds actual commit-before-lost-response exact
retry, dirty draft409/explicit rebase, tabs/late A response, read-only viewer UI,
and16screens at1440/390 in light/dark. Stored provider fields do not run workers.

Updater orchestration is separately tested in `tests/process/task010-update`.
Production HTTPS uses `tests/browser/task010/production-readonly.cjs` with private
client/token files, existing records and own login/open/logout only. Root runs it
only after safe deployment. Neither local nor Linux synthetic QA proves a physical
second PC or full V1.
