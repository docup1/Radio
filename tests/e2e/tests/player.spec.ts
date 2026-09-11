import { expect, test } from '@playwright/test'
import { FIXTURE, radioAudio, seedLibrary } from '../support/seed'

const S1 = 'E2E Song 1'
const S2 = 'E2E Song 2'
const S3 = 'E2E Song 3'

async function expectPlaying(page: import('@playwright/test').Page) {
  await expect
    .poll(
      async () => {
        const s = await radioAudio(page)
        return s.found ? s.paused : null
      },
      { timeout: 30_000 },
    )
    .toBe(false)
}

async function expectAdvancing(page: import('@playwright/test').Page, secs = 0.5) {
  await expect
    .poll(
      async () => {
        const s = await radioAudio(page)
        return s.found ? s.currentTime : null
      },
      { timeout: 30_000 },
    )
    .toBeGreaterThan(secs)
}

async function register(page: import('@playwright/test').Page) {
  await page.goto('/register')
  const username = `e2e_player_${Date.now().toString(36)}`
  await page.locator('input[autocomplete="username"]').fill(username)
  await page.locator('input[type="password"]').fill('password1')
  await page.getByRole('button', { name: 'Зарегистрироваться' }).click()
  await page.waitForURL('**/profile')
  return username
}

test('стрим продолжает играть после скипа и естественного конца песни', async ({ page }) => {
  await register(page)

  await seedLibrary(page, [
    { name: S1, file: FIXTURE },
    { name: S2, file: FIXTURE },
    { name: S3, file: FIXTURE },
  ])

  // старт с профиля
  await page.getByRole('button', { name: /Старт/ }).click()
  await expect(page.locator('.status-text')).toHaveText('В эфире')

  // уходим в плеер
  await page.getByRole('button', { name: /Слушать/ }).click()
  await page.waitForURL(/\/streams\/.+\/listen$/)

  // первая песня реально звучит (currentTime движется)
  await expectPlaying(page)
  await expectAdvancing(page)

  // естественный конец первой песни: переход на вторую, паузы нет
  await expect(page.locator('.stream-player__song')).toContainText(S1, { timeout: 60_000 })
  await expect(page.locator('.stream-player__song')).toContainText(S2, { timeout: 60_000 })
  await expectPlaying(page)
  await expectAdvancing(page)

  // скип: вторая песня -> третья, стрим продолжает играть
  await page.getByRole('button', { name: /Скип/ }).click()
  await expect(page.locator('.stream-player__song')).toContainText(S3, { timeout: 60_000 })
  await expectPlaying(page)
  await expectAdvancing(page, 1)
})