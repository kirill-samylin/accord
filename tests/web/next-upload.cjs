'use strict';
// Actual multipart bytes travel through production Next's rewrite and Go deadlines.
const h = require('./next-fixture.cjs');
const http = require('node:http');
const crypto = require('node:crypto');
module.exports = async (f, pass) => {
  const cookies = await f.ctx.cookies(); const cookie = cookies.map(c => c.name + '=' + c.value).join('; ');
  async function body(title, size) {
    const page = await h.json(f, '/web/designs'); const boundary = 'AccordNext' + crypto.randomUUID();
    const fields = {csrf: page.csrf, action_id: page.actions.upload, title, scope: 'project', scope_key: '', epic_key: '', design_id: ''};
    let prefix = ''; for (const [key, value] of Object.entries(fields)) prefix += `--${boundary}\r\nContent-Disposition: form-data; name="${key}"\r\n\r\n${value}\r\n`;
    prefix += `--${boundary}\r\nContent-Disposition: form-data; name="file"; filename="boundary.zip"\r\nContent-Type: application/zip\r\n\r\n`;
    const tail = Buffer.from(`\r\n--${boundary}--\r\n`);
    return {prefix: Buffer.from(prefix), tail, headers: {Accept: 'application/json', Origin: f.origin, Cookie: cookie, 'Content-Type': 'multipart/form-data; boundary=' + boundary, 'Content-Length': Buffer.byteLength(prefix) + size + tail.length}};
  }
  async function sized(size) {
    const title='Synthetic proxy bytes '+size; const request = await body(title, size); const bytes = Buffer.alloc(size); bytes.write('PK\x05\x06', 0, 'binary');
    const status=await new Promise((resolve, reject) => {
      const req = http.request(f.origin + '/web/designs', {method: 'POST', headers: request.headers}, res => {res.resume(); res.on('end', () => resolve(res.statusCode));});
      req.on('error', reject); req.setTimeout(25000, () => req.destroy(new Error('proxy bytes bounded timeout'))); req.write(request.prefix); req.write(bytes); req.end(request.tail);
    }); if(status===200){const list=await h.json(f,'/web/designs'); const asset=list.designs.find(row=>row.title===title); h.assert(asset); h.assert.equal(asset.size,size); h.assert.equal(asset.sha256,h.sha(bytes));} return status;
  }
  h.assert.equal(await sized(50 * 1024 * 1024), 200); h.assert.equal(await sized(50 * 1024 * 1024 + 1), 413);
  pass('next-rewrite-inclusive50MiB-and-over-limit', {accepted: 50 * 1024 * 1024, rejected: 50 * 1024 * 1024 + 1});
  const slow = await body('Synthetic Next31s streaming', 22); let started = Date.now();
  const slowStatus = await new Promise((resolve, reject) => {
    const req = http.request(f.origin + '/web/designs', {method: 'POST', headers: slow.headers}, res => {res.resume(); res.on('end', () => resolve(res.statusCode));});
    req.on('error', reject); req.setTimeout(45000, () => req.destroy(new Error('slow31s proxy bound'))); req.write(slow.prefix); req.write(Buffer.from('PK\x05\x06', 'binary'));
    setTimeout(() => req.end(Buffer.concat([Buffer.alloc(18), slow.tail])), 31000);
  });
  h.assert.equal(slowStatus, 200); h.assert(Date.now() - started >= 31000); pass('next-rewrite-slow-body-over-global30s', {elapsed_ms: Date.now() - started});
  const incomplete = await body('Synthetic Next120s incomplete', 22); started = Date.now();
  const outcome = await new Promise((resolve, reject) => {
    let settled = false; let req; const timer = setTimeout(() => {if (!settled) {settled = true; req.destroy(); reject(new Error('Next incomplete upload survived132s'));}}, 132000);
    const finish = data => {if (settled) return; settled = true; clearTimeout(timer); req.destroy(); resolve(data);};
    req = http.request(f.origin + '/web/designs', {method: 'POST', headers: incomplete.headers}, res => {res.resume(); res.on('end', () => finish({status: res.statusCode}));});
    req.on('error', error => finish({network_code: error.code || 'error'})); req.write(incomplete.prefix); req.write(Buffer.from('PK\x05\x06', 'binary'));
  });
  const elapsed = Date.now() - started; h.assert(elapsed >= 118000 && elapsed <= 128000, 'incomplete120s proxy deadline tolerance'); h.assert(!outcome.status || outcome.status >= 400);
  h.assert.equal((await fetch(f.go.origin + '/readyz')).status, 200); h.assert.equal((await fetch(f.origin + '/app/login')).status, 200);
  const counts = JSON.parse(h.cp.execFileSync(h.required('ACCORD_QA_PYTHON'), ['-c', "import sqlite3,json,sys; db=sqlite3.connect('file:'+sys.argv[1]+'?mode=ro',uri=True); print(json.dumps(db.execute(\"SELECT count(*),coalesce(sum(reserved),0) FROM design_assets WHERE status='pending' OR title='Synthetic Next120s incomplete'\").fetchone()))", h.path.join(f.state, 'web/web.db')], {encoding: 'utf8'}));
  h.assert.deepEqual(counts, [0, 0]); h.assert.equal(h.fs.readdirSync(h.path.join(f.state, 'web/staging')).length, 0);
  pass('next-rewrite-actual120s-incomplete-deadline-cleanup', {elapsed_ms: elapsed, ...outcome});
};
