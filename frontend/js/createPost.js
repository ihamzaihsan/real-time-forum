import { displayErrors, navigate } from './router.js';
import { api, showToast } from './ui.js';

export async function handleCreatePost(event) {
    event.preventDefault();
    const form = event.currentTarget;
    const button = form.querySelector('button[type="submit"]');
    const title = form.elements.title.value.trim();
    const content = form.elements.content.value.trim();
    const categories = Array.from(form.elements.categories.selectedOptions, option => option.value);
    if (!title || !content || !categories.length) { displayErrors(['Add a title, your perspective, and at least one topic.']); return; }
    if (title.length > 200 || content.length > 2000) { displayErrors(['Use at most 200 characters for the title and 2,000 for the post.']); return; }
    button.disabled = true;
    try {
        await api('/create_post', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ title, content, categories }) });
        showToast('Your perspective is now part of the conversation.');
        navigate('/');
    } catch (error) { displayErrors([error.message]); }
    finally { button.disabled = false; }
}
