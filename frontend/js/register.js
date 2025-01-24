import { displayErrors } from './router.js';

export async function handleRegisterSubmit(event) {
    event.preventDefault();

    const userData = {
        username: document.getElementById('username').value.trim(),
        email: document.getElementById('email').value.trim(),
        password: document.getElementById('password').value.trim(),
        first_name: document.getElementById('first_name').value.trim(),
        last_name: document.getElementById('last_name').value.trim(),
        age: parseInt(document.getElementById('age').value, 10),
        gender: document.querySelector('input[name="gender"]:checked').value
    };

    // Validation patterns
    const emailPattern = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
    const passwordPattern = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\W).{8,}$/; // At least 8 chars, one upper, one lower, one special
    const errors = [];

    // Validation logic
    if (!userData.username) {
        errors.push('Username is required.');
    }
    if (!emailPattern.test(userData.email)) {
        errors.push('Invalid email format.');
    }
    // if (!passwordPattern.test(userData.password)) {
    //     errors.push('Password must be at least 8 characters long, include one uppercase letter, one lowercase letter, and one special character.');
    // }
    if (isNaN(userData.age) || userData.age <= 0) {
        errors.push('Age must be a positive number.');
    }
    if (!userData.gender) {
        errors.push('Gender is required.');
    }

    // In handleRegisterSubmit, replace the errors alert with:
    if (errors.length > 0) {
        displayErrors(errors);
        return;
    }

    // Proceed with submission if validation passes
    try {
        const response = await fetch('/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(userData)
        });

        if (response.ok) {
            const result = await response.json();
            localStorage.setItem('sessionToken', result.token);
            
            // Hide the register form
            document.getElementById('registerForm').style.display = 'none';
            
            // Show success message
            const content = document.getElementById('content');
            content.innerHTML += `
                <div class="success-message">
                    Registration successful! Redirecting to home page...
                </div>
            `;

            // Redirect to home page after a short delay
            setTimeout(() => {
                window.location.href = '/';
            }, 2000);
        }
    } catch (error) {
        alert('Error during registration: ' + error);
    }
}
