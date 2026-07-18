import { validateSession } from './app.js';
import { refreshNotifications, refreshActivityPage } from './activity.js';
import { updateUsersList, appendMessage, refreshCurrentChat } from './chat.js';
import { refreshComments } from './comments.js';
import { renderModeration } from './moderation.js';
import { showToast } from './ui.js';
import { refreshFeed, refreshCurrentPage, navigate, refreshTopics } from './router.js';

export function initMessageHandlers(wsClient) {
    wsClient.addMessageHandler('connected', async () => {
        await validateSession();
        await refreshCurrentPage();
        await refreshNotifications();
        await refreshActivityPage();
        await refreshCurrentChat(wsClient);
    });
    wsClient.addMessageHandler('account_changed', () => window.location.reload());
    wsClient.addMessageHandler('topics_changed', () => refreshTopics());
    wsClient.addMessageHandler('community_changed', async () => {
        if (window.location.pathname === '/moderation') renderModeration();
    });
    wsClient.addMessageHandler('content_changed', async () => {
        await refreshCurrentPage();
        await refreshActivityPage();
        await refreshNotifications();
    });
    wsClient.addMessageHandler('notification_changed', () => refreshNotifications());
    wsClient.addMessageHandler('new_post', () => refreshFeed());
    wsClient.addMessageHandler('reaction_updated', (content) => {
        const selector =
            content.kind === 'post'
                ? `[data-post-id="${Number(content.id)}"]`
                : `[data-comment-id="${Number(content.id)}"]`;
        document.querySelectorAll(selector).forEach((node) => {
            node.querySelector('.likes-count').textContent = content.likes;
            node.querySelector('.dislikes-count').textContent = content.dislikes;
        });
        if (content.kind === 'post') refreshFeed();
    });
    wsClient.addMessageHandler('post_deleted', (content) => {
        refreshActivityPage();
        refreshNotifications();
        if (window.location.pathname === `/post/${content.post_id}`) {
            showToast('This discussion has been removed.');
            navigate('/');
        } else refreshFeed();
    });
    wsClient.addMessageHandler('users_list', (content) => {
        wsClient.users = Array.isArray(content) ? content : [];
        wsClient.onlineUsers.clear();
        wsClient.users.forEach((user) => wsClient.onlineUsers.set(Number(user.id), user.isOnline));
        updateUsersList(wsClient, wsClient.users);
    });
    wsClient.addMessageHandler('message_sent', (content) => {
        if (window.location.pathname === '/chat' && Number(content.receiver_id) === wsClient.currentChatUser)
            appendMessage(content);
    });
    wsClient.addMessageHandler('private_message', (content) => {
        if (window.location.pathname === '/chat' && Number(content.sender_id) === wsClient.currentChatUser) {
            appendMessage(content);
        } else {
            const badge = document.getElementById('message-badge');
            badge.textContent = Number(badge.textContent || 0) + 1;
            badge.hidden = false;
            showToast(`${content.sender}: ${content.content}`);
        }
    });
    wsClient.addMessageHandler('error', (content) => showToast(content.message));
    wsClient.addMessageHandler('typing_status', (content) => {
        const indicator = document.getElementById('typingIndicator');
        if (indicator && Number(content.user_id) === wsClient.currentChatUser) {
            indicator.textContent = content.isTyping ? `${content.username} is typing…` : '';
            clearTimeout(wsClient.typingTimer);
            wsClient.typingTimer = setTimeout(() => {
                if (indicator.isConnected) indicator.textContent = '';
            }, 3000);
        }
    });
    wsClient.addMessageHandler('new_comment', async (content) => {
        const postId = window.location.pathname.match(/^\/post\/(\d+)$/)?.[1];
        if (postId && Number(postId) === Number(content.post_id)) {
            await refreshComments(postId);
        }
    });
}
