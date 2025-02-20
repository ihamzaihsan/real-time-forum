import { renderContent } from './router.js';

export async function handleCreatePost(e) {
    e.preventDefault();
    let title = document.getElementById('title').value.trim();
    let content = document.getElementById('postContent').value.trim();

    // Remove < and > characters from title and content
    title = title.replace(/[<>]/g, '');
    content = content.replace(/[<>]/g, '');

    const categoriesSelect = document.getElementById('categories');
    const categories = Array.from(categoriesSelect.selectedOptions).map(option => option.value);

    // Validate categories
    if (categories.length === 0) {
        alert('Please select at least one category');
        return;
    }

    const postData = {
        title,
        content,
        categories
    };

    try {
        const response = await fetch('/create_post', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': localStorage.getItem('sessionToken')
            },
            body: JSON.stringify(postData)
        });

        if (response.ok) {
            window.history.pushState({}, '', '/');
            renderContent('/');
        } else {
            const errorData = await response.text();
            alert('Error creating post: ' + errorData);
        }
    } catch (error) {
        console.error('Error creating post:', error);
        alert('Error creating post: ' + error.message);
    }
}