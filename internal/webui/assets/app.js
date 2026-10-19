'use strict';

// Enhancement only: every mutation remains an ordinary protected HTML form.
const statusLine = document.querySelector('.submit-status');
function announce(message) {
  if (statusLine) statusLine.textContent = message;
}
function stopHeartbeat() {
  document.dispatchEvent(new Event('accord:stop-heartbeat'));
}

for (const form of document.querySelectorAll('form[method="post"]')) {
  form.addEventListener('submit', async (event) => {
    if (form.dataset.submitting === 'true') {
      event.preventDefault();
      return;
    }
    if (!navigator.onLine) {
      event.preventDefault();
      stopHeartbeat();
      announce('Нет подключения к сети. Действие не отправлено. Введённый текст остаётся на странице.');
      return;
    }
    const file = form.querySelector('input[type="file"]')?.files?.[0];
    const maxSize = form.hasAttribute('data-login') ? 4096 : 50 * 1024 * 1024;
    if (file && file.size > maxSize) {
      event.preventDefault();
      announce(form.hasAttribute('data-login') ? 'Файл токена должен быть не больше 4 КиБ.' : 'Размер файла превышает 50 МиБ. Выберите другой файл.');
      return;
    }
    if (form.hasAttribute('data-login')) {
      const token = form.querySelector('[name="token"]');
      if (Boolean(token?.value.trim()) === Boolean(file)) {
        event.preventDefault();
        announce('Укажите один способ входа: личный токен или его файл.');
        return;
      }
    }
    stopHeartbeat();
    const preserveAnswer = /\/(answer|release)$/.test(form.getAttribute('action') || '');
    if (preserveAnswer) event.preventDefault();
    const body = preserveAnswer ? new URLSearchParams(new FormData(form)) : null;
    form.dataset.submitting = 'true';
    for (const button of form.querySelectorAll('button[type="submit"]')) button.disabled = true;
    form.setAttribute('aria-busy', 'true');
    announce(form.hasAttribute('data-upload') ? 'Файл загружается. Дождитесь результата…' : 'Отправляем. Дождитесь результата…');
    if (!preserveAnswer) return;
    try {
      const response = await fetch(form.action, {
        method: 'POST', body, credentials: 'same-origin', redirect: 'follow',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      });
      if (response.ok && response.redirected) {
        window.location.assign(response.url);
        return;
      }
      const html = new DOMParser().parseFromString(await response.text(), 'text/html');
      const message = html.querySelector('.message.error')?.textContent?.trim();
      if ([400, 422].includes(response.status)) {
        const replacement = Array.from(html.forms).find((candidate) => candidate.getAttribute('action') === form.getAttribute('action'));
        const newID = replacement?.querySelector('[name="action_id"]')?.value;
        if (newID) form.querySelector('[name="action_id"]').value = newID;
        const line = form.querySelector('[data-field-error]');
        const nextLine = replacement?.querySelector('[data-field-error]');
        if (line) {
          const text = nextLine?.textContent?.trim() || message || '';
          line.textContent = text;
          line.hidden = !text;
          const field = form.querySelector('[name="answer"], [name="reason"]');
          field?.setAttribute('aria-invalid', String(Boolean(text)));
        }
      }
      announce(message || 'Не удалось подтвердить результат. Текст сохранён на этой странице. Проверьте состояние вопроса перед повтором.');
      if ([401, 403, 409].includes(response.status)) {
        form.dataset.submitting = 'true';
        const link = document.createElement('a');
        link.textContent = response.status === 401 ? 'Войти снова' : 'Обновить состояние вопроса';
        link.href = response.status === 401 ? '/web/login' : window.location.pathname;
        statusLine?.append(' ', link);
        return;
      }
    } catch {
      announce('Связь прервалась. Результат не подтверждён. Текст сохранён. Повтор использует то же действие; проверьте состояние вопроса.');
    }
    form.dataset.submitting = 'false';
    form.removeAttribute('aria-busy');
    for (const button of form.querySelectorAll('button[type="submit"]')) button.disabled = false;
  });
}

window.addEventListener('pageshow', () => {
  for (const form of document.querySelectorAll('form[data-submitting="true"]')) {
    form.dataset.submitting = 'false';
    form.removeAttribute('aria-busy');
    for (const button of form.querySelectorAll('button[type="submit"]')) button.disabled = false;
  }
});

const createType = document.querySelector('[data-create] [name="type"]');
function updateCreateFields() {
  const isTask = createType?.value === 'task';
  const group = document.querySelector('[data-task-fields]');
  if (group) group.hidden = !isTask;
  const epic = document.querySelector('#create-epic');
  if (epic) epic.required = isTask;
  const description = document.querySelector('#create-description');
  if (description) description.required = isTask;
}
createType?.addEventListener('change', updateCreateFields);
updateCreateFields();

const upload = document.querySelector('[data-upload]');
upload?.querySelector('[name="file"]')?.addEventListener('change', (event) => {
  const file = event.target.files?.[0];
  const line = upload.querySelector('[data-file-status]');
  if (line && file) line.textContent = `${file.name} · ${(file.size / (1024 * 1024)).toFixed(2)} МиБ. Формат проверит сервер.`;
});

const editor = document.querySelector('[data-answer-editor]');
if (editor) {
  const toggle = editor.querySelector('[data-heartbeat-toggle]');
  const line = editor.querySelector('[data-heartbeat-status]');
  let enabled = false;
  let timer;
  let inFlight;
  let actionID = editor.dataset.action;
  const interval = Math.max(1000, Number(editor.dataset.interval) * 1000 || 10000);
  function stop(message) {
    enabled = false;
    clearTimeout(timer);
    inFlight?.abort();
    toggle.textContent = 'Продлевать время ответа';
    toggle.setAttribute('aria-pressed', 'false');
    line.textContent = message;
  }
  async function tick() {
    if (!enabled || document.hidden || !navigator.onLine) {
      stop('Продление остановлено: вкладка скрыта или нет сети. Проверьте время владения.');
      return;
    }
    const controller = new AbortController();
    inFlight = controller;
    const timeout = setTimeout(() => controller.abort(), 15000);
    try {
      const response = await fetch('/web/heartbeat', {
        method: 'POST', credentials: 'same-origin', signal: controller.signal,
        headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'Accept': 'application/json' },
        body: new URLSearchParams({csrf: editor.dataset.csrf, action_id: actionID,
          task_key: editor.dataset.task, generation: editor.dataset.generation}),
      });
      if (!response.ok) throw new Error('heartbeat refused');
      const result = await response.json();
      if (typeof result.next_action_id !== 'string' || !result.next_action_id) throw new Error('missing action');
      actionID = result.next_action_id;
      const lease = result.session?.my_tasks?.find((task) => task.key === editor.dataset.task)?.lease;
      const expiry = document.querySelector('[data-lease-expiry]');
      if (lease?.expires_at && expiry) {
        expiry.dataset.leaseExpiry = lease.expires_at;
        expiry.textContent = new Date(lease.expires_at).toLocaleString('ru-RU');
      }
      if (!enabled) return;
      line.textContent = 'Время ответа продлевается, пока эта вкладка открыта и видима.';
      timer = setTimeout(tick, interval);
    } catch {
      stop('Продление остановлено. Результат не подтверждён. Проверьте состояние вопроса; текст ответа остаётся на странице.');
    } finally {
      clearTimeout(timeout);
      if (inFlight === controller) inFlight = undefined;
    }
  }
  toggle.hidden = false;
  toggle.setAttribute('aria-pressed', 'false');
  line.textContent = 'Автоматическое продление выключено. Включите его явно на время ответа.';
  toggle.addEventListener('click', () => {
    if (enabled) { stop('Автоматическое продление выключено.'); return; }
    if (document.hidden || !navigator.onLine) { stop('Продление недоступно: нет сети или вкладка скрыта.'); return; }
    enabled = true;
    toggle.textContent = 'Остановить продление';
    toggle.setAttribute('aria-pressed', 'true');
    tick();
  });
  document.addEventListener('visibilitychange', () => {
    if (document.hidden) stop('Продление остановлено: вы ушли со вкладки. Включите его явно при возвращении.');
  });
  window.addEventListener('offline', () => stop('Нет сети. Продление остановлено.'));
  window.addEventListener('pagehide', () => stop('Продление остановлено.'));
  document.addEventListener('accord:stop-heartbeat', () => stop('Продление остановлено перед отправкой формы.'));
}

function updateRemaining() {
  const expiry = document.querySelector('[data-lease-expiry]');
  const remaining = document.querySelector('[data-lease-remaining]');
  if (!expiry || !remaining) return;
  const seconds = Math.ceil((Date.parse(expiry.dataset.leaseExpiry) - Date.now()) / 1000);
  remaining.textContent = seconds > 0 ? ` · осталось ${Math.floor(seconds / 60)} мин ${seconds % 60} с` : ' · срок по часам браузера истёк; проверьте состояние вопроса';
}
updateRemaining();
if (document.querySelector('[data-lease-expiry]')) setInterval(updateRemaining, 1000);
