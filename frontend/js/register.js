// Handle register form submission
async function handleRegisterSubmit(event) {
    event.preventDefault();
    
    const userData = {
        username: document.getElementById('username').value,
        email: document.getElementById('email').value,
        password: document.getElementById('password').value,
        first_name: document.getElementById('first_name').value,
        last_name: document.getElementById('last_name').value,
        age: parseInt(document.getElementById('age').value),
        gender: document.getElementById('gender').value
    };

    try {
        const response = await fetch('/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(userData)
        });

        if (response.ok) {
            window.location.href = '/';
        } else {
            const error = await response.text();
            alert('Registration failed: ' + error);
        }
    } catch (error) {
        alert('Error during registration: ' + error);
    }
}

export { handleRegisterSubmit };
