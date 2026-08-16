import { test, expect } from '@playwright/test';
import { MasterDataPage } from './pages/masterdata';

test.describe('Master Data CRUD', () => {
  let masterdataPage: MasterDataPage;

  test.beforeEach(async ({ page }) => {
    masterdataPage = new MasterDataPage(page);
    await masterdataPage.goto();
  });

  test('create a new unit', async ({ page }) => {
    await masterdataPage.selectKind('unit');
    
    const initialCount = await masterdataPage.getRowCount();
    
    await masterdataPage.createButton.click();
    
    await page.fill('input[name="code"]', 'TEST-UNIT');
    await page.fill('input[name="name"]', 'Đơn vị test');
    
    await page.click('button:has-text("Lưu")');
    
    await expect(page.locator('text=Đơn vị test')).toBeVisible();
    expect(await masterdataPage.getRowCount()).toBeGreaterThan(initialCount);
  });

  test('search filters results', async ({ page }) => {
    await masterdataPage.selectKind('account');
    
    await masterdataPage.search('111');
    
    const rows = page.locator('table tbody tr');
    const count = await rows.count();
    
    for (let i = 0; i < count; i++) {
      await expect(rows.nth(i)).toContainText('111');
    }
  });

  test('deactivate and reactivate', async ({ page }) => {
    await masterdataPage.selectKind('unit');
    
    const firstRow = page.locator('table tbody tr').first();
    const code = await firstRow.locator('td').first().textContent();
    
    await firstRow.locator('button:has-text("Ngừng")').click();
    
    await page.fill('input[name="reason"]', 'Test deactivate');
    await page.click('button:has-text("Xác nhận")');
    
    await expect(firstRow.locator('text=Ngừng hoạt động')).toBeVisible();
    
    await firstRow.locator('button:has-text("Kích hoạt")').click();
    
    await expect(firstRow.locator('text=Đang hoạt động')).toBeVisible();
  });
});
