import { writable } from 'svelte/store';
import type { User, Event, Photo } from './types';

const isBrowser = typeof window !== 'undefined';

function createAuthStore() {
    const { subscribe, set } = writable<User | null>(null);

    return {
        subscribe,
        set: (user: User | null) => {
            set(user);
            if (isBrowser) {
                if (user) {
                    localStorage.setItem('user', JSON.stringify(user));
                } else {
                    localStorage.removeItem('user');
                }
            }
        },
        initialize: () => {
            if (isBrowser) {
                const storedUser = localStorage.getItem('user');
                if (storedUser) {
                    try {
                        const user = JSON.parse(storedUser);
                        set(user);
                    } catch (e) {
                        console.error('Failed to parse stored user:', e);
                        localStorage.removeItem('user');
                    }
                }
            }
        }
    };
}

export const user = createAuthStore();
export const events = writable<Event[]>([]);
export const photos = writable<Photo[]>([]);

// Helper functions for managing state
export function addEvent(event: Event) {
    events.update(current => [...current, event]);
}

export function addPhoto(photo: Photo) {
    photos.update(current => [...current, photo]);
}

// Initialize auth state on app load
user.initialize();
if (isBrowser) {
    // Refresh user profile from server if we have a stored user
    const storedUser = localStorage.getItem('user');
    if (storedUser) {
        import('./services/api').then(api => {
            api.getUserProfile().catch(console.error);
        });
    }
}