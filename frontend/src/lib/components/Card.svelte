<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import { formatShortDate } from '$lib/utils';
	import type { FaceMatch } from '$lib/types';
	
	export let title: string;
	export let description: string | undefined = undefined;
	export let image: string | undefined = undefined;
	export let href: string | undefined = undefined;
	export let subtitle: string | undefined = undefined;
	export let stats: string | undefined = undefined;
	export let showDownload: boolean = false;
	export let onDownload: (() => void) | undefined = undefined;
	export let uploadDate: string | undefined = undefined;
	export let faceMatches: FaceMatch[] | undefined = undefined;
	export let showFaces: boolean = false;

	const dispatch = createEventDispatcher();

	let imageElement: HTMLImageElement;
	
	// Function to calculate scaled coordinates
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

	function handleDownload(e: MouseEvent) {
		e.stopPropagation(); // Prevent card click when downloading
		if (onDownload) {
			onDownload();
		} else if (image) {
			const a = document.createElement('a');
			a.href = image;
			a.download = 'photo.jpg'; // Default name
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
		}
	}

	function toggleFaces(e: MouseEvent) {
		e.stopPropagation(); // Prevent card click
		dispatch('toggleFaces');
	}

	// Different confidence thresholds for boxes and tags
	$: veryHighConfidenceMatches = faceMatches?.filter(match => match.confidence >= 0.8) ?? []; // 80% threshold for boxes
	
	// Create a filtered list that eliminates duplicates for display
	$: uniqueUserMatches = (() => {
		// Start with all medium confidence matches (70%+)
		const allMediumMatches = faceMatches?.filter(match => match.confidence >= 0.7) ?? [];
		
		// Create a map to track best match per user
		const bestMatchPerUser = new Map();
		
		// Find the best match for each user
		allMediumMatches.forEach(match => {
			if (!bestMatchPerUser.has(match.userId) || bestMatchPerUser.get(match.userId).confidence < match.confidence) {
				bestMatchPerUser.set(match.userId, match);
			}
		});
		
		// Convert Map values back to array
		return Array.from(bestMatchPerUser.values());
	})();
	
	// Ensure we don't display more tags than faces actually detected
	$: actualFaceCount = veryHighConfidenceMatches.length > 0 
		? veryHighConfidenceMatches.length 
		: (faceMatches && faceMatches.length > 0 ? faceMatches.length : 0);
	
	// Limit displayed tags to actual face count, taking highest confidence ones first
	$: mediumConfidenceMatches = uniqueUserMatches.sort((a, b) => b.confidence - a.confidence).slice(0, actualFaceCount);
	
	$: hasFaces = (faceMatches && faceMatches.length > 0); // Show icon if any faces are detected
</script>

<div 
	class="bg-white overflow-hidden shadow rounded-lg {!href ? 'cursor-pointer hover:shadow-md transition-shadow duration-200' : ''}"
	role="button"
	tabindex="0"
	on:click
	on:keydown
>
	{#if image}
		<div class="relative aspect-w-16 aspect-h-9 group">
			<img 
				bind:this={imageElement}
				src={image} 
				alt={title} 
				class="object-cover w-full h-full"
			/>
			
			{#if showFaces && hasFaces}
				{#each veryHighConfidenceMatches as match}
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

			{#if showDownload}
				<button
					class="absolute top-2 right-2 p-2 rounded-full bg-white/80 hover:bg-white text-gray-800 opacity-0 group-hover:opacity-100 transition-opacity duration-200"
					on:click={handleDownload}
					aria-label="Download photo"
				>
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
					</svg>
				</button>
			{/if}

			{#if hasFaces}
				<button
					class="absolute top-2 {showDownload ? 'right-12' : 'right-2'} p-2 rounded-full bg-white/80 hover:bg-white text-gray-800 opacity-0 group-hover:opacity-100 transition-opacity duration-200"
					on:click={toggleFaces}
					aria-label={showFaces ? "Hide face tags" : "Show face tags"}
				>
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 0 0118 0z" />
					</svg>
				</button>
			{/if}
		</div>
	{/if}
	{#if $$slots.default}
		<slot />
	{:else}
		<div class="px-4 py-5 sm:p-6">
			<h3 class="text-lg font-medium text-gray-900">{title}</h3>
			{#if subtitle}
				<p class="mt-1 text-sm text-gray-600 font-medium">{subtitle}</p>
			{/if}
			{#if description}
				<p class="mt-1 text-sm text-gray-500">{description}</p>
			{/if}
			{#if stats}
				<p class="mt-2 text-xs space-x-2">
					{#if stats.includes('Hosting')}
						<span class="inline-flex items-center rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-medium text-blue-800">
							Hosting
						</span>
					{/if}
					<span class="text-gray-500">{stats.replace('Hosting • ', '')}</span>
				</p>
			{/if}
			
			{#if mediumConfidenceMatches.length > 0}
				<div class="mt-2">
					<p class="text-xs font-medium text-gray-700">People in this photo:</p>
					<div class="mt-1 flex flex-wrap gap-1">
						{#each mediumConfidenceMatches as match}
							<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-emerald-100 text-emerald-800">
								{match.userName} ({Math.round(match.confidence * 100)}%)
							</span>
						{/each}
					</div>
				</div>
			{/if}
			
			{#if uploadDate}
				<p class="mt-1 text-xs text-gray-500">{formatShortDate(uploadDate)}</p>
			{/if}
			{#if href}
				<div class="mt-4">
					<a {href} class="text-sm font-medium text-emerald-600 hover:text-emerald-500">
						View details
						<span aria-hidden="true"> →</span>
					</a>
				</div>
			{/if}
		</div>
	{/if}
</div>