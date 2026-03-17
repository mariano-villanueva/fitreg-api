package services

import (
	"database/sql"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

type AdminService struct {
	repo     repository.AdminRepository
	notifSvc *NotificationService
}

func NewAdminService(repo repository.AdminRepository, notifSvc *NotificationService) *AdminService {
	return &AdminService{repo: repo, notifSvc: notifSvc}
}

func (s *AdminService) GetStats(userID int64) (map[string]int, error) {
	isAdmin, _ := s.repo.IsAdmin(userID)
	if !isAdmin {
		return nil, ErrForbidden
	}
	totalUsers, totalCoaches, totalRatings, pending, err := s.repo.GetStats()
	if err != nil {
		return nil, err
	}
	return map[string]int{
		"total_users":          totalUsers,
		"total_coaches":        totalCoaches,
		"total_ratings":        totalRatings,
		"pending_achievements": pending,
	}, nil
}

func (s *AdminService) ListUsers(userID int64, search, role, sortCol, sortOrder string, limit, offset int) ([]models.AdminUser, int, error) {
	isAdmin, _ := s.repo.IsAdmin(userID)
	if !isAdmin {
		return nil, 0, ErrForbidden
	}
	return s.repo.ListUsers(search, role, sortCol, sortOrder, limit, offset)
}

func (s *AdminService) UpdateUser(callerID, targetID int64, isCoach, isAdmin *bool) error {
	ok, _ := s.repo.IsAdmin(callerID)
	if !ok {
		return ErrForbidden
	}
	return s.repo.UpdateUserRoles(targetID, isCoach, isAdmin)
}

func (s *AdminService) ListPendingAchievements(userID int64) ([]models.AdminPendingAchievement, error) {
	isAdmin, _ := s.repo.IsAdmin(userID)
	if !isAdmin {
		return nil, ErrForbidden
	}
	achievements, err := s.repo.ListPendingAchievements()
	if err != nil {
		return nil, err
	}
	for i := range achievements {
		achievements[i].EventDate = truncateDate(achievements[i].EventDate)
		if achievements[i].ImageFileID != nil {
			if uuid, err := s.repo.GetFileUUID(*achievements[i].ImageFileID); err == nil {
				achievements[i].ImageURL = "/api/files/" + uuid + "/download"
			}
		}
	}
	return achievements, nil
}

func (s *AdminService) VerifyAchievement(achID, adminID int64) error {
	isAdmin, _ := s.repo.IsAdmin(adminID)
	if !isAdmin {
		return ErrForbidden
	}
	coachID, eventName, err := s.repo.VerifyAchievement(achID, adminID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	meta := map[string]interface{}{"achievement_id": achID, "event_name": eventName}
	_ = s.notifSvc.Create(coachID, "achievement_verified", "notif_achievement_verified_title", "notif_achievement_verified_body", meta, nil)
	return nil
}

func (s *AdminService) RejectAchievement(achID, adminID int64, reason string) error {
	isAdmin, _ := s.repo.IsAdmin(adminID)
	if !isAdmin {
		return ErrForbidden
	}
	coachID, eventName, err := s.repo.RejectAchievement(achID, reason)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	meta := map[string]interface{}{
		"achievement_id": achID,
		"event_name":     eventName,
		"reason":         reason,
	}
	_ = s.notifSvc.Create(coachID, "achievement_rejected", "notif_achievement_rejected_title", "notif_achievement_rejected_body", meta, nil)
	return nil
}
