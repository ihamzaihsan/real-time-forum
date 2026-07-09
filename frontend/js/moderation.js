import { api, escapeHTML as esc, emptyState, showToast } from './ui.js';
const json = (body) => ({
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
});
export async function reportPost(id) {
    const category = prompt('Category: irrelevant, obscene, illegal, insulting, or other', 'other');
    if (!category) return;
    const reason = prompt('Describe the concern (up to 2,000 characters)');
    if (!reason) return;
    try {
        await api('/reports', json({ post_id: Number(id), category, reason }));
        showToast('Report submitted. Replies appear in Community tools.');
    } catch (error) {
        showToast(error.message);
    }
}
export async function renderModeration(offset = 0) {
    const content = document.getElementById('content');
    content.innerHTML =
        '<div class="page-heading"><h1>Community tools</h1><p>Requests, replies, reports, and content review.</p></div><div id="communityTools">Loading...</div>';
    const list = document.getElementById('communityTools');
    try {
        const data = await api(`/community?limit=10&offset=${offset}`);
        if (!list.isConnected) return;
        const admin = data.role === 'admin',
            staff = ['admin', 'moderator'].includes(data.role);
        const action = (label, body) =>
            `<button class="button button-quiet" data-community="${esc(JSON.stringify(body))}">${esc(label)}</button>`;
        list.innerHTML = `<p>Your role: ${esc(data.role)}. Member review: ${data.premoderate ? 'enabled' : 'disabled'}.</p>`;
        if (data.role === 'member') list.innerHTML += action('Request moderator access', { action: 'request' });
        for (const [label, key, kind] of [
            ['Pending discussions', 'pending_posts', 'post'],
            ['Pending replies', 'pending_comments', 'comment'],
        ]) {
            if (!staff) continue;
            list.innerHTML +=
                `<h2>${label}</h2>` +
                (data[key] || [])
                    .map(
                        (item) =>
                            `<article class="post-card"><a href="/post/${Number(kind === 'post' ? item.id : item.post_id)}" data-route>${esc(item.title)}</a><p>${esc(item.content)}</p>${kind === 'post' && item.image_path ? `<img class="post-image" src="/uploads/${encodeURIComponent(item.image_path)}" alt="Pending attachment">` : ''}${action('Approve', { action: 'approve', kind, id: item.id })}${action('Reject', { action: 'reject', kind, id: item.id })}</article>`,
                    )
                    .join('');
        }
        list.innerHTML +=
            '<h2>Moderator requests</h2>' +
            (data.requests || [])
                .map(
                    (item) =>
                        `<article class="post-card"><strong>${esc(item.username)} - ${esc(item.status)}</strong><p>${esc(item.message)}</p><p>${esc(item.reply)}</p>${admin && item.status === 'pending' ? action('Accept', { action: 'accept', id: item.id }) + action('Decline', { action: 'decline', id: item.id }) : ''}</article>`,
                )
                .join('');
        list.innerHTML +=
            '<h2>Reports and replies</h2>' +
            (data.reports || [])
                .map(
                    (item) =>
                        `<article class="post-card"><h3>${esc(item.title)}</h3><p>${esc(item.category)}: ${esc(item.reason)}</p><p>${esc(item.username)} - ${item.answered ? 'Answered' : 'Awaiting response'}</p><p>${esc(item.reply)}</p>${admin && !item.answered ? `<button class="button button-quiet" data-report="${item.id}" data-action="reply">Reply</button><button class="button button-dark" data-report="${item.id}" data-action="delete">Reply and delete discussion</button>` : ''}</article>`,
                )
                .join('');
        if (admin) {
            list.innerHTML +=
                '<h2>Account roles</h2>' +
                (data.users || [])
                    .map(
                        (user) =>
                            `<p>${esc(user.username)} (${esc(user.role)}) ${user.role !== 'admin' ? action('Promote', { action: 'role', id: user.id, role: 'moderator' }) + action('Demote', { action: 'role', id: user.id, role: 'member' }) : ''}</p>`,
                    )
                    .join('');
            list.innerHTML +=
                '<h2>Topics</h2>' +
                action('Create topic', { action: 'topic_create' }) +
                (data.topics || [])
                    .map(
                        (topic) =>
                            `<p>${esc(topic.name)} ${action('Delete topic', { action: 'topic_delete', id: topic.id })}</p>`,
                    )
                    .join('');
        }
        list.innerHTML += `<div class="feed-pagination"><button class="button button-quiet" id="communityPrev" ${offset === 0 ? 'disabled' : ''}>Previous</button><span>Page ${offset / 10 + 1}</span><button class="button button-quiet" id="communityNext">Next</button></div>`;
        list.querySelectorAll('[data-community]').forEach((button) =>
            button.addEventListener('click', async () => {
                const body = JSON.parse(button.dataset.community);
                if (body.action === 'request') {
                    body.message = prompt('Why would you like to moderate?');
                    if (!body.message) return;
                }
                if (['accept', 'decline'].includes(body.action)) {
                    body.reply = prompt('Reply to the requester');
                    if (!body.reply) return;
                }
                if (body.action === 'topic_create') {
                    body.name = prompt('Topic name (1-32 bytes)');
                    if (!body.name) return;
                }
                if (['reject', 'topic_delete'].includes(body.action) && !confirm('Confirm removal?')) return;
                button.disabled = true;
                try {
                    await api('/community', json(body));
                    if (list.isConnected) renderModeration(offset);
                } catch (error) {
                    showToast(error.message);
                    button.disabled = false;
                }
            }),
        );
        list.querySelectorAll('[data-report]').forEach((button) =>
            button.addEventListener('click', async () => {
                const reply = prompt('Reply to the reporter');
                if (!reply) return;
                button.disabled = true;
                try {
                    await api(
                        '/moderate',
                        json({ report_id: Number(button.dataset.report), action: button.dataset.action, reply }),
                    );
                    if (list.isConnected) renderModeration(offset);
                } catch (error) {
                    showToast(error.message);
                    button.disabled = false;
                }
            }),
        );
        document.getElementById('communityPrev').onclick = () => renderModeration(Math.max(0, offset - 10));
        document.getElementById('communityNext').onclick = () => renderModeration(offset + 10);
    } catch (error) {
        if (list.isConnected) list.innerHTML = emptyState('Could not load community tools', error.message);
    }
}
