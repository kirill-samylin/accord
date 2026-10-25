'use strict';
// Real net/http listener: server byte boundary and both deadline directions.
const {fs, path, cp, assert, required, binary, root, token, port, server, stop, sha, manifest} = require('./harness.cjs');
const http = require('node:http');
const crypto = require('node:crypto');
let srv;
const report = {schema: 1, binary_sha256: sha(fs.readFileSync(binary)), test_sha256: manifest(), node: process.version, scope: process.env.ACCORD_QA_UPLOAD_BOUNDARY_ONLY ? 'byte-boundary-only' : 'full', checks: [], limitations: ['HTTP loopback synthetic process fixture']};
function multipart(fields, filename, size) {
  const boundary = 'AccordDeadline' + crypto.randomUUID();
  let prefix = '';
  for (const [key, value] of Object.entries(fields)) prefix += `--${boundary}\r\nContent-Disposition: form-data; name="${key}"\r\n\r\n${value}\r\n`;
  prefix += `--${boundary}\r\nContent-Disposition: form-data; name="file"; filename="${filename}"\r\nContent-Type: application/zip\r\n\r\n`;
  const tail = Buffer.from(`\r\n--${boundary}--\r\n`);
  return {prefix: Buffer.from(prefix), tail, headers: {Accept: 'application/json', 'Content-Type': 'multipart/form-data; boundary=' + boundary, 'Content-Length': Buffer.byteLength(prefix) + size + tail.length}};
}
async function main() {
  fs.mkdirSync(root, {recursive: true, mode: 0o700}); fs.chmodSync(root, 0o700);
  const state = path.join(root, 'upload-' + crypto.randomUUID()); fs.mkdirSync(state, {mode: 0o700});
  const file = token(state, 'upload-' + crypto.randomUUID());
  srv = await server(binary, state, await port()); const {origin} = srv; report.process_roles = {server_pid: srv.child.pid, client: 'real net/http multipart'};
  const page = await fetch(origin + '/web/login', {headers: {Accept: 'application/json'}, redirect: 'manual'});
  assert.equal(page.status, 200); const initial = await page.json(); assert(initial.ok && initial.page.csrf);
  const csrf = initial.page.csrf;
  const preauth = page.headers.getSetCookie().map(s => s.split(';')[0]).join('; ');
  const form = new FormData(); form.append('csrf', csrf); form.append('token', fs.readFileSync(file, 'utf8').trim());
  const login = await fetch(origin + '/web/login', {method: 'POST', headers: {Accept: 'application/json', Origin: origin, Cookie: preauth}, body: form, redirect: 'manual'});
  assert.equal(login.status, 200, 'synthetic process JSON login failed'); assert((await login.json()).ok);
  const cookie = login.headers.getSetCookie().map(s => s.split(';')[0]).join('; ');
  async function designs() {
    const response = await fetch(origin + '/web/designs', {headers: {Accept: 'application/json', Cookie: cookie}, redirect: 'manual'});
    assert.equal(response.status, 200); const result = await response.json(); assert(result.ok && result.page); return result.page;
  }
  async function fields(title) {
    const page = await designs(); assert(page.csrf && page.actions.upload);
    return {csrf: page.csrf, action_id: page.actions.upload, title, scope: 'project', scope_key: '', epic_key: '', design_id: ''};
  }
  const beforeHTML = (await designs()).designs ?? [];
  const rejectedHTML = await fetch(origin + '/web/designs', {method: 'POST', headers: {Accept: 'text/html', Origin: origin, Cookie: cookie}, body: new URLSearchParams(await fields('Synthetic rejected HTML')), redirect: 'manual'});
  assert.equal(rejectedHTML.status, 406, 'non-JSON upload reached form processing'); assert.equal(rejectedHTML.headers.get('location'), null);
  assert.equal((await rejectedHTML.json()).error.code, 'JSON_REQUIRED'); assert.deepEqual((await designs()).designs ?? [], beforeHTML);
  report.checks.push({id: 'non-json-post406-no-upload-mutation', result: 'PASS'}); console.log('non-json-post406-no-upload-mutation PASS');
  async function sized(size) {
    const title = 'Synthetic boundary ' + size; const body = multipart(await fields(title), 'boundary.zip', size);
    const bytes = Buffer.alloc(size); bytes.write('PK\x05\x06', 0, 'binary');
    const status = await new Promise((resolve, reject) => {
      const req = http.request(origin + '/web/designs', {method: 'POST', headers: {...body.headers, Origin: origin, Cookie: cookie}}, res => {res.resume(); res.on('end', () => resolve(res.statusCode));});
      req.on('error', reject); req.setTimeout(15000, () => req.destroy(new Error('byte boundary timeout')));
      req.write(body.prefix); req.write(bytes); req.end(body.tail);
    });
    if (status === 200) {const asset = ((await designs()).designs ?? []).find(row => row.title === title); assert(asset); assert.equal(asset.size, size); assert.equal(asset.sha256, sha(bytes));}
    return status;
  }
  assert.equal(await sized(50 * 1024 * 1024), 200, 'inclusive50MiB JSON boundary failed');
  assert.equal(await sized(50 * 1024 * 1024 + 1), 413, 'server accepted over50MiB');
  report.checks.push({id: 'server-inclusive50MiB-and-over-limit', result: 'PASS', accepted_bytes: 50 * 1024 * 1024, rejected_bytes: 50 * 1024 * 1024 + 1}); console.log('server-inclusive50MiB-and-over-limit PASS');
  if (process.env.ACCORD_QA_UPLOAD_BOUNDARY_ONLY) {report.result = 'PASS'; return;}
  const slow = multipart(await fields('Synthetic31s streaming'), 'slow.zip', 22); const slowStart = Date.now();
  const slowStatus = await new Promise((resolve, reject) => {
    let timer;
    const req = http.request(origin + '/web/designs', {method: 'POST', headers: {...slow.headers, Origin: origin, Cookie: cookie}}, res => {res.resume(); res.on('end', () => {clearTimeout(timer); resolve(res.statusCode);});});
    req.on('error', error => {clearTimeout(timer); reject(error);}); req.setTimeout(45000, () => req.destroy(new Error('slow31s request timeout')));
    req.write(slow.prefix); req.write(Buffer.from('PK\x05\x06', 'binary'));
    timer = setTimeout(() => req.end(Buffer.concat([Buffer.alloc(18), slow.tail])), 31000);
  });
  const slowElapsed = Date.now() - slowStart; assert.equal(slowStatus, 200); assert(slowElapsed >= 31000 && slowElapsed < 45000, 'slow31s stream tolerance');
  const slowAsset = ((await designs()).designs ?? []).find(row => row.title === 'Synthetic31s streaming'); assert(slowAsset); assert.equal(slowAsset.size, 22); assert.equal(slowAsset.sha256, sha(Buffer.concat([Buffer.from('PK\x05\x06', 'binary'), Buffer.alloc(18)])));
  report.checks.push({id: 'server-slow-body-over-global30s', result: 'PASS', elapsed_ms: slowElapsed}); console.log('server-slow-body-over-global30s PASS');
  const body = multipart(await fields('Synthetic120s deadline'), 'deadline.zip', 22);
  const start = Date.now();
  const outcome = await new Promise((resolve, reject) => {
    let settled = false;
    const timer = setTimeout(() => {if (!settled) {settled = true; req.destroy(); reject(new Error('upload not terminated within130s'));}}, 130000);
    const finish = (data) => {if (settled) return; settled = true; clearTimeout(timer); req.destroy(); resolve(data);};
    const req = http.request(origin + '/web/designs', {method: 'POST', headers: {...body.headers, Origin: origin, Cookie: cookie}}, res => {res.resume(); res.on('end', () => finish({status: res.statusCode}));});
    req.on('error', err => finish({network_code: err.code || 'error'}));
    // Send valid metadata and only the ZIP signature; never complete the body.
    // The server, rather than a client timeout, must terminate this request.
    req.write(body.prefix); req.write(Buffer.from('PK\x05\x06', 'binary'));
  });
  const elapsed = Date.now() - start;
  assert(elapsed >= 118000 && elapsed <= 128000, 'upload deadline outside120s tolerance');
  assert(!outcome.status || outcome.status >= 400, 'incomplete upload unexpectedly succeeded');
  assert.equal((await fetch(origin + '/readyz')).status, 200, 'server unhealthy after deadline');
  const counts = JSON.parse(cp.execFileSync(required('ACCORD_QA_PYTHON'), ['-c', "import sqlite3,json,sys; db=sqlite3.connect('file:'+sys.argv[1]+'?mode=ro',uri=True); print(json.dumps(db.execute(\"SELECT count(*),coalesce(sum(reserved),0) FROM design_assets WHERE status='pending' OR filename='deadline.zip'\").fetchone()))", path.join(state, 'web/web.db')], {encoding:'utf8'}));
  assert.deepEqual(counts, [0, 0], 'timed-out upload leaked row/reservation');
  assert.equal(fs.readdirSync(path.join(state, 'web/staging')).length, 0, 'timed-out upload left staging bytes');
  report.checks.push({id: 'actual120s-incomplete-upload-deadline', result: 'PASS', elapsed_ms: elapsed, ...outcome}); console.log('actual120s-incomplete-upload-deadline PASS');
  report.binary_sha256 = sha(fs.readFileSync(binary)); report.result = 'PASS';
}
main().catch(err => {report.result = 'FAIL'; report.failure = String(err.message).slice(0, 180); console.error(report.failure); process.exitCode = 1;}).finally(async () => {
  if (srv) try {await stop(srv);} catch {report.result = 'FAIL'; report.cleanup_failure = true; process.exitCode = 1;}
  report.finished_at = new Date().toISOString(); fs.writeFileSync(path.join(root, 'upload-results.json'), JSON.stringify(report, null, 2), {mode: 0o600});
});
