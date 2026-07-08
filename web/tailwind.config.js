/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // Reuses the repo-root /frontend brand palette to reduce divergence.
        brand: {
          50: '#eef7f5',
          100: '#d6ebe6',
          200: '#b0dcd4',
          500: '#1f7a6b',
          700: '#14594e',
          950: '#0c2320',
        },
      },
    },
  },
  plugins: [],
};
