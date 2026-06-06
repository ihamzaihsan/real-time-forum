import { initRouter } from './router.js';
import { WebSocketClient } from './websocket.js';
import { initMessageHandlers } from './message.js';
import { hydrateIcons } from './ui.js';
import { updateUsersList } from './chat.js';
import { initTheme } from './theme.js';

async function validateSession() {
    const previous = localStorage.getItem('userId');
    try {
        const response = await fetch('/check-auth', { headers: { Accept: 'application/json' } });
        if (response.ok) {
            const session = await response.json();
            localStorage.setItem('userId', session.user_id);
            localStorage.setItem('username', session.username);
            localStorage.setItem('isAdmin', String(session.is_admin));
            if (previous && previous !== String(session.user_id)) window.location.reload();
        } else if (response.status === 401) {
            ['sessionToken', 'username', 'userId', 'isAdmin'].forEach(key => localStorage.removeItem(key));
            if (previous) { window.wsClient.disconnect(); window.location.replace('/login'); }
        }
    } catch (error) { console.warn('Session check unavailable:', error); }
}

document.addEventListener('DOMContentLoaded', async () => {
    initTheme();
    localStorage.removeItem('sessionToken'); // Remove tokens left by the older frontend.
    window.wsClient = new WebSocketClient();
    initMessageHandlers(window.wsClient);
    hydrateIcons();
    await validateSession();
    initRouter();
    window.addEventListener('connectionchange', () => {
        if (localStorage.getItem('userId') && window.wsClient.users.length) updateUsersList(window.wsClient, window.wsClient.users);
    });
    window.addEventListener('storage', event => {
        if (event.key === 'userId' || event.key === null) window.location.reload();
    });
    window.wsClient.connect();
    setInterval(validateSession, 30000);
});
