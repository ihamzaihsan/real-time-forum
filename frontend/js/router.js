// Route definitions: Map URL paths to corresponding content functions
const routes = {
    '/': homeContent,
    '/register': registerContent,
    '/logout': logoutContent
};

// Home page content function
function homeContent() {
    document.getElementById('content').innerHTML = `
        <h1>Welcome to the Home Page</h1>
        <p>This is the home page content.</p>
    `;
}

import { handleRegisterSubmit } from './register.js';

// Register page content function
function registerContent() {
    document.getElementById('content').innerHTML = `
        <h1>Register</h1>
        <form id="registerForm">
            <input type="text" id="username" placeholder="Username" required><br> 
            <input type="email" id="email" placeholder="Email" required><br>
            <input type="password" id="password" placeholder="Password" required><br>
            <input type="text" id="first_name" placeholder="First Name" required><br>
            <input type="text" id="last_name" placeholder="Last Name" required><br>
            <input type="number" id="age" placeholder="Age" required><br>
            <div class="gender-selection">
                <label>Gender:</label>
                <input type="radio" id="male" name="gender" value="male" required>
                <label for="male">Male</label>
                <input type="radio" id="female" name="gender" value="female" required>
                <label for="female">Female</label>
            </div>
            <button type="submit">Register</button>
        </form>
    `;

    document.getElementById('registerForm').addEventListener('submit', handleRegisterSubmit);
}

// Handle navigation when a link is clicked
function handleRoute(event) {
    event.preventDefault();
    const path = event.target.getAttribute('href'); // Get the target URL path
    window.history.pushState({}, '', path); // Update the browser URL
    renderContent(path); // Render content based on the current path
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
    
    if (sessionToken) {
        console.log(sessionToken);
        if (registerLink) registerLink.style.display = 'none';
        if (logoutLink) logoutLink.style.display = 'block';
    } else {
        if (registerLink) registerLink.style.display = 'block';
        if (logoutLink) logoutLink.style.display = 'none';
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

// Display errors for form validation
function displayErrors(errors) {
    // Create or get error container
    let errorContainer = document.getElementById('error-container');
    if (!errorContainer) {
        errorContainer = document.createElement('div');
        errorContainer.id = 'error-container';
        errorContainer.style.color = 'red';
        errorContainer.style.marginBottom = '10px';
        const form = document.getElementById('registerForm');
        form.insertBefore(errorContainer, form.firstChild);
    }
    
    // Display errors
    errorContainer.innerHTML = errors.map(error => `<p>${error}</p>`).join('');
}

// Export initRouter and displayErrors for use in other modules
export { initRouter, displayErrors };
