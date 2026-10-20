'use strict';
// A fresh headed Chrome profile with CDP noDefaults proves actual visibility.
const h = require('./next-fixture.cjs');
const {chromium} = require(h.required('ACCORD_QA_PLAYWRIGHT'));
const crypto = require('node:crypto');
async function visibility(p, wanted) {
  for (let i = 0; i < 100; i++) {if (await p.evaluate(() => document.visibilityState) === wanted) return; await h.sleep(30);}
  throw new Error('actual headed document did not become ' + wanted);
}
module.exports = async (f, pass) => {
  const port = await h.port(); const profile = h.path.join(h.root, 'headed-' + crypto.randomUUID()); h.fs.mkdirSync(profile, {mode: 0o700});
  const child = h.cp.spawn(h.required('ACCORD_QA_CHROME'), ['--remote-debugging-address=127.0.0.1', '--remote-debugging-port=' + port, '--user-data-dir=' + profile, '--no-first-run', '--no-default-browser-check', 'about:blank'], {env: h.env, stdio: 'ignore'});
  let browser;
  try {
    let ready = false; for (let i = 0; i < 100; i++) {try {ready = (await fetch('http://127.0.0.1:' + port + '/json/version')).status === 200;} catch {} if (ready) break; await h.sleep(50);} h.assert(ready);
    browser = await chromium.connectOverCDP('http://127.0.0.1:' + port, {noDefaults: true});
    const user = await h.login({...f, browser}, f.human, {}, false, browser.contexts()[0]); const p = user.p; await p.bringToFront(); await visibility(p, 'visible'); await p.clock.install();
    let count = 0; const isPoll = r => new URL(r.url()).pathname === '/web/data/dashboard'; p.on('request', r => {if (isPoll(r)) count++;});
    await h.sleep(50); f.go.child.kill('SIGSTOP');
    try {
      const request = p.waitForRequest(isPoll); await p.getByRole('button', {name: 'Обновить состояние', exact: true}).click(); await request;
      const aborted = p.waitForEvent('requestfailed', isPoll); const background = await user.ctx.newPage(); await background.goto('about:blank'); await background.bringToFront(); await visibility(p, 'hidden'); await aborted;
      const hidden = count; await p.clock.fastForward(45000); h.assert.equal(count, hidden, 'poll continued while actually hidden');
      f.go.child.kill('SIGCONT'); const resumed = p.waitForResponse(isPoll); await background.close(); await p.bringToFront(); await visibility(p, 'visible'); h.assert.equal((await resumed).status(), 200);
      await p.screenshot({animations:'disabled',path: h.path.join(h.root, 'headed-visible-restored.png')});
      pass('actual-headed-hidden-inflight-abort-zero-polls-visible-resume', {hidden_requests: 0, cdp_no_defaults: true});
    } finally {f.go.child.kill('SIGCONT');}
  } finally {if (browser) await browser.close(); if (child.exitCode === null && !child.signalCode) await h.stop({child});}
};
