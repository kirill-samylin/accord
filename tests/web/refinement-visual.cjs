'use strict';
const {assert, fs, path, root, goto} = require('./control-fixture.cjs');
module.exports = async (f, pass) => {
  const measured = [];
  const pages = {overview: '/web/', agents: '/web/agents', agent: '/web/agents/' + f.firstSession.session_id, task: '/web/tasks/' + f.tasks[0].key, form: '/web/tasks#create'};
  for (const width of [1440, 768, 390]) {
    await f.p.setViewportSize({width, height: 900});
    for (const theme of ['light', 'dark']) {
      for (const [name, route] of Object.entries(pages)) {
        await goto(f, route);
        if ((await f.p.locator('html').getAttribute('data-theme') || 'light') !== theme) await f.p.locator('[data-theme-toggle]').click();
        await f.p.evaluate(() => document.fonts.ready);
        assert(await f.p.evaluate(() => document.fonts.check('14px InterVariable')));
        const dimensions = await f.p.evaluate(() => ({document: document.documentElement.scrollWidth, viewport: innerWidth, scale: visualViewport.scale}));
        assert(dimensions.document <= width, name + ' overflow ' + width);
        if (name === 'overview') {
          const region = f.p.getByRole('region', {name: 'Последние события проекта'});
          assert.equal(await region.locator('.event-row').count(), 20);
          const bounds = await region.evaluate(el => ({height: el.clientHeight, full: el.scrollHeight}));
          assert(bounds.height <= (width < 768 ? Math.min(440, 900 * 0.65) : 520)); assert(bounds.full > bounds.height);
          if (width === 1440 && theme === 'light') {
            await region.focus(); await f.p.keyboard.press('PageDown');
            await f.p.waitForFunction(() => document.querySelector('[data-dashboard-activity]').scrollTop > 0);
            await f.p.getByRole('link', {name: 'Вся активность →'}).click();
            await f.p.waitForURL(f.origin + '/web/activity');
            assert((await f.p.locator('.activity-list .event-row').count()) >= 20);
            await goto(f, '/web/');
          }
        }
        if (name === 'agents') assert(await f.p.locator('.assigned-task a').count() > 0);
        if (name === 'form') {await f.p.locator('#create-title').fill('Синтетический длинный заголовок ' + 'контекст '.repeat(12)); await f.p.locator('#create-description').fill('Описание и критерии приёмки\n' + 'Длинный текст '.repeat(25));}
        const file = `${name}-${width}-${theme}.png`;
        await f.p.screenshot({path: path.join(root, file), fullPage: true});
        if (name === 'form') await f.p.screenshot({path: path.join(root, `form-viewport-${width}-${theme}.png`)});
        measured.push({name, width, theme, file, ...dimensions});
      }
    }
  }
  for (const width of [320, 390]) {
    await f.p.setViewportSize({width, height: 844}); await goto(f, '/web/');
    await f.p.locator('[data-command-open]').click();
    await f.p.screenshot({path: path.join(root, `commands-${width}.png`)});
    await f.p.keyboard.press('Escape');
  }
  await goto(f, '/web/agents?owner=synthetic-nobody');
  assert(await f.p.getByRole('heading', {name: 'Сессий по этим условиям нет'}).isVisible());
  await f.p.screenshot({path: path.join(root, 'agents-empty-390.png'), fullPage: true});
  fs.writeFileSync(path.join(root, 'visual-measurements.json'), JSON.stringify(measured, null, 2), {mode: 0o600});
  pass('visual-matrix-captured-font-loaded-no-overflow', {screenshots: measured.length + 9, viewports: [1440, 768, 390, 320], themes: ['light', 'dark'], long_claimed_title: f.tasks[0].key});
};
