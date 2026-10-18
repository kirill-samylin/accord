# Конфигурация и расширение workflow — контракт C/D

18.10.2026; WORK-031 → [TASK-003](TASK-003-working-service.md).
Это исполнимый контракт следующих изменений, не отчёт об их реализации.
Приоритет: [requirements 1.1](requirements.md), затем
[service-contract](service-contract.md), затем [исходник §10](source/TZ-agent-board-service-v1.0.md).
Не задерживает независимую поставку уже проверенного native A/B.

## 1. Версии, владельцы и совместимость

Разные счётчики: YAML `version: 1` — язык конфигурации;
`config_version` — монотонная применённая редакция; `PRAGMA user_version` — БД.
C получает schema2: notes и additive данные/события. D получает schema3:
config_history/config_active/token scopes. Последовательные миграции принадлежат
одному исполнителю store/integration; не создавать параллельные migration2.
DDL, backfill и user_version фиксируются одной SQLite write-транзакцией.
До миграции — проверенный backup; crash до commit оставляет прежнюю схему.
Старый binary отвергает более новую БД до DDL. Нет автоматического downgrade:
откат binary на schema1 требует restore schema1 backup и recovery reset;
это потеря более поздних изменений, явно указанная в процедуре оператора.

C не меняет публичный typed `Evidence`, только добавляет поля из §5;
D один владеет заменой на `map[string]json.RawMessage` с типизированными getters.
Не декодировать числа через float64: integer и tokens обязаны сохраняться точно.
Старые JSON evidence остаются валидными; неизвестный тип обязательного поля —
ошибка, не empty string. Transport/SDK schema `evidence` — JSON object,
остальной DTO strict; произвольные верхнеуровневые поля по-прежнему запрещены.
`outcome` transition сохраняется для старых клиентов; `evidence.verdict/result`
каноничны. Если переданы оба, значения должны совпасть; ответ содержит оба.

Миграция schema3 создаёт config_version1 `bootstrap_compat`: реальные
предыдущие project/TTL, семь пилотных ролей, прежние права и reopen, без
новых обязательных evidence полей. Offline apply требует явные TTL/bootstrap
аргументы при отсутствии persisted config: не угадывать флаги живого процесса.
Новый процесс не переопределяет persisted config startup-флагами: расхождение
явно заданных TTL/project → startup error; неявные defaults не конфликтуют.
Новый проект получает тот же compatibility profile; `config init --profile
davinci` выводит полный шаблон §2, который оператор применяет явно.

## 2. YAML, форма и defaults

Минимально достаточный полный шаблон; `OWNER` заменяется локальным login,
проверяется validate. Это не секрет и не GitHub OAuth identity.

```yaml
version: 1
project: {name: accord, ids: {epic_prefix: E, task_prefix: T, note_prefix: N}}
auth: {mode: local, local: {users: [OWNER]}, agent_token_ttl_days: 90}
leases: {task_ttl: 15m, role_ttl: 15m, max_silent: 10m, sweep_interval: 30s}
roles:
  coordinator: {kind: agent, singleton: false, can_create: [task, epic], can_cancel: true, can_curate_notes: true, evidence_on_done: [summary]}
  analyst: {kind: agent, evidence_on_done: [artifact, summary]}
  architect: {kind: agent, singleton: true, evidence_on_done: [artifact, summary]}
  developer: {kind: agent, evidence_on_done: [commit, pr, run_summary], recommended_on_done: [tokens_spent]}
  reviewer: {kind: agent, evidence_on_done: [verdict, verified_commit, findings], independent_from: [developer]}
  security: {kind: agent, evidence_on_done: [verdict, verified_commit, findings], independent_from: [developer]}
  design-reviewer: {kind: agent, evidence_on_done: [verdict, verified_commit, screens], independent_from: [developer, design-owner]}
  tester: {kind: agent, evidence_on_done: [run_id, verified_commit, result, run_summary], independent_from: [developer]}
  devops: {kind: agent, evidence_on_done: [summary]}
  llm-quality: {kind: agent, evidence_on_done: [summary, artifact]}
  human:
    kind: human
    categories:
      product_rule: {answer_by: any, complexity_max: high}
      ux_path: {answer_by: [OWNER], complexity_max: high}
      technical: {answer_by: any, complexity_max: normal}
    evidence_on_done: [answer]
  design-owner:
    kind: human
    members: [OWNER]
    categories: {visual_defect: {}, new_state: {}, ds_import: {}}
    evidence_on_done: [commit, pr]
states: {contract_delivered_at: done}
reopen: {tester: [developer], reviewer: [developer], security: [developer], design-reviewer: [developer, design-owner], developer: [analyst, architect], coordinator: ['*'], human: ['*']}
rules:
  evidence_required: {mode: block}
  independent_review: {mode: block}
  singleton_role: {mode: block}
  one_task_per_agent: {mode: block}
  epic_owner: {mode: warn, shared_roles: [reviewer, security, tester, design-reviewer]}
  area_overlap: {mode: warn}
  duplicate_check: {mode: warn, threshold: 0.8}
  cycle_check: {mode: block}
areas: []
questions: {escalate_after: 48h, default_option_note: 'Работа идёт по варианту 1 до ответа'}
notes: {max_per_role: 30, max_bytes_per_role: 24000, max_per_epic: 20, max_bytes_per_epic: 16000}
mcp: {tool_budget_tokens: 3000, tools_by_role: {}}
```

Незаполненный tools_by_role разворачивается в стабильные 12 tools каждой роли;
непустой требует запись для каждой роли. Это permission на вызов, не побочный
эффект describe для tools/list. Базовые grants всегда дополнительно проверяются.
Остальные defaults: singleton/can_cancel/can_curate_notes=false;
can_create=[task] для agent, [] для human; members не ограничивает agent;
categories={} означает human без категоризации, complexity_max=high;
answer_by=any — любой разрешённый local user с human role scope.
Непереданные evidence_on_done/recommended_on_done/independent_from=[];
непереданный evidence_schema={type:object,additionalProperties:false,properties:{}}.
Defaults materialize до canonical hash, explicit [] не заменяется default.
Builtin поля добавляются к evidence_schema; custom поле требует явную схему.
Builtin роли задают безопасный минимальный контракт §5 независимо от YAML.

## 3. JSON Schema и дополнительная валидация

Следующий JSON — основа поставляемого `schema/agent-board-v1.json`.
Валидатор дополняет его семантикой ниже, не считает одну shape-проверку успехом.
Draft [2020-12 Core](https://json-schema.org/draft/2020-12/json-schema-core) /
[Validation](https://json-schema.org/draft/2020-12/json-schema-validation).

```json
{
 "$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,
 "required":["version","project","auth","leases","roles","states","reopen","rules","areas","questions","notes","mcp"],
 "properties":{
  "version":{"const":1},
  "project":{"type":"object","additionalProperties":false,"required":["name","ids"],"properties":{"name":{"$ref":"#/$defs/name"},"repository":{"type":"string","maxLength":512},"ids":{"type":"object","additionalProperties":false,"required":["epic_prefix","task_prefix","note_prefix"],"properties":{"epic_prefix":{"const":"E"},"task_prefix":{"const":"T"},"note_prefix":{"const":"N"}}}}},
  "auth":{"type":"object","additionalProperties":false,"required":["mode","local","agent_token_ttl_days"],"properties":{"mode":{"enum":["local","github"]},"agent_token_ttl_days":{"type":"integer","minimum":1,"maximum":365},"local":{"type":"object","additionalProperties":false,"required":["users"],"properties":{"users":{"$ref":"#/$defs/names"}}},"github":{"type":"object","additionalProperties":false,"properties":{"allowed_logins":{"$ref":"#/$defs/names"},"allowed_org":{"$ref":"#/$defs/name"},"allowed_team":{"$ref":"#/$defs/name"}}}}},
  "leases":{"type":"object","additionalProperties":false,"required":["task_ttl","role_ttl","max_silent","sweep_interval"],"properties":{"task_ttl":{"$ref":"#/$defs/duration"},"role_ttl":{"$ref":"#/$defs/duration"},"session_ttl":{"$ref":"#/$defs/duration"},"max_silent":{"$ref":"#/$defs/duration"},"sweep_interval":{"$ref":"#/$defs/duration"}}},
  "roles":{"type":"object","minProperties":1,"maxProperties":64,"propertyNames":{"pattern":"^[a-z][a-z0-9-]{0,47}$"},"additionalProperties":{"$ref":"#/$defs/role"}},
  "states":{"type":"object","additionalProperties":false,"required":["contract_delivered_at"],"properties":{"contract_delivered_at":{"const":"done"}}},
  "reopen":{"type":"object","additionalProperties":{"$ref":"#/$defs/names"}},
  "rules":{"type":"object","additionalProperties":false,"required":["evidence_required","independent_review","singleton_role","one_task_per_agent","epic_owner","area_overlap","duplicate_check","cycle_check"],"properties":{"evidence_required":{"$ref":"#/$defs/rule"},"independent_review":{"$ref":"#/$defs/rule"},"singleton_role":{"$ref":"#/$defs/rule"},"one_task_per_agent":{"$ref":"#/$defs/rule"},"epic_owner":{"type":"object","additionalProperties":false,"required":["mode","shared_roles"],"properties":{"mode":{"$ref":"#/$defs/mode"},"shared_roles":{"$ref":"#/$defs/names"}}},"area_overlap":{"$ref":"#/$defs/rule"},"duplicate_check":{"type":"object","additionalProperties":false,"required":["mode","threshold"],"properties":{"mode":{"$ref":"#/$defs/mode"},"threshold":{"type":"number","minimum":0,"maximum":1}}},"cycle_check":{"$ref":"#/$defs/rule"}}},
  "areas":{"type":"array","maxItems":256,"items":{"type":"object","additionalProperties":false,"required":["name"],"properties":{"name":{"type":"string","minLength":1,"maxLength":512},"aliases":{"$ref":"#/$defs/names"},"single_editor_role":{"$ref":"#/$defs/name"}}}},
  "questions":{"type":"object","additionalProperties":false,"required":["escalate_after","default_option_note"],"properties":{"escalate_after":{"$ref":"#/$defs/duration"},"default_option_note":{"type":"string","maxLength":2048}}},
  "notes":{"type":"object","additionalProperties":false,"required":["max_per_role","max_bytes_per_role","max_per_epic","max_bytes_per_epic"],"properties":{"max_per_role":{"$ref":"#/$defs/limit"},"max_bytes_per_role":{"$ref":"#/$defs/limit"},"max_per_epic":{"$ref":"#/$defs/limit"},"max_bytes_per_epic":{"$ref":"#/$defs/limit"}}},
  "mcp":{"type":"object","additionalProperties":false,"required":["tool_budget_tokens","tools_by_role"],"properties":{"tool_budget_tokens":{"$ref":"#/$defs/limit"},"tools_by_role":{"type":"object","additionalProperties":{"$ref":"#/$defs/names"}}}},
  "github":{"type":"object","additionalProperties":false,"required":["webhook_secret_env","task_key_pattern","require_approval_for_merge","events"],"properties":{"webhook_secret_env":{"$ref":"#/$defs/env"},"task_key_pattern":{"type":"string","maxLength":512},"require_approval_for_merge":{"type":"boolean"},"events":{"type":"array","uniqueItems":true,"items":{"enum":["pull_request","check_suite","pull_request_review"]}}}},
  "notifications":{"type":"object","additionalProperties":false,"required":["webhooks"],"properties":{"webhooks":{"type":"array","maxItems":16,"items":{"type":"object","additionalProperties":false,"required":["url","secret_env","events"],"properties":{"url":{"type":"string","format":"uri","maxLength":2048},"secret_env":{"$ref":"#/$defs/env"},"events":{"$ref":"#/$defs/names"}}}}}}
 },
 "$defs":{
  "name":{"type":"string","minLength":1,"maxLength":128},"names":{"type":"array","maxItems":256,"uniqueItems":true,"items":{"$ref":"#/$defs/name"}},
  "duration":{"type":"string","minLength":2,"maxLength":32},"limit":{"type":"integer","minimum":1,"maximum":1048576},"env":{"type":"string","pattern":"^[A-Z_][A-Z0-9_]{0,127}$"},
  "mode":{"enum":["block","warn","off"]},"rule":{"type":"object","additionalProperties":false,"required":["mode"],"properties":{"mode":{"$ref":"#/$defs/mode"}}},
  "category":{"type":"object","additionalProperties":false,"properties":{"answer_by":{"oneOf":[{"const":"any"},{"$ref":"#/$defs/names"}]},"complexity_max":{"enum":["low","normal","high"]}}},
  "role":{"type":"object","additionalProperties":false,"required":["kind"],"properties":{"kind":{"enum":["agent","human"]},"singleton":{"type":"boolean"},"can_create":{"type":"array","uniqueItems":true,"items":{"enum":["task","epic"]}},"can_cancel":{"type":"boolean"},"can_curate_notes":{"type":"boolean"},"evidence_on_done":{"$ref":"#/$defs/names"},"recommended_on_done":{"$ref":"#/$defs/names"},"independent_from":{"$ref":"#/$defs/names"},"members":{"$ref":"#/$defs/names"},"categories":{"type":"object","additionalProperties":{"$ref":"#/$defs/category"}},"evidence_schema":{"type":"object"}}}
 }
}
```

Input ≤256KiB UTF8, one YAML document; reject duplicate keys, tags, anchors,
aliases/merge keys and non-string map keys. No env substitution or file includes.
Reject unknown schema keywords, unresolved refs/roles/users/tools, negative or
overflow durations, empty configured names. Limits here — parser/resource caps,
не продуктовые defaults. Durations Go-like including h/m/s/ms; task/role/session
1s..24h (bridge cap), sweep≤min(TTL), max_silent≤min(task,role,session), escalation>0.
session_ttl defaults role_ttl; heartbeat hint=min(task,role,session)/3.
MCP description budget: deterministic upper bound UTF8 bytes of the embedded
tool description strings ≤tool_budget_tokens; label as conservative bytes bound,
not an exact model tokenizer. No truncation; error identifies over-budget role.

Descriptions only per original§11.2; schema bytes reported separately, never
claimed as measured model-context tokens. Existing short descriptions fit3000.
Evidence schema vocabulary: type/properties/required/additionalProperties/items,
enum/const/minimum/maximum/minLength/maxLength/minItems/maxItems/pattern/format,
$defs/$ref; only local #/$defs refs, resolved acyclic graph, depth≤8, ≤128 nodes,
≤64 object properties. additionalProperties=false; max evidence32KiB UTF8,
refs/artifact/screens arrays≤32; no remote load, scripts, expressions, $id.
Formats uri/date-time must be explicitly asserted, SHA via fixed hex validation;
Go regexp-compatible patterns only, regexp compile error rejects configuration.
Config shape accepts future integration sections; validate returns capability
errors if auth.github, enabled github/events or notifications/webhooks are not
implemented by that binary. No successful apply of silently ignored features.
Prefix changes and states.contract_delivered_at≠done rejected CONFIG_UNSUPPORTED:
current clients parse E/T/N and positive gates cannot be delivered before done.

## 4. Apply, история и live-процесс

CLI: `config validate FILE [--profile ...]`; `config apply --db DB --file FILE
--expected-version N --actor LABEL`; `config get/history --db DB`.
JSON validate result={valid,canonical_sha256,errors:[{path,code,message}],warnings}.
No secret values accepted in config; *_env are names only. raw_yaml retained
locally in history0600, read API returns canonical configuration and metadata,
not raw source/comments. Config access не mint/revoke/admin grant.
GET /api/v1/config → {config_version,sha256,config}; GET /config/history
paginated limit1..200 numeric version order, no bearer-protected write endpoint.

Write lock → load active version → CAS → semantic validate against board →
INSERT config_history(version PK,sha256,canonical_json,raw_yaml,actor,applied_at)
→ UPDATE config_active(id=1,version FK) → config_applied(version,hash,actor) →
commit. Initial expected0 only without active record. Same hash+same expected
is audited no-op returning existing version; stale expected always409 conflict,
even if hash matches. Re-applying historical config creates newer version,
never rewinds version. Raw SHA and canonical SHA different fields if both kept.

Reject409 CONFIG_IN_USE with exact session/task/note keys on: removed role
referenced by any nonterminal task/live session/note; kind/singleton/member/
category/permission/evidence/reopen/area or blocking-rule change affecting a
live session/task; any TTL change with live session/lease; limits below current
undeleted note count/bytes. Conservative rejection on any relevant semantic
change is acceptable; do not attempt unprovable schema-subset comparisons.
Expiry materialization may commit independently, candidate config never partly
applies. No automatic lease release/session termination or history rewriting.
Historical role definitions live in config_history; deleting inactive roles does
not erase old tasks/stats. New schema must not revalidate historical completion.

`Store.cfg` is bootstrap-only. Every transaction reads immutable active config
after SQLite writer lock into board.Config; all predicates/renew/Describe use
that snapshot. No mutable shared map or per-process cached cfg for authority;
local CLI apply in another process must affect next transaction. Reads use one
snapshot too; output contains config_version. Scheduler reads interval afresh;
bridge gets unchanged TTL because incompatible live apply is refused. Existing
request_id replay returns exact old DTO under current token auth, no renewal.
New operations use active rules, including existing sessions/tool permissions.

## 5. Роли, evidence и безопасность правил

Store lookup replaces all core.ValidRole/Singleton switches, включая queue,
reopen/gates/attempt provenance. Bridge accepts syntactically valid requested
role only when successful describe echoes that exact role; subsequent responses
must match remembered identity. Не доверять новому имени только из ответа.
Custom role name≤48; no special authority by похожему имени; grants config-only.

Token scopes stored as explicit role-name array; new token defaults configured
agent roles only; human/design-owner require explicit local-admin `--roles` and
owner in auth.local.users. Schema3 grandfather legacy tokens to exact seven
pilot roles, including human (existing policy), event migration_legacy_scopes;
no wildcard means newly added role needs local `token scopes set --id --roles`.
Scopes change audit+CAS; removing live session's role rejects until expiry;
revocation remains immediate and does not silently drop task leases. Local
users are allowlisted token owners, not independent browser/GitHub identity.
Claim and done for human require role scope, owner allowlist, optional members,
category answer_by intersection, complexity≤max. Unknown category or missing
complexity on categorized human task fails; missing complexity on agent stays
unset. design-owner member restriction always applies, cannot be rule-off'ed.

Builtin evidence types: commit/verified_commit=40|64 hex; pr=absolute HTTPS URL;
artifact/screens=array nonempty references≤2048bytes each; summary≤4096,
run_summary≤1024, findings/answer≤16384, run_id≤128, tokens_spent integer≥0.
Minimal preserved floor: developer commit+run_summary; analyst/architect
artifact+summary; human answer; coordinator/devops summary; llm-quality
artifact+summary; design-owner commit+pr. reviewer/security require verdict+
verified_commit+findings; design-reviewer verdict+verified_commit+screens;
tester result+verified_commit+run_summary. davinci adds developer.pr/tester.run_id.
PR existence is checked only when webhook integration is enabled, never invented.
Verifier roles retain exact target SHA/generation and immutable author sessions;
`independent_from` allows target roles, not author-supplied provenance. Extend
review target from developer to configured commit-producing design-owner;
design-reviewer supports both. Custom verifier has nonempty independent_from
plus evidence verified_commit and exactly verdict or result schema; positive
predicate approve/passed; all such roles participate in global latest-assignment
obligations. Other custom roles use completed predicate with their schema.

Mode block rejects; warn succeeds with typed rule_warning+response warnings;
off skips optional rule. Hard floor always: token/role authorization, live
session+generation ownership, one task, DAG, valid evidence types/provenance,
positive exact-SHA gates, independent server sessions, no foreign live release.
Evidence_required warn/off controls extra required fields only; missing extra
evidence marks evidence_complete=false and cannot satisfy dependent gates.
independent_review warn/off can control extra configured independence checks,
never minimum author-session separation. singleton_role warn/off rejected when
any role singleton=true (change singleton flag only after expiry). one_task and
cycle require block. Thus original §10.2 off values explicitly fail
CONFIG_UNSUPPORTED under corrected REQ06/12; no hidden unsafe compatibility.
Planner not auto-added; coordinator remains non-singleton in supplied profiles.

## 6. Workflow, areas, duplicates и зависимости C

Create.ready pointer default true; false allows draft description missing,
but role/epic/title valid. created→ready validates all required fields. New
priority default3 for epic, inherit epic for task; explicit pilot values stay.
Areas omitted inherit snapshot parent areas (explicit [] stays empty);
external_id unique per(type,external_id), optional≤256; category≤128;
complexity low|normal|high, required for categorized human. Update adds these
fields/priority/areas, never role/generation/evidence/state through patch.
Reopen uses active matrix+reason+done evidence_ref, returns_count++ once;
coordinator/human wildcard included explicitly by compatibility init only.

Epic owner may remain empty until first claim CAS in same transaction.
off ignores owner for eligibility; warn partitions own/unowned before foreign,
then task priority, ready_at, epic priority, numeric task ID; foreign claim emits
warning. block excludes foreign unless role singleton/shared_roles. Handoff:
new `transition` state=open + `owner` + expected_version + reason on open epic,
coordinator or local admin, owner must exist. Never transfers task/role leases;
reject while affected epic has live task leases. Normal metadata update cannot
change owner. Explicit next task_key uses same eligibility, never bypass filter.

Area names/aliases: canonical relative POSIX paths; no absolute, backslash,
empty/./../NUL segment, drive prefix; strip one trailing slash; reject alias
duplicates, canonical-vs-alias collisions and aliases resolving multiple roots.
Resolve exact alias first, then canonical prefix children by segment. Store
canonical areas; path parent/child intersects; no FS/symlink access. Declared
roots may overlap. single_editor_role means only that role may claim work
writing intersecting area; authorization always block, not overlap warning.
area_overlap checks active task leases at claim/area-update; warn includes
task/session/owner/areas; block refuses; off skips; release/expiry clears overlap.

Duplicates: Unicode casefold+collapse whitespace titles, set character trigrams
on Unicode codepoints; titles<3 codepoints use whole normalized title as one
gram; Dice≥threshold, same epic/nonterminal tasks, exclude self on update.
Create/update warn returns similar[{key,score}]; block allows force:true only
coordinator+reason, audited override. Off skips similarity, not duplicate link
integrity. duplicate_of source→target requires distinct existing same-kind
objects, reason and cancel authority; source no live lease (epic no live child),
source cancelled+link+event atomically; target unchanged, no merging attempts.
Epic blocks/contract_of use same DAG across task/epic nodes, no containment
cycles (epic→own child fails); evaluator includes terminal children/gates.
All changes reconcile already-done epics transactionally, as existing contract.

C persists created_at/first_claim_at/first_done_at and per-attempt tokens,
reopen/blocked-change event payloads; no backfilled invented timing. Notes
limits resolved from b.Config, not constants; describe returns configured
role notes/rules/current task/queue; next returns epic notes+waiting counts.
Stats uses stored role/owner/epic at event time, deleted-role config history,
unknown fields flagged incomplete; config changes never rewrite past metrics.

## 7. Назначения и обязательные проверки

| Исполнитель | Единственная область записи / условия |
| --- | --- |
| C store owner | core additive DTO, store notes/lists/stats/events; schema2 + backup schema check + migration tests |
| C transport owner | 12-й note tool/REST lists/stats/resource/read summaries; не меняет schema/core параллельно |
| D config owner | internal/config parser/schema/defaults, schema JSON и templates; offline tests; no live mutation |
| D integration owner после C | schema3/config persistence/scopes, board.Config snapshot; core Evidence; all hardcoded predicates/bridge role check; backup/restore/version manifest |
| Reviewer + security | независимый approve правил/auth/migration/data concurrency before apply live |
| Tester после approve | process E2E; затем независимое review новых тестов |

Критерии: invalid/unknown/alias/external-schema inputs keep active config;
two-process apply same expected gives one winner; failed apply leaves no history;
all schemas migrate1→2→3 atomically, newer rejected; backup/restore verifies2/3;
remote scopes cannot mint/admin; new custom role works through real bridge;
live TTL/roles/evidence changes rejected; role-session fencing after singleton
change; human category+complexity checked twice; all warn/off safety bypasses
rejected; latest security/design-reviewer negative blocks across epics; old
approve never bypasses new check; notes limit atomic; historical stats remain
incomplete honestly; reload visible in separate running server without restart.
Оставшиеся E-функции (OAuth/webhooks/import/UI) не считаются этим реализованными.
