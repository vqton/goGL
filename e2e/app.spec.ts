import { test, expect } from '@playwright/test';

test.describe('Landing page', () => {
  test('loads successfully', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/goGL/);
  });

  test('shows navigation links', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('nav')).toBeVisible();
  });
});

test.describe('Master Data module', () => {
  test('page loads', async ({ page }) => {
    await page.goto('/web/master-data');
    await expect(page.locator('text=Danh mục cơ bản')).toBeVisible();
  });

  test('kind selector works', async ({ page }) => {
    await page.goto('/web/master-data');
    await page.click('button:has-text("Tài khoản")');
    await expect(page.locator('table')).toBeVisible();
  });
});

test.describe('Ledger module', () => {
  test('page loads', async ({ page }) => {
    await page.goto('/web/ledger');
    await expect(page.locator('text=Sổ cái')).toBeVisible();
  });
});
