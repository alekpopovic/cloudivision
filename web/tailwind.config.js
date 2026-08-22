/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./src/**/*.{html,ts}"
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#EEF3FF',
          100: '#DCE7FF',
          200: '#B9CEFF',
          500: '#4F7CFF',
          700: '#3159C7',
          800: '#1A3A68',
          900: '#10284A',
          950: '#081426',
          teal: '#2DD4BF'
        }
      },
      boxShadow: {
        panel: '0 18px 48px -28px rgb(8 20 38 / 0.25)'
      }
    }
  },
  plugins: []
};
