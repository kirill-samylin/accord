'use strict';
const h = require('./next-fixture.cjs');
module.exports = async (f, pass) => {
  const p = f.p; const measured = [];
  const routes = {overview: '/app', tasks: '/app/tasks', questions: '/app/questions', agents: '/app/agents', projects: '/app/projects', activity: '/app/activity', stats: '/app/stats', designs: '/app/designs', history: '/app/history', settings: '/app/settings', task: '/app/tasks/' + f.tasks[111].key, epic: '/app/tasks/' + f.epic, agent: '/app/agents/' + f.agentSession.session_id};
  async function shot(name, width, theme) {
    await p.evaluate(() => document.fonts.ready); h.assert(await p.evaluate(() => document.fonts.check('14px Inter')));
    const dimensions = await p.evaluate(() => ({document: document.documentElement.scrollWidth, viewport: innerWidth, scale: visualViewport.scale, sidebar: document.querySelector('.desktop-sidebar')?.getBoundingClientRect().width}));
    h.assert(dimensions.document <= width, name + ' page overflow'); h.assert.equal(dimensions.scale, 1);
    const file = `${name}-${width}-${theme}.png`; await p.screenshot({animations:'disabled',path: h.path.join(h.root, file), fullPage: true}); measured.push({name, width, theme, file, ...dimensions});
  }
  for (const theme of ['light', 'dark']) {
    await p.setViewportSize({width: 1440, height: 900}); await h.goto(f, '/app');
    if ((await p.locator('html').getAttribute('data-theme') || 'light') !== theme) await p.getByRole('button', {name: theme === 'dark' ? 'Тёмная тема' : 'Светлая тема', exact: true}).click();
    for (const width of [1440, 768, 390]) {
      await p.setViewportSize({width, height: width === 390 ? 844 : 900});
      for (const [name, route] of Object.entries(routes)) {
        await h.goto(f, route); if(width===768) {h.assert(await p.locator('.desktop-sidebar').isVisible()); h.assert(!(await p.locator('.mobile-nav').isVisible()));} else if(width<768) {h.assert(await p.locator('.mobile-nav').isVisible()); h.assert(await p.getByRole('button',{name:'Открыть навигацию',exact:true}).isVisible());} if(name==='overview') {await p.locator('.dashboard-bottom .skeleton-lines').waitFor({state:'detached'}); h.assert(await p.locator('.dashboard-bottom .task-row').count()>0,'recent task selection still missing');} if (name === 'task' && width < 768) {await p.locator('.mobile-properties summary').click(); await shot('task-properties',width,theme); await p.locator('.main-scroll').evaluate(el=>{el.scrollTop=0;});}
        await shot(name, width, theme);
        if (name === 'overview' && width === 1440) {
          const bounds = await p.evaluate(() => ({agent: document.querySelector('.agent-strip')?.getBoundingClientRect().top, metrics: document.querySelector('.metrics-grid')?.getBoundingClientRect().top, sidebar: document.querySelector('.sidebar')?.getBoundingClientRect().width, height:document.querySelector('.sidebar')?.getBoundingClientRect().height, viewport:innerHeight}));
          h.assert(bounds.agent < bounds.metrics, 'agent strip not before metrics'); h.assert.equal(bounds.sidebar, 240); if(bounds.height<bounds.viewport&&!f.findings.some(x=>x.id==='sidebar-height'))f.findings.push({id:'sidebar-height',result:'FAIL',actual:bounds.height,expected:bounds.viewport});
        }
      }
    }
  }
  await p.setViewportSize({width: 1440, height: 900}); await h.goto(f, '/app/tasks');
  const searchTrigger = p.getByRole('button', {name: /^Поиск/}).first(); await searchTrigger.focus(); await p.keyboard.press('Meta+k'); await p.getByRole('dialog').waitFor(); await p.getByPlaceholder('Найти задачу или перейти к разделу…').fill('ёжик'); await p.getByRole('option').filter({hasText: f.tasks[111].key}).waitFor();
  await shot('command-search', 1440, 'dark'); await p.keyboard.press('Escape'); await p.getByRole('dialog').waitFor({state:'detached'}); try{await p.waitForFunction(()=>[...document.querySelectorAll('.desktop-sidebar button.nav-row')].find(x=>x.textContent.includes('Поиск'))===document.activeElement,{}, {timeout:1500});}catch {f.findings.push({id:'command-focus-return',result:'FAIL',active:await p.evaluate(()=>document.activeElement?.tagName)});} await p.keyboard.press('Meta+k'); await p.getByPlaceholder('Найти задачу или перейти к разделу…').fill('ёжик'); await p.getByRole('option').filter({hasText:f.tasks[111].key}).waitFor(); await p.keyboard.press('ArrowDown'); await p.keyboard.press('Enter'); await p.waitForURL(f.origin+'/app/tasks/'+f.tasks[111].key); await h.goto(f,'/app/tasks');
  await p.getByRole('button', {name: /^Фильтры/}).click(); await shot('task-filter', 1440, 'dark'); await p.keyboard.press('Escape');
  await p.getByRole('button', {name: 'Быстрый просмотр ' + f.task, exact: true}).click(); await p.locator('.task-peek h1').waitFor(); await shot('task-peek', 1440, 'dark'); await p.keyboard.press('Escape');
  for (const width of [767, 390, 320]) {
    await p.setViewportSize({width, height: 844}); await h.goto(f, '/app'); await p.getByRole('button', {name: 'Открыть навигацию', exact: true}).click(); await p.getByRole('dialog').waitFor(); await shot('drawer', width, 'dark'); await p.keyboard.press('Escape'); await p.getByRole('dialog').waitFor({state:'detached'}); try{await p.waitForFunction(()=>document.querySelector('button[aria-label="Открыть навигацию"]')===document.activeElement,{}, {timeout:1500});}catch{f.findings.push({id:'drawer-focus-return',result:'FAIL',width});}
    await p.getByRole('button', {name: 'Поиск', exact: true}).click(); await p.getByRole('dialog').waitFor(); await shot('command-mobile', width, 'dark'); await p.keyboard.press('Escape');
  }
  await p.setViewportSize({width: 390, height: 844}); await h.goto(f, '/app/tasks?owner=synthetic-nobody'); await p.getByRole('heading', {name: 'Задачи не найдены', exact: true}).waitFor(); await shot('task-empty', 390, 'dark');
  await h.goto(f, '/app/tasks'); await p.getByRole('button', {name: 'Новая задача', exact: true}).first().click(); const dialog = p.getByRole('dialog'); await dialog.locator('[name=title]').fill('Synthetic form ' + 'длинный контекст '.repeat(10)); await dialog.locator('[name=description]').fill('Synthetic acceptance\n' + 'длинное описание '.repeat(30));
  for(let i=0;i<20;i++){await p.keyboard.press('Tab'); h.assert(await dialog.evaluate(el=>el.contains(document.activeElement)),'dialog focus escaped');} await shot('create-dialog', 390, 'dark'); h.assert.equal(await dialog.locator('[name=title]').evaluate(el => getComputedStyle(el).fontSize), '16px'); await p.keyboard.press('Escape');
  await p.emulateMedia({reducedMotion: 'reduce'}); await h.goto(f, '/app'); h.assert.equal(await p.locator('.icon-button').first().evaluate(el => getComputedStyle(el).transitionDuration), '0s');
  await p.setViewportSize({width: 1440, height: 900});
  h.fs.writeFileSync(h.path.join(h.root, 'next-visual-measurements.json'), JSON.stringify(measured, null, 2), {mode: 0o600});
  pass('visual-10screens-details-responsive-dark-light-font-dialog-captured', {screenshots: measured.length, known_findings:f.findings.map(x=>x.id), widths: [1440, 768, 767, 390, 320], note: 'Actual PNG inspection separately recorded; capture alone is not design APPROVE.'});
};
