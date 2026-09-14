<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Modal from './Modal.svelte';
	import { formatDate } from '$lib/utils';
	import type { Photo, FaceMatch, User, Event } from '$lib/types';
	import { onMount } from 'svelte';
	import { user } from '$lib/stores';
	import { deletePhoto } from '$lib/services/api';

	export let isOpen = false;
	export let photos: Photo[] = [];
	export let currentIndex = 0;
	export let albumTitle: string = "Photo Gallery";
	export let showFaces: boolean = false;
	export let currentUser: User | null = null;
	export let currentEvent: Event | undefined = undefined;

	// Subscribe to user store
	user.subscribe(value => {
		currentUser = value;
	});

	const dispatch = createEventDispatcher();

	let imageElement: HTMLImageElement;

	function handleClose() {
		dispatch('close');
	}

	function nextPhoto() {
		if (currentIndex < photos.length - 1) {
			currentIndex++;
		}
	}

	function prevPhoto() {
		if (currentIndex > 0) {
			currentIndex--;
		}
	}

	function downloadCurrentPhoto() {
		const photo = photos[currentIndex];
		if (!photo) return;
		
		const a = document.createElement('a');
		a.href = photo.url;
		a.download = `photo-${photo.id}.jpg`;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
	}

	async function handleDelete() {
		const photo = photos[currentIndex];
		if (!photo) return;

		if (!confirm('Are you sure you want to delete this photo? This action cannot be undone.')) {
			return;
		}

		try {
			await deletePhoto(photo.id);

			// Remove the photo from the array
			photos = photos.filter(p => p.id !== photo.id);
			
			// If there are no more photos, close the carousel
			if (photos.length === 0) {
				handleClose();
			} else if (currentIndex >= photos.length) {
				// If we deleted the last photo, move to the previous one
				currentIndex = photos.length - 1;
			}

			// Notify parent about the deletion
			dispatch('delete', { photoId: photo.id });
		} catch (error) {
			console.error('Failed to delete photo:', error);
			alert('Failed to delete photo. Please try again.');
		}
	}

	function getScaledCoordinates(match: FaceMatch) {
		if (!imageElement) return match.location;
		
		const scale = imageElement.width / imageElement.naturalWidth;
		return {
			x: match.location.x * scale,
			y: match.location.y * scale,
			width: match.location.width * scale,
			height: match.location.height * scale
		};
	}

	// Check if user has permission to delete the current photo
	$: canDeletePhoto = (photo: Photo) => {
		if (!currentUser || !photo) return false;
		// User can delete if they uploaded the photo
		if (photo.uploadedBy === currentUser.id) return true;
		// Event host can delete any photo in their event
		if (currentEvent?.guests?.some((g) => g.isHost === true && g.name === currentUser?.name)) return true;
		return false;
	};

	onMount(() => {
		if (imageElement) {
			const resizeObserver = new ResizeObserver(() => {
				// Force reactive update when image size changes
				imageElement = imageElement;
			});
			resizeObserver.observe(imageElement);
			return () => resizeObserver.disconnect();
		}
	});

	$: currentPhoto = photos[currentIndex];
	$: highConfidenceMatches = currentPhoto?.faceMatches?.filter(match => match.confidence >= 0.8) ?? [];
	
	// Create a filtered list that eliminates duplicates for display
	$: uniqueUserMatches = (() => {
		const faceMatches = currentPhoto?.faceMatches ?? [];
		// Create a map to track best match per user
		const bestMatchPerUser = new Map();
		
		// Find the best match for each user
		faceMatches.forEach(match => {
			if (!bestMatchPerUser.has(match.userId) || bestMatchPerUser.get(match.userId).confidence < match.confidence) {
				bestMatchPerUser.set(match.userId, match);
			}
		});
		
		// Convert Map values back to array
		return Array.from(bestMatchPerUser.values())
			.filter(match => match.confidence >= 0.8) // Keep high confidence matches
			.sort((a, b) => b.confidence - a.confidence); // Sort by confidence
	})();
</script>

<Modal {isOpen} title={albumTitle} on:close={handleClose}>
	<div class="relative">
		<div class="aspect-w-16 aspect-h-9 relative">
			<img
				bind:this={imageElement}
				src={currentPhoto?.url}
				alt={`Photo uploaded by ${currentPhoto?.uploadedByName || currentPhoto?.uploadedBy}`}
				class="object-contain w-full h-full"
			/>
			{#if showFaces && highConfidenceMatches.length > 0}
				{#each uniqueUserMatches as match}
					{@const scaled = getScaledCoordinates(match)}
					<div
						class="absolute bg-black/10 border-2 border-emerald-400"
						style="left: {scaled.x}px; top: {scaled.y}px; width: {scaled.width}px; height: {scaled.height}px;"
					>
						<div class="absolute -bottom-6 left-0 bg-emerald-400 text-white px-2 py-0.5 text-xs rounded-md whitespace-nowrap">
							{match.userName}
						</div>
					</div>
				{/each}
			{/if}
		</div>
		<div class="absolute top-0 left-0 right-0 flex justify-between p-4">
			{#if photos.length > 1}
				<div class="flex gap-2">
					<button
						class="p-2 rounded-full bg-white/80 hover:bg-white text-gray-800"
						on:click={prevPhoto}
						disabled={currentIndex === 0}
						aria-label="Previous photo"
					>
						<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
						</svg>
					</button>
					<button
						class="p-2 rounded-full bg-white/80 hover:bg-white text-gray-800"
						on:click={nextPhoto}
						disabled={currentIndex === photos.length - 1}
						aria-label="Next photo"
					>
						<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
						</svg>
					</button>
				</div>
			{:else}
				<div></div> <!-- Empty div to maintain flex justify-between spacing -->
			{/if}
			<div class="flex gap-2">
				{#if highConfidenceMatches.length > 0}
					<button
						class="p-2 rounded-full bg-white/80 hover:bg-white text-gray-800"
						on:click={() => showFaces = !showFaces}
						aria-label={showFaces ? "Hide face tags" : "Show face tags"}
					>
						<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 0 0118 0z" />
						</svg>
					</button>
				{/if}
				<button
					class="p-2 rounded-full bg-white/80 hover:bg-white text-gray-800"
					on:click={downloadCurrentPhoto}
					aria-label="Download photo"
				>
					<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
					</svg>
				</button>
				{#if canDeletePhoto(currentPhoto)}
					<button
						class="p-2 rounded-full bg-white/80 hover:bg-white text-red-600"
						on:click={handleDelete}
						aria-label="Delete photo"
					>
						<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-7 7-7-7" />
						</svg>
					</button>
				{/if}
			</div>
		</div>
		{#if currentPhoto}
			<div class="mt-4 text-center">
				<p class="text-sm text-gray-500">
					Uploaded by {currentPhoto.uploadedByName || currentPhoto.uploadedBy} on {formatDate(currentPhoto.uploadedAt)}
				</p>
				{#if highConfidenceMatches.length > 0}
					<div class="mt-2">
						<p class="text-sm font-medium text-gray-700">People in this photo:</p>
						<div class="mt-1 flex flex-wrap justify-center gap-2">
							{#each uniqueUserMatches as match}
								<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-100 text-emerald-800">
									{match.userName} ({(match.confidence * 100).toFixed(1)}% match)
								</span>
							{/each}
						</div>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</Modal>