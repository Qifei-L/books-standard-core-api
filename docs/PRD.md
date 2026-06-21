# Books Standard Core — Product Requirements Document

Version: 0.1  
Date: 2026-06-21  
Status: Draft

---

## 1. 背景与目标

Books Standard Core 是一个面向小微企业的 SaaS 记账系统，对标 Xero，提供双录（AR/AP）+ 总账 + 财务报表的完整闭环。

**核心痛点：** Xero 对小微企业定价过高；本系统以自用为起点，逐步开放多租户。

**成功标准（v1）：**
- 一家公司可独立完成：开票 → 收款 → 查损益表 的全流程
- 账务数据双录正确，借贷平衡

---

## 2. 用户角色

| 角色 | 描述 |
|------|------|
| **Admin** | 公司管理员，全部权限 |
| **Accountant** | 可录入凭证和发票，不可修改系统配置 |
| **Viewer** | 只读，查看报表 |

> v1 仅实现 Admin 角色，其余角色 v2 扩展。

---

## 3. 功能范围

### 3.1 必须有（v1 MVP）

#### 组织管理
- 创建/编辑组织（名称、币种、财年起始月）
- 多租户隔离：所有数据按 `org_id` 隔离

#### 用户与认证
- 邮箱 + 密码登录
- JWT Access Token（15 min）+ httpOnly Refresh Cookie（30 天）
- 同一邮箱只属于一个组织

#### 科目表（Chart of Accounts）
- 五类科目：`asset`（资产）、`liability`（负债）、`equity`（权益）、`income`（收入）、`expense`（费用）
- 新建 / 编辑 / 停用科目
- 自带默认 COA（按行业模板）
- 科目编码在组织内唯一

#### 联系人（Contacts）
- 类型：`customer`（客户）、`supplier`（供应商）、`both`
- 字段：名称、邮箱、电话、地址、税号
- 停用联系人不可用于新单据

#### 产品与服务（Items）
- 字段：编码、名称、默认销售科目、默认采购科目、默认单价、默认税率
- 创建发票行时可选 Item 自动填充

#### 税率（Tax Rates）
- 字段：名称、税率（%）、类型（`sales` / `purchase`）
- 创建时关联 VAT 科目

#### 销售发票（Invoices / AR）
- 状态流转：`draft` → `approved` → `paid` / `void`
- 行项目：描述、数量、单价、税率、科目、金额
- 自动计算小计 / 税额 / 合计
- 发票编号自动生成（可自定义前缀）
- 记账（approved）时自动生成 GL 分录：DR AR / CR 收入+VAT
- 支持部分付款，`amount_due` 实时更新
- 支持作废（void），已付款不可作废

#### 采购账单（Bills / AP）
- 与发票对称：`draft` → `approved` → `paid` / `void`
- 记账时：DR 费用+VAT / CR AP
- 支持供应商参考号

#### 收付款（Payments）
- AR 收款：DR 银行账户 / CR AR
- AP 付款：DR AP / CR 银行账户
- 付款来源科目须为 `asset` 类（现金/银行）
- 支持部分付款

#### 信用票据（Credit Notes）
- 销售信用票据：冲减 AR 发票
- 采购信用票据：冲减 AP 账单
- 状态：`draft` → `approved` → `applied` / `void`

#### 手工凭证（Manual Journals）
- 借贷必须平衡才能保存
- 状态：`posted` / `voided`（已过账才可作废）
- 记录来源类型（invoice / bill / payment / manual）

#### 银行账户（Bank Accounts）
- 在科目表中标记为银行账户
- 支持手动录入交易流水

#### 财务报表
| 报表 | 说明 |
|------|------|
| 试算平衡表 | 指定日期范围，借贷汇总 |
| 损益表（P&L） | 收入 - 费用 = 净利润 |
| 资产负债表 | 资产 = 负债 + 权益（含留存收益） |
| 应收账款账龄 | 按联系人分组，current/30/60/90+ 天 |
| 应付账款账龄 | 同上 |
| 科目明细账 | 单科目全期交易明细 |

---

### 3.2 计划有（v2）

- 多币种 + 汇率（表结构已预留 `currency` 字段）
- 询价单 / 销售订单 → 转发票
- 采购订单 → 转账单
- 用户角色管理（Accountant / Viewer）
- 银行对账（Bank Reconciliation）
- 固定资产模块
- 导出 PDF / Excel
- 邮件发送发票
- OAuth 登录（Google / Microsoft）

### 3.3 明确不做（v1）

- 薪资（Payroll）
- 库存管理
- 集团合并报表
- 多公司间内部交易抵消

---

## 4. 非功能需求

| 项目 | 要求 |
|------|------|
| 安全 | JWT + httpOnly cookie；所有业务接口须认证；org_id 隔离不可跨租户查询 |
| 性能 | 列表接口 < 200ms（p95，单租户数据量 < 10 万条凭证） |
| 可靠性 | 金额全部用 `NUMERIC(15,2)` 存储，禁用 float 做最终金额 |
| 审计 | 关键操作（approve / void / payment）写审计日志 |
| 可测试性 | 核心业务逻辑有集成测试，跑真实 DB |
| 部署 | Docker Compose 一键起；GitHub Actions CI |

---

## 5. 数据模型概览

```
organizations
  └── users
  └── accounts (COA)
  └── contacts
  └── tax_rates
  └── items
  └── invoices → invoice_lines
  │     └── payments (type=ar)
  │     └── credit_note_applications
  └── bills → bill_lines
  │     └── payments (type=ap)
  │     └── credit_note_applications
  └── credit_notes → credit_note_lines
  └── journal_entries → journal_lines
  └── audit_logs
```

---

## 6. API 设计原则

- RESTful，JSON
- 所有响应包裹在 `{ "data": ... }`
- 错误统一格式：`{ "error": { "code": "...", "message": "..." } }`
- 认证：`Authorization: Bearer <accessToken>`
- 路由：`/api/v1/...`（加版本号）
- 分页：cursor-based（`?cursor=&limit=50`）

---

## 7. 里程碑

| 版本 | 内容 | 目标日期 |
|------|------|----------|
| **v0.1** | 当前代码：Auth + COA + Contacts + AR/AP + 手工凭证 + 报表基础 | 2026-06-21 ✅ |
| **v0.2** | Items + Tax Rates + 发票行关联 Item/Tax + 信用票据 + 审计日志 | TBD |
| **v0.3** | 银行账户 + 银行流水录入 + OpenAPI spec | TBD |
| **v0.4** | 集成测试套件 + CI/CD + 修复 float→decimal | TBD |
| **v1.0** | 全流程 UAT 通过 + 部署文档 | TBD |
