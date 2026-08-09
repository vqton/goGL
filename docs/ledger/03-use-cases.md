# Ledger Module — Use Cases (Happy / Alternative / Exception)

> UC numbering continues the cash module convention. Actors: KTTG = Kế toán tổng
> hợp, KTT = Kế toán trưởng, GD = Giám đốc, SRC = source module (cash/bank/…).

## UC-L1 — Post a cash transaction to the GL automatically

- **Primary actor:** SRC (cash module), via `LedgerWriter` seam.
- **Trigger:** cash service posts a voucher (V1 payout, V3 receipt, V4 transfer).
- **Happy path:**
  1. Cash validates + commits its own transaction.
  2. Cash calls `LedgerWriter.Post(ctx, LedgerEntry{...})`.
  3. Ledger maps to `JournalEntry` (Source=CASH, SourceRef=VoucherID, balanced lines).
  4. R1–R6 pass; VoucherNo assigned (R10); entry inserted POSTED in one tx.
  5. Cash marks `LedgerPosted=true` on its voucher (idempotent guard).
- **Alternative A1 (retry):** network/DB failure after commit → cash retries →
  `PostFromSource` finds existing `(CASH, VoucherID)` → returns existing entry id,
  no duplicate (R5).
- **Exception E1 (unbalanced mapping):** R1/R2 fail → whole tx rolls back → error
  returned to cash → voucher marked as `LedgerError` and surfaced in a posting
  error queue; human (KTTG) resolves. **Never silently dropped.**
- **Exception E2 (period closed):** R4 fail → 409; cash keeps voucher posted but
  flags `LedgerPending` until period reopens or KTTG books to next period.

## UC-L2 — Create a manual journal entry (phiếu kế toán)

- **Primary actor:** KTTG.
- **Happy path:**
  1. KTTG opens "Phiếu kế toán" → picks date, description, adds lines (account,
     debit, credit, note).
  2. System live-validates: account exists/active/postable (R3), balanced (R1),
     one-side rule (R2), period open (R4).
  3. KTTG saves as DRAFT → POST `/entries`; or posts directly.
  4. Draft is listed for review; KTTG opens it → POST `/entries/:id/post`.
  5. Entry reaches POSTED, appears in all books.
- **Alternative A2 (quick-entry):** entry fully pre-filled by a template (bút toán
  định kỳ) → same validation → post.
- **Alternative A3 (approve-step config):** if workflow requires KTT approval,
  `/post` denied for KTTG, allowed for KTT (403 otherwise).
- **Exception E3 (validation):** 422 lists each failing rule (e.g. account 112
  not postable, Σ≠0, negative amount); no partial state.
- **Exception E4 (draft delete):** DELETE allowed on DRAFT only; POSTED → 409 (R7).

## UC-L3 — View Sổ Cái (ledger per account)

- **Primary actor:** KTTG, KTT, GD (read).
- **Happy path:** choose period range + account (or "all") → GET
  `/books/ledger` → renders Sổ Cái form: opening balance, per-entry TKHN
  (đối ứng), debit/credit, running balance, closing, signature block.
- **Alternative A5:** export to print layout (web template, per Phụ lục III mẫu).
- **Exception E5:** account with no activity in range → empty book with opening
  balance; period range invalid → 422.

## UC-L4 — Bảng cân đối số phát sinh (trial balance)

- **Primary actor:** KTT (prepares monthly basis for tax filing), GD.
- **Happy path:** select period → GET `/books/trial-balance` → per account:
  Số dư đầu kỳ (Nợ/Có), Số phát sinh kỳ (Nợ/Có), Số dư cuối kỳ (Nợ/Có).
  Tổng Nợ = Tổng Có on every column (invariant shown on the report).
- **Exception E6:** mismatch in totals → report renders with a red "CHÊNH LỆCH"
  marker and blocks export; only resolvable by fixing entries (append-only).

## UC-L5 — Sổ Nhật ký chung (general journal)

- **Primary actor:** KTTG, KTT.
- **Happy path:** select period → GET `/books/general-journal` → chronologically
  ordered entries with TKHN/đối ứng, Nợ, Có, running Σ; Σ column totals printed.
- **Exception E7:** out-of-order date entry (entered for an earlier date than last
  posted in period) → entry still accepted but flagged "NHẬP NGƯỢC NGÀY" — legal
  to record; report sorts by (VoucherDate, VoucherNo).

## UC-L6 — Close the accounting period (khoá sổ)

- **Primary actor:** KTT (with GD visibility).
- **Happy path:**
  1. KTT runs `close/run` → templates execute kết chuyển (511→911, chi phí→911,
     911→421) → closing entries POSTED with VoucherNo sequence.
  2. KTT reviews ClosingRecord + books, confirms.
  3. Period → CLOSED; opening balances roll to next period automatically.
- **Alternative A6 (partial close):** skip kết chuyển templates, close period
  with a written reason (books remain, no new postings).
- **Exception E8 (unposted drafts):** period has DRAFT entries → close blocked
  with list of drafts; KTTG must post, delete, or KTT moves them to next period.
- **Exception E9 (reopen):** KTT reopens with a reason → audited; previously
  POSTED entries unchanged; new postings allowed; must re-run closing afterwards.

## UC-L7 — Opening balances (số dư đầu kỳ)

- **Primary actor:** KTT (go-live migration).
- **Happy path:** KTT posts `OpeningBalance` for each account of the opening
  period → system verifies Σ Debit = Σ Credit across the set (implied R1) → books
  open with correct Số dư đầu kỳ.
- **Exception E10 (unbalanced opening):** rejected 422 with the gap; KTT corrects
  the source balances before re-submitting.

## UC-L8 — Reversal (đảo bút toán / bút toán điều chỉnh)

- **Primary actor:** KTT.
- **Happy path:** KTT opens a POSTED entry → POST `/entries/:id/reverse` →
  system creates negated entry (R9), marks original REVERSED, both remain in
  books; note links them.
- **Exception E11 (already reversed):** second reverse → 409.
- **Exception E12 (reversal in closed period):** reversal must land in an OPEN
  period; else 409 and KTT must reopen or book as điều chỉnh in current period.

## UC-L9 — Manual repost (error queue)

- **Primary actor:** KTTG.
- **Happy path:** posting error queue shows E1 failures → KTTG fixes cause
  (e.g. missing account) → `POST /postings/:source/:sourceRef` → success.
- **Exception E13 (cause unfixable in source):** KTTG manually books an adjusting
  entry (UC-L2) and marks the queue item resolved-with-note.

## Use-case coverage → module tasks

| UC | Phase (roadmap) |
|---|---|
| L1, L2, L3 | P2–P3 |
| L4, L5 | P3 |
| L7 | P2 (go-live) |
| L6, L8, L9 | P4 |
| L2 (templates) | P5 |
| Print polish | P6 |
