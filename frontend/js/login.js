import { displayErrors } from './router.js';
import { api } from './ui.js';

export async function handleLoginSubmit(event) {
    event.preventDefault();
    const form = event.currentTarget;
    const button = form.querySelector('button[type="submit"]');
    const username = form.elements.username.value.trim();
    const password = form.elements.password.value;
    if (!username || !password) {
        displayErrors(['Username and password are required.']);
        return;
    }
    button.disabled = true;
    try {
        const result = await api('/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        });
        localStorage.removeItem('sessionToken');
        localStorage.setItem('isAdmin', String(result.is_admin));
        localStorage.setItem('role', result.role);
        localStorage.setItem('userId', result.user_id);
        // Use the canonical username even when signing in with an email address.
        const profile = await api(`/profile/${result.user_id}`).catch(() => null);
        localStorage.setItem('username', profile?.username || username);
        window.location.assign('/');
    } catch (error) {
        displayErrors([error.message]);
    } finally {
        button.disabled = false;
    }
}
