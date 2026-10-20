'use strict';
// Unit-only DOM doubles exercise safe rendering and schema rejection without E2E.
const { test } = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const source = fs.readFileSync(`${__dirname}/assets/dashboard.js`, 'utf8');
class Element {
  constructor(tag) { this.tag = tag; this.children = []; this.dataset = {}; this.textContent = ''; }
  append(...children) { this.children.push(...children); }
  replaceChildren(...children) { this.children = children; this.replaced = true; }
  setAttribute(name, value) { this[name] = value; }
  contains(value) { return value === this || this.children.some((child) => child.contains(value)); }
}
function harness() {
  const selectors = {};
  for (const key of ['overview', 'dashboard-agents', 'dashboard-activity', 'dashboard-agents-time', 'dashboard-activity-time', 'dashboard-live', 'dashboard-option']) selectors[`[data-${key}]`] = new Element('div');
  const document = {
    activeElement: null, querySelector: (selector) => selectors[selector], querySelectorAll: () => [],
    createElement: (tag) => new Element(tag), createDocumentFragment: () => new Element('fragment'),
  };
  const window = {};
  vm.runInNewContext(source, { document, window, Date, Number, Object, Array });
  return { api: window.AccordDashboard, selectors, document };
}
function snapshot() {
  return {
    agents: { observed_at: '2026-10-20T12:00:01Z', sessions: [{ session_id: 'a'.repeat(64), role: 'developer', owner: '<script>bad</script>', machine: 'PC', run_id: 'run1', expires_at: '2026-10-20T13:00:00Z', status: 'active', task: { key: 'T-1', title: '<img src=x onerror=alert(1)>' } }] },
    activity: { observed_at: '2026-10-20T12:00:02Z', events: [{ cursor: 1, type: 'task_created', task_key: 'T-1', at: '2026-10-20T12:00:00Z', reason: '<script>bad</script>' }] },
  };
}
function descendants(node) { return [node, ...node.children.flatMap(descendants)]; }
test('valid panels create only trusted links and literal text', () => {
  const h = harness(); const value = snapshot();
  assert.equal(h.api.validate(value), true);
  assert.equal(h.api.apply(value), true);
  const nodes = descendants(h.selectors['[data-dashboard-agents]']);
  assert.ok(nodes.some((node) => node.textContent === 'T-1 · <img src=x onerror=alert(1)>'));
  assert.ok(nodes.some((node) => node.title === '<script>bad</script> · PC'));
  assert.ok(nodes.filter((node) => node.tag === 'a').every((node) => /^\/web\/(agents\/[a-f0-9]{64}|tasks\/T-1)$/.test(node.href)));
  assert.equal(h.selectors['[data-dashboard-agents-time]'].dateTime, value.agents.observed_at);
  assert.equal(h.selectors['[data-dashboard-activity-time]'].dateTime, value.activity.observed_at);
});
test('malformed session, event and unsafe route keys are rejected', () => {
  const h = harness();
  for (const mutate of [
    (v) => { v.agents.sessions[0].session_id = '../../login'; },
    (v) => { v.agents.sessions[0].task.key = 'javascript:alert(1)'; },
    (v) => { v.agents.sessions[0].status = 'running'; },
    (v) => { v.agents.sessions[0].role = 'human'; },
    (v) => { v.agents.sessions[0].task = false; },
    (v) => { v.activity.events[0].at = 'tomorrow'; },
    (v) => { v.activity.events[0].data = { owner: {} }; },
    (v) => { v.activity.events[0].cursor = 1.5; },
    (v) => { v.activity.events = new Array(21).fill(v.activity.events[0]); },
  ]) { const value = snapshot(); mutate(value); assert.equal(Boolean(h.api.validate(value)), false); }
});
test('empty collections render explicit empty states', () => {
  const h = harness(); const value = snapshot();
  value.agents.sessions = null; value.activity.events = [];
  assert.equal(h.api.validate(value), true); h.api.apply(value);
  assert.ok(descendants(h.selectors['[data-dashboard-agents]']).some((node) => node.textContent.includes('Активных сессий нет')));
});
test('collection containing keyboard focus keeps its DOM and timestamp', () => {
  const h = harness(); const value = snapshot();
  h.document.activeElement = h.selectors['[data-dashboard-agents]'];
  assert.equal(h.api.apply(value), false);
  assert.equal(h.selectors['[data-dashboard-agents]'].replaced, undefined);
  assert.equal(h.selectors['[data-dashboard-agents-time]'].dateTime, undefined);
  assert.equal(h.selectors['[data-dashboard-activity]'].replaced, true);
});
