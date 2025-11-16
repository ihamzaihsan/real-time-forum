import { initRouter } from './router.js';
import { WebSocketClient } from './websocket.js';
import { initMessageHandlers } from './message.js';
import { hydrateIcons } from './ui.js';
import { updateUsersList } from './chat.js';
import { initTheme } from './theme.js';

async function validateSession() {
    if (!localStorage.getItem('sessionToken')) return;
    try {
        const response = await fetch('/check-auth');
        if (response.status === 401) {
            window.wsClient.disconnect();
            ['sessionToken', 'username', 'userId'].forEach(key => localStorage.removeItem(key));
            window.location.replace('/login');
        }
    } catch (error) { console.warn('Session check unavailable:', error); }
}

document.addEventListener('DOMContentLoaded', async () => {
    initTheme();
    window.wsClient = new WebSocketClient();
    initMessageHandlers(window.wsClient);
    hydrateIcons();
    initRouter();
    window.addEventListener('connectionchange', () => {
        if (localStorage.getItem('sessionToken') && window.wsClient.users.length) updateUsersList(window.wsClient, window.wsClient.users);
    });
    await validateSession();
    window.wsClient.connect();
    setInterval(validateSession, 30000);
});
