'use strict';
// Playwright normally forces focus/visibility in every launched page. A separate
// synthetic Chrome profile + documented CDP noDefaults avoids that override.
const h = require('./control-fixture.cjs');
const {chromium} = require(h.required('ACCORD_QA_PLAYWRIGHT'));
async function visibility(p, wanted) {
  let actual;
  for (let i = 0; i < 60; i++) {actual = await p.evaluate(() => document.visibilityState); if(actual === wanted) return; await h.sleep(50);}
  throw new Error('actual headed visibility remained ' + actual + ', wanted ' + wanted);
}
module.exports = async f => {
  const n = await h.port(), profile = h.path.join(h.root, 'chrome-profile-' + h.crypto.randomUUID());
  h.fs.mkdirSync(profile, {mode: 0o700});
  const child = h.cp.spawn(h.required('ACCORD_QA_CHROME'), ['--remote-debugging-address=127.0.0.1', '--remote-debugging-port=' + n,
    '--user-data-dir=' + profile, '--no-first-run', '--no-default-browser-check', 'about:blank'], {stdio: 'ignore', env: h.env});
  let browser;
  try {
    let ready = false;
    for (let i = 0; i < 100; i++) {
      h.assert.equal(child.exitCode, null, 'synthetic Chrome exited');
      try {ready = (await fetch('http://127.0.0.1:' + n + '/json/version')).status === 200;} catch {}
      if(ready) break; await h.sleep(50);
    }
    h.assert(ready, 'private Chrome CDP not ready');
    browser = await chromium.connectOverCDP('http://127.0.0.1:' + n, {noDefaults: true});
    const {ctx, p: logged} = await h.login(browser, f.origin, f.human, {}, browser.contexts()[0]);
    await logged.close(); const p = await ctx.newPage();
    await h.pausedClock(p); await p.goto(f.origin + '/web/');
    let requests = 0; p.on('request', r => {if(r.url().endsWith('/web/data/overview')) requests++;});
    const background = await ctx.newPage(); await background.goto('about:blank'); await background.bringToFront();
    await visibility(p, 'hidden'); await p.clock.fastForward(45000);
    h.assert.equal(requests, 0, 'poll occurred in actual hidden document');
    await background.close(); await p.bringToFront(); await visibility(p, 'visible');
    const response = p.waitForResponse(r => r.url().endsWith('/web/data/overview'));
    await p.clock.fastForward(1); h.assert.equal((await response).status(), 200); h.assert.equal(requests, 1);
    await p.screenshot({path: h.path.join(h.root, 'headed-visible-restored.png')});
    return {visibility: 'Chrome actual background/foreground tabs; documented CDP noDefaults=true', hidden_requests: 0, resumed_requests: requests};
  } finally {
    try {if(browser) await browser.close();} finally {await h.stop({child});}
  }
};
