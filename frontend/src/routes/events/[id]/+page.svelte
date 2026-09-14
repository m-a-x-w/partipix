<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import PhotoCarousel from '$lib/components/PhotoCarousel.svelte';
	import PhotoUploadModal from '$lib/components/PhotoUploadModal.svelte';
	import ShareEventModal from '$lib/components/ShareEventModal.svelte';
	import { getEventById, updateGuestStatus, deleteUserEvent } from '$lib/services/api';
	import { user } from '$lib/stores';
	import { goto } from '$app/navigation';
	import type { Event, Photo } from '$lib/types';
	import { formatDate, formatShortDate } from '$lib/utils';

	let event: Event | null = null;
	let loading = true;
	let error: string | null = null;

	// Photo categories
	let yourPhotos: Photo[] = [];
	let photosYoureIn: Photo[] = [];
	let allPhotos: Photo[] = []; // Changed from otherPhotos to allPhotos

	let selectedAlbum: string | null = null;
	let selectedPhotos: Photo[] = [];
	let isCarouselOpen = false;

	let statusUpdateLoading = false;
	let isUploadModalOpen = false;
	let isShareModalOpen = false;

	let selectedTab: 'yours' | 'in' | 'all' = 'yours'; 
	let showFacesMap: Record<string, boolean> = {};
	let isGuestListOpen = false;

	import type { User } from '$lib/types';
	
	let currentUser: User | null = null;

	user.subscribe(value => {
		currentUser = value;
	});

	onMount(async () => {
		try {
			loading = true;
			event = await getEventById($page.params.id);
			
			// Sort photos into categories based on the backend response
			if (event.photos) {
				// Add detailed logging for debugging
				console.log('Raw photo data from API:', event.photos.slice(0, 3));
				
				 // Check upload dates
				console.log('Photo upload dates:', event.photos.map(p => ({
					id: p.id,
					uploadedAt: p.uploadedAt,
					formattedDate: formatShortDate(p.uploadedAt)
				})));

				// Ensure faceMatches is available on each photo
				event.photos = event.photos.map(photo => {
					console.log(`Photo ${photo.id} faceMatches:`, photo.faceMatches);
					return {
						...photo,
						faceMatches: photo.faceMatches || [] // Ensure faceMatches exists even if not provided by API
					};
				});
				
				// Your photos - ones you uploaded
				yourPhotos = event.photos.filter(p => p.uploadedBy === currentUser?.id);
				
				// Photos you're in - where you're tagged/recognized
				photosYoureIn = event.photos.filter(p => 
					p.uploadedBy !== currentUser?.id && 
					p.peopleInPhoto?.includes(currentUser?.id ?? '')
				);
				
				// All photos - include all photos in the event
				allPhotos = event.photos;

				// Log photo categories for debugging
				console.log('Your photos:', yourPhotos.map(p => ({
					id: p.id, 
					hasFaces: p.faceMatches && p.faceMatches.length > 0,
					faceCount: p.faceMatches?.length || 0
				})));
				
				console.log('Photos you\'re in:', photosYoureIn.map(p => ({
					id: p.id, 
					hasFaces: p.faceMatches && p.faceMatches.length > 0,
					faceCount: p.faceMatches?.length || 0
				})));
			}
		} catch (e) {
			console.error('Failed to fetch event', e);
			error = 'Failed to load event. Please try again.';
		} finally {
			loading = false;
		}
	});

	async function handleStatusUpdate(newStatus: 'uploaded' | 'not_uploading' | 'will_upload') {
		if (!event) return;
		try {
			statusUpdateLoading = true;
			error = null;
			const updatedEvent = await updateGuestStatus(event.id, newStatus);
			event = updatedEvent;
		} catch (e) {
			console.error('Failed to update status:', e);
			error = e instanceof Error ? e.message : 'Failed to update status. Please try again.';
		} finally {
			statusUpdateLoading = false;
		}
	}

	async function handlePhotoUpload(uploadEvent: CustomEvent<Photo[]>) {
		const newPhotos = uploadEvent.detail;
		
		// Sort new photos into appropriate categories
		newPhotos.forEach(photo => {
			// Check if this is a photo you uploaded
			if (photo.uploadedBy === currentUser?.id) {
				yourPhotos = [...yourPhotos, photo];
			} 
			// Check if you're recognized in the photo
			else if (photo.peopleInPhoto?.includes(currentUser?.id ?? '')) {
				photosYoureIn = [...photosYoureIn, photo];
			}
		});
		
		// Add all new photos to the all photos album
		allPhotos = [...allPhotos, ...newPhotos];
		
		// Update the event's total photos
		if (event) {
			event.photos = [...(event.photos || []), ...newPhotos];

			// Schedule a cleanup in case facial recognition completes later
			setTimeout(async () => {
				try {
					if (!event?.id) return;
					// Refetch the event to get updated photo data
					const updatedEvent = await getEventById(event.id);
					if (!updatedEvent?.photos) return;

					// Re-categorize all photos
					yourPhotos = updatedEvent.photos.filter(p => p.uploadedBy === currentUser?.id);
					photosYoureIn = updatedEvent.photos.filter(p => 
						p.uploadedBy !== currentUser?.id && 
						p.peopleInPhoto?.includes(currentUser?.id ?? '')
					);
					allPhotos = updatedEvent.photos;

					event = updatedEvent;
				} catch (e) {
					console.error('Failed to update photo categories:', e);
				}
			}, 5000); // Check after 5 seconds to allow for facial recognition processing
		}
	}

	function openAlbum(albumName: string, photos: Photo[], initialPhotoIndex: number = 0) {
		selectedAlbum = albumName;
		selectedPhotos = photos;
		isCarouselOpen = true;
		// Use the initialPhotoIndex to set the starting photo in the carousel
		if (isCarouselOpen) {
			// Small timeout to ensure carousel component has loaded
			setTimeout(() => {
				currentCarouselIndex = initialPhotoIndex;
			}, 10);
		}
	}

	let currentCarouselIndex = 0;

	function closeCarousel() {
		isCarouselOpen = false;
		selectedAlbum = null;
		selectedPhotos = [];
		currentCarouselIndex = 0;
	}

	function handlePhotoDelete(evt: CustomEvent<{ photoId: string }>) {
		const photoId = evt.detail.photoId;
		
		// Update all photo arrays
		yourPhotos = yourPhotos.filter((p: Photo) => p.id !== photoId);
		photosYoureIn = photosYoureIn.filter((p: Photo) => p.id !== photoId);
		allPhotos = allPhotos.filter((p: Photo) => p.id !== photoId);

		// Update photos in the current event
		if (event) {
			event = {
				...event,
				photos: event.photos?.filter((p: Photo) => p.id !== photoId) || []
			};
		}
	}

	// Get status color based on guest status
	function getStatusColor(status: string): { bg: string; text: string } {
		switch (status) {
			case 'uploaded':
				return { bg: 'bg-emerald-100', text: 'text-emerald-800' };
			case 'will_upload':
				return { bg: 'bg-yellow-100', text: 'text-yellow-800' };
			case 'not_uploading':
			default:
				return { bg: 'bg-gray-100', text: 'text-gray-800' };
		}
	}

	function toggleFacesForPhoto(photoId: string) {
		showFacesMap = {
			...showFacesMap,
			[photoId]: !showFacesMap[photoId]
		};
	}

	// Add function to handle event deletion
	async function handleEventDelete() {
		if (!event || !confirm('Are you sure you want to delete this event? This action cannot be undone and will delete all photos in the event.')) {
			return;
		}

		try {
			await deleteUserEvent(event.id);
			goto('/events');  // Redirect to events list after deletion
		} catch (e) {
			console.error('Failed to delete event:', e);
			error = e instanceof Error ? e.message : 'Failed to delete event';
		}
	}
</script>

<div class="bg-white">
	<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8">
		{#if loading}
			<div class="flex justify-center py-12">
				<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-600"></div>
			</div>
		{:else if error}
			<div class="rounded-md bg-red-50 p-4 my-8">
				<div class="flex">
					<div class="flex-shrink-0">
						<svg class="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 101.414 1.414L10 11.414l1.293 1.293a1 1 001.414-1.414L11.414 10l1.293-1.293a1 1 000-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
						</svg>
					</div>
					<div class="ml-3">
						<p class="text-sm text-red-700">{error}</p>
					</div>
				</div>
				<div class="mt-4">
					<Button href="/events" variant="outline">Back to Events</Button>
				</div>
			</div>
		{:else if event}
			<div class="sm:flex sm:items-center sm:justify-between">
				<div class="sm:flex-auto">
					<h1 class="text-2xl font-semibold text-gray-900">{event.title}</h1>
					<p class="mt-2 text-sm text-gray-700">
						{event.description} • {formatDate(event.date)}
					</p>
					<p class="mt-1 text-sm text-gray-500">
						{event.location ? `Location: ${event.location} • ` : ''}
						{event.participants} participants
					</p>
					{#if event.code}
						<div class="mt-2">
							<div class="flex items-center gap-2 mb-4">
								<span class="text-sm font-medium text-gray-700">Event Code: </span>
								<span class="inline-flex items-center px-3 py-1.5 rounded-md text-xl font-bold font-mono bg-gray-100 text-gray-800">
									{event.code.toUpperCase()}
								</span>
							</div>
							<div class="flex flex-col sm:flex-row gap-2 sm:items-center">
								<Button 
									variant="outline" 
									size="sm"
									on:click={() => isShareModalOpen = true}
									className="w-full sm:w-auto justify-center"
								>
									Share Event
								</Button>
							</div>
						</div>
					{/if}
				</div>
				<div class="mt-6 sm:mt-0 flex flex-col sm:flex-row gap-2">
					<Button variant="primary" on:click={() => isUploadModalOpen = true} className="w-full sm:w-auto justify-center">Upload Photos</Button>
					{#if event.guests?.some(g => g.isHost && g.name === currentUser?.name)}
						<Button variant="outline" className="w-full sm:w-auto justify-center !text-red-600 hover:!bg-red-50 hover:!border-red-600" on:click={handleEventDelete}>Delete Event</Button>
					{/if}
				</div>
			</div>

			<!-- Guest List as Dropdown -->
			<div class="mt-6">
				<button
					class="w-full flex items-center justify-between bg-white px-4 py-3 text-left text-gray-900 shadow rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500"
					on:click={() => isGuestListOpen = !isGuestListOpen}
					aria-expanded={isGuestListOpen}
				>
					<span class="flex items-center">
						<span class="text-lg font-medium">Guest List ({event.participants} {event.participants === 1 ? 'person' : 'people'})</span>
						{#if currentUser}
							<span class="ml-4 text-sm text-gray-500">
								Your status: 
								<span class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium 
									${getStatusColor(event.guests?.find(g => g.name === currentUser?.name)?.status ?? 'not_uploading').bg} 
									${getStatusColor(event.guests?.find(g => g.name === currentUser?.name)?.status ?? 'not_uploading').text}`}
								>
									{#if event.guests?.find(g => g.name === currentUser?.name)?.status === 'uploaded'}
										Uploaded
									{:else if event.guests?.find(g => g.name === currentUser?.name)?.status === 'will_upload'}
										Will Upload
									{:else}
										Not Uploading
									{/if}
								</span>
							</span>
						{/if}
					</span>
					<svg class={`h-5 w-5 text-gray-500 transform ${isGuestListOpen ? 'rotate-180' : ''}`} xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
					</svg>
				</button>

				{#if isGuestListOpen}
					<div class="mt-2 bg-white shadow rounded-lg overflow-hidden">
						{#if currentUser}
							<div class="px-4 py-3 border-b border-gray-200 flex justify-end">
								<div class="flex items-center gap-2">
									<label for="status-select" class="text-sm text-gray-700">Update your status:</label>
									<select
										id="status-select"
										class="rounded-md border-gray-300 py-1.5 pl-3 pr-8 text-sm focus:border-emerald-500 focus:outline-none focus:ring-emerald-500"
										value={event.guests?.find(g => g.name === currentUser?.name)?.status ?? 'not_uploading'}
										disabled={statusUpdateLoading}
										on:change={e => handleStatusUpdate(e.currentTarget.value as 'uploaded' | 'not_uploading' | 'will_upload')}
									>
										<option value="not_uploading">Not Uploading</option>
										<option value="will_upload">Will Upload</option>
										<option value="uploaded">Uploaded</option>
									</select>
									{#if statusUpdateLoading}
										<div class="animate-spin rounded-full h-4 w-4 border-b-2 border-emerald-600"></div>
									{/if}
								</div>
							</div>
						{/if}
						{#if event.guests && event.guests.length > 0}
							<ul role="list" class="divide-y divide-gray-200 max-h-64 overflow-y-auto">
								{#each [...event.guests].sort((a, b) => {
									if (a.isHost) return -1;
									if (b.isHost) return 1;
									return a.name.localeCompare(b.name);
								}) as guest}
									{@const statusColor = getStatusColor(guest.status)}
									<li class="px-4 py-3 sm:px-6">
										<div class="flex items-center justify-between">
											<div class="flex items-center">
												<span class="text-sm font-medium text-gray-900">
													{guest.name}
													{#if guest.isHost}
														<span class="ml-2 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
															Host
														</span>
													{/if}
												</span>
											</div>
											<div class="ml-4">
												<span class={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}>
													{guest.status === 'uploaded' ? 'Uploaded' : 
														guest.status === 'will_upload' ? 'Will Upload' : 'Not Uploading'}
												</span>
											</div>
										</div>
									</li>
								{/each}
							</ul>
						{:else}
							<div class="px-4 py-3 text-center text-gray-500">
								No guests have joined yet.
							</div>
						{/if}
					</div>
				{/if}
			</div>

			<div class="mt-6">
				{#if yourPhotos.length === 0 && photosYoureIn.length === 0 && allPhotos.length === 0}
						<div class="bg-gray-50 rounded-lg p-12 text-center">
							<h2 class="text-lg font-medium text-gray-900 mb-2">No Photos Yet</h2>
							<p class="text-sm text-gray-500 mb-6">
								Be the first to upload photos from this event.
							</p>
							<Button variant="primary" on:click={() => isUploadModalOpen = true}>Upload Photos</Button>
						</div>
					{:else}
						<div class="bg-white shadow rounded-lg">
							<!-- Tabs -->
							<div class="border-b border-gray-200">
								<nav class="-mb-px flex" aria-label="Tabs">
									<button
										class={`w-1/3 py-4 px-1 text-center border-b-2 font-medium text-sm
											${selectedTab === 'yours' ? 
											'border-emerald-500 text-emerald-600' : 
											'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}`}
										on:click={() => selectedTab = 'yours'}
										aria-current={selectedTab === 'yours' ? 'page' : undefined}
									>
										Your Photos ({yourPhotos.length})
									</button>
									<button
										class={`w-1/3 py-4 px-1 text-center border-b-2 font-medium text-sm
											${selectedTab === 'in' ? 
											'border-emerald-500 text-emerald-600' : 
											'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}`}
										on:click={() => selectedTab = 'in'}
										aria-current={selectedTab === 'in' ? 'page' : undefined}
									>
										Photos You're In ({photosYoureIn.length})
									</button>
									<button
										class={`w-1/3 py-4 px-1 text-center border-b-2 font-medium text-sm
											${selectedTab === 'all' ? 
											'border-emerald-500 text-emerald-600' : 
											'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}`}
										on:click={() => selectedTab = 'all'}
										aria-current={selectedTab === 'all' ? 'page' : undefined}
									>
										All Photos ({allPhotos.length})
									</button>
								</nav>
							</div>

							<!-- Tab Content -->
							<div class="p-4">
								{#if selectedTab === 'yours'}
									<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
										{#each yourPhotos as photo}
											<Card
												title=""
												uploadDate={photo.uploadedAt}
												image={photo.url}
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
												on:click={() => openAlbum('Your Photos', yourPhotos, yourPhotos.indexOf(photo))}
												on:toggleFaces={() => toggleFacesForPhoto(photo.id)}
											/>
										{/each}
									</div>
								{:else if selectedTab === 'in'}
									{#if (currentUser?.referencePhotos?.length || 0) < 3}
										<div class="text-center py-8">
											<svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
											</svg>
											<h3 class="mt-2 text-sm font-medium text-gray-900">Add More Reference Photos</h3>
											<p class="mt-1 text-sm text-gray-500">You need at least 3 reference photos to use facial recognition. Add more photos in your profile.</p>
											<div class="mt-6">
												<Button href="/profile" variant="primary">Go to Profile</Button>
											</div>
										</div>
									{:else}
										<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
											{#each photosYoureIn as photo}
												<Card
													title={photo.uploadedByName || photo.uploadedBy}
													uploadDate={photo.uploadedAt}
													image={photo.url}
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
													on:click={() => openAlbum('Photos You\'re In', photosYoureIn, photosYoureIn.indexOf(photo))}
													on:toggleFaces={() => toggleFacesForPhoto(photo.id)}
												/>
											{/each}
										</div>
									{/if}
								{:else}
									<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
										{#each allPhotos as photo}
											<Card
												title={photo.uploadedByName || photo.uploadedBy}
												subtitle={photo.peopleInPhoto?.length ? `${photo.peopleInPhoto.length} ${photo.peopleInPhoto.length === 1 ? 'person' : 'people'} tagged` : ''}
												uploadDate={photo.uploadedAt}
												image={photo.url}
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
												on:click={() => openAlbum('All Photos', allPhotos, allPhotos.indexOf(photo))}
												on:toggleFaces={() => toggleFacesForPhoto(photo.id)}
											/>
										{/each}
									</div>
								{/if}
							</div>
						</div>
					{/if}
				</div>
		{/if}
	</div>
</div>

{#if isCarouselOpen}
	<PhotoCarousel
		bind:isOpen={isCarouselOpen}
		photos={selectedPhotos}
		albumTitle={selectedAlbum || "Photo Gallery"}
		currentIndex={currentCarouselIndex}
		showFaces={true}
		on:close={closeCarousel}
		on:delete={handlePhotoDelete}
		currentEvent={event || undefined}
	/>
{/if}

{#if isUploadModalOpen}
	<PhotoUploadModal
		isOpen={true}
		eventId={$page.params.id}
		on:close={() => isUploadModalOpen = false}
		on:upload={handlePhotoUpload}
	/>
{/if}

{#if isShareModalOpen && event}
	<ShareEventModal
		isOpen={true}
		eventId={event.id}
		eventCode={event.code || ''}
		origin={window?.location?.origin || ''}
		on:close={() => isShareModalOpen = false}
	/>
{/if}