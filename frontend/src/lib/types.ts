export interface Event {
	id: string;
	title: string;
	description: string;
	date: string;
	location?: string;
	participants: number;
	photos?: Photo[];
	guests?: Array<{
		name: string;
		status: 'uploaded' | 'not_uploading' | 'will_upload';
		isHost?: boolean;
	}>;
	code?: string;
	createdAt?: string;
	updatedAt?: string;
	createdBy?: string;
}

export interface Guest {
	name: string;
	status: 'uploaded' | 'not_uploading' | 'will_upload';
	isHost?: boolean;
}

export interface FaceMatch {
    userId: string;
    userName: string;
    confidence: number;
    location: {
        x: number;
        y: number;
        width: number;
        height: number;
    };
}

export interface Photo {
	id: string;
	url: string;
	uploadedBy: string;
	uploadedByName: string;
	uploadedAt: string;
	eventId: string;
	peopleInPhoto: string[];
	faceMatches: FaceMatch[];
}

export interface User {
	id: string;
	name: string;
	email: string;
	avatar?: string;
	token: string;
	password?: string;
	isAdmin?: boolean;
	events?: Event[];
	photos?: Photo[];
	referencePhotos?: ReferencePhoto[];
	createdAt?: string;
	updatedAt?: string;
}

export interface ReferencePhoto {
	id?: string;
	url: string;
	uploadedAt?: string;
}

export interface AdminStats {
    totalUsers: number;
    totalEvents: number;
    totalPhotos: number;
    activeEvents: number;
}

export interface UpdateUserRequest {
    name?: string;
    email?: string;
    isAdmin?: boolean;
}