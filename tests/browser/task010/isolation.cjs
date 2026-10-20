'use strict';
const h = require('../../process/task010/harness.cjs');
const {assert, origin} = h;
async function run(f, owner, pass) {
  const p = owner.page;
  const second = await owner.context.newPage(); second.setDefaultTimeout(15000);
  await second.goto(origin + '/app/p/' + f.b + '/profiles');
  const opening = second.getByRole('button', {name: 'Открыть проект', exact: true});
  await second.locator('.control-shell, .welcome .registry-form button[type=submit]').first().waitFor();
  if (await opening.isVisible().catch(() => false)) await opening.click();
  await second.locator('.profile-card').waitFor();
  assert.doesNotMatch(await second.locator('main').innerText(), /QA Chrome created/);
  assert(new URL(p.url()).pathname.includes(f.a));
  await p.goto(origin + '/app/p/' + f.a + '/profiles');
  await p.locator('.profile-card').first().waitFor();
  let captured, release, fail;
  const observed = new Promise(resolve => {captured = resolve;});
  const gate = new Promise(resolve => {release = resolve;});
  let request, done;
  const delivery = new Promise(resolve => {done = resolve;});
  const matcher = url => url.pathname === '/web/p/' + f.a + '/profiles';
  await p.route(matcher, async route => {
    if (request) return route.continue();
    request = route.request();
    try {
      const actual = await route.fetch(); assert.equal(actual.status(), 200); captured(); await gate;
      try {await route.fulfill({response: actual}); done('delivered');} catch {done('cancelled');}
    } catch (error) {fail = error; captured(); done('failed'); await route.abort().catch(() => {});}
  });
  try {
    await p.getByRole('button', {name: 'Обновить', exact: true}).click();
    let timer; await Promise.race([observed, new Promise((_, reject) => {timer = setTimeout(() => reject(new Error('actual profile A response not observed')), 12000);})]); clearTimeout(timer);
    if (fail) throw fail;
    await p.getByRole('button', {name: 'Переключить проект', exact: true}).click();
    await p.getByRole('menuitem', {name: /QA Beta/}).click(); await p.waitForURL(new RegExp('/app/p/' + f.b));
    await p.locator('.control-shell').waitFor(); release();
    let timeout; await Promise.race([delivery, new Promise((_, reject) => {timeout = setTimeout(() => reject(new Error('held profile read did not settle')), 10000);})]); clearTimeout(timeout);
    assert.equal(await p.locator('.project-switch strong').innerText(), 'QA Beta');
    await p.getByRole('link', {name: 'Агенты', exact: true}).first().click();
    await p.locator('.profile-card').waitFor(); assert.doesNotMatch(await p.locator('main').innerText(), /QA Chrome created/);
    await second.reload(); await second.locator('.profile-card').waitFor();
    assert.doesNotMatch(await second.locator('main').innerText(), /QA Chrome created/);
    pass('Chrome-two-tabs-profile-project-isolation-and-actual-late-A-read-after-switch');
  } finally {release(); await p.unroute(matcher); await second.close();}
}
module.exports = {run};
