export class WebSocketClient {
    constructor() {
        this.socket = null;
        this.messageHandlers = new Map();
        this.currentChatUser = null;
        this.currentChatName = '';
        this.onlineUsers = new Map();
        this.users = [];
        this.reconnectTimer = null;
        this.stopped = false;
        this.pending = new Map();
        this.retry = null;
    }

    addMessageHandler(type, handler) { this.messageHandlers.set(type, handler); }

    setStatus(text, connected = false) {
        const status = document.getElementById('connectionStatus');
        if (status) { status.textContent = text; status.classList.toggle('connected', connected); }
        window.dispatchEvent(new CustomEvent('connectionchange'));
    }

    finishPending(id, error, message) {
        const request = this.pending.get(id);
        if (!request) return;
        clearTimeout(request.timer);
        this.pending.delete(id);
        if (error) request.reject(new Error(error));
        else { if (this.retry?.id === id) this.retry = null; request.resolve(message); }
    }

    sendPrivateMessage(receiverId, text) {
        if (this.socket?.readyState !== WebSocket.OPEN) return Promise.reject(new Error('Reconnect before sending.'));
        if (this.pending.size) return Promise.reject(new Error('Wait for the current message to be confirmed.'));
        const key = JSON.stringify([receiverId, text]);
        const id = this.retry?.key === key ? this.retry.id : crypto.randomUUID();
        this.retry = { key, id };
        return new Promise((resolve, reject) => {
            const timer = setTimeout(() => this.finishPending(id, 'Message not confirmed. Check your history before retrying.'), 12000);
            this.pending.set(id, { resolve, reject, timer });
            try {
                this.socket.send(JSON.stringify({ type: 'private_message', content: { receiver_id: receiverId, message: text, client_id: id } }));
            } catch {
                this.finishPending(id, 'Connection interrupted. Your draft is still here.');
            }
        });
    }

    connect() {
        if (this.socket && [WebSocket.OPEN, WebSocket.CONNECTING].includes(this.socket.readyState)) return;
        this.stopped = false;
        clearTimeout(this.reconnectTimer);
        const url = new URL('/ws', window.location.href);
        url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        // The browser sends the HttpOnly session cookie during the handshake.
        const socket = new WebSocket(url.href);
        this.socket = socket;
        this.setStatus('Connecting');
        socket.onopen = async () => {
            this.setStatus('Live', true);
            try { await this.messageHandlers.get('connected')?.(); }
            catch (error) { console.warn('Could not refresh after reconnect:', error); }
        };
        socket.onmessage = async event => {
            try {
                const message = JSON.parse(event.data);
                if (message.type === 'message_sent') this.finishPending(message.content.client_id, null, message.content);
                if (message.type === 'message_error') this.finishPending(message.content.client_id, message.content.message);
                await this.messageHandlers.get(message.type)?.(message.content);
            } catch (error) { console.error('Could not handle a live update:', error); }
        };
        socket.onclose = () => {
            if (this.socket !== socket) return;
            for (const id of this.pending.keys()) this.finishPending(id, 'Connection interrupted. Check your history before retrying.');
            this.setStatus('Reconnecting');
            if (!this.stopped) this.reconnectTimer = setTimeout(() => this.connect(), 5000);
        };
        socket.onerror = () => this.setStatus('Connection interrupted');
    }

    disconnect() {
        this.stopped = true;
        clearTimeout(this.reconnectTimer);
        for (const id of this.pending.keys()) this.finishPending(id, 'Connection closed. Your draft is still here.');
        const socket = this.socket;
        this.socket = null;
        if (socket) socket.close(1000, 'Disconnected');
        this.onlineUsers.clear();
        this.users = [];
        this.currentChatUser = null;
        this.retry = null;
    }
}
