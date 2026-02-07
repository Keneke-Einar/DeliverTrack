import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // === УМНЫЙ ПРОКСИ ДЛЯ ЛОГИНА ===
      '/login': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
        // 🔥 МАГИЯ ЗДЕСЬ:
        // Если браузер просит HTML (страницу), мы НЕ отправляем это на бэкенд.
        bypass: (req) => {
          if (req.headers.accept && req.headers.accept.includes('text/html')) {
            return req.url; // Оставляем запрос фронтенду
          }
        }
      },
      
      // === УМНЫЙ ПРОКСИ ДЛЯ РЕГИСТРАЦИИ ===
      '/register': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
        bypass: (req) => {
          if (req.headers.accept && req.headers.accept.includes('text/html')) {
            return req.url;
          }
        }
      },

      // Для остальных API запросов просто пересылаем
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      }
    }
  }
})