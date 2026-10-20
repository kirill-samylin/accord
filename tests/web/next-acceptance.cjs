'use strict';
const h = require('./next-fixture.cjs');
const report = {schema: 1, started_at: new Date().toISOString(), checks: [], source: h.required('ACCORD_QA_SOURCE_MANIFEST'), test_sha256: h.manifest(), scope: process.env.ACCORD_QA_NEXT_SCOPE || 'full', limitations: ['One physical macOS host; synthetic loopback Go + production Next + native MCP + Chrome.', 'Production HTTPS/VPS/second PC NOT-RUN. External process/model/cost/multi-project contracts N/A.', 'Transport offline fault does not prove a committed response was lost; idempotent retry is verified against real Go.']};
let f;
const pass = (id, evidence = {}) => {report.checks.push({id, result: 'PASS', ...evidence}); console.log(id + ' PASS');};
async function main() {
  const source = JSON.parse(h.fs.readFileSync(report.source, 'utf8')); const mismatches=[];
  for(const [file, digest] of Object.entries(source.source)) {if(h.sha(h.fs.readFileSync(h.path.resolve(__dirname, '../..', file))) !== digest) mismatches.push(file);}
  h.assert.deepEqual(mismatches, [], 'source differs from frozen manifest'); report.revision=source.revision; report.verified_source_files=Object.keys(source.source).length;
  report.binary_sha256=h.sha(h.fs.readFileSync(h.binary)); h.assert.equal(report.binary_sha256,source.artifacts['bin/agent-board']);
  report.next_build_id=h.fs.readFileSync(h.path.join(h.webDir,'.local/next/BUILD_ID'),'utf8').trim(); h.assert.equal(report.next_build_id,source.build_id);
  report.next_routes_sha256=h.sha(h.fs.readFileSync(h.path.join(h.webDir,'.local/next/routes-manifest.json'))); h.assert.equal(report.next_routes_sha256,source.artifacts['web/.local/next/routes-manifest.json']);
  f = await h.fixture(); report.processes = {go: f.go.child.pid, next: f.next.child.pid, worker: f.worker.proc.pid}; report.browser = f.browser.version(); report.node = process.version;
  report.source_manifest_sha256 = h.sha(h.fs.readFileSync(report.source)); report.binary_sha256 = f.goBinary;
  if(report.scope==='security') {
    report.limitations.push('Security-only subset; known presentation defects keep this old-build overall FAIL.'); f.findings.push({id:'command-focus-return',result:'FAIL',evidence:'artifact4-focus-01'},{id:'sidebar-height',result:'FAIL',evidence:'artifact4-full-05'});
    f.epic=await h.create(f,{type:'epic',title:'Synthetic security epic',description:'Isolated security acceptance'}); await f.p.clock.install(); await require('./next-security.cjs')(f,pass);
    h.assert.deepEqual(f.sourceErrors,[]);h.assert.deepEqual(f.consoleErrors,[]);h.assert.deepEqual(f.failedAssets,[]);pass('runtime-no-hydration-csp-or-static-errors');h.assert.deepEqual(f.findings,[],'known product findings remain open');return;
  }
  if(report.scope==='remaining') {
    report.limitations.push('Remaining subset: journeys/data/assets/visual are outside this run; known presentation findings from artifact4-full-05 keep overall FAIL.');
    f.findings.push({id:'command-focus-return',result:'FAIL',evidence:'artifact4-focus-01'},{id:'sidebar-height',result:'FAIL',evidence:'artifact4-full-05'});
    f.epic=await h.create(f,{type:'epic',title:'Synthetic remaining epic',description:'Isolated remaining network/security scope'}); await f.p.clock.install();
    await require('./next-poll.cjs')(f,pass); await require('./next-visibility.cjs')(f,pass); await require('./next-upload.cjs')(f,pass); await require('./next-security.cjs')(f,pass);
    h.assert.deepEqual(f.sourceErrors,[]); h.assert.deepEqual(f.consoleErrors,[]); h.assert.deepEqual(f.failedAssets,[]); pass('runtime-no-hydration-csp-or-static-errors'); h.assert.deepEqual(f.findings,[],'known product findings remain open'); return;
  }
  await require('./next-journeys.cjs')(f, pass);
  if (report.scope === 'journeys') return;
  await require('./next-transport.cjs')(f, pass);
  await require('./next-data.cjs')(f, pass);
  await require('./next-assets.cjs')(f, pass);
  await require('./next-visual.cjs')(f, pass);
  await require('./next-focus.cjs')(f, pass);
  await require('./next-poll.cjs')(f, pass);
  await require('./next-visibility.cjs')(f, pass);
  if (report.scope !== 'no-deadlines') await require('./next-upload.cjs')(f, pass);
  await require('./next-security.cjs')(f, pass);
  h.assert.deepEqual(f.sourceErrors, []); h.assert.deepEqual(f.consoleErrors, []); h.assert.deepEqual(f.failedAssets, []);
  pass('runtime-no-hydration-csp-or-static-errors'); h.assert.deepEqual(f.findings,[],'known product findings remain open');
}
main().then(() => {report.result = 'PASS';}).catch(async error => {
  report.result = 'FAIL'; if(error.fixtureDiagnostics) report.fixture_diagnostics=error.fixtureDiagnostics; report.failure = String(error.stack).slice(0, 1600); console.error(report.failure);
  if (f?.p) await f.p.screenshot({animations:'disabled',path: h.path.join(h.root, 'failure.png'), fullPage: true}).catch(() => {});
}).finally(async () => {
  if (f) {try {await h.cleanup(f);} catch (error) {report.cleanup_failure = String(error.message); report.result = 'FAIL';} report.product_findings = f.findings; report.go_processes = f.goProcesses; report.next_exit={code:f.next?.child.exitCode,signal:f.next?.child.signalCode}; report.page_errors = f.sourceErrors; report.console_errors = f.consoleErrors; report.failed_assets = f.failedAssets;}
  h.fs.mkdirSync(h.root, {recursive: true, mode: 0o700}); report.finished_at = new Date().toISOString();
  h.fs.writeFileSync(h.path.join(h.root, 'next-results.json'), JSON.stringify(report, null, 2), {mode: 0o600});
  if (report.result !== 'PASS') process.exitCode = 1;
});
