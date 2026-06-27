DROP INDEX IF EXISTS auth.users_email_active;
ALTER TABLE auth.users
  DROP COLUMN email,
  DROP COLUMN last_name,
  DROP COLUMN first_name;
