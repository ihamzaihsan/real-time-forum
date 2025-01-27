// Route definitions: Map URL paths to corresponding content functions
const routes = {
    '/': homeContent,
    '/register': registerContent,
    '/logout': logoutContent,
    '/login': loginContent,
    '/chat': chatContent
};
function homeContent() {
    document.getElementById('content').innerHTML = `
        <div class="home-container">
            <h1>Welcome to Real Time Forum</h1>
            <p>This is a place where you can connect with others in real-time!</p>
        </div>
    `;
}


import { handleRegisterSubmit } from './register.js';
import { handleLoginSubmit } from './login.js';

function chatContent() {
    const container = document.getElementById('content');
    container.innerHTML = `
        <div class="chat-container">
            <div class="online-users-sidebar">
                <h3>Users</h3>
                <div class="users-list" id="onlineUsers"></div>
            </div>
            <div class="chat-main">
                <div id="selectedUserName" class="selected-user"></div>
                <div class="chat-messages" id="messageHistory"></div>
                <form id="messageForm" class="chat-input">
                    <input type="text" id="messageInput" placeholder="Type a message...">
                    <button type="submit">Send</button>
                </form>
            </div>
        </div>
    `;

    // Initialize message form handler
    const messageForm = document.getElementById('messageForm');
    if (messageForm) {
        messageForm.addEventListener('submit', (e) => {
            e.preventDefault();
            const messageInput = document.getElementById('messageInput');
            const content = messageInput.value.trim();
            
            if (content && window.wsClient && window.wsClient.currentChatUser) {
                window.wsClient.sendPrivateMessage(window.wsClient.currentChatUser, content);
                messageInput.value = '';
                
                // Add message to UI immediately
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
        });
    }

    // Reconnect WebSocket when entering chat
    if (window.wsClient) {
        window.wsClient.connect();
        
    }
}
// Register page content function
function registerContent() {
    document.getElementById('content').innerHTML = `
        <div class="register-container">
            <h1>Create Account</h1>
            <form id="registerForm" class="register-form">
                <div class="form-group">
                    <input type="text" id="username" placeholder="Username" required>
                </div>
                <div class="form-group">
                    <input type="email" id="email" placeholder="Email" required>
                </div>
                <div class="form-group">
                    <input type="password" id="password" placeholder="Password" required>
                </div>
                <div class="form-group">
                    <input type="text" id="first_name" placeholder="First Name" required>
                </div>
                <div class="form-group">
                    <input type="text" id="last_name" placeholder="Last Name" required>
                </div>
                <div class="form-group">
                    <input type="number" id="age" placeholder="Age" required>
                </div>
                <div class="form-group gender-group">
                    <label>Gender:</label>
                    <div class="gender-options">
                        <label class="gender-label">
                            <input type="radio" id="male" name="gender" value="male" required>
                            Male
                        </label>
                        <label class="gender-label">
                            <input type="radio" id="female" name="gender" value="female" required>
                            Female
                        </label>
                    </div>
                </div>
                <button type="submit" class="register-btn">Create Account</button>
            </form>
        </div>
    `;

    document.getElementById('registerForm').addEventListener('submit', handleRegisterSubmit);
}

function loginContent() {
    // Check if user is already logged in
    const sessionToken = localStorage.getItem('sessionToken');
    if (sessionToken) {
        // Redirect to home page if already authenticated
        window.history.pushState({}, '', '/');
        renderContent('/');
        return;
    }

    // Show login form only if not authenticated
    document.getElementById('content').innerHTML = `
        <div class="login-container">
            <h1>Login</h1>
            <form id="loginForm" class="login-form">
                <div class="form-group">
                    <input type="text" id="username" placeholder="Username or Email" required>
                </div>
                <div class="form-group">
                    <input type="password" id="password" placeholder="Password" required>
                </div>
                <button type="submit" class="login-btn">Login</button>
            </form>
        </div>
    `;

    document.getElementById('loginForm').addEventListener('submit', handleLoginSubmit);
}
// Handle navigation when a link is clicked
function handleRoute(event) {
    event.preventDefault();
    const path = event.target.getAttribute('href');
    window.history.pushState({}, '', path);
    renderContent(path);
    
    // Initialize WebSocket connection when navigating to chat
    if (path === '/chat' && window.wsClient) {
        window.wsClient.connect();

    }
}

// Logout content function
async function logoutContent(event) {
    if (event) {
        event.preventDefault(); // Prevent default link navigation
    }

    const sessionToken = localStorage.getItem('sessionToken');

    // Check if session token exists before making the logout request
    if (!sessionToken) {
        console.error('No session token found.');
        alert('You are not logged in.');
        return;
    }

    try {
        const response = await fetch('/logout', {
            method: 'POST',
            headers: {
                'Authorization': sessionToken
            }
        });

        if (response.ok) {
            // Clear session token and update UI
            localStorage.removeItem('sessionToken');
            updateNavigation(); // Update navigation to reflect logged-out state
            window.history.pushState({}, '', '/'); // Redirect to the home page
            renderContent('/'); // Render home page content
        } else {
            console.error('Logout failed with status:', response.status);
            alert('Failed to log out. Please try again.');
        }
    } catch (error) {
        console.error('Logout failed:', error);
        alert('An error occurred during logout. Please try again later.');
    }
}

// Render content based on the current route
function renderContent(path) {
    const render = routes[path] || routes['/']; // Default to home if no matching route
    render(); // Call the function for the current route
}

// Update navigation links visibility based on login state
function updateNavigation() {
    const sessionToken = localStorage.getItem('sessionToken');
    const registerLink = document.querySelector('a[href="/register"]');
    const logoutLink = document.querySelector('a[href="/logout"]');
    const loginLink = document.querySelector('a[href="/login"]');
    const chatLink = document.querySelector('a[href="/chat"]');
    
    if (sessionToken) {
        if (registerLink) registerLink.style.display = 'none';
        if (logoutLink) logoutLink.style.display = 'block';
        if (loginLink) loginLink.style.display = 'none';
        if (chatLink) chatLink.style.display = 'block';
    } else {
        if (registerLink) registerLink.style.display = 'block';
        if (logoutLink) logoutLink.style.display = 'none';
        if (loginLink) loginLink.style.display = 'block';
        if (chatLink) chatLink.style.display = 'none';
    }
}

// Initialize router: Set up event listeners for navigation links
function initRouter() {
    document.querySelectorAll('a').forEach(link => {
        const path = link.getAttribute('href');

        if (path === '/logout') {
            // Attach the logout handler for the logout link
            link.addEventListener('click', logoutContent);
        } else {
            // Attach the route handler for all other links
            link.addEventListener('click', handleRoute);
        }
    });

    // Listen for back/forward navigation using browser history
    window.onpopstate = () => {
        renderContent(window.location.pathname);
    };

    // Handle initial page load (render the content for the current URL)
    renderContent(window.location.pathname);

    // Update navigation UI on URL changes
    updateNavigation();
    window.addEventListener('popstate', updateNavigation);
}
function throttle(func, limit) {
    let inThrottle;
    return function(...args) {
        if (!inThrottle) {
            func.apply(this, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    }
}
// Display errors for form validation
function displayErrors(errors) {
    const form = document.querySelector('form');
    if (!form) return; // Exit if no form is found
    
    // Remove any existing error messages
    const existingErrors = document.querySelector('.error-messages');
    if (existingErrors) {
        existingErrors.remove();
    }

    // Create and insert new error messages
    const errorDiv = document.createElement('div');
    errorDiv.className = 'error-messages';
    errorDiv.innerHTML = errors.map(error => `<p>${error}</p>`).join('');
    form.insertBefore(errorDiv, form.firstChild);
}

// Export initRouter and displayErrors for use in other modules
export { initRouter, displayErrors };


