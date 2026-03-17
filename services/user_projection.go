package services

import (
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

// rowToUserProfile converts a models.UserRow (DB scan) to a models.UserProfile (API response).
func rowToUserProfile(row models.UserRow) models.UserProfile {
	u := models.UserProfile{
		ID:        row.ID,
		GoogleID:  row.GoogleID,
		Email:     row.Email,
		Name:      row.Name,
		Language:  "es",
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.CustomAvatar.Valid {
		u.CustomAvatar = row.CustomAvatar.String
		u.AvatarURL = row.CustomAvatar.String
	}
	if row.Sex.Valid {
		u.Sex = row.Sex.String
	}
	if row.BirthDate.Valid {
		bd := truncateDate(row.BirthDate.String)
		u.BirthDate = bd
		u.Age = models.CalculateAge(bd)
	}
	if row.WeightKg.Valid {
		u.WeightKg = row.WeightKg.Float64
	}
	if row.HeightCm.Valid {
		u.HeightCm = int(row.HeightCm.Int64)
	}
	if row.Language.Valid {
		u.Language = row.Language.String
	}
	if row.IsCoach.Valid {
		u.IsCoach = row.IsCoach.Bool
	}
	if row.IsAdmin.Valid {
		u.IsAdmin = row.IsAdmin.Bool
	}
	if row.CoachDescription.Valid {
		u.CoachDescription = row.CoachDescription.String
	}
	if row.CoachPublic.Valid {
		u.CoachPublic = row.CoachPublic.Bool
	}
	if row.OnboardingCompleted.Valid {
		u.OnboardingCompleted = row.OnboardingCompleted.Bool
	}
	return u
}

// fillCoachInfo populates coach fields on a UserProfile if the user has an active coach.
func fillCoachInfo(repo repository.UserRepository, studentID int64, u *models.UserProfile) {
	coachID, name, avatar, found := repo.GetActiveCoach(studentID)
	if !found {
		return
	}
	u.CoachID = coachID
	u.CoachName = name
	u.CoachAvatar = avatar
}

// truncateDate truncates a datetime string to YYYY-MM-DD.
func truncateDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
