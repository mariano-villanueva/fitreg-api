ALTER TABLE coach_achievements
    ADD COLUMN image_file_id BIGINT NULL,
    ADD CONSTRAINT fk_achievement_image FOREIGN KEY (image_file_id) REFERENCES files(id) ON DELETE SET NULL;
