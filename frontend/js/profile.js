export async function loadProfileData() {
    try {
        const response = await fetch('/profile', {
            headers: {
                'Authorization': localStorage.getItem('sessionToken')
            }
        });
        
        if (!response.ok) {
            throw new Error('Failed to fetch profile data');
        }

        const profileData = await response.json();
        
        // Set avatar initials
        const initials = `${profileData.first_name[0]}${profileData.last_name[0]}`.toUpperCase();
        document.getElementById('profileAvatar').textContent = initials;
        
        // Set profile information
        document.getElementById('fullName').textContent = 
            `${profileData.first_name} ${profileData.last_name}`;
        document.getElementById('username').textContent = profileData.username;
        document.getElementById('profileEmail').textContent = profileData.email;
        document.getElementById('profileAge').textContent = profileData.age;
        document.getElementById('profileGender').textContent = profileData.gender;
    } catch (error) {
        console.error('Error loading profile:', error);
    }
}
