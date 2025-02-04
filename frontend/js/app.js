// app.js

// Import the router logic from router.js
import { initRouter } from './router.js';
import { WebSocketClient } from './websocket.js';
import { initMessageHandlers } from './message.js';
import { initNotifications } from './notifications.js';

// Initialize the router when the DOM content is loaded
document.addEventListener('DOMContentLoaded', () => {
    const wsClient = new WebSocketClient();
    initRouter();
    initMessageHandlers(wsClient);
    initNotifications();
    wsClient.connect(); 
    

    wsClient.addMessageHandler('chat', (content) => {
        console.log('Received chat message:', content);
        const chatMessages = document.getElementById('chatMessages');
        if (chatMessages) {
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message';
            messageDiv.innerHTML = `
                <span class="sender">${content.sender}:</span>
                <span class="text">${content.message}</span>
            `;
            chatMessages.appendChild(messageDiv);
            chatMessages.scrollTop = chatMessages.scrollHeight;
        }
    });

    window.wsClient = wsClient;
});