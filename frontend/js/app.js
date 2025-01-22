// app.js

// Import the router logic from router.js
import { initRouter } from './router.js';

// Initialize the router when the DOM content is loaded
document.addEventListener('DOMContentLoaded', () => {
    // Initialize the routing logic
    initRouter();
    
    // Connect to WebSocket
    const socket = new WebSocket('ws://localhost:8080/ws');

    socket.onopen = () => {
        console.log('WebSocket connection established');
    };

    socket.onmessage = (event) => {
        console.log('Message from server:', event.data);
        // Handle incoming messages (e.g., update UI)
    };

    socket.onclose = () => {
        console.log('WebSocket connection closed');
    };

    socket.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
});