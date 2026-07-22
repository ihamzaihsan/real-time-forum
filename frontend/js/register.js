import { displayErrors } from './router.js';
import { api } from './ui.js';

export async function handleRegisterSubmit(event) {
    event.preventDefault();
    const form = event.currentTarget;
    const button = form.querySelector('button[type="submit"]');
    const fields = form.elements;
    const user = {
        username: fields.username.value.trim(),
        email: fields.email.value.trim(),
        request_moderator: Boolean(fields.request_moderator?.checked),
        password: fields.password.value,
        password_confirmation: fields.password_confirmation.value,
        first_name: fields.first_name.value.trim(),
        last_name: fields.last_name.value.trim(),
        age: Number(fields.age.value),
        gender: fields.gender.value,
    };
    if (!user.username || !user.first_name || !user.last_name || !user.gender) {
        displayErrors(['Please complete all the fields.']);
        return;
    }
    if (user.password.length < 8 || new TextEncoder().encode(user.password).length > 72) {
        displayErrors(['Use a password of at least 8 characters and no more than 72 bytes.']);
        return;
    }
    if (!Number.isInteger(user.age) || user.age < 1 || user.age > 120) {
        displayErrors(['Enter an age between 1 and 120.']);
        return;
    }
    if (user.password !== user.password_confirmation) {
        displayErrors(['Passwords do not match.']);
        return;
    }
    button.disabled = true;
    try {
        const result = await api('/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(user),
        });
        localStorage.removeItem('sessionToken');
        localStorage.setItem('isAdmin', String(result.is_admin));
        localStorage.setItem('role', result.role);
        localStorage.setItem('userId', result.user_id);
        localStorage.setItem('username', user.username);
        window.location.assign('/');
    } catch (error) {
        displayErrors([error.message]);
    } finally {
        button.disabled = false;
    }
}
