import { type Page, type Locator } from '@playwright/test';

export class MasterDataPage {
  readonly page: Page;
  readonly kindSelector: Locator;
  readonly searchInput: Locator;
  readonly table: Locator;
  readonly createButton: Locator;
  readonly importButton: Locator;
  readonly exportButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.kindSelector = page.locator('[data-kind]');
    this.searchInput = page.locator('input[placeholder*="Tìm kiếm"]');
    this.table = page.locator('table');
    this.createButton = page.locator('button:has-text("Thêm mới")');
    this.importButton = page.locator('button:has-text("Nhập CSV")');
    this.exportButton = page.locator('button:has-text("Xuất CSV")');
  }

  async goto() {
    await this.page.goto('/web/master-data');
  }

  async selectKind(kind: string) {
    await this.page.click(`button[data-kind="${kind}"]`);
  }

  async search(query: string) {
    await this.searchInput.fill(query);
  }

  async getRowCount(): Promise<number> {
    return this.table.locator('tbody tr').count();
  }
}
