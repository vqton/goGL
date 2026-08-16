import { type Page, type Locator } from '@playwright/test';

export class LedgerPage {
  readonly page: Page;
  readonly periodSelector: Locator;
  readonly accountFilter: Locator;
  readonly table: Locator;
  readonly newEntryButton: Locator;
  readonly importButton: Locator;
  readonly exportButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.periodSelector = page.locator('select[name="period"]');
    this.accountFilter = page.locator('input[placeholder*="Tài khoản"]');
    this.table = page.locator('table');
    this.newEntryButton = page.locator('button:has-text("Bút toán mới")');
    this.importButton = page.locator('button:has-text("Nhập CSV")');
    this.exportButton = page.locator('button:has-text("Xuất CSV")');
  }

  async goto() {
    await this.page.goto('/web/ledger');
  }

  async filterByAccount(account: string) {
    await this.accountFilter.fill(account);
  }

  async getRowCount(): Promise<number> {
    return this.table.locator('tbody tr').count();
  }
}
