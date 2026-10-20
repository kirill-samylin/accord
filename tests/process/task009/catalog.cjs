'use strict';
const h = require('./harness.cjs');
const {assert, BrowserSession, Native, response, goOrigin, fs, origin} = h;
async function create(owner, name, url) {
  const page = await owner.catalog();
  const result = await owner.action(page, 'create', '/web/catalog/projects', {name, repository_url: url, branch: 'main'});
  assert.equal(result.status, 200, 'project create rejected');
  return result.data.redirect.split('/').at(-1);
}
async function member(owner, project, identity, role, operation = 'grant') {
  const page = await owner.detail(project);
  const existing = page.members.find(value => value.identity_id === identity);
  const fields = {action: operation, identity_id: identity, ...(role ? {role} : {}), ...(existing ? {version: existing.version} : {})};
  const key = operation === 'grant' ? 'member:grant' : 'member:' + identity + ':' + operation;
  return owner.action(page, key, '/web/catalog/projects/' + project + '/members', fields);
}
async function changeState(owner, project, state) {
  const page = await owner.detail(project);
  return owner.action(page, 'update', '/web/catalog/projects/' + project + '/update', {name: page.project.name, version: page.project.version, state});
}
async function repositories(f, pass) {
  const page = await f.owner.detail(f.a);
  const endpoint = '/web/catalog/projects/' + f.a + '/repositories';
  const fields = {url: 'https://github.com/example/second.git', branch: 'develop'};
  const same = await Promise.all([f.owner.action(page, 'repository:add', endpoint, fields), f.owner.action(page, 'repository:add', endpoint, fields)]);
  assert.deepEqual(same.map(result => result.status), [200, 200], 'concurrent catalog replay rejected');
  const after = await f.owner.detail(f.a);
  assert.equal(after.project.repositories.filter(repo => repo.url === fields.url).length, 1, 'durable add duplicated repository');
  assert.equal((await f.owner.action(page, 'repository:add', endpoint, {...fields, branch: 'different'})).status, 409);
  for (const url of ['file:///tmp/repo', 'https://user:password@example.com/repo', 'https://example.com/repo?token=synthetic', 'https://example.com/repo#secret', '/tmp/repo', 'ext::run']) {
    const fresh = await f.owner.detail(f.a);
    assert.equal((await f.owner.action(fresh, 'repository:add', endpoint, {url})).status, 400, 'unsafe repository accepted');
  }
  const repo = after.project.repositories.find(repo => repo.url === fields.url);
  let current = await f.owner.detail(f.a);
  assert.equal((await f.owner.action(current, 'repository:' + repo.id + ':update', endpoint + '/' + repo.id,
    {action: 'update', version: repo.version, url: 'ssh://git@github.com/example/second.git', branch: 'main'})).status, 200);
  current = await f.owner.detail(f.a);
  const updated = current.project.repositories.find(value => value.id === repo.id);
  assert.equal((await f.owner.action(current, 'repository:' + repo.id + ':remove', endpoint + '/' + repo.id, {action: 'remove', version: updated.version})).status, 200);
  pass('repository-crud-validation-concurrent-durable-replay');
}
async function acl(f, pass) {
  for (const [project, token, role] of [[f.a, f.aOnly, 'editor'], [f.b, f.bOnly, 'editor'], [f.a, f.shared, 'editor'], [f.b, f.shared, 'viewer'], [f.b, f.viewer, 'viewer']]) {
    assert.equal((await member(f.owner, project, token.id, role)).status, 200);
  }
  f.aWeb = await new BrowserSession().login(f.aOnly);
  f.bWeb = await new BrowserSession().login(f.bOnly);
  f.sharedWeb = await new BrowserSession().login(f.shared);
  f.viewerWeb = await new BrowserSession().login(f.viewer);
  assert.equal((await f.bWeb.catalog()).registry.projects.length, 1);
  assert.equal((await f.aWeb.catalog()).registry.projects.length, 1);
  assert.equal((await f.sharedWeb.catalog()).registry.projects.length, 2);
  const unknown = 'p_' + '0'.repeat(32);
  for (const web of [f.aWeb, f.bWeb]) {
    const forbidden = web === f.aWeb ? f.b : f.a;
    const hidden = await web.read('/web/catalog/projects/' + forbidden);
    const missing = await web.read('/web/catalog/projects/' + unknown);
    assert.equal(hidden.status, 404); assert.deepEqual(hidden.data, missing.data, 'hidden project existence disclosed');
    assert.equal((await web.read('/web/')).status, 404, 'legacy default alias bypassed ACL');
  }
  const bCatalog = await f.bWeb.catalog(); const oldCookie = f.bWeb.cookie;
  const logout = await f.bWeb.action(bCatalog, 'logout', '/web/logout', {});
  assert.equal(logout.status, 200, 'B-only logout failed');
  assert.equal((await new BrowserSession(oldCookie).read('/web/projects')).status, 401, 'logout cookie resurrected');
  f.bWeb = await new BrowserSession().login(f.bOnly);
  assert.equal((await f.viewerWeb.detail(f.b)).members, undefined, 'viewer received member metadata');
  const nativeViewer = new Native(f.viewer, f.b); nativeViewer.open('human');
  nativeViewer.reject('describe', {role: 'architect', run_id: 'forbidden', machine: f.viewer.machine}, ['FORBIDDEN_ROLE']);
  nativeViewer.reject('create', {type: 'epic', title: 'forbidden', description: 'forbidden'}, ['FORBIDDEN_ROLE']);
  nativeViewer.reject('heartbeat', {}, ['FORBIDDEN_ROLE']);
  const defaultPage = await f.owner.detail(f.defaultId);
  assert.equal(defaultPage.project.can_archive, false);
  assert.equal((await member(f.owner, f.a, f.ownerIdentity, '', 'revoke')).status, 409, 'last owner removed');
  pass('A-only-B-only-shared-viewer-lastowner-global-logout-ACL');
}
async function authBoundary(f, pass) {
  const n = new Native(f.aOnly, f.a); n.open('developer');
  for (const originValue of ['', 'null', origin]) {
    const result = await response(goOrigin + '/api/v1/p/' + f.a + '/tasks', {headers: {Authorization: 'Bearer ' + fs.readFileSync(f.aOnly.file, 'utf8').trim(), 'X-Session': n.session, Origin: originValue}});
    assert.equal(result.status, 403, 'native Origin boundary bypassed');
  }
  const page = await f.owner.detail(f.a);
  const route = '/web/catalog/projects/' + f.a + '/repositories';
  assert.equal((await f.owner.post(route, {csrf: 'invalid', action_id: page.registry.actions['repository:add'], url: 'https://example.com/repo.git'})).status, 403);
  const cross = await response(goOrigin + route, {method: 'POST', headers: {Cookie: f.owner.cookie, Accept: 'application/json', Origin: 'http://example.com'}, body: new URLSearchParams({csrf: page.registry.csrf})});
  assert.equal(cross.status, 403);
  pass('native-Origin-and-browser-CSRF-boundaries');
}
module.exports = {create, member, changeState, repositories, acl, authBoundary};
