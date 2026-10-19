'use strict';
const h = require('./control-fixture.cjs');
const {assert, call, goto, login, binary, path, port, server, stop, token, fs, crypto, sleep} = h;
const pollURL = '/web/data/overview';
const isPoll = r => r.url().endsWith(pollURL);
async function status(p, pattern) {await p.locator('[data-overview-status]').filter({hasText: pattern}).waitFor();}
async function tick(p, ms = 15000) {await p.clock.fastForward(ms);}
async function until(check, message) {
  for (let i = 0; i < 100; i++) {if (await check()) return; await sleep(25);} throw new Error(message);
}
module.exports = async (f, pass) => {
  const question = await call(f.worker, 'create', {type: 'task', role: 'human', epic_key: f.epic,
    title: 'Synthetic retained draft', description: 'Draft must survive background overview reads'});
  const draft = await f.ctx.newPage(); await draft.goto(f.origin + '/web/tasks/' + question.key);
  const claimed = draft.waitForResponse(r => r.request().method() === 'POST');
  await draft.getByRole('button', {name: 'Взять и ответить'}).click(); assert.equal((await claimed).status(), 303);
  await draft.locator('#answer').fill('Synthetic draft <do not replace>');
  const before = await call(f.worker, 'get', {task_key: question.key});
  await h.pausedClock(f.p); await goto(f, '/web/');
  let requests = 0, inflight = 0, maximum = 0;
  const started = r => {if (isPoll(r)) {requests++; inflight++; maximum = Math.max(maximum, inflight);}};
  const ended = r => {if (isPoll(r)) inflight--;};
  f.p.on('request', started); f.p.on('requestfinished', ended); f.p.on('requestfailed', ended);
  const first = f.p.waitForResponse(isPoll); await tick(f.p); assert.equal((await first).status(), 200);
  await status(f.p, 'Счётчики обновлены');
  const counters = await f.p.locator('[data-overview-value]').allTextContents();
  const observed = await f.p.locator('[data-overview-time]').getAttribute('datetime');
  await f.ctx.setOffline(true); await status(f.p, 'Нет сети'); const offlineRequests = requests;
  await tick(f.p, 45000); assert.equal(requests, offlineRequests);
  await f.ctx.setOffline(false); const online = f.p.waitForResponse(isPoll); await tick(f.p, 1); assert.equal((await online).status(), 200);
  await status(f.p, 'Счётчики обновлены');

  // SIGSTOP pauses only this synthetic server: the request traverses Chrome/HTTP
  // and waits at a real Go process boundary; no response interception is used.
  f.srv.child.kill('SIGSTOP');
  try {
    await until(() => inflight === 0, 'prior poll not settled');
    const pending = requests; await tick(f.p); await until(() => requests === pending + 1, 'poll not started');
    await f.p.evaluate(() => {window.dispatchEvent(new Event('online')); window.dispatchEvent(new Event('online'));});
    await tick(f.p, 1000); assert.equal(requests, pending + 1); assert.equal(maximum, 1);
    await tick(f.p, 8000); await status(f.p, 'Последние данные сохранены');
    assert.deepEqual(await f.p.locator('[data-overview-value]').allTextContents(), counters);
    // Timestamp may change after the successful reconnect, but failure must retain it.
    const failureTime = await f.p.locator('[data-overview-time]').getAttribute('datetime');
    await tick(f.p, 1000); assert.equal(await f.p.locator('[data-overview-time]').getAttribute('datetime'), failureTime);
  } finally {f.srv.child.kill('SIGCONT');}
  const recovered = f.p.waitForResponse(isPoll); await tick(f.p, 60000); assert.equal((await recovered).status(), 200);
  await status(f.p, 'Счётчики обновлены');
  assert.equal(await draft.locator('#answer').inputValue(), 'Synthetic draft <do not replace>');
  const after = await call(f.worker, 'get', {task_key: question.key});
  assert.equal(after.lease.expires_at, before.lease.expires_at); assert.equal(after.lease.generation, before.lease.generation);
  pass('poll-visible-offline-one-inflight-timeout-retains-recovery-no-renew-no-draft-rewrite',
    {requests, max_inflight: maximum, real_server_sigstop: true, clock_control: 'Playwright timers advanced; Go wall clock unchanged', lease_unchanged: true, initial_observed: observed});

  // Headless Chrome supports window lifecycle differently across releases. This
  // is a measured browser capability probe, never a fake hidden-state PASS.
  const cdp = await f.ctx.newCDPSession(f.p);
  let hidden = false;
  try {
    const {windowId} = await cdp.send('Browser.getWindowForTarget');
    await cdp.send('Browser.setWindowBounds', {windowId, bounds: {windowState: 'minimized'}});
    hidden = await f.p.evaluate(() => document.visibilityState === 'hidden');
    if (hidden) {
      const count = requests; await tick(f.p, 30000); assert.equal(requests, count);
      await cdp.send('Browser.setWindowBounds', {windowId, bounds: {windowState: 'normal'}});
      assert.equal(await f.p.evaluate(() => document.visibilityState), 'visible');
      const resumed = f.p.waitForResponse(isPoll); await tick(f.p, 1); assert.equal((await resumed).status(), 200);
    } else await cdp.send('Browser.setWindowBounds', {windowId, bounds: {windowState: 'normal'}});
  } finally {await cdp.detach();}
  if (hidden) pass('poll-real-hidden-pauses-visible-resumes', {visibility: 'Chrome window minimized/normal'});
  else pass('poll-real-hidden-pauses-visible-resumes', await require('./control-visibility.cjs')(f));

  const revokedToken = token(f.state, 'poll-revoked-' + crypto.randomUUID());
  const revokedUser = await login(f.browser, f.origin, revokedToken);
  await revokedUser.p.close(); revokedUser.p = await revokedUser.ctx.newPage();
  await h.pausedClock(revokedUser.p); await goto({...f, p: revokedUser.p}, '/web/');
  h.revoke(f.state, revokedToken); let authPolls = 0; revokedUser.p.on('request', r => {if(isPoll(r)) authPolls++;});
  const unauthorized = revokedUser.p.waitForResponse(isPoll); await tick(revokedUser.p); assert.equal((await unauthorized).status(), 401);
  await status(revokedUser.p, 'Вход больше не действует'); await tick(revokedUser.p, 120000); assert.equal(authPolls, 1);
  await revokedUser.ctx.close();
  pass('poll-real401-terminal-stop', {status: 401, requests_after_terminal: 0});

  // Separate real Go session TTL; fixed client Date deliberately prevents the
  // earlier browser-side expiry guard so the real server409 branch is reached.
  const expiryState = path.join(h.root, 'expiry-' + crypto.randomUUID()); fs.mkdirSync(expiryState, {mode: 0o700});
  const expiryToken = token(expiryState, 'expiry-' + crypto.randomUUID());
  const expiryServer = await server(binary, expiryState, await port(), true, {session: '2s'});
  let expiryUser;
  try {
    expiryUser = await login(f.browser, expiryServer.origin, expiryToken);
    await expiryUser.p.close(); expiryUser.p = await expiryUser.ctx.newPage();
    await h.pausedClock(expiryUser.p, Date.now() - 60000);
    await expiryUser.p.goto(expiryServer.origin + '/web/');
    await sleep(2200);
    let expiryPolls = 0; expiryUser.p.on('request', r => {if(isPoll(r)) expiryPolls++;});
    const lost = expiryUser.p.waitForResponse(isPoll); await tick(expiryUser.p); const response = await lost;
    assert.equal(response.status(), 409); assert.equal((await response.json()).code, 'LEASE_LOST');
    await status(expiryUser.p, 'Срок рабочей сессии истёк'); await tick(expiryUser.p, 120000); assert.equal(expiryPolls, 1);
    await expiryUser.p.reload(); await expiryUser.p.getByRole('button', {name: 'Продолжить', exact: true}).waitFor();
    const continued = expiryUser.p.waitForResponse(r => r.request().method() === 'POST');
    await expiryUser.p.getByRole('button', {name: 'Продолжить', exact: true}).click(); assert.equal((await continued).status(), 303);
    await expiryUser.p.locator('[data-overview]').waitFor();
    pass('poll-real409-terminal-no-get-renew-explicit-continue', {status: 409, code: 'LEASE_LOST', requests_after_terminal: 0,
      clock_control: 'client Date set60s behind; actual Go session TTL2s expired in wall time'});
  } finally {if(expiryUser) await expiryUser.ctx.close(); await stop(expiryServer);}
  f.p.off('request', started); f.p.off('requestfinished', ended); f.p.off('requestfailed', ended);
};
