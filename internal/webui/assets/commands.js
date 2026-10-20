'use strict';
// Native dialog supplies modal focus containment; navigation stays real GET links.
(() => {
  const dialog = document.querySelector('#command-palette');
  const trigger = document.querySelector('[data-command-open]');
  if (!dialog || !trigger || typeof dialog.showModal !== 'function') return;
  const input = dialog.querySelector('[data-command-query]');
  const items = Array.from(dialog.querySelectorAll('[data-command-item]'));
  const empty = dialog.querySelector('[data-command-empty]');
  let previousFocus = null;
  function filter() {
    const query = input.value.trim().toLocaleLowerCase('ru');
    for (const item of items) item.hidden = !item.textContent.toLocaleLowerCase('ru').includes(query);
    empty.hidden = items.some((item) => !item.hidden);
  }
  function open() {
    if (dialog.open) return;
    // Avoid two overlapping modal surfaces when the mobile menu was open.
    document.querySelector('[data-drawer-close]')?.click();
    previousFocus = document.activeElement;
    dialog.showModal();
    input.value = '';
    filter();
    input.focus();
  }
  trigger.hidden = false;
  document.documentElement.dataset.commands = 'true';
  trigger.addEventListener('click', open);
  input.addEventListener('input', filter);
  dialog.querySelector('[data-command-close]').addEventListener('click', () => dialog.close());
  dialog.addEventListener('close', () => { previousFocus?.focus(); });
  dialog.addEventListener('click', (event) => { if (event.target === dialog) dialog.close(); });
  document.addEventListener('keydown', (event) => {
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k' && !event.altKey) {
      event.preventDefault();
      if (dialog.open) dialog.close(); else open();
    }
  });
  dialog.addEventListener('keydown', (event) => {
    if (!['ArrowDown', 'ArrowUp'].includes(event.key)) return;
    const visible = items.filter((item) => !item.hidden);
    if (!visible.length) return;
    event.preventDefault();
    const current = visible.indexOf(document.activeElement);
    const next = event.key === 'ArrowDown' ? (current + 1) % visible.length : (current < 0 ? visible.length - 1 : (current - 1 + visible.length) % visible.length);
    visible[next].focus();
  });
})();
