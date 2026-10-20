'use strict';
const h = require('../../process/task011/harness.cjs');
const {fs, path, assert, root, origin, startNext, stop} = h;
const {chromium} = require(process.env.ACCORD_TASK011_PLAYWRIGHT || '/Users/shagi/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
async function login(browser, token, setup = () => {}) {
  const context = await browser.newContext({viewport: {width: 1440, height: 1000}}); setup(context);
  const page = await context.newPage(); page.setDefaultTimeout(15000); page.setDefaultNavigationTimeout(20000);
  await page.goto(origin + '/app/login'); await page.getByRole('button', {name: 'Файл токена', exact: true}).click();
  await page.locator('input[name=token_file]').setInputFiles(token.file);
  await page.getByRole('button', {name: 'Войти в Accord', exact: true}).click();
  await page.waitForURL(/\/app\/p\/p_[a-f0-9]{32}/); await page.locator('.control-shell').waitFor();
  return {context, page};
}
async function open(page, project, suffix = '/profiles') {
  await page.goto(origin + '/app/p/' + project + suffix);
  const button = page.getByRole('button', {name: 'Открыть проект', exact: true});
  await page.locator('.control-shell, .welcome .registry-form button[type=submit]').first().waitFor();
  if (await button.isVisible().catch(() => false)) await button.click();
  await page.locator('.control-shell').waitFor();
}
async function shot(page, name) {
  await page.evaluate(async () => {document.querySelector('main').scrollTop = 0; scrollTo(0, 0); await document.fonts.ready; await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));});
  assert.equal(await page.locator('.breadcrumb-bar').count(), 1, 'duplicate shell header');
  const size = await page.evaluate(() => ({width: innerWidth, scroll: document.documentElement.scrollWidth}));
  assert(size.scroll <= size.width + 1, 'horizontal overflow: ' + name);
  await page.screenshot({path: path.join(root, name + '.png'), fullPage: true, animations: 'disabled'});
}
async function run(f, pass) {
  let next, browser;
  const errors = [], responses = [], prefetch = [], assetFailures = [], readFailures = [];
  try {
    next = await startNext();
    browser = await chromium.launch({headless: true, executablePath: process.env.ACCORD_TASK011_CHROME || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
    f.browserVersion = browser.version();
    const setup = context => context.on('page', page => {
      page.on('pageerror', error => errors.push(String(error.message).slice(0, 160)));
      page.on('response', response => {const url = new URL(response.url()); if (url.origin === origin) {responses.push(response.status()); if(response.status()>=400)readFailures.push({path:url.pathname,status:response.status()}); if (/^\/_next\/static\//.test(url.pathname) && response.status() >= 400) assetFailures.push(url.pathname);}});
      page.on('requestfailed', request => {if (/^\/_next\/static\//.test(new URL(request.url()).pathname)) assetFailures.push(new URL(request.url()).pathname);});
      page.on('request', request => {if (request.headers()['next-router-prefetch']) prefetch.push(new URL(request.url()).pathname);});
    });
    const owner = await login(browser, f.ownerToken, setup);
    await open(owner.page, f.a, '/tasks/' + f.uiTask.key);
    await require('./keyboard.cjs').run(f, owner, pass);
    await require('./forms.cjs').run(f, owner, pass);
    await require('./workflow.cjs').run(f, owner, pass);
    await require('./isolation.cjs').run(f, owner, pass);
    const viewer = await login(browser, f.viewer, setup); await open(viewer.page, f.a, '/tasks/' + f.uiTask.key);
    await viewer.page.getByRole('tab', {name: 'Координация', exact: true}).click();
    assert.equal(await viewer.page.getByRole('button', {name: 'Изменить назначение', exact: true}).count(), 0);
    assert.equal(await viewer.page.getByRole('button', {name: 'Управлять связью', exact: true}).count(), 0);
    assert.equal(await viewer.page.getByRole('button', {name: 'Изменить состояние', exact: true}).count(), 0);
    await viewer.context.close(); pass('Chrome-viewer-coordination-read-without-mutation-controls');
    const p = owner.page;
    for (const [screen, key] of [['assignment', f.uiTask.key], ['blocker', f.uiBlocked.key], ['checks', f.uiDone.key], ['epic', f.uiWorkflowEpic.key]]) {
      await open(p, f.a, '/tasks/' + key); await p.getByRole('tab', {name: 'Координация', exact: true}).click();
      for (const theme of ['light', 'dark']) {
        await p.setViewportSize({width: 1440, height: 1000});
        if ((await p.locator('html').getAttribute('data-theme') || 'light') !== theme)
          await p.getByRole('button', {name: theme === 'dark' ? 'Тёмная тема' : 'Светлая тема', exact: true}).click();
        assert.equal(await p.locator('html').getAttribute('data-theme'), theme);
        for (const width of [1440, 390]) {
          await p.setViewportSize({width, height: width === 390 ? 844 : 1000});
          await shot(p, screen + '-' + width + '-' + theme);
        }
      }
    }
    await p.setViewportSize({width: 1440, height: 1000});
    const logout = p.getByRole('button', {name: 'Выйти', exact: true});
    await p.getByRole('button', {name: 'Справка и сессия', exact: true}).click(); await logout.click(); await p.waitForURL(/\/app\/login$/); await p.goto(origin + '/app/p/' + f.a + '/tasks/' + f.uiTask.key); await p.waitForURL(/\/app\/login/);
    pass('Chrome-own-logout-denies-private-coordination-after-navigation');
    assert.equal(assetFailures.length, 0, 'Chrome asset failures'); assert.equal(errors.length, 0, 'Chrome page errors'); assert.equal(prefetch.length, 0, 'private automatic prefetch');
    assert(!responses.some(status => status >= 500), 'unexpected server failure');
    pass('Chrome-coordination-16screens-desktop-mobile-themes-nooverflow-pageerrors', {screenshots: 16, responses: responses.length, chrome: f.browserVersion});
  } catch (error) {
    h.save(path.join(root, 'browser-diagnostics.json'),{errors,readFailures,prefetch,assetFailures});
    if (browser) for (const context of browser.contexts()) for (const page of context.pages()) await page.screenshot({path: path.join(root, 'browser-failure-' + Date.now() + '.png'), fullPage: true}).catch(() => {});
    throw error;
  } finally {try {await browser?.close();} finally {await stop(next);}}
}
module.exports = {run, login, open, shot};
