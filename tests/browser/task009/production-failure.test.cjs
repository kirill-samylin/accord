'use strict';
// Execute only synthetic input/transport failures. No product build or deployed endpoint.
const {test} = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const {spawn} = require('node:child_process');
const root = fs.mkdtempSync(path.resolve(__dirname, '../../../.local/task009-qa/production-fix-'));
const config = path.join(root, 'synthetic-client.json'), token = path.join(root, 'synthetic.token');
fs.writeFileSync(config, JSON.stringify({origin: 'https://127.0.0.1:1'}), {mode: 0o600});
fs.writeFileSync(token, 'synthetic-invalid-credential', {mode: 0o600});
function run(name, overrides) {
  const out = path.join(root, name);
  const env = {...process.env}; for (const key of Object.keys(env)) if (key.startsWith('ACCORD_QA_')) delete env[key];
  Object.assign(env, {ACCORD_QA_DIR: out, ACCORD_QA_CLIENT_FILE: config, ACCORD_QA_TOKEN_FILE: token,
    ACCORD_QA_REVISION: '0'.repeat(40), ...overrides});
  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, [path.join(__dirname, 'production-readonly.cjs')], {env, stdio: ['ignore', 'pipe', 'pipe']});
    let output = ''; child.stdout.on('data', chunk => {output += chunk;}); child.stderr.on('data', chunk => {output += chunk;});
    const deadline = setTimeout(() => {child.kill('SIGKILL'); reject(new Error('Synthetic harness deadline'));}, 35000);
    child.on('error', error => {clearTimeout(deadline); reject(error);});
    child.on('exit', code => {
      clearTimeout(deadline);
      try {
        assert.equal(code, 1); assert(!output.includes('synthetic-invalid-credential'));
        const file = path.join(out, 'production-readonly-results.json');
        assert.equal(fs.statSync(file).mode & 0o777, 0o600);
        const report = JSON.parse(fs.readFileSync(file, 'utf8'));
        assert.equal(report.result, 'FAIL'); assert.equal(report.browser_closed, true);
        assert(!JSON.stringify(report).includes('127.0.0.1'));
        resolve(report);
      } catch (error) {reject(error);}
    });
  });
}
test('missing private input after safe output validation still writes redacted report', async () => {
  const report = await run('missing-input', {ACCORD_QA_PLAYWRIGHT: require.resolve('./production-support.cjs'), ACCORD_QA_TOKEN_FILE: path.join(root, 'absent.token')});
  assert.equal(report.failure.stage, 'INPUT');
});
test('actual Chrome transport failure still closes owned Chrome and writes report', {skip: !process.env.ACCORD_QA_FAILURE_PLAYWRIGHT || !process.env.ACCORD_QA_FAILURE_CHROME}, async () => {
  const report = await run('chrome-transport', {ACCORD_QA_PLAYWRIGHT: process.env.ACCORD_QA_FAILURE_PLAYWRIGHT, ACCORD_QA_CHROME: process.env.ACCORD_QA_FAILURE_CHROME});
  assert.equal(report.failure.stage, 'PUBLIC_ROOT'); assert(report.browser);
  assert.equal(report.checks.length, 0); assert.equal(report.browser_mutation_kinds.length, 0);
});
