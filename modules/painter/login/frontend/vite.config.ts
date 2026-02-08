import react from '@vitejs/plugin-react'
import path from 'path'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
    base: "/ui",
    plugins: [react()],
    css: {
        modules: {
            localsConvention: "camelCase",
            generateScopedName: "[name]_[local]_[hash:base64:5]"
        }
    },
    resolve: {
        alias: {
            '@': path.resolve(__dirname, './src')
        },
        extensions: ['.mjs', '.js', '.mts', '.ts', '.jsx', '.tsx', '.json']
    },
    server: {
        proxy: {
            '^\/(auth|main-page|geo)': {
                target: 'http://localhost:8080',
                changeOrigin: true,
                headers: {
                    "X-JWT-Name": "painter",
                    "X-JWT-Email": "painter_qiao@qq.com",
                    "X-JWT-Sub": "1",
                    "X-JWT-Role": "admin",
                    "X-JWT-LastLogin": "0",
                }
            },
        }
    },
})
