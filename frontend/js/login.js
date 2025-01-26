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
        console.log('Login response not OK:', {
            status: response.status,
            statusText: response.statusText,
            error: errorText
        });
        displayErrors([errorText]);
        return;
    }

    const result = await response.json();
    console.log('Login success response:', result);

    localStorage.clear();
    localStorage.setItem('sessionToken', result.token);
    localStorage.setItem('username', loginData.username);
    localStorage.setItem('userId', result.user_id.toString());
    if (window.wsClient) {
        window.wsClient.connect();
    }

    window.location.href = '/';
    
} catch (error) {
    console.log('Login error details:', error);
    displayErrors(['Error during login. Please try again.']);
}

}
