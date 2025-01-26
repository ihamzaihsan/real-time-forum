export function initMessageHandlers(wsClient) {
    
    wsClient.addMessageHandler('users_list', (content) => {
        wsClient.updateUsersList(content);
    });

    // Add handler for pong messages
    wsClient.addMessageHandler('pong', (content) => {
        console.log('Received pong response:', content);
    });
    wsClient.addMessageHandler('private_message', (content) => {
        if (wsClient.currentChatUser) {
            const messageContainer = document.getElementById('messageHistory');
            const currentUserId = parseInt(localStorage.getItem('userId'));
            const messageElement = document.createElement('div');
            const isCurrentUser = Number(message.sender_id) === currentUserId;

            
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
    });

    const messageForm = document.getElementById('messageForm');
    if (messageForm) {
        messageForm.addEventListener('submit', (e) => {
            e.preventDefault();
            const messageInput = document.getElementById('messageInput');
            const content = messageInput.value.trim();
            
            if (content && wsClient.currentChatUser) {
                wsClient.sendPrivateMessage(wsClient.currentChatUser, content);
                
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
                
                messageInput.value = '';
            }
        });
    }
}
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
