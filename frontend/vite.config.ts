import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// envDir points at the repository root so VITE_* variables live next to the
// backend and CV settings in a single .env file.
export default defineConfig({
  plugins: [react()],
  envDir: '..',
  server: {
    port: 5173,
    host: true,
  },
});
