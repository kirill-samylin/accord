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
  return {prefix: Buffer.from(prefix), tail, headers: {'Content-Type': 'multipart/form-data; boundary=' + boundary, 'Content-Length': Buffer.byteLength(prefix) + size + tail.length}};
}
async function main() {
  fs.mkdirSync(root, {recursive: true, mode: 0o700}); fs.chmodSync(root, 0o700);
  const state = path.join(root, 'upload-' + crypto.randomUUID()); fs.mkdirSync(state, {mode: 0o700});
  const file = token(state, 'upload-' + crypto.randomUUID());
  srv = await server(binary, state, await port()); const {origin} = srv; report.process_roles = {server_pid: srv.child.pid, client: 'real net/http multipart'};
  const page = await fetch(origin + '/web/login');
  const html = await page.text(); const csrf = html.match(/name="csrf" value="([^"]+)"/)[1];
  const preauth = page.headers.getSetCookie().map(s => s.split(';')[0]).join('; ');
  const form = new FormData(); form.append('csrf', csrf); form.append('token', fs.readFileSync(file, 'utf8').trim());
  const login = await fetch(origin + '/web/login', {method: 'POST', headers: {Origin: origin, Cookie: preauth}, body: form, redirect: 'manual'});
  assert.equal(login.status, 303, 'synthetic process login failed');
  const cookie = login.headers.getSetCookie().map(s => s.split(';')[0]).join('; ');
  async function fields(title) {
    const p = await fetch(origin + '/web/designs', {headers: {Cookie: cookie}}); assert.equal(p.status, 200);
    const body = await p.text(); const upload = body.split('data-upload>')[1];
    return {csrf: upload.match(/name="csrf" value="([^"]+)"/)[1], action_id: upload.match(/name="action_id" value="([^"]+)"/)[1], title, scope: 'project', scope_key: '', epic_key: '', design_id: ''};
  }
  async function sized(size) {
    const body = multipart(await fields('Synthetic boundary ' + size), 'boundary.zip', size);
    const bytes = Buffer.alloc(size); bytes.write('PK\x05\x06', 0, 'binary');
    return new Promise((resolve, reject) => {
      const req = http.request(origin + '/web/designs', {method: 'POST', headers: {...body.headers, Origin: origin, Cookie: cookie}}, res => {res.resume(); res.on('end', () => resolve(res.statusCode));});
      req.on('error', reject); req.setTimeout(15000, () => req.destroy(new Error('byte boundary timeout')));
      req.write(body.prefix); req.write(bytes); req.end(body.tail);
    });
  }
  assert.equal(await sized(50 * 1024 * 1024), 303, 'inclusive50MiB boundary failed');
  assert.equal(await sized(50 * 1024 * 1024 + 1), 413, 'server accepted over50MiB');
  report.checks.push({id: 'server-inclusive50MiB-and-over-limit', result: 'PASS', accepted_bytes: 50 * 1024 * 1024, rejected_bytes: 50 * 1024 * 1024 + 1}); console.log('server-inclusive50MiB-and-over-limit PASS');
  if (process.env.ACCORD_QA_UPLOAD_BOUNDARY_ONLY) {report.result = 'PASS'; return;}
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
