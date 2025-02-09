import { showWindowNotification } from './notifications.js';
import { updateUsersList } from './chat.js';
import { renderContent } from './router.js';
export function initMessageHandlers(wsClient) {
    wsClient.addMessageHandler('users_list', (content) => {
        // Populate the onlineUsers Map
        content.forEach(user => {
            wsClient.onlineUsers.set(user.id, user.isOnline);
        });
        
        updateUsersList(wsClient, content);
    });

    wsClient.addMessageHandler('private_message', (content) => {
        const messageContainer = document.getElementById('messageHistory');
        // Only display message if it's from the current chat user
        if (wsClient.currentChatUser && 
            messageContainer && 
            (Number(content.sender_id) === wsClient.currentChatUser ||  Number(content.sender_id) === parseInt(localStorage.getItem('userId')))) {
            
            const currentUserId = parseInt(localStorage.getItem('userId'));
            const messageElement = document.createElement('div');
            const isCurrentUser = Number(content.sender_id) === currentUserId;
            
            messageElement.className = `message ${isCurrentUser ? 'sent' : 'received'}`;
            messageElement.innerHTML = `
                <div class="message-content">
                    <span class="message-text">${content.message}</span>
                    <span class="message-time">${new Date().toLocaleTimeString()}</span>
                </div>
            `;
            messageContainer.appendChild(messageElement);
            messageContainer.scrollTop = messageContainer.scrollHeight;
        }
        
        // Show notification for messages from other users when not in their chat
        if (!window.location.pathname.includes('/chat') || 
            Number(content.sender_id) !== wsClient.currentChatUser) {
            const badge = document.getElementById('message-badge');
            if (badge) {
                const currentCount = parseInt(badge.textContent) || 0;
                badge.textContent = currentCount + 1;
                badge.style.display = 'inline';
            }
            showWindowNotification(content);
        }
    });
    const messageForm = document.getElementById('messageForm');
    if (messageForm) {
        messageForm.addEventListener('submit', (e) => {
            e.preventDefault();
            const messageInput = document.getElementById('messageInput');
            const content = messageInput.value.trim();
            
            if (content && wsClient.currentChatUser) {
                const isReceiverOnline = wsClient.onlineUsers.get(Number(wsClient.currentChatUser));
                if (isReceiverOnline) {
                    let messageSent = sendPrivateMessage(wsClient.socket, wsClient.currentChatUser, content);
                    if (messageSent){
                        messageInput.value = '';
                    }
                    const messageHistory = document.getElementById('messageHistory');
                    const messageElement = document.createElement('div');
                    messageElement.className = 'message sent';
                    messageElement.innerHTML = `
                        <div class="message-content">
                            <span class="message-text">${content}</span>
                            <span class="message-time">${new Date().toLocaleTimeString()}</span>
                        </div>
                    `;
                    messageHistory.appendChild(messageElement);
                    messageHistory.scrollTop = messageHistory.scrollHeight;
                }
                messageInput.value = '';
            }
        });
    }}


export function createMessageElement(message) {
    const div = document.createElement('div');
    div.className = 'message';
    div.innerHTML = `
        <span class="sender">${message.sender_id}</span>
        <span class="content">${message.content}</span>
        <span class="timestamp">${new Date(message.created_at).toLocaleTimeString()}</span>
    `;
    return div;
}

export function sendPrivateMessage(socket, receiverId, content) {
    const timestamp = new Date().toISOString();
    
    const isReceiverOnline = window.wsClient.onlineUsers.get(Number(receiverId));
    if (!isReceiverOnline) {
        window.history.pushState({}, '', '/');
        renderContent('/');
        return false;
    }

    if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({
            type: 'private_message',
            content: {
                receiver_id: receiverId,
                message: content,
                username: localStorage.getItem('username'),
                timestamp: timestamp
            }
        }));
        return true;
    }
    return false;
}
