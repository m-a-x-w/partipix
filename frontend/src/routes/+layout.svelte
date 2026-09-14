<script lang="ts">
	import '../app.css';
	import Button from '$lib/components/Button.svelte';
	import { user } from '$lib/stores';
	import type { User } from '$lib/types';

	let currentUser: User | null = null;
	user.subscribe(value => {
		currentUser = value;
	});
</script>

<div class="min-h-screen bg-gray-50">
	<nav class="bg-white shadow-sm w-full relative z-10">
		<div class="max-w-[90rem] mx-auto px-4 sm:px-6 lg:px-8">
			<div class="flex justify-between h-16">
				<div class="flex">
					<div class="flex-shrink-0 flex items-center">
						<a href="/" class="text-xl font-bold text-emerald-600">Partipix</a>
					</div>
					<div class="hidden sm:ml-6 sm:flex sm:space-x-8">
						<a
							href="/events"
							class="border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
						>
							Events
						</a>
						<a
							href="/photos"
							class="border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
						>
							Photos of You
						</a>
						{#if currentUser?.isAdmin}
							<a
								href="/admin"
								class="border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
							>
								Admin
							</a>
						{/if}
					</div>
				</div>
				<div class="hidden sm:ml-6 sm:flex sm:items-center">
					{#if currentUser}
						<Button href="/profile" variant="primary">Profile</Button>
					{:else}
						<Button href="/signin" variant="primary">Sign In</Button>
					{/if}
				</div>
			</div>
		</div>
	</nav>

	<main class="flex-1 bg-white min-h-[calc(100vh-4rem)]">
		<slot />
	</main>
</div>