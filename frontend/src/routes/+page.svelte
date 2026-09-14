<script lang="ts">
	import Button from '$lib/components/Button.svelte';
	import { user } from '$lib/stores';
</script>

<!-- Main Container with Background -->
<div class="relative isolate overflow-hidden bg-white w-full">
	<!-- Background Elements -->
	<div class="fixed inset-0 -z-20 transform-gpu overflow-hidden blur-3xl">
		<div class="relative left-[calc(50%-11rem)] aspect-[1155/678] w-[36.125rem] -translate-x-1/2 rotate-[30deg] bg-gradient-to-tr from-emerald-600 to-teal-600 opacity-20 sm:left-[calc(50%-30rem)] sm:w-[72.1875rem] animate-[move 25s linear infinite]" style="clip-path: polygon(74.1% 44.1%, 100% 61.6%, 97.5% 26.9%, 85.5% 0.1%, 80.7% 2%, 72.5% 32.5%, 60.2% 62.4%, 52.4% 68.1%, 47.5% 58.3%, 45.2% 34.5%, 27.5% 76.7%, 0.1% 64.9%, 17.9% 100%, 27.6% 76.8%, 76.1% 97.7%, 74.1% 44.1%)"></div>
	</div>
	<div class="fixed inset-0 -z-20 transform-gpu overflow-hidden blur-3xl">
		<div class="relative left-[calc(50%+3rem)] aspect-[1155/678] w-[36.125rem] -translate-x-1/2 bg-gradient-to-tr from-emerald-600 to-teal-600 opacity-20 sm:left-[calc(50%+36rem)] sm:w-[72.1875rem] animate-[move 30s linear infinite reverse]" style="clip-path: polygon(74.1% 44.1%, 100% 61.6%, 97.5% 26.9%, 85.5% 0.1%, 80.7% 2%, 72.5% 32.5%, 60.2% 62.4%, 52.4% 68.1%, 47.5% 58.3%, 45.2% 34.5%, 27.5% 76.7%, 0.1% 64.9%, 17.9% 100%, 27.6% 76.8%, 76.1% 97.7%, 74.1% 44.1%)"></div>
	</div>
	<div class="fixed inset-0 -z-20 transform-gpu overflow-hidden blur-3xl">
		<div class="relative left-[calc(50%-3rem)] aspect-[1155/678] w-[36.125rem] -translate-x-1/2 rotate-[15deg] bg-gradient-to-tr from-emerald-600 to-teal-600 opacity-20 sm:left-[calc(50%-36rem)] sm:w-[72.1875rem] animate-[move 28s linear infinite]" style="clip-path: polygon(74.1% 44.1%, 100% 61.6%, 97.5% 26.9%, 85.5% 0.1%, 80.7% 2%, 72.5% 32.5%, 60.2% 62.4%, 52.4% 68.1%, 47.5% 58.3%, 45.2% 34.5%, 27.5% 76.7%, 0.1% 64.9%, 17.9% 100%, 27.6% 76.8%, 76.1% 97.7%, 74.1% 44.1%)"></div>
	</div>
	<div class="fixed inset-0 -z-20 transform-gpu overflow-hidden blur-3xl">
		<div class="relative left-[calc(50%+15rem)] aspect-[1155/678] w-[36.125rem] -translate-x-1/2 rotate-[-15deg] bg-gradient-to-tr from-emerald-600 to-teal-600 opacity-20 sm:left-[calc(50%+45rem)] sm:w-[72.1875rem] animate-[move 32s linear infinite reverse]" style="clip-path: polygon(74.1% 44.1%, 100% 61.6%, 97.5% 26.9%, 85.5% 0.1%, 80.7% 2%, 72.5% 32.5%, 60.2% 62.4%, 52.4% 68.1%, 47.5% 58.3%, 45.2% 34.5%, 27.5% 76.7%, 0.1% 64.9%, 17.9% 100%, 27.6% 76.8%, 76.1% 97.7%, 74.1% 44.1%)"></div>
	</div>

	<!-- Hero Section -->
	<section class="relative isolate overflow-hidden">
		<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-24 sm:py-32">
			<div class="text-center">
				<h1 class="text-5xl font-bold tracking-tight text-gray-900 sm:text-6xl">
					Share Your Photos with{' '}
					<span class="text-emerald-600">Smart Organization</span> and{' '}
					<span class="text-emerald-600">Privacy</span>
				</h1>
				<p class="mt-6 text-xl leading-8 text-gray-600 max-w-2xl mx-auto">
					Create private photo-sharing events where facial recognition automatically organizes photos by person.
					Perfect for weddings, parties, and special occasions.
				</p>
				<div class="mt-10 flex items-center justify-center gap-x-6">
					{#if $user}
						<Button href="/events/new" variant="primary" size="lg">Create Event</Button>
						<Button href="/events/join" variant="outline" size="lg">Join Event</Button>
					{:else}
						<Button href="/signup" variant="primary" size="lg">Create Account</Button>
						<Button href="/events/join" variant="outline" size="lg">Join Event</Button>
					{/if}
				</div>
			</div>
		</div>
	</section>

	<!-- Key Features Section -->
	<section class="relative isolate py-24 sm:py-32">
		<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
			<div class="mx-auto max-w-2xl text-center">
				<h2 class="text-4xl font-bold tracking-tight text-gray-900 sm:text-5xl">Key Features</h2>
				<p class="mt-6 text-lg leading-8 text-gray-600">
					Everything you need to share and organize your event photos
				</p>
			</div>
			<div class="mx-auto mt-16 w-full">
				<div class="grid grid-cols-1 gap-x-8 gap-y-16 lg:grid-cols-3">
					<div class="bg-white rounded-2xl shadow-lg p-8 relative overflow-hidden">
						<div class="absolute inset-0 bg-gradient-to-r from-emerald-100 to-teal-100 opacity-50"></div>
						<div class="relative">
							<div class="h-12 w-12 rounded-lg bg-emerald-600 flex items-center justify-center mb-6">
								<svg class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />
								</svg>
							</div>
							<h3 class="text-xl font-semibold text-gray-900">Smart Organization</h3>
							<p class="mt-4 text-gray-600">
								Facial recognition automatically groups photos by person, making it easy to find all photos of
								yourself or others.
							</p>
						</div>
					</div>
					<div class="bg-white rounded-2xl shadow-lg p-8 relative overflow-hidden">
						<div class="absolute inset-0 bg-gradient-to-r from-teal-100 to-emerald-100 opacity-50"></div>
						<div class="relative">
							<div class="h-12 w-12 rounded-lg bg-emerald-600 flex items-center justify-center mb-6">
								<svg class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" />
								</svg>
							</div>
							<h3 class="text-xl font-semibold text-gray-900">Privacy Controls</h3>
							<p class="mt-4 text-gray-600">
								Control which photos of you are shared with the group. Blacklist photos you don't want others to see
							</p>
						</div>
					</div>
					<div class="bg-white rounded-2xl shadow-lg p-8 relative overflow-hidden">
						<div class="absolute inset-0 bg-gradient-to-r from-green-100 to-emerald-100 opacity-50"></div>
						<div class="relative">
							<div class="h-12 w-12 rounded-lg bg-emerald-600 flex items-center justify-center mb-6">
								<svg class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
								</svg>
							</div>
							<h3 class="text-xl font-semibold text-gray-900">Easy Sharing</h3>
							<p class="mt-4 text-gray-600">
								Share your event with friends and family. They can easily upload and view photos from any
								device.
							</p>
						</div>
					</div>
				</div>
			</div>
		</div>
	</section>

	<!-- How It Works Section -->
	<section class="relative isolate py-24 sm:py-32">
		<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
			<div class="mx-auto max-w-2xl text-center">
				<h2 class="text-4xl font-bold tracking-tight text-gray-900 sm:text-5xl">How It Works</h2>
				<p class="mt-6 text-lg leading-8 text-gray-600">
					Simple steps to create and manage your photo-sharing event
				</p>
			</div>
			<div class="mx-auto mt-16 w-full">
				<div class="grid grid-cols-1 gap-x-8 gap-y-16 lg:grid-cols-3">
					<div class="bg-white rounded-2xl shadow-lg p-8 relative overflow-hidden">
						<div class="absolute inset-0 bg-gradient-to-r from-emerald-100 to-teal-100 opacity-50"></div>
						<div class="relative">
							<div class="h-12 w-12 rounded-lg bg-emerald-600 flex items-center justify-center mb-6">
								<svg class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
								</svg>
							</div>
							<h3 class="text-xl font-semibold text-gray-900">Create an Event</h3>
							<p class="mt-4 text-gray-600">
								Set up your event with a title, description, and date. You can also add a location and
								customize privacy settings.
							</p>
						</div>
					</div>
					<div class="bg-white rounded-2xl shadow-lg p-8 relative overflow-hidden">
						<div class="absolute inset-0 bg-gradient-to-r from-teal-100 to-emerald-100 opacity-50"></div>
						<div class="relative">
							<div class="h-12 w-12 rounded-lg bg-emerald-600 flex items-center justify-center mb-6">
								<svg class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z" />
								</svg>
							</div>
							<h3 class="text-xl font-semibold text-gray-900">Invite Participants</h3>
							<p class="mt-4 text-gray-600">
								Share the event link with your guests. They can join and start uploading photos right
								away.
							</p>
						</div>
					</div>
					<div class="bg-white rounded-2xl shadow-lg p-8 relative overflow-hidden">
						<div class="absolute inset-0 bg-gradient-to-r from-green-100 to-emerald-100 opacity-50"></div>
						<div class="relative">
							<div class="h-12 w-12 rounded-lg bg-emerald-600 flex items-center justify-center mb-6">
								<svg class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 001.5-1.5V6a1.5 1.5 0 00-1.5-1.5H3.75A1.5 1.5 0 002.25 6v12a1.5 1.5 0 001.5 1.5zm10.5-11.25h.008v.008h-.008V8.25zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z" />
								</svg>
							</div>
							<h3 class="text-xl font-semibold text-gray-900">Share and Enjoy</h3>
							<p class="mt-4 text-gray-600">
								View and download photos from your event. You control who can see and download the
								photos.
							</p>
						</div>
					</div>
				</div>
			</div>
		</div>
	</section>

	<!-- Call to Action Section -->
	<section class="relative isolate py-24 sm:py-32">
		<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
			<div class="mx-auto max-w-2xl text-center">
				<h2 class="text-4xl font-bold tracking-tight text-gray-900 sm:text-5xl">Ready to Start Sharing?</h2>
				<p class="mt-6 text-lg leading-8 text-gray-600">
					{#if $user}
						Create a new event or join an existing one to start sharing photos.
					{:else}
						Create your account now and start organizing your event photos with smart facial recognition.
					{/if}
				</p>
				<div class="mt-10 flex items-center justify-center gap-x-6">
					{#if $user}
						<Button href="/events/new" variant="primary" size="lg">Create Event</Button>
						<Button href="/events/join" variant="outline" size="lg">Join Event</Button>
					{:else}
						<Button href="/signup" variant="primary" size="lg">Sign Up Now</Button>
						<Button href="/events/join" variant="outline" size="lg">Join an Event</Button>
					{/if}
				</div>
			</div>
		</div>
	</section>

	<!-- Footer Section -->
	<footer class="bg-gray-900 py-6 w-full">
		<div class="mx-auto w-full px-4 sm:px-6 lg:px-8">
			<div class="flex flex-col md:flex-row justify-between items-center gap-4">
				<a href="/" class="text-2xl font-bold text-white">PartiPix</a>
				<div class="flex flex-wrap justify-center gap-x-8 gap-y-2">
					<a href="/about" class="text-sm text-gray-400 hover:text-white transition-colors">About</a>
					<a href="/privacy" class="text-sm text-gray-400 hover:text-white transition-colors">Privacy</a>
					<a href="/terms" class="text-sm text-gray-400 hover:text-white transition-colors">Terms</a>
					<a href="/contact" class="text-sm text-gray-400 hover:text-white transition-colors">Contact</a>
				</div>
			</div>
			<div class="mt-4 border-t border-gray-800 pt-4">
				<p class="text-center text-sm text-gray-400">© 2025 PartiPix. All rights reserved.</p>
			</div>
		</div>
	</footer>
</div>

<style>
	@keyframes move {
		0% {
			transform: translateY(0) rotate(0deg);
		}
		50% {
			transform: translateY(-100px) rotate(180deg);
		}
		100% {
			transform: translateY(0) rotate(360deg);
		}
	}
</style>
