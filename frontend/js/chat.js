import { api, escapeHTML as esc, initials, emptyState, timeLabel, showToast } from './ui.js';
import { navigate } from './router.js';

let chatVersion = 0;
let loading = false;
let historyOffset = 0;

function createMessageElement(message) {
    const node = document.createElement('div');
    const sent = Number(message.sender_id) === Number(localStorage.getItem('userId'));
    node.className = `message ${sent ? 'sent' : 'received'}`;
    if (message.id) node.dataset.messageId = message.id;
    const text = document.createElement('span');
    text.className = 'message-text';
    text.textContent = message.content ?? message.message ?? '';
    const time = document.createElement('span');
    time.className = 'message-time';
    time.textContent = timeLabel(message.created_at || message.timestamp || new Date());
    node.append(text, time);
    return node;
}

export function appendMessage(message) {
    const history = document.getElementById('messageHistory');
    if (!history) return;
    history.querySelector('.empty-state')?.remove();
    history.append(createMessageElement(message));
    historyOffset++;
    history.scrollTop = history.scrollHeight;
}

async function loadMessages(wsClient, userId, older = false) {
    if (loading && older) return;
    const version = chatVersion;
    const history = document.getElementById('messageHistory');
    if (!history) return;
    loading = true;
    if (!older) {
        historyOffset = 0;
        history.innerHTML = '<div class="empty-state"><p>Opening your conversation…</p></div>';
    }
    const loadButton = history.querySelector('.chat-load-more');
    if (loadButton) loadButton.disabled = true;
    const oldHeight = history.scrollHeight;
    try {
        const messages = await api(`/messages/${userId}?offset=${historyOffset}&limit=10`);
        if (version !== chatVersion || Number(wsClient.currentChatUser) !== Number(userId)) return;
        const list = Array.isArray(messages) ? messages : [];
        historyOffset += list.length;
        const hasOlderMessages = list.length === 10;
        if (!older) history.innerHTML = '';
        history.querySelector('.chat-load-more')?.remove();
        const fragment = document.createDocumentFragment();
        list.slice().reverse().forEach(message => {
            if (!history.querySelector(`[data-message-id="${Number(message.id)}"]`)) fragment.append(createMessageElement(message));
        });
        if (older) history.prepend(fragment);
        else history.append(fragment);
        if (!history.querySelector('.message')) history.innerHTML = emptyState('This is the start of something.', 'Say hello. Your conversation begins here.');
        if (hasOlderMessages) {
            const button = document.createElement('button');
            button.type = 'button';
            button.className = 'button button-quiet chat-load-more';
            button.textContent = 'Load earlier messages';
            button.addEventListener('click', () => loadMessages(wsClient, userId, true));
            history.prepend(button);
        }
        history.scrollTop = older ? history.scrollHeight - oldHeight : history.scrollHeight;
    } catch (error) {
        if (version !== chatVersion) return;
        if (!older) history.innerHTML = emptyState('Couldn’t load the conversation.', error.message);
        else showToast(error.message);
    } finally {
        if (version === chatVersion) { loading = false; if (loadButton) loadButton.disabled = false; }
    }
}

function updateChatHeader(wsClient) {
    if (!wsClient?.currentChatUser) return;
    const user = (wsClient.users || []).find(person => Number(person.id) === Number(wsClient.currentChatUser));
    const name = user?.username || wsClient.currentChatName || 'Community member';
    const heading = document.getElementById('selectedUserName');
    if (!heading) return;
    heading.textContent = name;
    document.getElementById('chatAvatar').textContent = initials(name);
    document.getElementById('chatStatus').textContent = user?.isOnline ? 'Here now · Say hello' : 'Away · You can still read your conversation';
    const form = document.getElementById('messageForm');
    form.hidden = false;
    const available = Boolean(user?.isOnline && wsClient.socket?.readyState === WebSocket.OPEN);
    form.querySelector('button').disabled = !available;
    const input = document.getElementById('messageInput');
    input.disabled = !available;
    input.placeholder = available ? 'Say something kind…' : 'Messaging is available when you’re both online';
}

export function updateUsersList(wsClient, users) {
    const list = document.getElementById('onlineUsers');
    if (!list) return;
    const sorted = (Array.isArray(users) ? users : []).slice().sort((a, b) => {
        if (a.lastMessageTime || b.lastMessageTime) {
            const difference = (Date.parse(b.lastMessageTime) || 0) - (Date.parse(a.lastMessageTime) || 0);
            if (difference) return difference;
        }
        return a.username.localeCompare(b.username, undefined, { sensitivity: 'base' });
    });
    list.innerHTML = sorted.length ? sorted.map(user => `<button type="button" class="user-item ${user.isOnline ? 'online' : 'offline'} ${Number(user.id) === Number(wsClient.currentChatUser) ? 'active' : ''}" data-userid="${Number(user.id)}" aria-label="Open conversation with ${esc(user.username)}"><span class="avatar tone-${Number(user.id) % 4}">${esc(initials(user.username))}<span class="user-status"></span></span><span><span class="user-name">${esc(user.username)}</span><span class="user-presence">${user.isOnline ? 'Here now' : 'Away'}</span></span><span class="user-chevron" aria-hidden="true">↗</span></button>`).join('') : '<p class="muted">You’re the first one here. Invite someone to join the conversation.</p>';
    list.querySelectorAll('[data-userid]').forEach(button => button.addEventListener('click', () => {
        const user = sorted.find(person => Number(person.id) === Number(button.dataset.userid));
        wsClient.currentChatUser = Number(user.id);
        wsClient.currentChatName = user.username;
        if (window.location.pathname !== '/chat') navigate('/chat');
        else {
            chatVersion++;
            loading = false;
            document.getElementById('typingIndicator').textContent = '';
            updateChatHeader(wsClient);
            loadMessages(wsClient, user.id);
            list.querySelectorAll('.user-item').forEach(node => node.classList.toggle('active', node === button));
        }
    }));
    updateChatHeader(wsClient);
}

export function initializeChat(wsClient) {
    chatVersion++;
    loading = false;
    if (!wsClient) return;
    if (wsClient.currentChatUser) {
        updateChatHeader(wsClient);
        loadMessages(wsClient, wsClient.currentChatUser);
    }
    const form = document.getElementById('messageForm');
    const input = document.getElementById('messageInput');
    form.addEventListener('submit', event => {
        event.preventDefault();
        const text = input.value.trim();
        if (!text || !wsClient.currentChatUser) return;
        if (wsClient.socket?.readyState !== WebSocket.OPEN || !wsClient.onlineUsers.get(wsClient.currentChatUser)) {
            showToast('Reconnect and wait for your friend to come online.');
            return;
        }
        wsClient.socket.send(JSON.stringify({ type: 'private_message', content: { receiver_id: wsClient.currentChatUser, message: text } }));
        appendMessage({ sender_id: Number(localStorage.getItem('userId')), content: text, timestamp: new Date() });
        input.value = '';
        sendTyping(false);
    });
    let typingTimer;
    const sendTyping = typing => {
        if (wsClient.socket?.readyState === WebSocket.OPEN && wsClient.currentChatUser) wsClient.socket.send(JSON.stringify({ type: 'typing_status', content: { receiver_id: wsClient.currentChatUser, isTyping: typing } }));
    };
    input.addEventListener('input', () => {
        sendTyping(Boolean(input.value.trim()));
        clearTimeout(typingTimer);
        const recipient = wsClient.currentChatUser;
        typingTimer = setTimeout(() => {
            if (window.location.pathname === '/chat' && wsClient.currentChatUser === recipient) sendTyping(false);
        }, 1200);
    });
}
