document.getElementById('createPostForm').addEventListener('submit', async (event) => {
    event.preventDefault();

    const postData = {
        title: document.getElementById('postTitle').value.trim(),
        content: document.getElementById('postContent').value.trim(),
        imagePath: document.getElementById('postImagePath').value.trim(),
        categoryIDs: Array.from(document.getElementById('postCategories').selectedOptions).map(option => option.value),
        userID: localStorage.getItem('userId')
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

        if (!response.ok) {
            const errorText = await response.text();
            console.error('Error creating post:', errorText);
            return;
        }

        const result = await response.json();
        console.log('Post created successfully:', result);
    } catch (error) {
        console.error('Error creating post:', error);
    }
});

async function loadCategories() {
    try {
        const response = await fetch('/categories');
        const categories = await response.json();
        const categoriesSelect = document.getElementById('postCategories');
        categories.forEach(category => {
            const option = document.createElement('option');
            option.value = category.id;
            option.textContent = category.name;
            categoriesSelect.appendChild(option);
        });
    } catch (error) {
        console.error('Error loading categories:', error);
    }
}

async function loadPosts() {
    try {
        const response = await fetch('/posts');
        const posts = await response.json();
        const postsContainer = document.getElementById('postsContainer');
        postsContainer.innerHTML = '';
        posts.forEach(post => {
            const postElement = document.createElement('div');
            postElement.className = 'post';
            postElement.innerHTML = `
                <h3>${post.title}</h3>
                <p>${post.content}</p>
                <p>Categories: ${post.categories.map(category => category.name).join(', ')}</p>
                <p>Posted by User ${post.user_id} on ${new Date(post.created_at).toLocaleString()}</p>
            `;
            postsContainer.appendChild(postElement);
        });
    } catch (error) {
        console.error('Error loading posts:', error);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    loadCategories();
    loadPosts();
});
