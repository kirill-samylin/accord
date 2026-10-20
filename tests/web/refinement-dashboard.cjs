'use strict';
const h = require('./control-fixture.cjs');
const {assert, call, goto, sleep} = h;
const overview = r => r.url().endsWith('/web/data/overview');
const dashboard = r => r.url().endsWith('/web/data/dashboard');
const status = (p, text) => p.locator('[data-overview-status]').filter({hasText: text}).waitFor();
async function poll(p, advance = 15000) {
  const response = p.waitForResponse(dashboard); await p.clock.fastForward(advance);
  const r = await response; assert.equal(r.status(), 200); return r.json();
}
async function snapshot(p) {
  return {agents: await p.locator('[data-dashboard-agents]').innerHTML(), activity: await p.locator('[data-dashboard-activity]').innerHTML(),
    counters: await p.locator('[data-overview-value]').allTextContents(), time: await p.locator('[data-overview-time]').getAttribute('datetime')};
}
module.exports = async (f, pass) => {
  const p = f.p; await p.setViewportSize({width: 1440, height: 900});
  await h.pausedClock(p); await goto(f, '/web/');
  let counts = {overview: 0, dashboard: 0}, inflight = 0, maximum = 0;
  const started = r => {if (overview(r) || dashboard(r)) {counts[dashboard(r) ? 'dashboard' : 'overview']++; maximum = Math.max(maximum, ++inflight);}};
  const ended = r => {if (overview(r) || dashboard(r)) inflight--;};
  p.on('request', started); p.on('requestfinished', ended); p.on('requestfailed', ended);
  const first = p.waitForResponse(overview); await p.clock.fastForward(15000); assert.equal((await first).status(), 200);
  await status(p, 'Счётчики обновлены'); assert.deepEqual(counts, {overview: 1, dashboard: 0});
  const enabled = p.waitForResponse(dashboard); await p.locator('[data-dashboard-live]').check(); await p.clock.fastForward(1);
  assert.equal((await enabled).status(), 200); await status(p, 'агенты и события обновлены');
  assert.deepEqual(counts, {overview: 1, dashboard: 1});
  assert.equal(await p.locator('[data-dashboard-agents] .assigned-task a').innerText(), f.tasks[0].key + ' · ' + f.tasks[0].title);
  assert.equal(await p.locator('[data-dashboard-agents] img').count(), 0);
  await call(f.worker, 'release', {task_key: f.tasks[0].key, generation: f.longClaim.task.lease.generation, reason: 'Synthetic refresh release <img src=x onerror="globalThis.qaXss=4">'});
  const held = await call(f.worker, 'next', {task_key: f.tasks[4].key});
  const changed = await poll(p); await status(p, 'агенты и события обновлены');
  assert.equal(changed.overview.tasks_in_progress, 1); assert.equal(changed.overview.leased_tasks, 1);
  assert.equal(await p.locator('[data-overview-value=tasks_in_progress]').innerText().then(Number), changed.overview.tasks_in_progress);
  assert.equal(await p.locator('[data-dashboard-agents] .assigned-task a').innerText(), held.task.key + ' · ' + held.task.title);
  const cursors = await p.locator('[data-dashboard-activity] .event-row > .micro').allTextContents();
  assert.deepEqual(cursors.map(s => Number(s.trim().slice(1))), changed.activity.events.map(event => event.cursor));
  assert.equal(await p.locator('[data-dashboard-activity] img').count(), 0); assert.equal(await p.evaluate(() => globalThis.qaXss || 0), 0);
  assert.equal(counts.overview, 1); assert.equal(maximum, 1);
  const link = p.locator('[data-dashboard-agents] .entity-name').first(); await link.focus();
  await call(f.worker, 'release', {task_key: held.task.key, generation: held.task.lease.generation, reason: 'Synthetic focus release'});
  await poll(p); await status(p, 'Обновление списка отложено');
  assert(await link.evaluate(el => el === document.activeElement));
  assert.equal(await p.locator('[data-dashboard-agents] .assigned-task a').count(), 1, 'focused collection replaced');
  await p.locator('[data-dashboard-live]').focus(); await poll(p); await status(p, 'агенты и события обновлены');
  assert.equal(await p.locator('[data-dashboard-agents] .assigned-task a').count(), 0);
  const leaseAfterReads = await call(f.worker, 'get', {task_key: f.tasks[0].key});
  assert.equal(leaseAfterReads.state, 'ready');
  // Compare the actual native lease around read-only dashboard refreshes.
  const finalHeld = await call(f.worker, 'next', {task_key: f.tasks[0].key}); await poll(p);
  const before = await call(f.worker, 'get', {task_key: f.tasks[0].key});
  const good = await snapshot(p); await f.ctx.setOffline(true); await status(p, 'Нет сети');
  const count = counts.dashboard; await p.clock.fastForward(45000); assert.equal(counts.dashboard, count);
  assert.deepEqual(await snapshot(p), good);
  await f.ctx.setOffline(false); await poll(p, 1); await status(p, 'агенты и события обновлены');
  const retained = await snapshot(p);
  f.srv.child.kill('SIGSTOP');
  try {
    await p.clock.fastForward(15000);
    for (let i = 0; i < 100 && inflight !== 1; i++) await sleep(25);
    assert.equal(inflight, 1); await p.clock.fastForward(9000); await status(p, 'Последние данные сохранены');
    assert.deepEqual(await snapshot(p), retained);
  } finally {f.srv.child.kill('SIGCONT');}
  await poll(p, 60000); await status(p, 'агенты и события обновлены');
  const after = await call(f.worker, 'get', {task_key: finalHeld.task.key});
  assert.equal(after.lease.expires_at, before.lease.expires_at); assert.equal(after.lease.generation, before.lease.generation);
  pass('dashboard-default-optin-single-poll-native-release-claim-focus-safe-xss-no-renew', {counts: {...counts}, maximum_inflight: maximum, lease_unchanged: true});
  pass('dashboard-offline-timeout-last-good-recovery', {real_server_sigstop: true, clock: 'browser timers only'});
  h.revoke(f.state, f.human);
  const denied = p.waitForResponse(dashboard); await p.clock.fastForward(15000); assert.equal((await denied).status(), 401);
  await status(p, 'Вход больше не действует'); const terminal = counts.dashboard;
  await p.clock.fastForward(120000); assert.equal(counts.dashboard, terminal);
  pass('dashboard-real-revoke401-terminal-no-more-polls', {status: 401, requests_after_terminal: 0});
  p.off('request', started); p.off('requestfinished', ended); p.off('requestfailed', ended);
};
