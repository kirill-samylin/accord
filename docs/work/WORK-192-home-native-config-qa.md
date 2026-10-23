# WORK192 — HOME artifact location config-only comparison

23.10.2026. Отдельный tester; root координатор, reviewer169. PLAN APPROVE169
получен; новые sources подготовлены, actual execution **NOT-RUN** до SOURCE
APPROVE и отдельного exact root GO.

## Граница эксперимента

WORK190 Standard actual daemon3 не дошёл до Go test entry за100s, несмотря на
stable PID/UID/PGID и опубликованный gate. Binary/spec/gates/logs находились
под Desktop. Root предложил изменить одну ось: расположение этих runtime
artifacts, без предположения о доказанном TCC/ProcessType root cause.
WORK190/188/184 failures/manual flags не меняются.

Новый fresh root:
`/Users/shagi/.config/accord-gui/diagnostics/qa192-1792762953184003000/`.
Binary, specs, gate и logs только в HOME diagnostics; source copy/manifest
также здесь. Root private0700, files private; runner проверяет canonical path
и UID501. `shutil.copy2` перенёс прежний helper без rebuild или warmup;
содержимое exact SHA81b182 сохранено. Xattr/privacy/quarantine/TCC settings
не редактируются, GUI не перемещается.

Current persistent `WorkingDirectory=/Users/shagi` сохранён и проверяется
explicitly. EnvironmentVariables/ExitTimeOut15/ThrottleInterval прежние.
Native helper hardcoded GUI cwd по-прежнему Desktop/davinci-gui-; это отдельная
оставшаяся граница доступа, а не вынесенный из Desktop checkout.

Запланирован Standard первым, Interactive только при полном Standard PASS
и отсутствии ownership ambiguity. Любой configuration phase FAIL теперь
немедленно прекращает дальнейшие contexts; gate/outer timeout тоже fail-stop.
Все прежние20s helper gate/25s runner gate/100s outer deadlines, stable2
samples, exact percontext256bit nonce/self PID UID PGID checks и строгая
postgate family tracking не меняются. Native5s/Wait/PGID guards сохранены.
Модель, native exec/resume и MCP tools/call запрещены; только config list.

Own labels новые qa192, lifecycle one-shot, без persistent изменений.
Prebootout/postabsence/pins/GUI3316 baseline/current worker checks и finally
report сохранены. Birth/observed-family ограничения прежние. Дополнительный
TCC log extractor в эту проверку не включён; отсутствие OS diagnostics не
будет выдаваться за PASS или доказанную причину.

## Source-ready fingerprints

- `run_classes.py`221 строк — `a93b7cad06ac3f9218a6671142a0129cad7dd8b071180c0d14ab6cee78955b7c`;
- unchanged `process_inventory.py` — `969ddbc8c5d6bfcc73127cfc188312919c428165f9375c7cd2f2f8a8b12119b5`;
- unchanged private Go test — `75ae54889c15f123405005da270503d4243504a1b9f6c7344bb169096a0fe912`;
- exact copied binary `qa190-worker.test` — `81b1827610510bb1804def5a0bb20194393faa92258f049d03aadbe833a9e1fd`.

Owning source363 pins к committed62e354525b83a2b671ab37ede938ffe7e838e24b,
AST и unchanged copied binary verification PASS. Wrapper delta к190 прочитан.
Build/helper/native/model execution NOT-RUN, LaunchAgents не создавались.
GUI/production/постоянный worker/config/plist не изменялись.

## Actual: Standard phases PASS6; overall FAIL/incomplete

169 SOURCE APPROVE и exact root GO получены на указанные fingerprints.
Единственный запуск 23.10.2026 в13:47:27 UTC: actual Standard spawn daemon3,
stable2 samples UID501/PID=PGID52461, gate published в13:47:28 UTC. Helper
действительно вошёл в Go test и подтвердил `startup_gate=accepted`.
Все3 discovery и3 exact-override verification phases **PASS**:

- discovery1956ms → verification46ms;
- discovery33ms → verification33ms;
- discovery36ms → verification34ms.

Private `standard/launchd-result.json` имеет result PASS. Модель/native
exec/resume/MCP tools/call не запускались. Это config-only runtime proof.

Общий runner **FAIL / collection incomplete**: сразу после готового phase
report, в13:47:30.404 UTC, OS census того же PID/UID/PGID/starttime показал
comm `(qa190-worker.tes)` вместо exact executable path. Strict postgate
assertion завершила сценарий; exit-state hypothesis не подменяет ownership
proof. Runtime состояния процесса в этом census не сохранялись, поэтому
подтверждённый zombie classification не заявляется. **Interactive NOT-RUN**,
автоматических повторов или ослабления assertion нет.

Prebootout13:47:30.442 UTC top/helper и observed family уже отсутствовали;
postbootout13:47:30.487 UTC это подтвердил. Own labels absent, all9 pins/source363
и GUI3316 baseline unchanged, persistent worker running, cleanup failures нет.
Manual-inspection flag сохраняется из-за assertion. Numeric PID signals не было.

`comparison.json` в HOME root имеет SHA256
`7c7210da37d6b12c8cc35e475414a2606454b732dac333dcf37d47f053c2d288`.
Фактические phases и process snapshots сохранены рядом. Старые184/188/190FAIL
сохраняются. Сравнение одного и того же81b182 binary из Desktop и HOME сужает
artifact-location axis, но не доказывает TCC или другой конкретный root cause,
не проверяет Background в HOME и не является native GUI workflow acceptance.
