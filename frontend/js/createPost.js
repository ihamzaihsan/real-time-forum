import { displayErrors, navigate } from './router.js';
import { api, showToast } from './ui.js';

export async function handleCreatePost(event) {
    event.preventDefault();
    const form = event.currentTarget;
    const button = form.querySelector('button[type="submit"]');
    const title = form.elements.title.value.trim();
    const content = form.elements.content.value.trim();
    const categories = Array.from(form.elements.categories.selectedOptions, (option) => option.value);
    if (!title || !content || !categories.length) {
        displayErrors(['Add a title, your perspective, and at least one topic.']);
        return;
    }
    if (title.length > 200 || content.length > 2000) {
        displayErrors(['Use at most 200 characters for the title and 2,000 for the post.']);
        return;
    }
    button.disabled = true;
    try {
        const body = new FormData();
        body.set('title', title);
        body.set('content', content);
        categories.forEach((category) => body.append('categories', category));
        const image = form.elements.image.files[0];
        if (image && image.size > 20 * 1024 * 1024) throw new Error('Image exceeds the 20 MiB limit.');
        if (image) body.set('image', image);
        const result = await api('/create_post', { method: 'POST', body });
        showToast(
            result.status === 'pending'
                ? 'Discussion saved for review.'
                : 'Your perspective is now part of the conversation.',
        );
        navigate(result.status === 'pending' ? '/activity' : '/');
    } catch (error) {
        displayErrors([error.message]);
    } finally {
        button.disabled = false;
    }
}
