const paths = {
    moon: '<path d="M20.5 13A8.7 8.7 0 0 1 11 3.5 9 9 0 1 0 20.5 13Z"/>',
    sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5"/>',
    search: '<circle cx="10.5" cy="10.5" r="6.5"/><path d="m16 16 4 4"/>',
    home: '<rect x="3" y="3" width="7" height="7" rx="2"/><rect x="14" y="3" width="7" height="7" rx="2"/><rect x="3" y="14" width="7" height="7" rx="2"/><rect x="14" y="14" width="7" height="7" rx="2"/>',
    chat: '<path d="M21 11.5a8.5 8.5 0 0 1-8.5 8.5H4l-2 2V11.5A8.5 8.5 0 0 1 10.5 3h2a8.5 8.5 0 0 1 8.5 8.5Z"/><path d="M7 10h9M7 14h6"/>',
    user: '<circle cx="12" cy="8" r="4"/><path d="M4 21v-2a8 8 0 0 1 16 0v2"/>',
    edit: '<path d="m16 3 5 5-12 12-6 1 1-6Z M13 6l5 5"/>',
    plus: '<path d="M12 5v14M5 12h14"/>',
    arrow: '<path d="M5 12h14m-6-6 6 6-6 6"/>',
    up: '<path d="m6 12 6-6 6 6M12 6v14"/>',
    down: '<path d="m6 12 6 6 6-6M12 4v14"/>',
    logout: '<path d="M9 4H4v16h5m5-13 5 5-5 5M9 12h11"/>',
    layers: '<path d="m12 3 10 6-10 6L2 9Zm-9 11 9 5 9-5M3 18l9 5 9-5"/>',
    back: '<path d="M19 12H5m6-6-6 6 6 6"/>',
    send: '<path d="m22 2-7 20-4-9-9-4Zm0 0L11 13"/>',
    clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    trash: '<path d="M3 6h18M9 6V3h6v3M5 6l1 15h12l1-15M10 10v7m4-7v7"/>',
};

export function icon(name) {
    return `<svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths[name] || paths.chat}</svg>`;
}

export function hydrateIcons(root = document) {
    root.querySelectorAll('[data-icon]').forEach(node => { node.innerHTML = icon(node.dataset.icon); });
}

export function escapeHTML(value = '') {
    return String(value ?? '').replace(/[&<>"']/g, character => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character]);
}

export function initials(name = '') {
    return String(name).split(/[\s_.-]+/).filter(Boolean).map(part => part[0]).slice(0, 2).join('').toUpperCase() || '?';
}

export function dateLabel(value) {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? 'Just now' : date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

export function timeLabel(value) {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? '' : date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
}

export async function api(path, options = {}) {
    const headers = new Headers(options.headers);
    headers.set('Accept', 'application/json');
    const response = await fetch(path, { ...options, headers });
    if (!response.ok) {
        const error = new Error((await response.text()).trim() || 'Something went wrong. Please try again.');
        error.status = response.status;
        throw error;
    }
    if (response.status === 204) return null;
    const text = await response.text();
    return text ? JSON.parse(text) : null;
}

export function emptyState(title, detail, symbol = 'chat') {
    return `<div class="empty-state">${icon(symbol)}<h3>${escapeHTML(title)}</h3><p>${escapeHTML(detail)}</p></div>`;
}

export function showToast(message) {
    const node = document.createElement('div');
    node.className = 'notification show';
    node.textContent = message;
    document.getElementById('notification-container').append(node);
    setTimeout(() => node.remove(), 5000);
}
