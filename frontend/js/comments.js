export async function loadComments(postId) {
    try {
        const response = await fetch(`/comments?post_id=${postId}`);
        const comments = await response.json();
        return comments;
    } catch (error) {
        console.error('Error loading comments:', error);
        return [];
    }
}
export async function createComment(postId, content) {
    try {
        const response = await fetch('/comment', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': localStorage.getItem('sessionToken')
            },
            body: JSON.stringify({
                post_id: parseInt(postId),
                content: content,
                username: localStorage.getItem('username')
            })
        });

        if (!response.ok) {
            const errorText = await response.text();
            console.log('Server error:', errorText);
            throw new Error('Failed to create comment');
        }

        return await response.json();
    } catch (error) {
        console.error('Error creating comment:', error);
        throw error;
    }
}
export function renderCommentSection(postId) {
    return `
        <div class="comments-section">
            <h3>Comments</h3>
            <form id="commentForm" class="comment-form">
                <textarea id="commentContent" placeholder="Write your comment..." required></textarea>
                <button type="submit">Post Comment</button>
            </form>
            <div id="commentsList" class="comments-list"></div>
        </div>
    `;
}

export function initializeComments(postId) {
    const commentForm = document.getElementById('commentForm');
    const commentsList = document.getElementById('commentsList');

    // Load existing comments first
    loadComments(postId).then(comments => {
        renderComments(comments);

        // Then set up the form handler
        commentForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const content = document.getElementById('commentContent').value.trim();
            if (!content) return;

            try {
                const newComment = await createComment(postId, content);
                // Get current comments and add new one
                const currentComments = await loadComments(postId);
                renderComments(currentComments);
                document.getElementById('commentContent').value = '';
            } catch (error) {
                console.error('Failed to post comment:', error);
            }
        });
    });
}

export function renderComments(comments) {
    const commentsList = document.getElementById('commentsList');
    if (!commentsList) return;
    
    const commentsArray = Array.isArray(comments) ? comments : [];
    
    commentsList.innerHTML = commentsArray.map(comment => {
        const timestamp = comment.created_at ? new Date(comment.created_at).toLocaleString() : 'Just now';
        const likes = typeof comment.likes === 'number' ? comment.likes : 0;
        const dislikes = typeof comment.dislikes === 'number' ? comment.dislikes : 0;
        
        return `
            <div class="comment" data-comment-id="${comment.id || ''}">
                <div class="comment-header">
                    <span class="comment-author">${comment.username || 'Anonymous'}</span>
                    <span class="comment-date">${timestamp}</span>
                </div>
                <div class="comment-content">${comment.content || ''}</div>
                <div class="comment-actions">
                    <button onclick="handleCommentLike(${comment.id}, true)" class="like-btn">
                        👍 <span class="likes-count">${likes}</span>
                    </button>
                    <button onclick="handleCommentLike(${comment.id}, false)" class="dislike-btn">
                        👎 <span class="dislikes-count">${dislikes}</span>
                    </button>
                </div>
            </div>
        `;
    }).join('');
}

