# TASK012 updater and restore process QA

Separate tester WORK157 after SOURCE + AUTHOR TESTS APPROVE153. Uses the real
old TASK011 schema4 binary, new schema4/deduplication1 binary, SQLite and reviewed
operator functions. Every output is NEW private `.local/task012-update-qa/run-*`.
No credentials or production files are read. Python is always `-B`.

```sh
ACCORD_UPDATE_BINARY=/absolute/reviewed/agent-board \
ACCORD_UPDATE_BINARY_SHA256=EXACT_CANDIDATE_SHA256 \
ACCORD_UPDATE_REVISION=REVIEWED_SOURCE_REVISION \
ACCORD_UPDATE_OLD_BINARY=/absolute/old/agent-board \
ACCORD_UPDATE_OLD_INFRA=/absolute/old/release/infra \
PYTHONDONTWRITEBYTECODE=1 python3 -B tests/process/task012-update/run.py
```

Mac old binary SHA is daf342c2bb7d714af7d4aceea6910de31eb9e6ddc0751b926ac667284e0bf8ac
(f2edac9, product identical to TASK011 e53c18c). Linux old ELF SHA is
a03f64d5ff008b7f63023117fb7b1c15c33e5617ce7ac08ffc6440610315269d
(e53c18c9643b9ee086141d0f227ab485f4f88cac). Old installed guard is also pinned,
SHA c1703c392fc62891e150e23e52cdc68132f05f1c5dad570c53404f4de07b50ed.
New binary hash is mandatory on both platforms. No `.git` lookup is required;
the exact same suite can run from a packaged Linux fixture with these inputs.

Eight top-level checks: actual old-writer seed; success/finalize/fullrestore;
start failure/hold/resume; abrupt worker exit73 after Go pointer, then resume;
manual marker; helper drift; known partial helper; unknown partial helper.
Seed is default+two projects/seven DB with tokens, ACL/epochs/repositories,
profiles/revisions/session snapshots, live lease, tasks/notes, assignment/dependency.
Schema1/2/3 SQLite history files are explicitly synthetic opaque byte fixtures;
they do not claim to reproduce historical production migrations.

Independent SQL oracle checks every existing row across all seven DB. Independent
file oracle checks history checksum files and authority-marker bytes/UID/mode.
The old installed guard accepts the old binary before update. The new installed
guard must deny old schema4 before the migration command while startup hold is
present. Offline restore checks all seven DB, preserved task/profile/note/replay
history and old audit rows, explicit recovery audit append, revoked credentials,
expired/cleared sessions, advanced membership epochs and retained boot hold.
Source snapshot and active fixture bytes remain unchanged by offline restore.

Reuses unchanged TASK011 environment/fixture adapters by import. Actual: Go
processes, updater/backup/restore functions, SQLite, installed helper replacement,
capability denial, abrupt subprocess death and recovery. Linux additionally
checks actual `/proc/PID/exe` and argv. Simulated: systemd lifecycle/introspection,
Next bundle/runtime, ingress and root-owned configuration; UID adapter uses the
invoking user. Hardened stage, real systemd/Next/HTTPS and production remain
separate checks. One host is never evidence of two physical PCs.

Report records test/operator/old-helper hashes, binary hashes and supplied source
revision. Source drift fails the run. Historical FAIL reports remain private;
a later PASS never overwrites them. Product modifications are outside this suite.
