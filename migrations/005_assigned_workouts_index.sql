-- Add composite index to speed up daily summary queries per student/date
CREATE INDEX idx_aw_student_date ON assigned_workouts(student_id, due_date);
