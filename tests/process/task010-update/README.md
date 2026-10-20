# TASK010 isolated update process QA

Requires Python3.12+ and verified old5454/new schema3 executables. It does not
invoke live systemctl, Next, nginx, canonical ports or production paths.
Systemd/Next/UID adapters are explicit report limitations; actual Go subprocesses,
SQLite, full snapshots, file guards, crash/resume and Linux `/proc` checks run.

Local Darwin:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -B tests/process/task010-update/run.py
```

Linux, inside a private synthetic source export containing these four Python
files and the exact candidate `infra/accord/**`:

```sh
ACCORD_UPDATE_OLD_BINARY=/opt/accord/releases/5454eb5c98ac088e5ead3daaaad637423112ddad/agent-board \
ACCORD_UPDATE_BINARY=/opt/accord/releases/REVIEWED_COMMIT/agent-board \
ACCORD_UPDATE_BINARY_SHA256=VERIFIED_LINUX_ELF_SHA256 \
ACCORD_UPDATE_REVISION=REVIEWED_COMMIT \
PYTHONDONTWRITEBYTECODE=1 python3 -B tests/process/task010-update/run.py
```

Old executable SHA is checked against the independently recorded platform proof;
Linux new SHA must be supplied from the reviewed release manifest. There is no
Git or Node runtime dependency. Output goes only into a NEW ignored private
`.local/task010-update-qa/run-*` directory. Report stores synthetic test/operator
hashes and statuses, never credentials or database row content. Timeout kills
the owned worker process group. Each worker checks its ephemeral ports are free.
Unknown markers/partial temporary/manual helper bytes remain for inspection.

The same worker abruptly exits after Go pointer publication and before Next
pointer publication. A different process resumes the existing journal and
snapshot; schema2 rows in all seven registry/board/web databases are compared
independently against the post-update state. Finalize creates schema3 full backup
without deleting the pre-update schema2 snapshot. This is not a physical second
PC, actual systemd update, Next acceptance or production HTTPS test.
