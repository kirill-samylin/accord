'use strict';
const h = require('./harness.cjs');
const {assert, path, root, binary, oldBinary, fs, cp, env, python, cli, admin, catalogToken, BrowserSession, Native} = h;
async function run(f, pass, runtime) {
  await runtime.start(f.state);
  const catalog = await f.owner.catalog();
  const action = catalog.registry.actions.create;
  await runtime.stop();
  const published = python('inspect', f.state).projects.filter(([, state]) => state !== 'provisioning').map(([id]) => id);
  const pending = python('pending', f.state, {action, identity: f.ownerIdentity});
  const pendingHashes = Object.fromEntries(['board.db', 'board.db-journal'].map(name => [name,
    h.sha(fs.readFileSync(path.join(f.state, 'projects-data/projects', pending.id, name)))]));
  const output = cp.spawnSync(process.env.ACCORD_TASK009_PYTHON || 'python3', [path.join(__dirname, 'lifecycle.py'), f.state, root, binary], {env, encoding: 'utf8', timeout: 120000});
  assert.equal(output.status, 0, 'actual full snapshot/restore failed (private output suppressed)');
  const restored = JSON.parse(output.stdout);
  assert(restored.projects >= 3); assert(restored.quarantined >= 1);
  for (const [name, digest] of Object.entries(pendingHashes)) assert.equal(h.sha(fs.readFileSync(path.join(restored.restored, 'projects-data/quarantine', pending.id, name))), digest, 'restore changed unpublished recovery bytes');
  pass('actual-full-registry-multi-board-pending-snapshot-offline-restore-auth-reset', {projects: restored.projects, quarantined: restored.quarantined});
  // The real old executable is evaluated by the installed guard contract, never
  // launched as a server over a registry whose ACL it cannot enforce.
  for (const missingMarker of [false, true]) {
    const marker = path.join(f.state, 'projects-data/MULTIPROJECT_REQUIRES_CAPABILITY');
    if (missingMarker) fs.renameSync(marker, marker + '.qa-held');
    try {
      const check = cp.spawnSync(process.env.ACCORD_TASK009_PYTHON || 'python3', [path.join(h.repo, 'infra/accord/projects/guard.py'), '--binary', oldBinary, '--projects-data', path.join(f.state, 'projects-data')], {env, encoding: 'utf8', timeout: 10000});
      assert.notEqual(check.status, 0, 'old actual binary passed ACL capability guard');
    } finally {if (missingMarker) fs.renameSync(marker + '.qa-held', marker);}
  }
  pass('actual-legacy-binary-downgrade-denied-with-and-without-marker');
  // Quarantine preserves the bytes seen by the offline operator. Do not first
  // ask SQLite to recover a synthetic stale journal and then require that stale
  // journal still exists; that would test an unsupported preservation promise.
  admin(f.state, 'reject-provisioning', ['--project', pending.id, '--reason', 'provisioning-recovery']);
  assert(fs.existsSync(path.join(f.state, 'projects-data/quarantine', pending.id, 'board.db-journal')));
  for (const [name, digest] of Object.entries(pendingHashes)) assert.equal(h.sha(fs.readFileSync(path.join(f.state, 'projects-data/quarantine', pending.id, name))), digest, 'ordinary rejection changed unpublished recovery bytes');
  await runtime.start(f.state);
  assert.equal((await f.owner.action(catalog, 'create', '/web/catalog/projects', {name: 'Synthetic unpublished'})).status, 409, 'rejected old create capability revived');
  assert(!(await f.owner.catalog()).registry.projects.some(project => project.id === pending.id));
  const partialCatalog = await f.owner.catalog();
  await runtime.stop();
  const partial = python('pending', f.state, {action: partialCatalog.registry.actions.create, identity: f.ownerIdentity, partial_quarantine: true});
  await runtime.start(f.state);
  assert.equal((await f.owner.action(partialCatalog, 'create', '/web/catalog/projects', {name: 'Synthetic unpublished'})).status, 409, 'partial rename revived old action');
  await runtime.stop();
  admin(f.state, 'reject-provisioning', ['--project', partial.id, '--reason', 'provisioning-recovery']);
  pass('offline-pending-rejection-private-quarantine-and-partial-rename-no-action-revival');
  await runtime.start(restored.restored);
  const revoked = new Native(f.aOnly, f.a);
  revoked.reject('describe', {role: 'developer', run_id: 'revoked', machine: f.aOnly.machine}, ['UNAUTHORIZED']);
  assert.equal((await new BrowserSession(f.owner.cookie).read('/web/projects')).status, 401);
  await runtime.stop();
  const fresh = catalogToken(restored.restored, 'synthetic-restored-owner');
  for (const project of published) admin(restored.restored, 'recover-owner', ['--project', project, '--identity', fresh.id, '--reason', 'owner-recovery']);
  await runtime.start(restored.restored);
  const web = await new BrowserSession().login(fresh); assert.equal((await web.catalog()).registry.projects.length, published.length);
  const native = new Native(fresh, f.a); native.open('developer'); assert.match(native.call('get', {task_key: f.aNative.task}).title, /Alpha/);
  await web.open(f.a);
  const downloaded = await fetch(h.goOrigin + f.assetA.download_url, {headers: {Cookie: web.cookie}, signal: AbortSignal.timeout(10000)});
  assert.equal(downloaded.status, 200); assert.equal(h.sha(Buffer.from(await downloaded.arrayBuffer())), f.assetA.sha256);
  pass('restored-old-tokens-cookie-rejected-fresh-explicit-owner-history-and-design-bytes');
  await runtime.stop();
}
module.exports = {run};
