import { api, showToast } from './ui.js';
import { navigate } from './router.js';

export async function deletePost(postId, event) {
    event?.stopPropagation();
    if (!confirm('Delete this discussion? This cannot be undone.')) return;
    try {
        await api('/delete-post', { method: 'POST', headers: { 'Content-Type': 'application/x-www-form-urlencoded' }, body: new URLSearchParams({ post_id: postId }) });
        showToast('Discussion deleted.');
        navigate('/');
    } catch (error) { showToast(error.message); }
}
