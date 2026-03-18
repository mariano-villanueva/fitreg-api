package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/fitreg/api/config"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
	"github.com/golang-jwt/jwt/v5"
)

// GoogleTokenInfo holds data from Google's token verification endpoint.
type GoogleTokenInfo struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Aud     string `json:"aud"`
}

// AuthResponse is the response returned after successful authentication.
type AuthResponse struct {
	Token string              `json:"token"`
	User  *models.UserProfile `json:"user"`
}

// AuthService contains business logic for authentication.
type AuthService struct {
	repo           repository.UserRepository
	googleClientID string
	jwtSecret      string
}

// NewAuthService constructs an AuthService.
func NewAuthService(repo repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		repo:           repo,
		googleClientID: cfg.GoogleClientID,
		jwtSecret:      cfg.JWTSecret,
	}
}

// GoogleLogin verifies a Google credential token, finds or creates the user,
// and returns a JWT + user profile.
func (s *AuthService) GoogleLogin(credential string) (*AuthResponse, error) {
	if credential == "" {
		return nil, fmt.Errorf("credential is required")
	}

	tokenInfo, err := s.verifyGoogleToken(credential)
	if err != nil {
		return nil, fmt.Errorf("invalid Google token: %w", err)
	}

	if tokenInfo.Aud != s.googleClientID {
		return nil, fmt.Errorf("token audience mismatch")
	}

	user, err := s.findOrCreateUser(tokenInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to process user: %w", err)
	}

	token, err := s.generateJWT(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

// verifyGoogleToken makes an outbound HTTP call to Google's tokeninfo endpoint.
// Tech debt: this outbound HTTP call ideally belongs behind a provider interface for testability.
// Acceptable for this refactor phase; can be extracted in a future improvement.
func (s *AuthService) verifyGoogleToken(idToken string) (*GoogleTokenInfo, error) {
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
		// Log the full Google response for server-side debugging, but never send it to the client.
		log.Printf("Google token verification failed (status %d): %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("token verification failed")
	}

	var tokenInfo GoogleTokenInfo
	if err := json.Unmarshal(body, &tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to parse token info: %w", err)
	}

	return &tokenInfo, nil
}

func (s *AuthService) findOrCreateUser(tokenInfo *GoogleTokenInfo) (*models.UserProfile, error) {
	row, err := s.repo.FindByGoogleID(tokenInfo.Sub)

	if err == nil {
		// Existing user -- update login info
		if err := s.repo.UpdateOnLogin(tokenInfo.Sub, tokenInfo.Email, tokenInfo.Name, tokenInfo.Picture); err != nil {
			log.Printf("ERROR update user on login: %v", err)
		}

		row.Email = tokenInfo.Email
		row.Name = tokenInfo.Name
		row.AvatarURL = sql.NullString{String: tokenInfo.Picture, Valid: tokenInfo.Picture != ""}
		u := rowToUserProfile(row)
		hasCoach, err := s.repo.HasActiveCoach(row.ID)
		if err != nil {
			log.Printf("ERROR check has coach on login: %v", err)
		}
		u.HasCoach = hasCoach
		if hasCoach {
			fillCoachInfo(s.repo, row.ID, &u)
		}
		return &u, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new user
	id, err := s.repo.Create(tokenInfo.Sub, tokenInfo.Email, tokenInfo.Name, tokenInfo.Picture)
	if err != nil {
		return nil, err
	}

	row, err = s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	u := rowToUserProfile(row)
	return &u, nil
}

func (s *AuthService) generateJWT(userID int64, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
