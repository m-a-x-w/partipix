<script lang="ts">
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import FaceVerificationModal from '$lib/components/FaceVerificationModal.svelte';
	import { events } from '$lib/stores';
	import { getUserProfile, updateUserProfile, signOut, uploadReferencePhotosForVerification, confirmReferencePhoto, deleteReferencePhoto } from '$lib/services/api';
	import { onMount } from 'svelte';
	import { user } from '$lib/stores';
	import { goto } from '$app/navigation';
	import type { User } from '$lib/types';
	import type { DetectedFace } from '$lib/services/api';

	let isEditModalOpen = false;
	let editedUser: Partial<User> = {};
	let password = '';
	let confirmPassword = '';
	let isLoading = true;
	let error: string | null = null;
	let currentUser: User | null = null;
	let isUploading = false;
	let isDragging = false;
	let detectedFaces: DetectedFace[] = [];
	let currentFaceIndex = 0;
	let isVerificationModalOpen = false;
	let reviewedPhotoIds = new Set<string>();

	// Fix TypeScript error in stats calculation
	$: userStats = currentUser ? {
		eventsAttended: currentUser.events?.length || 0,
		photosUploaded: currentUser.photos?.length || 0,
		photosTaggedIn: currentUser.photos?.filter(p => p.peopleInPhoto?.includes(currentUser?.id || ''))?.length || 0
	} : null;

	// Subscribe to user store
	user.subscribe(value => {
		currentUser = value;
	});

	onMount(async () => {
		if (!currentUser) {
			goto('/signin');
			return;
		}

		await refreshProfile();
	});

	function openEditModal() {
		console.log('Opening edit modal');
		// Ensure editedUser is initialized with current user data
		if (currentUser) {
			editedUser = {
				name: currentUser.name,
				email: currentUser.email
			 };
			password = '';
			confirmPassword = '';
		}
		isEditModalOpen = true;
	}

	async function refreshProfile() {
		isLoading = true;
		error = null;

		try {
			const profile = await getUserProfile();
			user.set({ ...profile, token: currentUser?.token || '' });
		} catch (e) {
			error = 'Failed to load profile';
			console.error(e);
			if (e instanceof Error && e.message.includes('401')) {
				await handleSignOut();
			}
		} finally {
			isLoading = false;
		}
	}

	async function handleSaveProfile() {
		if (!editedUser.name || !editedUser.email) {
			error = 'Name and email are required';
			return;
		}

		if (password && password !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}

		if (password && password.length < 8) {
			error = 'Password must be at least 8 characters long';
			return;
		}

		try {
			const updates = {
				...editedUser,
				...(password ? { password } : {})
			};
			const updatedProfile = await updateUserProfile(updates);
			user.set({ ...updatedProfile, token: currentUser?.token || '' });
			isEditModalOpen = false;
			error = null;
		} catch (e) {
			error = 'Failed to update profile';
			console.error(e);
		}
	}

	async function handleSignOut() {
		try {
			await signOut();
			goto('/signin');
		} catch (e) {
			console.error('Error signing out:', e);
			// Still redirect to signin page even if the API call fails
			goto('/signin');
		}
	}

	function handleDragEnter(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		// Only allow dragging if we have space for more photos
		const photoCount = currentUser?.referencePhotos?.length ?? 0;
		if (photoCount < 5) {
			isDragging = true;
		}
	}

	function handleDragLeave(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		isDragging = false;
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		// Add visual feedback if we can't accept more photos
		const photoCount = currentUser?.referencePhotos?.length ?? 0;
		if (photoCount >= 5 && e.dataTransfer) {
			e.dataTransfer.dropEffect = 'none';
		}
	}

	async function handleDrop(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		isDragging = false;

		// Prevent drop if we already have 5 photos
		const photoCount = currentUser?.referencePhotos?.length ?? 0;
		if (photoCount >= 5) {
			error = 'Maximum 5 photos allowed. Please delete some photos first.';
			return;
		}

		if (!e.dataTransfer?.files || e.dataTransfer.files.length === 0) return;

		const files = Array.from(e.dataTransfer.files);
		await processFiles(files);
	}

	async function handleFileInput(event: Event) {
		const input = event.target as HTMLInputElement;
		if (!input.files || input.files.length === 0) return;
		
		const files = Array.from(input.files);
		await processFiles(files);
		input.value = ''; // Reset input
	}

	async function processFiles(files: File[]) {
		// Check if adding these files would exceed the limit
		const currentCount = currentUser?.referencePhotos?.length || 0;
		if (currentCount + files.length > 5) {
			error = `Can only upload ${5 - currentCount} more photo${5 - currentCount === 1 ? '' : 's'}`;
			return;
		}

		const validImageTypes = ['image/jpeg', 'image/png', 'image/webp', 'image.heic', 'image.heif'];
		const maxSize = 5 * 1024 * 1024; // 5MB

		// Validate file types and sizes
		const invalidFiles = files.filter(
			file => !validImageTypes.includes(file.type) || file.size > maxSize
		);

		if (invalidFiles.length > 0) {
			error = 'Invalid files. Please upload only images (JPG, PNG, WebP, HEIC, HEIF) under 5MB';
			return;
		}

		isUploading = true;
		error = null;

		try {
			// Get detected faces first
			const faces = await uploadReferencePhotosForVerification(files);
			console.log("Received detected faces:", faces);
			
			if (faces.length === 0) {
				error = 'No faces were detected in the uploaded photos. Please try a clearer photo of your face.';
				return;
			}

			// Store the faces in our state
			detectedFaces = faces;

			// Show verification modal for the first face
			currentFaceIndex = 0;
			isVerificationModalOpen = true;
		} catch (e) {
			if (e instanceof Error) {
				error = e.message;
			} else {
				error = 'Failed to upload photos';
			}
			console.error(e);
		} finally {
			isUploading = false;
		}
	}

	async function handleFaceConfirm() {
		try {
			const currentFace = detectedFaces[currentFaceIndex];
			console.log("Confirming face:", currentFace);
			
			// Send the exact IDs we received from the backend
			const photo = await confirmReferencePhoto(currentFace.originalPhotoId, currentFace.id);
			console.log("Reference photo saved:", photo);

			// Update the user store with the new reference photo
			if (currentUser) {
				const updatedUser = {
					...currentUser,
					referencePhotos: [...(currentUser.referencePhotos || []), photo]
				};
				user.set(updatedUser);
			 }

			// Reset state
			detectedFaces = [];
			currentFaceIndex = 0;
			reviewedPhotoIds.clear();
			isVerificationModalOpen = false;
		} catch (e) {
			if (e instanceof Error) {
				error = e.message;
			} else {
				error = 'Failed to save reference photo';
			}
			console.error('Error saving reference photo:', e);
			
			// Clean up the rejected face
			const currentFace = detectedFaces[currentFaceIndex];
			if (currentFace) {
				reviewedPhotoIds.add(currentFace.originalPhotoId);
			}
			
			// Close modal and reset state since the photo was rejected
			isVerificationModalOpen = false;
			detectedFaces = [];
			currentFaceIndex = 0;
		}
	}

	function handleFaceReject() {
		const currentFace = detectedFaces[currentFaceIndex];

		// Move to next face or close modal
		if (currentFaceIndex < detectedFaces.length - 1) {
			currentFaceIndex++;
		} else {
			isVerificationModalOpen = false;
			detectedFaces = [];
			reviewedPhotoIds.clear();
		}
	}

	async function handleDeletePhoto(photoId: string) {
		if (!confirm('Are you sure you want to delete this reference photo?')) {
			return;
		}

		try {
			// Remove the photo from UI immediately for better UX
			if (currentUser) {
				const updatedUser = {
					...currentUser,
					referencePhotos: currentUser.referencePhotos?.filter(p => p.id !== photoId) || []
				};
				user.set(updatedUser);
			}

			// Then try to delete it in the backend
			await deleteReferencePhoto(photoId);
		} catch (e) {
			// On error, refresh the profile to get the correct state
			error = 'Failed to delete photo';
			console.error(e);
			await refreshProfile();
		}
	}
</script>

<div class="min-h-screen bg-gray-50 py-8 px-4 sm:px-6 lg:px-8">
	{#if error}
		<div class="max-w-3xl mx-auto mb-4">
			<div class="bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded relative" role="alert">
				<span class="block sm:inline">{error}</span>
			</div>
		</div>
	{/if}

	<div class="max-w-3xl mx-auto space-y-6">
		<div class="bg-white shadow sm:rounded-lg">
			{#if isLoading}
				<div class="p-8 flex justify-center">
					<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-emerald-500"></div>
				</div>
			{:else if currentUser}
				<div class="px-4 py-5 sm:p-6">
					<div class="flex justify-between items-start">
						<div>
							<h3 class="text-lg leading-6 font-medium text-gray-900">Profile Information</h3>
							<div class="mt-4">
								<div class="text-sm text-gray-500">Name</div>
								<div class="mt-1 text-sm text-gray-900">{currentUser.name}</div>
							</div>
							<div class="mt-4">
								<div class="text-sm text-gray-500">Email</div>
								<div class="mt-1 text-sm text-gray-900">{currentUser.email}</div>
							</div>
							<div class="mt-6">
								<h3 class="text-lg font-medium text-gray-900">Reference Photos</h3>
								<p class="mt-2 text-sm text-gray-600">Add clear photos of your face from different angles to improve facial recognition</p>
								
								{#if currentUser.referencePhotos?.length}
									<div class="mt-4 grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-4">
										{#each currentUser.referencePhotos as photo}
											<div class="relative group aspect-square">
												<img 
													src={photo.url} 
													alt="" 
													class="w-full h-full object-cover rounded-lg ring-1 ring-gray-200"
													on:error={async () => {
														// If image fails to load, remove it from UI and delete it from backend
														if (currentUser && photo.id) {
															const updatedUser = {
																...currentUser,
																referencePhotos: currentUser.referencePhotos?.filter(p => p.id !== photo.id) || []
															};
															user.set(updatedUser);
															try {
																await deleteReferencePhoto(photo.id);
															} catch (e) {
																console.error('Failed to delete missing photo:', e);
															}
														}
													}}
												/>
												<div class="absolute inset-0 bg-black bg-opacity-0 group-hover:bg-opacity-40 transition-all duration-200 rounded-lg flex items-center justify-center">
													<button
														class="opacity-0 group-hover:opacity-100 transform translate-y-2 group-hover:translate-y-0 transition-all duration-200"
														on:click={() => photo.id && handleDeletePhoto(photo.id)}
														aria-label="Delete photo"
													>
														<div class="bg-white/90 hover:bg-white p-2 rounded-full text-red-600 hover:text-red-700 shadow-lg">
															<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
																<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
															</svg>
														</div>
													</button>
												</div>
											</div>
										{/each}
										{#if currentUser.referencePhotos.length < 5}
											<div 
												class="relative aspect-square cursor-pointer"
												role="button"
												tabindex="0"
												aria-label="Add photo by drag and drop or click"
												on:dragenter={handleDragEnter}
												on:dragleave={handleDragLeave}
												on:dragover={handleDragOver}
												on:drop={handleDrop}
											>
												<label class="block w-full h-full">
													<div class="{`w-full h-full rounded-lg border-2 border-dashed
														${isDragging ? 'border-emerald-500 bg-emerald-50' : 'border-gray-300'}
														hover:border-emerald-500 flex flex-col items-center justify-center transition-all duration-200`}">
														{#if isUploading}
															<div class="animate-spin rounded-full h-6 w-6 border-2 border-emerald-500 border-t-transparent"></div>
														{:else}
															<svg class="h-8 w-8 text-gray-400 group-hover:text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
																<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
															</svg>
															<span class="mt-2 block text-sm font-medium text-gray-900">
																{isDragging ? 'Drop here' : 'Add photo'}
															</span>
														{/if}
													</div>
													<input 
														type="file" 
														accept="image/*" 
														class="sr-only" 
														on:change={handleFileInput}
														disabled={isUploading}
													/>
												</label>
											</div>
										{/if}
									</div>
									<p class="mt-3 text-sm text-gray-500">
										{currentUser.referencePhotos.length}/5 photos added
									</p>
								{:else}
									<div class="mt-4 max-w-lg">
										<div 
											class="relative group cursor-pointer"
											role="button"
											tabindex="0"
											aria-label="Upload photos by drag and drop or click"
											on:dragenter={handleDragEnter}
											on:dragleave={handleDragLeave}
											on:dragover={handleDragOver}
											on:drop={handleDrop}
										>
											<label class="block">
												<div class="{`mt-1 flex justify-center px-6 pt-5 pb-6 border-2 border-dashed 
													${isDragging ? 'border-emerald-500 bg-emerald-50' : 'border-gray-300'} 
													hover:border-emerald-500 rounded-lg transition-all duration-200`}">
													<div class="space-y-1 text-center">
														{#if isUploading}
															<div class="flex flex-col items-center">
																<div class="animate-spin rounded-full h-8 w-8 border-2 border-emerald-500 border-t-transparent"></div>
																<p class="mt-2 text-sm text-gray-500">Uploading...</p>
															</div>
														{:else}
															<svg class="mx-auto h-12 w-12 text-gray-400" stroke="currentColor" fill="none" viewBox="0 0 48 48">
																<path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
															</svg>
															<div class="flex flex-col text-sm text-gray-600">
																<p class="font-medium text-emerald-600 hover:text-emerald-500">
																	{isDragging ? 'Drop photos here' : 'Upload photos'}
																</p>
																<p class="text-xs text-gray-500 mt-1">
																	Drag & drop or click to select • Up to 5 photos, max 5MB each
																</p>
															</div>
														{/if}
													</div>
												</div>
												<input 
													type="file" 
													accept="image/*" 
													multiple 
													class="sr-only" 
													on:change={handleFileInput}
													disabled={isUploading}
												/>
											</label>
										</div>
									</div>
								{/if}

								{#if error}
									<p class="mt-2 text-sm text-red-600">{error}</p>
								{/if}
							</div>
						</div>
						<div class="flex space-x-3">
							<Button variant="outline" on:click={openEditModal}>
								Edit Profile
							</Button>
							<Button variant="outline" on:click={handleSignOut}>
								Sign Out
							</Button>
						</div>
					</div>
				</div>
			{/if}
		</div>

		{#if !isLoading && currentUser && userStats}
			<div class="bg-white shadow sm:rounded-lg overflow-hidden">
				<div class="px-4 py-5 sm:p-6">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Activity Stats</h3>
					<div class="mt-5 grid grid-cols-1 gap-5 sm:grid-cols-3">
						<div class="px-4 py-5 bg-gray-50 shadow-sm rounded-lg overflow-hidden sm:p-6">
							<dt class="text-sm font-medium text-gray-500 truncate">Events Attended</dt>
							<dd class="mt-1 text-3xl font-semibold text-emerald-600">{userStats.eventsAttended}</dd>
						</div>
						<div class="px-4 py-5 bg-gray-50 shadow-sm rounded-lg overflow-hidden sm:p-6">
							<dt class="text-sm font-medium text-gray-500 truncate">Photos Uploaded</dt>
							<dd class="mt-1 text-3xl font-semibold text-emerald-600">{userStats.photosUploaded}</dd>
						</div>
						<div class="px-4 py-5 bg-gray-50 shadow-sm rounded-lg overflow-hidden sm:p-6">
							<dt class="text-sm font-medium text-gray-500 truncate">Photos Tagged In</dt>
							<dd class="mt-1 text-3xl font-semibold text-emerald-600">{userStats.photosTaggedIn}</dd>
						</div>
					</div>
				</div>
			</div>
		{/if}
	</div>
</div>

{#if isEditModalOpen}
	<Modal 
		title="Edit Profile" 
		isOpen={isEditModalOpen}
		on:close={() => isEditModalOpen = false}
	>
		<form class="space-y-6" on:submit|preventDefault={handleSaveProfile}>
			<div>
				<label for="name" class="block text-sm font-medium text-gray-700">Name</label>
				<div class="mt-1">
					<input
						id="name"
						name="name"
						type="text"
						required
						bind:value={editedUser.name}
						class="block w-full appearance-none rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-emerald-500 focus:outline-none focus:ring-emerald-500 sm:text-sm"
					/>
				</div>
			</div>

			<div>
				<label for="email" class="block text-sm font-medium text-gray-700">Email</label>
				<div class="mt-1">
					<input
						id="email"
						name="email"
						type="email"
						required
						bind:value={editedUser.email}
						class="block w-full appearance-none rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-emerald-500 focus:outline-none focus:ring-emerald-500 sm:text-sm"
					/>
				</div>
			</div>

			<div class="border-t border-gray-200 pt-4">
				<h4 class="text-sm font-medium text-gray-900">Change Password</h4>
				<p class="mt-1 text-sm text-gray-500">Leave blank to keep your current password</p>
				
				<div class="mt-4">
					<label for="password" class="block text-sm font-medium text-gray-700">New Password</label>
					<div class="mt-1">
						<input
							id="password"
							name="password"
							type="password"
							bind:value={password}
							class="block w-full appearance-none rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-emerald-500 focus:outline-none focus:ring-emerald-500 sm:text-sm"
						/>
					</div>
				</div>

				<div class="mt-4">
					<label for="confirm-password" class="block text-sm font-medium text-gray-700">Confirm New Password</label>
					<div class="mt-1">
						<input
							id="confirm-password"
							name="confirm-password"
							type="password"
							bind:value={confirmPassword}
							class="block w-full appearance-none rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-emerald-500 focus:outline-none focus:ring-emerald-500 sm:text-sm"
						/>
					</div>
				</div>
			</div>

			{#if error}
				<div class="text-sm text-red-600">{error}</div>
			{/if}

			<div class="flex justify-end space-x-3">
				<Button variant="outline" on:click={() => isEditModalOpen = false}>Cancel</Button>
				<Button type="submit" variant="primary">Save Changes</Button>
			</div>
		</form>
	</Modal>
{/if}

{#if isVerificationModalOpen && detectedFaces[currentFaceIndex]}
	<FaceVerificationModal
		isOpen={true}
		faceImageUrl={detectedFaces[currentFaceIndex].url}
		onConfirm={handleFaceConfirm}
		onReject={handleFaceReject}
	/>
{/if}