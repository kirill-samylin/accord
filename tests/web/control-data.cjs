'use strict';
const h = require('./control-fixture.cjs');
const {assert, call, goto, overview} = h;
const taskKeys = p => p.locator('.task-list .task-card h3 a, .kanban .task-card h3 a').evaluateAll(nodes => nodes.map(n => n.getAttribute('href').split('/').pop()));
const eventCursors = p => p.locator('.activity-list .event-row > .micro').allTextContents().then(rows => rows.map(s => Number(s.trim().slice(1))));
module.exports = async (f, pass) => {
  const tasks = await h.nativeTasks(f);
  const expected = {tasks_total: tasks.length};
  for (const state of ['created', 'ready', 'in_progress', 'done', 'cancelled']) expected['tasks_' + state] = tasks.filter(t => t.state === state).length;
  expected.tasks_blocked = tasks.filter(t => t.blocked && !['done', 'cancelled'].includes(t.state)).length;
  const first = await overview(f);
  assert(first.tasks_total > 100); for (const [key, n] of Object.entries(expected)) assert.equal(first[key], n, key);
  assert.equal(first.open_epics, 2); assert.equal(first.active_agent_sessions, 2); assert.equal(first.active_human_sessions, 1);
  await goto(f, '/web/');
  assert.equal(Number(await f.p.locator('[data-overview-value=tasks_total]').innerText()), tasks.length);
  pass('overview-whole-project-authoritative-counts', {total: tasks.length, states: expected, open_epics: 2});

  const late = f.tasks.at(-1);
  for (const query of ['ёжик', 'пОзДнЕе описание', late.key, '<img src=x']) {
    await goto(f, '/web/tasks?q=' + encodeURIComponent(query));
    assert.deepEqual(await taskKeys(f.p), [late.key]);
    assert.equal(await f.p.locator('main img').count(), 0); assert.equal(await f.p.evaluate(() => globalThis.qaXss || 0), 0);
  }
  await goto(f, '/web/tasks?include_closed=true&layout=list');
  const firstKeys = await taskKeys(f.p); assert.equal(firstKeys.length, 100); assert(!firstKeys.includes(late.key));
  const next = await f.p.locator('.pagination a').getAttribute('href');
  assert.equal(new URL(next, f.origin).searchParams.get('layout'), 'list');
  await goto(f, next); const rest = await taskKeys(f.p);
  assert.equal(new Set([...firstKeys, ...rest]).size, tasks.length); assert.equal(firstKeys.length + rest.length, tasks.length);
  await goto(f, next + '&q=changed', 400);
  await goto(f, '/web/tasks?role=developer&state=ready&epic_key=' + f.otherEpic + '&layout=list');
  const filtered = await taskKeys(f.p); assert.deepEqual(filtered, [f.tasks[110].key]);
  await f.p.getByRole('link', {name: 'Доска', exact: true}).click();
  assert.deepEqual(await taskKeys(f.p), filtered); assert.equal(await f.p.locator('.kanban-column').count(), 5);
  assert.match(await f.p.locator('main').innerText(), /Колонки содержат задачи этой страницы/);
  await f.p.locator('.filters [name=q]').fill('no-such-task'); await f.p.locator('.filters button').click();
  await f.p.waitForURL(url => url.searchParams.get('q') === 'no-such-task');
  assert.equal(new URL(f.p.url()).searchParams.has('cursor'), false); assert.equal((await taskKeys(f.p)).length, 0);
  pass('search-cyrillic-xss-beyond-page-list-board-filter-cursor', {first_page: firstKeys.length, second_page: rest.length, searched_key: late.key});

  await goto(f, '/web/agents'); assert.equal(await f.p.locator('.agent-row').count(), 2);
  const firstID = f.firstSession.session_id, secondID = f.secondSession.session_id;
  for (const id of [firstID, secondID]) assert.equal(await f.p.locator('a[href="/web/agents/' + id + '"]').count(), 1);
  await goto(f, '/web/agents?kind=human'); assert.equal(await f.p.locator('.agent-row').count(), 1);
  assert.equal(await f.p.locator('.agent-row a.entity-name').innerText().then(t => t.trim()), 'Человек');
  await goto(f, '/web/agents?kind=agent&limit=1'); const agentFirst = await f.p.locator('.agent-row a.entity-name').getAttribute('href');
  const agentNext = await f.p.locator('.pagination a').getAttribute('href');
  await goto(f, agentNext); assert.notEqual(await f.p.locator('.agent-row a.entity-name').getAttribute('href'), agentFirst);
  await goto(f, agentNext + '&owner=changed', 400);
  await goto(f, '/web/agents?kind=human&role=developer', 400);
  const claim = await call(f.worker, 'next', {task_key: f.tasks[5].key});
  await goto(f, '/web/agents/' + firstID);
  assert.match(await f.p.locator('main').innerText(), /synthetic-first/);
  assert.equal(await f.p.locator('section .task-card h3 a[href="/web/tasks/' + claim.task.key + '"]').count(), 1);
  assert.match(await f.p.locator('main').innerText(), /Сессия с действующим доступом не подтверждает запуск процесса/);
  const before = await call(f.worker, 'get', {task_key: claim.task.key});
  for (const route of ['/web/', '/web/agents', '/web/agents/' + firstID, '/web/data/overview']) {
    const r = await f.ctx.request.get(f.origin + route); assert.equal(r.status(), 200);
  }
  const after = await call(f.worker, 'get', {task_key: claim.task.key});
  assert.equal(after.lease.expires_at, before.lease.expires_at); assert.equal(after.lease.generation, before.lease.generation);
  await call(f.worker, 'release', {task_key: claim.task.key, generation: claim.task.lease.generation, reason: 'Synthetic release'});
  await goto(f, '/web/agents/' + firstID); assert.equal(await f.p.locator('.task-card').count(), 0);
  const victim = await call(f.second, 'next', {task_key: f.tasks[6].key});
  const held = await overview(f); assert.equal(held.leased_tasks, 1);
  h.revoke(f.state, f.secondAgent);
  const revoked = await overview(f); assert.equal(revoked.active_agent_sessions, 1); assert.equal(revoked.leased_tasks, 0);
  assert.equal(revoked.tasks_in_progress, 1, 'workflow must remain distinct from authorized lease count');
  await goto(f, '/web/agents/' + secondID);
  assert.equal(await f.p.locator('[data-state=credential_invalid]').count(), 1); assert.equal(await f.p.locator('.task-card').count(), 0);
  assert(Date.parse(f.secondSession.expires_at) > Date.now());
  await goto(f, '/web/agents?kind=agent&state=credential_invalid'); assert.equal(await f.p.locator('.agent-row').count(), 1);
  assert.match(await f.p.locator('main').innerText(), new RegExp(h.path.basename(f.secondAgent, '.token')));
  pass('sessions-human-distinction-current-claim-release-revoked-effective-no-process', {agent_sessions_before: 2, after: 1, revoked_future_ttl: true, revoked_task: victim.task.key, read_lease_unchanged: true});

  await goto(f, '/web/activity?limit=7&type=task_created');
  const cursors = await eventCursors(f.p); assert.equal(cursors.length, 7); assert(cursors.every((n, i) => i === 0 || n < cursors[i - 1]));
  const older = await f.p.locator('.pagination a').getAttribute('href');
  const added = await call(f.worker, 'create', {type: 'task', epic_key: f.epic, role: 'developer', title: 'Synthetic new insertion', description: 'After first page'});
  await goto(f, older); const olderCursors = await eventCursors(f.p); assert(olderCursors.every(n => n < cursors.at(-1)));
  assert.equal(new Set([...cursors, ...olderCursors]).size, cursors.length + olderCursors.length);
  await goto(f, '/web/activity?task_key=' + added.key + '&session_id=' + firstID + '&type=task_created');
  const exact = await eventCursors(f.p); assert.equal(exact.length, 1); assert(exact[0] > cursors[0]);
  await goto(f, '/web/activity?type=not-a-type', 400);
  pass('activity-desc-filter-before-limit-insertion-cursor', {first: cursors, second: olderCursors, inserted: exact[0]});

  await goto(f, '/web/stats?days=1&group_by=owner');
  const text = await f.p.locator('main').innerText(); assert.match(text, /Данные неполные/); assert.match(text, /66\.7%/);
  const metrics = await f.p.locator('.metric strong').allTextContents(); assert.deepEqual(metrics.map(t => Number(t.trim())), [2, 17, 2, 0]);
  assert.equal(await f.p.locator('[name=days]').inputValue(), '1'); assert.equal(await f.p.locator('[name=group_by]').inputValue(), 'owner');
  await goto(f, '/web/projects'); assert.equal(await f.p.locator('.project-card').count(), 1); assert.match(await f.p.locator('main').innerText(), /agent-board/);
  assert.equal(await f.p.locator('main form[method=post]').count(), 0);
  await goto(f, '/web/settings'); const settings = await f.p.locator('main').innerText();
  assert.match(settings, /600 с/); assert.match(settings, /180 с/); assert.match(settings, /Ревизия конфигурации\s+Нет данных/);
  assert.equal(await f.p.locator('main input, main select, main textarea').count(), 0);
  for (const file of [f.agent, f.human, f.secondAgent]) assert(!(await f.p.content()).includes(h.fs.readFileSync(file, 'utf8').trim()));
  pass('stats-unknown-coverage-project-single-readonly-settings-real', {completions: 2, tokens: 17, missing_tokens: 2, missing_share: 2 / 3, project_count: 1});
};
