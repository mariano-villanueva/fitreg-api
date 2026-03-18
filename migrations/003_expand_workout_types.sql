-- Expand workouts.type ENUM to include all activity types
-- consistent with assigned_workouts.type (VARCHAR) and the FE WORKOUT_TYPES constant

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
