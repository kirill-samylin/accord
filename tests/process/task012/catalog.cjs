'use strict';
const h = require('./harness.cjs');
const {assert} = h;
async function create(owner, name) {
  const page = await owner.catalog();
  const result = await owner.action(page, 'create', '/web/catalog/projects', {name, repository_url: 'https://example.com/qa.git', branch: 'main'});
  assert.equal(result.status, 200, 'project create rejected');
  return result.data.redirect.split('/').at(-1);
}
async function member(owner, project, identity, role, operation = 'grant') {
  const page = await owner.detail(project);
  const existing = page.members.find(value => value.identity_id === identity);
  const fields = {action: operation, identity_id: identity, ...(role ? {role} : {}), ...(existing ? {version: existing.version} : {})};
  const key = operation === 'grant' ? 'member:grant' : 'member:' + identity + ':' + operation;
  const result = await owner.action(page, key, '/web/catalog/projects/' + project + '/members', fields);
  assert.equal(result.status, 200, 'membership operation rejected');
  return result;
}
async function state(owner, project, state) {
  const page = await owner.detail(project);
  const result = await owner.action(page, 'update', '/web/catalog/projects/' + project + '/update', {name: page.project.name, version: page.project.version, state});
  assert.equal(result.status, 200, 'project lifecycle rejected');
}
async function upload(web, project, title) {
  await web.open(project);
  const prefix = '/web/p/' + project;
  const page = await web.read(prefix + '/designs'); assert.equal(page.status, 200);
  const form = new FormData(), bytes = Buffer.from('PK\x05\x06' + '\0'.repeat(18), 'binary');
  for (const [key, value] of Object.entries({csrf: page.data.page.csrf, action_id: page.data.page.actions.upload, scope: 'project', scope_key: page.data.page.session.project, title, epic_key: '', design_id: ''})) form.set(key, value);
  form.set('file', new Blob([bytes], {type: 'application/zip'}), title + '.zip');
  const result = await h.response(h.goOrigin + prefix + '/designs', {method: 'POST', headers: {Cookie: web.cookie, Accept: 'application/json', Origin: h.origin}, body: form});
  assert.equal(result.status, 200);
  const list = await web.read(prefix + '/designs');
  return list.data.page.designs.find(value => value.title === title);
}
module.exports = {create, member, state, upload};
