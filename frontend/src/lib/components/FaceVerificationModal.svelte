<script lang="ts">
	import Modal from './Modal.svelte';
	import Button from './Button.svelte';
	
	export let isOpen = false;
	export let faceImageUrl: string;
	export let onConfirm: () => void;
	export let onReject: () => void;
</script>

<Modal {isOpen} title="Verify Your Face" on:close={onReject}>
	<div class="p-4">
		<p class="text-sm text-gray-600 mb-4">
			We detected this face in your photo. Please confirm this is you before we save it as a reference photo.
		</p>
		
		<div class="aspect-square w-64 mx-auto mb-4">
			<img 
				src={faceImageUrl} 
				alt="Detected face" 
				class="w-full h-full object-cover rounded-lg"
			/>
		</div>

		<div class="flex justify-end space-x-3">
			<Button variant="outline" on:click={onReject}>That's Not Me</Button>
			<Button variant="primary" on:click={onConfirm}>Yes, That's Me</Button>
		</div>
	</div>
</Modal>