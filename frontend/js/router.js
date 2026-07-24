import { renderActivity, renderNotifications, editPost, refreshNotifications } from './activity.js';
import { reportPost, renderModeration } from './moderation.js';
import { handleRegisterSubmit } from './register.js';
import { handleLoginSubmit } from './login.js';
import { handleCreatePost } from './createPost.js';
import { renderCommentSection, initializeComments, refreshComments } from './comments.js';
import { handleLike } from './likes.js';
import { loadProfileData } from './profile.js';
import { deletePost } from './deletePost.js';
import { initializeChat, updateUsersList } from './chat.js';
import { api, icon, escapeHTML as esc, initials, dateLabel, emptyState, showToast } from './ui.js';

let activeTopic = '';
let searchQuery = '';
let sortOrder = 'newest';
let feed = [];
let renderVersion = 0;
let feedOffset = 0;
let feedTotal = 0;
let feedRequest = 0;
let searchTimer;

export function navigate(path) {
    window.history.pushState({}, '', path);
    renderContent(path);
}

async function renderContent(path) {
    const version = ++renderVersion;
    const authenticated = Boolean(localStorage.getItem('userId'));
    if (path === '/profile') path = `/profile/${localStorage.getItem('userId') || ''}`;
    if (path === '/logout') {
        await logout();
        return;
    }
    const authPage = path === '/login' || path === '/register' || path === '/complete-profile';
    if (authenticated && authPage) {
        navigate('/');
        return;
    }
    const privatePage =
        path === '/chat' ||
        path === '/create_post' ||
        path.startsWith('/profile/') ||
        path === '/moderation' ||
        path === '/activity' ||
        path === '/notifications';
    if (!authenticated && privatePage) {
        navigate('/login');
        return;
    }
    document.body.classList.toggle('auth-page', authPage);
    document.body.classList.toggle('chat-page', path === '/chat');
    document.title = `${authPage ? (path === '/login' ? 'Welcome back' : 'Join the conversation') : path === '/chat' ? 'Messages' : path === '/create_post' ? 'New discussion' : 'The common room'} · Yaplane`;
    updateNavigation(path);
    const content = document.getElementById('content');
    content.innerHTML = '';
    if (path === '/login') loginContent();
    else if (path === '/register') registerContent();
    else if (path === '/complete-profile') completeProfileContent();
    else if (path === '/chat') chatContent();
    else if (path === '/activity') await renderActivity();
    else if (path === '/notifications') await renderNotifications();
    else if (path === '/moderation') await renderModeration();
    else if (path === '/create_post') await createPostContent(version);
    else if (/^\/profile\/\d+$/.test(path)) profileContent(path.split('/')[2]);
    else if (/^\/post\/\d+$/.test(path)) await singlePostContent(path.split('/')[2], version);
    else await homeContent(version);
    if (path === '/login' || path === '/register') {
        const form = content.querySelector('form');
        if (path === '/register') {
            const label = document.createElement('label');
            label.innerHTML = '<input type="checkbox" name="request_moderator"> Request moderator consideration';
            form?.append(label);
        }
        const error = new URLSearchParams(window.location.search).get('error');
        if (error) displayErrors([error]);
        api('/auth-providers')
            .then((providers) => {
                if (!form?.isConnected) return;
                const box = document.createElement('div');
                box.className = 'form-footer';
                providers.forEach((provider) => {
                    const link = document.createElement('a');
                    link.href = '/auth/' + provider;
                    link.className = 'button button-quiet';
                    link.textContent = 'Continue with ' + (provider === 'github' ? 'GitHub' : 'Google');
                    box.append(link);
                });
                form.after(box);
            })
            .catch(() => {});
    }
}

function updateNavigation(path = window.location.pathname) {
    const authenticated = Boolean(localStorage.getItem('userId'));
    const userId = localStorage.getItem('userId');
    document.getElementById('moderationLink').hidden = !authenticated;
    const username = localStorage.getItem('username') || 'You';
    document.querySelectorAll('nav [data-route]').forEach((link) => {
        const selected =
            link.getAttribute('href') === path || (link.id === 'profileLink' && path.startsWith('/profile'));
        link.classList.toggle('active', selected);
        if (selected) link.setAttribute('aria-current', 'page');
        else link.removeAttribute('aria-current');
    });
    document.getElementById('profileLink').href = authenticated ? `/profile/${userId}` : '/profile';
    document.querySelector('.sidebar-footer a').hidden = !authenticated;
    const actions = document.querySelector('.header-actions');
    if (!authenticated) {
        actions.innerHTML =
            '<a href="/login" data-route class="button button-quiet">Log in</a><a href="/register" data-route class="button button-dark"><span class="join-long">Join the community</span><span class="join-short">Join us</span>' +
            icon('arrow') +
            '</a>';
        document.getElementById('onlineUsers').innerHTML =
            '<div class="members-invitation">' +
            icon('chat') +
            '<p>Meet your next conversation.</p><a href="/login" data-route>Sign in to connect ↗</a></div>';
    } else if (!document.getElementById('headerProfile')) {
        actions.innerHTML =
            '<span id="connectionStatus" class="connection-status" role="status">Connecting</span><a href="/create_post" data-route class="button button-dark header-create">' +
            icon('plus') +
            ' New discussion</a><a href="/profile" data-route id="headerProfile" class="avatar" aria-label="Your profile"></a>';
    }
    const profile = document.getElementById('headerProfile');
    if (profile) {
        profile.textContent = initials(username);
        profile.href = `/profile/${userId}`;
    }
}

function sculpture() {
    return '<div class="sculpture" aria-hidden="true"><div class="orbit orbit-one"></div><div class="orbit orbit-two"></div><div class="orbit-core"></div><span class="spark spark-one">✳</span><span class="spark spark-two">+</span><span class="sculpture-shadow"></span></div>';
}

async function homeContent(version) {
    document.getElementById('content').innerHTML = `
        <div class="feed-heading"><div><h2>The conversation</h2><p>Fresh perspectives from the community.</p></div><label class="sort-label">${icon('clock')}<select id="feedSort" aria-label="Sort discussions"><option value="newest">Latest first</option><option value="popular">Most liked</option></select></label></div>
        <div class="feed-controls"><div id="topicFilters" class="filter-tabs" aria-label="Filter discussions by topic"></div><span id="feedCount" class="muted"></span></div>
        <div id="searchSummary" class="search-summary" hidden></div>
        <div id="posts-container" class="posts-list" aria-live="polite"><div class="skeleton-card"></div><div class="skeleton-card"></div></div>
        <div id="feedPagination" class="feed-pagination" aria-label="Discussion pages"></div>
        <div class="feed-end">You’re right where you belong. <span>✳</span></div>`;
    document.getElementById('feedSort').value = sortOrder;
    document.getElementById('feedSort').addEventListener('change', (event) => {
        sortOrder = event.target.value;
        refreshFeed(true);
    });
    try {
        const categories = await api('/categories');
        if (version !== renderVersion) return;
        const topics = Array.isArray(categories) ? categories : [];
        if (activeTopic && !topics.includes(activeTopic)) activeTopic = '';
        document.getElementById('topicFilters').innerHTML = ['', ...topics]
            .map(
                (topic) =>
                    `<button type="button" class="filter-tab" data-topic="${esc(topic)}" aria-pressed="${topic === activeTopic}">${esc(topic || 'All discussions')}</button>`,
            )
            .join('');
        await refreshFeed(true);
    } catch (error) {
        if (version === renderVersion)
            document.getElementById('posts-container').innerHTML =
                emptyState('The room is taking a moment.', error.message) +
                '<button class="button button-quiet" id="retryFeed">Try again</button>';
        document.getElementById('retryFeed')?.addEventListener('click', () => renderContent('/'));
    }
}

function renderFeed() {
    const container = document.getElementById('posts-container');
    if (!container) return;
    const query = searchQuery.trim().toLowerCase();
    const posts = feed;
    document.getElementById('feedCount').textContent = `${feedTotal} discussion${feedTotal === 1 ? '' : 's'}`;
    document.querySelectorAll('[data-topic]').forEach((button) => {
        button.classList.toggle('active', button.dataset.topic === activeTopic);
        button.setAttribute('aria-pressed', String(button.dataset.topic === activeTopic));
    });
    const summary = document.getElementById('searchSummary');
    summary.hidden = !query;
    summary.textContent = `Results for “${searchQuery.trim()}” across discussions`;
    container.innerHTML = posts.length
        ? posts
              .map(
                  (post) => `
        <article class="post-card" data-post-id="${Number(post.id)}">
            <div class="post-author"><span class="avatar avatar-soft tone-${Number(post.id) % 4}">${esc(initials(post.username))}</span><div><strong>${esc(post.username)}</strong><span>${esc(dateLabel(post.created_at))} <span class="meta-dot">·</span> Shared a thought</span></div><span class="post-category">${esc(post.categories?.[0] || 'Discussion')}</span></div>
            <h3><a href="/post/${Number(post.id)}" data-route>${esc(post.title)}</a></h3>
            <p class="post-preview">${esc(post.content)}</p>${attachmentHTML(post)}
            <div class="post-footer"><div class="post-reactions"><button class="reaction-btn" data-like="${Number(post.id)}" aria-label="Like discussion">${icon('up')}<span class="likes-count">${post.likes || 0}</span></button><button class="reaction-btn" data-dislike="${Number(post.id)}" aria-label="Dislike discussion">${icon('down')}<span class="dislikes-count">${post.dislikes || 0}</span></button></div><a class="text-link" href="/post/${Number(post.id)}" data-route>Join discussion ${icon('arrow')}</a></div>
        </article>`,
              )
              .join('')
        : emptyState(
              query || activeTopic ? 'No conversations here yet.' : 'Every community starts with a hello.',
              query || activeTopic
                  ? 'Try another topic or search, or start a discussion of your own.'
                  : 'Share a question, an idea, or something you’ve been thinking about.',
          ) +
          '<a href="/create_post" data-route class="button button-dark empty-cta">Start a discussion ' +
          icon('plus') +
          '</a>';
}

export async function refreshFeed(reset = false) {
    const container = document.getElementById('posts-container');
    if (!container) return;
    if (reset) feedOffset = 0;
    const request = ++feedRequest;
    const parameters = new URLSearchParams({
        q: searchQuery.trim(),
        category: activeTopic,
        sort: sortOrder,
        limit: '10',
        offset: String(feedOffset),
    });
    container.setAttribute('aria-busy', 'true');
    try {
        const result = await api(`/posts?${parameters}`);
        if (request !== feedRequest || document.getElementById('posts-container') !== container) return;
        feed = result.posts;
        feedTotal = result.total;
        if (feedOffset >= feedTotal && feedOffset > 0) {
            feedOffset = Math.max(0, Math.ceil(feedTotal / 10) - 1) * 10;
            return refreshFeed();
        }
        renderFeed();
        const pagination = document.getElementById('feedPagination');
        pagination.innerHTML = `<button type="button" class="button button-quiet" id="previousPage" ${feedOffset === 0 ? 'disabled' : ''}>Previous</button><span>Page ${Math.floor(feedOffset / 10) + 1} of ${Math.max(1, Math.ceil(feedTotal / 10))}</span><button type="button" class="button button-quiet" id="nextPage" ${feedOffset + 10 >= feedTotal ? 'disabled' : ''}>Next</button>`;
        document.getElementById('previousPage').addEventListener('click', () => {
            feedOffset = Math.max(0, feedOffset - 10);
            refreshFeed();
        });
        document.getElementById('nextPage').addEventListener('click', () => {
            feedOffset += 10;
            refreshFeed();
        });
    } catch (error) {
        if (request !== feedRequest || document.getElementById('posts-container') !== container) return;
        container.innerHTML =
            emptyState('Could not load discussions.', error.message) +
            '<button type="button" class="button button-quiet" id="retryFeed">Try again</button>';
        document.getElementById('feedPagination').innerHTML = '';
        document.getElementById('retryFeed').addEventListener('click', () => refreshFeed());
    } finally {
        if (request === feedRequest && container.isConnected) container.removeAttribute('aria-busy');
    }
}

function selectTopic(topic) {
    activeTopic = topic;
    if (window.location.pathname !== '/') navigate('/');
    else refreshFeed(true);
}

export async function refreshCurrentPage() {
    if (document.getElementById('posts-container')) return refreshFeed();
    const id = window.location.pathname.match(/^\/post\/(\d+)$/)?.[1];
    const article = document.getElementById('post-content');
    if (!id || !article) return;
    try {
        const post = await api(`/post/${id}`);
        if (document.getElementById('post-content') !== article) return;
        const likes = article.querySelector('.likes-count');
        const dislikes = article.querySelector('.dislikes-count');
        article.querySelector('h1').textContent = post.title;
        article.querySelector('.post-content').textContent = post.content;
        const state = article.querySelector('[data-review-status]');
        if (state) {
            state.hidden = post.status !== 'pending';
        }
        const attachment = article.querySelector('.post-attachment');
        if (attachment) attachment.innerHTML = attachmentHTML(post);
        const topics = article.querySelector('.post-categories');
        if (topics)
            topics.innerHTML = (post.categories || [])
                .map((topic) => `<span class="category">${esc(topic)}</span>`)
                .join('');
        if (likes) likes.textContent = post.likes;
        if (dislikes) dislikes.textContent = post.dislikes;
        await refreshComments(id);
    } catch (error) {
        if (document.getElementById('post-content') !== article) return;
        if (error.status === 404) {
            showToast('This discussion has been removed.');
            navigate('/');
        } else showToast(error.message);
    }
}

function pageHeading(kicker, title, detail) {
    return `<div class="page-heading"><span class="eyebrow">${kicker}</span><h1>${title}</h1><p>${detail}</p></div>`;
}

async function createPostContent(version) {
    document.getElementById('content').innerHTML =
        pageHeading(
            'SOMETHING ON YOUR MIND?',
            'Start a conversation.',
            'A question, a discovery, a different perspective. It all belongs here.',
        ) +
        `
        <section class="form-panel"><form id="createPostForm"><label for="title">Give it a title <span class="field-hint">200 characters max</span></label><input id="title" name="title" maxlength="200" placeholder="What would you like to talk about?" required><label for="postContent">Your perspective <span class="field-hint">2,000 characters max</span></label><textarea id="postContent" name="content" maxlength="2000" rows="7" placeholder="Tell us more. A little context goes a long way…" required></textarea><label for="image">Optional image</label><input id="image" name="image" type="file" accept="image/jpeg,image/png,image/gif"><p class="field-help">JPEG, PNG, GIF. Maximum 20 MiB (20,971,520 bytes), 40 million pixels.</p><label for="categories">Find its corner</label><p class="field-help">Choose one or more topics. Hold Ctrl or ⌘ to select multiple.</p><select id="categories" name="categories" multiple required aria-describedby="categoryHelp"></select><span id="categoryHelp" class="field-help">Choose at least one topic.</span><div class="form-footer"><a href="/" data-route class="button button-quiet">Cancel</a><button class="button button-dark" type="submit">Publish discussion ${icon('arrow')}</button></div></form></section>`;
    const form = document.getElementById('createPostForm');
    form.addEventListener('submit', handleCreatePost);
    try {
        const categories = await api('/categories');
        if (version !== renderVersion) return;
        document.getElementById('categories').innerHTML = (categories || [])
            .map((category) => `<option value="${esc(category)}">${esc(category)}</option>`)
            .join('');
    } catch (error) {
        if (version === renderVersion) displayErrors([error.message]);
    }
}

async function singlePostContent(postId, version) {
    document.getElementById('content').innerHTML =
        '<a href="/" data-route class="back-link">' +
        icon('back') +
        ' Back to discussions</a><article id="post-content" class="post-detail"><div class="skeleton-card"></div></article><div id="comments-section"></div>';
    document.getElementById('post-content').dataset.postId = postId;
    try {
        const post = await api(`/post/${postId}`);
        if (version !== renderVersion) return;
        const authenticated = Boolean(localStorage.getItem('userId'));
        document.getElementById('post-content').innerHTML =
            `<div class="post-author"><span class="avatar avatar-soft">${esc(initials(post.username))}</span><div><strong>${esc(post.username)}</strong><span>${esc(dateLabel(post.created_at))}</span></div></div><h1>${esc(post.title)}</h1><div class="post-content">${esc(post.content)}</div><div class="post-attachment">${attachmentHTML(post)}</div><div class="post-categories">${(post.categories || []).map((category) => `<span class="category">${esc(category)}</span>`).join('')}</div><div class="post-footer"><div class="post-reactions"><button class="reaction-btn" data-like="${Number(post.id)}" aria-label="Like discussion">${icon('up')}<span class="likes-count">${post.likes || 0}</span></button><button class="reaction-btn" data-dislike="${Number(post.id)}" aria-label="Dislike discussion">${icon('down')}<span class="dislikes-count">${post.dislikes || 0}</span></button></div>${Number(post.user_id) === Number(localStorage.getItem('userId')) || ['moderator', 'admin'].includes(localStorage.getItem('role')) ? `<button class="button button-quiet delete-btn" id="deleteDiscussion">${icon('trash')} Delete</button>` : ''}</div>`;
        if (Number(post.user_id) === Number(localStorage.getItem('userId'))) {
            const edit = document.createElement('button');
            edit.type = 'button';
            edit.className = 'button button-quiet';
            edit.textContent = 'Edit discussion';
            edit.onclick = () => editPost(post.id);
            document.getElementById('post-content').append(edit);
        }
        {
            const status = document.createElement('p');
            status.textContent = 'Awaiting review';
            status.dataset.reviewStatus = '';
            status.hidden = post.status !== 'pending';
            document.getElementById('post-content').append(status);
        }
        if (authenticated) {
            const report = document.createElement('button');
            report.type = 'button';
            report.className = 'button button-quiet';
            report.textContent = 'Report discussion';
            report.addEventListener('click', () => reportPost(post.id));
            document.getElementById('post-content').append(report);
        }
        document.getElementById('deleteDiscussion')?.addEventListener('click', (event) => deletePost(post.id, event));
        document.getElementById('comments-section').innerHTML = renderCommentSection(authenticated);
        initializeComments(postId);
    } catch (error) {
        if (version === renderVersion)
            document.getElementById('post-content').innerHTML = emptyState(
                'Couldn’t open this discussion.',
                error.message,
            );
    }
}

function profileContent(userId) {
    document.getElementById('content').innerHTML =
        pageHeading('THE PERSON BEHIND THE PERSPECTIVE', 'A little about you.', 'Your own corner of the common room.') +
        `<section class="profile-container"><div class="profile-header"><div id="profileAvatar" class="profile-avatar">…</div><div><h2 id="fullName">Loading profile…</h2><p id="username" class="muted"></p></div></div><div class="profile-details"><div class="detail-row"><span>Email address</span><strong id="profileEmail"></strong></div><div class="detail-row"><span>Age</span><strong id="profileAge"></strong></div><div class="detail-row"><span>Gender</span><strong id="profileGender"></strong></div></div></section>`;
    loadProfileData(userId);
}

function chatContent() {
    document.getElementById('content').innerHTML =
        pageHeading(
            'MAKE A CONNECTION',
            'A conversation, just for you.',
            'Select someone from the community to say hello.',
        ) +
        `<section class="chat-main"><div class="chat-heading"><span class="avatar avatar-soft" id="chatAvatar">${icon('chat')}</span><div><h2 id="selectedUserName">Your messages</h2><span id="chatStatus" class="muted">Choose a person to start</span></div></div><div class="chat-messages" id="messageHistory">${emptyState('Good things start with hello.', 'Pick someone from the people panel to open your conversation.')}</div><div id="typingIndicator" class="typing-indicator" role="status"></div><form id="messageForm" class="chat-input" hidden><label class="sr-only" for="messageInput">Your message</label><input id="messageInput" placeholder="Say something kind…" autocomplete="off" maxlength="2000" required><button type="submit" class="button button-dark" aria-label="Send message">${icon('send')}</button></form></section>`;
    const badge = document.getElementById('message-badge');
    badge.hidden = true;
    badge.textContent = '0';
    initializeChat(window.wsClient);
    if (window.wsClient) updateUsersList(window.wsClient, window.wsClient.users || []);
}

function authLayout(form, registering = false) {
    document.getElementById('content').innerHTML =
        `<div class="auth-layout"><section class="auth-story"><span class="eyebrow">WELCOME TO YAPLANE</span><h1>Good company.<br><em>Better conversations.</em></h1><p>A space for the endlessly curious.<br>And a seat with your name on it.</p>${sculpture()}<div class="auth-story-footer"><span class="tiny-star">✳</span><span>A little space for big ideas.</span></div></section><section class="auth-form-wrap"><span class="eyebrow">${registering ? 'YOUR NEXT CONVERSATION STARTS HERE' : 'YOUR SEAT IS STILL HERE'}</span><h2>${registering ? 'Find your people.' : 'Welcome back.'}</h2><p>${registering ? 'Join the common room. Bring your perspective.' : 'Pick up where the conversation left off.'}</p>${form}</section></div>`;
}

function loginContent() {
    authLayout(
        `<form id="loginForm"><label for="username">Username or email</label><input id="username" name="username" autocomplete="username" placeholder="you@example.com" required><label for="password">Password</label><input type="password" id="password" name="password" autocomplete="current-password" placeholder="Your password" required><button type="submit" class="button button-dark full-width">Come on in ${icon('arrow')}</button></form><p class="auth-switch">New around here? <a href="/register" data-route>Join the community</a></p>`,
    );
    document.getElementById('loginForm').addEventListener('submit', handleLoginSubmit);
}

function registerContent() {
    authLayout(
        `<form id="registerForm"><div class="form-grid"><div><label for="first_name">First name</label><input id="first_name" name="first_name" autocomplete="given-name" required></div><div><label for="last_name">Last name</label><input id="last_name" name="last_name" autocomplete="family-name" required></div></div><label for="username">Username</label><input id="username" name="username" autocomplete="username" required><label for="email">Email address</label><input id="email" name="email" type="email" autocomplete="email" required><label for="password">Password</label><input id="password" name="password" type="password" autocomplete="new-password" minlength="8" maxlength="72" required><span class="field-help">At least 8 characters; at most 72 UTF-8 bytes.</span><label for="password_confirmation">Confirm password</label><input id="password_confirmation" name="password_confirmation" type="password" autocomplete="new-password" required><div class="form-grid"><div><label for="age">Age</label><input id="age" name="age" type="number" min="1" max="120" required></div><fieldset class="gender-group"><legend>Gender</legend><div class="gender-options"><label><input type="radio" name="gender" value="male" required> Male</label><label><input type="radio" name="gender" value="female" required> Female</label></div></fieldset></div><button type="submit" class="button button-dark full-width">Make yourself at home ${icon('arrow')}</button></form><p class="auth-switch">Already part of the room? <a href="/login" data-route>Log in</a></p>`,
        true,
    );
    document.getElementById('registerForm').addEventListener('submit', handleRegisterSubmit);
}

async function logout() {
    try {
        await api('/logout', { method: 'POST' });
        window.wsClient?.disconnect();
        ['sessionToken', 'userId', 'username', 'isAdmin', 'role'].forEach((key) => localStorage.removeItem(key));
        window.wsClient?.connect();
        navigate('/');
    } catch (error) {
        showToast(error.message);
    }
}

export function displayErrors(errors) {
    const form = document.querySelector('#content form');
    if (!form) return;
    form.querySelector('.error-messages')?.remove();
    const node = document.createElement('div');
    node.className = 'error-messages';
    node.setAttribute('role', 'alert');
    node.innerHTML = errors.map((error) => `<p>${esc(error)}</p>`).join('');
    form.prepend(node);
}

export function initRouter() {
    document.addEventListener('click', (event) => {
        const topic = event.target.closest('[data-topic]');
        if (topic) {
            selectTopic(topic.dataset.topic);
            return;
        }
        const reaction = event.target.closest('[data-like], [data-dislike]');
        if (reaction) {
            if (!localStorage.getItem('userId')) {
                navigate('/login');
                return;
            }
            handleLike(Number(reaction.dataset.like || reaction.dataset.dislike), Boolean(reaction.dataset.like));
            return;
        }
        const link = event.target.closest('a[data-route]');
        if (!link || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey || event.button !== 0) return;
        event.preventDefault();
        navigate(link.getAttribute('href'));
    });
    window.addEventListener('popstate', () => renderContent(window.location.pathname));
    const search = document.getElementById('globalSearch');
    search.addEventListener('input', () => {
        searchQuery = search.value;
        clearTimeout(searchTimer);
        searchTimer = setTimeout(() => {
            if (window.location.pathname !== '/') navigate('/');
            else refreshFeed(true);
        }, 250);
    });
    document.addEventListener('keydown', (event) => {
        if (event.key === '/' && !event.target.closest('input, textarea, select, [contenteditable]')) {
            event.preventDefault();
            search.focus();
        }
    });
    window.addEventListener('contentedited', () => {
        if (window.location.pathname === '/activity') renderActivity();
        else refreshCurrentPage();
    });
    renderContent(window.location.pathname);
    refreshNotifications();
    refreshTopics().catch(() => {
        document.getElementById('topicNavigation').innerHTML = '<p class="muted">Topics are unavailable.</p>';
    });
}

function attachmentHTML(post) {
    return post.image_path
        ? `<img class="post-image" src="/uploads/${encodeURIComponent(post.image_path)}" alt="Discussion attachment" loading="lazy">`
        : '';
}

function completeProfileContent() {
    document.getElementById('content').innerHTML =
        pageHeading(
            'FINISH YOUR ACCOUNT',
            'Tell us about yourself.',
            'Your verified provider email is retained. Enter your required forum profile details.',
        ) +
        `<section class="form-panel"><form id="completionForm"><label>Username<input name="username" required minlength="3" maxlength="32"></label><label>First name<input name="first_name" required maxlength="80"></label><label>Last name<input name="last_name" required maxlength="80"></label><label>Age<input name="age" type="number" min="1" max="120" required></label><label>Gender<select name="gender" required><option value="">Choose</option><option value="male">Male</option><option value="female">Female</option></select></label><button class="button button-dark" type="submit">Complete registration</button></form></section>`;
    document.getElementById('completionForm').addEventListener('submit', async (event) => {
        event.preventDefault();
        const form = event.currentTarget;
        const input = Object.fromEntries(new FormData(form));
        input.age = Number(input.age);
        const button = form.querySelector('button');
        button.disabled = true;
        try {
            await api('/complete-profile', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(input),
            });
            window.location.assign('/');
        } catch (error) {
            displayErrors([error.message]);
            button.disabled = false;
        }
    });
}

export async function refreshTopics() {
    const topics = await api('/categories');
    document.getElementById('topicNavigation').innerHTML = topics
        .map(
            (topic, index) =>
                `<button class="topic-button" data-topic="${esc(topic)}"><span class="topic-dot tone-${index % 4}"></span>${esc(topic)}</button>`,
        )
        .join('');
    if (activeTopic && !topics.includes(activeTopic)) activeTopic = '';
    const filters = document.getElementById('topicFilters');
    if (filters)
        filters.innerHTML = ['', ...topics]
            .map(
                (topic) =>
                    `<button type="button" class="filter-tab" data-topic="${esc(topic)}" aria-pressed="${topic === activeTopic}">${esc(topic || 'All discussions')}</button>`,
            )
            .join('');
    const select = document.querySelector('#createPostForm #categories');
    if (select) {
        const selected = Array.from(select.selectedOptions, (option) => option.value);
        select.innerHTML = topics
            .map(
                (topic) =>
                    `<option value="${esc(topic)}" ${selected.includes(topic) ? 'selected' : ''}>${esc(topic)}</option>`,
            )
            .join('');
    }
}
