<script lang="ts">
	export let variant: 'primary' | 'secondary' | 'outline' = 'primary';
	export let size: 'sm' | 'md' | 'lg' = 'md';
	export let href: string | undefined = undefined;
	export let type: 'button' | 'submit' | 'reset' = 'button';
	export let disabled: boolean = false;
	export let className: string = '';

	const classes = {
		primary: 'bg-emerald-600 text-white hover:bg-emerald-500 disabled:bg-emerald-300',
		secondary: 'bg-gray-600 text-white hover:bg-gray-500 disabled:bg-gray-300',
		outline: 'bg-transparent text-gray-900 border border-gray-300 hover:bg-gray-50 disabled:bg-gray-50 disabled:text-gray-400'
	};

	const sizes = {
		sm: 'px-3 py-2 text-sm',
		md: 'px-4 py-2 text-base',
		lg: 'px-6 py-3 text-lg'
	};

	$: baseClasses = `inline-flex items-center justify-center rounded-md font-medium focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 ${classes[variant]} ${sizes[size]} ${className} disabled:cursor-not-allowed`;
</script>

{#if href && !disabled}
	<a
		{href}
		class={baseClasses}
	>
		<slot />
	</a>
{:else}
	<button
		{type}
		{disabled}
		class={baseClasses}
		on:click
	>
		<slot />
	</button>
{/if}