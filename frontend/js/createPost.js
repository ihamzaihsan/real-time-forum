import { renderContent } from './router.js';

export async function handleCreatePost(e) {
    e.preventDefault();
    const title = document.getElementById('title').value;
    const content = document.getElementById('postContent').value.trim();
    console.log(content);
    const categoriesSelect = document.getElementById('categories');
    const categories = Array.from(categoriesSelect.selectedOptions).map(option => option.value);

    const postData = {
        title,
        content,
        categories,
        username: localStorage.getItem('username')
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
        }
    } catch (error) {
        console.error('Error creating post:', error);
    }
}