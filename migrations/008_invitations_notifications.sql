-- Drop old coach_students table (clean start per spec)
DROP TABLE IF EXISTS assigned_workout_segments;
DROP TABLE IF EXISTS assigned_workouts;
DROP TABLE IF EXISTS coach_ratings;
DROP TABLE IF EXISTS coach_students;

-- Recreate coach_students with new schema
CREATE TABLE coach_students (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    coach_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    invitation_id BIGINT NULL,
    status ENUM('active', 'finished') NOT NULL DEFAULT 'active',
    started_at DATETIME NOT NULL DEFAULT NOW(),
    finished_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT NOW(),
    FOREIGN KEY (coach_id) REFERENCES users(id),
    FOREIGN KEY (student_id) REFERENCES users(id)
);

-- Recreate dependent tables
CREATE TABLE assigned_workouts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    coach_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50),
    distance_km DECIMAL(10,2),
    duration_seconds INT,
    notes TEXT,
    status ENUM('pending','completed','skipped') NOT NULL DEFAULT 'pending',
    due_date DATE,
    created_at DATETIME NOT NULL DEFAULT NOW(),
    updated_at DATETIME NOT NULL DEFAULT NOW() ON UPDATE NOW(),
    FOREIGN KEY (coach_id) REFERENCES users(id),
    FOREIGN KEY (student_id) REFERENCES users(id)
);

CREATE TABLE assigned_workout_segments (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    assigned_workout_id BIGINT NOT NULL,
    order_index INT NOT NULL DEFAULT 0,
    segment_type ENUM('simple','interval') NOT NULL DEFAULT 'simple',
    repetitions INT DEFAULT 1,
    value DECIMAL(10,2),
    unit VARCHAR(10),
    intensity VARCHAR(20),
    work_value DECIMAL(10,2),
    work_unit VARCHAR(10),
    work_intensity VARCHAR(20),
    rest_value DECIMAL(10,2),
    rest_unit VARCHAR(10),
    rest_intensity VARCHAR(20),
    FOREIGN KEY (assigned_workout_id) REFERENCES assigned_workouts(id) ON DELETE CASCADE
);

CREATE TABLE coach_ratings (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    coach_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    rating INT NOT NULL,
    comment TEXT,
    created_at DATETIME NOT NULL DEFAULT NOW(),
    updated_at DATETIME NOT NULL DEFAULT NOW() ON UPDATE NOW(),
    FOREIGN KEY (coach_id) REFERENCES users(id),
    FOREIGN KEY (student_id) REFERENCES users(id),
    UNIQUE KEY uk_coach_student_rating (coach_id, student_id)
);

-- Invitations table
CREATE TABLE invitations (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    type ENUM('coach_invite', 'student_request') NOT NULL,
    sender_id BIGINT NOT NULL,
    receiver_id BIGINT NOT NULL,
    message TEXT NULL,
    status ENUM('pending', 'accepted', 'rejected', 'cancelled') NOT NULL DEFAULT 'pending',
    created_at DATETIME NOT NULL DEFAULT NOW(),
    updated_at DATETIME NOT NULL DEFAULT NOW() ON UPDATE NOW(),
    FOREIGN KEY (sender_id) REFERENCES users(id),
    FOREIGN KEY (receiver_id) REFERENCES users(id),
    INDEX idx_sender_status (sender_id, status),
    INDEX idx_receiver_status (receiver_id, status)
);

-- Add FK from coach_students to invitations (after invitations table exists)
ALTER TABLE coach_students ADD FOREIGN KEY (invitation_id) REFERENCES invitations(id);

-- Notifications table
CREATE TABLE notifications (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    metadata JSON,
    actions JSON NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id),
    INDEX idx_user_read_created (user_id, is_read, created_at DESC)
);

-- Notification preferences table
CREATE TABLE notification_preferences (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    workout_assigned BOOLEAN NOT NULL DEFAULT TRUE,
    workout_completed_or_skipped BOOLEAN NOT NULL DEFAULT TRUE,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE KEY uk_user_prefs (user_id)
);
