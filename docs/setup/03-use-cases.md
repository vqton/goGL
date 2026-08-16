# Setup Module (Khởi tạo hệ thống / Cấu hình doanh nghiệp) — Use Cases

> UC numbering: S = setup. Each UC lists happy path, alternative paths,
> exception paths. Roles per `01-brd §3`; rules R1–R13 per `02-spec §3`.

## UC-S1 — First-run initialization (wizard)

**Actor:** Kế toán trưởng (`ke_toan_truong`) or `admin` (first boot).
**Preconditions:** store empty (Status == EMPTY); authz OK.

**Happy path (S1-H)**
1. Actor opens Khởi tạo → wizard step 1 "Thông tin doanh nghiệp".
2. System shows profile form: Tên* (theo ĐKDN), Tên tiếng Anh, MST*,
   mã ĐVQHNS, Địa chỉ*, Người đại diện*, Loại hình, Ngành nghề, Đơn vị tiền tệ
   (VND, read-only v1), Ngày bắt đầu hạch toán, Chế độ kế toán
   (TT 99/2025 • default | TT 133/2016), Năm tài chính bắt đầu.
3. Actor saves. System validates R1–R5 (single company, 12-month FY, MST
   format, VND, regime) → Status PROFILED → audit.
4. Wizard auto-advances: `masterdata.SetRegime` (idempotent) → REGIME_SET.
5. `masterdata.SeedAccounts` (Phụ lục 2 TT 99/2025 or TT 133/2016) +
   Quy chế hạch toán audit note → ACCOUNTS_SEEDED.
6. `ledger.OpenPeriod("YYYY-01".."YYYY-12")` → PERIODS_OPEN.
7. Wizard shows "Nhập số dư đầu kỳ" (go to UC-S3) and waits; Status
   BALANCES_DRAFT.

**Alternative A1:** Company picks TT 133/2016 → SeedAccounts seeds the TT 133
variant; regime recorded (R5).
**Alternative A2:** Crash between steps → re-run `initialize` resumes from last
committed Status (idempotent, never double-seeds — R6).
**Alternative A3:** Năm tài chính khác calendar (fiscal year) → allowed, still
12 months (Điều 12); period ids shift accordingly.

**Exceptions**
- **E1:** Store already initialized → 409 "đã khởi tạo" (R1).
- **E2:** MST malformed (not 10-digit / not 13-with-dash) → 422 field error
  (VN + EN) (R3).
- **E3:** FiscalYearStart invalid (not a month start, or not 12 months) → 422 (R2).
- **E4:** Concurrent initialize → one wins; other 409 "đang khởi tạo".
- **E5:** Not authorized → 403 (fail closed).

## UC-S2 — Edit company profile

**Actor:** `ke_toan_truong`. **Precondition:** Status < BALANCES_LOCKED.

**Happy path (S2-H):** open profile → edit Address, LegalRepresentative,
Industry, NameEN → save → R1–R5 → audit → 200.
**Alternative A1:** Edit MST before any period posted → allowed (R3 validated).
**Alternative A2:** Edit after BALANCES_LOCKED → blocked (only read; change
requires reopen, UC-S5).

**Exceptions:** **E1** 409 state too far along; **E2** 422 invalid values;
**E3** 403; **E4** 404 no profile.

## UC-S3 — Enter opening balances

**Actor:** `ke_toan_tong_hop`. **Precondition:** Status == BALANCES_DRAFT.

**Happy path (S3-H)**
1. Actor opens Số dư đầu kỳ → list of TK (balance-sheet groups: TS, Nợ, VCSH).
2. Actor adds a balance: TK 1111 "Tiền mặt VND", Có 0, Nợ 500.000.000 đ.
3. System validates R7 (one side), R8 (draft), R10 (no object needed for 1111)
   → upsert `OB:1111:` → audit → 200; running Σ updates.

**Alternative A1:** Balance with đối tượng — TK 131 with khách hàng KH-0001,
Nợ 45.500.000 (R10: object required + ACTIVE in masterdata).
**Alternative A2:** Credit-side balance (e.g. TK 331 NCC): Có 20.000.000, Nợ 0.

**Exceptions**
- **E1:** Both Nợ and Có non-zero / negative → 422 (R7).
- **E2:** TK not found / not ACTIVE / not AllowPost → 422 (R10).
- **E3:** Object required but missing or INACTIVE (KH/NCC/vật tư/TSCĐ) → 422.
- **E4:** State not draft (already locked) → 409 (R8).
- **E5:** 403.

## UC-S4 — Check balance & lock opening balances

**Actor:** `ke_toan_truong` (lock). **Precondition:** BALANCES_DRAFT.

**Happy path (S4-H)**
1. Actor runs "Kiểm tra cân đối". System sums Σ Nợ, Σ Có (VND) → reports
   equal → "✅ Cân đối (Tổng Nợ = Tổng Có = X)".
2. Actor clicks "Khóa số dư" → re-check passes → Status BALANCES_LOCKED →
   audit (actor, at).
3. Optionally "Kích hoạt" → Status ACTIVE → ledger period 1 postable.

**Alternative A1:** Run check with mismatch → report diff + offending TK list
(422 with data) — the actor fixes entries (UC-S3) and re-checks.
**Alternative A2:** Lock with a known tiny discrepancy → blocked (R9) unless
chief override + reason (alternative to E1 below).

**Exceptions**
- **E1:** Not balanced → 422 with Σ Nợ, Σ Có, diff (R9); no lock.
- **E2:** State already locked → 409.
- **E3:** 403 (only ke_toan_truong/admin may lock).

## UC-S5 — Reopen locked balances

**Actor:** `ke_toan_truong`. **Precondition:** Status ∈ {BALANCES_LOCKED,
ACTIVE}.

**Happy path (S5-H):** postings reference an edited TK? → no → reopen with
reason "sửa số dư đầu kỳ TK 131" → Status BALANCES_DRAFT → audit → edit
(UC-S3) → re-check/lock.
**Alternative A1 (override):** postings reference the TK → 409 → chief submits
reason → forced reopen + audit (R12).

**Exceptions:** **E1** 409 có phát sinh (no override); **E2** 403 role too low;
**E3** 404.

## UC-S6 — Import opening balances from CSV

**Actor:** `ke_toan_tong_hop`. **Precondition:** Status == BALANCES_DRAFT;
template v1.

**Happy path (S6-H):** download template → fill → upload → dry-run → report
"N hợp lệ, M lỗi (dòng X: TK không tồn tại; dòng Y: chưa khai đối tượng)" →
fix → Commit → idempotent upsert by `OB:{account}:{object}` → success report →
audit.
**Alternative A1:** New-only vs Update-existing flag.

**Exceptions:** **E1** old template → 422 "Dùng template mới nhất"; **E2**
malformed file → 422; **E3** row errors isolated (never silent partial commit);
**E4** balance-check fails after import → 422 with diff; **E5** 403.

## UC-S7 — Activate company (go live)

**Actor:** `ke_toan_truong`. **Precondition:** BALANCES_LOCKED + balance-check
passed.

**Happy path (S7-H):** "Kích hoạt" → Status ACTIVE → ledger period 1 open for
posting → audit → wizard completes; dashboard shows company ACTIVE with regime
badge (TT 99/2025).
**Exceptions:** **E1** 409 not locked; **E2** 422 balance-check fails;
**E3** 403.

## UC-S8 — View setup status / profile (read-only)

**Actor:** any authenticated role (incl. `giam_doc`, `kiem_toan`).
**Happy path (S8-H):** Status page shows step checklist (EMPTY→ACTIVE), current
status, counts (số TK, số kỳ mở, Σ Nợ/Σ Có), profile summary, audit trail.
**Exceptions:** **E1** 403 anonymous; **E2** 404 no profile (pre-init).

## UC-S9 — Switch accounting regime (TT 99 ↔ TT 133)

**Actor:** `ke_toan_truong`. **Precondition:** Status < BALANCES_LOCKED (or
ACTIVE with FY-boundary guard, mirroring masterdata UC-M13).

**Happy path (S9-H):** Chế độ → select variant → dry-run COA diff → confirm →
`masterdata.SetRegime` + reseed diff + audit → 200; profile.regime updated.
**Exceptions:** **E1** 409 ledger has posted entries in the affected period
(regime change only at FY boundary — TT 99 Điều 31 / Luật Kế toán Điều 12);
**E2** 403; **E3** 422 state not eligible.

## NFR / cross-cutting use cases

- **UC-S10 Audit trail:** every setup mutation (profile, regime, balance
  create/edit/delete, lock/reopen/activate/import) appends to `audit_logs`
  with user + timestamp + reason. Golden test.
- **UC-S11 Idempotent resume:** crash at any step → re-run `initialize` never
  double-seeds COA/periods; status monotonic. Property test across all steps.
- **UC-S12 Balance invariant:** for any valid balance set, Σ Nợ == Σ Có;
  lock blocked otherwise. Property test.
- **UC-S13 Concurrency:** two parallel `initialize` → one succeeds, one 409.
- **UC-S14 Cross-module guard:** a posted cash voucher cannot be created while
  its period is un-opened/CLOSED (ledger R4) — setup `ACTIVE` is the gate that
  makes period 1 postable.
