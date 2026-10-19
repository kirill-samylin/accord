'use strict';
const h = require('./harness.cjs');
const crypto = require('node:crypto');
const {chromium} = require(h.required('ACCORD_QA_PLAYWRIGHT'));
async function login(browser, origin, file, options = {}, existingContext) {
  const ctx = existingContext || await browser.newContext({viewport: {width: 1440, height: 900}, ...options});
  const p = await ctx.newPage(); await p.goto(origin + '/web/login');
  await p.locator('#token-file').setInputFiles(file);
  const response = p.waitForResponse(r => r.request().method() === 'POST');
  await p.getByRole('button', {name: 'Войти в проект →', exact: true}).click();
  h.assert.equal((await response).status(), 303); await p.waitForURL(origin + '/web/');
  h.assert.equal(await p.locator('.identity').count(), 1);
  return {ctx, p};
}
const call = (worker, name, input = {}) => worker.call(name, {...(['get', 'events'].includes(name) ? {} : {request_id: crypto.randomUUID()}), ...input});
const machine = file => h.path.basename(file, '.token');
async function open(worker, file, run) {
  return call(worker, 'describe', {role: 'developer', run_id: run, machine: machine(file)});
}
function revoke(state, file) {
  const owner = machine(file);
  const listed = JSON.parse(h.cli(['token', 'list', '--db', h.path.join(state, 'board.db')]));
  const found = listed.find(t => t.owner === owner); h.assert(found, 'synthetic token metadata absent');
  h.cli(['token', 'revoke', '--db', h.path.join(state, 'board.db'), '--id', found.id]);
}
async function fixture() {
  h.fs.mkdirSync(h.root, {recursive: true, mode: 0o700}); h.fs.chmodSync(h.root, 0o700);
  const state = h.path.join(h.root, 'state-' + crypto.randomUUID()); h.fs.mkdirSync(state, {mode: 0o700});
  const human = h.token(state, 'human-' + crypto.randomUUID());
  const agent = h.token(state, 'agent-' + crypto.randomUUID());
  const secondAgent = h.token(state, 'second-agent-' + crypto.randomUUID());
  let srv, browser, worker, second;
  try {
  srv = await h.server(h.binary, state, await h.port(), true, {task: '3m'});
  browser = await chromium.launch({executablePath: h.required('ACCORD_QA_CHROME'), headless: true});
  worker = await new h.Worker(srv.origin, agent).start();
  second = await new h.Worker(srv.origin, secondAgent).start();
  const firstSession = await open(worker, agent, 'synthetic-first');
  const secondSession = await open(second, secondAgent, 'synthetic-second');
  const user = await login(browser, srv.origin, human);
  const staticFailures = []; user.ctx.on('response', r => {if(new URL(r.url()).pathname.startsWith('/web/static/') && r.status() >= 400) staticFailures.push({path: new URL(r.url()).pathname, status: r.status()});});
  return {staticFailures, state, human, agent, secondAgent, srv, browser, worker, second, firstSession, secondSession, ...user, origin: srv.origin};
  } catch (error) {
    const errors = [error];
    for (const close of [() => worker?.close(), () => second?.close(), () => browser?.close(), () => srv && h.stop(srv)]) {
      try {await close();} catch (cleanup) {errors.push(cleanup);}
    }
    throw new AggregateError(errors, 'fixture startup failed; own resource cleanup attempted');
  }
}
async function goto(f, url, expected = 200) {
  // Pace navigation below the real global admission budget (each HTML page
  // loads multiple CSS/JS assets); this is workload pacing, not a readiness wait.
  await h.sleep(Math.max(0, 200 - (Date.now() - (f.lastNavigation || 0)))); f.lastNavigation = Date.now();
  const r = await f.p.goto(f.origin + url); h.assert.equal(r.status(), expected, 'route status ' + url);
}
async function overview(f) {
  const r = await f.ctx.request.get(f.origin + '/web/data/overview'); h.assert.equal(r.status(), 200);
  h.assert.match(r.headers()['content-type'], /^application\/json/); return r.json();
}
async function nativeTasks(f) {
  const rows = []; let cursor = '';
  do {const response = await fetch(f.origin + '/api/v1/tasks?include_closed=true&limit=100' + (cursor ? '&cursor=' + encodeURIComponent(cursor) : ''), {headers: {Authorization: 'Bearer ' + h.fs.readFileSync(f.agent, 'utf8').trim(), 'X-Session': f.firstSession.session_id}}); h.assert.equal(response.status, 200); const page = await response.json(); rows.push(...page.tasks); cursor = page.next_cursor || '';} while(cursor);
  return rows;
}
async function seed(f) {
  async function epic(title) {
    await goto(f, '/web/tasks'); await f.p.locator('#create-type').selectOption('epic');
    await f.p.locator('#create-title').fill(title); await f.p.locator('#create-description').fill('Synthetic epic context');
    const response = f.p.waitForResponse(r => r.request().method() === 'POST');
    await f.p.locator('[data-create] button[type=submit]').click(); h.assert.equal((await response).status(), 303);
    await f.p.waitForURL(/\/web\/tasks\/E-\d+$/); return f.p.url().split('/').pop();
  }
  f.epic = await epic('Synthetic control epic');
  f.otherEpic = await epic('Synthetic second epic');
  f.tasks = [];
  for (let i = 0; i < 112; i++) {
    await h.sleep(12); // Keep synthetic seed below the real100requests/s admission.

    const title = i === 111 ? 'ЁЖиК <img src=x onerror="globalThis.qaXss=1"> beyond page' : 'Synthetic control ' + i;
    f.tasks.push(await call(f.worker, 'create', {type: 'task', epic_key: i === 110 ? f.otherEpic : f.epic,
      role: i === 109 ? 'analyst' : 'developer', title, description: i === 111 ? 'ПозДнЕе ОПИСАНИЕ' : 'Synthetic control context'}));
  }
  for (const [index, tokens] of [[0, undefined], [1, 17]]) {
    const claimed = await call(f.worker, 'next', {task_key: f.tasks[index].key});
    await call(f.worker, 'transition', {task_key: claimed.task.key, generation: claimed.task.lease.generation,
      state: 'done', evidence: {commit: '1'.repeat(40), run_summary: 'Synthetic completion'}, ...(tokens === undefined ? {} : {tokens_spent: tokens})});
  }
  await call(f.worker, 'transition', {task_key: f.tasks[2].key, state: 'cancelled', reason: 'Synthetic cancellation'});
  await call(f.worker, 'link', {from: f.tasks[3].key, to: f.tasks[4].key, type: 'blocks', condition: 'completed'});
}
// Install and pause before loading the timed UI. A fixed1hour gap gives CDP
// calls room without ever reading a moving browser Date back across RPC.
async function pausedClock(p, anchor = Date.now()) {
  await p.clock.install({time: new Date(anchor - 3600000)});
  await p.clock.pauseAt(new Date(anchor));
}
module.exports = {...h, crypto, login, call, open, revoke, fixture, goto, overview, nativeTasks, seed, pausedClock};
