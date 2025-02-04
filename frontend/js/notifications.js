export function initNotifications() {
    if (!("Notification" in window)) {
        return;
    }
    
    if (Notification.permission !== "granted") {
        Notification.requestPermission();
    }
}

export function showWindowNotification(content) {
    const container = document.getElementById('notification-container');
    const notification = document.createElement('div');
    notification.className = 'notification';
    notification.innerHTML = `
        <div class="notification-content">
            <strong>Message from ${content.sender}</strong>
            <p>${content.message}</p>
        </div>
    `;
    
    container.appendChild(notification);

    setTimeout(() => notification.classList.add('show'), 100);

    setTimeout(() => {
        notification.classList.remove('show');
        setTimeout(() => notification.remove(), 300);
    }, 5000);
}
