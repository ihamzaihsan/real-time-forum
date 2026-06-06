import { api, escapeHTML as esc, emptyState, showToast } from './ui.js';

export async function reportPost(postId) {
    const reason = window.prompt('Why should this discussion be reviewed? (500 characters max)');
    if (reason === null) return;
    if (!reason.trim() || Array.from(reason.trim()).length > 500) { showToast('Enter a reason of 1–500 characters.'); return; }
    try {
        await api('/reports', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ post_id: Number(postId), reason: reason.trim() }) });
        showToast('Report submitted for moderator review.');
    } catch (error) { showToast(error.message); }
}

export async function renderModeration(offset = 0) {
    const container = document.getElementById('content');
    container.innerHTML = '<div class="page-heading"><h1>Moderation</h1><p>Review reported discussions.</p></div><section id="reportList" class="form-panel"></section>';
    const list = document.getElementById('reportList');
    try {
        const reports = await api(`/reports?limit=10&offset=${offset}`);
        if (!list.isConnected) return;
        list.innerHTML = reports.length ? reports.map(report => `<article class="comment"><h2><a href="/post/${Number(report.post_id)}" data-route>${esc(report.title)}</a></h2><p>${esc(report.reason)}</p><p class="muted">Reported by ${esc(report.username)}</p><button class="button button-quiet" data-report="${Number(report.id)}" data-action="dismiss">Dismiss</button> <button class="button button-dark" data-report="${Number(report.id)}" data-action="delete">Remove discussion</button></article>`).join('') : emptyState('No reports here.', 'Reported discussions will appear here.');
        list.querySelectorAll('[data-report]').forEach(button => button.addEventListener('click', async () => {
            if (button.dataset.action === 'delete' && !confirm('Remove this discussion and its replies?')) return;
            button.disabled = true;
            try {
                await api('/moderate', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ report_id: Number(button.dataset.report), action: button.dataset.action }) });
                if (list.isConnected) renderModeration(offset);
            } catch (error) { showToast(error.message); button.disabled = false; }
        }));
        const controls = document.createElement('div');
        for (const [label, target, enabled] of [['Previous', offset - 10, offset > 0], ['Next', offset + 10, reports.length === 10]]) {
            const button = document.createElement('button');
            button.className = 'button button-quiet'; button.textContent = label; button.disabled = !enabled;
            button.addEventListener('click', () => renderModeration(target));
            controls.append(button);
        }
        list.append(controls);
    } catch (error) {
        if (list.isConnected) list.innerHTML = emptyState('Could not load reports.', error.message);
    }
}
