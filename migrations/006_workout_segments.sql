CREATE TABLE IF NOT EXISTS assigned_workout_segments (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    assigned_workout_id BIGINT NOT NULL,
    order_index INT NOT NULL DEFAULT 0,
    segment_type ENUM('simple', 'interval') NOT NULL DEFAULT 'simple',
    repetitions INT DEFAULT 1,
    value DECIMAL(8,2) DEFAULT 0,
    unit ENUM('km', 'm', 'min', 'sec') DEFAULT 'km',
    intensity ENUM('easy', 'moderate', 'fast', 'sprint') DEFAULT 'easy',
    work_value DECIMAL(8,2) DEFAULT 0,
    work_unit ENUM('km', 'm', 'min', 'sec') DEFAULT 'km',
    work_intensity ENUM('easy', 'moderate', 'fast', 'sprint') DEFAULT 'fast',
    rest_value DECIMAL(8,2) DEFAULT 0,
    rest_unit ENUM('km', 'm', 'min', 'sec') DEFAULT 'km',
    rest_intensity ENUM('easy', 'moderate', 'fast', 'sprint') DEFAULT 'easy',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_aws_workout (assigned_workout_id),
    CONSTRAINT fk_aws_workout FOREIGN KEY (assigned_workout_id) REFERENCES assigned_workouts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
