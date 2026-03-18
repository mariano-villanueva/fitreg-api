package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "userID"

func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for public routes
			if r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/api/auth/") {
				next.ServeHTTP(w, r)
				return
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from Authorization header only
			var tokenString string
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
				if tokenString == authHeader {
					http.Error(w, `{"error":"Invalid authorization format"}`, http.StatusUnauthorized)
					return
				}
			}
			if tokenString == "" {
				http.Error(w, `{"error":"Missing authorization"}`, http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"Invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"Invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			userIDFloat, ok := claims["user_id"].(float64)
			if !ok {
				http.Error(w, `{"error":"Invalid user ID in token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, int64(userIDFloat))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) int64 {
	userID, _ := ctx.Value(userIDKey).(int64)
	return userID
}
