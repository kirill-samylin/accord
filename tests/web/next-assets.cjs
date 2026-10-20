'use strict';
const h = require('./next-fixture.cjs');
module.exports = async (f, pass) => {
  const p = f.p; f.assets = [];
  async function upload(filename, buffer, scope, key = '', design = '') {
    await h.goto(f, '/app/designs' + (design ? '?design_id=' + design : ''));
    await p.getByRole('button', {name: 'Загрузить', exact: true}).click(); const dialog = p.getByRole('dialog');
    await dialog.locator('[name=title]').fill('Synthetic ' + filename);
    if (design) {h.assert.equal(await dialog.locator('[name=scope]').inputValue(),scope,'new version lost inherited scope'); h.assert.equal(await dialog.locator('[name=scope_key]').inputValue(),key,'new version lost inherited key');}
    else if(scope === 'project') h.assert.equal(await dialog.locator('[name=scope]').inputValue(),'project','default upload scope not project');
    else {await dialog.locator('[name=scope]').selectOption(scope); await dialog.locator('[name=scope_key]').fill(key); if(scope === 'task') await dialog.locator('[name=epic_key]').fill(f.epic);}
    await dialog.locator('[name=file]').setInputFiles({name: filename, mimeType: 'application/octet-stream', buffer});
    await p.screenshot({animations:'disabled',path:h.path.join(h.root,'upload-'+filename.replace(/[^a-z0-9.-]/g,'_')+'.png')});
    await h.posted(p, () => dialog.getByRole('button', {name: /Загрузить (файл|новую версию)/}).click()); await dialog.waitFor({state: 'detached'});
    const resources = await f.worker.resource('accord://designs'); const asset = resources.assets.find(a => a.filename === filename); h.assert(asset, 'upload absent from native design resource');
    h.assert.equal(asset.sha256, h.sha(buffer)); h.assert.equal(asset.scope, scope); h.assert.equal(asset.scope_key, scope === 'project' ? 'agent-board' : key);
    const browser = await h.json(f, '/web/designs?design_id=' + asset.design_id); const dto = browser.designs.find(row => row.asset_id === asset.asset_id); h.assert(dto);
    const download = await f.ctx.request.get(f.origin + dto.download_url); h.assert.equal(download.status(), 200); h.assert.equal(h.sha(await download.body()), h.sha(buffer)); h.assert.match(download.headers()['content-disposition'], /^attachment/);
    const native = await fetch(f.go.origin + asset.download_url, {headers: {Authorization: 'Bearer ' + h.fs.readFileSync(f.agent, 'utf8').trim(), 'X-Session': f.agentSession.session_id}}); h.assert.equal(native.status, 200); h.assert.equal(h.sha(Buffer.from(await native.arrayBuffer())), asset.sha256);
    f.assets.push(dto); return dto;
  }
  const png = await p.screenshot({animations:'disabled',clip: {x: 0, y: 0, width: 4, height: 4}});
  const first = await upload('synthetic.png', png, 'task', f.task);
  const second = await upload('synthetic-v2.svg', Buffer.from('<svg xmlns="http://www.w3.org/2000/svg" onload="globalThis.qaXss=3"><rect width="4" height="4"/><script>globalThis.qaXss=3</script></svg>'), 'task', f.task, first.design_id);
  h.assert.equal(first.version, 1); h.assert.equal(second.version, 2);
  await upload('synthetic.html', Buffer.from('<!doctype html><script>globalThis.qaXss=4</script><p>Synthetic design</p>'), 'epic', f.epic);
  await upload('synthetic.zip', Buffer.from('PK\x05\x06' + '\0'.repeat(18), 'binary'), 'project');
  await h.goto(f, '/app/designs'); await p.getByRole('button', {name: 'Просмотреть Synthetic synthetic.png, версия 1', exact: true}).click(); await p.getByRole('dialog').waitFor(); await p.locator('.asset-detail-preview img').waitFor();
  h.assert.equal(await p.evaluate(() => globalThis.qaXss || 0), 0); await p.keyboard.press('Escape');
  await p.getByRole('button', {name: 'Загрузить', exact: true}).click(); const dialog = p.getByRole('dialog'); await dialog.locator('[name=title]').fill('Synthetic invalid file');
  await dialog.locator('[name=file]').setInputFiles({name: 'invalid.zip', mimeType: 'application/zip', buffer: Buffer.from('badzip')});
  await h.posted(p, () => dialog.getByRole('button', {name: 'Загрузить файл', exact: true}).click(), 400); await dialog.locator('.mutation-notice').waitFor(); h.assert.equal(await dialog.locator('[name=title]').inputValue(), 'Synthetic invalid file');
  h.assert.equal(await dialog.locator('[name=file]').evaluate(el => el.files[0].name), 'invalid.zip'); await p.keyboard.press('Escape');
  const oversized = h.path.join(h.root, 'oversized.zip'); const fd = h.fs.openSync(oversized, 'wx', 0o600); h.fs.ftruncateSync(fd, 50 * 1024 * 1024 + 1); h.fs.closeSync(fd);
  await p.getByRole('button', {name: 'Загрузить', exact: true}).click(); await dialog.locator('[name=title]').fill('Synthetic over limit'); await dialog.locator('[name=file]').setInputFiles(oversized);
  let posts = 0; const observe = r => {if (r.method() === 'POST' && new URL(r.url()).pathname === '/web/designs') posts++;}; p.on('request', observe);
  await dialog.getByRole('button', {name: 'Загрузить файл', exact: true}).click(); await dialog.locator('.mutation-notice').waitFor(); h.assert.match(await dialog.locator('.mutation-notice').innerText(), /50 МиБ/); h.assert.equal(posts, 0); p.off('request', observe); await p.keyboard.press('Escape');
  pass('design-3scopes-version2-preview-native-download-sha-validation-file-retention-50MiB-preflight', {assets: f.assets.length});
};
