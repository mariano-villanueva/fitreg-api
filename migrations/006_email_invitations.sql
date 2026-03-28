-- 006_email_invitations.sql
ALTER TABLE invitations
  ADD COLUMN receiver_email VARCHAR(255) NULL AFTER receiver_id,
  ADD COLUMN invite_token   VARCHAR(64)  NULL UNIQUE AFTER receiver_email;
