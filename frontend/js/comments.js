import { openEditor, remove } from './activity.js';
import { api, icon, escapeHTML as esc, initials, dateLabel, emptyState, showToast } from './ui.js';
import { handleCommentLike } from './likes.js';

export async function loadComments(postId) {
    const comments = await api(`/comments?post_id=${Number(postId)}`);
    return Array.isArray(comments) ? comments : [];
}

export async function refreshComments(postId) {
    const list = document.getElementById('commentsList');
    if (!list) return;
    const request = Number(list.dataset.request || 0) + 1;
    list.dataset.request = request;
    let comments;
    try {
        comments = await loadComments(postId);
    } catch (error) {
        if (document.getElementById('commentsList') === list && Number(list.dataset.request) === request) throw error;
        return;
    }
    if (document.getElementById('commentsList') === list && Number(list.dataset.request) === request)
        renderComments(comments);
}

export function renderCommentSection(authenticated = true) {
    return `<section class="comments-section"><h2>Keep the conversation going.</h2>${authenticated ? `<form id="commentForm" class="comment-form"><label for="commentContent">Your reply <span class="field-hint">200 characters max</span></label><textarea id="commentContent" name="comment" placeholder="Add your perspective…" maxlength="200" required></textarea><button type="submit" class="button button-dark">Post reply ${icon('arrow')}</button><div id="commentError" role="alert" class="field-help"></div></form>` : '<p class="field-help"><a href="/login" data-route class="text-link">Log in to add your perspective ↗</a></p>'}<div id="commentsList" class="comments-list" aria-live="polite"><p class="muted">Loading replies…</p></div></section>`;
}

export async function initializeComments(postId) {
    const list = document.getElementById('commentsList');
    const form = document.getElementById('commentForm');
    if (!list) return;
    let refreshVersion = 0;
    const isCurrent = () => document.getElementById('commentsList') === list;

    async function refresh() {
        const version = ++refreshVersion;
        try {
            await refreshComments(postId);
            return true;
        } catch (error) {
            if (!isCurrent() || version !== refreshVersion) return false;
            const detail =
                error instanceof TypeError
                    ? 'Couldn’t reach Yaplane. Check your connection and try again.'
                    : error.message;
            list.innerHTML = emptyState('Couldn’t load replies.', detail);
            const retry = document.createElement('button');
            retry.type = 'button';
            retry.className = 'button button-quiet';
            retry.textContent = 'Reload replies';
            retry.addEventListener('click', async () => {
                retry.disabled = true;
                await refresh();
            });
            list.append(retry);
            return false;
        }
    }

    // Bind immediately: slow initial loading must never trigger a native form navigation.
    form?.addEventListener('submit', async (event) => {
        event.preventDefault();
        const content = form.elements.comment.value.trim();
        if (!content) return;
        const feedback = document.getElementById('commentError');
        if (Array.from(content).length > 200) {
            feedback.textContent = 'Comment must be 200 characters or less.';
            return;
        }
        const button = form.querySelector('button');
        button.disabled = true;
        feedback.textContent = '';
        try {
            const result = await api('/comment', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ post_id: Number(postId), content }),
            });
        } catch (error) {
            if (isCurrent())
                feedback.textContent =
                    error instanceof TypeError
                        ? 'Couldn’t confirm your reply. Your draft is still here. Check the replies before trying again.'
                        : error.message;
            button.disabled = false;
            return;
        }
        if (!isCurrent()) return;
        form.reset();
        feedback.textContent = 'Reply saved. Pending replies remain private until approval.';
        const refreshed = await refresh();
        if (isCurrent() && !refreshed)
            feedback.textContent = 'Reply posted, but the replies couldn’t refresh. Use “Reload replies” below.';
        button.disabled = false;
    });
    await refresh();
}

export function renderComments(comments) {
    const list = document.getElementById('commentsList');
    if (!list) return;
    list.innerHTML = comments.length
        ? comments
              .map(
                  (comment) =>
                      `<article class="comment" data-comment-id="${Number(comment.id)}"><div class="comment-header"><span class="avatar avatar-soft">${esc(initials(comment.username))}</span><strong>${esc(comment.username || 'Community member')}</strong><span class="comment-date">${esc(dateLabel(comment.created_at))}</span></div><p class="comment-content">${esc(comment.content)}</p>${comment.status === 'pending' ? '<p>Awaiting review</p>' : ''}<div class="comment-actions"><button class="reaction-btn" data-comment-like="${Number(comment.id)}" aria-label="Like reply">${icon('up')}<span class="likes-count">${comment.likes || 0}</span></button><button class="reaction-btn" data-comment-dislike="${Number(comment.id)}" aria-label="Dislike reply">${icon('down')}<span class="dislikes-count">${comment.dislikes || 0}</span></button>${Number(comment.user_id) === Number(localStorage.getItem('userId')) ? `<button class="button button-quiet" data-edit-comment="${comment.id}">Edit</button>` : ''}${Number(comment.user_id) === Number(localStorage.getItem('userId')) || localStorage.getItem('role') === 'admin' ? `<button class="button button-quiet" data-delete-comment="${comment.id}">Delete</button>` : ''}</div></article>`,
              )
              .join('')
        : emptyState('Be the first to add a thought.', 'The best conversations have more than one perspective.');
    list.querySelectorAll('[data-edit-comment]').forEach(
        (button) =>
            (button.onclick = () =>
                openEditor(
                    'comment',
                    comments.find((item) => item.id === Number(button.dataset.editComment)),
                )),
    );
    list.querySelectorAll('[data-delete-comment]').forEach(
        (button) =>
            (button.onclick = () =>
                remove('comment', Number(button.dataset.deleteComment), () =>
                    refreshComments(document.getElementById('post-content').dataset.postId),
                )),
    );
    list.querySelectorAll('[data-comment-like], [data-comment-dislike]').forEach((button) =>
        button.addEventListener('click', () => {
            if (!localStorage.getItem('userId')) {
                showToast('Log in to react to replies.');
                return;
            }
            handleCommentLike(
                Number(button.dataset.commentLike || button.dataset.commentDislike),
                Boolean(button.dataset.commentLike),
            );
        }),
    );
}
