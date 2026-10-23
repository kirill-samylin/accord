# WORK195 — real GUI native control acceptance

23.10.2026. Отдельный QA, root координатор, reviewer169. Root exact GO на один
повтор уже169-approved frozen adapter490d; source/config не менялись.
Исторические WORK172 FAIL и config-only184/188/190/192 evidence сохранены.

## Проверенная версия и scope

Production server273ccbb15c99dfac91c37f34d2926abf01ca866f, persistent Mac
worker62e354525b83a2b671ab37ede938ffe7e838e24b / SHA10e845ff702e2b1f9d9bcac4ac8cdd278c4aba65e5fd839104dd0cc822c6d429.
Frozen adapter `.local/task013-live/qa/qa172-diagnostic-62e3545/adapter-execute.cjs`
SHA490d13fa844fd3200df4800b320803710658bc411a6785ef7cfc33a608ccb61f.
Перед runtime PASS strict hashes adapter/binary/config/plist и GUI3316 baseline;
authenticated host idle/two-controller checks выполнены до QA profile/task/run.

Использован реальный persistent LaunchAgent и Codex, GUI checkout davinci-gui-;
не fixture worker. Сохранён user-consented workspace-write/exact10-tool config.
QA instructions read-only, native README read/hash, существующие dirty files
сохраняются. Никакие permissions/auth/provider/sandbox settings не менялись.
Временные QA profile/task/epic отдельные; постоянный developer profile не менялся.

## Actual PASS5, exit0

Initial turn40.802s; весь native run до final completed94.799s (backend timestamps).

- HTTPS UI: два отдельных principals на этом Mac имеют явный control доступ к
  постоянному host; нет смешения с editor-only OS permission.
- Owner UI Start: native Codex выполнил actual Describe/Next, exact QA task claim,
  чтение README и фактический SHA256 marker, completed с proven terminal handoff.
- Owner UI Resume longer read-only turn: same native UUID, generation2 running.
  Colleague UI сохранил уточнение queued и отправил Stop; actual applied/proven
  stopped receipt, provider UUID сохранён, task lease отсутствует.
- Owner UI Resume generation3: тот же exact native UUID, queued clarification
  applied; semantic README hash/clarification/resumed markers подтверждены.
  Новый board session и task fence выше прежнего, completed terminal handoff.
- HTTPS UI без page errors, GUI3316 file baseline неизменен; снимок managed page
  сохранён приватно без output-detail.

Readback: run generations1→2→3, одна native conversation; две task claim attempts
fences1→2, distinct board sessions, обе ended/managed_completed, lease отсутствует.
Generation2 native stop произошёл до дополнительного task claim; третью task
lease не заявляем. Worker journal ownrun completed/gen3, receipts started/stopped
gen2 и completed gen1/3, pending reports0; prior journal runs сохранены.
Local diagnostic ownrun пустой. Worker state/epoch не менялись вручную.

## Cleanup и evidence

QA task/epic cancelled с технической причиной, не fake done; QA profile archived.
GUI baseline, binary/config/adapter hashes unchanged; persistent LaunchAgent
top-level running. Browser закрыт, cleanup failures нет. Native GUI run не
повторялся после успешного единственного запуска.

Fresh private evidence `.local/task013-live/qa/live-gui-1792765508578/`:

- `results.json` SHA3eae9067f6fbae162660d5e6538a20a34bf65b04c2098b307abd230f2dfbaec9;
- executed frozen copy и `qa195-provenance.json`;
- initial/stopped/final run, final task и cleanup run private readbacks;
- `qa195-safe-runtime-proof.json` SHAbcc24865e87e5d9b863eb34a410e08a462b95fc52cb6cb6ce96ce3a4baaf108f;
- before/after GUI baseline и `live-managed-private.png`.

Это actual Mac Codex + HTTPS GUI workflow PASS. Два browser principals на одном
Mac не являются двумя физическими ПК. Native Windows/Claude/second-PC control
**NOT-RUN**. Причина прежних intermittent native preparation failures этим
успехом не доказана; historical failures остаются. TASK013/TASK014 closure
и production decisions ведёт root, QA не объявляет весь новый automation scope
завершённым.
