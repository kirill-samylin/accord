'use strict';
// Only trusted local routes are constructed; server text is always a text node.
(() => {
  const root = document.querySelector('[data-overview]');
  const agentList = document.querySelector('[data-dashboard-agents]');
  const eventList = document.querySelector('[data-dashboard-activity]');
  if (!root || !agentList || !eventList) return;
  const roles = { coordinator: 'Координатор', analyst: 'Аналитик', architect: 'Архитектор', developer: 'Разработчик', reviewer: 'Ревьюер', tester: 'Тестировщик', human: 'Человек' };
  const states = { created: 'Создано', ready: 'Готово к работе', in_progress: 'В работе', done: 'Завершено', cancelled: 'Отменено' };
  const labels = {};
  // Labels are emitted by the server from the same vocabulary as SSR rows.
  for (const label of document.querySelectorAll('[data-event-type]')) labels[label.dataset.eventType] = label.textContent;
  const text = (value) => typeof value === 'string';
  const optionalText = (value) => value === undefined || text(value);
  const date = (value) => text(value) && Number.isFinite(Date.parse(value));
  const sid = (value) => text(value) && /^[a-f0-9]{64}$/.test(value);
  const key = (value) => text(value) && /^[ET]-[1-9][0-9]*$/.test(value);
  const list = (value, limit) => value === null || Array.isArray(value) && value.length <= limit;
  const task = (value) => value == null || typeof value === 'object' && key(value.key) && text(value.title);
  function validSession(value) {
    return value && sid(value.session_id) && Object.hasOwn(roles, value.role) && value.role !== 'human'
      && text(value.owner) && text(value.machine) && text(value.run_id) && value.status === 'active'
      && date(value.expires_at) && task(value.task);
  }
  function validEvent(value) {
    const data = value?.data;
    return value && Number.isSafeInteger(value.cursor) && value.cursor >= 0 && text(value.type) && date(value.at)
      && (value.task_key === undefined || key(value.task_key)) && (value.session_id === undefined || sid(value.session_id))
      && optionalText(value.reason) && (data == null || typeof data === 'object'
        && ['owner', 'role', 'old_state', 'new_state'].every((field) => optionalText(data[field])));
  }
  function validate(snapshot) {
    return snapshot && snapshot.agents && snapshot.activity && date(snapshot.agents.observed_at)
      && date(snapshot.activity.observed_at) && list(snapshot.agents.sessions, 6) && list(snapshot.activity.events, 20)
      && (snapshot.agents.sessions || []).every(validSession) && (snapshot.activity.events || []).every(validEvent);
  }
  function node(tag, className, value) {
    const element = document.createElement(tag);
    if (className) element.className = className;
    if (value !== undefined) element.textContent = value;
    return element;
  }
  function link(className, value, href) {
    const element = node('a', className, value);
    element.href = href;
    return element;
  }
  function formatDate(value) { return new Date(value).toLocaleString('ru-RU', { timeZoneName: 'short' }); }
  function sessionRow(value) {
    const row = node('article', 'agent-row');
    const identity = node('div', 'agent-identity');
    const avatar = node('span', 'avatar', 'A');
    avatar.setAttribute('aria-hidden', 'true');
    const facts = node('div');
    facts.append(link('entity-name', roles[value.role], `/web/agents/${value.session_id}`));
    for (const [className, content] of [['hint truncate', `${value.owner} · ${value.machine}`], ['micro truncate', `Запуск: ${value.run_id}`]]) {
      const fact = node('div', className, content);
      fact.title = content;
      facts.append(fact);
    }
    identity.append(avatar, facts);
    const ownership = node('div');
    const badge = node('span', 'badge', 'Сессия активна');
    badge.dataset.state = value.status;
    ownership.append(badge);
    if (value.task) {
      const assigned = node('p', 'assigned-task');
      const title = `${value.task.key} · ${value.task.title}`;
      const taskLink = link('', title, `/web/tasks/${value.task.key}`);
      taskLink.title = title;
      assigned.append(taskLink, node('small', '', 'Задача во владении'));
      ownership.append(assigned);
    } else ownership.append(node('p', 'hint', 'Нет действующего владения задачей'));
    const time = node('time', 'micro', formatDate(value.expires_at));
    time.dateTime = value.expires_at;
    row.append(identity, ownership, time);
    return row;
  }
  function eventRow(value) {
    const row = node('article', 'event-row');
    const dot = node('span', 'event-dot');
    dot.setAttribute('aria-hidden', 'true');
    const facts = node('div');
    facts.append(node('strong', '', labels[value.type] || value.type));
    const meta = node('div', 'meta');
    if (value.task_key) meta.append(link('', value.task_key, `/web/tasks/${value.task_key}`));
    if (value.session_id) {
      const sessionLink = link('truncate', `Сессия ${value.session_id}`, `/web/agents/${value.session_id}`);
      sessionLink.title = value.session_id;
      meta.append(sessionLink);
    }
    facts.append(meta);
    const data = value.data;
    if (data?.role || data?.owner) facts.append(node('p', 'hint', [roles[data.role] || data.role, data.owner].filter(Boolean).join(' · ')));
    if (data?.new_state) facts.append(node('p', 'hint', [data.old_state, data.new_state].filter(Boolean).map((state) => states[state] || state).join(' → ')));
    if (value.reason) facts.append(node('p', 'prose', value.reason));
    const time = node('time', 'micro', formatDate(value.at));
    time.dateTime = value.at;
    facts.append(time);
    row.append(dot, facts, node('span', 'micro', `#${value.cursor}`));
    return row;
  }
  function apply(snapshot) {
    // Build both fragments before publishing either collection.
    const agents = document.createDocumentFragment();
    const events = document.createDocumentFragment();
    for (const value of snapshot.agents.sessions || []) agents.append(sessionRow(value));
    for (const value of snapshot.activity.events || []) events.append(eventRow(value));
    if (!(snapshot.agents.sessions || []).length) agents.append(node('div', 'empty', 'Активных сессий нет. Они появятся после подключения агентов.'));
    if (!(snapshot.activity.events || []).length) events.append(node('div', 'empty', 'В журнале пока нет событий.'));
    // Keep the focused link stable while somebody navigates with a keyboard.
    const focusedAgents = agentList.contains(document.activeElement);
    const focusedEvents = eventList.contains(document.activeElement);
    if (!focusedAgents) agentList.replaceChildren(agents);
    if (!focusedEvents) eventList.replaceChildren(events);
    for (const collection of ['agents', 'activity']) {
      if (collection === 'agents' && focusedAgents || collection === 'activity' && focusedEvents) continue;
      const time = document.querySelector(`[data-dashboard-${collection}-time]`);
      if (time) { time.dateTime = snapshot[collection].observed_at; time.textContent = formatDate(snapshot[collection].observed_at); }
    }
    return !focusedAgents && !focusedEvents;
  }
  const toggle = document.querySelector('[data-dashboard-live]');
  document.querySelector('[data-dashboard-option]').hidden = false;
  const note = document.querySelector('[data-dashboard-note]');
  if (note) note.hidden = false;
  window.AccordDashboard = { validate, apply, enabled: () => toggle.checked };
})();
