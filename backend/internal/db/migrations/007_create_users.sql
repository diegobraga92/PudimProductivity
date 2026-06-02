CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    role        TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed a default admin user (replace email/password in production)
INSERT INTO users (id, email, role)
VALUES (gen_random_uuid(), 'admin@pudimproductivity.com', 'admin')
ON CONFLICT (email) DO NOTHING;

-- Seed a default regular user for local development
INSERT INTO users (id, email, role)
VALUES (gen_random_uuid(), 'user@pudimproductivity.com', 'user')
ON CONFLICT (email) DO NOTHING;