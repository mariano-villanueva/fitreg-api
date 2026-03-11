ALTER TABLE assigned_workouts
    ADD COLUMN image_file_id BIGINT NULL,
    ADD CONSTRAINT fk_assigned_workouts_image FOREIGN KEY (image_file_id) REFERENCES files(id) ON DELETE SET NULL;
