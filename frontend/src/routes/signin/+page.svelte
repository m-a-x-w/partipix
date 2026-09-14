<script lang="ts">
	import Button from '$lib/components/Button.svelte';
	import { signIn } from '$lib/services/api';
	import { user } from '$lib/stores';
	import { goto } from '$app/navigation';

	let email = '';
	let password = '';
	let isLoading = false;
	let error: string | null = null;

	async function handleSubmit() {
		if (!email || !password) {
			error = 'Please fill in all fields';
			return;
		}

		isLoading = true;
		error = null;

		try {
			const userData = await signIn({ email, password });
			user.set(userData); // Set user data in store before navigation
			await goto('/profile');
		} catch (e) {
			if (e instanceof Error) {
				error = e.message;
			} else {
				error = 'An unexpected error occurred';
			}
			console.error('Sign in error:', e);
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
	<div class="sm:mx-auto sm:w-full sm:max-w-md">
		<h2 class="mt-6 text-center text-3xl font-bold tracking-tight text-gray-900">
			Sign in to your account
		</h2>
		<p class="mt-2 text-center text-sm text-gray-600">
			Or{' '}
			<a href="/signup" class="font-medium text-emerald-600 hover:text-emerald-500">
				create a new account
			</a>
		</p>
	</div>

	<div class="mt-4 sm:mx-auto sm:w-full sm:max-w-md">
		<div class="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
			{#if error}
				<div class="mb-4 bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded relative" role="alert">
					<span class="block sm:inline">{error}</span>
				</div>
			{/if}

			<form class="space-y-6" on:submit|preventDefault={handleSubmit}>
				<div>
					<label for="email" class="block text-sm font-medium text-gray-700">Email address</label>
					<div class="mt-1">
						<input
							id="email"
							name="email"
							type="email"
							autocomplete="email"
							required
							bind:value={email}
							class="block w-full appearance-none rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-emerald-500 focus:outline-none focus:ring-emerald-500 sm:text-sm"
						/>
					</div>
				</div>

				<div>
					<label for="password" class="block text-sm font-medium text-gray-700">Password</label>
					<div class="mt-1">
						<input
							id="password"
							name="password"
							type="password"
							autocomplete="current-password"
							required
							bind:value={password}
							class="block w-full appearance-none rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-emerald-500 focus:outline-none focus:ring-emerald-500 sm:text-sm"
						/>
					</div>
				</div>

				<div>
					<Button
						type="submit"
						variant="primary"
						className="w-full"
						disabled={isLoading}
					>
						{#if isLoading}
							<div class="flex items-center justify-center">
								<div class="animate-spin rounded-full h-5 w-5 border-b-2 border-white mr-2"></div>
								Signing in...
							</div>
						{:else}
							Sign in
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
</div>