package main

// The tests for sections 018-020 (the s018_*.go ... s020_gorm.go files; services.go is their
// map). The HTTP tests need no network. The database tests need the PostgreSQL from
// docker-compose.yml and are SKIPPED without it.
//
//	go test -v -run 'TestUsersAPI|Repo' ./basics/000_essentials
//	docker-compose up -d && go test -v -run Repo ./basics/000_essentials

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

// init runs before the tests: Gin's test mode keeps its debug route listing out of the output.
func init() { gin.SetMode(gin.TestMode) }

// TestUsersAPI runs ONE table of requests against BOTH routers. The net/http and the Gin
// version of the API must keep the same contract - the same statuses and the same bodies on
// success. The steps share one repository, so they run in order.
func TestUsersAPI(t *testing.T) {
	routers := []struct {
		name string
		new  func(UserRepository) http.Handler
	}{
		{"net/http", usersMux},
		{"gin", func(repo UserRepository) http.Handler { return usersGin(repo) }},
	}
	steps := []struct {
		name, method, path, body string
		wantStatus               int
		wantJSON                 string // "" = check only the status
	}{
		{"create", "POST", "/users", `{"name":"Ada","email":"ada@example.com"}`, 201, `{"id":1,"name":"Ada","email":"ada@example.com"}`},
		{"duplicate", "POST", "/users", `{"name":"Ada","email":"ada@example.com"}`, 409, `{"error":"already exists"}`},
		{"missing name", "POST", "/users", `{"email":"x@example.com"}`, 400, ""},
		{"broken JSON", "POST", "/users", `{"name":`, 400, ""},
		{"get", "GET", "/users/1", "", 200, `{"id":1,"name":"Ada","email":"ada@example.com"}`},
		{"get missing", "GET", "/users/2", "", 404, `{"error":"not found"}`},
		{"id not a number", "GET", "/users/abc", "", 400, `{"error":"id must be a number"}`},
		{"list", "GET", "/users", "", 200, `[{"id":1,"name":"Ada","email":"ada@example.com"}]`},
		{"wrong method", "PATCH", "/users/1", "", 405, ""},
		{"delete", "DELETE", "/users/1", "", 204, ""},
		{"delete again", "DELETE", "/users/1", "", 404, `{"error":"not found"}`},
		{"empty list", "GET", "/users", "", 200, `[]`},
	}
	for _, router := range routers {
		t.Run(router.name, func(t *testing.T) {
			h := router.new(NewMemoryUserRepo())
			for _, s := range steps {
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, httptest.NewRequest(s.method, s.path, strings.NewReader(s.body)))
				if rec.Code != s.wantStatus {
					t.Errorf("%s: %s %s = %d, want %d (body %s)", s.name, s.method, s.path, rec.Code, s.wantStatus, rec.Body)
					continue
				}
				if s.wantJSON != "" {
					assertJSON(t, s.name, rec.Body.Bytes(), s.wantJSON)
				}
			}
		})
	}
}

// assertJSON compares DECODED values, so the field order and a trailing newline do not matter.
func assertJSON(t *testing.T, step string, got []byte, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Errorf("%s: body %q is not JSON: %v", step, got, err)
		return
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("%s: bad expectation %q: %v", step, want, err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("%s: body = %s, want %s", step, got, want)
	}
}

// TestRecoverPanics checks that a panic becomes a 500 that reveals nothing.
func TestRecoverPanics(t *testing.T) {
	h := recoverPanics(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("password=hunter2")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "hunter2") {
		t.Errorf("got %d %s, want a 500 without the panic message", rec.Code, rec.Body)
	}
}

// testRepositoryContract is the contract of UserRepository as a test. One helper run against
// every implementation is how you KNOW they are interchangeable.
func testRepositoryContract(t *testing.T, repo UserRepository) {
	t.Helper()
	ctx := t.Context()

	ada := User{Name: "Ada", Email: "ada@example.com"}
	if err := repo.Create(ctx, &ada); err != nil || ada.ID == 0 {
		t.Fatalf("Create = %v, id %d; want no error and an id", err, ada.ID)
	}
	if err := repo.Create(ctx, &User{Name: "Copy", Email: "ada@example.com"}); !errors.Is(err, ErrDuplicate) {
		t.Errorf("Create with a used email = %v, want ErrDuplicate", err)
	}
	if got, err := repo.ByID(ctx, ada.ID); err != nil || got != ada {
		t.Errorf("ByID(%d) = %+v, %v; want %+v", ada.ID, got, err, ada)
	}
	if _, err := repo.ByID(ctx, ada.ID+1000); !errors.Is(err, ErrNotFound) {
		t.Errorf("ByID of a missing id = %v, want ErrNotFound", err)
	}
	if all, err := repo.List(ctx); err != nil || len(all) != 1 {
		t.Errorf("List = %v, %v; want one user", all, err)
	}
	if err := repo.Delete(ctx, ada.ID); err != nil {
		t.Errorf("Delete = %v", err)
	}
	if err := repo.Delete(ctx, ada.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete = %v, want ErrNotFound", err)
	}
	if _, err := repo.ByID(ctx, ada.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("ByID after Delete = %v, want ErrNotFound", err)
	}
}

func TestMemoryUserRepo(t *testing.T) {
	testRepositoryContract(t, NewMemoryUserRepo())
}

func TestPostgresUserRepo(t *testing.T) {
	db := openTestDB(t)
	if err := resetUsersTable(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	testRepositoryContract(t, NewPostgresUserRepo(db))
}

func TestGormUserRepo(t *testing.T) {
	db, err := openGorm()
	if err != nil {
		t.Skipf("PostgreSQL is not reachable (docker-compose up -d): %v", oneLine(err))
	}
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { sqlDB.Close() })
	}
	if err := resetGormTables(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	testRepositoryContract(t, NewGormUserRepo(db))
}

// openTestDB connects to PostgreSQL, or SKIPS the test - a missing database is not a failure
// of the code under test. t.Cleanup closes the pool when the test ends.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := openPostgres(t.Context())
	if err != nil {
		t.Skipf("PostgreSQL is not reachable (docker-compose up -d): %v", oneLine(err))
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// --- 018c / 018d: security ---

// newTestAuth is the demos' AuthService with the cheapest bcrypt cost, so the tests stay fast
// (also under -race). Its clock stands still at 2026-01-01 12:00 UTC.
func newTestAuth(t *testing.T) *AuthService {
	t.Helper()
	auth := NewAuthService([]byte("test-secret-use-32-random-bytes!"), bcrypt.MinCost)
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	auth.now = func() time.Time { return now }
	for _, a := range [][3]string{{"ada@example.com", "ada-password-1", "admin"}, {"bob@example.com", "bob-password-1", "user"}} {
		if err := auth.Register(a[0], a[1], a[2]); err != nil {
			t.Fatal(err)
		}
	}
	return auth
}

// TestSecureAPI runs one table against secureMux AND secureGin: both must give the same
// statuses for the same credentials and tokens.
func TestSecureAPI(t *testing.T) {
	auth := newTestAuth(t)
	adminToken, _ := auth.IssueToken("ada@example.com", "admin")
	userToken, _ := auth.IssueToken("bob@example.com", "user")
	expired, _ := signToken(auth.secret, "bob@example.com", "user", time.Minute, auth.now().Add(-time.Hour))
	otherKey, _ := NewAuthService([]byte("another-secret-of-32-bytes-long!"), bcrypt.MinCost).IssueToken("bob@example.com", "admin")

	steps := []struct {
		name, method, path, body, token string
		wantStatus                      int
	}{
		{"login: wrong password", "POST", "/login", `{"email":"ada@example.com","password":"nope-nope"}`, "", 401},
		{"login: unknown email", "POST", "/login", `{"email":"eve@example.com","password":"nope-nope"}`, "", 401},
		{"login: no password", "POST", "/login", `{"email":"ada@example.com"}`, "", 400},
		{"login: broken JSON", "POST", "/login", `{"email":`, "", 400},
		{"login: ok", "POST", "/login", `{"email":"bob@example.com","password":"bob-password-1"}`, "", 200},
		{"me: no token", "GET", "/me", "", "", 401},
		{"me: user", "GET", "/me", "", userToken, 200},
		{"me: expired", "GET", "/me", "", expired, 401},
		{"me: signed with another key", "GET", "/me", "", otherKey, 401},
		{"me: truncated", "GET", "/me", "", userToken[:len(userToken)-3], 401},
		{"admin: user", "DELETE", "/admin/users/7", "", userToken, 403},
		{"admin: admin", "DELETE", "/admin/users/7", "", adminToken, 204},
	}
	routers := []struct {
		name string
		h    http.Handler
	}{{"net/http", secureMux(auth)}, {"gin", secureGin(auth)}}
	for _, router := range routers {
		t.Run(router.name, func(t *testing.T) {
			for _, s := range steps {
				req := httptest.NewRequest(s.method, s.path, strings.NewReader(s.body))
				if s.token != "" {
					req.Header.Set("Authorization", "Bearer "+s.token)
				}
				rec := httptest.NewRecorder()
				router.h.ServeHTTP(rec, req)
				if rec.Code != s.wantStatus {
					t.Errorf("%s: status %d, want %d (body %s)", s.name, rec.Code, s.wantStatus, rec.Body)
				}
				if rec.Code == http.StatusUnauthorized && s.path == "/me" && rec.Header().Get("WWW-Authenticate") == "" {
					t.Errorf("%s: a 401 without WWW-Authenticate", s.name)
				}
				if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
					t.Errorf("%s: no security headers", s.name)
				}
			}
		})
	}
}

// TestVerifyRejects checks every way a token must fail.
func TestVerifyRejects(t *testing.T) {
	auth := newTestAuth(t)
	good, _ := auth.IssueToken("bob@example.com", "user")
	claims, err := auth.Verify(good)
	if err != nil || claims.Subject != "bob@example.com" || claims.Role != "user" {
		t.Fatalf("Verify(good) = %+v, %v", claims, err)
	}
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	wrongIssuer, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{Role: "user", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: "someone-else", ExpiresAt: jwt.NewNumericDate(auth.now().Add(time.Hour))}}).SignedString(auth.secret)
	noExpiry, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{Role: "user", RegisteredClaims: jwt.RegisteredClaims{
		Issuer: tokenIssuer}}).SignedString(auth.secret)

	for _, tt := range []struct {
		name, token string
		want        error
	}{
		{"alg none", none, jwt.ErrTokenSignatureInvalid},
		{"wrong issuer", wrongIssuer, jwt.ErrTokenInvalidIssuer},
		{"no expiry", noExpiry, jwt.ErrTokenRequiredClaimMissing},
		{"garbage", "not.a.token", jwt.ErrTokenMalformed},
	} {
		if _, err := auth.Verify(tt.token); !errors.Is(err, tt.want) {
			t.Errorf("%s: Verify = %v, want %v", tt.name, err, tt.want)
		}
	}
	if _, err := parseToken(auth.secret, good, auth.now().Add(time.Hour)); !errors.Is(err, jwt.ErrTokenExpired) {
		t.Errorf("an hour later: Verify = %v, want ErrTokenExpired", err)
	}
}

// TestRegister checks the password rules and that only a hash is stored.
func TestRegister(t *testing.T) {
	auth := newTestAuth(t)
	if err := auth.Register("eve@example.com", "short", "user"); err == nil {
		t.Error("a 5-character password was accepted")
	}
	if err := auth.Register("eve@example.com", strings.Repeat("x", 73), "user"); !errors.Is(err, bcrypt.ErrPasswordTooLong) {
		t.Errorf("a 73-byte password: err = %v, want bcrypt.ErrPasswordTooLong", err)
	}
	if err := auth.Register("ada@example.com", "another-password", "user"); !errors.Is(err, ErrDuplicate) {
		t.Errorf("a used email: err = %v, want ErrDuplicate", err)
	}
	if hash := auth.accounts["ada@example.com"].passwordHash; strings.Contains(string(hash), "ada-password-1") {
		t.Error("the plain password is stored")
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := securityHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	for _, tt := range []struct {
		target   string
		wantHSTS bool
	}{{"http://example.com/", false}, {"https://example.com/", true}} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.target, nil))
		if got := rec.Header().Get("Strict-Transport-Security") != ""; got != tt.wantHSTS {
			t.Errorf("%s: HSTS sent = %t, want %t", tt.target, got, tt.wantHSTS)
		}
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s: nosniff missing", tt.target)
		}
	}
}

func TestRateLimit(t *testing.T) {
	h := rateLimit(rate.Every(time.Hour), 2)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	status := func(remote string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remote
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	got := []int{status("192.0.2.1:1"), status("192.0.2.1:2"), status("192.0.2.1:3"), status("198.51.100.9:1")}
	if want := []int{200, 200, 429, 200}; !slices.Equal(got, want) {
		t.Errorf("statuses = %v, want %v (another port is the same client, another IP is not)", got, want)
	}
}

func TestCORS(t *testing.T) {
	h := cors("https://app.example.com")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	for _, tt := range []struct{ origin, wantAllow string }{
		{"https://app.example.com", "https://app.example.com"},
		{"https://evil.example", ""},
	} {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", tt.origin)
		req.Header.Set("Access-Control-Request-Method", "DELETE")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllow || rec.Code != http.StatusNoContent {
			t.Errorf("preflight from %s: %d Allow-Origin=%q, want 204 %q", tt.origin, rec.Code, got, tt.wantAllow)
		}
	}
}

// TestGinTrustedProxies pins down the dangerous default and its fix.
func TestGinTrustedProxies(t *testing.T) {
	clientIP := func(r *gin.Engine) string {
		r.GET("/ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })
		req := httptest.NewRequest(http.MethodGet, "/ip", nil)
		req.RemoteAddr = "203.0.113.7:4000"
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Body.String()
	}
	if got := clientIP(gin.New()); got != "1.2.3.4" {
		t.Errorf("default engine: ClientIP = %s; Gin's default changed, update 018d", got)
	}
	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}
	if got := clientIP(r); got != "203.0.113.7" {
		t.Errorf("SetTrustedProxies(nil): ClientIP = %s, want the TCP peer 203.0.113.7", got)
	}
}
