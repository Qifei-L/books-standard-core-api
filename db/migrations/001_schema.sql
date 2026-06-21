-- +goose Up

CREATE TABLE organizations (
    id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name            TEXT        NOT NULL,
    legal_name              TEXT,
    country_code            TEXT        NOT NULL DEFAULT 'AU',
    currency                TEXT        NOT NULL DEFAULT 'AUD',
    timezone                TEXT        NOT NULL DEFAULT 'UTC',
    fiscal_year_start_month INT         NOT NULL DEFAULT 1 CHECK (fiscal_year_start_month BETWEEN 1 AND 12),
    registration_no         TEXT,
    address                 TEXT,
    phone                   TEXT,
    email                   TEXT,
    logo_url                TEXT,
    slug                    TEXT        UNIQUE,
    settings                JSONB       NOT NULL DEFAULT '{}',
    is_active               BOOLEAN     NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    name          TEXT        NOT NULL,
    is_active     BOOLEAN     NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE roles (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        REFERENCES organizations(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    description TEXT,
    permissions TEXT[]      NOT NULL DEFAULT '{}',
    is_system   BOOLEAN     NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (org_id, name)
);

CREATE TABLE org_members (
    id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id    UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role_id   UUID        NOT NULL REFERENCES roles(id),
    is_active BOOLEAN     NOT NULL DEFAULT true,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, org_id)
);

CREATE TABLE refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id     UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE accounts (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID        NOT NULL REFERENCES organizations(id),
    code       TEXT        NOT NULL,
    name       TEXT        NOT NULL,
    type       TEXT        NOT NULL CHECK (type IN ('asset','liability','equity','income','expense')),
    is_active  BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, code)
);

CREATE TABLE contacts (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID        NOT NULL REFERENCES organizations(id),
    name       TEXT        NOT NULL,
    email      TEXT,
    phone      TEXT,
    type       TEXT        NOT NULL CHECK (type IN ('customer','supplier','both')),
    is_active  BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE invoices (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID          NOT NULL REFERENCES organizations(id),
    contact_id   UUID          NOT NULL REFERENCES contacts(id),
    number       TEXT          NOT NULL,
    issue_date   DATE          NOT NULL,
    due_date     DATE,
    status       TEXT          NOT NULL DEFAULT 'draft'
                               CHECK (status IN ('draft','approved','paid','voided')),
    subtotal     NUMERIC(15,2) NOT NULL DEFAULT 0,
    tax_amount   NUMERIC(15,2) NOT NULL DEFAULT 0,
    total        NUMERIC(15,2) NOT NULL DEFAULT 0,
    amount_due   NUMERIC(15,2) NOT NULL DEFAULT 0,
    currency     TEXT          NOT NULL DEFAULT 'AUD',
    notes        TEXT,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    UNIQUE (org_id, number)
);

CREATE TABLE invoice_lines (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id   UUID          NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    description  TEXT          NOT NULL,
    quantity     NUMERIC(15,4) NOT NULL DEFAULT 1,
    unit_price   NUMERIC(15,4) NOT NULL,
    tax_rate     NUMERIC(5,4)  NOT NULL DEFAULT 0,
    amount       NUMERIC(15,2) NOT NULL,
    account_code TEXT          NOT NULL,
    line_no      INT           NOT NULL
);

CREATE TABLE bills (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID          NOT NULL REFERENCES organizations(id),
    contact_id   UUID          NOT NULL REFERENCES contacts(id),
    number       TEXT,
    reference    TEXT,
    issue_date   DATE          NOT NULL,
    due_date     DATE,
    status       TEXT          NOT NULL DEFAULT 'draft'
                               CHECK (status IN ('draft','approved','paid','voided')),
    subtotal     NUMERIC(15,2) NOT NULL DEFAULT 0,
    tax_amount   NUMERIC(15,2) NOT NULL DEFAULT 0,
    total        NUMERIC(15,2) NOT NULL DEFAULT 0,
    amount_due   NUMERIC(15,2) NOT NULL DEFAULT 0,
    currency     TEXT          NOT NULL DEFAULT 'AUD',
    notes        TEXT,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE TABLE bill_lines (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    bill_id      UUID          NOT NULL REFERENCES bills(id) ON DELETE CASCADE,
    description  TEXT          NOT NULL,
    quantity     NUMERIC(15,4) NOT NULL DEFAULT 1,
    unit_price   NUMERIC(15,4) NOT NULL,
    tax_rate     NUMERIC(5,4)  NOT NULL DEFAULT 0,
    amount       NUMERIC(15,2) NOT NULL,
    account_code TEXT          NOT NULL,
    line_no      INT           NOT NULL
);

CREATE TABLE payments (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID          NOT NULL REFERENCES organizations(id),
    type         TEXT          NOT NULL CHECK (type IN ('ar','ap')),
    reference_id UUID          NOT NULL,
    date         DATE          NOT NULL,
    amount       NUMERIC(15,2) NOT NULL,
    account_code TEXT          NOT NULL,
    reference    TEXT,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE TABLE journal_entries (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        NOT NULL REFERENCES organizations(id),
    date        DATE        NOT NULL,
    reference   TEXT,
    description TEXT        NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'posted'
                            CHECK (status IN ('posted','voided')),
    source_type TEXT,
    source_id   UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE journal_lines (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id     UUID          NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
    account_code TEXT          NOT NULL,
    description  TEXT,
    debit        NUMERIC(15,2) NOT NULL DEFAULT 0,
    credit       NUMERIC(15,2) NOT NULL DEFAULT 0,
    line_no      INT           NOT NULL
);

-- updated_at trigger (shared function, one trigger per table)
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_organizations_updated_at BEFORE UPDATE ON organizations FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_users_updated_at         BEFORE UPDATE ON users         FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_roles_updated_at         BEFORE UPDATE ON roles         FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_org_members_updated_at   BEFORE UPDATE ON org_members   FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_accounts_updated_at      BEFORE UPDATE ON accounts      FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_contacts_updated_at      BEFORE UPDATE ON contacts      FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_invoices_updated_at      BEFORE UPDATE ON invoices      FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_bills_updated_at         BEFORE UPDATE ON bills         FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_org_members_user    ON org_members(user_id);
CREATE INDEX idx_org_members_org     ON org_members(org_id);
CREATE INDEX idx_roles_org           ON roles(org_id);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_invoices_org_status ON invoices(org_id, status);
CREATE INDEX idx_bills_org_status    ON bills(org_id, status);
CREATE INDEX idx_payments_ref        ON payments(reference_id);
CREATE INDEX idx_journal_entries_org ON journal_entries(org_id, date);
CREATE INDEX idx_journal_lines_entry ON journal_lines(entry_id);
CREATE INDEX idx_journal_lines_acct  ON journal_lines(account_code);

-- +goose Down
DROP TRIGGER IF EXISTS trg_bills_updated_at         ON bills;
DROP TRIGGER IF EXISTS trg_invoices_updated_at      ON invoices;
DROP TRIGGER IF EXISTS trg_contacts_updated_at      ON contacts;
DROP TRIGGER IF EXISTS trg_accounts_updated_at      ON accounts;
DROP TRIGGER IF EXISTS trg_org_members_updated_at   ON org_members;
DROP TRIGGER IF EXISTS trg_roles_updated_at         ON roles;
DROP TRIGGER IF EXISTS trg_users_updated_at         ON users;
DROP TRIGGER IF EXISTS trg_organizations_updated_at ON organizations;
DROP FUNCTION IF EXISTS set_updated_at;

DROP TABLE IF EXISTS journal_lines;
DROP TABLE IF EXISTS journal_entries;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS bill_lines;
DROP TABLE IF EXISTS bills;
DROP TABLE IF EXISTS invoice_lines;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS contacts;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;
