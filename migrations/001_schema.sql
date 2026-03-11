-- FitReg complete schema (consolidated)
-- Run on a fresh database: mysql -u root -p fitreg < migrations/001_schema.sql

-- Drop all tables in reverse dependency order
SET FOREIGN_KEY_CHECKS = 0;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS assigned_workout_segments;
DROP TABLE IF EXISTS assigned_workouts;
DROP TABLE IF EXISTS coach_ratings;
DROP TABLE IF EXISTS coach_achievements;
DROP TABLE IF EXISTS coach_students;
DROP TABLE IF EXISTS invitations;
DROP TABLE IF EXISTS workouts;
DROP TABLE IF EXISTS users;
SET FOREIGN_KEY_CHECKS = 1;

-- ============================================================
-- USERS
-- ============================================================
CREATE TABLE users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    google_id VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    sex ENUM('M','F','other'),
    weight_kg DECIMAL(5,2),
    birth_date DATE NULL,
    height_cm INT NULL,
    onboarding_completed BOOLEAN DEFAULT FALSE,
    language VARCHAR(5) DEFAULT 'es',
    is_coach BOOLEAN DEFAULT FALSE,
    is_admin BOOLEAN DEFAULT FALSE,
    coach_description TEXT,
    coach_public BOOLEAN DEFAULT FALSE,
    coach_locality VARCHAR(255) DEFAULT NULL,
    coach_level VARCHAR(255) DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_users_google_id (google_id),
    INDEX idx_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- WORKOUTS
-- ============================================================
CREATE TABLE workouts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    date DATE NOT NULL,
    distance_km DECIMAL(6,2) NOT NULL DEFAULT 0,
    duration_seconds INT NOT NULL DEFAULT 0,
    avg_pace VARCHAR(10),
    calories INT DEFAULT 0,
    avg_heart_rate INT DEFAULT 0,
    type ENUM('easy','tempo','intervals','long_run','race','other') DEFAULT 'easy',
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_workouts_user_id (user_id),
    INDEX idx_workouts_date (date),
    CONSTRAINT fk_workouts_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- INVITATIONS
-- ============================================================
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- COACH - STUDENTS
-- ============================================================
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
    FOREIGN KEY (student_id) REFERENCES users(id),
    FOREIGN KEY (invitation_id) REFERENCES invitations(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- COACH ACHIEVEMENTS
-- ============================================================
CREATE TABLE coach_achievements (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    coach_id BIGINT NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    event_date DATE NOT NULL,
    distance_km DECIMAL(6,2),
    result_time VARCHAR(10),
    position INT,
    extra_info VARCHAR(500),
    is_verified BOOLEAN DEFAULT FALSE,
    rejection_reason VARCHAR(200),
    verified_by BIGINT,
    verified_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_ca_coach (coach_id),
    CONSTRAINT fk_ca_coach FOREIGN KEY (coach_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_ca_verifier FOREIGN KEY (verified_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- COACH RATINGS
-- ============================================================
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- ASSIGNED WORKOUTS
-- ============================================================
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
    expected_fields JSON NULL,
    result_time_seconds INT NULL,
    result_distance_km DECIMAL(10,2) NULL,
    result_heart_rate INT NULL,
    result_feeling INT NULL,
    status ENUM('pending','completed','skipped') NOT NULL DEFAULT 'pending',
    due_date DATE,
    created_at DATETIME NOT NULL DEFAULT NOW(),
    updated_at DATETIME NOT NULL DEFAULT NOW() ON UPDATE NOW(),
    FOREIGN KEY (coach_id) REFERENCES users(id),
    FOREIGN KEY (student_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- ASSIGNED WORKOUT SEGMENTS
-- ============================================================
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- NOTIFICATIONS
-- ============================================================
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- NOTIFICATION PREFERENCES
-- ============================================================
CREATE TABLE notification_preferences (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    workout_assigned BOOLEAN NOT NULL DEFAULT TRUE,
    workout_completed_or_skipped BOOLEAN NOT NULL DEFAULT TRUE,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE KEY uk_user_prefs (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
