ALTER TABLE auth.users
  ADD COLUMN first_name VARCHAR(128),
  ADD COLUMN last_name VARCHAR(128),
  ADD COLUMN email VARCHAR(255);

-- E-posta unique olmalı, ama NULL değerlere izin ver (soft-deleted kullanıcılar için)
CREATE UNIQUE INDEX IF NOT EXISTS users_email_active
  ON auth.users (email)
  WHERE deleted_at IS NULL AND email IS NOT NULL;
