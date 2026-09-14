<script lang="ts">
    import QRCode from 'qrcode';
    import { onMount } from 'svelte';

    export let url: string;
    export let size: number = 200;
    let canvas: HTMLCanvasElement;

    onMount(() => {
        QRCode.toCanvas(canvas, url, {
            width: size,
            margin: 2,
            color: {
                dark: '#059669', // emerald-600
                light: '#ffffff'
            }
        });
    });
</script>

<canvas 
    bind:this={canvas}
    class="rounded-lg"
    width={size}
    height={size}
></canvas>
