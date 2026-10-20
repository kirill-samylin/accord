'use strict';
// Keep a real multipart body incomplete while the owner changes current access.
const http = require('node:http');
const {assert, origin, goOrigin, crypto, sleep, python} = require('./harness.cjs');
const {member} = require('./catalog.cjs');
async function run(f, pass) {
  await f.sharedWeb.open(f.a);
  const page = await f.sharedWeb.read('/web/p/' + f.a + '/designs'); assert.equal(page.status, 200);
  const title = 'Synthetic revoked stream';
  const fields = {csrf: page.data.page.csrf, action_id: page.data.page.actions.upload,
    title, scope: 'project', scope_key: page.data.page.session.project, epic_key: '', design_id: ''};
  const boundary = 'Task009' + crypto.randomUUID();
  let prefix = '';
  for (const [name, value] of Object.entries(fields)) prefix += `--${boundary}\r\nContent-Disposition: form-data; name="${name}"\r\n\r\n${value}\r\n`;
  prefix += `--${boundary}\r\nContent-Disposition: form-data; name="file"; filename="revoked.zip"\r\nContent-Type: application/zip\r\n\r\n`;
  const zip = Buffer.from('PK\x05\x06' + '\0'.repeat(18), 'binary');
  const tail = Buffer.from(`\r\n--${boundary}--\r\n`);
  let request;
  const result = new Promise((resolve, reject) => {
    request = http.request(goOrigin + '/web/p/' + f.a + '/designs', {method: 'POST',
      headers: {Origin: origin, Cookie: f.sharedWeb.cookie, Accept: 'application/json',
        'Content-Type': 'multipart/form-data; boundary=' + boundary,
        'Content-Length': Buffer.byteLength(prefix) + zip.length + tail.length}}, response => {
      const chunks = []; response.on('data', chunk => chunks.push(chunk));
      response.on('end', () => {try {resolve({status: response.statusCode, data: JSON.parse(Buffer.concat(chunks).toString())});} catch {reject(new Error('upload response was not JSON'));}});
    });
    request.once('error', reject);
    request.setTimeout(15000, () => request.destroy(new Error('synthetic upload deadline')));
    request.write(prefix); request.write(zip.subarray(0, 4));
  });
  // Install a rejection observer immediately; the final await below still
  // distinguishes an explicit server permission denial from transport failure.
  result.catch(() => {});
  try {
    const until = Date.now() + 5000;
    let pending;
    while (Date.now() < until) {
      pending = python('upload_inspect', f.state, {project: f.a, title});
      if (pending.pending === 1) break;
      await sleep(30);
    }
    assert.equal(pending.pending, 1, 'real upload reservation was not observed');
    const started = Date.now();
    assert.equal((await member(f.owner, f.a, f.shared.id, 'viewer', 'set')).status, 200);
    assert(Date.now() - started < 3000, 'incomplete upload held the global membership guard');
    request.end(Buffer.concat([zip.subarray(4), tail]));
    const response = await result;
    assert.equal(response.status, 403, 'upload final publication ignored current viewer permission');
    const cleared = python('upload_inspect', f.state, {project: f.a, title});
    assert.deepEqual(cleared, {pending: 0, ready: 0, reserved: 0, staging: 0});
    assert.equal((await member(f.owner, f.a, f.shared.id, 'editor', 'set')).status, 200);
    pass('real-incomplete-upload-short-guard-downgrade-final-publication-denied-cleaned');
  } finally {request.destroy();}
}
module.exports = {run};
