'use strict';
const {assert, origin} = require('../../process/task009/harness.cjs');
async function run(f, owner, pass) {
  const p = owner.page;
  await p.goto(origin + '/app/projects/' + f.a);
  const name = p.locator('input[name=name]').first(); await name.waitFor();
  await name.fill('QA Alpha unsent draft');
  await p.getByRole('link', {name: 'Все проекты', exact: true}).click();
  await p.locator(`a[href="/app/projects/${f.b}"]`).click();
  await p.locator('input[name=name]').first().waitFor();
  assert.equal(await p.locator('input[name=name]').first().inputValue(), 'QA Beta', 'catalog A draft leaked into B');
  await p.goto(origin + '/app/projects/' + f.a);
  await p.getByText('Подключить репозиторий', {exact: true}).first().click();
  const add = p.locator('details').filter({has: p.locator('summary', {hasText: 'Подключить репозиторий'})});
  await add.locator('input[name=url]').fill('file:///tmp/rejected');
  const invalid = p.waitForResponse(response => response.request().method() === 'POST' && response.url().endsWith('/repositories'));
  await add.getByRole('button', {name: 'Подключить репозиторий', exact: true}).click();
  assert.equal((await invalid).status(), 400);
  await add.getByRole('button', {name: 'Обновить форму', exact: true}).waitFor();
  assert.equal(await p.getByText('Результат отправки неизвестен.', {exact: false}).count(), 0, 'definitive validation became uncertain');
  await add.getByRole('button', {name: 'Обновить форму', exact: true}).click();
  await add.locator('input[name=url]').fill('https://github.com/example/ui-connected.git');
  const saved = p.waitForResponse(response => response.request().method() === 'POST' && response.url().endsWith('/repositories'));
  await add.getByRole('button', {name: 'Подключить репозиторий', exact: true}).click();
  assert.equal((await saved).status(), 200);
  await p.locator('.repository-card input[name=url]:visible').last().waitFor();
  assert((await f.owner.detail(f.a)).project.repositories.some(repository => repository.url === 'https://github.com/example/ui-connected.git'));
  pass('Chrome-catalog-detail-draft-isolation-repository-validation-corrected-save');
  await p.goto(origin + '/app/projects/' + f.a);
  await p.getByText('Подключить репозиторий', {exact: true}).first().click();
  const retryForm = p.locator('details').filter({has: p.locator('summary', {hasText: 'Подключить репозиторий'})});
  await retryForm.locator('input[name=url]').fill('https://github.com/example/ui-lost-response.git');
  const captured = [];
  const endpoint = '/web/catalog/projects/' + f.a + '/repositories';
  const matcher = url => url.pathname === endpoint;
  let first = true, committedStatus;
  await p.route(matcher, async route => {
    if (route.request().method() !== 'POST') return route.continue();
    captured.push(route.request().postData());
    if (!first) return route.continue();
    first = false;
    try {const actual = await route.fetch({timeout: 10000}); committedStatus = actual.status();}
    catch {committedStatus = 'transport-failure';}
    await route.abort('failed');
  });
  try {
    await retryForm.getByRole('button', {name: 'Подключить репозиторий', exact: true}).click();
    await p.getByRole('button', {name: 'Повторить ту же отправку', exact: true}).waitFor();
    assert.equal(committedStatus, 200, 'lost-response scenario did not observe actual committed response');
    assert.match(await p.locator('main').innerText(), /Переход в другой проект пока недоступен/);
    await p.getByRole('button', {name: 'Повторить ту же отправку', exact: true}).click();
    await p.getByRole('button', {name: 'Повторить ту же отправку', exact: true}).waitFor({state: 'hidden'});
    assert.equal(captured.length, 2);
    assert.equal(captured[0], captured[1], 'retry changed original project/capability/body');
    assert.equal((await f.owner.detail(f.a)).project.repositories.filter(repository => repository.url === 'https://github.com/example/ui-lost-response.git').length, 1);
    pass('Chrome-real-committed-lost-response-lock-and-exact-durable-retry');
  } finally {await p.unroute(matcher);}
}
module.exports = {run};
