<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { user } from '$lib/stores';
	import { joinEvent } from '$lib/services/api';

	let error: string | null = null;
	let joining = false;

	onMount(async () => {
		const code = $page.params.id.toUpperCase();

		// Check for authentication
		if (!$user) {
			// Save the intended destination in the URL so we can redirect back after login
			goto(`/signin?redirect=/events/join/${code}`);
			return;
		}

		// Validate code length
		if (code.length !== 8) {
			error = 'Invalid event code. Code must be 8 characters long.';
			return;
		}

		try {
			joining = true;
			const event = await joinEvent({ code });
			goto(`/events/${event.id}`);
		} catch (e) {
			console.error('Failed to join event', e);
			error = 'Invalid event code or failed to join event.';
		} finally {
			joining = false;
		}
	});
</script>

<div class="bg-white min-h-[80vh] flex items-center">
	<div class="mx-auto max-w-2xl px-4 sm:px-6 lg:px-8 py-16 w-full">
		{#if error}
			<div class="text-center">
				<h1 class="text-4xl font-bold tracking-tight text-gray-900 sm:text-5xl mb-8">
					{error}
				</h1>
				<div class="flex justify-center gap-4">
					<a href="/events/join" class="text-emerald-600 hover:text-emerald-500">
						Try another code
					</a>
					<span class="text-gray-300">|</span>
					<a href="/events/new" class="text-emerald-600 hover:text-emerald-500">
						Create your own event
					</a>
				</div>
			</div>
		{:else}
			<div class="text-center">
				<h1 class="text-4xl font-bold tracking-tight text-gray-900 sm:text-5xl">
					Joining Event...
				</h1>
				<div class="mt-8 flex justify-center">
					<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-600"></div>
				</div>
			</div>
		{/if}
	</div>
</div>