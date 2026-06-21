# Data Model

Version: 0.1  
Date: 2026-06-21

---

## ER 关系总览

```
organizations
├── users
│   └── refresh_tokens
├── accounts
├── contacts
├── invoices
│   ├── invoice_lines
│   └── payments (type=ar)
├── bills
│   ├── bill_lines
│   └── payments (type=ap)
└── journal_entries
    └── journal_lines
```

---

## 表详情

### organizations — 组织（租户）

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| name | TEXT NOT NULL | 公司名称 |
| currency | TEXT DEFAULT 'USD' | 默认币种 |
| created_at | TIMESTAMPTZ | |

**说明：** 每个租户一行。所有业务表都通过 `org_id` 隔离。

---

### users — 用户

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| org_id | UUID FK → organizations | |
| email | TEXT UNIQUE NOT NULL | 登录邮箱，全局唯一 |
| password_hash | TEXT NOT NULL | bcrypt |
| name | TEXT NOT NULL | 显示名 |
| role | TEXT DEFAULT 'admin' | admin / accountant / viewer |
| is_active | BOOLEAN DEFAULT true | 停用不可登录 |
| created_at | TIMESTAMPTZ | |

**约束：** 一个邮箱只属于一个组织。

---

### refresh_tokens — 刷新令牌

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| user_id | UUID FK → users ON DELETE CASCADE | |
| token_hash | TEXT UNIQUE NOT NULL | SHA-256(rawToken) |
| expires_at | TIMESTAMPTZ NOT NULL | 30 天 |
| created_at | TIMESTAMPTZ | |

---

### accounts — 科目表（COA）

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| org_id | UUID FK → organizations | |
| code | TEXT NOT NULL | 科目编码，org 内唯一 |
| name | TEXT NOT NULL | 科目名称 |
| type | TEXT NOT NULL | asset / liability / equity / income / expense |
| is_active | BOOLEAN DEFAULT true | 停用科目不可用于新凭证 |
| created_at | TIMESTAMPTZ | |

**约束：** UNIQUE(org_id, code)

**余额方向：**
- asset / expense → 借方正常（debit normal）
- liability / equity / income → 贷方正常（credit normal）

**待补充字段（v0.2）：**
- `parent_code TEXT` — 科目层级（一级/二级）
- `is_bank_account BOOLEAN` — 标记为银行账户

---

### contacts — 联系人

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| org_id | UUID FK → organizations | |
| name | TEXT NOT NULL | |
| email | TEXT | |
| phone | TEXT | |
| type | TEXT NOT NULL | customer / supplier / both |
| is_active | BOOLEAN DEFAULT true | |
| created_at | TIMESTAMPTZ | |

**待补充字段（v0.2）：**
- `tax_id TEXT` — 税号
- `billing_address TEXT` — 账单地址
- `payment_terms_days INT` — 默认账期（天）

---

### invoices — 销售发票（AR）

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| org_id | UUID FK → organizations | |
| contact_id | UUID FK → contacts | 须为 customer 或 both |
| number | TEXT NOT NULL | 发票号，org 内唯一 |
| issue_date | DATE NOT NULL | 开票日期 |
| due_date | DATE | 到期日 |
| status | TEXT DEFAULT 'draft' | draft / approved / paid / voided |
| subtotal | NUMERIC(15,2) | 税前小计 |
| tax_amount | NUMERIC(15,2) | 税额合计 |
| total | NUMERIC(15,2) | 含税合计 |
| amount_due | NUMERIC(15,2) | 未付余额，初始 = total |
| currency | TEXT DEFAULT 'USD' | |
| notes | TEXT | 备注 |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

**状态流转：**
```
draft → approved → paid（amount_due 归零）
      → voided（仅 draft/approved 可作废）
```

**约束：** UNIQUE(org_id, number)

**记账规则（approved 时生成 GL）：**
```
DR  1100 AR            total
  CR  4000 收入账户    subtotal  （按行项目科目分摊）
  CR  2100 VAT Payable tax_amount
```

---

### invoice_lines — 发票行项目

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| invoice_id | UUID FK → invoices ON DELETE CASCADE | |
| description | TEXT NOT NULL | 品名/描述 |
| quantity | NUMERIC(15,4) DEFAULT 1 | 数量 |
| unit_price | NUMERIC(15,4) NOT NULL | 单价 |
| tax_rate | NUMERIC(5,4) DEFAULT 0 | 税率，如 0.1 = 10% |
| amount | NUMERIC(15,2) NOT NULL | quantity × unit_price（不含税） |
| account_code | TEXT NOT NULL | 对应收入科目 |
| line_no | INT NOT NULL | 行号，从 1 开始 |

**待补充字段（v0.2）：**
- `item_id UUID FK → items` — 关联产品/服务

---

### bills — 采购账单（AP）

与 invoices 结构基本对称，差异：

| 列 | 差异说明 |
|----|---------|
| number | 允许为空（供应商发票号可能后填） |
| reference | 供应商参考号 |
| contact_id | 须为 supplier 或 both |

**记账规则（approved 时生成 GL）：**
```
DR  5000 费用账户     subtotal  （按行项目科目分摊）
DR  2100 VAT Input   tax_amount
  CR  2000 AP         total
```

---

### bill_lines — 账单行项目

与 invoice_lines 结构相同，`account_code` 对应费用科目。

---

### payments — 收付款

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| org_id | UUID FK → organizations | |
| type | TEXT NOT NULL | ar（收款）/ ap（付款） |
| reference_id | UUID NOT NULL | → invoices.id 或 bills.id |
| date | DATE NOT NULL | 付款日期 |
| amount | NUMERIC(15,2) NOT NULL | 本次付款金额 |
| account_code | TEXT NOT NULL | 现金/银行科目，须为 asset 类 |
| reference | TEXT | 参考号/备注 |
| created_at | TIMESTAMPTZ | |

**记账规则：**
- AR 收款：`DR {account_code} / CR 1100 AR`
- AP 付款：`DR 2000 AP / CR {account_code}`

**问题：** `reference_id` 无 FK 约束（指向两张不同的表），需在应用层校验。

---

### journal_entries — 凭证头

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| org_id | UUID FK → organizations | |
| date | DATE NOT NULL | 记账日期 |
| reference | TEXT | 参考号 |
| description | TEXT NOT NULL | 摘要 |
| status | TEXT DEFAULT 'posted' | posted / voided |
| source_type | TEXT | invoice / bill / payment / manual |
| source_id | UUID | 来源单据 ID |
| created_at | TIMESTAMPTZ | |

**约束：** 只有 `source_type='manual'` 的凭证可手动作废。

---

### journal_lines — 凭证行

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| entry_id | UUID FK → journal_entries ON DELETE CASCADE | |
| account_code | TEXT NOT NULL | 科目编码 |
| description | TEXT | 行摘要 |
| debit | NUMERIC(15,2) DEFAULT 0 | 借方金额 |
| credit | NUMERIC(15,2) DEFAULT 0 | 贷方金额 |
| line_no | INT NOT NULL | |

**约束（应用层）：** SUM(debit) = SUM(credit)，每行不能借贷同时非零。

---

## 已知问题 & 待办

| # | 问题 | 优先级 |
|---|------|--------|
| 1 | `payments.reference_id` 无 FK，两张表共用一列 | 中 |
| 2 | `invoice_lines / bill_lines` 的 `account_code` 是文本，不是 FK | 中 |
| 3 | `journal_lines.account_code` 同上 | 中 |
| 4 | 缺 `items` 表（产品/服务目录） | 高 |
| 5 | 缺 `tax_rates` 表 | 高 |
| 6 | 缺 `credit_notes` / `credit_note_lines` 表 | 高 |
| 7 | 缺 `audit_logs` 表 | 中 |
| 8 | `accounts` 缺 `parent_code`、`is_bank_account` | 低 |
| 9 | `contacts` 缺 `tax_id`、`billing_address`、`payment_terms_days` | 低 |
| 10 | 所有表缺 `updated_by UUID` 审计字段 | 低 |
