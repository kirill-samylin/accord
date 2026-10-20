'use strict';
const h = require('./control-fixture.cjs');
const {assert, goto, call} = h;
const keys = p => p.locator('.task-list .task-card .key').allTextContents().then(rows => rows.map(s => s.trim()));
module.exports = async (f, pass) => {
  const p = f.p; await goto(f, '/web/');
  const trigger = p.locator('[data-command-open]'); await trigger.focus();
  await p.keyboard.press('Control+k'); const dialog = p.locator('#command-palette');
  assert(await dialog.isVisible()); assert(await p.locator('[data-command-query]').evaluate(el => el === document.activeElement));
  await p.locator('[data-command-query]').fill('агенты');
  assert.equal(await dialog.locator('[data-command-item]:visible').count(), 1);
  await p.keyboard.press('ArrowDown'); assert.match(await p.locator(':focus').getAttribute('href'), /\/web\/agents/);
  await p.keyboard.press('ArrowUp'); assert.match(await p.locator(':focus').getAttribute('href'), /\/web\/agents/);
  await p.keyboard.press('Escape'); assert(!(await dialog.isVisible())); assert(await trigger.evaluate(el => el === document.activeElement));
  await trigger.click(); await p.locator('[data-command-query]').fill('ёжик');
  assert(await dialog.locator('[data-command-empty]').isVisible());
  await dialog.getByRole('button', {name: 'Найти', exact: true}).click();
  await p.waitForURL(url => url.pathname === '/web/tasks' && url.searchParams.get('q') === 'ёжик');
  assert.deepEqual(await keys(p), [f.tasks[0].key]); assert.equal(await p.evaluate(() => globalThis.qaXss || 0), 0);
  await p.locator('[data-command-open]').click();
  await dialog.getByRole('link', {name: '+ Создать задачу', exact: true}).click();
  await p.waitForURL(/\/web\/tasks#create$/); assert(await p.locator('[data-create]').isVisible());
  assert.equal(await p.locator('#create-type').inputValue(), 'task');
  const before = await h.overview(f);
  for (const width of [320, 390]) {
    await p.setViewportSize({width, height: 844}); await goto(f, '/web/');
    assert(await p.locator('[data-command-open]').isVisible()); await p.locator('[data-command-open]').click();
    const box = await dialog.boundingBox(); assert(box.x >= 0 && box.x + box.width <= width, 'modal exceeds viewport');
    await p.mouse.click(2, 2); assert(!(await dialog.isVisible()));
    assert(await p.locator('[data-command-open]').evaluate(el => el === document.activeElement));
  }
  assert.equal((await h.overview(f)).tasks_total, before.tasks_total, 'navigation mutated tasks');
  const noJS = await h.login(f.browser, f.origin, f.human, {javaScriptEnabled: false, viewport: {width: 390, height: 844}});
  try {
    assert(await noJS.p.locator('.global-search').isVisible());
    await noJS.p.locator('#global-q').fill(f.tasks[0].key); await noJS.p.locator('.global-search button').click();
    await noJS.p.waitForURL(url => url.searchParams.get('q') === f.tasks[0].key);
    assert.deepEqual(await keys(noJS.p), [f.tasks[0].key]);
  } finally {await noJS.ctx.close();}
  pass('commands-keyboard-filter-search-focus-create-mobile320390-nojs', {viewports: [320, 390], no_mutation: true});
  await p.setViewportSize({width: 1440, height: 900});
  for (const [kind, expectedRole] of [['human', ['human']], ['agent', ['developer']], ['all', ['human', 'developer']]]) {
    await goto(f, '/web/agents?kind=' + kind);
    assert.equal(await p.locator('.filters [name=kind]').inputValue(), kind);
    const roles = await p.locator('.filters [name=role] option').evaluateAll(nodes => nodes.map(n => n.value));
    for (const role of expectedRole) assert(roles.includes(role));
    if (kind === 'human') assert(!roles.includes('developer'));
    if (kind === 'agent') assert(!roles.includes('human'));
    await p.locator('.filters [name=role]').selectOption(expectedRole[0]); await p.locator('.filters button').click();
    await p.waitForURL(url => url.searchParams.get('role') === expectedRole[0]);
    assert.equal(new URL(p.url()).searchParams.get('kind'), kind);
    assert((await p.locator('.agent-row').count()) > 0);
    const labels = await p.locator('.agent-row .entity-name').allTextContents();
    assert(labels.every(label => label.trim() === (expectedRole[0] === 'human' ? 'Человек' : 'Разработчик')));
  }
  await goto(f, '/web/'); await p.getByRole('link', {name: /Всего задач/}).click();
  await p.waitForURL(url => url.searchParams.get('include_closed') === 'true');
  const all = await keys(p); assert(all.includes(f.tasks[1].key)); assert(all.includes(f.tasks[2].key));
  pass('agent-kind-role-forms-and-total-includes-closed', {kinds: ['human', 'agent', 'all'], closed_keys: [f.tasks[1].key, f.tasks[2].key]});
  await goto(f, '/web/tasks/' + f.tasks[3].key);
  const cursors = await p.locator('.task-timeline .event-row > .micro').allTextContents().then(rows => rows.map(s => Number(s.trim().slice(1))));
  assert.equal(cursors.length, 10); assert(cursors.every((n, i) => i === 0 || n < cursors[i - 1]));
  const exact = await call(f.worker, 'events', {task_key: f.tasks[3].key, since_cursor: 0, limit: 200});
  assert.deepEqual(cursors, exact.events.map(e => e.cursor).sort((a, b) => b - a).slice(0, 10));
  await p.locator('.task-timeline').getByRole('link', {name: 'Вся история →'}).click();
  await p.waitForURL(url => url.pathname === '/web/activity' && url.searchParams.get('task_key') === f.tasks[3].key);
  assert.equal(await p.locator('.activity-list .event-row').count(), exact.events.length);
  pass('task-timeline-ten-latest-full-history-exact-filter', {latest: cursors, full_count: exact.events.length});
};
