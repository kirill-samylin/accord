'use strict';
// TASK006 independent acceptance: real Go/native/Chrome, synthetic data only.
const h = require('./control-fixture.cjs');
const {fs, path, root, binary, sha, assert, call, goto} = h;
const report = {schema: 1, started_at: new Date().toISOString(), checks: [],
  source_manifest_sha256: sha(fs.readFileSync(h.required('ACCORD_QA_SOURCE_MANIFEST'))),
  binary_sha256: sha(fs.readFileSync(binary)), test_sha256: h.manifest(),
  limitations: ['one macOS host, literal HTTP loopback, synthetic data',
    'browser timers accelerated; Go state/HTTP responses never mocked', 'production HTTPS and second physical PC NOT-RUN']};
let f;
const pass = (id, evidence = {}) => {report.checks.push({id, result: 'PASS', ...evidence}); console.log(id + ' PASS');};
async function seed() {
  await goto(f, '/web/tasks#create'); await f.p.locator('#create-type').selectOption('epic');
  await f.p.locator('#create-title').fill('Synthetic refinement epic');
  await f.p.locator('#create-description').fill('Synthetic isolated acceptance');
  const posted = f.p.waitForResponse(r => r.request().method() === 'POST');
  await f.p.locator('[data-create] button[type=submit]').click(); assert.equal((await posted).status(), 303);
  await f.p.waitForURL(/\/web\/tasks\/E-\d+$/); f.epic = f.p.url().split('/').pop();
  f.tasks = [];
  for (const title of ['Задача ЁЖиК <img src=x onerror="globalThis.qaXss=1"> ' + 'длинное-имя-'.repeat(10), 'Synthetic done', 'Synthetic cancelled', 'Synthetic timeline', 'Synthetic refresh']) {
    f.tasks.push(await call(f.worker, 'create', {type: 'task', epic_key: f.epic, role: 'developer', title,
      description: 'Synthetic description ' + 'длинный контекст без потери доступности '.repeat(12)}));
  }
  const done = await call(f.worker, 'next', {task_key: f.tasks[1].key});
  await call(f.worker, 'transition', {task_key: done.task.key, generation: done.task.lease.generation, state: 'done', evidence: {commit: '1'.repeat(40), run_summary: 'Synthetic completion'}});
  await call(f.worker, 'transition', {task_key: f.tasks[2].key, state: 'cancelled', reason: 'Synthetic cancellation'});
  for (let i = 0; i < 6; i++) {
    const held = await call(f.worker, 'next', {task_key: f.tasks[3].key});
    await call(f.worker, 'release', {task_key: held.task.key, generation: held.task.lease.generation, reason: 'Synthetic timeline ' + i});
  }
  f.longClaim = await call(f.worker, 'next', {task_key: f.tasks[0].key});
}
async function main() {
  f = await h.fixture(); report.browser = f.browser.version(); report.node = process.version;
  await seed();
  await require('./refinement-features.cjs')(f, pass);
  await require('./refinement-visual.cjs')(f, pass);
  await require('./refinement-dashboard.cjs')(f, pass);
  assert.deepEqual(f.staticFailures, []); report.result = 'PASS';
}
main().catch(async error => {
  report.result = 'FAIL'; report.failure = String(error.stack).slice(0, 1200); console.error(report.failure);
  if (f) await f.p.screenshot({path: path.join(root, 'failure.png'), fullPage: true}).catch(() => {});
}).finally(async () => {
  if (f) {
    f.srv.child.kill('SIGCONT');
    for (const close of [() => f.worker.close(), () => f.second.close(), () => f.browser.close(), () => h.stop(f.srv)]) {
      try {await close();} catch {report.result = 'FAIL'; report.cleanup_failure = true;}
    }
    report.static_failures = f.staticFailures;
  }
  fs.mkdirSync(root, {recursive: true, mode: 0o700}); report.finished_at = new Date().toISOString();
  fs.writeFileSync(path.join(root, 'refinement-results.json'), JSON.stringify(report, null, 2), {mode: 0o600});
  if (report.result !== 'PASS') process.exitCode = 1;
});
