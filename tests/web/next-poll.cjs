'use strict';
const h = require('./next-fixture.cjs');
const dashboard = r => new URL(r.url()).pathname === '/web/data/dashboard';
module.exports = async (f, pass) => {
  const p = f.p; await p.setViewportSize({width: 1440, height: 900}); await p.clock.setSystemTime(new Date());
  let requests = 0, inflight = 0, maximum = 0;
  const started = r => {if (dashboard(r)) {requests++; maximum = Math.max(maximum, ++inflight);}};
  const ended = r => {if (dashboard(r)) inflight--;}; p.on('request', started); p.on('requestfinished', ended); p.on('requestfailed', ended); await h.goto(f,'/app'); await p.getByRole('button',{name:'Обновить состояние',exact:true}).waitFor({state:'visible'});
  const snapshot = async () => ({metrics: await p.locator('.metric-value').allTextContents(), agents: await p.locator('.agent-strip').innerHTML(), events: await p.locator('.dashboard-bottom .event-list').innerHTML()});
  const first = p.waitForResponse(dashboard); await p.getByRole('button', {name: 'Обновить состояние', exact: true}).click(); h.assert.equal((await first).status(), 200); await h.sleep(50);
  const good = await snapshot(); await f.ctx.setOffline(true); const count = requests; await p.clock.fastForward(45000); h.assert.equal(requests, count); h.assert.deepEqual(await snapshot(), good);
  const resumed = p.waitForResponse(dashboard); await f.ctx.setOffline(false); h.assert.equal((await resumed).status(), 200); await h.sleep(70);
  const retained = await snapshot(); f.go.child.kill('SIGSTOP');
  try {
    const active = p.waitForRequest(dashboard); await p.getByRole('button', {name: 'Обновить состояние', exact: true}).click(); await active; await h.sleep(50); h.assert.equal(inflight, 1);
    await p.clock.fastForward(9000); await p.locator('.error-notice').filter({hasText: 'Показаны данные'}).waitFor(); h.assert.deepEqual(await snapshot(), retained); const timedout = requests;
    await p.clock.fastForward(15000); h.assert.equal(requests, timedout, 'failed poll ignored backoff'); h.assert(maximum <= 1, 'concurrent dashboard reads');
  } finally {f.go.child.kill('SIGCONT');}
  const recovery = p.waitForResponse(dashboard); await p.getByRole('button', {name: 'Обновить состояние', exact: true}).click(); h.assert.equal((await recovery).status(), 200); await p.locator('.error-notice').waitFor({state: 'detached'});
  await p.clock.setSystemTime(new Date()); p.off('request', started); p.off('requestfinished', ended); p.off('requestfailed', ended);
  pass('dashboard-real-sigstop8s-timeout-backoff-singleflight-offline-lastgood-recovery', {maximum_inflight: maximum});
};
module.exports.dashboard = dashboard;
