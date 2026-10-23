# WORK184 — controlled ProcessType configuration-only comparison

23.10.2026. QA; root координатор, reviewer169. Только private temporary
LaunchAgents и config-only native CLI reads. Модель, MCP tools/call, persistent
worker/config/plist, GUI/source и production server не изменяются.

## Основание

WORK172 actual persistent worker отказал с `prepare_discovery_timeout` за5159ms,
без ownership suffix; pre-thread, task lease/attempt отсутствовали. Cleanup и
GUI baseline сохранены. WORK177 actual Go readNative config-only PASS12 этого
не воспроизвёл. Root нашёл concrete difference: persistent ProcessType Background,
а177 temporary job копировал env/WorkingDirectory без ProcessType, то есть
Standard. Старый177 PASS остаётся честным proof этих ограниченных условий.

Installed man5 и официальный
[Apple launchd.plist source](https://github.com/apple-oss-distributions/launchd/blob/main/man/launchd.plist.5)
подтверждают Background/Standard/Interactive и различия resource policies.
Это hypothesis resource classification, ещё не доказанный root cause.
Persistent read-only runtime proof: `spawn type = background (5)`.

## Source до execution

Private `.local/task013-live/qa/qa184-1792761134226988000/` содержит отдельную
копию363 owning source files; exact SHA manifest независимо сравнен с committed
`62e354525b83a2b671ab37ede938ffe7e838e24b`. Будущая automation implementation
в основном working tree не меняет этот probe. WORK177 source/binary/evidence
не изменяются. Новый production CLI surface отсутствует.

Три one-shot own labels: Background, Standard, Interactive; каждый выполняет
3 discovery/verification samples actual Go readNative/startProcess в GUI cwd.
Все inherited operational keys сохранены: EnvironmentVariables, WorkingDirectory,
ExitTimeOut15 и ThrottleInterval; только ProcessType различается. Own label,
program args, logs и fixed one-shot lifecycle отделены. Верификация использует
inert `/usr/bin/false` binding, никаких MCP/model actions.

Фиксируются UTC timestamps, actual launchctl spawn type/state, top-level exit,
durations и fixed preparation/ownership codes. Background/Interactive runtime
class проверяется; Standard exact observed name сохраняется (launchctl может
называть default class daemon). Три classes должны быть различны runtime.

После reviewer замечания private Go helper fail-fast при
`errors.Is(err,errOwnership)`: записывает stage и прекращает дальнейшие samples.
Runner прекращает все classes на ownership suffix, missing report, timeout или
несогласованном exit/result. Plain timeout с завершённым Stop/Wait допускает
продолжение controlled comparison; ошибки не переименовываются в PASS.
Никаких generic pkill или слепых numeric PID signals. Owned labels cleanup,
GUI baseline, native/persistent/source/runner pins проверяются independently;
finally всегда пытается опубликовать report.

Source fingerprints:

- runner `81517722ead73f17d55a1bf8aa7af57648acbe1e2f2f1cfd8e1f92c69400024f`;
- private Go helper `653b37345971fcbccaba58425bcac2167dc0ecf9b2da3152f9de575e6b0459b5`;
- compiled test `d48cacdef69e547d815080bb28ee84c743d7090f592b25bb0e0e52ec27593182`.

Build/gofmt/Python syntax/source363 pins PASS. Final source передан169;
actual controlled comparison **NOT-RUN** до source APPROVE и root GO.

## Actual execution — FAIL, collection incomplete

Reviewer169 final SOURCE APPROVE и отдельный root GO получены на указанные
fingerprints. Единственный bounded запуск 23.10.2026 начал Background job
в13:18:53 UTC. Actual runtime `spawn type = background (5)` подтверждён;
job позже наблюдался running, но за100s не вывел даже Go-test entry.
Private log остался0bytes, sample report отсутствует. Outer deadline сработал,
runner завершился exit1. **Standard и Interactive NOT-RUN**: fail-closed
остановка всех дальнейших cases соблюдена. Timeout не увеличивался, модель
и MCP tools/call не запускались.

Private report: `.local/task013-live/qa/qa184-1792761134226988000/comparison.json`,
SHA256 `020227282df72ade01848c6e9ef363a0b853bdd4fe4e4acaa0163dca54299450`.
`result=FAIL`, `collection_complete=false`,
`timeout_requires_manual_inspection=true`. Owned labels absent, source363 и
binary/runner/native/config pins unchanged, GUI3316 baseline unchanged,
persistent worker running. Persistent plist/config не изменялись.

До bootout PID/PGID helper не зафиксированы: дополнительный запрос root на
этот census поступил после автоматической cleanup. Read-only fresh OS scan
в13:21:30 UTC не обнаружил exact owned helper executable и config-inspector
`codex ... mcp list --json`; сохранён
`post-cleanup-inspection-private.json`. Эта проверка отсутствия не является
nonce/ownership proof завершения ранее не наблюдавшихся descendant groups.
Manual-inspection flag остаётся; numeric PID signals и generic pkill не
использовались. Cleanup собственного label выполнен штатным bootout.

Это свидетельство задержки уже до helper test entry в Background context,
а не доказанный native discovery root cause и не сравнение трёх policies.
WORK177 PASS остаётся отдельным историческим результатом для его Standard
context. Дальнейшие contexts требуют отдельного review/root GO.
