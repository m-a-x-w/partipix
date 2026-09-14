<script lang="ts">
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import { enhance } from '$app/forms';
	import { onMount } from 'svelte';
	import { getEvents, joinEvent } from '$lib/services/api';
	import { user } from '$lib/stores';
	import type { Event as AppEvent } from '$lib/types';
	import { formatDate } from '$lib/utils';

	let events: AppEvent[] = [];
	let eventCode = '';
	let loading = true;
	let error: string | null = null;
	let joining = false;
	let sortOrder: 'newest' | 'oldest' | 'title' = 'newest';
	let unsortedEvents: AppEvent[] = [];

	onMount(async () => {
		try {
			loading = true;
			unsortedEvents = await getEvents();
		} catch (e) {
			console.error('Failed to fetch events', e);
			error = 'Failed to load events. Please try again.';
		} finally {
			loading = false;
		}
	});

	$: events = [...unsortedEvents].sort((a, b) => {
		switch (sortOrder) {
			case 'newest':
				return new Date(b.date).getTime() - new Date(a.date).getTime();
			case 'oldest':
				return new Date(a.date).getTime() - new Date(b.date).getTime();
			case 'title':
				return a.title.localeCompare(b.title);
			default:
				return 0;
		}
	});

	async function handleJoinEvent() {
		if (!eventCode) return;
		
		try {
			joining = true;
			error = null;
			const event = await joinEvent({ code: eventCode });
			unsortedEvents = [event, ...unsortedEvents];
			eventCode = '';
		} catch (e) {
			console.error('Failed to join event', e);
			error = 'Invalid event code or failed to join event.';
		} finally {
			joining = false;
		}
	}

	function handleEventCodeInput(event: { currentTarget: HTMLInputElement }) {
		eventCode = event.currentTarget.value.toUpperCase();
	}
</script>

<div class="bg-white">
	<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 pb-16">
		<div class="mb-8 rounded-lg bg-gray-50 p-6">
			<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-6">
				<div class="flex-shrink-0">
					<h2 class="text-lg font-semibold text-gray-900">Join an Event</h2>
					<p class="mt-1 text-sm text-gray-700">
						{#if $user}
							Enter an event code to join an existing event.
						{:else}
							Sign in to join events and share photos with others.
						{/if}
					</p>
				</div>

				{#if $user}
					{#if error}
						<div class="rounded-md bg-red-50 p-4">
							<div class="flex">
								<div class="text-sm text-red-700">
									{error}
								</div>
							</div>
						</div>
					{/if}

					<form on:submit|preventDefault={handleJoinEvent} class="flex-grow sm:flex sm:items-center sm:justify-end">
						<div class="w-full sm:max-w-md">
							<label for="event-code" class="sr-only">Event Code</label>
							<div class="relative flex items-center gap-3">
								<input
									type="text"
									name="event-code"
									id="event-code"
									value={eventCode}
									on:input={handleEventCodeInput}
									maxlength="8"
									class="block w-full rounded-md border-0 py-2.5 text-center text-lg sm:text-3xl font-bold font-mono tracking-widest text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-emerald-600"
									placeholder="_ _ _ _ _ _ _ _"
								/>
								<Button 
									type="submit" 
									variant="primary"
									disabled={joining || !eventCode}
								>
									{#if joining}
										<div class="flex items-center justify-center">
											<div class="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
											Joining...
										</div>
									{:else}
										Join Event
									{/if}
								</Button>
							</div>
						</div>
					</form>
				{:else}
					<div class="flex space-x-4">
						<Button href="/signin" variant="primary">Sign In</Button>
						<Button href="/signup" variant="outline">Create Account</Button>
					</div>
				{/if}
			</div>
		</div>

		<!-- Events List Section -->
		<div class="sm:flex sm:items-center">
			<div class="sm:flex-auto">
				<h1 class="text-2xl font-semibold text-gray-900">Your Events</h1>
				<p class="mt-2 text-sm text-gray-700">
					Events you've created or joined. These are private to you.
				</p>
			</div>
			<div class="mt-4 sm:mt-0 sm:ml-16 sm:flex-none flex items-center gap-4">
				<select
					bind:value={sortOrder}
					class="rounded-md border-gray-300 py-1.5 pl-3 pr-8 text-sm focus:border-emerald-500 focus:outline-none focus:ring-emerald-500"
				>
					<option value="newest">Newest First</option>
					<option value="oldest">Oldest First</option>
					<option value="title">By Title</option>
				</select>
				<Button href="/events/new" variant="primary">Create Event</Button>
			</div>
		</div>

		{#if loading}
			<div class="mt-8 flex justify-center">
				<div class="animate-spin rounded-full h-10 w-10 border-b-2 border-emerald-600"></div>
			</div>
		{:else if events.length === 0}
			<div class="mt-8 text-center py-12 bg-gray-50 rounded-lg">
				{#if $user}
					<h3 class="text-lg font-medium text-gray-900">No events found</h3>
					<p class="mt-2 text-sm text-gray-500">
						You haven't created or joined any events yet. Create a new event or join one with a code.
					</p>
					<div class="mt-6">
						<Button href="/events/new" variant="primary">Create Event</Button>
					</div>
				{:else}
					<h3 class="text-lg font-medium text-gray-900">Sign in to view events</h3>
					<p class="mt-2 text-sm text-gray-500">
						Create an account or sign in to create and join photo-sharing events.
					</p>
					<div class="mt-6 space-x-4">
						<Button href="/signin" variant="primary">Sign In</Button>
						<Button href="/signup" variant="outline">Create Account</Button>
					</div>
				{/if}
			</div>
		{:else}
			<div class="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
				{#each events as event}
					<Card
						title={event.title}
						description={event.description}
						subtitle={formatDate(event.date)}
						stats={[
							event.guests?.some(g => g.isHost && g.name === $user?.name) ? 
								'Hosting' : null,
							`${event.guests?.length ? `${event.guests.length} guests` : 'No guests yet'}`,
							`Code: ${event.code ? event.code.toUpperCase() : 'N/A'}`
						].filter(Boolean).join(' • ')}
						href="/events/{event.id}"
					/>
				{/each}
			</div>
		{/if}
	</div>
</div>