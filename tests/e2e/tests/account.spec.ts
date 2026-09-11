import { expect, test } from '@playwright/test'

test('в личном кабинете и настройках нет кнопки удаления аккаунта', async ({ page }) => {
  await page.goto('/register')
  const username = `e2e_account_${Date.now().toString(36)}`
  await page.locator('input[autocomplete="username"]').fill(username)
  await page.locator('input[type="password"]').fill('password1')
  await page.getByRole('button', { name: 'Зарегистрироваться' }).click()
  await page.waitForURL('**/profile')

  await expect(page.getByRole('button', { name: 'Удалить аккаунт' })).toHaveCount(0)
  await expect(page.locator('body')).not.toContainText('Удалить аккаунт')

  await page.goto('/settings')
  await page.waitForLoadState('networkidle')
  await expect(page.getByRole('button', { name: 'Удалить аккаунт' })).toHaveCount(0)
  await expect(page.locator('body')).not.toContainText('Удалить аккаунт')
})