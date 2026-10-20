'use strict';
const { test } = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const source = fs.readFileSync(`${__dirname}/assets/commands.js`, 'utf8');
function harness(supported = true) {
  const handlers = new Map();
  const document = { activeElement: null, documentElement: { dataset: {} } };
  const element = (textContent = '') => ({
    textContent, hidden: false, value: '',
    addEventListener(name, callback) { handlers.set([this.id, name].join(':'), callback); },
    focus() { document.activeElement = this; },
  });
  const dialog = element(); dialog.id = 'dialog'; dialog.open = false;
  const input = element(); input.id = 'input';
  const trigger = element(); trigger.id = 'trigger'; trigger.hidden = true;
  const close = element(); close.id = 'close';
  const empty = element(); empty.id = 'empty';
  const items = [element('Задачи'), element('Агенты'), element('Статистика')];
  const original = element(); original.focus();
  dialog.querySelector = (selector) => selector === '[data-command-query]' ? input : selector === '[data-command-close]' ? close : empty;
  dialog.querySelectorAll = () => items;
  if (supported) dialog.showModal = () => { dialog.open = true; };
  dialog.close = () => { dialog.open = false; handlers.get('dialog:close')?.(); };
  document.querySelector = (selector) => selector === '#command-palette' ? dialog : selector === '[data-command-open]' ? trigger : null;
  document.addEventListener = (name, callback) => handlers.set(`document:${name}`, callback);
  vm.runInNewContext(source, { document, Array });
  function fire(target, name, props = {}) {
    const event = { key: '', prevented: false, preventDefault() { this.prevented = true; }, ...props };
    handlers.get(`${target}:${name}`)?.(event);
    return event;
  }
  return { document, dialog, input, trigger, close, empty, items, original, fire };
}
test('keyboard shortcut opens native modal, arrows navigate and close restores focus', () => {
  const h = harness();
  assert.equal(h.trigger.hidden, false);
  assert.equal(h.fire('document', 'keydown', { ctrlKey: true, key: 'k' }).prevented, true);
  assert.equal(h.dialog.open, true);
  assert.equal(h.document.activeElement, h.input);
  h.fire('dialog', 'keydown', { key: 'ArrowDown' });
  assert.equal(h.document.activeElement, h.items[0]);
  h.fire('dialog', 'keydown', { key: 'ArrowUp' });
  assert.equal(h.document.activeElement, h.items[2]);
  h.fire('close', 'click');
  assert.equal(h.document.activeElement, h.original);
});
test('filter hides irrelevant links and reports empty navigation without fetching', () => {
  const h = harness(); h.fire('trigger', 'click');
  h.input.value = 'АГЕН'; h.fire('input', 'input');
  assert.equal(h.items[0].hidden, true); assert.equal(h.items[1].hidden, false);
  assert.equal(h.empty.hidden, true);
  h.input.value = 'T-999'; h.fire('input', 'input');
  assert.equal(h.empty.hidden, false);
  assert.equal(h.fire('dialog', 'keydown', { key: 'ArrowDown' }).prevented, false);
});
test('unsupported native dialog leaves enhancement hidden and GET search intact', () => {
  const h = harness(false);
  assert.equal(h.trigger.hidden, true);
  assert.equal(h.document.documentElement.dataset.commands, undefined);
});
