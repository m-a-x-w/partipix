<script lang="ts">
	import Button from '$lib/components/Button.svelte';
	import { joinEvent } from '$lib/services/api';
	import { user } from '$lib/stores';
	import { goto } from '$app/navigation';

	let eventCode = '';
	let error: string | null = null;
	let joining = false;

	function handleEventCodeInput(event: { currentTarget: HTMLInputElement }) {
		eventCode = event.currentTarget.value.toUpperCase();
	}

	async function handleSubmit() {
		if (!eventCode) return;
		
		try {
			joining = true;
			error = null;
			const event = await joinEvent({ code: eventCode });
			goto(`/events/${event.id}`);
		} catch (e) {
			console.error('Failed to join event', e);
			error = 'Invalid event code or failed to join event.';
		} finally {
			joining = false;
		}
	}
</script>

<div class="bg-white min-h-[80vh] flex items-center">
	<div class="mx-auto max-w-2xl px-4 sm:px-6 lg:px-8 py-16 w-full">
		<div class="text-center">
			<h1 class="text-4xl font-bold tracking-tight text-gray-900 sm:text-5xl">
				Join an Event
			</h1>
			<p class="mt-4 text-lg leading-8 text-gray-600">
				{#if $user}
					Enter the event code to join and start sharing photos.
				{:else}
					Create an account or sign in to join events and share photos with others.
				{/if}
			</p>
		</div>

		{#if $user}
			<div class="mt-16">
				{#if error}
					<div class="mb-6 rounded-md bg-red-50 p-4">
						<div class="flex">
							<div class="text-sm text-red-700">
								{error}
							</div>
						</div>
					</div>
				{/if}

				<form on:submit|preventDefault={handleSubmit} class="space-y-8">
					<div>
						<label for="event-code" class="block text-sm font-medium text-center text-gray-700 mb-4">
							Enter Event Code
						</label>
						<div class="relative flex flex-col items-center gap-6">
							<input
								type="text"
								name="event-code"
								id="event-code"
								value={eventCode}
								on:input={handleEventCodeInput}
								maxlength="8"
								class="block w-full max-w-md rounded-md border-0 py-4 text-center text-xl sm:text-4xl font-bold font-mono tracking-widest text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-emerald-600"
								placeholder="_ _ _ _ _ _ _ _"
							/>
							<Button 
								type="submit" 
								variant="primary"
								size="lg"
								disabled={joining || !eventCode}
								className="w-full max-w-md"
							>
								{#if joining}
									<div class="flex items-center justify-center">
										<div class="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
										Joining Event...
									</div>
								{:else}
									Join Event
								{/if}
							</Button>
						</div>
					</div>
				</form>

				<p class="mt-8 text-center text-sm text-gray-500">
					Don't have an event code? <a href="/events/new" class="text-emerald-600 hover:text-emerald-500">Create your own event</a>
				</p>
			</div>
		{:else}
			<div class="mt-16 flex flex-col items-center gap-4">
				<Button href="/signin" variant="primary" size="lg">Sign In</Button>
				<Button href="/signup" variant="outline" size="lg">Create Account</Button>
			</div>
		{/if}
	</div>
</div>