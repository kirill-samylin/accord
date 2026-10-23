# WORK204 — узкие native Git permissions

23.10.2026. Автор `/root/automation_worker_impl`. Основание — WORK201 и
CONTRACT APPROVE202, затем явный implementation GO root. SOURCE READY/FROZEN
для независимого review202; runtime/production не принимаются этим документом.
Base candidate: `3e6ac81`. Владение: прежний193 worker и новые worker unit files;
root-owned HTTP tests, WORK205 preview, TASK/201/202, GUI и production не менялись.

## Реализовано

- Default-off local `codex_git_metadata_writes` применяется только к automation
  developer/Codex. Нужны отдельно заданный workspace-write и local developer role.
  Manual/Claude/analyst/reviewer/tester сохраняют прежний путь. Флаг включён в
  preset digest; false через omitempty сохраняет прежние digest bytes. True
  без нового local acknowledgement не проходит validation; host consent и
  server policy обновляются обычным явным порядком, regrant не автоматизирован.
- Генерируется named profile extends `:workspace`, write только для exact
  canonical checkout/.git, read для config/hooks, command network false и
  approval policy never. Нет legacy sandbox override в этом режиме. Остальные
  native режимы сохраняют прежний sandbox. Gitdir file, linked common directory,
  symlink Gitdir/config/hooks отклоняются. Унаследованные temp writes сохраняются;
  это filesystem grant, не whitelist только команды commit.
- Существующий managed-MCP preflight сохраняется. Из него в private runtime
  копируются только connector names/transports. Обе config inspections используют
  whole-map disabled MCP overrides; effective map и все active-layer connector
  names сверяются. Новых несупервизируемых discovery calls не добавлено.
- Закрытый config-only dialogue выполняется внутри supervisor. Child stdin pipe
  отделён от parent lifeline. Только initialize id1 → ответ → initialized →
  config/read id2 с exact cwd/includeLayers → ответ → requirements/read id3 →
  ответ → child stdin EOF → natural owned exit0. Нет arbitrary WriteStdin API,
  thread/model/login/tool/config-write calls, daemon/proxy/socket listener.
- Fixed framing: requests≤8KiB, line≤2MiB, total stdout≤4MiB, максимум35 frames
  (3 responses+32 configWarning), 20s dialogue включая EOF/exit. Parent deadline
  ограничен остающимся attempt deadline. Stalled reads/writes и cancellation
  получают bounded owned cleanup reserve≤7s. Unknown stop/reap остаётся
  errOwnership/needs_recovery с checkout intent; нулевой exit не выдумывается.
- JSON duplicate keys, batch, unknown/duplicate/out-of-order IDs, result+error,
  RPC errors, malformed/oversized/truncated frames и server requests запрещены.
  Notifications разрешены только configWarning и не выводятся. Raw config,
  layers, stderr и server errors не попадают в provider output/server/journal;
  helper выдаёт только safe digest, executable hash, Git identity и nonce exit
  proof. Windows использует helper внутри owned Job; runtime Windows NOT-RUN.
- Parser требует actual effective selected profile со всеми exact grants,
  effective approval never, disabled connector map и читаемые active layers.
  Active explicit legacy settings, unknown permission fields/sources, встроенный
  profile override, дополнительные selected-profile roots/network/parent/grants,
  противоречащие/неподдерживаемые managed constraints блокируют режим. Nullable
  typed sandbox_mode:null не считается explicit legacy key. Если CLI скрывает
  additional-properties permission tree, результат blocked, а не inferred PASS.
- Две inspections после существующего MCP preflight сравнивают permission-
  relevant digest перед native spawn. User trust metadata и file mtime/version
  сами по себе не дают permission drift. Environment snapshot общий с native;
  executable SHA/Gitdir identity, authority, consent, epoch, path pins и deadline
  повторно проверяются. Только безопасный proof сохраняется в private attempt.
  Unsupported proof даёт фиксированный local diagnostic без fallback.

## Файлы

Новые product files: git_permissions, permission_projection,
permission_inspection, config_dialogue, supervised_proof, inspection_windows.
Последний отделяет Windows helper proof от raw provider output; общий
proofReader перенесён из Unix без изменения legacy nonce semantics.

Изменены automation_config/launch/state, engine, native_consent,
bridge_config/preparation и process/supervisor платформенные файлы. Новые
git_permissions/config_dialogue/permission_preparation/permission_launch tests
используют filesystem fixtures и mock processes/io.Pipe, без OS child/model/RPC.
Все23 Go files scope≤300 строк. Зависимости не обновлялись.

## Проверки и pin

- PASS: `go test -race ./internal/worker ./internal/automationipc ./cmd/agent-board`
  с canonical private TMPDIR: worker13.155s, IPC1.398s, cmd3.105s.
- PASS: default false legacy digest compatibility; true changes digest; role/
  manual/provider separation; exact profile и отказ external Gitdir; hidden/
  broadened grants, legacy/unknown layers и managed contradictions; trust-only
  metadata stability; fixed interactive dialogue/no secret publication; malformed
  frames/unknown requests/IDs/limits; stalled input/cancelled no-spawn; uncertain
  stop quarantine+retained lock; authority/epoch/digest TOCTOU; actual launch
  preparation chain на mocks без legacy fallback/late spawn.
- PASS: Linux amd64 и Windows amd64 worker test cross-build; diff-check.
- NOT-RUN: actual app-server config RPC/effective shape/EOF behavior, native
  single commit, OS-enforced negative sentinels, Windows physical, полный
  workflow/control/UI и production. Новые process/E2E пишет отдельный QA после
  независимого SOURCE APPROVE.

Private source manifest: `.local/work204/source-manifest.json`, 23 Go files,
sorted relative path + NUL + SHA256 + newline:
`eea3f68a765ec122278cda976774493cbc0859e5ea22ee947f3f98cce53c9714`.
В manifest только204 scope, root205/HTTP files исключены. Это fingerprint, не
Git commit или доказательство установки.

## Оставшиеся границы

Config proof подтверждает configured grants; OS enforcement отдельно проверяет
QA. CLI app-server может иметь startup side effects: их отсутствие не заявляется
по исходникам worker. Actual QA нужен с private HOME и before/after hashes; прежний
user-config drift не реконструирован и не восстанавливался. Profiles beta и
поддерживаемость installed CLI/OS определяются config-only и negative runtime
проверкой. Same-user ownership/trust и Unix detached-descendant ограничения
остаются прежними. Нет approval review/fullaccess/global config edits, push,
merge, deploy или обхода native отказа.


## Review202 P1 — effective legacy settings

Исправлено: ненулевые sandbox_mode/sandbox_workspace_write в effective Config
также блокируют profile, даже если explicit originating layer не возвращён.
Nullable typed defaults остаются допустимыми; неоднозначное ненулевое значение
не объявляется безопасным derived default до actual config-only QA.

Regression отдельно клонирует только effective map, сохраняя clean layers без
алиасов, и проверяет оба legacy поля, empty-layers refusal и null positive.
Targeted race PASS1.612s. Предыдущий pin82b121e… заменён текущим23-file pin выше;
source/runtime границы прежние. Root WORK205 files в scope не включены.
