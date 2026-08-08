# Processes, Workflows, Data Flows — Cashier (Thủ quỹ) Module

## 1. Core process map

```
[Preparer]        [Approver]          [Cashier]              [Cash Accountant]
   │                    │                   │                      │
 đề nghị thu/chi        │                   │                      │
   │───────► lập phiếu  │                   │                      │
   │            │       │                   │                      │
   │            └───────► duyệt             │                      │
   │                    │ ────────────────► │ Ghi sổ (post)        │
   │                    │                   │   └──────► S07-DN    │
   │                    │                   │   └──────► ledger seam│ (TK111)
   │                    │                   │   khóa sổ ngày /     │
   │                    │                   │   kiểm kê            │
   │                    │                   │   └──────► biên bản   │
   │                    │                   │                      │
   │                    │ ◄────────── chênh lệch báo cáo ──────────┤
   │                    │                                          │
   │                    │              đối chiếu cuối tháng ◄──────┤
   │                    │              (3 bên ký)                  │
```

## 2. Process P1: Thu tiền mặt (cash receipt)

1. Người nộp (customer/employee) presents payment + related docs.
2. Accountant creates Phiếu thu (01-TT): fund, date, counterparty, amount, diễn giải, lines Nợ 1111 / Có (131, 141, 511…).
3. Approver approves.
4. Cashier receives cash, counts, verifies amount = voucher, signs as receiver, **Ghi sổ**.
5. Cash book (S07-DN) balance increases; ledger posts; voucher printed + signed by cashier & người nộp.
6. If customer settles AR → linked to sales/AR module ref.

Rules: no receipt without voucher (R1); numbering sequential (BR1); approve ≠ prepare (R6).

## 3. Process P2: Chi tiền mặt (cash payment)

1. Request (employee/supplier claim / approved invoice) → Accountant creates Phiếu chi (02-TT).
2. Chief/Director approves (limit-based).
3. Cashier verifies supporting docs, counts cash out, receiver signs, **Ghi sổ**.
4. Balance decreases; ledger posts; voucher signed by cashier + receiver.
5. If settles AP / payroll / tax → linked to those modules.

Rules: cannot pay more than available cash (BR2); no direct erasure (BR3).

## 4. Process P3: Khóa sổ ngày & kiểm kê quỹ (daily close + count)

1. End of day, Cashier lists today's S07-DN entries.
2. Physical count of safe; enter counted amount.
3. If count == book balance → day closed, biên bản equal.
4. If difference → biên bản chênh lệch, notify Chief (R7), open CashCount until resolved.
5. Accountant verifies the daily entries landed in ledger (parallel S07a-DN, sign column G).

## 5. Process P4: Đối chiếu cuối tháng (monthly reconciliation)

1. Cashier exports S07-DN month totals (tồn cuối tháng).
2. Accountant exports S07a-DN month totals.
3. Both must match; system generates biên bản đối chiếu quỹ.
4. Cashier + Cash Accountant + Chief sign (electronic); month marked closed/reconciled.
5. Any diff → investigate, corrective/reversal vouchers (Điều 30), re-run.

## 6. Process P5: Sửa sai (correction — Điều 30)

1. Detect error in posted voucher.
2. Create reversal voucher (opposite sign, same amount, same date reference) — no direct edit.
3. Post reversal, link pair, audit trail records both.
4. If error affects previous month already closed → post correction in current month, disclose in notes.

## 7. Data flows

### DF1: Posting flow (write path)

```
Client ──POST /cash/vouchers/:id/post──► Handler
  ├─ resolve actor (X-User-Id) ──────────► Service.PostVoucher
  │                                         ├─ role:cashier? Casbin+service check
  │                                         ├─ state==approved?
  │                                         ├─ fund open? period open?
  │                                         ├─ payment: balance ≥ amount?
  │                                         ├─ tx: append S07-DN (running balance)
  │                                         ├─ tx: update voucher → posted
  │                                         ├─ ledger seam: write TK111 entry
  │                                         └─ audit: transition log
  └──201 + voucher JSON ◄───────────────────┘
```

### DF2: Read flow (cash book)

```
Client ──GET /cash/books?fund=&from=&to──► Handler ──► Service.GetCashBook
  ├─ authorize (role in {cashier, cash_accountant, chief})
  └──200 rows[] ──► S07-DN template render / JSON
```

### DF3: Sequence numbering (concurrency-safe)

```
Service.NextRefNo(fundID, period, type)
  ├─ tx on cash_sequences
  │    INSERT ... ON CONFLICT(fund_id,period,type) DO UPDATE SET seq=seq+1 RETURNING seq
  └─ refNo = prefix + period + padded(seq)   // e.g. PT/2026-08/000123
```

### DF4: Reconciliation flow

```
S07-DN (cashier side) ─┐
                        ├─► system compare ──► equal? ──► biên bản → sign (3 bên) ──► month closed
S07a-DN (accountant) ──┘        └── diff? ──► investigation → reversal/corrective → re-run
```

## 8. State transition matrix

| From | Action | To | Guards |
|---|---|---|---|
| draft | update | draft | actor=cash accountant |
| draft | approve | approved | approver role; actor ≠ CreatedBy |
| draft | void | voided | reason required |
| approved | post | posted | role:cashier; fund cashier; balance OK; period open |
| approved | void | voided | reason required |
| posted | reconcile | reconciled | month reconciliation; all counts resolved |
| posted | void (Điều 30) | voided | reversal voucher posted; chief approval |
| reconciled | — | (immutable) | none |

## 9. Workflow automation hooks (future)

- Post-approval notify cashier (in-app).
- Daily-close reminder at configured time.
- Chênh lệch → auto-notify chief + open action item in `task` module.
- Month-end reconciliation checklist gating `reporting` outputs.
