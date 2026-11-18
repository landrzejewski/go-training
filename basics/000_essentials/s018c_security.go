package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

// ============================================================================================
// 018c - HTTP security with net/http
// ============================================================================================
//
// What you build: protection for an API, one layer at a time. Every layer assumes that the
// others can fail - "defence in depth".
//
//	1. security headers   tell the browser to be strict with every response
//	2. passwords, secrets keep a slow, salted hash (bcrypt) - never the password itself
//	3. tokens (JWT)       a signed note that says who the user is, and until when
//	4. login and roles    steps 2 and 3 together: who are you (401), what may you do (403)
//	5. browsers           cookies, cross-site request forgery (CSRF), CORS
//	6. abuse              rate limits, and secrets that never reach a log
//	7. TLS                the encrypted connection that everything above relies on
//
// Covered elsewhere: server timeouts and body limits (018), errors that reveal nothing (018),
// SQL placeholders against injection (019). Section 018d does the same in Gin.

// httpSecurity runs the steps of section 018c.
func httpSecurity() {
	fmt.Println("--- 018c HTTP security with net/http ---")
	headersDemo()   // 1. security headers
	passwordsDemo() // 2. passwords and other secrets
	tokensDemo()    // 3. tokens (JWT)
	loginDemo()     // 4. login and roles
	browserDemo()   // 5. browsers
	abuseDemo()     // 6. abuse
	tlsDemo()       // 7. TLS

	// More: the OWASP Top 10 (owasp.org/Top10) and the OWASP Cheat Sheet Series. In CI, run
	// govulncheck ./... (known vulnerabilities in the code you call) and gosec (risky code).
}

// --------------------------------------------------------------------------------------------
// Step 1 - security headers
// --------------------------------------------------------------------------------------------

// headersDemo is step 1: a few response headers make the browser stricter with your answers.
func headersDemo() {
	step("1. security headers: on every response, HSTS only over HTTPS")
	h := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"ok": "yes"})
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)) // https: r.TLS is set
	show("over HTTPS:")
	for _, h := range []struct{ name, meaning string }{
		{"X-Content-Type-Options", "trust the Content-Type, never guess"},
		{"Content-Security-Policy", "load nothing, never appear in another site's <iframe>"},
		{"Referrer-Policy", "do not tell other sites which URL linked to them"},
		{"Cache-Control", "keep private answers out of caches"},
		{"Strict-Transport-Security", "HSTS: use only HTTPS for this host"},
	} {
		show("  %s: %s", h.name, rec.Header().Get(h.name))
		show("      -> %s", h.meaning)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://api.example.com/", nil))
	show("over plain HTTP, Strict-Transport-Security is sent: %t", rec.Header().Get("Strict-Transport-Security") != "")
}

// setSecurityHeaders sets the headers. Both securityHeaders (here) and ginSecurityHeaders
// (018d) call it.
func setSecurityHeaders(h http.Header, overTLS bool) {
	h.Set("X-Content-Type-Options", "nosniff")                                     // trust Content-Type, never guess
	h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'") // load nothing, no <iframe>
	h.Set("Referrer-Policy", "no-referrer")                                        // URLs do not leak to other sites
	h.Set("Cache-Control", "no-store")                                             // private data stays out of caches
	if overTLS {
		// HSTS (HTTP Strict Transport Security): "use only HTTPS for this host, for two years".
		// Browsers ignore it over plain HTTP, so it is sent only over TLS.
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	}
}

// securityHeaders is the middleware (018, step 4) that sets the headers. They belong on EVERY
// response, errors included, so it goes outermost in the chain.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w.Header(), r.TLS != nil) // r.TLS is set when the request came over HTTPS
		next.ServeHTTP(w, r)
	})
}

// --------------------------------------------------------------------------------------------
// Step 2 - passwords
// --------------------------------------------------------------------------------------------

// passwordsDemo is step 2. Never store passwords, not even encrypted: store a HASH that is slow
// to compute and SALTED (mixed with random bytes, so equal passwords get different hashes).
// bcrypt does both, so stolen hashes cannot be cracked in bulk. A fast hash such as sha256 is
// exactly the wrong tool here. (argon2id and scrypt in golang.org/x/crypto are good, too.)
func passwordsDemo() {
	step("2. passwords and other secrets: a bcrypt hash, a safe compare, random values")

	// The COST sets how slow bcrypt is: DefaultCost is 10, and every +1 doubles the time.
	hash, err := bcrypt.GenerateFromPassword([]byte("correct horse"), bcrypt.DefaultCost)
	if err != nil {
		show("bcrypt: %v", err)
		return
	}
	show("bcrypt hash: %s... = version 2a, cost 10, then the salt and the hash (%d bytes, new every time)",
		hash[:7], len(hash))
	show("the right password matches: %t", bcrypt.CompareHashAndPassword(hash, []byte("correct horse")) == nil)
	show("a wrong password matches:   %t", bcrypt.CompareHashAndPassword(hash, []byte("correct house")) == nil)
	_, err = bcrypt.GenerateFromPassword([]byte(strings.Repeat("x", 73)), bcrypt.DefaultCost)
	show("a 73-byte password: %v", err) // bcrypt reads only 72 bytes, so it refuses longer ones

	// Comparing secrets (API keys, tokens) with == stops at the first byte that differs, so the
	// response time tells an attacker how much of a guess was right. ConstantTimeCompare always
	// takes the same time.
	apiKey := []byte("k3y-of-the-service")
	show("a wrong API key is accepted (subtle.ConstantTimeCompare): %t",
		subtle.ConstantTimeCompare(apiKey, []byte("k3y-guess")) == 1)

	// Secret values - session IDs, password reset links, API keys - come from crypto/rand, never
	// from math/rand, whose numbers can be predicted.
	show("rand.Text(): %d random characters (128 bits) - the session cookie of step 5 uses it", len(rand.Text()))
}

// --------------------------------------------------------------------------------------------
// Step 3 - tokens (JWT)
// --------------------------------------------------------------------------------------------

// tokensDemo is step 3. After a login the server hands out a TOKEN, and the client sends it
// with every request. A JWT (JSON Web Token) is three base64 parts: header.payload.signature.
// The payload holds the CLAIMS, and the signature, made with the server's secret key, proves
// that nobody changed them. The claims in the output: role (ours), iss (issuer: which server
// signed it), sub (subject: who the user is), iat (issued at) and exp (expires at); iat and
// exp are Unix seconds.
func tokensDemo() {
	step("3. tokens (JWT): signed, not encrypted")
	token, err := signToken(demoSecret, "bob@example.com", "user", 15*time.Minute, demoNow)
	if err != nil {
		show("sign: %v", err)
		return
	}
	parts := strings.Split(token, ".")
	header, _ := base64.RawURLEncoding.DecodeString(parts[0])
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	show("JWT header:  %s", header)
	show("JWT payload: %s", payload) // anyone can read this: never put a secret in a claim
	claims, err := parseToken(demoSecret, token, demoNow)
	show("parseToken -> sub=%s role=%s, issued %s, expires %s, err=%v", claims.Subject, claims.Role,
		claims.IssuedAt.UTC().Format("15:04"), claims.ExpiresAt.UTC().Format("15:04 MST"), err)

	// Bob edits his payload to make himself an admin. The signature no longer matches.
	forgedPayload := strings.Replace(string(payload), `"role":"user"`, `"role":"admin"`, 1)
	forged := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(forgedPayload)) + "." + parts[2]
	_, err = parseToken(demoSecret, forged, demoNow)
	show("forged role   -> %v", err)

	_, err = parseToken(demoSecret, token, demoNow.Add(time.Hour)) // the token lived 15 minutes
	show("an hour later -> %v", err)

	// "alg":"none" means "no signature at all". The library makes you write the word Unsafe to
	// create such a token, and parseToken refuses it.
	unsigned, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	_, err = parseToken(demoSecret, unsigned, demoNow)
	show("alg none      -> %v", err)

	// The server stores nothing about a token, so it cannot take one back before it expires.
	// Keep the lifetime short (minutes); for long logins, use server-side sessions instead.

	// HS256 signs and checks with one shared secret. When other services must check tokens
	// without being able to sign them, use a key pair instead (RS256 or EdDSA).
}

// demoSecret and demoNow are the signing key and the clock of the demos. A real secret is 32+
// random bytes from the configuration, never in git; a real clock is time.Now. A clock that
// stands still makes every run print the same tokens.
var (
	demoSecret = []byte("demo-secret-use-32-random-bytes!")
	demoNow    = time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
)

// Claims is what a token says about its holder: a role, plus the standard JWT claims (sub,
// iss, iat, exp - see tokensDemo) in jwt.RegisteredClaims.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// tokenIssuer is the "iss" claim of every token this program signs.
const tokenIssuer = "essentials"

// signToken signs the claims with HMAC-SHA256 ("HS256"), valid for ttl from now.
func signToken(secret []byte, subject, role string, ttl time.Duration, now time.Time) (string, error) {
	claims := Claims{Role: role, RegisteredClaims: jwt.RegisteredClaims{
		Subject:   subject,
		Issuer:    tokenIssuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// parseToken checks a token as of now - signature, algorithm, issuer, expiry - and returns its
// claims.
func parseToken(secret []byte, token string, now time.Time) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return secret, nil },
		// The token names its own algorithm ("alg") - accept only HS256, or a token with
		// "alg":"none" (no signature at all) gets through.
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		return nil, err
	}
	return &claims, nil
}

// --------------------------------------------------------------------------------------------
// Step 4 - login and roles
// --------------------------------------------------------------------------------------------

// loginDemo is step 4: a small API with a public login, a route for any logged-in user and an
// admin-only route. AUTHENTICATION answers "who are you?" - without a valid token, 401.
// AUTHORIZATION answers "may you do this?" - with a token but the wrong role, 403.
func loginDemo() {
	step("4. login and roles: 401 = not logged in, 403 = logged in but not allowed")
	auth := newDemoAuth()
	api := secureMux(auth)
	show("POST /login, wrong password -> %s", callAPI(api, "POST", "/login", `{"email":"ada@example.com","password":"guess"}`))
	show("POST /login, unknown email  -> %s", callAPI(api, "POST", "/login", `{"email":"eve@example.com","password":"guess"}`))
	login := callAPI(api, "POST", "/login", `{"email":"ada@example.com","password":"ada-password-1"}`)
	show("POST /login, right password -> %s...", login[:min(len(login), 22)])

	// The tokens for the calls below are made directly, exactly as Login makes them - so the
	// demo needs nobody's password again.
	adminToken, _ := auth.IssueToken("ada@example.com", "admin")
	userToken, _ := auth.IssueToken("bob@example.com", "user")
	for _, c := range []struct{ who, method, path, token string }{
		{"no token", "GET", "/me", ""},
		{"Bob", "GET", "/me", userToken},
		{"Bob", "DELETE", "/admin/users/7", userToken},
		{"Ada (admin)", "DELETE", "/admin/users/7", adminToken},
		{"a bad token", "GET", "/me", userToken[:len(userToken)-2]},
	} {
		show("%-11s %-6s %-14s -> %s", c.who, c.method, c.path,
			callAPI(api, c.method, c.path, "", "Authorization", "Bearer "+c.token))
	}
}

// ErrInvalidCredentials is the ONE error of a failed login. An unknown email and a wrong
// password look exactly the same, so nobody can find out which accounts exist.
var ErrInvalidCredentials = errors.New("invalid credentials")

// account is a user who can log in. Only the bcrypt hash of the password is kept.
type account struct {
	email        string
	passwordHash []byte
	role         string
}

// AuthService puts steps 2 and 3 together: it registers accounts with bcrypt hashes, checks
// passwords at login and hands out signed tokens. The net/http API below and the Gin API in
// 018d share it.
type AuthService struct {
	secret     []byte           // the signing key of the tokens
	bcryptCost int              // bcrypt.DefaultCost in production, bcrypt.MinCost in tests
	tokenTTL   time.Duration    // how long a token is valid: short, as step 3 explains
	now        func() time.Time // time.Now; the demos use a clock that stands still
	dummyHash  []byte           // compared for unknown emails, see Login

	mu       sync.RWMutex
	accounts map[string]account
}

func NewAuthService(secret []byte, bcryptCost int) *AuthService {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("no such account"), bcryptCost)
	return &AuthService{secret: secret, bcryptCost: bcryptCost, tokenTTL: 15 * time.Minute,
		now: time.Now, dummyHash: dummy, accounts: map[string]account{}}
}

// Register stores a new account with a bcrypt hash of the password.
func (a *AuthService) Register(email, password, role string) error {
	if len(password) < 8 {
		return &InputError{Msg: "the password needs at least 8 characters"}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), a.bcryptCost) // > 72 bytes: an error
	if err != nil {
		return fmt.Errorf("register %s: %w", email, err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.accounts[email]; ok {
		return fmt.Errorf("register %s: %w", email, ErrDuplicate)
	}
	a.accounts[email] = account{email: email, passwordHash: hash, role: role}
	return nil
}

// Login checks the password and returns a signed token, or ErrInvalidCredentials.
func (a *AuthService) Login(email, password string) (string, error) {
	a.mu.RLock()
	acc, ok := a.accounts[email]
	a.mu.RUnlock()
	if !ok {
		// Do the slow comparison anyway: if unknown emails were answered faster than wrong
		// passwords, the response time alone would tell which accounts exist.
		_ = bcrypt.CompareHashAndPassword(a.dummyHash, []byte(password))
		return "", ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword(acc.passwordHash, []byte(password)) != nil {
		return "", ErrInvalidCredentials
	}
	return a.IssueToken(acc.email, acc.role)
}

// IssueToken signs a token for subject (step 3).
func (a *AuthService) IssueToken(subject, role string) (string, error) {
	return signToken(a.secret, subject, role, a.tokenTTL, a.now())
}

// Verify checks a token (step 3) and returns its claims.
func (a *AuthService) Verify(token string) (*Claims, error) {
	return parseToken(a.secret, token, a.now())
}

// newDemoAuth is the AuthService of the demos: two accounts, and the clock of step 3.
func newDemoAuth() *AuthService {
	auth := NewAuthService(demoSecret, bcrypt.DefaultCost)
	auth.now = func() time.Time { return demoNow }
	_ = auth.Register("ada@example.com", "ada-password-1", "admin")
	_ = auth.Register("bob@example.com", "bob-password-1", "user")
	return auth
}

// loginRequest is the body of POST /login. The binding tags are for the Gin version (018d).
type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// secureMux is the API of this step: POST /login is public, GET /me needs a valid token, and
// DELETE /admin/users/{id} also needs the role "admin". chain (018, step 4) wraps each
// protected route in the middleware it needs, and securityHeaders (step 1) wraps the whole
// mux, so every answer carries the headers.
func secureMux(auth *AuthService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		var in loginRequest
		if err := decodeJSON(w, r, &in); err != nil {
			writeError(w, err)
			return
		}
		if in.Email == "" || in.Password == "" {
			writeError(w, &InputError{Msg: "email and password are required"})
			return
		}
		token, err := auth.Login(in.Email, in.Password)
		if err != nil {
			writeError(w, err) // ErrInvalidCredentials -> 401, through statusFor (018)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"token": token})
	})
	me := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := claimsFrom(r.Context()) // put there by requireAuth
		writeJSON(w, http.StatusOK, map[string]string{"user": c.Subject, "role": c.Role})
	})
	deleteUser := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent) // a real API would delete the user here
	})
	mux.Handle("GET /me", chain(me, requireAuth(auth)))
	mux.Handle("DELETE /admin/users/{id}", chain(deleteUser, requireAuth(auth), requireRole("admin")))
	return securityHeaders(mux)
}

// requireAuth lets a request through only with a valid "Authorization: Bearer <token>" header,
// and passes the token's claims on to the handler in the request's context.
func requireAuth(auth *AuthService) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := bearerClaims(auth, r.Header.Get("Authorization"))
			if err != nil {
				// The client learns nothing about WHY (expired? forged?); a real service logs err.
				w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims)))
		})
	}
}

// requireRole runs after requireAuth and lets through only the given role.
func requireRole(role string) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c := claimsFrom(r.Context()); c == nil || c.Role != role {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// bearerClaims reads "Bearer <token>" from an Authorization header and verifies the token.
func bearerClaims(auth *AuthService, header string) (*Claims, error) {
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || token == "" {
		return nil, errors.New("no bearer token")
	}
	return auth.Verify(token)
}

// claimsKey is the context key under which requireAuth stores the claims. An unexported type
// cannot collide with a key from another package (011).
type claimsKey struct{}

// claimsFrom returns the claims that requireAuth stored, or nil.
func claimsFrom(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey{}).(*Claims)
	return c
}

// --------------------------------------------------------------------------------------------
// Step 5 - browsers
// --------------------------------------------------------------------------------------------

// browserDemo is step 5. Browsers need extra care, because they attach cookies BY THEMSELVES -
// even to a request that a page on evil.example started.
func browserDemo() {
	step("5. browsers: safe cookies, CSRF protection, CORS")

	// COOKIES. A cookie is a small name=value pair that the server asks the browser to keep, in
	// a Set-Cookie response header. From then on the browser sends it back BY ITSELF, in a
	// Cookie request header, with every request to the same host. That makes cookies the
	// classic way to keep a user logged in - and it is also why CSRF (below) exists.
	//
	// A login session with a cookie, request by request:
	//
	//	browser                                     server
	//	POST /login  (email, password)      ----->  checks the password, creates a session ID
	//	                                    <-----  Set-Cookie: session=K7Q...; HttpOnly; Secure; SameSite=Lax
	//	GET /me      Cookie: session=K7Q... ----->  looks the session ID up: it is Bob
	//	POST /logout Cookie: session=K7Q... ----->  deletes the session on the server
	//	                                    <-----  Set-Cookie: session=; Max-Age=0  (the browser drops it)
	//
	// The VALUE is a random session ID (rand.Text, step 2) that points to data kept on the
	// server - in a map, a database or Redis. Never put the user's ID or role into a plain
	// cookie: users can edit their cookies. (A signed token from step 3 would also do.)
	//
	// On the server, reading and deleting look like this:
	//
	//	c, err := r.Cookie("session") // err is http.ErrNoCookie when the browser sent none
	//	user, ok := sessions[c.Value] // never trust the value itself: look it up
	//	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1}) // log out
	//
	// Deleting means sending the same name and the same Path again, with MaxAge -1 (it goes out
	// as Max-Age=0): the browser then drops the cookie at once.
	//
	// After the value come the ATTRIBUTES: instructions for the browser. The output below
	// explains the ones set here. SameSite has three settings:
	//   - Strict: never sent with a request that another site started - not even when the user
	//     follows a link to you, so they arrive looking logged out;
	//   - Lax: sent when the user follows a link to you, but not with POSTs, iframes or
	//     fetch calls from other sites - the usual choice;
	//   - None: always sent; allowed only together with Secure, for widgets embedded elsewhere.
	// Always set SameSite yourself: only Chrome-based browsers treat a cookie without it as Lax.
	//
	// Also worth knowing:
	//   - Domain is left out on purpose: then only this exact host gets the cookie, while
	//     Domain=example.com would hand it to every subdomain as well;
	//   - browsers accept Secure cookies on http://localhost, so development works without TLS;
	//   - a name that starts with "__Host-" makes the browser insist on Secure, Path=/ and no
	//     Domain, so no subdomain and no plain-HTTP page can plant or overwrite the cookie.
	//     A real application would call this one "__Host-session";
	//   - a cookie holds about 4 KB at most.
	//
	// Cookie or Authorization header? A browser sends ANY cookie by itself - convenient, but it
	// needs the CSRF protection below; HttpOnly only hides the cookie from scripts. A token in
	// the Authorization header must be added by JavaScript, and a token kept in localStorage
	// can be read by any XSS script (a script someone injected into your page). The usual rule:
	// browsers get an HttpOnly cookie plus CSRF protection; mobile apps and other services send
	// "Authorization: Bearer <token>" (step 4).
	setSession := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name: "session", Value: rand.Text(), Path: "/", MaxAge: 3600,
			HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
		})
		w.WriteHeader(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	setSession.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/login", nil))
	_, attributes, _ := strings.Cut(rec.Header().Get("Set-Cookie"), ";")
	show("Set-Cookie: session=<26 random characters>;%s", attributes)
	meaning := map[string]string{
		"Path=/":       "sent with requests to every path of this site",
		"Max-Age=3600": "the browser forgets it after an hour (without Max-Age: when the browser closes)",
		"HttpOnly":     "page scripts (document.cookie) cannot read it, so an XSS script cannot copy it",
		"Secure":       "sent only over HTTPS (step 7), never as readable text",
		"SameSite=Lax": "not sent with POSTs from other sites - the first defence against CSRF",
	}
	for _, attr := range strings.Split(strings.TrimSpace(attributes), "; ") { // as the browser got them
		show("  %-12s -> %s", attr, meaning[attr])
	}

	// CSRF (cross-site request forgery): a page on evil.example posts a form to your site, and
	// the browser adds your cookie to it - so the request runs as you.
	//
	// SameSite=Lax above is one defence, and only for cookies. http.CrossOriginProtection
	// (Go 1.25) is a second one: it reads the Sec-Fetch-Site and Origin headers that browsers
	// add, and rejects a POST, PUT, PATCH or DELETE that comes from another site.
	//
	// GET, HEAD and OPTIONS always pass, so they must never change anything. A token in the
	// Authorization header needs none of this: browsers never add it on their own.
	cop := http.NewCrossOriginProtection()
	_ = cop.AddTrustedOrigin("https://admin.example.com") // another site of your own
	transfer := cop.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "transferred")
	}))
	for _, c := range []struct {
		label, method string
		header        []string
		note          string
	}{
		{"same-origin POST", "POST", []string{"Sec-Fetch-Site", "same-origin"}, ""},
		{"cross-site POST", "POST", []string{"Sec-Fetch-Site", "cross-site"}, "  <- the forged request"},
		{"cross-site GET", "GET", []string{"Sec-Fetch-Site", "cross-site"}, "  (allowed: a GET changes nothing)"},
		{"POST from a trusted origin", "POST", []string{"Sec-Fetch-Site", "cross-site", "Origin", "https://admin.example.com"}, ""},
		{"POST without browser headers", "POST", nil, "  (curl: not a browser, so no victim's cookie)"},
	} {
		show("CSRF: %-28s -> %s%s", c.label, callAPI(transfer, c.method, "/transfer", "", c.header...), c.note)
	}

	// CORS (cross-origin resource sharing): a page from another site may SEND requests to your
	// API, but the browser does not let it READ the answers. CORS headers relax that rule for
	// the sites you list. They protect nothing on the server - curl ignores them - and
	// Access-Control-Allow-Origin: * must never be combined with cookies.
	api := cors("https://app.example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"ok": "yes"})
	}))
	// Before a DELETE (or a JSON body, or an Authorization header) the browser first asks for
	// permission with an OPTIONS request: the PREFLIGHT. Its status does not matter - only the
	// Access-Control-Allow-* headers do.
	for _, c := range []struct{ origin, note string }{
		{"https://app.example.com", "the browser may send the DELETE"},
		{"https://evil.example", "no Allow-Origin: the browser blocks the DELETE"},
	} {
		origin := c.origin
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/users/1", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "DELETE")
		api.ServeHTTP(rec, req)
		show("CORS preflight from %-24s -> %d Allow-Origin=%q Allow-Methods=%q", origin, rec.Code,
			rec.Header().Get("Access-Control-Allow-Origin"), rec.Header().Get("Access-Control-Allow-Methods"))
		show("  -> %s", c.note)
	}
}

// cors is a middleware that lets pages from the listed origins call the API from a browser.
func cors(allowed ...string) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Vary", "Origin") // the answer depends on Origin: caches must know that
			origin := r.Header.Get("Origin")
			ok := origin != "" && slices.Contains(allowed, origin)
			if ok {
				w.Header().Set("Access-Control-Allow-Origin", origin) // the one origin, never "*"
			}
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				if ok {
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
					w.Header().Set("Access-Control-Max-Age", "600") // the browser may remember it 10 min
				}
				w.WriteHeader(http.StatusNoContent) // a preflight never reaches the handler
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// --------------------------------------------------------------------------------------------
// Step 6 - abuse
// --------------------------------------------------------------------------------------------

// abuseDemo is step 6. A RATE LIMIT caps how many requests one client may send: it protects a
// login from password guessing and the whole service from overload.
func abuseDemo() {
	step("6. abuse: a rate limit per client, and secrets kept out of logs")

	// Here: 3 requests at once, then one more per minute.
	h := rateLimit(rate.Every(time.Minute), 3)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	codes := make([]string, 0, 5)
	var last *httptest.ResponseRecorder
	for range 5 {
		last = httptest.NewRecorder()
		h.ServeHTTP(last, httptest.NewRequest(http.MethodPost, "/login", nil)) // httptest sets RemoteAddr 192.0.2.1:1234
		codes = append(codes, strconv.Itoa(last.Code))
	}
	other := httptest.NewRequest(http.MethodPost, "/login", nil)
	other.RemoteAddr = "198.51.100.9:5000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, other)
	show("rate limit, burst 3: %s (Retry-After: %s); another client: %d",
		strings.Join(codes, " "), last.Header().Get("Retry-After"), rec.Code)
	// Often a proxy or an API gateway in front of the service rate limits too: defence in depth.

	// Secrets must not leak into logs. A type with a LogValue method (slog.LogValuer) decides
	// how slog shows it, and a String method (fmt.Stringer) covers fmt. Never log tokens,
	// passwords or full card numbers.
	var logged strings.Builder
	quietLogger(&logged).Info("login attempt", "user", "ada@example.com", "password", redacted("ada-password-1"))
	show("slog:   %s", strings.TrimSpace(logged.String()))
	show("fmt %%v: %v", redacted("ada-password-1"))
}

// rateLimit is a middleware that gives every client (by IP address) a bucket of permits from
// golang.org/x/time/rate. The bucket holds burst permits, each request takes one, and limit
// permits per second flow back in; an empty bucket means 429 Too Many Requests. (The library
// calls the permits "tokens" - they have nothing to do with the JWTs of step 3.)
func rateLimit(limit rate.Limit, burst int) middleware {
	var (
		mu       sync.Mutex
		limiters = map[string]*rate.Limiter{} // grows forever: real code evicts quiet clients
	)
	retryAfter := strconv.Itoa(int(math.Ceil(1 / float64(limit)))) // seconds until the next permit
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// RemoteAddr is the address of the TCP connection: it cannot be faked. Behind a
			// proxy it is the proxy's address, and the client's is in X-Forwarded-For - which
			// you may trust only from your own proxy (018d shows how that goes wrong in Gin).
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			mu.Lock()
			l, ok := limiters[ip]
			if !ok {
				l = rate.NewLimiter(limit, burst)
				limiters[ip] = l
			}
			mu.Unlock()
			if !l.Allow() {
				w.Header().Set("Retry-After", retryAfter)
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// redacted is a string that never shows up in logs or in fmt output.
type redacted string

func (redacted) LogValue() slog.Value { return slog.StringValue("REDACTED") }
func (redacted) String() string       { return "REDACTED" }

// quietLogger writes text logs to w without the time, so the output stays the same.
func quietLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
}

// --------------------------------------------------------------------------------------------
// Step 7 - TLS
// --------------------------------------------------------------------------------------------

// tlsDemo is step 7. TLS (Transport Layer Security) does two jobs for every connection:
//   - ENCRYPTION: nobody on the way - a cafe's Wi-Fi, an internet provider, a proxy - can
//     read or change the traffic;
//   - AUTHENTICATION: the client checks that it really talks to api.example.com, and not to
//     somebody pretending to be it.
//
// HTTPS is plain HTTP inside a TLS connection, on port 443. Without it every password, token
// and cookie above crosses the network as readable text. That is why Secure cookies (step 5)
// and HSTS (step 1) insist on HTTPS.
func tlsDemo() {
	step("7. TLS: the encrypted connection everything else relies on")

	// The server proves who it is with a CERTIFICATE: a small file that says "this public key
	// belongs to api.example.com, valid from ... until ...". It is signed by a CERTIFICATE
	// AUTHORITY (CA), such as Let's Encrypt - an organisation whose own certificate every
	// operating system and browser already trusts. Next to it the server keeps the matching
	// PRIVATE KEY, which never leaves the server: cert.pem is public, key.pem is secret.
	//
	// The HANDSHAKE, simplified. It happens once per connection, before any HTTP:
	//
	//	client                                        server
	//	"hello, I speak TLS 1.2 and 1.3"    ------>
	//	                                    <------   "TLS 1.3, and here is my certificate"
	//	the client checks the certificate:
	//	  1. is it signed by a CA I trust?
	//	  2. is it issued for the host I dialled?
	//	  3. is it still valid today?
	//	                                    <------   a signature made with key.pem
	//	  4. does it match the public key in the certificate?
	//	both sides agree on fresh keys for this connection
	//	GET /data   (encrypted)             ------>
	//	                                    <------   200 OK   (encrypted)
	//
	// If any check fails, the client hangs up before sending a single byte of HTTP. Check 4 is
	// why a copied certificate is useless: only the holder of key.pem can make that signature.

	// The demo: httptest starts a server with a built-in TEST certificate. It is SELF-SIGNED -
	// signed by itself instead of by a CA - so normally nobody trusts it. srv.Client() returns a
	// client that has been told to trust exactly this certificate, the way a browser trusts a CA.
	srv := httptest.NewUnstartedServer(securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "secret data")
	})))
	srv.Config.ErrorLog = log.New(io.Discard, "", 0) // the refused handshake below would be logged
	srv.StartTLS()
	defer srv.Close()
	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		show("TLS request failed: %v", err)
		return
	}
	defer resp.Body.Close()
	show("connected with %s; HSTS sent: %t", tls.VersionName(resp.TLS.Version),
		resp.Header.Get("Strict-Transport-Security") != "")

	// resp.TLS describes the connection, including the certificate the server presented - the
	// one the client just checked.
	cert := resp.TLS.PeerCertificates[0]
	show("the certificate the client checked:")
	show("  issued for   %v and %v   (check 2: the client dialled 127.0.0.1)", cert.DNSNames, cert.IPAddresses)
	show("  issued to    %s, signed by %s - the same name: self-signed   (check 1)",
		strings.Join(cert.Subject.Organization, ", "), strings.Join(cert.Issuer.Organization, ", "))
	show("  valid until  %s   (check 3; a test certificate - real ones last months)", cert.NotAfter.Format("2006-01-02"))

	// A normal client, with only the usual trusted CAs, refuses the test certificate. That is
	// TLS working: it refuses an attacker's self-made certificate in exactly the same way.
	_, err = (&http.Client{Timeout: 5 * time.Second}).Get(srv.URL)
	if verr, ok := errors.AsType[*tls.CertificateVerificationError](err); ok {
		show("a normal client refuses it: %v", verr.Err)
	}

	// In production it is the server from 018 plus a certificate:
	//
	//	srv := newHTTPServer(handler)                             // 018, step 5: with timeouts
	//	srv.Addr = ":443"
	//	srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12} // TLS 1.0 and 1.1 are deprecated
	//	                                                          // (RFC 8996); 1.2 is Go's default,
	//	                                                          // this writes the minimum down
	//	err := srv.ListenAndServeTLS("cert.pem", "key.pem")
	//
	// Where the certificate comes from:
	//   - Let's Encrypt gives certificates for free, valid for 90 days, and
	//     golang.org/x/crypto/acme/autocert fetches and renews them by itself:
	//	m := &autocert.Manager{Prompt: autocert.AcceptTOS,
	//		HostPolicy: autocert.HostWhitelist("api.example.com"), Cache: autocert.DirCache("certs")}
	//	srv.TLSConfig = m.TLSConfig()
	//	err := srv.ListenAndServeTLS("", "") // no files: the certificate comes from m
	//     Let's Encrypt must be able to reach the server from the internet on port 443.
	//   - Often TLS ends at a load balancer or a Kubernetes ingress in front of the service
	//     ("TLS termination"), and the service itself speaks plain HTTP inside the network.
	//     018d (steps 1 and 5) shows what that means for the code.
	//   - On your own machine, mkcert makes a certificate your browser trusts, and
	//     go run $(go env GOROOT)/src/crypto/tls/generate_cert.go --host localhost
	//     makes a self-signed one.
	//
	// Plain http:// should only redirect to https://, and HSTS (step 1) then makes the browser
	// skip http:// altogether next time:
	//
	//	redirect := newHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//		http.Redirect(w, r, "https://"+r.Host+r.URL.RequestURI(), http.StatusMovedPermanently)
	//	}))
	//	redirect.Addr = ":80"
	//	go redirect.ListenAndServe() // a second server - with timeouts too
	//
	// Never "fix" a certificate error with tls.Config{InsecureSkipVerify: true}. It switches
	// off the authentication half of TLS, so anybody in between is accepted. In tests, use
	// srv.Client(), or add the test certificate to tls.Config.RootCAs.
	//
	// mTLS (mutual TLS) adds the other direction: the CLIENT shows a certificate too. It is
	// common between the services of one company.
}
