import { handleRegisterSubmit } from './register.js';
import { handleLoginSubmit } from './login.js';
import { handleCreatePost } from './createPost.js';

// Route definitions: Map URL paths to corresponding content functions
const routes = {
    '/': { component: homeContent, requiresAuth: true },
    '/register': { component: registerContent, requiresAuth: false },
    '/logout': { component: logoutContent, requiresAuth: true },
    '/login': { component: loginContent, requiresAuth: false },
    '/chat': { component: chatContent, requiresAuth: true },
    '/post/:id': { component: singlePostContent, requiresAuth: true  },
    '/create_post': { component: createPostContent, requiresAuth: true }
};

function renderContent(path) {
    const isAuthenticated = checkAuth();
    
    // Handle dynamic routes first
    const postMatch = path.match(/^\/post\/(\d+)$/);
    if (postMatch) {
        if (!isAuthenticated) {
            window.history.pushState({}, '', '/login');
            routes['/login'].component();
            return;
        }
        routes['/post/:id'].component();
        return;
    }

    // Check if path exists in routes
    if (!routes[path]) {
        // Invalid path - redirect based on auth status
        if (isAuthenticated) {
            window.history.pushState({}, '', '/');
            routes['/'].component();
        } else {
            window.history.pushState({}, '', '/login');
            routes['/login'].component();
        }
        return;
    }

    const route = routes[path];

    if (path === '/logout') {
        window.history.pushState({}, '', '/login');
        loginContent();
        return;
    }

    if (!isAuthenticated && route.requiresAuth) {
        // Redirect to login if trying to access protected route
        window.history.pushState({}, '', '/login');
        routes['/login'].component();
        return;
    }

    if (isAuthenticated && path === '/') {
        // If logged in and accessing root, show home
        route.component();
    } else if (!isAuthenticated && path !== '/login' && path !== '/register') {
        // If not logged in and not trying to access login/register, redirect to login
        window.history.pushState({}, '', '/login');
        routes['/login'].component();
    } else {
        // Normal route handling
        route.component();
    }
}function checkAuth() {
    const sessionToken = localStorage.getItem('sessionToken');
    return !!sessionToken;
}

function homeContent() {
    document.getElementById('content').innerHTML = `
        <div class="home-container">
            <div id="posts-container" class="posts-container"></div>
        </div>
    `;

    fetch('/posts')
        .then(response => response.json())
        .then(posts => {
            const postsContainer = document.getElementById('posts-container');
            posts.forEach(post => {
                postsContainer.innerHTML += `
                    <div class="post-card" data-post-id="${post.id}">
                        <h2>${post.title}</h2>
                        <p class="post-meta">Posted by ${post.username} on ${new Date(post.created_at).toLocaleDateString()}</p>
                        <p class="post-content">${post.content}</p>
                        <div class="post-categories">
                            ${post.categories ? post.categories.map(cat => `<span class="category">${cat}</span>`).join('') : ''}
                        </div>
                        <div class="post-reactions">
                            <span>👍 ${post.likes}</span>
                            <span>👎 ${post.dislikes}</span>
                        </div>
                    </div>
                `;
            });

            // Add click event listeners after adding posts
            const postCards = document.querySelectorAll('.post-card');
            postCards.forEach(card => {
                card.addEventListener('click', (e) => {
                    e.preventDefault();
                    const postId = card.dataset.postId;
                    window.history.pushState({}, '', `/post/${postId}`);
                    renderContent(`/post/${postId}`);
                });
            });
        });
}
function createPostContent() {
    document.getElementById('content').innerHTML = `
        <div class="create-post-container">
            <h2>Create New Post</h2>
            <form id="createPostForm">
                <input type="text" id="title" placeholder="Post Title" required>
                <textarea id="postContent" placeholder="Write your post here..." required></textarea>
                <div class="categories-section">
                    <select id="categories" multiple>
                        <option value="Technology">Technology</option>
                        <option value="Science">Science</option>
                        <option value="Sports">Sports</option>
                        <option value="News">News</option>
                        <option value="Entertainment">Entertainment</option>
                    </select>
                </div>
                <button type="submit">Create Post</button>
            </form>
        </div>
    `;

    document.getElementById('createPostForm').addEventListener('submit', handleCreatePost);
}

function singlePostContent() {
    const postId = window.location.pathname.split('/')[2];
    
    document.getElementById('content').innerHTML = `
        <div class="single-post-container">
            <div id="post-content">Loading post...</div>
            <div id="comments-section"></div>
        </div>
    `;

    fetch(`/post/${postId}`)
        .then(response => response.json())
        .then(post => {
            document.getElementById('post-content').innerHTML = `
                <h2>${post.title}</h2>
                <p class="post-meta">Posted by ${post.username} on ${new Date(post.created_at).toLocaleDateString()}</p>
                <div class="post-content">${post.content}</div>
                <div class="post-categories">
                    ${post.categories ? post.categories.map(cat => `<span class="category">${cat}</span>`).join('') : ''}
                </div>
                <div class="post-reactions">
                    <span>👍 ${post.likes}</span>
                    <span>👎 ${post.dislikes}</span>
                </div>
            `;

            // Only show comments section if there are comments
            if (post.comments && post.comments.length > 0) {
                const commentsSection = document.getElementById('comments-section');
                commentsSection.innerHTML = `
                    <h3>Comments</h3>
                    <div class="comments-list">
                        ${post.comments.map(comment => `
                            <div class="comment">
                                <p class="comment-meta">By ${comment.username} on ${new Date(comment.created_at).toLocaleDateString()}</p>
                                <p class="comment-content">${comment.content}</p>
                            </div>
                        `).join('')}
                    </div>
                `;
            }
        });
}


function chatContent() {
    document.getElementById('content').innerHTML = `
        <div class="chat-main">
            <div id="selectedUserName" class="selected-user"></div>
            <div class="chat-messages" id="messageHistory"></div>
            <form id="messageForm" class="chat-input">
                <input type="text" id="messageInput" placeholder="Type a message...">
                <button type="submit">Send</button>
            </form>
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
    document.body.className = 'login-page';
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
    document.body.className = 'login-page';
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
        event.preventDefault();
    }

    const sessionToken = localStorage.getItem('sessionToken');

    // If no session token, redirect to login immediately
    if (!sessionToken) {
        window.history.pushState({}, '', '/login');
        loginContent();
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
            localStorage.removeItem('sessionToken');
            updateNavigation();
            window.history.pushState({}, '', '/login');
            loginContent();
        } else {
            console.error('Logout failed with status:', response.status);
            // Still redirect to login on failure
            window.history.pushState({}, '', '/login');
            loginContent();
        }
    } catch (error) {
        console.error('Logout failed:', error);
        // Also redirect to login on error
        window.history.pushState({}, '', '/login');
        loginContent();
    }
}



// Update navigation links visibility based on login state
function updateNavigation() {
    const sessionToken = localStorage.getItem('sessionToken');
    const homeLink = document.querySelector('a[href="/"]');
    const registerLink = document.querySelector('a[href="/register"]');
    const logoutLink = document.querySelector('a[href="/logout"]');
    const loginLink = document.querySelector('a[href="/login"]');
    const createPostLink = document.querySelector('a[href="/create_post"]');
    
    if (sessionToken) {
        if (homeLink) homeLink.style.display = 'block';
        if (registerLink) registerLink.style.display = 'none';
        if (logoutLink) logoutLink.style.display = 'block';
        if (loginLink) loginLink.style.display = 'none';
        if (createPostLink) createPostLink.style.display = 'block';
    } else {
        if (homeLink) homeLink.style.display = 'none';
        if (registerLink) registerLink.style.display = 'block';
        if (logoutLink) logoutLink.style.display = 'none';
        if (loginLink) loginLink.style.display = 'block';
        if (createPostLink) createPostLink.style.display = 'none';
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
export { initRouter, displayErrors, renderContent };



