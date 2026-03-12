-- Upgrade script for existing production databases
-- Run this if you already have the base schema and need to add missing columns

-- Files table (if not exists)
CREATE TABLE IF NOT EXISTS files (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_key VARCHAR(500) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_files_user (user_id),
    CONSTRAINT fk_files_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Add feeling column to workouts
ALTER TABLE workouts ADD COLUMN IF NOT EXISTS feeling INT NULL AFTER avg_heart_rate;

-- Add fartlek to workout type enum
ALTER TABLE workouts MODIFY COLUMN type ENUM('easy','tempo','intervals','long_run','race','fartlek','other') DEFAULT 'easy';

-- Add image and public visibility to coach achievements
ALTER TABLE coach_achievements ADD COLUMN IF NOT EXISTS image_file_id BIGINT NULL;
ALTER TABLE coach_achievements ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT TRUE;

-- Add image to assigned workouts
ALTER TABLE assigned_workouts ADD COLUMN IF NOT EXISTS image_file_id BIGINT NULL;

-- Add foreign keys (ignore errors if already exist)
-- ALTER TABLE coach_achievements ADD CONSTRAINT fk_achievement_image FOREIGN KEY (image_file_id) REFERENCES files(id) ON DELETE SET NULL;
-- ALTER TABLE assigned_workouts ADD CONSTRAINT fk_assigned_workouts_image FOREIGN KEY (image_file_id) REFERENCES files(id) ON DELETE SET NULL;
