'use strict';

// Read-only polling is isolated from theme, navigation and all mutation forms.
(() => {
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
      const dashboard = window.AccordDashboard;
      const livePanels = Boolean(dashboard?.enabled());
      const response = await fetch(livePanels ? '/web/data/dashboard' : '/web/data/overview', {
        method: 'GET', credentials: 'same-origin', cache: 'no-store',
        redirect: 'error', headers: { Accept: 'application/json' }, signal: controller.signal,
      });
      if (response.status === 401) {
        terminal('Вход больше не действует. Войдите снова после обновления страницы.');
        return;
      }
      if (!response.headers.get('Content-Type')?.toLowerCase().startsWith('application/json')) throw new Error('Unexpected content type');
      const payload = await response.json();
      const snapshot = livePanels ? payload?.overview : payload;
      if (response.status === 409 && payload?.code === 'LEASE_LOST') {
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
      if (!validSnapshot(snapshot) || livePanels && !dashboard.validate(payload)) throw new Error('Invalid overview');
      if (!eligible()) return;
      const panelsApplied = livePanels && dashboard.enabled() && dashboard.apply(payload);
      for (const counter of counters) counter.textContent = String(snapshot[counter.dataset.overviewValue]);
      if (time) {
        time.dateTime = snapshot.observed_at;
        time.textContent = new Date(snapshot.observed_at).toLocaleString('ru-RU', { timeZoneName: 'short' });
      }
      failures = 0;
      say(panelsApplied ? 'Счётчики, агенты и события обновлены. Следующее обновление через 15 секунд.' : livePanels && dashboard.enabled() ? 'Счётчики обновлены. Обновление списка отложено, пока на нём находится фокус.' : 'Счётчики обновлены. Следующее обновление через 15 секунд.');
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
  document.querySelector('[data-dashboard-live]')?.addEventListener('change', () => {
    resumeRequested = Boolean(request);
    schedule(0);
  });
  if (navigator.onLine) schedule();
  else say('Нет сети. Показаны последние данные; обновление приостановлено.');
})();
