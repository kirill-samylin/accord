'use strict';
// Repeatable browser/native acceptance; no HTTP mocks or browser header overrides.
const h = require('./harness.cjs');
const {fs, path, cp, assert, required, binary, root, sha, sleep, cli, token, port, server, stop, Worker} = h;
const {chromium} = require(required('ACCORD_QA_PLAYWRIGHT'));
const http = require('node:http');
const report = {schema: 1, test_sha256: h.manifest(), node: process.version, scope: process.env.ACCORD_QA_STOP_AFTER || 'full', checks: [], limitations: ['one macOS host; no physical second PC', 'HTTP loopback fixture; production HTTPS deployment is separate']};
let srv, browser, worker, state, origin, n, human, second, native, sid, epic, task, question;
const pass = (id, measurements = {}) => {report.checks.push({id, result: 'PASS', ...measurements}); console.log(id + ' PASS');};
const selector = (suffix) => `form[action$="${suffix}"]`;
async function postButton(page, button, expected = 303) {
  const response = page.waitForResponse(r => r.request().method() === 'POST');
  await button.click(); const r = await response; assert.equal(r.status(), expected, 'unexpected form status');
  if (expected === 303) await page.waitForLoadState('networkidle');
  return r;
}
async function login(file, js = true) {
  const ctx = await browser.newContext({javaScriptEnabled: js, viewport: {width: 1280, height: 900}});
  const p = await ctx.newPage(); await p.goto(origin + '/web/login');
  await p.locator('#token-file').setInputFiles(file);
  const r = await postButton(p, p.getByRole('button', {name: 'Войти', exact: true}));
  assert.equal(await r.headerValue('referrer-policy'), 'same-origin');
  assert.equal(await p.locator('.identity').count(), 1, 'login failed');
  return {ctx, p};
}
async function createUI(p, type, title, role) {
  await p.goto(origin + '/web/?view=tasks');
  await p.locator('#create-type').selectOption(type);
  await p.locator('#create-title').fill(title);
  await p.locator('#create-description').fill('Synthetic process/browser QA');
  if (type === 'task') {await p.locator('#create-epic').fill(epic); await p.locator('#create-role').selectOption(role);}
  await postButton(p, p.locator('[data-create] button[type=submit]'));
  await p.waitForURL(/\/web\/tasks\/[ET]-\d+$/);
  return p.url().split('/').at(-1);
}
async function nativeAPI(url, headers = {}) { return fetch(origin + url, {headers}); }
const auth = () => ({Authorization: 'Bearer ' + fs.readFileSync(native, 'utf8').trim(), 'X-Session': sid});
async function getTask(key) {return worker.call('get', {task_key: key});}
async function newQuestion(title) {
  return worker.call('create', {request_id: crypto.randomUUID(), type: 'task', epic_key: epic, role: 'human', title, description: 'Synthetic question context', priority: 1});
}
const crypto = require('node:crypto');
async function upload(p, filename, bytes, scope, key = '', design = '') {
  await p.goto(origin + '/web/designs' + (design ? '?design_id=' + design : ''));
  await p.locator('#design-title').fill('Synthetic ' + filename);
  await p.locator('#design-scope').selectOption(scope);
  await p.locator('#design-scope-key').fill(key);
  if (scope === 'task') await p.locator('#design-epic').fill(epic);
  await p.locator('#design-file').setInputFiles({name: filename, mimeType: 'application/octet-stream', buffer: bytes});
  await postButton(p, p.locator('[data-upload] button[type=submit]'));
  const list = await worker.resource('accord://designs');
  const found = list.assets.find(a => a.filename === filename && (!design || a.design_id === design));
  assert(found, 'uploaded metadata absent from native resource');
  assert.equal(found.sha256, sha(bytes)); assert.equal(found.scope, scope);
  assert.equal(found.scope_key, scope === 'project' ? 'agent-board' : key);
  const res = await nativeAPI(found.download_url, auth()); assert.equal(res.status, 200);
  assert.equal(sha(Buffer.from(await res.arrayBuffer())), sha(bytes));
  assert.match(res.headers.get('content-disposition'), /^attachment/);
  assert.equal(res.headers.get('content-type'), 'application/octet-stream');
  assert.equal(res.headers.get('x-content-type-options'), 'nosniff'); assert.match(res.headers.get('content-security-policy'), /sandbox/);
  const single = await worker.resource('accord://designs/' + found.asset_id);
  assert.equal(single.sha256, found.sha256);
  return found;
}
async function slowStream(p) {
  await p.goto(origin + '/web/designs');
  const fields = await p.locator('[data-upload]').evaluate(f => Object.fromEntries(new FormData(f)));
  fields.title = 'Synthetic slow upload'; fields.scope = 'project'; fields.scope_key = ''; fields.epic_key = ''; fields.design_id = '';
  const boundary = 'AccordQABoundary'; let pre = '';
  for (const [key, value] of Object.entries(fields)) if (key !== 'file') pre += `--${boundary}\r\nContent-Disposition: form-data; name="${key}"\r\n\r\n${value}\r\n`;
  pre += `--${boundary}\r\nContent-Disposition: form-data; name="file"; filename="slow.zip"\r\nContent-Type: application/zip\r\n\r\n`;
  const bytes = Buffer.from('PK\x05\x06' + '\x00'.repeat(18), 'binary');
  const tail = Buffer.concat([bytes.subarray(4), Buffer.from(`\r\n--${boundary}--\r\n`)]);
  const cookies = await p.context().cookies();
  const cookie = cookies.map(c => c.name + '=' + c.value).join('; ');
  const started = Date.now();
  const status = await new Promise((resolve, reject) => {
    const req = http.request(origin + '/web/designs', {method: 'POST', headers: {Origin: origin, Cookie: cookie, 'Content-Type': 'multipart/form-data; boundary=' + boundary, 'Content-Length': Buffer.byteLength(pre) + bytes.length + Buffer.byteLength(`\r\n--${boundary}--\r\n`)}}, res => {res.resume(); res.on('end', () => resolve(res.statusCode));});
    req.on('error', reject); req.setTimeout(45000, () => req.destroy(new Error('slow upload exceeded bound')));
    req.write(pre); req.write(bytes.subarray(0, 4)); setTimeout(() => req.end(tail), 31000);
  });
  assert.equal(status, 303, 'upload longer than global30s failed');
  pass('slow-stream-over-30s', {elapsed_ms: Date.now() - started, status});
}
async function main() {
  fs.mkdirSync(root, {recursive: true, mode: 0o700}); fs.chmodSync(root, 0o700);
  state = path.join(root, 'state-' + crypto.randomUUID()); fs.mkdirSync(state, {mode: 0o700});
  human = token(state, 'human-' + crypto.randomUUID()); second = token(state, 'second-' + crypto.randomUUID()); native = token(state, 'worker-' + crypto.randomUUID());
  n = await port(); srv = await server(binary, state, n); origin = srv.origin;
  browser = await chromium.launch({executablePath: required('ACCORD_QA_CHROME'), headless: true, args: ['--no-first-run']});
  report.browser = browser.version(); report.binary_sha256 = sha(fs.readFileSync(binary)); report.old_binary_sha256 = sha(fs.readFileSync(required('ACCORD_QA_OLD_BINARY')));
  const a = await login(human); const b = await login(second); const basic = await login(human, false);
  await basic.ctx.close(); pass('login-real-file-js-and-basic-post');
  for (const size of [{width: 1280, height: 900}, {width: 390, height: 844}]) {
    await a.p.setViewportSize(size);
    for (const url of ['/web/?view=questions', '/web/?view=tasks', '/web/designs', '/web/history']) {
      const r = await a.p.goto(origin + url); assert.equal(r.status(), 200);
      assert.equal(await a.p.locator('nav[aria-label="Основные разделы"] a').count(), 4);
      assert(await a.p.locator('h1').isVisible());
      assert(await a.p.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'horizontal overflow');
    }
  }
  pass('four-sections-desktop-mobile', {viewports: 2, pages: 8});
  epic = await createUI(a.p, 'epic', 'Synthetic QA epic');
  task = await createUI(a.p, 'task', 'Synthetic <img src=x onerror="globalThis.qaXss=1"> task', 'developer');
  assert.equal(await a.p.evaluate(() => globalThis.qaXss || 0), 0);
  assert.equal(await a.p.locator('article h2 img').count(), 0);
  worker = await new Worker(origin, native).start();
  const listed = await worker.rpc('tools/list', {}); assert.equal(listed.tools.length, 12);
  const machine = path.basename(native, '.token');
  const opened = await worker.call('describe', {request_id: 'qa-open', role: 'developer', run_id: 'qa-' + crypto.randomUUID(), machine}); sid = opened.session_id;
  report.process_roles = {server_pid: srv.child.pid, native_stdio_worker_pid: worker.proc.pid, browser: 'headless Chrome', client: 'Playwright'};
  const claimed = await worker.call('next', {request_id: 'qa-next', task_key: task}); assert.equal(claimed.task.key, task);
  pass('human-create-native-12-tools-describe-next-xss');
  question = (await newQuestion('Synthetic decision')).key;
  await a.p.goto(origin + '/web/tasks/' + question); await postButton(a.p, a.p.getByRole('button', {name: 'Взять и ответить'}));
  const before = await getTask(question); await a.p.reload(); const reread = await getTask(question);
  assert.equal(reread.lease.expires_at, before.lease.expires_at, 'read renewed lease');
  const beat = a.p.waitForResponse(r => r.url().endsWith('/web/heartbeat')); await a.p.locator('[data-heartbeat-toggle]').click(); assert.equal((await beat).status(), 200);
  const renewed = await getTask(question); assert(Date.parse(renewed.lease.expires_at) > Date.parse(before.lease.expires_at));
  await a.p.locator('[data-heartbeat-toggle]').click();
  const url = a.p.url(); const oldID = await a.p.locator(selector('/answer') + ' [name=action_id]').inputValue();
  await a.p.locator('#answer').fill('   '); await postButton(a.p, a.p.getByRole('button', {name: 'Ответить и завершить'}), 400);
  await a.p.locator('#answer-error').waitFor({state: 'visible'});
  assert.equal(await a.p.locator('#answer').inputValue(), '   '); assert.equal(a.p.url(), url);
  assert.equal(await a.p.locator('#answer').getAttribute('aria-invalid'), 'true');
  assert.notEqual(await a.p.locator(selector('/answer') + ' [name=action_id]').inputValue(), oldID);
  await a.p.locator('#answer').fill('Synthetic final answer <script>globalThis.qaXss=2</script>');
  await postButton(a.p, a.p.getByRole('button', {name: 'Ответить и завершить'}));
  const answered = await getTask(question); assert.equal(answered.state, 'done'); assert.match(answered.evidence.answer, /Synthetic final answer/);
  const events = await worker.call('events', {since_cursor: 0, limit: 200, task_key: question});
  const decisions = events.events.filter(e => e.type === 'human_answered'); assert.equal(decisions.length, 1);
  await a.p.goto(origin + '/web/history'); assert.match(await a.p.locator('main').innerText(), /Synthetic final answer/); assert.equal(await a.p.evaluate(() => globalThis.qaXss || 0), 0);
  pass('explicit-heartbeat-read-no-renew-validation-correct-answer-history');
  if (process.env.ACCORD_QA_STOP_AFTER === 'browser-core') {report.result = 'PASS'; return;}
  const race = (await newQuestion('Synthetic race')).key;
  await Promise.all([a.p.goto(origin + '/web/tasks/' + race), b.p.goto(origin + '/web/tasks/' + race)]);
  const statuses = await Promise.all([a.p, b.p].map(async p => {const response = p.waitForResponse(r => r.request().method() === 'POST'); await p.getByRole('button', {name: 'Взять и ответить'}).click(); return (await response).status();}));
  assert.deepEqual([...statuses].sort(), [303, 409]);
  const winner = statuses[0] === 303 ? a.p : b.p;
  await winner.locator('#answer').waitFor(); await winner.locator('#answer').fill('Synthetic stale draft');
  const held = await getTask(race); const until = Date.parse(held.lease.expires_at);
  while (Date.now() <= until + 100) await sleep(Math.min(500, until + 101 - Date.now()));
  await postButton(winner, winner.getByRole('button', {name: 'Ответить и завершить'}), 409);
  assert.equal(await winner.locator('#answer').inputValue(), 'Synthetic stale draft'); assert(await winner.getByRole('button', {name: 'Ответить и завершить'}).isDisabled());
  pass('two-browser-exact-claim-race-stale409-draft', {statuses});
  if (process.env.ACCORD_QA_STOP_AFTER === 'browser-race') {report.result = 'PASS'; return;}
  const releaseQuestion = (await newQuestion('Synthetic release')).key;
  await a.p.goto(origin + '/web/tasks/' + releaseQuestion); await postButton(a.p, a.p.getByRole('button', {name: 'Взять и ответить'}));
  await a.p.getByText('Вернуть вопрос в очередь', {exact: true}).click();
  await a.p.locator('#release-reason').fill('   ');
  await postButton(a.p, a.p.getByRole('button', {name: 'Вернуть в очередь', exact: true}), 400);
  await a.p.locator('#release-error').waitFor({state: 'visible'}); assert.equal(await a.p.locator('#release-reason').inputValue(), '   ');
  await a.p.locator('#release-reason').fill('Synthetic return to queue');
  await postButton(a.p, a.p.getByRole('button', {name: 'Вернуть в очередь', exact: true}));
  const released = await getTask(releaseQuestion); assert.equal(released.state, 'ready'); assert(!released.lease);
  pass('release-whitespace400-corrected-ready');
  const png = await a.p.screenshot({clip: {x: 0, y: 0, width: 1, height: 1}});
  const first = await upload(a.p, 'safe.png', png, 'task', task);
  const v2 = await upload(a.p, 'safe-v2.svg', Buffer.from('<svg xmlns="http://www.w3.org/2000/svg" onload="globalThis.qaXss=3"><script>globalThis.qaXss=3</script></svg>'), 'task', task, first.design_id);
  assert.equal(v2.version, 2); assert.equal(first.version, 1);
  await upload(a.p, 'opaque.html', Buffer.from('<!doctype html><script>globalThis.qaXss=4</script><p>Synthetic</p>'), 'epic', epic);
  await upload(a.p, 'opaque.zip', Buffer.from('PK\x05\x06' + '\x00'.repeat(18), 'binary'), 'project');
  await a.p.goto(origin + '/web/designs'); assert.equal(await a.p.locator('img.design-preview').count(), 1); assert.equal(await a.p.evaluate(() => globalThis.qaXss || 0), 0);
  await a.p.waitForFunction(() => document.querySelector('img.design-preview')?.naturalWidth > 0);
  const anonymousFile = await fetch(origin + '/web/designs/' + first.asset_id + '/download', {redirect: 'manual'}); assert.notEqual(anonymousFile.status, 200, 'anonymous browser file access');
  pass('design-browser-upload-3scopes-versions-native-resource-rest-sha');
  let foreign = false; try {await worker.resource('accord://designs?session_id=foreign-session');} catch {foreign = true;} assert(foreign, 'foreign resource session accepted');
  for (const headers of [{}, {Authorization: 'Bearer wrong'}, {Authorization: 'Bearer ' + (await a.ctx.cookies()).find(c => c.name.includes('session'))?.value}, {...auth(), Origin: ''}, {...auth(), Origin: 'null'}]) {
    const r = await nativeAPI('/api/v1/designs', headers); assert([401, 403].includes(r.status), 'native auth/origin boundary bypass');
  }
  const csrf = await a.p.locator('[data-upload] [name=csrf]').inputValue();
  const bad = await a.ctx.request.post(origin + '/web/tasks', {headers: {Origin: origin}, form: {csrf: 'invalid', action_id: 'invalid', type: 'epic', title: 'rejected'}}); assert.equal(bad.status(), 403);
  const noOrigin = await a.ctx.request.post(origin + '/web/tasks', {form: {csrf}}); assert.equal(noOrigin.status(), 403);
  const nullOrigin = await a.ctx.request.post(origin + '/web/tasks', {headers: {Origin: 'null'}, form: {csrf}}); assert.equal(nullOrigin.status(), 403);
  pass('auth-native-origin-csrf-foreign-session');
  await a.p.locator('#design-title').fill('Oversized synthetic');
  const oversized = path.join(root, 'oversized.zip');
  const oversizedFD = fs.openSync(oversized, 'wx', 0o600); fs.ftruncateSync(oversizedFD, 50 * 1024 * 1024 + 1); fs.closeSync(oversizedFD);
  await a.p.locator('#design-file').setInputFiles(oversized);
  let sent = false; const listener = r => {if(r.method() === 'POST' && r.url().endsWith('/web/designs')) sent = true;}; a.p.on('request', listener);
  await a.p.locator('[data-upload] button[type=submit]').click(); await a.p.locator('.submit-status').filter({hasText: 'превышает 50 МиБ'}).waitFor(); assert.equal(sent, false); a.p.off('request', listener);
  pass('browser-50MiB-limit-preflight');
  await slowStream(a.p);
  const savedAssets = await worker.resource('accord://designs');
  // Old C2 projects only the fields it knew. Full new event bytes are checked
  // independently through state-hash.py before and after both binary versions.
  const eventIdentity = rows => sha(Buffer.from(JSON.stringify(rows.map(({cursor, type, task_key, session_id, at}) => ({cursor, type, task_key, session_id, at})))));
  const eventHash = eventIdentity(decisions);
  await worker.close(); worker = null; await stop(srv); srv = await server(binary, state, n);
  await a.p.goto(origin + '/web/history'); assert(!a.p.url().endsWith('/login')); assert.match(await a.p.locator('main').innerText(), /Synthetic final answer/);
  await a.p.goto(origin + '/web/tasks/' + task); assert.match(await a.p.locator('main').innerText(), /Synthetic/);
  await a.p.goto(origin + '/web/designs'); assert.equal(await a.p.locator('.design-card').count(), savedAssets.assets.length);
  for (const asset of savedAssets.assets) {const r = await a.ctx.request.get(origin + '/web/designs/' + asset.asset_id + '/download'); assert.equal(r.status(), 200); assert.equal(sha(await r.body()), asset.sha256);}
  pass('restart-browser-cookie-tasks-history-design-bytes', {assets: savedAssets.assets.length});
  await stop(srv); srv = null;
  const lifecycle = cp.spawnSync(required('ACCORD_QA_PYTHON'), [path.join(__dirname, 'lifecycle.py'), state, root, binary, required('ACCORD_QA_BACKUP_MODULE')], {encoding: 'utf8', env: h.env, timeout: 30000});
  assert.equal(lifecycle.status, 0, 'offline snapshot/auth reset failed (output suppressed)');
  const restored = JSON.parse(lifecycle.stdout).restored;
  pass('offline-snapshot-integrity-hashes-restore-auth-reset');
  const boardHash = () => cp.execFileSync(required('ACCORD_QA_PYTHON'), [path.join(__dirname, 'state-hash.py'), state, question], {encoding: 'utf8'}).trim();
  const originalBoard = boardHash();
  const beforeFiles = JSON.parse(cp.execFileSync(required('ACCORD_QA_PYTHON'), ['-c', 'import pathlib,hashlib,json,sys; p=pathlib.Path(sys.argv[1]); print(json.dumps({str(f.relative_to(p)):hashlib.sha256(f.read_bytes()).hexdigest() for f in p.rglob("*") if f.is_file()}))', path.join(state, 'web')], {encoding:'utf8'}));
  srv = await server(required('ACCORD_QA_OLD_BINARY'), state, n, false);
  const oldRead = JSON.parse(cli(['call', 'describe', '--server', origin, '--token-file', native], {request_id: 'old-read', role: 'developer', run_id: 'old-qa', machine}));
  const oldEvents = JSON.parse(cli(['call', 'events', '--server', origin, '--token-file', native], {session_id: oldRead.session_id, since_cursor: 0, limit: 200, task_key: question}));
  assert.equal(eventIdentity(oldEvents.events.filter(e => e.type === 'human_answered')), eventHash);
  await stop(srv); srv = null;
  const afterFiles = JSON.parse(cp.execFileSync(required('ACCORD_QA_PYTHON'), ['-c', 'import pathlib,hashlib,json,sys; p=pathlib.Path(sys.argv[1]); print(json.dumps({str(f.relative_to(p)):hashlib.sha256(f.read_bytes()).hexdigest() for f in p.rglob("*") if f.is_file()}))', path.join(state, 'web')], {encoding:'utf8'}));
  assert.deepEqual(afterFiles, beforeFiles); assert.equal(boardHash(), originalBoard, 'old C2 altered raw answer/event rows or schema');
  srv = await server(binary, state, n); await a.p.goto(origin + '/web/history'); assert.match(await a.p.locator('main').innerText(), /Synthetic final answer/);
  await stop(srv); srv = null; assert.equal(boardHash(), originalBoard, 'new restart altered raw answer/event rows or schema');
  pass('actual-old-C2-rollback-new-restart-events-webfiles', {raw_answer_events_schema_sha256: originalBoard});
  srv = await server(binary, restored, n);
  const revoked = await nativeAPI('/api/v1/designs', auth()); assert.equal(revoked.status, 401);
  await a.p.goto(origin + '/web/history'); assert(a.p.url().endsWith('/web/login') || (await a.p.locator('.message.error').count()) > 0);
  const fresh = token(restored, 'restored-' + crypto.randomUUID()); const c = await login(fresh);
  await c.p.goto(origin + '/web/history'); assert.match(await c.p.locator('main').innerText(), /Synthetic final answer/);
  for (const asset of savedAssets.assets) {const r = await c.ctx.request.get(origin + '/web/designs/' + asset.asset_id + '/download'); assert.equal(r.status(), 200); assert.equal(sha(await r.body()), asset.sha256);}
  pass('restored-old-token-cookie-revoked-fresh-login-history-files');
  report.result = 'PASS';
}
main().catch(async err => {report.result = 'FAIL'; report.failure = String(err.message).slice(0, 240); console.error(report.failure); if(browser) for(const c of browser.contexts()) for(const p of c.pages()) {await p.screenshot({path:path.join(root, 'failure-' + Date.now() + '.png'), fullPage:true}).catch(()=>{});}}).finally(async () => {
  for (const cleanup of [() => worker?.close(), () => browser?.close(), () => srv && stop(srv)]) {
    try {await cleanup();} catch {report.result = 'FAIL'; report.cleanup_failure = true;}
  }
  report.finished_at = new Date().toISOString(); fs.writeFileSync(path.join(root, 'results.json'), JSON.stringify(report, null, 2), {mode:0o600});
  if(report.result !== 'PASS') process.exitCode = 1;
});
