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
    }

    addMessageHandler(type, handler) { this.messageHandlers.set(type, handler); }

    setStatus(text, connected = false) {
        const status = document.getElementById('connectionStatus');
        if (status) { status.textContent = text; status.classList.toggle('connected', connected); }
        window.dispatchEvent(new CustomEvent('connectionchange'));
    }

    connect() {
        const token = localStorage.getItem('sessionToken');
        if (!token) return;
        if (this.socket && [WebSocket.OPEN, WebSocket.CONNECTING].includes(this.socket.readyState)) return;
        this.stopped = false;
        clearTimeout(this.reconnectTimer);
        const url = new URL('/ws', window.location.href);
        url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        url.searchParams.set('token', token);
        const socket = new WebSocket(url.href);
        this.socket = socket;
        this.setStatus('Connecting');
        socket.onopen = () => this.setStatus('Live', true);
        socket.onmessage = async event => {
            try {
                const message = JSON.parse(event.data);
                await this.messageHandlers.get(message.type)?.(message.content);
            } catch (error) { console.error('Could not handle a live update:', error); }
        };
        socket.onclose = () => {
            if (this.socket !== socket) return;
            this.setStatus('Reconnecting');
            if (!this.stopped && localStorage.getItem('sessionToken')) this.reconnectTimer = setTimeout(() => this.connect(), 5000);
        };
        socket.onerror = () => this.setStatus('Connection interrupted');
    }

    disconnect() {
        this.stopped = true;
        clearTimeout(this.reconnectTimer);
        const socket = this.socket;
        this.socket = null;
        if (socket) socket.close(1000, 'Signed out');
        this.onlineUsers.clear();
        this.users = [];
        this.currentChatUser = null;
    }
}
