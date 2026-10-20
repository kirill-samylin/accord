# WORK-151 — координация: operations и интеграция

20.10.2026. Root. Scope: infra/accord/projects, tests/infra, канонические docs,
Git/release/VPS integration. Backend148 и frontend149 имеют отдельное владение;
review150 независим. Контракт принят по WORK147 в docs/coordination.md.

## План/границы

Schema4/coordination1 floor до DDL; full snapshots2/3/4 и исторические1/2/3
migration copies/checksums; additive logical preservation всех старых tables,
runtime/systemd guards прежние. Initial TASK009 apply не повторять, DB rollback
запрещён. Authority marker UID/mode/bytes не менять. Реальные данные не скачивать.

Статус: реализация оператора IN PROGRESS, source review/QA/Linux/VPS NOT-RUN.
Исходная production версия cbf2020 только передана; до поставки нужна проверка.

Объём: guard требует coordination1 и schema4; layout/validation принимает полные
2/3/4; migration history включает strict schema3 files; upgrade_storage проверяет4.
Новые infra unit для snapshots/metadata/history и отказа прежнему schema3 binary.

## Промежуточная проверка

Root targeted storage14/guard7 и fullinfra155 PASS; independent150 повторил155
и подтвердил slice APPROVE. SQLite ResourceWarnings прежних fixtures не FAIL.
Новые7coordination unit используют реальные SQLite snapshots/history/copy; native
maintenance-reset только mocked в offline orchestration unit, это не process PASS.

Readonly production preflight PASS: actual currentcbf2020 Go/Next pointers, journal
complete, installed schema3 guard/runtime/effective config, validated current full
backup, markers0, services/timer active. Чтение не изменяло pointers/данные/guards.
Private baseline `.local/task011/deploy/before-inventory-private.json`; реальные
DB/credentials не копировались. Schema4/release/поставка пока NOT-RUN.

Для actual3→4 QA собран clean macOS schema3 baseline из343b93c в отдельном
managed coordination-baseline checkout. Go VCS revision/modified=false и
capabilities проверены, SHA4e11d2569120dada3df15061bcfc5193aacb7931a048110f27242f0b9419e085.
Executable/proof сохранены private `.local/task011/legacy`; временный checkout
архивирован после сборки. Это baseline fixture, не новая production поставка.

## Source freeze

WORK150 общий SOURCE APPROVE; root independently matches74files manifest
5cd77400d904fe72783b3beabda63b1fa3a4137ddfc914e651a3347548cc672a.
Полный root make check PASS (Go/all/race/vet/build, frontend, infra155,
Next67/types/format/build), log `.local/task011/full-check-source.log`.
Tester152 теперь разрешено писать/запускать новые process/E2E. Source сохраняется
до QA findings; релиз/production пока NOT-RUN.

## Диагностический cache incident

Первый readonly remote inventory использовал прежний transport.py без remote -B
и создал10 root-owned CPython caches в admin/{projects,next,web}/__pycache__
в одном окне19:56:52UTC. Это операционная ошибка root, не изменение product/DB.
До stage обнаружено отдельным audit. Каждый файл сверён по exact timestamp/UID/
regular mode/SHA, CPython magic и compiled code against installed source; каждый
source SHA совпал с completedcbf journal. Только эти известные caches перемещены
с сохранением bytes+audit в root-only
`/root/accord-task011-preflight-python-cache-20261020T195652Z`. Unknown recovery/
temp artifacts не трогались. После перемещения reject_compiled_helpers,
previous release require_helpers(after=True) и require_runtime PASS.
Private cache-audit-private.json/cache-remediation-proof.json сохранены в deploy.
Все task-local transport wrappers теперь запускают remote Python -B. Общий
исторический transport и product/infra source не редактировались.

Detailed readonly archive proof `cache-archive-detailed-proof.json` включает для
каждого10files исходный/архивный SHA+mtime/UID/mode/regular, installedsource SHA
vscompletedcbf helperhash и runtime magic/header/compiled-code equivalence. Archive
root0700/exact10+audit inventory и canonical cache absence проверены. Independent150
сам сверил10source SHA через git showcbf и exactbefore/afterset: incident evidence
APPROVE. Detailed proof SHA256
`f649652a1e6c490d7e2c20d80b6baf7c1f80346040dfbde50a7ededc275a1fac`.

## Immutable release и Linux

Из clean managed checkout e53c18c после test/localQA APPROVE собраны GoLinux
(trimpath,CGO0,VCSmodified=false) и productionNext upstream7892. Source74manifest
совпал с approved. Proof/release outputs сохранены `.local/task011/{deploy,releases}`.
GoSHAa03f64d5ff008b7f63023117fb7b1c15c33e5617ce7ac08ffc6440610315269d,
NextBuildIDHHViYVPwDuQx7uj4Uk1YC, manifestcb97c3dd40df2aa8340dd0e3ca2573af4d44a3f956c8498734efadb978b212c9.
Transferred53Go/1256Next exactregularmembers/root-owned withoutactivechange.
Frozen463committed QA/source files перенесены в отдельные root-only Linux fixtures:
actualproduct13/13 и updater8/8 PASS, newbinaryhash/oldcbf/source/testsunchanged и
ports3100/3101free послеchildcleanup. Simulatedsystemd/Next/UID вupdater остаются
simulated; prodпути/службы fixtures неиспользовали.

Actualstage through reviewed upgrade.py PASS: actualhardenedNode/sharp/ownedPID,
NRestarts0/MemoryPeak46825472; temporaryunitremoved7894free; oldcbfpointers
unchanged, stagedjournal/helpers/effectiveconfig/markers0 PASS.
Independent exactLinux/stageevidencereview доapply ожидается; production schema3.

## Actual apply/HTTPS/finalize

После independent150 exactLinux/stageAPPROVE выполнены reviewed capable apply и
finalize. Fresh pre-snapshot schema3/web2 default/registry/история сохранил3DB;
logicalrows preservedtrue, publishedschema4. Реальный guardbefore dc...41be
заменён на c1703c392fc62891e150e23e52cdc68132f05f1c5dad570c53404f4de07b50ed,
совпадающий сnewrelease helper; old/newSHA привязаны apply-preservation-proof.json.
Authoritymarker UID999/0600/regular/bytes==preSnapshot, effectivelaunch/backup/
helpers/ownedGoPID+ELFSHA/NextBuildID/cap4 PASS. Journalcomplete, markers0,7894free,
Go/Next/nginxNRestarts0, services/timersactive. Userdata не скачивались/сбрасывались.

Approved readonlyHTTPS script5cb...8395 выполнен без измененияsource: PASS7/7,
56screens/858browser200/errors0/prefetch0, existing2tasks real coordination read,
собственные login/open/logout/browserclosed; nobusinessmutations. ReportSHA
06aa3b53658adf869192d83f6a0c19dd06c92bfa3233cd8e2154d39fc59603a2.
Final format2 snapshot `/var/backups/accord-web/projects-20261020T204623Z-d90daa53b0614bfab1cdcbf041a9abca`
validated default4/web2+registry/history (12files,0extraProjects/3realDB), manifest
28b9886fd12e53d0559091d94b7771346a86b016e500ac46b1d96c31fc600a39. Pre-snapshot
204410Z-98c7caa68b0e4312a5db97aaa09301cc validated/hash a4e066...d013.
Fullvalues/paths вTASK011 иprivateproofs. Final independent150review: FINAL APPROVE. Обязательных замечаний нет.

Clean source/release managed checkout coordination-candidate архивирован после
успешной поставки; все нужные ignored Go/Next bundles/logs/proofs сохранены в
primary `.local/task011/releases/e53c18c...` и deploy. Baseline checkout ранее
также архивирован, его executable/proof сохранены отдельно вlegacy.
