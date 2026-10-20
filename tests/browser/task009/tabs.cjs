'use strict';
const {assert, origin} = require('../../process/task009/harness.cjs');
async function run(f, owner, pass) {
  const p = owner.page;
  let release;
  let caught;
  const held = new Promise(resolve => {caught = resolve;});
  const gate = new Promise(resolve => {release = resolve;});
  const match = url => url.pathname === '/web/p/' + f.a + '/data/dashboard';
  let observed = 0;
  let routeDone, networkDone, observedRequest, interceptError;
  const routed = new Promise(resolve => {routeDone = resolve;});
  const settled = new Promise(resolve => {networkDone = resolve;});
  const finished = request => {if (request === observedRequest) networkDone({finished: true});};
  const failed = request => {if (request === observedRequest) networkDone({cancelled: request.failure()?.errorText === 'net::ERR_ABORTED'});};
  p.on('requestfinished', finished); p.on('requestfailed', failed);
  await p.route(match, async route => {
    if (observedRequest) return route.continue();
    observedRequest = route.request();
    try {
      const actual = await route.fetch({timeout: 10000}); assert.equal(actual.status(), 200);
      observed++;
      caught(); await gate;
      try {await route.fulfill({response: actual}); routeDone({fulfilled: true});}
      catch {routeDone({fulfilled: false});}
    } catch (error) {interceptError = error; caught(); routeDone({error: true}); await route.abort('failed').catch(() => {});}
  });
  try {
    await p.goto(origin + '/app/p/' + f.a);
    await p.locator('.control-shell').waitFor();
    let timer;
    await Promise.race([held, new Promise((_, reject) => {timer = setTimeout(() => reject(new Error('actual A dashboard read was not observed')), 12000);})]); clearTimeout(timer);
    if (interceptError) throw interceptError;
    await p.getByRole('button', {name: 'Переключить проект', exact: true}).click();
    await p.getByRole('menuitem', {name: /QA Beta/}).click();
    await p.waitForURL(new RegExp('/app/p/' + f.b));
    await p.locator('.control-shell').waitFor();
    assert.equal(await p.locator('.project-switch strong').innerText(), 'QA Beta');
    release();
    let completionTimer;
    const [delivery, network] = await Promise.race([Promise.all([routed, settled]), new Promise((_, reject) => {completionTimer = setTimeout(() => reject(new Error('held A response did not settle')), 10000);})]);
    clearTimeout(completionTimer);
    assert(network.cancelled || delivery.fulfilled && network.finished, 'no explicit A delivery or client cancellation proof');
    await p.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    assert.equal(await p.locator('.project-switch strong').innerText(), 'QA Beta');
    assert.doesNotMatch(await p.locator('main').innerText(), /QA Alpha/);
    assert(observed >= 1);
    await p.reload();
    await p.locator('.control-shell').waitFor();
    assert.equal(await p.locator('.project-switch strong').innerText(), 'QA Beta');
    pass('Chrome-actual-late-A-dashboard-response-cancelled-or-ignored-after-B-switch');
  } finally {release(); p.off('requestfinished', finished); p.off('requestfailed', failed); await p.unroute(match);}
}
module.exports = {run};
