ALTER TABLE users ADD COLUMN birth_date DATE NULL AFTER weight_kg;
ALTER TABLE users ADD COLUMN height_cm INT NULL AFTER birth_date;
ALTER TABLE users ADD COLUMN onboarding_completed BOOLEAN DEFAULT FALSE AFTER height_cm;
ALTER TABLE users DROP COLUMN age;
