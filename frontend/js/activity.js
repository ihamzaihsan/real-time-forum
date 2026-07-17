import { api, escapeHTML as esc, emptyState, showToast } from './ui.js';
const json = (body) => ({
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
});
let activityOffset = 0,
    inboxOffset = 0,
    notificationVersion = 0;
function pages(container, offset, next, render) {
    const controls = document.createElement('div');
    controls.className = 'feed-pagination';
    for (const [label, target, enabled] of [
        ['Previous', offset - 10, offset > 0],
        ['Next', offset + 10, next],
    ]) {
        const button = document.createElement('button');
        button.className = 'button button-quiet';
        button.textContent = label;
        button.disabled = !enabled;
        button.onclick = () => render(Math.max(0, target));
        controls.append(button);
    }
    container.append(controls);
}
export async function renderActivity(offset = 0) {
    activityOffset = offset;
    const content = document.getElementById('content');
    content.innerHTML =
        '<div class="page-heading"><h1>My activity</h1><p>Your discussions, replies, and current reactions.</p></div><div id="activityList">Loading...</div>';
    const list = document.getElementById('activityList');
    try {
        const data = await api(`/api/activity?limit=10&offset=${offset}`);
        if (!list.isConnected) return;
        const link = (item) =>
            item.post_id
                ? `<a href="/post/${Number(item.post_id)}" data-route>${esc(item.title)}</a>`
                : `<span>${esc(item.title)}</span>`;
        list.innerHTML =
            '<h2>Discussions</h2>' +
            data.posts
                .map(
                    (item) =>
                        `<article class="post-card"><a href="/post/${item.id}" data-route>${esc(item.title)}</a><p>${esc(item.content)}</p><p>${esc(item.status)}</p><button class="button button-quiet" data-edit-post="${item.id}">Edit</button><button class="button button-quiet" data-delete-post="${item.id}">Delete</button></article>`,
                )
                .join('') +
            '<h2>Replies</h2>' +
            data.comments
                .map(
                    (item) =>
                        `<article class="post-card">${link(item)}<p>${esc(item.content)}</p><p>${esc(item.status)}</p><button class="button button-quiet" data-edit-comment="${item.id}">Edit</button><button class="button button-quiet" data-delete-comment="${item.id}">Delete</button></article>`,
                )
                .join('') +
            '<h2>Current reactions</h2>' +
            data.reactions
                .map(
                    (item) =>
                        `<article class="post-card">${link(item)}<p>${item.is_like ? 'Liked' : 'Disliked'}${item.comment_id ? ' a reply: ' + esc(item.comment) : ' this discussion'}</p></article>`,
                )
                .join('');
        list.querySelectorAll('[data-edit-post]').forEach(
            (button) => (button.onclick = () => editPost(Number(button.dataset.editPost))),
        );
        list.querySelectorAll('[data-edit-comment]').forEach(
            (button) =>
                (button.onclick = () =>
                    openEditor(
                        'comment',
                        data.comments.find((item) => item.id === Number(button.dataset.editComment)),
                    )),
        );
        list.querySelectorAll('[data-delete-post]').forEach(
            (button) =>
                (button.onclick = () =>
                    remove('post', Number(button.dataset.deletePost), () => renderActivity(offset))),
        );
        list.querySelectorAll('[data-delete-comment]').forEach(
            (button) =>
                (button.onclick = () =>
                    remove('comment', Number(button.dataset.deleteComment), () => renderActivity(offset))),
        );
        if (!data.posts.length && !data.comments.length && !data.reactions.length)
            list.innerHTML += emptyState(
                'No activity on this page',
                'Start a discussion or return to the previous page.',
            );
        pages(
            list,
            offset,
            Object.values(data).some((items) => items.length === 10),
            renderActivity,
        );
    } catch (error) {
        if (list.isConnected) list.innerHTML = emptyState('Could not load activity', error.message);
    }
}
export async function remove(kind, id, after) {
    if (!confirm('Delete this ' + kind + '?')) return;
    try {
        const options =
            kind === 'post'
                ? {
                      method: 'POST',
                      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                      body: new URLSearchParams({ post_id: id }),
                  }
                : json({ id });
        await api(kind === 'post' ? '/delete-post' : '/delete-comment', options);
        await after?.();
    } catch (error) {
        showToast(error.message);
    }
}
export async function editPost(id) {
    try {
        const item = await api('/post/' + id);
        await openEditor('post', item);
    } catch (error) {
        showToast(error.message);
    }
}
export async function openEditor(kind, item) {
    document.getElementById('contentEditor')?.remove();
    const panel = document.createElement('section');
    panel.className = 'form-panel';
    panel.id = 'contentEditor';
    panel.innerHTML = `<h2>Edit ${kind === 'post' ? 'discussion' : 'reply'}</h2><p>Attachments are preserved. Member edits may return to review.</p><form><div role="alert" class="error-messages"></div>${kind === 'post' ? `<label>Title<input name="title" maxlength="200" value="${esc(item.title)}" required></label><label>Topics<select name="categories" multiple required></select></label>` : ''}<label>Text<textarea name="content" maxlength="${kind === 'post' ? 2000 : 200}" required>${esc(item.content)}</textarea></label><button class="button button-dark" type="submit">Save changes</button><button class="button button-quiet" type="button" data-cancel>Cancel</button></form>`;
    document.getElementById('content').prepend(panel);
    panel.scrollIntoView({ block: 'start' });
    panel.querySelector('[data-cancel]').onclick = () => panel.remove();
    const form = panel.querySelector('form');
    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        const button = form.querySelector('[type="submit"]');
        button.disabled = true;
        const body = { id: item.id, content: form.elements.content.value };
        if (kind === 'post') {
            body.title = form.elements.title.value;
            body.categories = Array.from(form.elements.categories.selectedOptions, (option) => option.value);
        }
        try {
            const result = await api('/edit-' + kind, json(body));
            showToast(result.status === 'pending' ? 'Changes saved for review.' : 'Changes saved.');
            panel.remove();
            window.dispatchEvent(new CustomEvent('contentedited'));
        } catch (error) {
            form.querySelector('[role="alert"]').textContent = error.message;
            button.disabled = false;
        }
    });
    if (kind === 'post') {
        try {
            const topics = await api('/categories');
            if (!panel.isConnected) return;
            form.elements.categories.innerHTML = topics
                .map(
                    (topic) =>
                        `<option value="${esc(topic)}" ${(item.categories || []).includes(topic) ? 'selected' : ''}>${esc(topic)}</option>`,
                )
                .join('');
        } catch (error) {
            form.querySelector('[role="alert"]').textContent = error.message;
        }
    }
}
export async function refreshNotifications() {
    if (!localStorage.getItem('userId')) {
        const badge = document.getElementById('notification-badge');
        if (badge) badge.hidden = true;
        return;
    }
    const version = ++notificationVersion;
    try {
        const data = await api(`/api/notifications?limit=10&offset=${inboxOffset}`);
        if (version !== notificationVersion) return;
        const badge = document.getElementById('notification-badge');
        if (badge) {
            badge.textContent = data.unread;
            badge.hidden = data.unread === 0;
        }
        const list = document.getElementById('inboxList');
        if (!list) return;
        list.innerHTML = data.items.length
            ? data.items
                  .map(
                      (item) =>
                          `<article class="post-card" data-notification-id="${item.id}"><p><strong>${esc(item.actor)}</strong> ${item.kind === 'comment' ? 'commented on' : item.kind === 'like' ? 'liked' : 'disliked'} <a href="/post/${item.post_id}" data-route>${esc(item.title)}</a></p><p>${item.is_read ? 'Read' : 'Unread'}</p>${item.is_read ? '' : `<button class="button button-quiet" data-read="${item.id}">Mark as read</button>`}</article>`,
                  )
                  .join('')
            : emptyState(
                  'No notifications on this page',
                  'Replies and reactions to your discussions will appear here.',
              );
        list.querySelectorAll('[data-read]').forEach(
            (button) => (button.onclick = () => readNotifications(Number(button.dataset.read))),
        );
        pages(list, inboxOffset, data.items.length === 10, renderNotifications);
    } catch (error) {
        const list = document.getElementById('inboxList');
        if (list) list.innerHTML = emptyState('Could not load notifications', error.message);
    }
}
async function readNotifications(id) {
    try {
        await api('/api/notifications', json({ id }));
        await refreshNotifications();
    } catch (error) {
        showToast(error.message);
    }
}
export async function renderNotifications(offset = 0) {
    inboxOffset = offset;
    document.getElementById('content').innerHTML =
        '<div class="page-heading"><h1>Notifications</h1><p>Saved updates for your discussions.</p></div><button class="button button-quiet" id="readAll">Mark all as read</button><div id="inboxList">Loading...</div>';
    document.getElementById('readAll').onclick = () => readNotifications(0);
    await refreshNotifications();
}
export function refreshActivityPage() {
    if (window.location.pathname === '/activity' && !document.getElementById('contentEditor'))
        return renderActivity(activityOffset);
}
