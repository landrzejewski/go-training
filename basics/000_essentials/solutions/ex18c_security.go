package solutions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// --- 18c.1 Security headers ---

// SecureHeaders sets the security headers on every response of next. Strict-Transport-Security
// is sent only over HTTPS (r.TLS != nil): browsers ignore it over plain HTTP.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		next.ServeHTTP(w, r) // the headers go first: after the handler writes, it is too late
	})
}

// --- 18c.2 A password store ---

// The errors of PasswordStore.
var (
	ErrUserExists         = errors.New("user already exists")
	ErrWeakPassword       = errors.New("password too weak")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type storedUser struct {
	hash []byte
	role string
}

// PasswordStore keeps users with bcrypt hashes of their passwords - never the passwords.
type PasswordStore struct {
	cost  int
	dummy []byte // compared for unknown users, so that they take as long as wrong passwords

	mu    sync.RWMutex
	users map[string]storedUser
}

// NewPasswordStore returns a store that hashes with the given bcrypt cost
// (bcrypt.DefaultCost in production, bcrypt.MinCost in tests).
func NewPasswordStore(cost int) *PasswordStore {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("no such user"), cost)
	return &PasswordStore{cost: cost, dummy: dummy, users: map[string]storedUser{}}
}

// Register stores a new user. A password shorter than 8 characters wraps ErrWeakPassword;
// one longer than 72 bytes wraps bcrypt.ErrPasswordTooLong; a used name wraps ErrUserExists.
func (s *PasswordStore) Register(user, password, role string) error {
	if len(password) < 8 {
		return fmt.Errorf("register %s: %w: at least 8 characters", user, ErrWeakPassword)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost) // the slow part: outside the lock
	if err != nil {
		return fmt.Errorf("register %s: %w", user, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[user]; ok {
		return fmt.Errorf("register %s: %w", user, ErrUserExists)
	}
	s.users[user] = storedUser{hash: hash, role: role}
	return nil
}

// Check returns the user's role if the password is right. An unknown user and a wrong password
// give the same ErrInvalidCredentials.
func (s *PasswordStore) Check(user, password string) (role string, err error) {
	s.mu.RLock()
	u, ok := s.users[user]
	s.mu.RUnlock()
	if !ok {
		_ = bcrypt.CompareHashAndPassword(s.dummy, []byte(password)) // same time as a real check
		return "", ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword(u.hash, []byte(password)) != nil {
		return "", ErrInvalidCredentials
	}
	return u.role, nil
}

// Hash returns the stored hash of a user, for tests and audits.
func (s *PasswordStore) Hash(user string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[user]
	return u.hash, ok
}

// Count returns the number of users.
func (s *PasswordStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}

// --- 18c.3 JWTs and a role middleware ---

// TokenIssuer is the "iss" claim of every token that IssueToken signs.
const TokenIssuer = "go-training"

// Claims are the claims of a token: a role plus the registered JWT claims (sub, iss, exp ...).
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// IssueToken signs a token with HS256 for subject and role, valid for ttl from now.
func IssueToken(secret []byte, subject, role string, ttl time.Duration, now time.Time) (string, error) {
	claims := Claims{Role: role, RegisteredClaims: jwt.RegisteredClaims{
		Subject:   subject,
		Issuer:    TokenIssuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// ParseToken verifies a token as of now and returns its claims. Only HS256 is accepted - never
// the algorithm the token names itself - and the issuer and an expiry are required.
func ParseToken(secret []byte, token string, now time.Time) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(TokenIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		return nil, err
	}
	return &claims, nil
}

type claimsKey struct{}

// ClaimsFrom returns the claims that RequireRole stored in the request's context.
func ClaimsFrom(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey{}).(*Claims)
	return c, ok
}

// RequireRole lets a request through only with a valid "Authorization: Bearer <token>" whose
// role is role; an empty role accepts any valid token. Without a valid token it answers
// 401 {"error":"unauthorized"} with a WWW-Authenticate header; with the wrong role,
// 403 {"error":"forbidden"}.
func RequireRole(secret []byte, role string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			claims, err := ParseToken(secret, token, time.Now())
			if !ok || err != nil {
				w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			if role != "" && claims.Role != role {
				writeJSONError(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims)))
		})
	}
}
