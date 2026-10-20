'use strict';
// Synthetic harness failure paths; no deployed service, token or user data.
const {test} = require('node:test');
const assert = require('node:assert/strict');
const {EventEmitter} = require('node:events');
const {actionResponse, diagnostic, finish} = require('./production-support.cjs');
const response = {url: () => 'https://fixture.invalid/web/login', request: () => ({method: () => 'POST'})};
test('owned observation settles only matching POST and removes listener', async () => {
  const page = new EventEmitter();
  assert.equal(await actionResponse(page, '/web/login', async () => page.emit('response', response), 20), response);
  assert.equal(page.listenerCount('response'), 0);
});
test('early click failure owns observation cancellation without later rejection', async () => {
  const page = new EventEmitter(), original = new Error('private synthetic secret');
  await assert.rejects(actionResponse(page, '/web/login', async () => {throw original;}, 5), error => error === original);
  await new Promise(resolve => setTimeout(resolve, 15)); assert.equal(page.listenerCount('response'), 0);
});
test('response deadline is awaited and removes observer', async () => {
  const page = new EventEmitter();
  await assert.rejects(actionResponse(page, '/web/login', async () => {}, 5), {name: 'TimeoutError'});
  assert.equal(page.listenerCount('response'), 0);
});
test('wrong response cannot satisfy wait', async () => {
  const page = new EventEmitter();
  await assert.rejects(actionResponse(page, '/web/logout', async () => page.emit('response', response), 5), {name: 'TimeoutError'});
});
test('sanitized diagnostics never contain arbitrary names, messages, URLs, bodies or stacks', () => {
  const error = Object.assign(new Error('token-private https://private.invalid/body'), {name: 'private-user', status: 503});
  assert.deepEqual(diagnostic(error, 'LOGIN_SUBMIT'), {stage: 'LOGIN_SUBMIT', name: 'Error', code: 'INSPECTION_ERROR', status: 503});
});
test('cleanup failure preserves original diagnostic, closes browser, writes report', async () => {
  const report = {result: 'FAIL', failure: diagnostic(new SyntaxError(), 'AUTH_CATALOG')}, calls = [];
  await finish(report, {logout: async () => {calls.push('logout'); throw new Error('private');}, close: async () => calls.push('close'), write: value => {calls.push('write'); assert.equal(value.failure.stage, 'AUTH_CATALOG');}});
  assert.deepEqual(calls, ['logout', 'close', 'write']); assert.equal(report.cleanup_logout, 'FAIL'); assert.equal(report.browser_closed, true);
});
test('browser close failure still persists report and FAIL', async () => {
  const report = {result: 'PASS'}; let wrote = false;
  await finish(report, {close: async () => {throw new Error('private');}, write: () => {wrote = true;}});
  assert(wrote); assert.equal(report.result, 'FAIL'); assert.equal(report.browser_closed, false);
});
test('normal cleanup and close preserve PASS', async () => {
  const report = {result: 'PASS'};
  await finish(report, {logout: async () => {}, close: async () => {}, write: () => {}});
  assert.equal(report.cleanup_logout, 'PASS'); assert.equal(report.browser_closed, true); assert.equal(report.result, 'PASS');
});
