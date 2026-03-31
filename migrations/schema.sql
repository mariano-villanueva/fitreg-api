-- ============================================================
-- 2026-03-31: Structured workout blocks
-- Adds parent_id + extends segment_type ENUM on all 3 segment tables
-- ============================================================

-- workout_segments (children of workouts)
DROP TABLE IF EXISTS workout_segments;
CREATE TABLE workout_segments (
  id             BIGINT        NOT NULL AUTO_INCREMENT PRIMARY KEY,
  workout_id     BIGINT        NOT NULL,
  parent_id      BIGINT        NULL,
  order_index    INT           NOT NULL DEFAULT 0,
  segment_type   ENUM('simple','interval','rest','block') NOT NULL DEFAULT 'simple',
  repetitions    INT           NOT NULL DEFAULT 1,
  value          DECIMAL(10,2) NULL,
  unit           VARCHAR(10)   NULL,
  intensity      VARCHAR(20)   NULL,
  work_value     DECIMAL(10,2) NULL,
  work_unit      VARCHAR(10)   NULL,
  work_intensity VARCHAR(20)   NULL,
  rest_value     DECIMAL(10,2) NULL,
  rest_unit      VARCHAR(10)   NULL,
  rest_intensity VARCHAR(20)   NULL,
  INDEX idx_ws_parent (parent_id),
  FOREIGN KEY (workout_id) REFERENCES workouts(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_id) REFERENCES workout_segments(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- workout_template_segments (children of workout_templates)
DROP TABLE IF EXISTS workout_template_segments;
CREATE TABLE workout_template_segments (
  id             BIGINT        NOT NULL AUTO_INCREMENT PRIMARY KEY,
  template_id    BIGINT        NOT NULL,
  parent_id      BIGINT        NULL,
  order_index    INT           NOT NULL DEFAULT 0,
  segment_type   ENUM('simple','interval','rest','block') NOT NULL DEFAULT 'simple',
  repetitions    INT           DEFAULT 1,
  value          DECIMAL(10,2),
  unit           VARCHAR(10),
  intensity      VARCHAR(20),
  work_value     DECIMAL(10,2),
  work_unit      VARCHAR(10),
  work_intensity VARCHAR(20),
  rest_value     DECIMAL(10,2),
  rest_unit      VARCHAR(10),
  rest_intensity VARCHAR(20),
  INDEX idx_wts_parent (parent_id),
  FOREIGN KEY (template_id) REFERENCES workout_templates(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_id) REFERENCES workout_template_segments(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- weekly_template_day_segments (children of weekly_template_days)
DROP TABLE IF EXISTS weekly_template_day_segments;
CREATE TABLE weekly_template_day_segments (
  id                     BIGINT        NOT NULL AUTO_INCREMENT PRIMARY KEY,
  weekly_template_day_id BIGINT        NOT NULL,
  parent_id              BIGINT        NULL,
  order_index            INT           NOT NULL DEFAULT 0,
  segment_type           ENUM('simple','interval','rest','block') NOT NULL DEFAULT 'simple',
  repetitions            INT           DEFAULT 1,
  value                  DECIMAL(10,2),
  unit                   VARCHAR(10),
  intensity              VARCHAR(20),
  work_value             DECIMAL(10,2),
  work_unit              VARCHAR(10),
  work_intensity         VARCHAR(20),
  rest_value             DECIMAL(10,2),
  rest_unit              VARCHAR(10),
  rest_intensity         VARCHAR(20),
  INDEX idx_wtds_parent (parent_id),
  FOREIGN KEY (weekly_template_day_id) REFERENCES weekly_template_days(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_id) REFERENCES weekly_template_day_segments(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
