-- ============================================================
-- FitReg — Production update 2026-03-18
-- ============================================================
-- Changes:
--   1. Expand workouts.type ENUM to include all activity types
--      (running, cycling, swimming, strength, cardio, yoga +
--       the existing running-specific types)
--
-- Safe to re-run: existing rows with valid values are unaffected.
-- If workouts.type already includes these values, MySQL will
-- silently accept the re-definition with no data loss.
-- ============================================================

ALTER TABLE workouts
  MODIFY COLUMN type ENUM(
    'running',
    'cycling',
    'swimming',
    'strength',
    'cardio',
    'yoga',
    'easy',
    'tempo',
    'intervals',
    'long_run',
    'race',
    'fartlek',
    'other'
  ) NOT NULL DEFAULT 'other';
