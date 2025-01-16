// router.js

// Route definitions: Map URL paths to corresponding content functions
const routes = {
    '/': homeContent,
    '/register': registerContent
};

// Home page content function
function homeContent() {
    document.getElementById('content').innerHTML = `
        <h1>Welcome to the Home Page</h1>
        <p>This is the home page content.</p>
    `;
}

// Register page content function
function registerContent() {
    document.getElementById('content').innerHTML = `
        <h1>Register</h1>
        <form id="registerForm">
            <input type="text" placeholder="Username" required>
            <input type="email" placeholder="Email" required>
            <input type="password" placeholder="Password" required>
            <button type="submit">Register</button>
        </form>
    `;
}

// Handle navigation when a link is clicked
function handleRoute(event) {
    event.preventDefault();
    const path = event.target.getAttribute('href'); // Get the target URL path
    window.history.pushState({}, '', path); // Update the browser URL
    renderContent(path); // Render content based on the current path
}

// Render content based on the current route
function renderContent(path) {
    const render = routes[path] || routes['/']; // Default to home if no matching route
    render(); // Call the function for the current route
}

// Initialize router: Set up event listeners for navigation links
function initRouter() {
    // Add event listeners to navigation links
    document.querySelectorAll('a').forEach(link => {
        link.addEventListener('click', handleRoute);
    });

    // Listen to the browser history popstate event for back/forward navigation
    window.onpopstate = () => {
        renderContent(window.location.pathname);
    };

    // Handle initial page load (render the content for the current URL)
    renderContent(window.location.pathname);
}

// Expose initRouter for use in app.js
export { initRouter };
