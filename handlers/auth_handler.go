package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/fitreg/api/models"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	DB             *sql.DB
	GoogleClientID string
	JWTSecret      string
}

func NewAuthHandler(db *sql.DB, googleClientID, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		DB:             db,
		GoogleClientID: googleClientID,
		JWTSecret:      jwtSecret,
	}
}

type GoogleLoginRequest struct {
	Credential string `json:"credential"`
}

type GoogleTokenInfo struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Aud     string `json:"aud"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req GoogleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Credential == "" {
		writeError(w, http.StatusBadRequest, "credential is required")
		return
	}

	// Verify the Google ID token
	tokenInfo, err := h.verifyGoogleToken(req.Credential)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid Google token: "+err.Error())
		return
	}

	// Check that the audience matches our client ID
	if tokenInfo.Aud != h.GoogleClientID {
		writeError(w, http.StatusUnauthorized, "Token audience mismatch")
		return
	}

	// Find or create user
	user, err := h.findOrCreateUser(tokenInfo)
	if err != nil {
		log.Printf("ERROR findOrCreateUser: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to process user")
		return
	}

	// Generate JWT
	token, err := h.generateJWT(user.ID, user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		Token: token,
		User:  user,
	})
}

func (h *AuthHandler) verifyGoogleToken(idToken string) (*GoogleTokenInfo, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token verification failed: %s", string(body))
	}

	var tokenInfo GoogleTokenInfo
	if err := json.Unmarshal(body, &tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to parse token info: %w", err)
	}

	return &tokenInfo, nil
}

type userRow struct {
	ID                  int64           `json:"id"`
	GoogleID            string          `json:"google_id"`
	Email               string          `json:"email"`
	Name                string          `json:"name"`
	AvatarURL           sql.NullString  `json:"-"`
	Sex                 sql.NullString  `json:"-"`
	BirthDate           sql.NullString  `json:"-"`
	WeightKg            sql.NullFloat64 `json:"-"`
	HeightCm            sql.NullInt64   `json:"-"`
	Language            sql.NullString  `json:"-"`
	IsCoach             sql.NullBool    `json:"-"`
	IsAdmin             sql.NullBool    `json:"-"`
	CoachDescription    sql.NullString  `json:"-"`
	CoachPublic         sql.NullBool    `json:"-"`
	OnboardingCompleted sql.NullBool    `json:"-"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type userJSON struct {
	ID                  int64     `json:"id"`
	GoogleID            string    `json:"google_id"`
	Email               string    `json:"email"`
	Name                string    `json:"name"`
	AvatarURL           string    `json:"avatar_url"`
	Sex                 string    `json:"sex"`
	BirthDate           string    `json:"birth_date"`
	Age                 int       `json:"age"`
	WeightKg            float64   `json:"weight_kg"`
	HeightCm            int       `json:"height_cm"`
	Language            string    `json:"language"`
	IsCoach             bool      `json:"is_coach"`
	IsAdmin             bool      `json:"is_admin"`
	CoachDescription    string    `json:"coach_description"`
	CoachPublic         bool      `json:"coach_public"`
	OnboardingCompleted bool      `json:"onboarding_completed"`
	HasCoach            bool      `json:"has_coach"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func rowToJSON(row userRow) userJSON {
	u := userJSON{
		ID:        row.ID,
		GoogleID:  row.GoogleID,
		Email:     row.Email,
		Name:      row.Name,
		Language:  "es",
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.AvatarURL.Valid {
		u.AvatarURL = row.AvatarURL.String
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

func (h *AuthHandler) findOrCreateUser(tokenInfo *GoogleTokenInfo) (*userJSON, error) {
	var row userRow
	err := h.DB.QueryRow(`
		SELECT id, google_id, email, name, avatar_url, sex, birth_date, weight_kg, height_cm, language, is_coach, is_admin, coach_description, coach_public, onboarding_completed, created_at, updated_at
		FROM users WHERE google_id = ?
	`, tokenInfo.Sub).Scan(
		&row.ID, &row.GoogleID, &row.Email, &row.Name, &row.AvatarURL,
		&row.Sex, &row.BirthDate, &row.WeightKg, &row.HeightCm, &row.Language, &row.IsCoach, &row.IsAdmin, &row.CoachDescription, &row.CoachPublic, &row.OnboardingCompleted, &row.CreatedAt, &row.UpdatedAt,
	)

	if err == nil {
		// Update name/email/avatar if changed
		if _, err := h.DB.Exec(`
			UPDATE users SET email = ?, name = ?, avatar_url = ?, updated_at = NOW() WHERE google_id = ?
		`, tokenInfo.Email, tokenInfo.Name, tokenInfo.Picture, tokenInfo.Sub); err != nil {
			logErr("update user on login", err)
		}

		row.Email = tokenInfo.Email
		row.Name = tokenInfo.Name
		row.AvatarURL = sql.NullString{String: tokenInfo.Picture, Valid: tokenInfo.Picture != ""}
		u := rowToJSON(row)
		var hasCoach bool
		if err := h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM coach_students WHERE student_id = ? AND status = 'active')", row.ID).Scan(&hasCoach); err != nil {
			logErr("check has coach on login", err)
		}
		u.HasCoach = hasCoach
		return &u, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new user
	result, err := h.DB.Exec(`
		INSERT INTO users (google_id, email, name, avatar_url) VALUES (?, ?, ?, ?)
	`, tokenInfo.Sub, tokenInfo.Email, tokenInfo.Name, tokenInfo.Picture)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		logErr("get last insert id for new user", err)
	}
	err = h.DB.QueryRow(`
		SELECT id, google_id, email, name, avatar_url, sex, birth_date, weight_kg, height_cm, language, is_coach, is_admin, coach_description, coach_public, onboarding_completed, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(
		&row.ID, &row.GoogleID, &row.Email, &row.Name, &row.AvatarURL,
		&row.Sex, &row.BirthDate, &row.WeightKg, &row.HeightCm, &row.Language, &row.IsCoach, &row.IsAdmin, &row.CoachDescription, &row.CoachPublic, &row.OnboardingCompleted, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	u := rowToJSON(row)
	return &u, nil
}

func (h *AuthHandler) generateJWT(userID int64, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.JWTSecret))
}
