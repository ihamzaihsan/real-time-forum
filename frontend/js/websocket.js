import { renderComments } from './comments.js';

    export class WebSocketClient {
        addMessageHandler(type, handler) {
            this.messageHandlers.set(type, handler);
        }

        constructor() {
            this.socket = null;
            this.messageHandlers = new Map();
            this.messageHistory = new Map();
            this.currentChatUser = null;
            this.onlineUsers = new Map();

            // Add this to your WebSocketClient class constructor
            this.addMessageHandler('new_comment', (content) => {
                const commentsList = document.getElementById('commentsList');
                if (commentsList) {
                    const currentPostId = window.location.pathname.split('/')[2];
                    if (currentPostId == content.post_id) {
                        renderComments([content]);
                    }
                }
            });
            
            this.addMessageHandler('typing_status', (content) => {
                const typingIndicator = document.getElementById('typingIndicator');
                if (typingIndicator) {
                    if (content.isTyping) {
                        typingIndicator.style.display = 'inline-block';
                        typingIndicator.textContent = `${content.username} is typing...`;
                    } else {
                        typingIndicator.style.display = 'none';
                        typingIndicator.textContent = '';
                    }
                                    }
            });
    
        }
        connect() {
        console.log('Attempting WebSocket connection...');
        const sessionToken = localStorage.getItem('sessionToken');

        if (!sessionToken) {
            console.error('No session token found, skipping WebSocket connection');
            return;
        }

        this.socket = new WebSocket(`ws://localhost:8080/ws?token=${sessionToken}`);
        this.socket.onopen = () => {
            console.log('WebSocket connected successfully');
        };

        this.socket.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                const handler = this.messageHandlers.get(message.type);
                if (handler) {
                    handler(message.content);
                }
            } catch (error) {
                console.error('Error handling WebSocket message:', error);
            }
        };

        this.socket.onclose = (event) => {
            console.warn('WebSocket connection closed:', event.reason || 'Unknown reason');
            // Attempt reconnect if the closure was not clean
            if (event.code !== 1000) {
                setTimeout(() => this.connect(), 5000);
            }
        };

        this.socket.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

}


