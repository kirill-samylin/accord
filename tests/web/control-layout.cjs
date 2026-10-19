'use strict';
const {assert, fs, path, root, goto} = require('./control-fixture.cjs');
module.exports = async (f, pass) => {
  const routes = ['/web/', '/web/questions', '/web/tasks', '/web/agents', '/web/agents/' + f.firstSession.session_id,
    '/web/projects', '/web/activity', '/web/stats', '/web/settings', '/web/designs', '/web/history', '/web/tasks/' + f.tasks.at(-1).key];
  const measured = [];
  for (const width of [1440, 390]) {
    await f.p.setViewportSize({width, height: 900});
    for (const route of routes) {
      await goto(f, route); assert(await f.p.locator('h1').isVisible());
      assert.equal(await f.p.locator('body').evaluate(el => getComputedStyle(el).fontFamily.includes('InterVariable')), true, 'self-hosted stylesheet was not applied');
      const dimensions = await f.p.evaluate(() => ({document: document.documentElement.scrollWidth, viewport: innerWidth}));
      assert(dimensions.document <= width, 'document overflow ' + route + ' @' + width);
      assert.equal(await f.p.locator('nav[aria-label="Основные разделы"] a').count(), 10);
      measured.push({route, width, ...dimensions});
    }
    await goto(f, '/web/');
    await f.p.screenshot({path: path.join(root, 'overview-' + width + '-light.png'), fullPage: true});
  }
  const opener = f.p.locator('button[data-drawer-open]'); await opener.focus(); await f.p.keyboard.press('Enter');
  assert.equal(await opener.getAttribute('aria-expanded'), 'true');
  assert.equal(await f.p.locator('[data-drawer-close]').evaluate(el => document.activeElement === el), true);
  assert.equal(await f.p.locator('.workspace').evaluate(el => el.inert), true);
  const last = f.p.locator('.sidebar button[type=submit]'); await last.focus(); await f.p.keyboard.press('Tab');
  assert.equal(await f.p.locator('.sidebar .brand').evaluate(el => document.activeElement === el), true);
  await f.p.keyboard.press('Shift+Tab'); assert.equal(await last.evaluate(el => document.activeElement === el), true);
  await f.p.keyboard.press('Escape'); assert.equal(await opener.getAttribute('aria-expanded'), 'false');
  assert.equal(await opener.evaluate(el => document.activeElement === el), true);
  assert.equal(await f.p.locator('.workspace').evaluate(el => el.inert), false);
  const oldTheme = await f.p.locator('html').getAttribute('data-theme'); await f.p.locator('[data-theme-toggle]').click();
  const theme = await f.p.locator('html').getAttribute('data-theme'); assert.notEqual(theme, oldTheme);
  await f.p.reload(); assert.equal(await f.p.locator('html').getAttribute('data-theme'), theme);
  await f.p.screenshot({path: path.join(root, 'overview-390-' + theme + '.png'), fullPage: true});
  await f.p.evaluate(() => document.fonts.ready);
  assert(await f.p.evaluate(() => document.fonts.check('14px InterVariable')));
  const font = await f.ctx.request.get(f.origin + '/web/static/InterVariable.woff2'); assert.equal(font.status(), 200);
  assert.match(font.headers()['content-type'], /^font\/woff2/);
  fs.writeFileSync(path.join(root, 'layout-measurements.json'), JSON.stringify(measured, null, 2), {mode: 0o600});
  pass('desktop-mobile-routes-keyboard-drawer-theme-selfhosted-font', {routes: routes.length, viewports: [1440, 390], no_document_overflow: true, drawer_focus_trap: true, theme_persisted: theme});
};
