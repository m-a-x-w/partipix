import type { Handle } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
    // Get auth token from cookies
    const token = event.cookies.get('partipix_auth_token');

    if (token) {
        try {
            // Use the token to fetch user profile
            const headers = new Headers({
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            });

            const response = await fetch('http://localhost:8080/api/users/profile', {
                headers,
                credentials: 'include'
            });

            if (response.ok) {
                const userData = await response.json();
                // Ensure we explicitly set isAdmin to false if it's not present
                event.locals.user = {
                    id: userData.id,
                    name: userData.name,
                    email: userData.email,
                    isAdmin: !!userData.isAdmin
                };
            } else {
                // If the token is invalid or expired, clear it
                event.cookies.delete('partipix_auth_token', { path: '/' });
            }
        } catch (e) {
            console.error('Failed to validate session:', e);
            // Clear the cookie on error
            event.cookies.delete('partipix_auth_token', { path: '/' });
        }
    }

    const response = await resolve(event);
    return response;
};