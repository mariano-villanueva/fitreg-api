package services

import (
	"errors"
	"testing"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

// stubCoachRepo satisfies repository.CoachRepository with zero-value defaults.
// Tests embed it and override only the methods they need.
type stubCoachRepo struct {
	isStudentOfFn   func(coachID, studentID int64) (bool, error)
	getWeeklyLoadFn func(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error)
}

func (s *stubCoachRepo) IsCoach(userID int64) (bool, error)  { return false, nil }
func (s *stubCoachRepo) IsAdmin(userID int64) (bool, error)  { return false, nil }
func (s *stubCoachRepo) IsStudentOf(coachID, studentID int64) (bool, error) {
	if s.isStudentOfFn != nil {
		return s.isStudentOfFn(coachID, studentID)
	}
	return false, nil
}
func (s *stubCoachRepo) GetStudents(coachID int64) ([]models.CoachStudentInfo, error) {
	return nil, nil
}
func (s *stubCoachRepo) GetRelationship(csID int64) (int64, int64, string, error) {
	return 0, 0, "", nil
}
func (s *stubCoachRepo) EndRelationship(csID int64) error { return nil }
func (s *stubCoachRepo) GetStudentWorkouts(studentID int64) ([]models.Workout, error) {
	return nil, nil
}
func (s *stubCoachRepo) GetDailySummary(coachID int64, date string, includeSegments bool) ([]models.DailySummaryItem, error) {
	return nil, nil
}
func (s *stubCoachRepo) GetUserName(id int64) (string, error) { return "", nil }
func (s *stubCoachRepo) GetWeeklyLoad(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
	if s.getWeeklyLoadFn != nil {
		return s.getWeeklyLoadFn(studentID, weeks)
	}
	return nil, nil
}

var _ repository.CoachRepository = (*stubCoachRepo)(nil) // compile-time check

// --- GetStudentLoad ---

func TestCoachService_GetStudentLoad_ReturnsLoad(t *testing.T) {
	repo := &stubCoachRepo{
		isStudentOfFn: func(coachID, studentID int64) (bool, error) { return true, nil },
		getWeeklyLoadFn: func(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
			return []models.WeeklyLoadEntry{{WeekStart: "2026-03-16", PlannedKm: 42.5}}, nil
		},
	}
	svc := &CoachService{repo: repo}

	load, err := svc.GetStudentLoad(10, 7, 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(load) != 1 || load[0].PlannedKm != 42.5 {
		t.Errorf("unexpected load: %+v", load)
	}
}

func TestCoachService_GetStudentLoad_NotStudent_ReturnsForbidden(t *testing.T) {
	repo := &stubCoachRepo{
		isStudentOfFn: func(coachID, studentID int64) (bool, error) { return false, nil },
	}
	svc := &CoachService{repo: repo}

	_, err := svc.GetStudentLoad(10, 99, 8)
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestCoachService_GetStudentLoad_RepoError_ReturnsError(t *testing.T) {
	repo := &stubCoachRepo{
		isStudentOfFn:   func(coachID, studentID int64) (bool, error) { return true, nil },
		getWeeklyLoadFn: func(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) { return nil, errors.New("db error") },
	}
	svc := &CoachService{repo: repo}

	_, err := svc.GetStudentLoad(10, 7, 8)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- GetMyLoad ---

func TestCoachService_GetMyLoad_ReturnsLoad(t *testing.T) {
	repo := &stubCoachRepo{
		getWeeklyLoadFn: func(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
			return []models.WeeklyLoadEntry{{WeekStart: "2026-03-16", PlannedKm: 30.0}}, nil
		},
	}
	svc := &CoachService{repo: repo}

	load, err := svc.GetMyLoad(42, 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(load) != 1 || load[0].PlannedKm != 30.0 {
		t.Errorf("unexpected load: %+v", load)
	}
}

func TestCoachService_GetMyLoad_RepoError_ReturnsError(t *testing.T) {
	repo := &stubCoachRepo{
		getWeeklyLoadFn: func(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
			return nil, errors.New("db error")
		},
	}
	svc := &CoachService{repo: repo}

	_, err := svc.GetMyLoad(42, 8)
	if err == nil {
		t.Error("expected error, got nil")
	}
}
