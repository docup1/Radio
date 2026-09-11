import { defineConfig, devices } from '@playwright/test'

const FRONTEND_URL = process.env.FRONTEND_URL || 'http://127.0.0.1:18000'

export default defineConfig({
  testDir: './tests',
  timeout: 150_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: [['list']],
  use: {
    baseURL: FRONTEND_URL,
    locale: 'ru-RU',
    viewport: { width: 1280, height: 800 },
    trace: 'retain-on-failure',
    launchOptions: {
      args: ['--autoplay-policy=no-user-gesture-required'],
    },
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})