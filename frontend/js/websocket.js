import { initMessageHandlers } from './message.js';
import { loadMessages } from './chat.js';
import { renderContent } from './router.js';
import { renderComments } from './comments.js';

    export class WebSocketClient {
        constructor() {
            this.socket = null;
            this.messageHandlers = new Map();
            this.messageHistory = new Map();
            this.currentChatUser = null;
            this.onlineUsers = new Map();

            // Add default handlers right in the constructor
            this.addMessageHandler('users_list', (content) => {
                this.updateUsersList(content);
            });

            this.addMessageHandler('pong', (content) => {
                console.log('Pong received:', content);
            });

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

            this.addMessageHandler('private_message', (content) => {
                // Show notification if user is not in chat page
                if (!window.location.pathname.includes('/chat')) {
                    this.showNotification(content);
                }
            });
        }

        showNotification(message) {
            // Check if browser supports notifications
            if (!("Notification" in window)) return;

            // Request permission if needed
            if (Notification.permission !== "granted") {
                Notification.requestPermission();
            }

            if (Notification.permission === "granted") {
                const notification = new Notification("New Message", {
                    body: `${message.sender_name}: ${message.message}`,
                    icon: "/path/to/icon.png"  // Add your notification icon
                });

                // Click notification to open chat
                notification.onclick = () => {
                    window.focus();
                    window.history.pushState({}, '', '/chat');
                    renderContent('/chat');
                };
            }
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
                } else {
                    console.warn('No handler registered for message type:', message.type);
                }
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
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

    addMessageHandler(type, handler) {
        this.messageHandlers.set(type, handler);
    }

    sendMessage(type, content) {
        console.log('Sending message:', { type, content });
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify({ type, content }));
            console.log('Message sent successfully');
        } else {
            console.error('WebSocket not ready. State:', this.socket?.readyState);
        }
    }

    
    
    sendPrivateMessage(receiverId, content) {
        const timestamp = new Date().toISOString();
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.sendMessage('private_message', {
                receiver_id: receiverId,
                message: content,
                username: localStorage.getItem('username'),  
                timestamp: timestamp,
            });
        }
    }
    
    

    updateUsersList(users) {
        const usersList = document.getElementById('onlineUsers');
        if (!usersList) return;
    
        usersList.innerHTML = '';
        users.forEach((user) => {
            const userElement = document.createElement('div');
            userElement.className = `user-item ${user.isOnline ? 'online' : 'offline'}`;
            userElement.setAttribute('data-userid', user.id);
            userElement.innerHTML = `
                <span class="user-status"></span>
                <span class="user-name">${user.username}</span>
            `;
            
            if (user.isOnline) {
                userElement.addEventListener('click', () => {
                    const messageHistory = document.getElementById('messageHistory');
                    if (messageHistory) {
                        messageHistory.innerHTML = '';
                    }
                    
                    document.querySelectorAll('.user-item').forEach(el => el.classList.remove('active'));
                    userElement.classList.add('active');
                    
                    const selectedUserName = document.getElementById('selectedUserName');
                    if (selectedUserName) {
                        selectedUserName.textContent = user.username;
                    }
        
                    this.currentChatUser = user.id;
                    loadMessages(this, user.id);  
                    
                    const messageForm = document.getElementById('messageForm');
                    if (messageForm) {
                        messageForm.style.display = 'flex';
                    }

                    // Navigate to chat page
                    window.history.pushState({}, '', '/chat');
                    renderContent('/chat');
                });
            }
            usersList.appendChild(userElement);
        });
    }
    
}


