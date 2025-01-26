import { WebSocketClient } from './websocket.js';

function loadMoreMessages() {
    const currentOffset = document.querySelectorAll('.message').length;
    wsClient.loadMessages(currentChatUser, currentOffset);
}

function throttle(func, limit) {
    let inThrottle;
    return function() {
        const args = arguments;
        const context = this;
        if (!inThrottle) {
            func.apply(context, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    }
}

export function loadMessages(wsClient, userId, offset = 0) {
    fetch(`/messages/${userId}?offset=${offset}&limit=10`, {
        headers: {
            'Authorization': localStorage.getItem('sessionToken')
        }
    })
    .then((response) => response.json())
    .then((messages) => displayMessages(messages))
    .catch((error) => console.error('Error loading messages:', error));
}

export function displayMessages(messages) {
    const messageHistory = document.getElementById('messageHistory');
    if (!messageHistory || !messages) return;

    messageHistory.innerHTML = '';
    const messageArray = Array.isArray(messages) ? messages : [];
    const currentUserId = parseInt(localStorage.getItem('userId'));
    if (!currentUserId) {
        console.error('No user ID found in localStorage');
        return;
    }    
    messageArray.reverse().forEach(message => {
        const messageElement = document.createElement('div');
        const isCurrentUser = parseInt(message.sender_id) === currentUserId;
        console.log('Current user ID:', currentUserId, typeof currentUserId);
        console.log('Message sender ID:', message.sender_id, typeof message.sender_id);
        messageElement.className = `message ${isCurrentUser ? 'sent' : 'received'}`;
        messageElement.innerHTML = `
            <div class="message-content">
                <span class="message-text">${message.content}</span>
                <span class="message-time">${new Date(message.created_at).toLocaleTimeString()}</span>
            </div>
        `;
        messageHistory.appendChild(messageElement);
    });
    messageHistory.scrollTop = messageHistory.scrollHeight;
}


