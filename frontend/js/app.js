// app.js

// Import the router logic from router.js
import { initRouter } from './router.js';
import { WebSocketClient } from './websocket.js';
import { initMessageHandlers } from './message.js';
import { initNotifications } from './notifications.js';
import { loadProfileData } from './profile.js';

  // Initialize the router when the DOM content is loaded
    document.addEventListener('DOMContentLoaded', () => {
        const wsClient = new WebSocketClient();
        initRouter();
        initMessageHandlers(wsClient);
        initNotifications();
        initSessionValidator();
        wsClient.connect(); 
        
        // Create navigation if it doesn't exist
        let nav = document.querySelector('nav');
        if (!nav) {
            nav = document.createElement('nav');
            document.body.insertBefore(nav, document.body.firstChild);
        }


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

    function initSessionValidator() {
        setInterval(async () => {
            if (window.location.pathname === '/register' || window.location.pathname === '/login') {
                return;
            }
            const response = await fetch('/check-auth');
            if (!response.ok) {
                localStorage.clear();
                window.location.href = '/login';
            }
        }, 30000); 
    }