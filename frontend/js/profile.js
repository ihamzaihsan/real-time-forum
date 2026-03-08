import { api, initials, showToast } from './ui.js';

export async function loadProfileData(userId) {
    const avatar = document.getElementById('profileAvatar');
    try {
        const profile = await api(`/profile/${userId}`);
        if (document.getElementById('profileAvatar') !== avatar) return;
        avatar.textContent = initials(`${profile.first_name} ${profile.last_name}`);
        document.getElementById('fullName').textContent = `${profile.first_name} ${profile.last_name}`;
        document.getElementById('username').textContent = `@${profile.username}`;
        document.getElementById('profileEmail').textContent = profile.email;
        document.getElementById('profileAge').textContent = profile.age;
        document.getElementById('profileGender').textContent = profile.gender;
    } catch (error) {
        if (document.getElementById('profileAvatar') !== avatar) return;
        document.getElementById('fullName').textContent = 'Profile unavailable';
        showToast(error.message);
    }
}
