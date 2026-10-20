'use strict';
// Real Go + production Next + native stdio MCP. No fabricated server replies.
const h = require('./harness.cjs');
const crypto = require('node:crypto');
const net = require('node:net');
const {chromium} = require(h.required('ACCORD_QA_PLAYWRIGHT'));
const NODE = h.required('ACCORD_QA_NODE');
const webDir = h.path.resolve(__dirname, '../../web');
const call = (worker, name, input = {}) => worker.call(name, {...(['get', 'events'].includes(name) ? {} : {request_id: crypto.randomUUID()}), ...input});
async function assertPortFree(n) {
  const socket = net.createServer();
  await new Promise((resolve, reject) => { socket.once('error', reject); socket.listen(n, '127.0.0.1', resolve); });
  await new Promise(resolve => socket.close(resolve));
}
async function ready(child, url, timeout = 30000) {
  const until = Date.now() + timeout;
  while (Date.now() < until) {
    h.assert.equal(child.exitCode, null, 'own process exited before readiness');
    try { if ((await fetch(url)).status === 200) return; } catch {}
    await h.sleep(100);
  }
  throw new Error('own process readiness timeout');
}
async function startGo(f, ttl = {}) {
  await assertPortFree(3101);
  const args = ['serve', '--db', h.path.join(f.state, 'board.db'), '--listen', '127.0.0.1:3101', '--task-ttl', ttl.task || '18s', '--session-ttl', ttl.session || '20m', '--web-origin', f.origin, '--web-data', h.path.join(f.state, 'web')];
  const log = h.fs.openSync(h.path.join(h.root, 'go.log'), 'a', 0o600);
  const child = h.cp.spawn(h.binary, args, {env: h.env, stdio: ['ignore', log, log]}); h.fs.closeSync(log);
  f.go = {child, origin: 'http://127.0.0.1:3101'}; (f.goProcesses ??= []).push(child.pid);
  await ready(child, f.go.origin + '/readyz');
}
async function json(f, path, ctx = f.ctx) {
  const response = await ctx.request.get(f.origin + path, {headers: {Accept: 'application/json'}});
  h.assert.equal(response.status(), 200, 'JSON GET status ' + path);
  const result = await response.json(); h.assert(result.ok && result.page, 'missing page DTO'); return result.page;
}
async function goto(f, path, p = f.p) {
  await h.sleep(70); // Pace many real requests below the unchanged admission budget.
  const response = await p.goto(f.origin + path);
  h.assert.equal(response.status(), 200, 'Next route HTTP status ' + path);
  await p.locator('h1').first().waitFor();
  await p.locator('.app-loading').waitFor({state: 'detached'});
  await p.waitForFunction(() => !document.querySelector('.refresh-indicator'));
  return response;
}
async function posted(p, action, status = 200) {
  const response = p.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.startsWith('/web/'));
  await action(); const r = await response; h.assert.equal(r.status(), status, 'browser mutation status'); return r.json();
}
async function login(f, file, options = {}, wrongFirst = false, existingContext) {
  const ctx = existingContext || await f.browser.newContext({viewport: {width: 1440, height: 900}, ...options});
  const p = await ctx.newPage(); observe(f,p); const trace = []; p.on('requestfailed', r => trace.push({path:new URL(r.url()).pathname, failure:r.failure()?.errorText})); p.on('response', r => {if(new URL(r.url()).pathname.startsWith('/web')) trace.push({path:new URL(r.url()).pathname,status:r.status(),location:r.headers().location});}); p.setDefaultTimeout(12000); p.setDefaultNavigationTimeout(20000);
  await goto(f, '/app/login', p);
  if (wrongFirst) {
    await p.locator('[name=token]').fill('synthetic-invalid-token');
    await posted(p, () => p.getByRole('button', {name: 'Войти в Accord', exact: true}).click(), 401);
    await p.locator('.mutation-notice').waitFor(); h.assert(await p.locator('[name=token]').isEnabled(), 'wrong token locked login');
  }
  await p.getByRole('button', {name: 'Файл токена', exact: true}).click();
  await p.locator('[name=token_file]').setInputFiles(file);
  await posted(p, () => p.getByRole('button', {name: 'Войти в Accord', exact: true}).click());
  await p.waitForURL(f.origin + '/app'); try {await p.locator('.sidebar').first().waitFor();} catch(error) {await p.screenshot({animations:'disabled',path: h.path.join(h.root, 'login-failure.png'), fullPage:true}); h.fs.writeFileSync(h.path.join(h.root, 'login-failure-text.txt'), await p.locator('body').innerText(), {mode:0o600}); h.fs.writeFileSync(h.path.join(h.root, 'login-failure-network.json'), JSON.stringify(trace,null,2), {mode:0o600}); throw error;}
  return {ctx, p};
}
async function fixture() {
  h.fs.mkdirSync(h.root, {recursive: true, mode: 0o700}); h.fs.chmodSync(h.root, 0o700);
  const f = {state: h.path.join(h.root, 'state-' + crypto.randomUUID()), sourceErrors: [], consoleErrors: [], failedAssets: [], contexts: [], findings: []};
  h.fs.mkdirSync(f.state, {mode: 0o700}); f.human = h.token(f.state, 'qa-human'); f.other = h.token(f.state, 'qa-second'); f.agent = h.token(f.state, 'qa-developer'); f.reviewer = h.token(f.state,'qa-reviewer');
  f.origin = 'http://127.0.0.1:' + await h.port();
  try {
    await startGo(f);
    const log = h.fs.openSync(h.path.join(h.root, 'next.log'), 'a', 0o600);
    f.next = {child: h.cp.spawn(NODE, [h.path.join(webDir, 'node_modules/next/dist/bin/next'), 'start', '-H', '127.0.0.1', '-p', new URL(f.origin).port], {cwd: webDir, env: {...h.env, NODE_ENV: 'production', ACCORD_GO_ORIGIN: f.go.origin}, stdio: ['ignore', log, log]})}; h.fs.closeSync(log);
    await ready(f.next.child, f.origin + '/app/login');
    f.browser = await chromium.launch({executablePath: h.required('ACCORD_QA_CHROME'), headless: true});
    f.worker = await new h.Worker(f.go.origin, f.agent).start();
    f.agentSession = await call(f.worker, 'describe', {role: 'developer', run_id: 'synthetic-next-native', machine: 'qa-developer'}); f.reviewWorker=await new h.Worker(f.go.origin,f.reviewer).start(); f.reviewSession=await call(f.reviewWorker,'describe',{role:'reviewer',run_id:'synthetic-review-session',machine:'qa-reviewer'});
    Object.assign(f, await login(f, f.human, {}, true)); f.contexts.push(f.ctx);
    f.ctx.on('page', p => observe(f, p));
    f.goBinary = h.sha(h.fs.readFileSync(h.binary));
    return f;
  } catch (error) { error.fixtureDiagnostics={page_errors:f.sourceErrors,console_errors:f.consoleErrors,failed_assets:f.failedAssets,processes:{go:f.go?.child.pid,next:f.next?.child.pid,worker:f.worker?.proc.pid}}; try {await cleanup(f);} catch (cleanupError) {error.message += '; cleanup: ' + cleanupError.message;} throw error; }
}
function observe(f, p) {
  p.on('pageerror', error => f.sourceErrors.push(String(error.message).slice(0, 180)));
  p.on('console', msg => {if (msg.type() === 'error' && /hydration|Content Security Policy|Refused to|Minified React|react\.dev/i.test(msg.text())) f.consoleErrors.push(msg.text().slice(0, 180));});
  p.on('response', r => {if (/\/_next\/static\/|\/fonts\//.test(r.url()) && r.status() >= 400) f.failedAssets.push({status: r.status(), path: new URL(r.url()).pathname});});
}
async function stopNext(proc) {
  if (!proc) return; if(proc.child.exitCode !== null || proc.child.signalCode){h.assert(proc.child.exitCode===0||proc.child.exitCode===143||proc.child.signalCode==='SIGTERM','Next exited unexpectedly before cleanup');return;}
  const done = new Promise(resolve => proc.child.once('exit', resolve)); proc.child.kill('SIGTERM');
  let timer; await Promise.race([done, new Promise(resolve => {timer = setTimeout(() => {proc.child.kill('SIGKILL'); resolve();}, 20000);})]); clearTimeout(timer);
  h.assert(proc.child.exitCode === 0 || proc.child.exitCode === 143 || proc.child.signalCode === 'SIGTERM', 'Next did not stop cleanly (exit='+proc.child.exitCode+', signal='+proc.child.signalCode+')');
}
async function stopGo(proc){if(!proc)return;if(proc.child.exitCode!==null||proc.child.signalCode)h.assert.equal(proc.child.exitCode,0,'Go exited unexpectedly before cleanup');else await h.stop(proc);}
async function cleanup(f) {
  const failures = [];
  for (const fn of [() => f.worker?.close(), () => f.reviewWorker?.close(), () => f.browser?.close(), () => f.next && stopNext(f.next), () => f.go && stopGo(f.go)]) {
    try {await fn();} catch (error) {failures.push(String(error.message));}
  }
  if (failures.length) throw new Error('own process cleanup failed: ' + failures.join(', '));
}
async function create(f, values, p = f.p) {
  const dialog = p.getByRole('dialog');
  await p.getByRole('button', {name: 'Новая задача', exact: true}).first().click(); await dialog.waitFor();
  await dialog.locator('[name=type]').selectOption(values.type || 'task');
  for (const [name, value] of Object.entries(values)) {if (name === 'type') continue; const el = dialog.locator(`[name=${name}]`); if (name === 'role') await el.selectOption(value); else await el.fill(value);}
  const result = await posted(p, () => dialog.getByRole('button', {name: 'Создать', exact: true}).click());
  await p.waitForURL(/\/app\/tasks\/[ET]-\d+$/); await dialog.waitFor({state: 'detached'});
  return result.redirect.split('/').pop();
}
async function get(f, key) {return call(f.worker, 'get', {task_key: key});}
function revoke(f, file) {
  const name = h.path.basename(file, '.token'); const rows = JSON.parse(h.cli(['token', 'list', '--db', h.path.join(f.state, 'board.db')]));
  const token = rows.find(row => row.owner === name); h.assert(token); h.cli(['token', 'revoke', '--db', h.path.join(f.state, 'board.db'), '--id', token.id]);
}
module.exports = {...h, call, fixture, cleanup, json, goto, posted, login, create, get, startGo, stopGo, revoke, observe, webDir, NODE};
