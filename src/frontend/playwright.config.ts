import { defineConfig, devices } from '@playwright/test'

// Reuse the installed system Chrome in CI/dev environments where Playwright
// browser downloads are intentionally disabled. Override with
// PLAYWRIGHT_EXECUTABLE_PATH when a different Chromium binary is available.
const executablePath = process.env.PLAYWRIGHT_EXECUTABLE_PATH || 'C:/Program Files/Google/Chrome/Application/chrome.exe'

export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  fullyParallel: false,
  reporter: [['list']],
  use: {
    baseURL: 'http://127.0.0.1:28311',
    headless: true,
    browserName: 'chromium',
    launchOptions: { executablePath },
    trace: 'retain-on-failure'
  },
  webServer: {
    command: 'npm run dev -- --mode e2e --host 127.0.0.1 --port 28311',
    url: 'http://127.0.0.1:28311',
    reuseExistingServer: false,
    timeout: 120_000,
    env: {}
  },
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } } },
    // Use Chromium at a mobile viewport so the suite remains runnable on a
    // workstation that has Chrome but no Playwright WebKit download.
    { name: 'mobile', use: { ...devices['Desktop Chrome'], viewport: { width: 390, height: 844 }, isMobile: true } }
  ]
})
