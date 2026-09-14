<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Modal from './Modal.svelte';
	import Button from './Button.svelte';
	import type { Photo } from '$lib/types';
	import { uploadEventPhotos } from '$lib/services/api';

	export let isOpen = false;
	export let eventId: string;

	const dispatch = createEventDispatcher<{
		close: void;
		upload: Photo[];
	}>();

	let files: FileList | null = null;
	let isDragging = false;
	let isUploading = false;
	let error: string | null = null;
	let uploadProgress = 0;
	let uploadStage: 'uploading' | 'processing' = 'uploading';
	let detectedFaces = 0;
	let totalFiles = 0;
	let processingStepMessages = [
		"Analyzing images...",
		"Detecting faces...",
		"Running facial recognition...",
		"Comparing with reference photos...",
		"Finalizing results..."
	];
	let currentProcessingStep = 0;

	function handleDragEnter(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		isDragging = true;
	}

	function handleDragLeave(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		isDragging = false;
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
	}

	async function handleDrop(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		isDragging = false;

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

	function updateProgress(percent: number, stage: string) {
		uploadProgress = percent;
		uploadStage = stage as 'uploading' | 'processing';
		
		// Update processing message steps based on progress
		if (stage === 'processing') {
			if (percent >= 85 && percent < 90) currentProcessingStep = 0;
			else if (percent >= 90 && percent < 94) currentProcessingStep = 1;
			else if (percent >= 94 && percent < 96) currentProcessingStep = 2;
			else if (percent >= 96 && percent < 98) currentProcessingStep = 3;
			else if (percent >= 98) currentProcessingStep = 4;
		}
	}

	async function processFiles(files: File[]) {
		const validImageTypes = ['image/jpeg', 'image/png', 'image/webp', 'image/heic', 'image/heif'];
		const maxSize = 10 * 1024 * 1024; // 10MB

		// Validate file types and sizes
		const invalidFiles = files.filter(
			file => !validImageTypes.includes(file.type) || file.size > maxSize
		);

		if (invalidFiles.length > 0) {
			error = 'Invalid files. Please upload only images (JPG, PNG, WebP, HEIC, HEIF) under 10MB';
			return;
		}

		isUploading = true;
		error = null;
		uploadProgress = 0;
		uploadStage = 'uploading';
		totalFiles = files.length;
		currentProcessingStep = 0;

		try {
			const photos = await uploadEventPhotos(eventId, files, updateProgress);
			detectedFaces = photos.reduce((count, photo) => count + (photo.peopleInPhoto?.length || 0), 0);
			
			// Wait a moment to show the facial recognition completion before closing
			setTimeout(() => {
				dispatch('upload', photos);
				dispatch('close');
			}, 1500);
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

	function handleClose() {
		// Only allow closing if not actively uploading
		if (!isUploading) {
			dispatch('close');
		}
	}

	// Format the upload progress message
	$: uploadProgressMessage = uploadStage === 'uploading' 
		? `Uploading... ${uploadProgress}%` 
		: processingStepMessages[currentProcessingStep];
		
	$: progressBarColor = uploadStage === 'uploading' ? 'bg-emerald-500' : 'bg-blue-500';
	
	$: processingComplete = uploadStage === 'processing' && uploadProgress === 100;
</script>

<Modal {isOpen} title="Upload Photos" on:close={handleClose}>
	<div class="p-4">
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
				<div class="max-w-lg flex justify-center px-6 pt-5 pb-6 border-2 border-gray-300 border-dashed rounded-lg hover:border-emerald-500 transition-colors duration-200">
					<div class="space-y-1 text-center w-full">
						{#if isUploading}
							<div class="w-full">
								<div class="mb-2 flex justify-between text-sm text-gray-700">
									<span>{totalFiles} photo{totalFiles !== 1 ? 's' : ''}</span>
									<span>{uploadProgressMessage}</span>
								</div>
								<div class="w-full bg-gray-200 rounded-full h-2.5">
									<div class="h-2.5 rounded-full {progressBarColor} transition-all duration-300" style="width: {uploadProgress}%"></div>
								</div>
								{#if processingComplete}
									<div class="mt-3 text-center text-sm text-emerald-600">
										<svg class="mx-auto h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
										</svg>
										<p>Detected {detectedFaces} faces in your photos</p>
									</div>
								{/if}
							</div>
						{:else}
							<svg class="mx-auto h-12 w-12 text-gray-400" stroke="currentColor" fill="none" viewBox="0 0 48 48" aria-hidden="true">
								<path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
							</svg>
							<div class="relative cursor-pointer rounded-md font-medium text-emerald-600 hover:text-emerald-500 focus-within:outline-none focus-within:ring-2 focus-within:ring-offset-2 focus-within:ring-emerald-500">
								<span>{isDragging ? 'Drop photos here' : 'Upload photos'}</span>
							</div>
							<p class="text-xs text-gray-500 mt-1">
								JPG, PNG, WebP, HEIC, HEIF up to 10MB
							</p>
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

		{#if error}
			<p class="mt-2 text-sm text-red-600">{error}</p>
		{/if}

		<div class="mt-4 flex justify-end">
			<Button variant="outline" on:click={handleClose} disabled={isUploading}>
				{isUploading ? 'Uploading...' : 'Cancel'}
			</Button>
		</div>
	</div>
</Modal>