export async function deletePost(postId, event) {
    event.stopPropagation(); 
    
    if (!confirm('Are you sure you want to delete this post?')) {
        return;
    }

    try {
        const response = await fetch('/delete-post', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
                'Authorization': localStorage.getItem('sessionToken')
            },
            body: `post_id=${postId}`
        });

        if (response.ok) {
            if (window.location.pathname.startsWith('/post/')) {
                window.history.pushState({}, '', '/');
                window.dispatchEvent(new PopStateEvent('popstate'));
            } else {
                const postElement = document.querySelector(`[data-post-id="${postId}"]`).parentElement;
                postElement.remove();
            }
        } else {
            console.error('Failed to delete post');
        }
    } catch (error) {
        console.error('Error deleting post:', error);
    }
}
