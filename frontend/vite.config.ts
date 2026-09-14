import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/uploads': {
				target: 'http://localhost:8080',
				changeOrigin: true
			},
			'/events/uploads': {
				target: 'http://localhost:8080',
				changeOrigin: true,
				rewrite: (path) => path.replace(/^\/events\/uploads/, '/uploads')
			}
		}
	}
});
