import { icon } from './ui.js';

const storageKey = 'yaplaneTheme';

export function initTheme() {
    const button = document.getElementById('themeToggle');
    const system = window.matchMedia('(prefers-color-scheme: dark)');
    let hasPreference = false;
    try { hasPreference = ['dark', 'light'].includes(localStorage.getItem(storageKey)); } catch { /* Use the system preference. */ }

    function apply(theme) {
        const dark = theme === 'dark';
        document.documentElement.dataset.theme = theme;
        button.setAttribute('aria-pressed', String(dark));
        button.title = dark ? 'Switch to light mode' : 'Switch to dark mode';
        button.innerHTML = icon(dark ? 'sun' : 'moon');
        document.querySelector('meta[name="theme-color"]').content = dark ? '#171c19' : '#f6f5f1';
    }

    apply(document.documentElement.dataset.theme || (system.matches ? 'dark' : 'light'));
    button.addEventListener('click', () => {
        const theme = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
        hasPreference = true;
        try { localStorage.setItem(storageKey, theme); } catch { /* The choice still works for this page. */ }
        apply(theme);
    });
    system.addEventListener('change', event => {
        if (!hasPreference) apply(event.matches ? 'dark' : 'light');
    });
    window.addEventListener('storage', event => {
        if (event.key !== storageKey && event.key !== null) return;
        hasPreference = event.newValue === 'dark' || event.newValue === 'light';
        apply(hasPreference ? event.newValue : system.matches ? 'dark' : 'light');
    });
}
