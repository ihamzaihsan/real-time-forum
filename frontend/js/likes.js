export function handleLike(postId, isLike) {
    const formData = new FormData();
    formData.append('post_id', postId);
    formData.append('is_like', isLike);

    fetch('/like', {
        method: 'POST',
        headers: {
            'Authorization': localStorage.getItem('sessionToken')
        },
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        
        const postElements = document.querySelectorAll(`[data-post-id="${postId}"]`);
        postElements.forEach(element => {
            const likesCount = element.querySelector('.likes-count');
            const dislikesCount = element.querySelector('.dislikes-count');
            if (likesCount) likesCount.textContent = data.likes;
            if (dislikesCount) dislikesCount.textContent = data.dislikes;
        });

        
        const postContent = document.getElementById('post-content');
        if (postContent) {
            const likesCount = postContent.querySelector('.likes-count');
            const dislikesCount = postContent.querySelector('.dislikes-count');
            if (likesCount) likesCount.textContent = data.likes;
            if (dislikesCount) dislikesCount.textContent = data.dislikes;
        }
    })
    .catch(error => console.error('Error:', error));
}

window.handleLike = handleLike;

export function handleCommentLike(commentId, isLike) {
    const formData = new FormData();
    formData.append('comment_id', commentId);
    formData.append('is_like', isLike);

    fetch('/comment/like', {
        method: 'POST',
        headers: {
            'Authorization': localStorage.getItem('sessionToken')
        },
        body: formData
    })
    .then(response => {
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
    })
    .then(data => {
        const commentElement = document.querySelector(`[data-comment-id="${commentId}"]`);
        if (commentElement) {
            const likesCount = commentElement.querySelector('.likes-count');
            const dislikesCount = commentElement.querySelector('.dislikes-count');
            
            if (likesCount) likesCount.textContent = data.likes;
            if (dislikesCount) dislikesCount.textContent = data.dislikes;
        }
    })
    .catch(error => {
        console.error('Like/Dislike operation failed:', error);
    });
}

window.handleCommentLike = handleCommentLike;
