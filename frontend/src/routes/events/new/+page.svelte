<script lang="ts">
	import Button from '$lib/components/Button.svelte';
	import { createEvent } from '$lib/services/api';
	import { goto } from '$app/navigation';
	import { user } from '$lib/stores';

	let title = '';
	let description = '';
	let date = '';
	let time = '';
	let location = '';
	let isLoading = false;
	let error: string | null = null;

	const today = new Date().toISOString().split('T')[0];

	async function handleSubmit() {
		try {
			isLoading = true;
			error = null;

			const eventDate = new Date(date);
			if (time) {
				const [hours, minutes] = time.split(':');
				eventDate.setHours(parseInt(hours), parseInt(minutes), 0);
			} else {
				eventDate.setHours(0, 0, 0);
			}
			const isoDate = eventDate.toISOString();

			const eventData = {
				title,
				description,
				date: isoDate,
				location
			};

			const event = await createEvent(eventData);
			goto(`/events/${event.id}`);
		} catch (e) {
			console.error('Failed to create event', e);
			error = 'Failed to create event. Please try again.';
			isLoading = false;
		}
	}
</script>

<div class="bg-white">
	<div class="mx-auto max-w-2xl px-4 sm:px-6 lg:px-8 py-8">
		{#if $user}
			<div class="sm:flex sm:items-center">
				<div class="sm:flex-auto">
					<h1 class="text-2xl font-semibold text-gray-900">Create New Event</h1>
					<p class="mt-2 text-sm text-gray-700">
						Create a new event and invite participants to share photos.
					</p>
				</div>
			</div>

			{#if error}
				<div class="mt-4 rounded-md bg-red-50 p-4">
					<div class="flex">
						<div class="text-sm text-red-700">
							{error}
						</div>
					</div>
				</div>
			{/if}

			<form on:submit|preventDefault={handleSubmit} class="mt-8 space-y-6">
				<div class="space-y-4">
					<div>
						<label for="title" class="block text-sm font-medium text-gray-700">Event Title</label>
						<div class="mt-1">
							<input
								type="text"
								name="title"
								id="title"
								bind:value={title}
								class="block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm"
								required
							/>
						</div>
					</div>

					<div>
						<label for="description" class="block text-sm font-medium text-gray-700">Description</label>
						<div class="mt-1">
							<textarea
								name="description"
								id="description"
								rows={3}
								bind:value={description}
								class="block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm"
							></textarea>
						</div>
					</div>

					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<label for="date" class="block text-sm font-medium text-gray-700">Event Date</label>
							<div class="mt-1">
								<input
									type="date"
									name="date"
									id="date"
									bind:value={date}
									min={today}
									class="block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm"
									required
								/>
							</div>
						</div>
						<div>
							<label for="time" class="block text-sm font-medium text-gray-700">Time (Optional)</label>
							<div class="mt-1">
								<input
									type="time"
									name="time"
									id="time"
									bind:value={time}
									class="block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm"
								/>
							</div>
						</div>
					</div>

					<div>
						<label for="location" class="block text-sm font-medium text-gray-700">Location (Optional)</label>
						<div class="mt-1">
							<input
								type="text"
								name="location"
								id="location"
								bind:value={location}
								placeholder="e.g. Central Park, New York"
								class="block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm"
							/>
						</div>
					</div>
				</div>

				<div class="flex justify-end space-x-4">
					<Button variant="outline" href="/events">Cancel</Button>
					<Button 
						type="submit" 
						variant="primary"
						disabled={isLoading}
					>
						{#if isLoading}
							<div class="flex items-center justify-center">
								<div class="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
								Creating...
							</div>
						{:else}
							Create Event
						{/if}
					</Button>
				</div>
			</form>
		{:else}
			<div class="text-center py-12">
				<h2 class="text-2xl font-semibold text-gray-900">Sign in to Create an Event</h2>
				<p class="mt-2 text-sm text-gray-500">
					Create an account or sign in to start creating and organizing photo-sharing events.
				</p>
				<div class="mt-6 space-x-4">
					<Button href="/signin" variant="primary">Sign In</Button>
					<Button href="/signup" variant="outline">Create Account</Button>
				</div>
			</div>
		{/if}
	</div>
</div>