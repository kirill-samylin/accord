# TASK011 updater process QA

Separate tester152 suite after source APPROVE150. Requires Python3.12+ and
verified old schema3/new schema4 executables. Runs synthetic default+two projects
only inside NEW private `.local/task011-update-qa/run-*` trees.

```sh
ACCORD_UPDATE_BINARY=/absolute/reviewed/agent-board \
ACCORD_UPDATE_REVISION=REVIEWED_SOURCE_REVISION \
PYTHONDONTWRITEBYTECODE=1 python3 -B tests/process/task011-update/run.py
```

Default macOS baseline `.local/task011/legacy/agent-board` is pinned to343b93c
SHA4e11d2569120dada3df15061bcfc5193aacb7931a048110f27242f0b9419e085.
For Linux set `ACCORD_UPDATE_OLD_BINARY` to the verified cbf2020 executable,
`ACCORD_UPDATE_OLD_REVISION=cbf202086431335d4657c722a0f75328a09dc4ea`,
`ACCORD_UPDATE_BINARY_SHA256` to the exact candidate ELF digest. Baseline Linux
SHA66f949aa6022ade9cf4881c1a8cd65201b35d94c6635db942fcd6f73d42546aa is checked.

Eight checks: seed old3 actual process/profiles2revisions/bound-snapshot/livelease,
successful apply/finalize; startup failure retains hold and resumes; real abrupt
subprocess exit between Go/Next pointers then resume; manual marker, helper
mismatch, expected temporary helper and unknown partial helper. Independent SQL
oracle compares old rows across all7DB and snapshots3/4 are both validated.
Root-owned-host checks are NOT verified by a user-UID relocation adapter.

Actual: Go processes, SQLite3→4, old/new binary capability rejection, backup,
independent preservation checks, real abrupt process death and resume; Linux
also checks actual `/proc` argv/executable hash. Simulated: systemd lifecycle
and introspection, Next bundle/runtime, production helper/root ownership,
ingress fixture. Real hardened stage/systemd/HTTPS/VPS backup requires separate
root-run deployment evidence. No production commands or paths are used.

Output report keeps exact test/operator hashes and fails if they change during
run. Existing manual markers and unknown partial files are preserved. Test
failure reports remain private and are never replaced by a later PASS.
