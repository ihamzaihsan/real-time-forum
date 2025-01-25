import { displayErrors } from './router.js';

export async function handleLoginSubmit(event) {
    event.preventDefault();

    const loginData = {
        username: document.getElementById('username').value.trim(),
        password: document.getElementById('password').value.trim()
    };

    // Validation
    const errors = [];
    if (!loginData.username) {
        errors.push('Username or email is required.');
    }
    if (!loginData.password) {
        errors.push('Password is required.');
    }

    if (errors.length > 0) {
        displayErrors(errors);
        return;
    }

    try {
        const response = await fetch('/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(loginData)
        });

        if (!response.ok) {
            const errorText = await response.text();
            displayErrors([errorText]);
            return;
        }

        const result = await response.json();
        localStorage.setItem('sessionToken', result.token);
        
        // Hide login form
        document.getElementById('loginForm').style.display = 'none';
        
        // Redirect to home page
        window.location.href = '/';
        
    } catch (error) {
        displayErrors(['Error during login. Please try again.']);
    }
}
