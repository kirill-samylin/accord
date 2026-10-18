# Accord JSONL v1: библиотечный контракт

Первая строка — объект Manifest: format=`accord-jsonl`, version=1,
source=`[A-Za-z0-9._-]{1,128}`. Необязательные high_water и config_version —
неотрицательные int64. Source сверяется с отдельно переданным source.
Далее каждая строка — `{"kind":"task","external_id":"...","data":{...}}`.
external_id непустой, ≤256 bytes, без NUL, unique по (kind, external_id).
Свойства регистрозависимы; неизвестные/повторные поля запрещены.
Структуры в types.go — точный schema v1; JSON Schema рядом с README.

Обязательные свойства (остальные ниже перечислены как optional):

- task: type=`task`, title, description, role, state, priority(integer).
  state: created/ready/in_progress/done/cancelled. Optional owner, requester,
  areas:string[], category, complexity, epic:Ref, evidence:Evidence, target:Ref.
- epic: title, description, state(open/done/cancelled), priority(integer).
  Optional owner, areas:string[].
- link: from:Ref, to:Ref, type(blocks/contract_of/spawned_by/related/duplicate_of).
  Optional condition(empty/completed/approved/passed).
- note: text, kind. Optional role, epic:Ref. Значения kind и роль валидирует
  конфигурационный validator; библиотека не утверждает их существование.
- history: origin_ref:Ref, type, at:RFC3339. Optional text,
  origin_session_label (namespaced, содержит `:`), tokens_spent(nonnegative int64),
  evidence:Evidence, source_state, role, config_version(nonnegative int64).

Ref = {kind, external_id}; разрешённые target kinds зависят от свойства.
Task.epic/Note.epic → epic; Task.target → task; Link.from/to → epic|task;
History.origin_ref → epic|task|link|note. Никаких URL/source/local-key ссылок.
Evidence — только optional strings commit/pr/run_summary/summary/answer/
verified_commit/findings/verdict и artifact:string[]. Это **недоверенные**
исторические данные. Полей leases, active sessions, author_session, generation,
completion_trust, credential hashes, config values, outbox и auth нет.
Свободный текст может содержать произвольный пользовательский контент; экспортёр
не способен определить секрет, вручную вписанный в description/comment/evidence.

`Parse(reader, source, validator)` читает ≤16MiB, ≤10000 records,
≤256KiB/line, nesting≤32, возвращает максимум100 diagnostics. При любой ошибке
Document=nil: валидный префикс никогда не выдаётся для apply. Exact raw bytes,
включая newline/CRLF, дают FileSHA256. Duplicate keys проверяются до decoder;
числа не переводятся в float64. Canonical record hash сортирует object keys,
нормализует escaping/whitespace, сохраняет numeric lexemes; int64 поля обязаны
иметь integer JSON spelling. Числа за int64 отвергаются.

С validator=nil это **только syntax/schema/references/DAG**, не config-valid.
`ParseWithReferences` разрешает ссылки на existing mapping того же source через
переданный callback; caller обязан привязать callback к текущему source.
Без callback пакет обязан замыкать все references. Библиотека проверяет
внутрифайловые cycles blocks/contract_of/spawned_by; graph с уже существующими
local links повторно проверяет store. Policy validator и authoritative apply
проверяют текущие роли/evidence/owner/areas/limits/uniqueness/leases под lock.

Plan/Result — только DTO. Библиотека не создаёт import_runs, не делает SQL,
не возвращает Applied=true и не может поднять доверие результата. Семантику
in_progress→ready, done→imported_unknown и local admin requeue реализует store.

`Encode` и `ExportFile` принимают NextRecord из **одного store read snapshot**;
не создают его сами. Snapshot owner отвечает за stable external IDs, полноту
ссылок/истории и sanitized config history (здесь разрешён только config_version).
Encoder проверяет поля и размер каждой записи, не ограничивает весь export16MiB.
Для большего файла CLI должен сообщить необходимость references-closed split.
ExportFile: temporary0600 в том же directory, encode+fsync+close,
atomic hardlink publish без overwrite, directory fsync. Ошибка до publish не
оставляет destination; ошибка после publish может оставить полный файл и не
означает rollback. Filesystems без hardlink возвращают ошибку без fallback.
SQLite disaster recovery/backup — отдельная операция.
