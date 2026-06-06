import { api, showToast } from './ui.js';

const pendingReactions = new Set();

async function react(kind, id, isLike) {
    const key = `${kind}:${id}`;
    if (pendingReactions.has(key)) return;
    pendingReactions.add(key);
    const body = new URLSearchParams();
    body.set(kind === 'post' ? 'post_id' : 'comment_id', id);
    body.set('is_like', String(isLike));
    try {
        const result = await api(kind === 'post' ? '/like' : '/comment/like', { method: 'POST', body });
        const selector = kind === 'post' ? `[data-post-id="${id}"]` : `[data-comment-id="${id}"]`;
        document.querySelectorAll(selector).forEach(node => {
            const likes = node.querySelector('.likes-count');
            const dislikes = node.querySelector('.dislikes-count');
            if (likes) likes.textContent = result.likes;
            if (dislikes) dislikes.textContent = result.dislikes;
        });
    } catch (error) { showToast(error.message); }
    finally { pendingReactions.delete(key); }
}

export function handleLike(postId, isLike) { return react('post', Number(postId), isLike); }
export function handleCommentLike(commentId, isLike) { return react('comment', Number(commentId), isLike); }
