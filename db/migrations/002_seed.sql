-- +goose Up

-- system roles (org_id = NULL = available to all orgs)
INSERT INTO roles (id, org_id, name, description, permissions, is_system) VALUES
(
    '00000000-0000-0000-0000-000000000101',
    NULL,
    'admin',
    'Full access to all features',
    ARRAY['*'],
    true
),
(
    '00000000-0000-0000-0000-000000000102',
    NULL,
    'accountant',
    'Can manage invoices, bills, payments, contacts and view reports',
    ARRAY[
        'invoices.read','invoices.write','invoices.approve','invoices.void',
        'bills.read','bills.write','bills.approve','bills.void',
        'payments.read','payments.write',
        'contacts.read','contacts.write',
        'accounts.read',
        'reports.read',
        'journal.read','journal.write','journal.void'
    ],
    true
),
(
    '00000000-0000-0000-0000-000000000103',
    NULL,
    'viewer',
    'Read-only access to all records and reports',
    ARRAY[
        'invoices.read',
        'bills.read',
        'payments.read',
        'contacts.read',
        'accounts.read',
        'reports.read',
        'journal.read'
    ],
    true
);

-- demo org
INSERT INTO organizations (id, display_name, legal_name, country_code, currency)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Demo Company',
    'Demo Company Pty Ltd',
    'AU',
    'USD'
);

-- demo user (password: demo1234)
INSERT INTO users (id, email, password_hash, name)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'demo@books.local',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj/RK.s5udem',
    'Demo Admin'
);

-- demo user is admin of demo org
INSERT INTO org_members (user_id, org_id, role_id)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000101'
);

-- default chart of accounts
INSERT INTO accounts (org_id, code, name, type) VALUES
('00000000-0000-0000-0000-000000000001', '1000', 'Cash & Bank',            'asset'),
('00000000-0000-0000-0000-000000000001', '1100', 'Accounts Receivable',    'asset'),
('00000000-0000-0000-0000-000000000001', '1200', 'Inventory',              'asset'),
('00000000-0000-0000-0000-000000000001', '1500', 'Fixed Assets',           'asset'),
('00000000-0000-0000-0000-000000000001', '2000', 'Accounts Payable',       'liability'),
('00000000-0000-0000-0000-000000000001', '2100', 'VAT Payable',            'liability'),
('00000000-0000-0000-0000-000000000001', '2200', 'Accrued Liabilities',    'liability'),
('00000000-0000-0000-0000-000000000001', '3000', 'Share Capital',          'equity'),
('00000000-0000-0000-0000-000000000001', '3100', 'Retained Earnings',      'equity'),
('00000000-0000-0000-0000-000000000001', '4000', 'Sales Revenue',          'income'),
('00000000-0000-0000-0000-000000000001', '4100', 'Other Income',           'income'),
('00000000-0000-0000-0000-000000000001', '5000', 'Cost of Goods Sold',     'expense'),
('00000000-0000-0000-0000-000000000001', '6000', 'Operating Expenses',     'expense'),
('00000000-0000-0000-0000-000000000001', '6100', 'Salaries & Wages',       'expense'),
('00000000-0000-0000-0000-000000000001', '6200', 'Rent',                   'expense'),
('00000000-0000-0000-0000-000000000001', '6900', 'Income Tax Expense',     'expense');

-- demo contacts
INSERT INTO contacts (id, org_id, name, email, type) VALUES
('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'Acme Corp',       'acme@example.com',   'customer'),
('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', 'Global Supplies', 'supply@example.com', 'supplier'),
('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', 'Beta Trading',    'beta@example.com',   'both');

-- +goose Down
DELETE FROM contacts    WHERE org_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM accounts    WHERE org_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM org_members WHERE org_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM users       WHERE id     = '00000000-0000-0000-0000-000000000002';
DELETE FROM organizations WHERE id   = '00000000-0000-0000-0000-000000000001';
DELETE FROM roles       WHERE id IN (
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000102',
    '00000000-0000-0000-0000-000000000103'
);
