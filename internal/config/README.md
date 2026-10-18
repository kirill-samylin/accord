# Offline configuration API

`Parse(input []byte, Options) (*Snapshot, Result)` accepts one bounded UTF-8
YAML/JSON document, applies defaults, validates references and safety floors,
and hashes canonical JSON. No network, environment expansion or filesystem IO.
`Result.Valid=false` always returns a nil snapshot and typed diagnostics.
Integration capabilities default disabled. A runtime must enable an option only
when it implements that feature; actual tool descriptions belong in Options.

`Schema()` and `DavinciTemplate()` return independent embedded asset copies.
The template deliberately contains `OWNER`, which validation rejects until
explicitly replaced with a local operator login. There is no hidden OAuth setup.

`Snapshot.Decode()` and `CanonicalJSON()` return independent copies; never retain
one mutable Config as a shared authority. Store integration must read persisted
canonical config in the same transaction as board decisions. `TTLs()` returns
validated durations plus min(task, role, session)/3 heartbeat hint. `ToolBudgets`
reports conservative UTF-8 description bytes and schema bytes separately;
neither count is a measured model-token count.

`ValidateEvidence(role, raw JSON)` returns Valid, Complete, Errors and Warnings.
It checks the restricted schema and immutable built-in minimum. Missing extra
fields fail in block, warn in warn, and remain Complete=false in off. Store must
still enforce token scopes, live leases, immutable author provenance, current
commit/generation, server session separation, positive gate verdicts and custom
verifier obligations. Shape success alone never proves a review passed.

Evidence schema supports the contract vocabulary only, local acyclic `$defs`
JSON Pointer references, asserted URI/date-time, Go regex, exact JSON numbers,
bounded depth/nodes/properties and byte-limited built-in evidence. Redefinition
of a built-in field must exactly match its supplied canonical schema; weakening
or replacing its type is rejected. Unknown fields need an explicit custom schema.
Numeric exponents are bounded before expansion to protect memory.

`CanonicalArea`, `Snapshot.ResolveArea`, `AreasIntersect` operate on path strings,
without following actual files or symlinks. Exact aliases precede canonical path
children. Authorization for single_editor_role and live overlap belongs to store.

This package does not migrate SQLite, persist/apply config, validate live-board
CONFIG_IN_USE conflicts, mint scoped tokens, expose endpoints or implement any
GitHub/notification transport. These remain the integration owner's work.
