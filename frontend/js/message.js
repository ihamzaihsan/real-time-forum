import { updateUsersList, appendMessage } from './chat.js';
import { renderComments, loadComments } from './comments.js';
import { showToast } from './ui.js';
import { refreshFeed } from './router.js';

export function initMessageHandlers(wsClient) {
    wsClient.addMessageHandler('new_post', () => refreshFeed());
    wsClient.addMessageHandler('users_list', content => {
        wsClient.users = Array.isArray(content) ? content : [];
        wsClient.onlineUsers.clear();
        wsClient.users.forEach(user => wsClient.onlineUsers.set(Number(user.id), user.isOnline));
        updateUsersList(wsClient, wsClient.users);
    });
    wsClient.addMessageHandler('private_message', content => {
        if (window.location.pathname === '/chat' && Number(content.sender_id) === wsClient.currentChatUser) {
            appendMessage(content);
        } else {
            const badge = document.getElementById('message-badge');
            badge.textContent = Number(badge.textContent || 0) + 1;
            badge.hidden = false;
            showToast(`${content.sender}: ${content.message}`);
        }
    });
    wsClient.addMessageHandler('typing_status', content => {
        const indicator = document.getElementById('typingIndicator');
        if (indicator && Number(content.user_id) === wsClient.currentChatUser) indicator.textContent = content.isTyping ? `${content.username} is typing…` : '';
    });
    wsClient.addMessageHandler('new_comment', async content => {
        const postId = window.location.pathname.match(/^\/post\/(\d+)$/)?.[1];
        if (postId && Number(postId) === Number(content.post_id)) {
            const comments = await loadComments(postId);
            if (window.location.pathname === `/post/${postId}`) renderComments(comments);
        }
    });
}
