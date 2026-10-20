'use strict';
const h = require('./harness.cjs');
const {fs, path, assert, root, binary, oldBinary, sha, save, cli, admin, legacyToken, catalogToken, freePorts, startServer, stop, BrowserSession, Native, python} = h;
const catalog = require('./catalog.cjs');
const data = require('./data.cjs');
const testManifest = () => Object.fromEntries([__dirname, path.join(__dirname, '../../browser/task009')].flatMap(directory => fs.readdirSync(directory).filter(name => /\.(cjs|py)$/.test(name)).map(name => [path.relative(h.repo, path.join(directory, name)), sha(fs.readFileSync(path.join(directory, name)))])));
const sourceTests = testManifest();
const goTool = process.env.ACCORD_TASK009_GO || path.join(h.repo, '.local/toolchains/go/bin/go');
const goVersion = fs.existsSync(goTool) ? h.cp.spawnSync(goTool, ['version', '-m', binary], {env: h.env, encoding: 'utf8', timeout: 10000}) : null;
if (goVersion) assert.equal(goVersion.status, 0, 'Go provenance inspection failed');
if (goVersion) fs.writeFileSync(path.join(root, 'go-buildinfo.txt'), goVersion.stdout, {mode: 0o600});
const report = {schema: 1, started_at: new Date().toISOString(), node: process.version,
  binary_sha256: sha(fs.readFileSync(binary)), old_binary_sha256: sha(fs.readFileSync(oldBinary)), checks: [],
  platform: process.platform, architecture: process.arch,
  go_version: goVersion ? goVersion.stdout.split('\n')[0].split(': ').at(-1) : process.env.ACCORD_TASK009_GO_VERSION || null,
  go_version_evidence: goVersion ? 'actual-go-version-m' : process.env.ACCORD_TASK009_GO_VERSION ? 'operator-build-provenance' : 'NOT-AVAILABLE',
  go_buildinfo_sha256: goVersion ? sha(Buffer.from(goVersion.stdout)) : null,
  next_build_id: fs.existsSync(path.join(h.repo, 'web/.local/next/BUILD_ID')) ? fs.readFileSync(path.join(h.repo, 'web/.local/next/BUILD_ID'), 'utf8').trim() : null,
  scope: process.env.ACCORD_TASK009_BROWSER === '0' ? 'process-only' : 'process-and-Chrome',
  source_revision: process.env.ACCORD_TASK009_REVISION || null, source_dirty: process.env.ACCORD_TASK009_SOURCE_DIRTY || 'not-inspected-by-tester',
  source_aggregate: process.env.ACCORD_TASK009_SOURCE_AGGREGATE || null,
  limitations: ['one ' + process.platform + ' host; no physical second PC', 'HTTP loopback; production HTTPS separate', 'repository references; no remote Git clone']};
const pass = (id, measurements = {}) => {report.checks.push({id, result: 'PASS', ...measurements}); console.log(id + ' PASS');};
let srv;
async function main() {
  await freePorts();
  const state = path.join(root, 'state'); fs.mkdirSync(state, {mode: 0o700});
  const ownerToken = legacyToken(state, 'synthetic-owner', oldBinary);
  const workerToken = legacyToken(state, 'synthetic-legacy-worker', oldBinary);
  srv = await startServer(state, {legacy: true, bin: oldBinary});
  const oldNative = new Native(ownerToken, '', oldBinary); oldNative.open('coordinator');
  const epic = oldNative.call('create', {type: 'epic', title: 'QA legacy preserved epic', description: 'Synthetic legacy history'});
  const task = oldNative.call('create', {type: 'task', epic_key: epic.key, role: 'developer', title: 'QA legacy preserved lease', description: 'Synthetic lease preservation'});
  const worker = new Native(workerToken, '', oldBinary); worker.open('developer');
  const claim = worker.call('next', {task_key: task.key}); assert(claim.lease, 'legacy live lease fixture missing');
  const owner = await new BrowserSession().login(ownerToken);
  const oldCookie = owner.cookie;
  const oldPage = await owner.read('/web/'); assert.equal(oldPage.status, 200);
  const oldHuman = oldPage.data.page.session.session_id;
  const oldBytes = Buffer.from('PK\x05\x06' + '\0'.repeat(18), 'binary');
  const oldAsset = await data.upload(owner, '', 'QA legacy preserved design', oldBytes);
  await stop(srv); srv = null;
  const before = python('capture', state); assert.equal(before.web_schema, 1);
  admin(state, 'bootstrap-owner', ['--identity', ownerToken.id]);
  const identities = {};
  for (const name of ['aOnly', 'bOnly', 'shared', 'viewer']) identities[name] = catalogToken(state, 'synthetic-' + name);
  const ownerIdentity = python('catalog_owner', state, {token_id: ownerToken.id}).identity;
  srv = await startServer(state);
  const first = await owner.catalog(); const defaultId = first.registry.projects[0].id;
  assert.equal(first.registry.projects[0].role, 'owner');
  const preservedPage = await owner.read('/web/'); assert.equal(preservedPage.status, 200);
  assert.equal(owner.cookie, oldCookie, 'migration rotated legacy browser cookie');
  assert.equal(preservedPage.data.page.session.session_id, oldHuman, 'migration replaced legacy native human session');
  const after = python('capture', state);
  assert.equal(after.board_schema, 2); assert.equal(after.web_schema, 2);
  for (const kind of ['board', 'web']) for (const [table, rows] of Object.entries(before[kind])) {
    for (const row of rows) assert(after[kind][table].includes(row), 'migration rewrote legacy ' + table);
  }
  assert(Object.keys(before.blobs).length, 'legacy ready blob fixture missing');
  assert.deepEqual(after.blobs, before.blobs, 'migration changed existing design blob bytes');
  const oldDownload = await fetch(h.goOrigin + oldAsset.download_url, {headers: {Cookie: owner.cookie}, signal: AbortSignal.timeout(10000)});
  assert.equal(oldDownload.status, 200); assert.equal(sha(Buffer.from(await oldDownload.arrayBuffer())), sha(oldBytes));
  assert.equal(new Native(workerToken).call('get', {task_key: task.key, session_id: worker.session}).lease.generation, claim.lease.generation);
  assert(python('inspect', state).binding_matches);
  pass('actual-legacy-binary-web1-to2-default-token-cookie-session-lease-row-preservation');
  const f = {state, ownerToken, owner, ownerIdentity, defaultId, ...identities};
  f.a = await catalog.create(owner, 'QA Alpha', 'https://github.com/example/alpha.git');
  f.b = await catalog.create(owner, 'QA Beta', 'ssh://git@github.com/example/beta.git');
  assert.notEqual(f.a, f.b);
  pass('isolated-project-create-and-repository-association');
  await catalog.repositories(f, pass);
  await catalog.acl(f, pass);
  await catalog.authBoundary(f, pass);
  await data.boardData(f, pass);
  await data.designs(f, pass);
  await data.epochs(f, pass);
  await require('./stream.cjs').run(f, pass);
  await data.archive(f, pass);
  await stop(srv); srv = null;
  srv = await startServer(state);
  assert.equal((await owner.read('/web/p/' + f.a + '/designs')).status, 200);
  assert.match(f.bNative.call('get', {task_key: f.bNative.task}).title, /Beta/);
  pass('registry-boards-browser-cookie-restart-preservation');
  save(path.join(root, 'fixture.json'), {state, a: f.a, b: f.b, defaultId, ownerCookie: owner.cookie,
    ownerToken, ownerIdentity, aOnly: f.aOnly, bOnly: f.bOnly, shared: f.shared, viewer: f.viewer,
    aTask: f.aNative.task, bTask: f.bNative.task, assetA: f.assetA, assetB: f.assetB});
  if (process.env.ACCORD_TASK009_BROWSER !== '0') {await require('../../browser/task009/browser.cjs').run(f, pass); report.chrome = f.browserVersion;}
  await stop(srv); srv = null;
  require('./startup.cjs').run(f, pass);
  const lifecycle = require('./lifecycle.cjs');
  await lifecycle.run(f, pass, {start: async stage => {srv = await startServer(stage);}, stop: async () => {await stop(srv); srv = null;}});
  report.result = 'PASS';
}
main().catch(error => {report.result = 'FAIL'; report.failure = String(error.message).slice(0, 220); console.error(report.failure); process.exitCode = 1;}).finally(async () => {
  try {await stop(srv);} catch {report.result = 'FAIL'; report.cleanup_failure = true; process.exitCode = 1;}
  report.finished_at = new Date().toISOString();
  report.tests = sourceTests;
  if (JSON.stringify(testManifest()) !== JSON.stringify(sourceTests)) {report.result = 'FAIL'; report.test_source_changed_during_run = true; process.exitCode = 1;}
  save(path.join(root, 'results.json'), report); console.log('private evidence: ' + path.relative(h.repo, root));
});
