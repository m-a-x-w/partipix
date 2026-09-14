import type { User, Event, Photo, AdminStats, UpdateUserRequest, ReferencePhoto } from '../types';
import { user } from '../stores';

const API_BASE_URL = 'http://localhost:8080/api';
const BACKEND_URL = 'http://localhost:8080';  // Add this line
const TOKEN_KEY = 'partipix_auth_token';
const isBrowser = typeof window !== 'undefined';

// Add helper function to handle file URLs
export function getFileUrl(url: string): string {
    // If the URL already starts with http, return it as is
    if (url.startsWith('http')) {
        return url;
    }

    // Don't add /api prefix for face detection URLs and temp files
    if (url.startsWith('/tmp/')) {
        return `${BACKEND_URL}${url}`;
    }

    // For photos that already have /uploads prefix, don't add it again
    if (url.startsWith('/uploads/')) {
        return `${BACKEND_URL}${url}`;
    }

    // For all other URLs (pfps, etc), add /uploads/ prefix
    return `${BACKEND_URL}/uploads/${url.startsWith('/') ? url.slice(1) : url}`;
}

// Request interfaces
export interface SignInRequest {
    email: string;
    password: string;
}

export interface SignUpRequest {
    name: string;
    email: string;
    password: string;
}

export interface CreateEventRequest {
    title: string;
    description: string;
    date: string;
    location?: string;
}

export interface JoinEventRequest {
    code: string;
}

export interface DetectedFace {
    id: string;
    url: string;
    originalPhotoId: string;
}

// Initialize auth token from localStorage if available
let authToken: string | null = isBrowser ? localStorage.getItem(TOKEN_KEY) : null;
let isRefreshing = false;
let refreshPromise: Promise<void> | null = null;

// Helper function to set auth token and update user store
function setAuthToken(token: string | null, userData: User | null) {
    authToken = token;
    if (isBrowser) {
        if (token) {
            localStorage.setItem(TOKEN_KEY, token);
            // Set cookie with HttpOnly flag - this will be accessible to the server
            document.cookie = `${TOKEN_KEY}=${token}; path=/; SameSite=Strict; Secure`;
        } else {
            localStorage.removeItem(TOKEN_KEY);
            // Clear the cookie
            document.cookie = `${TOKEN_KEY}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT; SameSite=Strict; Secure`;
        }
    }
    user.set(userData);
}

// Helper function to get auth headers
function getAuthHeaders(): HeadersInit {
    const headers: HeadersInit = {
        'Content-Type': 'application/json'
    };
    
    if (authToken) {
        headers['Authorization'] = `Bearer ${authToken}`;
    }
    
    return headers;
}

// Helper function to handle API responses
async function handleResponse<T>(response: Response, requestMethod?: string, requestBody?: BodyInit): Promise<T> {
    if (response.status === 401) {
        // Token might be expired, try to refresh
        if (!isRefreshing) {
            isRefreshing = true;
            refreshPromise = refreshToken().catch(() => {
                // If refresh fails, sign out
                signOut();
            }).finally(() => {
                isRefreshing = false;
                refreshPromise = null;
            });
        }

        if (refreshPromise) {
            await refreshPromise;
            // Retry the original request with the new token
            const retryResponse = await fetch(response.url, {
                method: requestMethod || 'GET',
                headers: getAuthHeaders(),
                body: requestMethod !== 'GET' ? requestBody : undefined,
                credentials: 'include'
            });
            
            if (!retryResponse.ok) {
                throw new Error(await retryResponse.text());
            }
            
            return retryResponse.json();
        }
    }
    
    if (!response.ok) {
        throw new Error(await response.text());
    }
    
    const data = await response.json();
    
    // Add backend URL to any image URLs in the response
    if (data) {
        if (Array.isArray(data)) {
            data.forEach(item => {
                if (item.url) {
                    item.url = getFileUrl(item.url);
                }
            });
        } else if (data.url) {
            data.url = getFileUrl(data.url);
        } else if (data.referencePhotos) {
            data.referencePhotos.forEach((photo: any) => {
                if (photo.url) {
                    photo.url = getFileUrl(photo.url);
                }
            });
        }
    }
    
    return data;
}

// Refresh token function
async function refreshToken(): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
        method: 'POST',
        headers: getAuthHeaders(),
        credentials: 'include'
    });
    
    if (!response.ok) {
        throw new Error('Failed to refresh token');
    }
    
    const data = await response.json();
    if (!data.token || !data.user) {
        throw new Error('Invalid response format');
    }
    
    setAuthToken(data.token, { ...data.user, token: data.token });
}

// Auth functions
export async function signIn(credentials: SignInRequest): Promise<User> {
    const body = JSON.stringify(credentials);
    const response = await fetch(`${API_BASE_URL}/auth/login`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        credentials: 'include',
        body
    });
    
    const userData = await handleResponse<User>(response, 'POST', body);
    // Set the auth token before returning the user data
    setAuthToken(userData.token, userData);
    return userData;
}

export async function signUp(userData: SignUpRequest): Promise<User> {
    const response = await fetch(`${API_BASE_URL}/auth/signup`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        credentials: 'include',
        body: JSON.stringify(userData)
    });
    
    const user = await handleResponse<User>(response);
    // Set the auth token before returning the user data
    setAuthToken(user.token, user);
    return user;
}

export async function signOut(): Promise<void> {
    try {
        const response = await fetch(`${API_BASE_URL}/auth/signout`, {
            method: 'POST',
            headers: getAuthHeaders(),
            credentials: 'include'
        });
        
        if (!response.ok) {
            console.error('Failed to sign out cleanly');
        }
    } catch (e) {
        console.error('Error during signout:', e);
    } finally {
        setAuthToken(null, null);
    }
}

// User profile functions
export async function getUserProfile(): Promise<User> {
    const response = await fetch(`${API_BASE_URL}/users/profile`, {
        headers: getAuthHeaders(),
        credentials: 'include'
    });
    
    return handleResponse<User>(response);
}

export async function updateUserProfile(updates: Partial<User>): Promise<User> {
    const body = JSON.stringify(updates);
    const response = await fetch(`${API_BASE_URL}/users/profile`, {
        method: 'PUT',
        headers: getAuthHeaders(),
        credentials: 'include',
        body
    });
    
    return handleResponse<User>(response, 'PUT', body);
}

// Event functions
export async function getEvents(): Promise<Event[]> {
    const response = await fetch(`${API_BASE_URL}/events`, {
        headers: getAuthHeaders(),
        credentials: 'include'
    });
    
    return handleResponse<Event[]>(response);
}

export async function getEventById(id: string): Promise<Event> {
    const response = await fetch(`${API_BASE_URL}/events/${id}`, {
        headers: getAuthHeaders(),
        credentials: 'include'
    });
    
    return handleResponse<Event>(response);
}

export async function createEvent(eventData: CreateEventRequest): Promise<Event> {
    const body = JSON.stringify(eventData);
    const response = await fetch(`${API_BASE_URL}/events`, {
        method: 'POST',
        headers: getAuthHeaders(),
        credentials: 'include',
        body
    });
    
    return handleResponse<Event>(response, 'POST', body);
}

export async function joinEvent(joinData: JoinEventRequest): Promise<Event> {
    const body = JSON.stringify(joinData);
    const response = await fetch(`${API_BASE_URL}/events/join`, {
        method: 'POST',
        headers: getAuthHeaders(),
        credentials: 'include',
        body
    });
    
    return handleResponse<Event>(response, 'POST', body);
}

export async function updateGuestStatus(eventId: string, status: 'uploaded' | 'not_uploading' | 'will_upload'): Promise<Event> {
    const body = JSON.stringify({ status });
    const response = await fetch(`${API_BASE_URL}/events/${eventId}/status`, {
        method: 'PUT',
        headers: getAuthHeaders(),
        credentials: 'include',
        body
    });
    
    return handleResponse<Event>(response, 'PUT', body);
}

export async function deleteUserEvent(eventId: string): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/events/${eventId}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
        credentials: 'include'
    });

    if (!response.ok) {
        throw new Error('Failed to delete event');
    }
}

// Reference photo functions
export async function uploadReferencePhotos(files: File[]): Promise<ReferencePhoto[]> {
    const formData = new FormData();
    files.forEach(file => formData.append('photos', file));

    // Remove all headers and just use Authorization for multipart/form-data
    const headers: HeadersInit = {};
    if (authToken) {
        headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_BASE_URL}/users/reference-photos`, {
        method: 'POST',
        headers,
        credentials: 'include',
        body: formData
    });

    return handleResponse<ReferencePhoto[]>(response, 'POST', formData);
}

export async function uploadReferencePhotosForVerification(files: File[]): Promise<DetectedFace[]> {
    const formData = new FormData();
    files.forEach(file => formData.append('photos', file));

    const headers: HeadersInit = {};
    if (authToken) {
        headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_BASE_URL}/users/reference-photos/verify`, {
        method: 'POST',
        headers,
        credentials: 'include',
        body: formData
    });

    const faces = await handleResponse<DetectedFace[]>(response, 'POST', formData);
    // Add backend URL to face image URLs
    return faces.map(face => ({
        ...face,
        url: getFileUrl(face.url)
    }));
}

export async function confirmReferencePhoto(photoId: string, faceId: string): Promise<ReferencePhoto> {
    const response = await fetch(`${API_BASE_URL}/users/reference-photos/confirm`, {
        method: 'POST',
        headers: getAuthHeaders(),
        credentials: 'include',
        body: JSON.stringify({ 
            photoId: photoId,
            faceId: faceId  // This is the full ID of the face crop file (without .jpg)
        })
    });

    return handleResponse<ReferencePhoto>(response, 'POST');
}

export async function deleteReferencePhoto(photoId: string): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/users/reference-photos/${photoId}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
        credentials: 'include'
    });

    if (!response.ok) {
        const error = await response.json().catch(() => ({ error: 'Failed to delete photo' }));
        // If photo not found, still update the UI but don't throw
        if (response.status === 404) {
            return;
        }
        throw new Error(error.error || 'Failed to delete photo');
    }
}

// Add photo upload function
export async function uploadEventPhotos(eventId: string, files: File[], progressCallback?: (percent: number, stage: string) => void): Promise<Photo[]> {
    const formData = new FormData();
    formData.append('eventId', eventId);
    files.forEach(file => formData.append('photos', file));

    const headers: HeadersInit = {};
    if (authToken) {
        headers['Authorization'] = `Bearer ${authToken}`;
    }

    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        
        // Track upload progress
        xhr.upload.addEventListener('progress', (event) => {
            if (event.lengthComputable && progressCallback) {
                const percentComplete = Math.round((event.loaded / event.total) * 100);
                // Only show up to 85% for the upload phase, saving the rest for server processing
                const adjustedPercent = Math.floor(percentComplete * 0.85);
                progressCallback(adjustedPercent, 'uploading');
            }
        });
        
        xhr.addEventListener('load', () => {
            if (xhr.status >= 200 && xhr.status < 300) {
                try {
                    if (progressCallback) {
                        // Notify that processing is happening on the server
                        progressCallback(85, 'processing');
                        
                        // Simulate gradual progress for the face detection on server
                        // This is just an estimate since we don't get real-time updates from server
                        let processingProgress = 85;
                        
                        // Estimate processing time based on number of files
                        // Roughly 1-2 seconds per image for face detection
                        const processingTimeEstimate = Math.min(8000, files.length * 1500); 
                        const stepSize = 3;
                        const steps = Math.floor((100 - 85) / stepSize);
                        const interval = processingTimeEstimate / steps;
                        
                        const progressInterval = setInterval(() => {
                            if (processingProgress >= 99) {
                                clearInterval(progressInterval);
                            } else {
                                processingProgress += stepSize;
                                progressCallback(Math.min(processingProgress, 99), 'processing');
                            }
                        }, interval);
                    }
                    
                    const response = JSON.parse(xhr.responseText);
                    // Add backend URL to photo URLs
                    const photos = response.map((photo: Photo) => ({
                        ...photo,
                        url: getFileUrl(photo.url)
                    }));
                    
                    // After a short delay to allow the progress animation to run
                    // and for the facial recognition to complete
                    setTimeout(() => {
                        if (progressCallback) {
                            progressCallback(100, 'processing');
                        }
                        resolve(photos);
                    }, files.length * 1500);
                } catch (e) {
                    reject(new Error('Failed to parse response'));
                }
            } else {
                try {
                    const errorResponse = JSON.parse(xhr.responseText);
                    reject(new Error(errorResponse.error || 'Upload failed'));
                } catch (e) {
                    reject(new Error(`Upload failed with status ${xhr.status}`));
                }
            }
        });
        
        xhr.addEventListener('error', () => {
            reject(new Error('Network error during upload'));
        });
        
        xhr.addEventListener('abort', () => {
            reject(new Error('Upload aborted'));
        });
        
        xhr.open('POST', `${API_BASE_URL}/photos`);
        if (authToken) {
            xhr.setRequestHeader('Authorization', `Bearer ${authToken}`);
        }
        xhr.withCredentials = true;
        
        // Notify that upload is starting
        if (progressCallback) {
            progressCallback(0, 'uploading');
        }
        
        xhr.send(formData);
    });
}

// Admin API functions
export async function getAdminStats(headers?: HeadersInit): Promise<AdminStats> {
    const response = await fetch(`${API_BASE_URL}/admin/stats`, {
        headers: headers || getAuthHeaders(),
        credentials: 'include'
    });

    return handleResponse<AdminStats>(response);
}

export async function listUsers(): Promise<User[]> {
    const response = await fetch(`${API_BASE_URL}/admin/users`, {
        headers: getAuthHeaders(),
        credentials: 'include'
    });

    return handleResponse<User[]>(response);
}

export async function updateUser(userId: string, data: UpdateUserRequest): Promise<User> {
    const body = JSON.stringify(data);
    const response = await fetch(`${API_BASE_URL}/admin/users/${userId}`, {
        method: 'PUT',
        headers: getAuthHeaders(),
        credentials: 'include',
        body
    });

    return handleResponse<User>(response, 'PUT', body);
}

export async function deleteUser(userId: string): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/admin/users/${userId}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
        credentials: 'include'
    });

    if (!response.ok) {
        throw new Error('Failed to delete user');
    }
}

export async function getAdminEvents(): Promise<Event[]> {
    const response = await fetch(`${API_BASE_URL}/admin/events`, {
        headers: getAuthHeaders(),
        credentials: 'include'
    });
    
    return handleResponse<Event[]>(response);
}

export async function updateEvent(eventId: string, data: Partial<Event>): Promise<Event> {
    const body = JSON.stringify(data);
    const response = await fetch(`${API_BASE_URL}/admin/events/${eventId}`, {
        method: 'PUT',
        headers: getAuthHeaders(),
        credentials: 'include',
        body
    });

    return handleResponse<Event>(response, 'PUT', body);
}

export async function deleteEvent(eventId: string): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/admin/events/${eventId}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
        credentials: 'include'
    });

    if (!response.ok) {
        throw new Error('Failed to delete event');
    }
}

export async function getPhotosWithUser(): Promise<Photo[]> {
    const response = await fetch(`${API_BASE_URL}/photos/with-user`, {
        headers: getAuthHeaders()
    });
    if (!response.ok) {
        throw new Error('Failed to fetch photos');
    }
    const photos = await response.json();
    return photos.map((photo: Photo) => ({
        ...photo,
        url: getFileUrl(photo.url)
    }));
}

export async function deletePhoto(photoId: string): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/photos/${photoId}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
        credentials: 'include'
    });

    if (!response.ok) {
        throw new Error('Failed to delete photo');
    }
}