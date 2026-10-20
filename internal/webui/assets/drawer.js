'use strict';
(() => {
  const sidebar = document.querySelector('.sidebar');
  const drawerOpen = document.querySelector('[data-drawer-open]');
  const drawerClose = document.querySelector('[data-drawer-close]');
  const backdrop = document.querySelector('[data-drawer-backdrop]');
  const workspace = document.querySelector('.workspace');
  const narrow = window.matchMedia('(max-width: 767px)');
  document.documentElement.dataset.enhanced = 'true';
  if (drawerOpen) drawerOpen.hidden = false;
  if (drawerClose) drawerClose.hidden = false;
  let previousFocus = null;
  function closeDrawer() {
    if (!sidebar || sidebar.dataset.open !== 'true') return;
    sidebar.dataset.open = 'false';
    document.body.dataset.drawerOpen = 'false';
    drawerOpen?.setAttribute('aria-expanded', 'false');
    if (backdrop) backdrop.hidden = true;
    if (workspace) workspace.inert = false;
    if (previousFocus instanceof HTMLElement) previousFocus.focus();
  }
  drawerOpen?.addEventListener('click', () => {
    if (!sidebar || !narrow.matches) return;
    previousFocus = document.activeElement;
    sidebar.dataset.open = 'true';
    document.body.dataset.drawerOpen = 'true';
    drawerOpen.setAttribute('aria-expanded', 'true');
    if (backdrop) backdrop.hidden = false;
    if (workspace) workspace.inert = true;
    drawerClose?.focus();
  });
  drawerClose?.addEventListener('click', closeDrawer);
  backdrop?.addEventListener('click', closeDrawer);
  narrow.addEventListener('change', () => { if (!narrow.matches) closeDrawer(); });
  document.addEventListener('keydown', (event) => {
    if (sidebar?.dataset.open !== 'true') return;
    if (event.key === 'Escape') { event.preventDefault(); closeDrawer(); }
    if (event.key !== 'Tab') return;
    const focusable = Array.from(sidebar.querySelectorAll('a[href], button, input')).filter((element) => !element.disabled && element.getClientRects().length);
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
    if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
  });

})();
