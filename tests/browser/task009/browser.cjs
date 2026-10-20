'use strict';
// Real Chrome against the production Next build and synthetic Go. No HTTP mocks.
const h = require('../../process/task009/harness.cjs');
const {fs, path, assert, root, origin, startNext, stop} = h;
const {chromium} = require(process.env.ACCORD_TASK009_PLAYWRIGHT || '/Users/shagi/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
async function login(browser, token, setup = () => {}) {
  const context = await browser.newContext({viewport: {width: 1440, height: 1000}});
  setup(context);
  const page = await context.newPage(); page.setDefaultTimeout(15000); page.setDefaultNavigationTimeout(20000);
  await page.goto(origin + '/app/login');
  await page.getByRole('button', {name: 'Файл токена', exact: true}).click();
  await page.locator('input[name=token_file]').setInputFiles(token.file);
  await page.getByRole('button', {name: 'Войти в Accord', exact: true}).click();
  await page.waitForURL(/\/app\/p\/p_[a-f0-9]{32}/);
  await page.locator('.control-shell').waitFor();
  return {context, page};
}
async function open(page, project, suffix = '') {
  await page.goto(origin + '/app/p/' + project + suffix);
  const button = page.getByRole('button', {name: 'Открыть проект', exact: true});
  await page.locator('.control-shell, .welcome .registry-form button[type=submit]').first().waitFor();
  if (await button.isVisible().catch(() => false)) await button.click();
  await page.locator('.control-shell').waitFor();
}
async function shot(page, name) {
  const overflow = await page.evaluate(() => ({width: innerWidth, scroll: document.documentElement.scrollWidth}));
  assert(overflow.scroll <= overflow.width + 1, 'horizontal overflow: ' + name);
  await page.screenshot({path: path.join(root, name + '.png'), fullPage: true, animations: 'disabled'});
}
async function run(f, pass) {
  let next, browser;
  const errors = [], statuses = [], prefetch = [];
  try {
    next = await startNext();
    browser = await chromium.launch({headless: true, executablePath: process.env.ACCORD_TASK009_CHROME || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
    f.browserVersion = browser.version();
    const setup = context => context.on('page', page => {
      page.on('pageerror', error => errors.push(String(error.message).slice(0, 160)));
      page.on('request', request => {if (request.headers()['next-router-prefetch']) prefetch.push(new URL(request.url()).pathname);});
      page.on('response', response => {if (new URL(response.url()).origin === origin) statuses.push(response.status());});
    });
    const owner = await login(browser, f.ownerToken, setup);
    const p = owner.page;
    await p.goto(origin + '/app/p/' + f.a + '/tasks');
    await p.getByRole('button', {name: 'Открыть проект', exact: true}).waitFor();
    await p.getByRole('button', {name: 'Открыть проект', exact: true}).click();
    await p.locator('.control-shell').waitFor();
    assert(new URL(p.url()).pathname.startsWith('/app/p/' + f.a));
    assert.match(await p.locator('main').innerText(), /Alpha/);
    const second = await owner.context.newPage(); second.setDefaultTimeout(15000);
    await open(second, f.b, '/tasks'); assert.match(await second.locator('main').innerText(), /Beta/);
    assert.match(await p.locator('main').innerText(), /Alpha/);
    await second.reload(); assert.match(await second.locator('main').innerText(), /Beta/);
    await p.reload(); assert.match(await p.locator('main').innerText(), /Alpha/);
    pass('Chrome-login-deep-explicit-open-two-tabs-reload-project-isolation');
    await p.getByRole('button', {name: 'Переключить проект', exact: true}).click();
    await p.getByRole('menuitem', {name: /QA Beta/}).click();
    await p.waitForURL(new RegExp('/app/p/' + f.b));
    await p.locator('.control-shell').waitFor();
    assert.equal(await p.locator('.project-switch strong').innerText(), 'QA Beta');
    await p.getByRole('link', {name: 'Задачи', exact: true}).first().click();
    assert(new URL(p.url()).pathname.startsWith('/app/p/' + f.b));
    assert.match(await p.locator('main').innerText(), /Beta/);
    assert.doesNotMatch(await p.locator('main').innerText(), /Alpha/);
    await p.goBack(); await p.locator('.control-shell').waitFor();
    pass('Chrome-switcher-qualified-navigation-back-cache-boundaries');
    await require('./create.cjs').run(f, owner, pass);
    await require('./settings.cjs').run(f, owner, pass);
    await require('./tabs.cjs').run(f, owner, pass);
    await require('./upload.cjs').run(f, owner, pass);
    const viewer = await login(browser, f.viewer, setup);
    await open(viewer.page, f.b, '/tasks');
    assert(await viewer.page.getByRole('button', {name: 'Новая задача', exact: true}).first().isDisabled());
    await viewer.page.goto(origin + '/app/projects/' + f.b);
    await viewer.page.getByRole('heading', {name: 'Настройки проекта', exact: true}).waitFor();
    assert.equal(await viewer.page.getByRole('button', {name: 'Сохранить название'}).count(), 0);
    assert.equal(await viewer.page.getByRole('heading', {name: 'Участники и доступ'}).count(), 0);
    await viewer.context.close();
    pass('Chrome-viewer-readonly-UI-owner-settings-hidden');
    await p.goto(origin + '/app/projects'); await p.getByRole('heading', {name: 'Проекты', exact: true}).waitFor();
    await shot(p, 'catalog-1440-light');
    await p.getByRole('button', {name: 'Тёмная тема'}).click(); await shot(p, 'catalog-1440-dark');
    await p.setViewportSize({width: 390, height: 844}); await shot(p, 'catalog-390-dark');
    await p.getByRole('button', {name: 'Светлая тема'}).click(); await shot(p, 'catalog-390-light');
    await p.setViewportSize({width: 1440, height: 1000});
    await open(p, f.a, '/designs'); await shot(p, 'designs-1440-light');
    await p.setViewportSize({width: 390, height: 844}); await shot(p, 'designs-390-light');
    assert.equal(errors.length, 0, 'Chrome runtime page errors');
    assert.equal(prefetch.length, 0, 'automatic private Next prefetch returned');
    assert(!statuses.some(status => status >= 500), 'unexpected browser server failure');
    pass('Chrome-desktop-mobile-light-dark-no-overflow-pageerrors-prefetch', {screenshots: 6, observed_responses: statuses.length});
  } catch (error) {
    if (browser) for (const context of browser.contexts()) for (const page of context.pages()) await page.screenshot({path: path.join(root, 'browser-failure-' + Date.now() + '.png'), fullPage: true}).catch(() => {});
    throw error;
  } finally {
    try {await browser?.close();} finally {await stop(next);}
  }
}
module.exports = {run, login, open, shot};
