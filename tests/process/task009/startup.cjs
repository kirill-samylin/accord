'use strict';
const {assert, fs, path, root, binary, cli, python} = require('./harness.cjs');
function run(f, pass) {
  for (const mode of ['missing', 'empty']) {
    const destination = path.join(root, 'authority-' + mode);
    fs.cpSync(f.state, destination, {recursive: true, errorOnExist: true, force: false});
    fs.chmodSync(destination, 0o700);
    const registry = path.join(destination, 'projects-data/registry.sqlite');
    if (mode === 'missing') fs.unlinkSync(registry); else fs.truncateSync(registry);
    const before = python('capture', destination);
    const started = Date.now();
    const result = cli(['serve', '--db', path.join(destination, 'board.db'), '--web-data', path.join(destination, 'web'),
      '--web-origin', 'http://127.0.0.1:3100', '--listen', '127.0.0.1:3101', '--projects-data', path.join(destination, 'projects-data')], undefined, binary, true);
    assert(Number.isInteger(result.status) && result.status !== 0 && Date.now() - started < 5000, 'authority loss did not fail before listening');
    assert(!result.stdout.includes('listening'), 'authority-loss fixture started listener');
    if (mode === 'missing') assert(!fs.existsSync(registry), 'lost authority was silently recreated');
    else assert.equal(fs.statSync(registry).size, 0, 'empty authority was silently initialized');
    assert.deepEqual(python('capture', destination), before, 'authority-loss startup changed preserved board/web data');
  }
  pass('actual-process-marker-plus-missing-or-empty-registry-fails-before-listener');
}
module.exports = {run};
