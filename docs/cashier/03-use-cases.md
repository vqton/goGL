# Use Cases — Cashier (Thủ quỹ) Module

Each use case: Preconditions → Trigger → Happy path → Alternative paths → Exception paths → Postconditions → Rules covered.

## UC-1: Create cash payment voucher (Lập phiếu chi)

**Actor:** Cash accountant. **Pre:** Fund exists and open; actor has `role:cash_accountant`.

**Happy path**
1. Accountant opens "Phiếu chi" form, selects fund (e.g. Q-VND), date, counterparty (supplier/employee/other), diễn giải.
2. Enters amount, selects related refs (invoice/contract), system suggests lines Nợ (e.g. 331) / Có (1111).
3. Saves. System validates double-entry sums, generates sequential `RefNo`, sets state `draft`.
4. System shows amount in words + signature placeholders. Voucher visible in "Đề nghị thu, chi" list.

**Alternative paths**
- A1: Amount entered in FX → system stores FXAmount+rate, converts to fund currency for booking (BR6).
- A2: Multiple analytic objects → multiple lines, sum still equal to AmountMinor.
- A3: Voucher number already taken (race) → retry with next sequence (BR1, ErrSequenceConflict → retry).

**Exception paths**
- E1: Lines don't balance → reject, error "bút toán không cân đối".
- E2: No counterparty → reject (R1).
- E3: Period closed → reject ErrPeriodClosed.

**Post:** Draft voucher with RefNo persisted; audit entry.

## UC-2: Approve payment voucher (Phê duyệt phiếu chi)

**Actor:** Chief accountant / Director. **Pre:** Voucher `draft`; actor has approval role; actor ≠ CreatedBy.

**Happy path**
1. Lists pending vouchers, opens voucher, reviews lines, clicks "Duyệt".
2. System verifies role + actor≠preparer (R6), sets `approved`, records ApprovedBy/ApprovedAt.
3. Voucher appears in Cashier's "Đề nghị thu, chi" queue.

**Alternative paths**
- A1: Approver rejects → voucher stays `draft` with rejection note (audit only).
- A2: Delegation → approver's delegate approved (configurable later).

**Exception paths**
- E1: Preparer tries to approve own voucher → 403 ErrUnauthorizedActor.
- E2: Voucher not in `draft` → ErrInvalidState.
- E3: Over approval limit → requires director (configurable limit; v1 hard rule per role).

## UC-3: Cashier posts voucher into cash book (Ghi sổ)

**Actor:** Cashier (Thủ quỹ). **Pre:** Voucher `approved`; actor has `role:cashier` and is fund's cashier; fund open.

**Happy path**
1. Cashier opens "Đề nghị thu, chi", selects voucher(s), clicks **Ghi sổ** (post date = voucher date or chosen date).
2. System checks fund balance for payments (BR2), appends S07-DN row with running `Tồn = đầu + thu − chi`.
3. State → `posted`; PostedBy/PostedAt set; ledger seam writes TK111 entry.
4. Cash book shows updated balance; voucher can be printed.

**Alternative paths**
- A1: Post by chosen date (MISA parity) → entry logged with both dates.
- A2: Batch Ghi sổ (multiple vouchers) → processed in one transaction, all-or-nothing.
- A3: Posting after daily close for same date → blocked (BR8) unless manager reopens.

**Exception paths**
- E1: Payment exceeds balance → ErrInsufficientBalance ("tồn quỹ không đủ"), voucher stays approved.
- E2: Cashier = preparer or approver → ErrUnauthorizedActor (R6).
- E3: Concurrent double-post → second attempt returns conflict, no double entry (BR10).
- E4: Period closed → ErrPeriodClosed.

**Post:** Voucher `posted`; S07-DN row appended; balance updated; audit.

## UC-4: Daily cash close & cash count (Khóa sổ ngày / Kiểm kê quỹ)

**Actor:** Cashier. **Pre:** All vouchers for the date posted (or exception-flagged).

**Happy path**
1. Cashier clicks "Khóa sổ ngày", enters physical counted amount.
2. System compares book balance vs counted.
3. Equal → day closed, `reconciled` marks set; biên bản with equal amounts.
4. Cashier prints daily S07-DN.

**Alternative paths**
- A1: Difference ≠ 0 → creates open CashCount, logs to chief accountant (R7), day flagged "chênh lệch", reconciliation required before month close.
- A2: Missing vouchers → cashier posts them first, then closes.

**Exception paths**
- E1: Counted amount negative/invalid → validation error.
- E2: Attempt to close already-closed date → conflict.

## UC-5: Monthly reconciliation (Đối chiếu quỹ cuối tháng)

**Actor:** Cashier + Cash accountant + Chief accountant. **Pre:** Month exists, day closures done.

**Happy path**
1. Cashier opens "Đối chiếu quỹ" for month; system shows S07-DN total (cashier side).
2. Cash accountant enters S07a-DN total (accountant side); system compares.
3. Equal → biên bản đối chiếu generated; cashier + accountant + chief sign electronically; state `reconciled`.
4. Book closed for month; all vouchers → `reconciled`.

**Alternative paths**
- A1: Difference → both parties review, correct via adjustment vouchers (never direct edit, BR3), re-run.
- A2: FX fund → reconciliation in fund currency with rate logs.

**Exception paths**
- E1: Unreconciled daily count open → month reconciliation blocked (must resolve chênh lệch first).

## UC-6: Void a voucher / correct posted entry (Điều 30)

**Actor:** Cash accountant. **Pre:** Voucher `draft`|`approved`|`posted`; actor `role:cash_accountant`; chief approval required for posted.

**Happy path (draft/approved)**
1. Select voucher → "Hủy" with reason.
2. State → `voided`; number retained (BR1); no balance impact.

**Alternative path (posted, Điều 30)**
1. Create reversal voucher (opposite type, same amount, references original).
2. Post reversal → both voucher and reversal `voided`/`reconciled`; balance restored; audit trail links pair.

**Exception paths**
- E1: Voiding `reconciled` entry without chief approval → denied.
- E2: Reversal amount ≠ original → rejected.

## UC-7: Print statutory forms

**Actor:** Cashier / Cash accountant. **Pre:** Data present.

**Happy path:** Select voucher/period → print Phiếu thu (01-TT), Phiếu chi (02-TT), S07-DN, S07a-DN, biên bản. HTML template → PDF/print (see 05-ui).

**Exception:** No data → empty-form notice, no crash.

## UC-8: Cashier views Sổ quỹ tiền mặt

**Actor:** Cashier / Cash accountant / Chief. **Pre:** none.

**Happy path:** Choose fund + period → S07-DN with columns A–3, running balance, totals. Filter by type/date. Export CSV/print.
**Exception:** Fund missing → 404.

---

## User journey (end-to-end month)

Day 1: Accountant creates phiếu chi (UC-1) → Chief approves (UC-2) → Cashier Ghi sổ (UC-3) → daily close + count (UC-4) → prints S07-DN.
Day 15: surprise cash count (UC-4 A1 path) → resolves chênh lệch.
Day 28: error found in posted entry → Điều 30 reversal (UC-6 alt) → re-post.
Month end: cashier vs accountant reconciliation (UC-5) → biên bản signed → month closed.
Any time: print forms (UC-7), view cash book (UC-8).
