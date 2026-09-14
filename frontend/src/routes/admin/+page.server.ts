import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { getAdminStats } from '$lib/services/api';

export const load: PageServerLoad = async ({ locals, cookies }) => {
    if (!locals.user?.isAdmin) {
        throw redirect(302, '/');
    }

    const token = cookies.get('partipix_auth_token');
    if (!token) {
        throw redirect(302, '/signin');
    }

    try {
        // Pass auth headers from the server context
        await getAdminStats({
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
        });
    } catch (e) {
        console.error('Admin validation failed:', e);
        throw redirect(302, '/signin');
    }

    return {};
};