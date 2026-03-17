package services

import (
	"database/sql"
	"errors"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

var (
	ErrAchievementVerified    = errors.New("cannot edit a verified achievement")
	ErrAchievementNotRejected = errors.New("only rejected achievements can be edited")
)

type AchievementService struct {
	repo     repository.AchievementRepository
	notifSvc *NotificationService
	userRepo repository.UserRepository
}

func NewAchievementService(repo repository.AchievementRepository, notifSvc *NotificationService, userRepo repository.UserRepository) *AchievementService {
	return &AchievementService{repo: repo, notifSvc: notifSvc, userRepo: userRepo}
}

func (s *AchievementService) ListMy(coachID int64) ([]models.CoachAchievement, error) {
	achievements, err := s.repo.List(coachID)
	if err != nil {
		return nil, err
	}
	for i := range achievements {
		achievements[i].EventDate = truncateDate(achievements[i].EventDate)
		s.populateImageURL(&achievements[i])
	}
	return achievements, nil
}

func (s *AchievementService) Create(coachID int64, req models.CreateAchievementRequest) (int64, error) {
	isCoach, err := s.repo.IsCoach(coachID)
	if err != nil {
		return 0, err
	}
	if !isCoach {
		return 0, ErrNotCoach
	}
	id, err := s.repo.Create(coachID, req)
	if err != nil {
		return 0, err
	}
	s.notifyAdmins(coachID, id, req.EventName)
	return id, nil
}

func (s *AchievementService) Update(achID, coachID int64, req models.UpdateAchievementRequest) error {
	isVerified, rejectionReason, err := s.repo.GetForEdit(achID, coachID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if isVerified {
		return ErrAchievementVerified
	}
	if rejectionReason == "" {
		return ErrAchievementNotRejected
	}
	if err := s.repo.Update(achID, coachID, req); err != nil {
		return err
	}
	s.notifyAdmins(coachID, achID, req.EventName)
	return nil
}

func (s *AchievementService) Delete(achID, coachID int64) error {
	isVerified, rejectionReason, err := s.repo.GetForEdit(achID, coachID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if isVerified || rejectionReason == "" {
		return ErrAchievementNotRejected
	}
	_, err = s.repo.Delete(achID, coachID)
	return err
}

func (s *AchievementService) SetVisibility(achID, coachID int64, isPublic bool) error {
	found, err := s.repo.SetVisibility(achID, coachID, isPublic)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

func (s *AchievementService) populateImageURL(a *models.CoachAchievement) {
	if a.ImageFileID == nil {
		return
	}
	uuid, err := s.repo.GetFileUUID(*a.ImageFileID)
	if err != nil {
		return
	}
	a.ImageURL = "/api/files/" + uuid + "/download"
}

func (s *AchievementService) notifyAdmins(coachID, achID int64, eventName string) {
	coachName, _, _ := s.userRepo.GetNameAndAvatar(coachID)
	adminIDs, err := s.repo.GetAdminIDs()
	if err != nil {
		return
	}
	meta := map[string]interface{}{
		"achievement_id": achID,
		"event_name":     eventName,
		"coach_name":     coachName,
	}
	for _, adminID := range adminIDs {
		_ = s.notifSvc.Create(adminID, "achievement_pending",
			"notif_achievement_pending_title", "notif_achievement_pending_body", meta, nil)
	}
}
