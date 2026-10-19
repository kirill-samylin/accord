'use strict';
// Pure unit harness: no browser/server/process fixture and no external packages.
const { test } = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const source = fs.readFileSync(`${__dirname}/assets/control.js`, 'utf8');

function harness(fetcher, expiry = '2099-01-01T00:00:00Z') {
  const timers = new Map();
  const listeners = {};
  const status = { textContent: 'Initial snapshot' };
  const time = { textContent: 'Initial time' };
  const counter = { dataset: { overviewValue: 'tasks_total' }, textContent: '42' };
  let timerId = 0;
  const overview = {
    dataset: { sessionExpiry: expiry },
    querySelector: (selector) => selector.includes('status') ? status : time,
    querySelectorAll: () => [counter],
  };
  const document = {
    visibilityState: 'visible', documentElement: { dataset: {} },
    querySelector: (selector) => selector === '[data-overview]' ? overview : null,
    addEventListener: (name, callback) => { listeners[name] = callback; },
  };
  const navigator = { onLine: true };
  let calls = 0;
  const window = {
    matchMedia: () => ({ matches: false, addEventListener: () => {} }),
    addEventListener: (name, callback) => { listeners[name] = callback; },
  };
  vm.runInNewContext(source, {
    document, navigator, window, localStorage: { getItem: () => null },
    Date, Number, String, Boolean, Array, Error, AbortController,
    setTimeout: (callback, delay) => { timers.set(++timerId, { callback, delay }); return timerId; },
    clearTimeout: (id) => timers.delete(id),
    fetch: async (...args) => { calls++; return fetcher(...args); },
  });
  return {
    document, navigator, timers, status, time, counter, listeners,
    get calls() { return calls; },
    async fire(delay) {
      const item = [...timers].find(([, value]) => value.delay === delay);
      assert.ok(item, `missing timer ${delay}`);
      timers.delete(item[0]);
      await item[1].callback();
      await Promise.resolve();
    },
  };
}
function snapshot() {
  return {
    project: 'Project', observed_at: '2026-10-19T12:00:00Z', tasks_total: 98,
    tasks_created: 0, tasks_ready: 0, tasks_in_progress: 0, tasks_done: 0,
    tasks_cancelled: 0, tasks_blocked: 0, open_epics: 0,
    active_agent_sessions: 0, active_human_sessions: 0, leased_tasks: 0, latest_event_cursor: 99,
  };
}
function response(status, data, contentType = 'application/json') {
  return { status, ok: status === 200, headers: { get: (key) => key === 'Content-Type' ? contentType : null }, json: async () => data };
}

test('poll reads only overview and changes counters after validated success', async () => {
  const h = harness(async (url, options) => {
    assert.equal(url, '/web/data/overview');
    assert.equal(options.method, 'GET');
    assert.equal(options.credentials, 'same-origin');
    assert.equal(options.redirect, 'error');
    return response(200, snapshot());
  });
  await h.fire(15000);
  assert.equal(h.counter.textContent, '98');
  assert.equal(h.time.dateTime, '2026-10-19T12:00:00Z');
  assert.ok([...h.timers.values()].some((t) => t.delay === 15000));
});

test('failure, wrong type and invalid shape retain last good snapshot and back off', async () => {
  for (const result of [response(503, {}), response(200, snapshot(), 'text/html'), response(200, { ...snapshot(), tasks_total: 'fake' })]) {
    const h = harness(async () => result);
    await h.fire(15000);
    assert.equal(h.counter.textContent, '42');
    assert.equal(h.time.textContent, 'Initial time');
    assert.match(h.status.textContent, /Последние данные сохранены/);
    assert.ok([...h.timers.values()].some((t) => t.delay === 30000));
  }
});

test('401 and expired board session are terminal across BFCache restore', async () => {
  for (const result of [response(401, {}), response(409, { code: 'LEASE_LOST' })]) {
    const h = harness(async () => result);
    await h.fire(15000);
    assert.equal(h.timers.size, 0);
    h.listeners.pageshow({ persisted: true });
    assert.equal(h.timers.size, 0);
    assert.equal(h.counter.textContent, '42');
  }
});

test('hidden/offline pages pause and back/forward restoration requests fresh data', async () => {
  const h = harness(async () => response(200, snapshot()));
  h.document.visibilityState = 'hidden';
  h.listeners.visibilitychange();
  assert.equal(h.timers.size, 0);
  h.document.visibilityState = 'visible';
  h.listeners.visibilitychange();
  await h.fire(0);
  assert.equal(h.calls, 1);
  h.navigator.onLine = false;
  h.listeners.offline();
  assert.equal(h.timers.size, 0);
  h.navigator.onLine = true;
  h.listeners.online();
  await h.fire(0);
  h.listeners.pagehide();
  assert.equal(h.timers.size, 0);
  h.listeners.pageshow({ persisted: true });
  await h.fire(0);
  assert.equal(h.calls, 3);
});

test('local session expiry stops before any request', async () => {
  const h = harness(async () => { throw new Error('must not fetch'); }, '2000-01-01T00:00:00Z');
  await h.fire(15000);
  assert.equal(h.calls, 0);
  assert.equal(h.timers.size, 0);
  assert.match(h.status.textContent, /истёк/);
});

test('one in-flight request; timeout abort retains counters', async () => {
  let activeSignal;
  const h = harness((_url, options) => new Promise((_resolve, reject) => {
    activeSignal = options.signal;
    options.signal.addEventListener('abort', () => reject(new Error('aborted')));
  }));
  const running = h.fire(15000);
  h.listeners.online();
  await h.fire(0);
  assert.equal(h.calls, 1);
  await h.fire(8000);
  await running;
  assert.equal(activeSignal.aborted, true);
  assert.equal(h.counter.textContent, '42');
  assert.ok([...h.timers.values()].some((t) => t.delay === 30000));
});
