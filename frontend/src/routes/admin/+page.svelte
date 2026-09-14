<script lang="ts">
	import { onMount } from 'svelte';
	import { user } from '$lib/stores';
	import { getAdminStats, listUsers, updateUser, deleteUser, getAdminEvents, updateEvent, deleteEvent } from '$lib/services/api';
	import { goto } from '$app/navigation';
	import { browser } from '$app/environment';
	import type { AdminStats, User, UpdateUserRequest, Event } from '$lib/types';
	import Card from '$lib/components/Card.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import { formatDate } from '$lib/utils';

	let stats: AdminStats | null = null;
	let users: User[] = [];
	let events: Event[] = [];
	let isLoading = true;
	let error: string | null = null;
	let userEditModalOpen = false;
	let eventEditModalOpen = false;
	let selectedUser: User | null = null;
	let selectedEvent: Event | null = null;
	let editedUser: UpdateUserRequest = {};
	let editedEvent: Partial<Event> = {};

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		isLoading = true;
		error = null;
		try {
			const [statsData, usersData, eventsData] = await Promise.all([
				getAdminStats(),
				listUsers(),
				getAdminEvents()
			]);
			stats = statsData;
			users = usersData;
			events = eventsData;
		} catch (e) {
			error = 'Failed to load admin data';
			console.error(e);
		} finally {
			isLoading = false;
		}
	}

	function openUserEditModal(user: User) {
		selectedUser = user;
		editedUser = {
			name: user.name,
			email: user.email,
			isAdmin: user.isAdmin
		};
		userEditModalOpen = true;
	}

	function openEventEditModal(event: Event) {
		selectedEvent = event;
		editedEvent = {
			title: event.title,
			description: event.description,
			date: event.date,
			location: event.location
		};
		eventEditModalOpen = true;
	}

	async function handleUpdateUser() {
		if (!selectedUser) return;

		try {
			await updateUser(selectedUser.id, editedUser);
			await loadData();
			userEditModalOpen = false;
		} catch (e) {
			error = 'Failed to update user';
			console.error(e);
		}
	}

	async function handleUpdateEvent() {
		if (!selectedEvent) return;

		try {
			await updateEvent(selectedEvent.id, editedEvent);
			await loadData();
			eventEditModalOpen = false;
		} catch (e) {
			error = 'Failed to update event';
			console.error(e);
		}
	}

	async function handleDeleteUser(userId: string) {
		if (!confirm('Are you sure you want to delete this user? This action cannot be undone.')) {
			return;
		}

		try {
			await deleteUser(userId);
			await loadData();
		} catch (e) {
			error = 'Failed to delete user';
			console.error(e);
		}
	}

	async function handleDeleteEvent(eventId: string) {
		if (!confirm('Are you sure you want to delete this event? This action cannot be undone.')) {
			return;
		}

		try {
			await deleteEvent(eventId);
			await loadData();
		} catch (e) {
			error = 'Failed to delete event';
			console.error(e);
		}
	}
</script>

<div class="min-h-screen bg-gray-50 py-8">
	<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
		<h1 class="text-3xl font-bold text-gray-900 mb-8">Admin Dashboard</h1>

		{#if error}
			<div class="bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded mb-6">
				{error}
			</div>
		{/if}

		{#if isLoading}
			<div class="flex justify-center py-12">
				<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-emerald-500"></div>
			</div>
		{:else}
			{#if stats}
				<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4 mb-8">
					<Card
						title="Total Users"
						description={stats.totalUsers.toString()}
					/>
					<Card
						title="Total Events"
						description={stats.totalEvents.toString()}
					/>
					<Card
						title="Total Photos"
						description={stats.totalPhotos.toString()}
					/>
					<Card
						title="Active Events"
						description={stats.activeEvents.toString()}
					/>
				</div>
			{/if}

			<Card title="Users">
				<div class="px-4 py-5 sm:p-6">
					<h2 class="text-lg font-medium text-gray-900 mb-4">Users</h2>
					<div class="overflow-x-auto">
						<table class="min-w-full divide-y divide-gray-200">
							<thead>
								<tr>
									<th class="px-6 py-3 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
									<th class="px-6 py-3 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Email</th>
									<th class="px-6 py-3 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Admin</th>
									<th class="px-6 py-3 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created</th>
									<th class="px-6 py-3 bg-gray-50 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
								</tr>
							</thead>
							<tbody class="bg-white divide-y divide-gray-200">
								{#each users as user}
									<tr>
										<td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
											{user.name}
										</td>
										<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
											{user.email}
										</td>
										<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
											{#if user.isAdmin}
												<span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-emerald-100 text-emerald-800">Yes</span>
											{:else}
												<span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-gray-100 text-gray-800">No</span>
											{/if}
										</td>
										<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
											{user.createdAt ? formatDate(user.createdAt) : 'N/A'}
										</td>
										<td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
											<button
												class="text-emerald-600 hover:text-emerald-900 mr-4"
												on:click={() => openUserEditModal(user)}>
												Edit
											</button>
											<button
												class="text-red-600 hover:text-red-900"
												on:click={() => handleDeleteUser(user.id)}>
												Delete
											</button>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			</Card>

			<div class="mt-8">
				<Card title="Events">
					<div class="px-4 py-5 sm:p-6">
						<h2 class="text-lg font-medium text-gray-900 mb-4">Events</h2>
						<div class="overflow-x-auto">
							<table class="min-w-full divide-y divide-gray-200">
								<thead>
									<tr>
										<th class="px-6 py-3 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Title</th>
										<th class="px-6 py-3 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Date</th>
										<th class="px-6 py-3 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Location</th>
										<th class="px-6 py-3 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Participants</th>
										<th class="px-6 py-3 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Code</th>
										<th class="px-6 py-3 bg-gray-50 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
									</tr>
								</thead>
								<tbody class="bg-white divide-y divide-gray-200">
									{#each events as event}
										<tr>
											<td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
												{event.title}
											</td>
											<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
												{formatDate(event.date)}
											</td>
											<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
												{event.location || 'N/A'}
											</td>
											<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
												{event.participants}
											</td>
											<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
												{event.code ? event.code.toUpperCase() : 'N/A'}
											</td>
											<td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
												<button
													class="text-emerald-600 hover:text-emerald-900 mr-4"
													on:click={() => openEventEditModal(event)}>
													Edit
												</button>
												<button
													class="text-red-600 hover:text-red-900"
													on:click={() => handleDeleteEvent(event.id)}>
													Delete
												</button>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</div>
				</Card>
			</div>
		{/if}
	</div>
</div>

<Modal bind:isOpen={userEditModalOpen} title="Edit User">
	{#if selectedUser}
		<form on:submit|preventDefault={handleUpdateUser} class="space-y-4">
			<div>
				<label for="name" class="block text-sm font-medium text-gray-700">Name</label>
				<input
					type="text"
					id="name"
					bind:value={editedUser.name}
					class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm"
				/>
			</div>
			<div>
				<label for="email" class="block text-sm font-medium text-gray-700">Email</label>
				<input
					type="email"
					id="email"
					bind:value={editedUser.email}
					class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm"
				/>
			</div>
			<div class="flex items-center">
				<input
					type="checkbox"
					id="isAdmin"
					bind:checked={editedUser.isAdmin}
					class="h-4 w-4 text-emerald-600 focus:ring-emerald-500 border-gray-300 rounded"
				/>
				<label for="isAdmin" class="ml-2 block text-sm text-gray-900">Admin privileges</label>
			</div>
			<div class="mt-5 sm:mt-6 space-x-3">
				<Button type="submit">Save Changes</Button>
				<Button type="button" variant="secondary" on:click={() => userEditModalOpen = false}>Cancel</Button>
			</div>
		</form>
	{/if}
</Modal>

<Modal bind:isOpen={eventEditModalOpen} title="Edit Event">
	{#if selectedEvent}
		<form on:submit|preventDefault={handleUpdateEvent} class="space-y-4">
			<div>
				<label for="title" class="block text-sm font-medium text-gray-700">Title</label>
				<input
					type="text"
					id="title"
					bind:value={editedEvent.title}
					class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm"
				/>
			</div>
			<div>
				<label for="description" class="block text-sm font-medium text-gray-700">Description</label>
				<textarea
					id="description"
					bind:value={editedEvent.description}
					rows="3"
					class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm"
				></textarea>
			</div>
			<div>
				<label for="date" class="block text-sm font-medium text-gray-700">Date</label>
				<input
					type="datetime-local"
					id="date"
					bind:value={editedEvent.date}
					class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm"
				/>
			</div>
			<div>
				<label for="location" class="block text-sm font-medium text-gray-700">Location</label>
				<input
					type="text"
					id="location"
					bind:value={editedEvent.location}
					class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-emerald-500 focus:ring-emerald-500 sm:text-sm"
				/>
			</div>
			<div class="mt-5 sm:mt-6 space-x-3">
				<Button type="submit">Save Changes</Button>
				<Button type="button" variant="secondary" on:click={() => eventEditModalOpen = false}>Cancel</Button>
			</div>
		</form>
	{/if}
</Modal>