# Ledger Module — Processes, Workflows & User Journeys

## 1. Core business process: "Ngày giao dịch → Sổ sách"

```
[Source module event]           [Ledger module]
 cash voucher, bank, invoice      │
        │                         │
        ▼                         ▼
 Source commits its state  ──▶  PostFromSource (tx)
        │                         │ R1–R6 + R10
        │                     POSTED JournalEntry
        │                         │
        ▼                         ▼
 Voucher flags            Sổ Nhật ký chung / Sổ Cái / Sổ chi tiết
 LedgerPosted=true                │
                                  ▼
                          Bảng cân đối số phát sinh (S06)
                                  │
                        Kết chuyển cuối kỳ (511/632/641/642→911→421)
                                  │
                        Khoá sổ kỳ → kỳ tiếp theo mở với số dư đầu kỳ
```

**Guiding rules (from Luật Kế toán 2015 / TT 99/2025):** ghi sổ kịp thời; số liệu
phải khớp chứng từ gốc; không tẩy xóa; sửa sai bằng bút toán điều chỉnh/đảo bút
toán; khoá sổ khi kết thúc kỳ.

## 2. Process map

| Process | Owner | Frequency | Entry → Exit | Docs |
|---|---|---|---|---|
| P1 Automatic posting from source modules | KTTG + SRC | every transaction | event → POSTED entry | UC-L1 |
| P2 Manual phiếu kế toán | KTTG | daily | draft → posted | UC-L2 |
| P3 Books & reports (Sổ Cái, NJC, BCĐPS, chi tiết) | KTTG/KTT | daily → monthly | query → print | UC-L3..L5 |
| P4 Month-end closing (kết chuyển + khoá sổ) | KTT | monthly (EOM) | templates → closing → CLOSED | UC-L6 |
| P5 Opening balances / go-live | KTT | once + yearly | balances → open period | UC-L7 |
| P6 Corrections (đảo bút toán) | KTT | ad hoc | posted → reversed | UC-L8 |
| P7 Posting error queue | KTTG | ad hoc | error → repost/manual fix | UC-L9 |

## 3. Month-end close (P4) — step-by-step

**Gate:** all source modules confirm no pending events for period M; zero DRAFT
entries in M (else E8). **Inputs:** enabled templates, unposted cash/bank/voucher
queue, open period M.

1. KTT runs `POST /periods/M/close/run`.
2. System executes templates in order (kết chuyển doanh thu → chi phí → lợi
   nhuận); each produces a POSTED entry (VoucherNo `KC-…`); records ClosingRecord.
3. System runs integrity checks: Σ debit = Σ credit globally per period; BCĐPS
   column totals balance; Sổ Cái opening = prior period closing; no float math.
4. Preview: KTT reviews ClosingRecord + regenerated books.
5. KTT confirms `POST /periods/M/close` → period CLOSED.
6. System rolls opening balances to M+1 (Số dư cuối kỳ M → Số dư đầu kỳ M+1).
7. Notify: KTTG + GD see "Kỳ M đã khoá sổ" on dashboard/web.

**Exception branch:** any check fails → close aborts, list of issues returned,
nothing half-closed (single tx per step).

## 4. User journeys

### Journey A — Chief accountant first-run (go-live)
1. Import/define chart of accounts (VAS list) → `POST /accounts` (batch).
2. Post opening balances for the first period (UC-L7).
3. Set voucher sequence start, enable kết chuyển templates.
4. Confirm period open; cashier pilot continues → cash postings flow in (UC-L1).
5. Check Sổ Cái 111 equals cash fund balance on the cash dashboard (reconciliation).

### Journey B — Daily KTTG
1. Review posting error queue → resolve (UC-L9).
2. Manual adjustments from paper chứng từ → phiếu kế toán (UC-L2).
3. Spot-check Sổ Nhật ký chung vs cash vouchers of the day.

### Journey C — Monthly KTT
1. Run EOM close (P4). Review ClosingRecord. Lock period.
2. Print Sổ Cái / BCĐPS for tax filing package (TT 99/2025 form).

### Journey D — GD read-only
1. Open dashboard → total assets, cash, revenue run-rate.
2. Drill into Sổ Cái of any account; export; no mutations possible (403).

## 5. Business rules catalogue (summary, full detail in 02-spec §3)

- BR-1 Balanced entry always (R1). BR-2 One-side-per-line (R2).
- BR-3 Postable account only (R3). BR-4 Open period only (R4).
- BR-5 Source idempotency (R5). BR-6 Append-only after POSTED (R6/R7).
- BR-7 Close/reopen/reverse by KTT only (R8, authz). BR-8 Exact reversal (R9).
- BR-9 Atomic voucher sequence (R10). BR-10 Integer VND money only (§7 spec).
- BR-11 Books reconcile to source modules; kết chuyển rules per TT 99 (511→911,
  chi phí bán hàng/quản lý 632/641/642→911, 911→4211).

## 6. Service-level expectations

- Posting path: p95 < 100 ms/entry (SQLite, local), matching cash module
  benchmark style (cash: 3.7 ms/op post, 86 ms/op 12-month book).
- Books render: Sổ Cái (12 months, 50k entries) < 2 s.
- Any 50k-entry period close completes < 10 s (template runs are batched).
