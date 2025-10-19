// Apply the theme before styles paint, so a saved dark theme never flashes light.
(() => {
    let saved;
    try { saved = localStorage.getItem('yaplaneTheme'); } catch { /* Storage may be unavailable. */ }
    const theme = saved === 'dark' || saved === 'light'
        ? saved
        : window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    document.documentElement.dataset.theme = theme;
})();
