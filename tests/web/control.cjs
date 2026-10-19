'use strict';
// Real Go/native/Chrome acceptance. Browser clock/transport controls are recorded.
const fns = require('./control-fixture.cjs');
const {fs, path, root, sha, binary, assert} = fns;
const report = {schema: 1, started_at: new Date().toISOString(), test_sha256: fns.manifest(),
  binary_sha256: sha(fs.readFileSync(binary)),
  source_manifest_sha256: process.env.ACCORD_QA_SOURCE_MANIFEST ? sha(fs.readFileSync(process.env.ACCORD_QA_SOURCE_MANIFEST)) : undefined,
  scope: process.env.ACCORD_QA_CONTROL_SCOPE || 'full-control', checks: [],
  limitations: ['one macOS host; literal HTTP loopback; synthetic data only',
    'browser timers may be advanced; actual Go state and HTTP replies are never mocked',
    'production HTTPS and physical second PC not run']};
let f;
function pass(id, values = {}) {report.checks.push({id, result: 'PASS', ...values}); console.log(id + ' PASS');}
async function main() {
  f = await fns.fixture(); report.browser = f.browser.version(); report.node = process.version;
  report.processes = {server: f.srv.child.pid, native_workers: [f.worker.proc.pid, f.second.proc.pid]};
  if (report.scope === 'visibility-only') {pass('poll-real-hidden-pauses-visible-resumes', await require('./control-visibility.cjs')(f)); report.result = 'PASS'; return;}
  await fns.seed(f);
  if (!['poll-only', 'layout-poll-only'].includes(report.scope)) await require('./control-data.cjs')(f, pass);
  if (report.scope !== 'data-only') {
    if (report.scope !== 'poll-only') await require('./control-layout.cjs')(f, pass);
    await require('./control-poll.cjs')(f, pass);
  }
  report.result = 'PASS';
}
main().catch(async err => {
  report.result = 'FAIL'; report.failure = String(err.message).slice(0, 300); console.error(report.failure);
  if (f) for (const ctx of f.browser.contexts()) for (const p of ctx.pages()) {
    await p.screenshot({path: path.join(root, 'failure-' + Date.now() + '.png'), fullPage: true}).catch(() => {});
  }
}).finally(async () => {
  if (f) {
    f.srv.child.kill('SIGCONT');
    for (const close of [() => f.worker.close(), () => f.second.close(), () => f.browser.close(), () => fns.stop(f.srv)]) {
      try {await close();} catch {report.result = 'FAIL'; report.cleanup_failure = true;}
    }
  }
  fs.mkdirSync(root, {recursive: true, mode: 0o700});
  report.static_failures = f?.staticFailures;
  report.finished_at = new Date().toISOString();
  fs.writeFileSync(path.join(root, 'control-results.json'), JSON.stringify(report, null, 2), {mode: 0o600});
  if (report.result !== 'PASS') process.exitCode = 1;
});
