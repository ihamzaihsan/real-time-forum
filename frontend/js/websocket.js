import { initMessageHandlers } from './message.js';
import { loadMessages } from './chat.js';
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
        // Check if the receiver is online before sending
        const usersList = document.getElementById('onlineUsers');
        const userElement = usersList.querySelector(`[data-userid="${receiverId}"]`);
        const isOffline = userElement?.classList.contains('offline');
    
        if (isOffline) {
            alert('Cannot send message. User is offline');
            return;
        }
    
        const timestamp = new Date().toISOString();
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.sendMessage('private_message', {
                receiver_id: receiverId,
                message: content,
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
                });
            }
            usersList.appendChild(userElement);
        });
    }
    
}


