import { test, expect } from '@playwright/test';
import { LedgerPage } from './pages/ledger';

test.describe('Ledger entries', () => {
  let ledgerPage: LedgerPage;

  test.beforeEach(async ({ page }) => {
    ledgerPage = new LedgerPage(page);
    await ledgerPage.goto();
  });

  test('page loads with entries', async ({ page }) => {
    await expect(ledgerPage.table).toBeVisible();
    
    const rows = ledgerPage.table.locator('tbody tr');
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
  });

  test('filter by account', async ({ page }) => {
    const initialCount = await ledgerPage.getRowCount();
    
    await ledgerPage.filterByAccount('111');
    
    const filteredCount = await ledgerPage.getRowCount();
    expect(filteredCount).toBeLessThanOrEqual(initialCount);
    
    const rows = ledgerPage.table.locator('tbody tr');
    for (let i = 0; i < filteredCount; i++) {
      await expect(rows.nth(i)).toContainText('111');
    }
  });

  test('create new entry', async ({ page }) => {
    const initialCount = await ledgerPage.getRowCount();
    
    await ledgerPage.newEntryButton.click();
    
    await page.fill('input[name="date"]', '2026-08-16');
    await page.fill('textarea[name="description"]', 'Test bút toán');
    
    await page.click('button:has-text("Thêm dòng")');
    
    await page.fill('input[name="account_code"]:last-of-type', '1111');
    await page.fill('input[name="debit"]:last-of-type', '1000000');
    
    await page.click('button:has-text("Thêm dòng")');
    
    await page.fill('input[name="account_code"]:last-of-type', '5111');
    await page.fill('input[name="credit"]:last-of-type', '1000000');
    
    await page.click('button:has-text("Lưu")');
    
    await expect(page.locator('text=Test bút toán')).toBeVisible();
  });
});
