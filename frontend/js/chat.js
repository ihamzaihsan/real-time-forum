import { WebSocketClient } from './websocket.js';

 // Add this variable to track if we're currently loading messages
    let isLoadingMessages = false;

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
export function initializeScrollListener() {
    const messageHistory = document.getElementById('messageHistory');
    const throttledLoadMore = throttle(() => {
        // Check if we're near the top of the scroll
        if (messageHistory.scrollTop < 100) {
            loadMoreMessages();
        }
    }, 500); // Throttle to 500ms

    messageHistory.addEventListener('scroll', throttledLoadMore);
}

export function loadMoreMessages() {
        if (isLoadingMessages) return;
        
        const currentOffset = document.querySelectorAll('.message').length;
        const messageHistory = document.getElementById('messageHistory');
        const oldScrollHeight = messageHistory.scrollHeight;
        
        isLoadingMessages = true;
        
        fetch(`/messages/${window.wsClient.currentChatUser}?offset=${currentOffset}&limit=10`, {
            headers: {
                'Authorization': localStorage.getItem('sessionToken')
            }
        })
        .then(response => response.json())
        .then(messages => {
            if (messages.length > 0) {
                const oldMessages = messageHistory.innerHTML;
                messages.forEach(message => {
                    const currentUserId = parseInt(localStorage.getItem('userId'));
                    const isCurrentUser = parseInt(message.sender_id) === currentUserId;
                    const messageElement = `
                        <div class="message ${isCurrentUser ? 'sent' : 'received'}">
                            <div class="message-content">
                                <span class="message-text">${message.content}</span>
                                <span class="message-time">${new Date(message.created_at).toLocaleTimeString()}</span>
                            </div>
                        </div>
                    `;
                    messageHistory.innerHTML = messageElement + messageHistory.innerHTML;
                });
                
                // Maintain scroll position
                messageHistory.scrollTop = messageHistory.scrollHeight - oldScrollHeight;
            }
        })
        .finally(() => {
            isLoadingMessages = false;
        });
    }

    export function loadMessages(wsClient, userId, offset = 0) {
        fetch(`/messages/${userId}?offset=${offset}&limit=10`, {
            headers: {
                'Authorization': localStorage.getItem('sessionToken')
            }
        })
        .then((response) => response.json())
        .then((messages) => {
            const messageHistory = document.getElementById('messageHistory');
            displayMessages(messages);
        })
        .catch((error) => console.error('Error loading messages:', error));
    }

    // Add this separate function to handle loading more messages
    export function handleLoadMore(userId) {
        const messageHistory = document.getElementById('messageHistory');
        const currentOffset = document.querySelectorAll('.message').length;
        const oldHeight = messageHistory.scrollHeight;
    
        fetch(`/messages/${userId}?offset=${currentOffset}&limit=10`, {
            headers: {
                'Authorization': localStorage.getItem('sessionToken')
            }
        })
        .then(response => response.json())
        .then(oldMessages => {
            oldMessages.forEach(msg => {
                const isCurrentUser = parseInt(msg.sender_id) === parseInt(localStorage.getItem('userId'));
                const messageElement = document.createElement('div');
                messageElement.className = `message ${isCurrentUser ? 'sent' : 'received'}`;
                messageElement.innerHTML = `
                    <div class="message-content">
                        <span class="message-text">${msg.content}</span>
                        <span class="message-time">${new Date(msg.created_at).toLocaleTimeString()}</span>
                    </div>
                `;
                messageHistory.insertBefore(messageElement, messageHistory.firstChild);
            });
            messageHistory.scrollTop = messageHistory.scrollHeight - oldHeight;
        });
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
