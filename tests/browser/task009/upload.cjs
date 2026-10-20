'use strict';
const {assert, origin, sha} = require('../../process/task009/harness.cjs');
async function run(f, owner, pass) {
  const p = owner.page;
  await p.goto(origin + '/app/p/' + f.a + '/designs');
  await p.getByRole('button', {name: 'Загрузить', exact: true}).click();
  const dialog = p.getByRole('dialog');
  await dialog.locator('input[name=title]').fill('QA scoped browser upload over64KiB');
  const bytes = Buffer.alloc(1 << 20); bytes.write('PK\x05\x06', 0, 'binary');
  await dialog.locator('input[name=file]').setInputFiles({name: 'browser-scoped.zip', mimeType: 'application/zip', buffer: bytes});
  const posted = p.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname === '/web/p/' + f.a + '/designs');
  await dialog.getByRole('button', {name: 'Загрузить файл', exact: true}).click();
  assert.equal((await posted).status(), 200, 'actual Chrome/Next scoped upload over64KiB failed');
  await dialog.waitFor({state: 'hidden'});
  const card = p.locator('.asset-card').filter({hasText: 'QA scoped browser upload over64KiB'});
  await card.waitFor();
  const metadata = await f.aNative.http('designs?limit=100'); assert.equal(metadata.status, 200);
  const asset = metadata.data.assets.find(value => value.title === 'QA scoped browser upload over64KiB');
  assert(asset && asset.size === bytes.length && asset.sha256 === sha(bytes));
  assert(asset.download_url.startsWith('/api/v1/p/' + f.a + '/'));
  pass('Chrome-real-Next-qualified-upload-over64KiB-native-size-SHA', {bytes: bytes.length});
}
module.exports = {run};
