ALTER TABLE users ADD COLUMN is_coach BOOLEAN DEFAULT FALSE AFTER language;

CREATE TABLE IF NOT EXISTS coach_students (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    coach_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_coach_student (coach_id, student_id),
    CONSTRAINT fk_cs_coach FOREIGN KEY (coach_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_cs_student FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS assigned_workouts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    coach_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    type ENUM('easy','tempo','intervals','long_run','race','fartlek','other') DEFAULT 'easy',
    distance_km DECIMAL(6,2) DEFAULT 0,
    duration_seconds INT DEFAULT 0,
    notes TEXT,
    status ENUM('pending','completed','skipped') DEFAULT 'pending',
    due_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_aw_coach (coach_id),
    INDEX idx_aw_student (student_id),
    INDEX idx_aw_status (status),
    CONSTRAINT fk_aw_coach FOREIGN KEY (coach_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_aw_student FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
