<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import QRCode from './QRCode.svelte';
    import Button from './Button.svelte';

    export let isOpen: boolean = false;
    export let eventId: string;
    export let eventCode: string;
    export let origin: string;

    const dispatch = createEventDispatcher();
    const shareUrl = `${origin}/events/join/${eventCode}`;

    function close() {
        dispatch('close');
    }

    async function copyToClipboard(text: string) {
        try {
            await navigator.clipboard.writeText(text);
        } catch (error) {
            console.error('Failed to copy to clipboard:', error);
        }
    }
</script>

{#if isOpen}
    <div class="fixed z-10 inset-0 overflow-y-auto" aria-labelledby="modal-title" role="dialog" aria-modal="true">
        <div class="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
            <div class="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" aria-hidden="true" on:click={close}></div>

            <span class="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">&#8203;</span>

            <div class="inline-block align-bottom bg-white rounded-lg px-4 pt-5 pb-4 text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-sm sm:w-full sm:p-6">
                <div>
                    <div class="text-center">
                        <h3 class="text-lg leading-6 font-medium text-gray-900" id="modal-title">
                            Share Event
                        </h3>
                        <div class="mt-4 flex justify-center">
                            <QRCode url={shareUrl} size={200} />
                        </div>
                        <div class="mt-4">
                            <p class="text-sm text-gray-500">
                                Share this QR code or event code with others to invite them to the event
                            </p>
                            <div class="mt-4">
                                <div class="flex items-center justify-center space-x-2">
                                    <span class="inline-flex items-center px-3 py-1.5 rounded-md text-xl font-mono font-bold bg-gray-100 text-gray-800">
                                        {eventCode}
                                    </span>
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        on:click={() => copyToClipboard(eventCode)}
                                    >
                                        Copy
                                    </Button>
                                </div>
                                <div class="mt-4">
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        className="w-full"
                                        on:click={() => copyToClipboard(shareUrl)}
                                    >
                                        Copy Link
                                    </Button>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="mt-5 sm:mt-6">
                    <Button variant="primary" className="w-full" on:click={close}>
                        Close
                    </Button>
                </div>
            </div>
        </div>
    </div>
{/if}
