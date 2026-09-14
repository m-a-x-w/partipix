<script lang="ts">
	import { onMount } from 'svelte';
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import PhotoCarousel from '$lib/components/PhotoCarousel.svelte';
	import { getPhotosWithUser } from '$lib/services/api';
	import { user } from '$lib/stores';
	import { formatDate } from '$lib/utils';
	import type { Photo } from '$lib/types';

	let photos: Photo[] = [];
	let loading = true;
	let error: string | null = null;
	let selectedPhotos: Photo[] = [];
	let isCarouselOpen = false;
	let showFacesMap: Record<string, boolean> = {};
	let currentIndex = 0;

	onMount(async () => {
		try {
			loading = true;
			error = null;
			photos = await getPhotosWithUser();
		} catch (e) {
			console.error('Failed to fetch photos:', e);
			error = 'Failed to load your photos. Please try again.';
		} finally {
			loading = false;
		}
	});

	function openPhotoGallery(startIndex = 0) {
		selectedPhotos = photos;
		isCarouselOpen = true;
		// Use a small timeout to ensure the carousel has been created before setting the index
		setTimeout(() => {
			currentIndex = startIndex;
		}, 10);
	}

	function closePhotoGallery() {
		isCarouselOpen = false;
		selectedPhotos = [];
		currentIndex = 0;
	}
	
	function toggleFacesForPhoto(photoId: string) {
		showFacesMap = {
			...showFacesMap,
			[photoId]: !showFacesMap[photoId]
		};
	}
</script>

<div class="min-h-screen bg-gray-50 py-8">
	<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
		<div class="sm:flex sm:items-center">
			<div class="sm:flex-auto">
				<h1 class="text-2xl font-semibold text-gray-900">Photos of You</h1>
				<p class="mt-2 text-sm text-gray-700">
					All photos where you appear, automatically detected using facial recognition.
				</p>
			</div>
		</div>

		{#if loading}
			<div class="flex justify-center py-12">
				<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-600"></div>
			</div>
		{:else if error}
			<div class="rounded-md bg-red-50 p-4 mt-8">
				<div class="flex">
					<div class="flex-shrink-0">
						<svg class="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293-1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
						</svg>
					</div>
					<div class="ml-3">
						<p class="text-sm text-red-700">{error}</p>
					</div>
				</div>
			</div>
		{:else if photos.length === 0}
			<div class="text-center py-12">
				<svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
				</svg>
				<h3 class="mt-2 text-sm font-medium text-gray-900">No Photos</h3>
				<p class="mt-1 text-sm text-gray-500">You haven't been tagged in any photos yet.</p>
				<div class="mt-6">
					<Button href="/events" variant="primary">Browse Events</Button>
				</div>
			</div>
		{:else}
			<div class="mt-8 grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
				{#each photos as photo}
					<button
						type="button"
						class="relative group border-none bg-transparent p-0 w-full text-left cursor-pointer focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 rounded-lg"
						on:click={() => openPhotoGallery(photos.indexOf(photo))}
						aria-label={`View photo uploaded by ${photo.uploadedBy} on ${formatDate(photo.uploadedAt)}`}
					>
						<Card
							image={photo.url}
							title={`Uploaded by ${photo.uploadedByName || photo.uploadedBy}`}
							uploadDate={photo.uploadedAt}
							showDownload={true}
							faceMatches={photo.faceMatches}
							showFaces={showFacesMap[photo.id] || false}
							onDownload={() => {
								const a = document.createElement('a');
								a.href = photo.url;
								a.download = `photo-${photo.id}.jpg`;
								document.body.appendChild(a);
								a.click();
								document.body.removeChild(a);
							}}
							on:toggleFaces={() => toggleFacesForPhoto(photo.id)}
						/>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>

{#if isCarouselOpen}
	<PhotoCarousel
		bind:isOpen={isCarouselOpen}
		photos={selectedPhotos}
		albumTitle="Photos of You"
		currentIndex={currentIndex}
		showFaces={true}
		on:close={closePhotoGallery}
	/>
{/if}