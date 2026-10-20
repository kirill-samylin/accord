'use strict';
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

})();
