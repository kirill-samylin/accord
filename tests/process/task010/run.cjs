'use strict';
const h = require('./harness.cjs');
const {fs, path, assert, root, binary, oldBinary, sha, save, admin, legacyToken, catalogToken, freePorts, startServer, stop, BrowserSession, Native, python} = h;
const catalog = require('./catalog.cjs');
const testFiles = () => Object.fromEntries([__dirname, path.join(__dirname, '../../browser/task010')].flatMap(d => fs.readdirSync(d).filter(n => /\.(cjs|py)$/.test(n)).map(n => [path.relative(h.repo, path.join(d, n)), sha(fs.readFileSync(path.join(d, n)))])));
const sourceTests = testFiles();
const baselineProof = process.env.ACCORD_TASK010_OLD_SHA256 ? {binary_sha256: process.env.ACCORD_TASK010_OLD_SHA256, source_revision: process.env.ACCORD_TASK010_OLD_REVISION} : JSON.parse(fs.readFileSync(path.join(h.repo, '.local/task010/legacy/proof.json')));
assert.equal(baselineProof.source_revision, '5454eb5c98ac088e5ead3daaaad637423112ddad', 'trusted baseline revision required');
assert.equal(sha(fs.readFileSync(oldBinary)), baselineProof.binary_sha256, 'actual baseline executable differs from trusted build proof');
const report = {schema: 1, started_at: new Date().toISOString(), node: process.version, platform: process.platform,
  source_revision: process.env.ACCORD_TASK010_REVISION || null, source_dirty: process.env.ACCORD_TASK010_SOURCE_DIRTY || 'not-inspected-by-tester',
  binary_sha256: sha(fs.readFileSync(binary)), old_binary_sha256: sha(fs.readFileSync(oldBinary)), old_source_revision: baselineProof.source_revision, checks: [],
  scope: process.env.ACCORD_TASK010_BROWSER === '0' ? 'actual-process-only' : 'actual-process-and-Chrome',
  limitations: ['one host; physical second PC NOT-RUN', 'HTTP loopback; production HTTPS separate', 'provider/worker execution not implemented']};
const pass = (id, extra = {}) => {report.checks.push({id, result: 'PASS', ...extra}); console.log(id + ' PASS');};
let srv;
function preserved(before, after) {
  for (const [name, tables] of Object.entries(before.databases)) for (const [table, rows] of Object.entries(tables)) assert.deepEqual(after.databases[name][table], rows, 'offline migration changed existing table ' + name + '/' + table);
  assert.deepEqual(after.blobs, before.blobs);
}
function restoredData(before, after) {
  assert.deepEqual(after.blobs, before.blobs); assert.deepEqual(after.repositories, before.repositories);
  for (const [id, board] of Object.entries(before.boards)) {
    assert.deepEqual(after.boards[id].task_history, board.task_history);
    assert.deepEqual(after.boards[id].notes, board.notes);
    assert.deepEqual(after.boards[id].idempotency, board.idempotency);
    for (const row of board.events) assert(after.boards[id].events.includes(row), 'restore discarded historical event');
  }
}
async function main() {
  await freePorts();
  const state = path.join(root, 'state'); fs.mkdirSync(state, {mode: 0o700});
  const ownerToken = legacyToken(state, 'QA-owner', oldBinary);
  const workerToken = legacyToken(state, 'QA-old-worker', oldBinary);
  srv = await startServer(state, {legacy: true, bin: oldBinary});
  await stop(srv); srv = null;
  admin(state, 'bootstrap-owner', ['--identity', ownerToken.id], oldBinary);
  const editor = catalogToken(state, 'QA-editor', oldBinary), viewer = catalogToken(state, 'QA-viewer', oldBinary);
  srv = await startServer(state, {bin: oldBinary});
  const owner = await new BrowserSession().login(ownerToken);
  const defaultId = (await owner.catalog()).registry.projects[0].id;
  const a = await catalog.create(owner, 'QA Alpha'), b = await catalog.create(owner, 'QA Beta');
  await catalog.member(owner, a, editor.id, 'editor'); await catalog.member(owner, a, viewer.id, 'viewer');
  const oldNative = new Native(ownerToken, a, oldBinary); oldNative.open('coordinator');
  const epic = oldNative.call('create', {type: 'epic', title: 'QA preserved epic', description: 'Synthetic only'});
  const task = oldNative.call('create', {type: 'task', epic_key: epic.key, role: 'developer', title: 'QA preserved task', description: 'Synthetic only'});
  // Legacy credential registry identity differs from token ID; use catalog editor
  // for the lease while retaining an old default legacy session separately.
  oldNative.call('note', {action: 'create', scope_type: 'epic', scope_id: epic.key, source_task_key: task.key, text: 'QA preserved old note'});
  const oldWorker = new Native(editor, a, oldBinary); oldWorker.open('developer');
  const claim = oldWorker.call('next', {task_key: task.key}); assert(claim.lease);
  const unbound = new Native(workerToken, '', oldBinary); const unboundView = unbound.open('developer'); assert(!unboundView.profile);
  const asset = await catalog.upload(owner, a, 'QA preserved design'); assert(asset);
  await owner.open(a); const oldPage = await owner.read('/web/p/' + a + '/profiles');
  // TASK009 has no profiles route; the ordinary agents page captures its binding.
  assert.equal(oldPage.status, 404);
  const oldSession = (await owner.read('/web/p/' + a + '/agents')).data.page.session.session_id;
  const oldCookie = owner.cookie;
  await stop(srv); srv = null;
  const before = python('capture', state);
  assert(Object.entries(before.schemas).filter(([name]) => name.endsWith('/board')).every(([, version]) => version === 2));
  const immutable2 = python('immutable', state);
  const snapshot2 = python('snapshot', state, {name: 'schema2-backup'});
  assert.equal(snapshot2.board_schema, 2);
  const migration = JSON.parse(admin(state, 'migrate-storage').stdout); assert.deepEqual(migration, {storage_migrated: true, boards: 3});
  const after = python('capture', state); preserved(before, after);
  assert(Object.entries(after.schemas).filter(([name]) => name.endsWith('/board')).every(([, version]) => version === 3));
  assert.deepEqual(JSON.parse(admin(state, 'migrate-storage').stdout), migration);
  preserved(after, python('capture', state));
  pass('actual5454-schema2-three-project-offline-migrate3-all-table-blob-byte-preservation-and-idempotent-repeat');
  const restored2 = python('restore', state, {name: 'schema2-restored', snapshot: snapshot2.snapshot, binary});
  assert(restored2.boot_guard && restored2.reset_complete); restoredData(immutable2, restored2.immutable);
  assert.equal(restored2.inspect.live_catalog_credentials, 0);
  assert(restored2.inspect.boards.every(b => b.live_tokens === 0 && b.browsers === 0));
  pass('actual-old-schema2-full-format2-restore-migration-authreset-keeps-offline-guard');
  const f = {state, ownerToken, owner, defaultId, a, b, editor, viewer, oldWorker, aTask: task.key, asset};
  srv = await startServer(state);
  assert.equal(owner.cookie, oldCookie);
  const page = await owner.read('/web/p/' + a + '/profiles'); assert.equal(page.status, 200); assert.equal(page.data.page.session.session_id, oldSession);
  assert.equal(oldWorker.call('get', {task_key: task.key}).lease.generation, claim.lease.generation);
  const hb = unbound.call('heartbeat', {}); assert(!hb.profile); assert.equal(hb.session_id, unboundView.session_id);
  const downloaded = await fetch(h.goOrigin + asset.download_url, {headers: {Cookie: owner.cookie}}); assert.equal(downloaded.status, 200); assert.equal(sha(Buffer.from(await downloaded.arrayBuffer())), asset.sha256);
  pass('postmigration-existing-cookie-native-session-unbound-session-livelease-design-download-preserved');
  await require('./native.cjs').run(f, pass);
  await require('./web.cjs').run(f, pass);
  if (process.env.ACCORD_TASK010_BROWSER !== '0') await require('../../browser/task010/browser.cjs').run(f, pass);
  await stop(srv); srv = null;
  const immutable3 = python('immutable', state);
  const snapshot3 = python('snapshot', state, {name: 'schema3-backup'}); assert.equal(snapshot3.board_schema, 3);
  const profileBefore = python('capture', state).profiles;
  const restored3 = python('restore', state, {name: 'schema3-restored', snapshot: snapshot3.snapshot, binary});
  restoredData(immutable3, restored3.immutable);
  assert.deepEqual(restored3.capture.profiles, profileBefore); assert(restored3.boot_guard && restored3.reset_complete);
  assert.equal(restored3.inspect.live_catalog_credentials, 0); assert(restored3.inspect.boards.every(b => b.live_tokens === 0 && b.browsers === 0));
  assert.deepEqual(python('guard', state, {old: oldBinary, new: binary}), {old: 'denied', new: 'allowed'});
  pass('actual-schema3-fullbackup-restore-keeps-profile-revisions-and-denies-old-capability');
  const damaged = python('corrupt', state, {name: 'damaged-schema3'});
  const failed = h.cli(['serve', '--db', path.join(damaged.state, 'board.db'), '--web-data', path.join(damaged.state, 'web'), '--web-origin', h.origin, '--projects-data', path.join(damaged.state, 'projects-data'), '--listen', '127.0.0.1:3101'], undefined, binary, true);
  assert.notEqual(failed.status, 0); assert(!failed.stdout.includes('listening'));
  assert.deepEqual(python('blobs', damaged.state), damaged.blobs);
  assert.deepEqual(python('profile_tables', damaged.state), ['agent_profiles'], 'startup repaired missing revision authority');
  pass('incomplete-published-schema3-startup-failclosed-no-oldblob-deletion');
  save(path.join(root, 'fixture-private.json'), {state, a, b, defaultId, ownerToken, editor, viewer});
  report.result = 'PASS';
}
main().catch(error => {report.result = 'FAIL'; report.failure = String(error.message).slice(0, 250); console.error(report.failure); process.exitCode = 1;}).finally(async () => {
  try {await stop(srv);} catch {report.result = 'FAIL'; report.cleanup_failure = true; process.exitCode = 1;}
  report.tests = sourceTests; report.finished_at = new Date().toISOString();
  if (JSON.stringify(testFiles()) !== JSON.stringify(sourceTests)) {report.result = 'FAIL'; report.tests_changed = true; process.exitCode = 1;}
  save(path.join(root, 'results.json'), report); console.log('private evidence: ' + path.relative(h.repo, root));
});
