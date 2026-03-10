-- FitReg complete schema
-- Drop all tables in reverse dependency order
DROP TABLE IF EXISTS coach_ratings;
DROP TABLE IF EXISTS coach_achievements;
DROP TABLE IF EXISTS assigned_workout_segments;
DROP TABLE IF EXISTS assigned_workouts;
DROP TABLE IF EXISTS coach_students;
DROP TABLE IF EXISTS workouts;
DROP TABLE IF EXISTS users;

CREATE TABLE users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    google_id VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    sex ENUM('M','F','other'),
    age INT,
    weight_kg DECIMAL(5,2),
    language VARCHAR(5) DEFAULT 'es',
    is_coach BOOLEAN DEFAULT FALSE,
    is_admin BOOLEAN DEFAULT FALSE,
    coach_description TEXT,
    coach_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_users_google_id (google_id),
    INDEX idx_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

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

CREATE TABLE coach_students (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    coach_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_coach_student (coach_id, student_id),
    CONSTRAINT fk_cs_coach FOREIGN KEY (coach_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_cs_student FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE coach_achievements (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    coach_id BIGINT NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    event_date DATE NOT NULL,
    distance_km DECIMAL(6,2),
    result_time VARCHAR(10),
    position INT,
    is_verified BOOLEAN DEFAULT FALSE,
    verified_by BIGINT,
    verified_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_ca_coach (coach_id),
    CONSTRAINT fk_ca_coach FOREIGN KEY (coach_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_ca_verifier FOREIGN KEY (verified_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE coach_ratings (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    coach_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    rating INT NOT NULL,
    comment TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_coach_student_rating (coach_id, student_id),
    INDEX idx_cr_coach (coach_id),
    CONSTRAINT fk_cr_coach FOREIGN KEY (coach_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_cr_student FOREIGN KEY (student_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE assigned_workouts (
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

CREATE TABLE assigned_workout_segments (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    assigned_workout_id BIGINT NOT NULL,
    order_index INT NOT NULL DEFAULT 0,
    segment_type ENUM('simple','interval') NOT NULL DEFAULT 'simple',
    repetitions INT DEFAULT 1,
    value DECIMAL(8,2) DEFAULT 0,
    unit ENUM('km','m','min','sec') DEFAULT 'km',
    intensity ENUM('easy','moderate','fast','sprint') DEFAULT 'easy',
    work_value DECIMAL(8,2) DEFAULT 0,
    work_unit ENUM('km','m','min','sec') DEFAULT 'km',
    work_intensity ENUM('easy','moderate','fast','sprint') DEFAULT 'fast',
    rest_value DECIMAL(8,2) DEFAULT 0,
    rest_unit ENUM('km','m','min','sec') DEFAULT 'km',
    rest_intensity ENUM('easy','moderate','fast','sprint') DEFAULT 'easy',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_aws_workout (assigned_workout_id),
    CONSTRAINT fk_aws_workout FOREIGN KEY (assigned_workout_id) REFERENCES assigned_workouts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
