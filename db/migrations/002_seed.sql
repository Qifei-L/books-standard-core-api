-- +goose Up
-- demo org + admin user (password: demo1234)
INSERT INTO organizations (id, name, currency)
VALUES ('00000000-0000-0000-0000-000000000001', 'Demo Company', 'USD');

INSERT INTO users (id, org_id, email, password_hash, name, role)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'demo@books.local',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj/RK.s5udem',
    'Demo Admin',
    'admin'
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
('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'Acme Corp',        'acme@example.com',    'customer'),
('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', 'Global Supplies',  'supply@example.com',  'supplier'),
('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', 'Beta Trading',     'beta@example.com',    'both');

-- +goose Down
DELETE FROM contacts  WHERE org_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM accounts  WHERE org_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM users     WHERE org_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM organizations WHERE id = '00000000-0000-0000-0000-000000000001';
