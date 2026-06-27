CREATE SCHEMA IF NOT EXISTS auth;

CREATE TABLE IF NOT EXISTS auth.tenants (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT tenants_name_unique UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS auth.users (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID         NOT NULL REFERENCES auth.tenants(id),
    username              VARCHAR(64)  NOT NULL,
    password_hash         VARCHAR(255) NOT NULL,
    force_password_change BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ
);

-- Soft-delete farkında partial unique index: aynı tenant'ta aynı username yaşayanlar arasında tek
CREATE UNIQUE INDEX IF NOT EXISTS users_tenant_username_active
    ON auth.users (tenant_id, username)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS auth.roles (
    id   SERIAL      PRIMARY KEY,
    name VARCHAR(32) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS auth.user_roles (
    user_id UUID    NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES auth.roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- Sabit rol seed'i
INSERT INTO auth.roles (name) VALUES ('admin'), ('operator'), ('viewer')
    ON CONFLICT DO NOTHING;
