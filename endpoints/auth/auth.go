package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"strings"
)

type Config struct {
	Users map[string]string
}

type contextKey struct{}

var userContextKey contextKey

func LoadFromEnv() (Config, error) {
	usersValue := strings.TrimSpace(os.Getenv("SCRIPTPORTAL_BASIC_AUTH_USERS"))
	if usersValue == "" {
		if os.Getenv("SCRIPTPORTAL_ALLOW_UNAUTHENTICATED") == "true" {
			return Config{}, nil
		}
		return Config{}, errors.New("SCRIPTPORTAL_BASIC_AUTH_USERS must be set (or SCRIPTPORTAL_ALLOW_UNAUTHENTICATED=true to run without authentication)")
	}

	users := make(map[string]string)
	for _, entry := range strings.Split(usersValue, ",") {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			return Config{}, errors.New("malformed SCRIPTPORTAL_BASIC_AUTH_USERS entry")
		}
		username := strings.TrimSpace(parts[0])
		password := strings.TrimSpace(parts[1])
		if username == "" || password == "" {
			return Config{}, errors.New("malformed SCRIPTPORTAL_BASIC_AUTH_USERS entry")
		}
		users[username] = password
	}

	return Config{Users: users}, nil
}

func (c Config) Enabled() bool {
	return len(c.Users) > 0
}

func (c Config) Authenticate(r *http.Request) (user string, ok bool) {
	if !c.Enabled() {
		return "", false
	}

	username, password, hasCredentials := r.BasicAuth()
	if !hasCredentials {
		return "", false
	}
	expectedPassword, exists := c.Users[username]
	if !exists {
		return "", false
	}
	expectedHash := sha256.Sum256([]byte(expectedPassword))
	actualHash := sha256.Sum256([]byte(password))
	if subtle.ConstantTimeCompare(expectedHash[:], actualHash[:]) != 1 {
		return "", false
	}
	return username, true
}

func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(userContextKey).(string)
	return user, ok
}

func (c Config) Require(next http.Handler) http.Handler {
	if !c.Enabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := c.Authenticate(r)
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="ScriptPortal"`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

func (c Config) RequireFunc(f http.HandlerFunc) http.HandlerFunc {
	return c.Require(f).ServeHTTP
}
