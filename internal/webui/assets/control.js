'use strict';

// This enhancement never submits forms, renews leases or replaces page content.
(() => {
  const themeButton = document.querySelector('[data-theme-toggle]');
  const systemTheme = window.matchMedia('(prefers-color-scheme: dark)');
  let themePreference = null;
  try { themePreference = localStorage.getItem('accord-theme'); } catch { /* Preference storage is optional. */ }
  function applyTheme() {
    const dark = themePreference === 'dark' || (themePreference !== 'light' && systemTheme.matches);
    document.documentElement.dataset.theme = dark ? 'dark' : 'light';
    themeButton?.setAttribute('aria-label', dark ? 'Включить светлую тему' : 'Включить тёмную тему');
  }
  applyTheme();
  systemTheme.addEventListener('change', applyTheme);
  if (themeButton) {
    themeButton.hidden = false;
    themeButton.addEventListener('click', () => {
      themePreference = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
      try { localStorage.setItem('accord-theme', themePreference); } catch { /* Theme still works for this page. */ }
      applyTheme();
    });
  }

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

  const overview = document.querySelector('[data-overview]');
  if (!overview) return;
  const status = overview.querySelector('[data-overview-status]');
  const time = overview.querySelector('[data-overview-time]');
  const counters = Array.from(overview.querySelectorAll('[data-overview-value]'));
  const fields = ['tasks_total', 'tasks_created', 'tasks_ready', 'tasks_in_progress', 'tasks_done', 'tasks_cancelled', 'tasks_blocked', 'open_epics', 'active_agent_sessions', 'active_human_sessions', 'leased_tasks', 'latest_event_cursor'];
  const expiry = Date.parse(overview.dataset.sessionExpiry);
  let timer = null;
  let request = null;
  let failures = 0;
  let stopped = false;
  let pageActive = true;
  let resumeRequested = false;

  function say(message) { if (status) status.textContent = message; }
  function terminal(message) {
    stopped = true;
    clearTimeout(timer);
    request?.abort();
    say(message);
  }
  function eligible() {
    return !stopped && pageActive && document.visibilityState === 'visible' && navigator.onLine;
  }
  function schedule(delay = 15000) {
    clearTimeout(timer);
    if (eligible()) timer = setTimeout(refresh, delay);
  }
  function validSnapshot(value) {
    return value && typeof value === 'object' && typeof value.project === 'string'
      && typeof value.observed_at === 'string' && Number.isFinite(Date.parse(value.observed_at))
      && fields.every((key) => Number.isSafeInteger(value[key]) && value[key] >= 0);
  }
  async function refresh() {
    if (!eligible() || request) return;
    if (Number.isFinite(expiry) && Date.now() >= expiry) {
      terminal('Срок рабочей сессии истёк. Обновите страницу и продолжите работу явно.');
      return;
    }
    const controller = new AbortController();
    request = controller;
    const timeout = setTimeout(() => controller.abort(), 8000);
    say('Обновляем счётчики…');
    let retryDelay = 15000;
    try {
      const response = await fetch('/web/data/overview', {
        method: 'GET', credentials: 'same-origin', cache: 'no-store',
        redirect: 'error', headers: { Accept: 'application/json' }, signal: controller.signal,
      });
      if (response.status === 401) {
        terminal('Вход больше не действует. Войдите снова после обновления страницы.');
        return;
      }
      if (!response.headers.get('Content-Type')?.toLowerCase().startsWith('application/json')) throw new Error('Unexpected content type');
      const snapshot = await response.json();
      if (response.status === 409 && snapshot?.code === 'LEASE_LOST') {
        terminal('Срок рабочей сессии истёк. Обновите страницу и продолжите работу явно.');
        return;
      }
      if (!response.ok) {
        const retryAfter = Number(response.headers.get('Retry-After'));
        if ([429, 503].includes(response.status) && Number.isFinite(retryAfter) && retryAfter > 0) {
          retryDelay = Math.max(15000, Math.min(120000, retryAfter * 1000));
        }
        throw new Error('Overview unavailable');
      }
      if (!validSnapshot(snapshot)) throw new Error('Invalid overview');
      if (!eligible()) return;
      for (const counter of counters) counter.textContent = String(snapshot[counter.dataset.overviewValue]);
      if (time) {
        time.dateTime = snapshot.observed_at;
        time.textContent = new Date(snapshot.observed_at).toLocaleString('ru-RU', { timeZoneName: 'short' });
      }
      failures = 0;
      say('Счётчики обновлены. Следующее обновление через 15 секунд.');
    } catch {
      if (eligible()) {
        failures += 1;
        retryDelay = Math.max(retryDelay, Math.min(120000, 15000 * (2 ** Math.min(failures, 3))));
        say('Не удалось обновить счётчики. Последние данные сохранены; повторим позже.');
      }
    } finally {
      clearTimeout(timeout);
      request = null;
      schedule(resumeRequested ? 0 : retryDelay);
      resumeRequested = false;
    }
  }
  document.addEventListener('visibilitychange', () => {
    if (!eligible()) {
      clearTimeout(timer);
      request?.abort();
    } else schedule(0);
  });
  window.addEventListener('offline', () => {
    clearTimeout(timer);
    request?.abort();
    if (!stopped) say('Нет сети. Показаны последние данные; обновление приостановлено.');
  });
  window.addEventListener('online', () => { if (!stopped) schedule(0); });
  window.addEventListener('pagehide', () => {
    pageActive = false;
    clearTimeout(timer);
    request?.abort();
  });
  window.addEventListener('pageshow', (event) => {
    pageActive = true;
    if (event.persisted && !stopped) {
      resumeRequested = Boolean(request);
      schedule(0);
    }
  });
  if (navigator.onLine) schedule();
  else say('Нет сети. Показаны последние данные; обновление приостановлено.');
})();
