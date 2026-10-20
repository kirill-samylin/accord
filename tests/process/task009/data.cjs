'use strict';
const h = require('./harness.cjs');
const {assert, Native, MCP, goOrigin, origin, response, sha} = h;
const {member, changeState} = require('./catalog.cjs');
async function boardData(f, pass) {
  f.aNative = new Native(f.aOnly, f.a); f.aNative.open('coordinator');
  f.bNative = new Native(f.bOnly, f.b); f.bNative.open('coordinator');
  for (const [native, name] of [[f.aNative, 'QA Alpha unique'], [f.bNative, 'QA Beta unique']]) {
    const epic = native.call('create', {type: 'epic', title: name, description: name});
    const task = native.call('create', {type: 'task', epic_key: epic.key, role: 'developer', title: name + ' task', description: name});
    native.call('create', {type: 'task', epic_key: epic.key, role: 'developer', title: name + ' second', description: name});
    native.epic = epic.key; native.task = task.key;
  }
  assert.equal(f.aNative.task, f.bNative.task, 'fixture must exercise colliding project-local task IDs');
  assert.notEqual(f.aNative.call('get', {task_key: f.aNative.task}).title, f.bNative.call('get', {task_key: f.bNative.task}).title);
  const wrong = await f.bNative.http('tasks/' + f.aNative.task, {headers: {'X-Session': f.aNative.session}});
  assert.equal(wrong.status, 401, 'foreign A session lacked explicit B denial');
  const tasks = f.aNative.call('tasks', {limit: 1}); assert(tasks.next_cursor, 'cursor fixture needs multiple tasks');
  f.bNative.reject('tasks', {limit: 1, cursor: tasks.next_cursor}, ['VALIDATION']);
  const pipe = new MCP(f.aOnly, f.a);
  try {
    await pipe.init();
    const opened = await pipe.call('describe', {request_id: 'mcp-open', role: 'developer', run_id: 'mcp-owned', machine: f.aOnly.machine});
    assert.equal(opened.isError, false);
    const task = await pipe.call('get', {task_key: f.aNative.task});
    assert.equal(task.isError, false); assert.match(task.data.title, /Alpha/);
    let denial;
    try {await pipe.rpc('resources/read', {uri: 'accord://designs?session_id=' + f.bNative.session});}
    catch (error) {if (!Number.isInteger(error.protocolCode)) throw error; denial = error;}
    assert(denial && /SESSION_MISMATCH/.test(denial.protocolMessage), 'foreign MCP resource lacked explicit session denial');
  } finally {await pipe.close();}
  pass('same-local-keys-nativeREST-stdioMCP-and-cursor-isolation');
}
async function upload(web, project, title, bytes) {
  if (project) await web.open(project);
  const prefix = project ? '/web/p/' + project : '/web';
  const page = await web.read(prefix + '/designs'); assert.equal(page.status, 200);
  const form = new FormData();
  for (const [key, value] of Object.entries({csrf: page.data.page.csrf, action_id: page.data.page.actions.upload, scope: 'project', scope_key: page.data.page.session.project, title, epic_key: '', design_id: ''})) form.set(key, value);
  form.set('file', new Blob([bytes], {type: 'application/zip'}), title + '.zip');
  const result = await response(goOrigin + prefix + '/designs', {method: 'POST', headers: {Cookie: web.cookie, Accept: 'application/json', Origin: origin}, body: form});
  assert.equal(result.status, 200, 'scoped design upload rejected');
  const list = await web.read(prefix + '/designs');
  const asset = list.data.page.designs.find(value => value.title === title); assert(asset, 'uploaded scoped asset missing');
  return asset;
}
async function designs(f, pass) {
  const bytes = Buffer.from('PK\x05\x06' + '\0'.repeat(18), 'binary');
  f.assetA = await upload(f.owner, f.a, 'QA Alpha design', bytes);
  f.assetB = await upload(f.owner, f.b, 'QA Beta design', bytes);
  assert(f.assetA.download_url.startsWith('/web/p/' + f.a + '/'));
  const metadata = await f.aNative.http('designs/' + f.assetA.asset_id); assert.equal(metadata.status, 200);
  assert(metadata.data.download_url.startsWith('/api/v1/p/' + f.a + '/'), 'native asset item URL lost scope');
  const downloaded = await fetch(goOrigin + f.assetA.download_url, {headers: {Cookie: f.owner.cookie}, signal: AbortSignal.timeout(10000)});
  assert.equal(downloaded.status, 200); assert.equal(sha(Buffer.from(await downloaded.arrayBuffer())), f.assetA.sha256);
  const foreignFile = await f.bWeb.read('/web/p/' + f.b + '/designs/' + f.assetA.asset_id + '/download');
  assert.equal(foreignFile.status, 404, 'foreign project asset accessible');
  assert.equal((await f.aWeb.read('/web/p/' + f.b + '/designs/' + f.assetB.asset_id + '/download')).status, 404);
  pass('scoped-design-upload-metadata-download-bytes-foreign-denied');
}
async function epochs(f, pass) {
  const oldNative = new Native(f.shared, f.a); oldNative.open('developer');
  await f.sharedWeb.open(f.a);
  const oldPage = await f.sharedWeb.read('/web/p/' + f.a + '/?view=tasks'); assert.equal(oldPage.status, 200);
  const unusedPage = await f.sharedWeb.read('/web/p/' + f.a + '/?view=tasks'); assert.equal(unusedPage.status, 200);
  const fields = {csrf: oldPage.data.page.csrf, action_id: oldPage.data.page.actions.create, type: 'epic', title: 'QA epoch original', description: 'epoch'};
  const before = await f.sharedWeb.post('/web/p/' + f.a + '/tasks', fields); assert.equal(before.status, 200);
  assert.equal((await member(f.owner, f.a, f.shared.id, 'viewer', 'set')).status, 200);
  assert.equal((await f.sharedWeb.post('/web/p/' + f.a + '/tasks', fields)).status, 403, 'viewer cached write replay accepted');
  assert.equal((await member(f.owner, f.a, f.shared.id, '', 'revoke')).status, 200);
  oldNative.reject('get', {task_key: f.aNative.task}, ['NOT_FOUND']);
  assert.equal((await f.sharedWeb.read('/web/p/' + f.a + '/')).status, 404);
  assert.equal((await f.sharedWeb.read('/web/catalog/projects/' + f.b)).status, 200, 'A revoke revoked B');
  assert.equal((await member(f.owner, f.a, f.shared.id, 'editor', 'grant')).status, 200);
  oldNative.reject('get', {task_key: f.aNative.task}, ['UNAUTHORIZED']);
  await f.sharedWeb.open(f.a);
  const reopened = await f.sharedWeb.read('/web/p/' + f.a + '/?view=tasks'); assert.equal(reopened.status, 200);
  for (const action_id of [fields.action_id, unusedPage.data.page.actions.create]) {
    const denied = await f.sharedWeb.post('/web/p/' + f.a + '/tasks', {...fields, csrf: reopened.data.page.csrf, action_id});
    assert([401, 409].includes(denied.status) && ['UNAUTHORIZED', 'IDEMPOTENCY_CONFLICT'].includes(denied.data.error?.code), 'regrant revived old browser action');
  }
  const fresh = new Native(f.shared, f.a); fresh.open('developer'); assert.match(fresh.call('get', {task_key: f.aNative.task}).title, /Alpha/);
  assert.equal((await member(f.owner, f.a, f.shared.id, 'owner', 'set')).status, 200);
  const managed = await f.sharedWeb.detail(f.a);
  const endpoint = '/web/catalog/projects/' + f.a + '/update';
  const update = {name: managed.project.name, state: 'active', version: managed.project.version};
  assert.equal((await f.sharedWeb.action(managed, 'update', endpoint, update)).status, 200);
  assert.equal((await member(f.owner, f.a, f.shared.id, '', 'revoke')).status, 200);
  assert.equal((await member(f.owner, f.a, f.shared.id, 'owner', 'grant')).status, 200);
  assert.equal((await f.sharedWeb.action(managed, 'update', endpoint, update)).status, 409, 'regrant revived cached catalog action');
  assert.equal((await member(f.owner, f.a, f.shared.id, 'editor', 'set')).status, 200);
  pass('downgrade-revoke-regrant-old-native-unused-cached-browser-and-catalog-actions');
}
async function archive(f, pass) {
  assert.equal((await changeState(f.owner, f.b, 'archived')).status, 200);
  assert.equal(f.bNative.call('get', {task_key: f.bNative.task}).key, f.bNative.task);
  f.bNative.reject('create', {type: 'epic', title: 'forbidden archived', description: 'forbidden'}, ['FORBIDDEN_ROLE']);
  assert.equal((await changeState(f.owner, f.b, 'active')).status, 200);
  const developer = new Native(f.aOnly, f.a); developer.open('developer');
  const claim = developer.call('next', {task_key: f.aNative.task}); assert(claim.lease);
  assert.equal((await changeState(f.owner, f.a, 'archived')).status, 409, 'live lease archived');
  developer.call('release', {task_key: f.aNative.task, generation: claim.lease.generation, reason: 'Synthetic QA cleanup'});
  pass('archive-read-write-isolation-and-live-lease-protection');
}
module.exports = {boardData, upload, designs, epochs, archive};
