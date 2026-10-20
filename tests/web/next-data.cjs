'use strict';
const h = require('./next-fixture.cjs');
module.exports = async (f, pass) => {
  f.tasks = [];
  f.otherEpic = await h.create(f,{type:'epic',title:'Synthetic other epic',description:'Second scoped grouping',owner:'qa-owner-other'});
  for (let i = 0; i < 112; i++) {
    await h.sleep(16);
    f.tasks.push(await h.call(f.worker, 'create', {type: 'task', epic_key: i === 110 ? f.otherEpic : f.epic, role: i === 109 ? 'analyst' : 'developer', title: i === 111 ? 'ЁЖиК <img src=x onerror="globalThis.qaXss=1"> ' + 'длинное-'.repeat(17) : 'Synthetic pagination ' + i, description: 'Synthetic criteria ' + 'проверяемый контекст '.repeat(25)}));
  }
  const done = await h.call(f.worker, 'next', {task_key: f.tasks[0].key}); await h.call(f.worker, 'transition', {task_key: done.task.key, generation: done.task.generation, state: 'done', evidence: {commit: '1'.repeat(40), run_summary: 'Synthetic verified fixture'}});
  await h.call(f.worker, 'transition', {task_key: f.tasks[1].key, state: 'cancelled', reason: 'Synthetic cancellation'});
  await h.call(f.worker, 'link', {from: f.tasks[2].key, to: f.tasks[3].key, type: 'blocks', condition: 'completed'});
  const page = await h.json(f, '/web/tasks?include_closed=true'); h.assert.equal(page.tasks.length, 100); h.assert(page.next_cursor); h.assert(page.epics.some(epic => epic.key === f.epic));
  const next = await h.json(f, '/web/tasks?include_closed=true&cursor=' + encodeURIComponent(page.next_cursor));
  const keys = [...page.tasks, ...next.tasks].map(task => task.key); h.assert.equal(new Set(keys).size, keys.length); h.assert(keys.includes(f.tasks[111].key));
  await h.goto(f, '/app/tasks?include_closed=true'); await f.p.getByRole('button', {name: 'Следующая страница'}).click(); await f.p.waitForURL(/cursor=/); await f.p.waitForFunction(key => [...document.querySelectorAll('.task-key')].some(el => el.textContent === key), f.tasks[111].key);
  await h.goto(f, '/app/tasks'); await f.p.getByRole('tab', {name: /^Эпики/}).click(); h.assert(await f.p.getByRole('link', {name: f.epic, exact: true}).isVisible(), 'open epic missing');
  await f.p.getByRole('button', {name: /^Фильтры/}).click(); await f.p.getByRole('combobox', {name:/^Роль/}).selectOption('developer'); await f.p.waitForURL(/role=developer/); h.assert.equal(await f.p.getByRole('tab', {name: /^Эпики/}).getAttribute('data-state'), 'active', 'filter reset local tab');
  await h.goto(f, '/app/tasks'); await f.p.getByRole('textbox', {name: 'Поиск задач', exact: true}).fill('ёжик'); await f.p.getByRole('textbox', {name: 'Поиск задач', exact: true}).press('Enter'); await f.p.waitForURL(/q=/); await f.p.getByRole('link', {name: f.tasks[111].key, exact: true}).waitFor(); h.assert.equal(await f.p.evaluate(() => globalThis.qaXss || 0), 0);
  await f.p.getByRole('button', {name: 'Доска', exact: true}).click(); await f.p.waitForURL(/layout=board/); await f.p.locator('.kanban-card').waitFor(); h.assert.match(await f.p.locator('.kanban').innerText(), /ЁЖиК/);
  const owner = await h.json(f, '/web/tasks?owner=qa-owner-other'); h.assert.deepEqual(owner.tasks.map(task => task.key), [f.tasks[110].key]);
  await h.goto(f, '/app/tasks'); await f.p.getByRole('button', {name: /^Фильтры/}).click(); await f.p.getByLabel('Исполнитель', {exact: true}).fill('qa-owner-other'); await f.p.getByRole('button', {name: 'Применить', exact: true}).click(); await f.p.waitForURL(/owner=/); await f.p.getByRole('link', {name: f.tasks[110].key, exact: true}).waitFor();
  pass('whole-project112-search-epics-tabs-owner-board-bound-paging', {seeded_tasks: 112, loaded_distinct: keys.length});
  await h.goto(f, '/app/agents'); h.assert.equal(await f.p.getByRole('tab', {name: 'Активные', exact: true}).getAttribute('data-state'), 'active');
  await f.p.getByRole('tab', {name: 'Все сессии', exact: true}).click(); await f.p.waitForURL(/include_closed=true/); h.assert.equal(await f.p.getByRole('tab', {name: 'Все сессии', exact: true}).getAttribute('data-state'), 'active');
  await h.goto(f, '/app'); const overview = await h.json(f, '/web/'); h.assert.equal(overview.overview.tasks_total, keys.length); h.assert.equal(await f.p.locator('.metric-value').first().innerText(), String(keys.length));
  // Dashboard deliberately labels an ascending bounded page as a selection, never project-wide newest.
  h.assert.match(await f.p.locator('.dashboard-bottom').innerText(), /Задачи проекта/i);
  await h.goto(f, '/app/activity'); const events = await h.json(f, '/web/activity'); h.assert(events.next_cursor); const older = await h.json(f, '/web/activity?cursor=' + events.next_cursor); h.assert(older.activity.every(event => event.cursor < Math.min(...events.activity.map(event => event.cursor))));
  await h.goto(f, '/app/stats'); h.assert.match(await f.p.locator('main').innerText(), /токен/i); h.assert(!/\$0\.00/.test(await f.p.locator('main').innerText()));
  const ssr = await f.ctx.request.get(f.origin + '/web/tasks/' + f.task); h.assert.equal(ssr.status(), 200); h.assert.match(ssr.headers()['content-type'], /text\/html/); h.assert.match(await ssr.text(), /Synthetic edited task/);
  const json = await h.json(f, '/web/tasks/' + f.task); h.assert.equal(json.task.title, (await h.get(f, f.task)).title);
  pass('agents-filter-truth-overview-native-activity-stats-json-ssr-agreement');
};
