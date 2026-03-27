-- Add password_hash to users for JWT authentication
ALTER TABLE users ADD COLUMN password_hash VARCHAR(255);

-- Update existing demo users with bcrypt hash of "demo123"
-- $2a$10$CiRm4LcxQYX7jsCGog2skuVBVV9xtKb3t1cKKQiL72XZmae8iKPr6
UPDATE users SET password_hash = '$2a$10$CiRm4LcxQYX7jsCGog2skuVBVV9xtKb3t1cKKQiL72XZmae8iKPr6';

-- Make password_hash required for new users going forward
-- (existing rows already have a value from the UPDATE above)
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
