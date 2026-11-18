package solutions

// The examples use call from ex18_http_test.go, so copy that file too.

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func ExampleSecureHeaders() {
	h := SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	call(h, "GET", "http://example.com/", "")
	for _, target := range []string{"http://example.com/", "https://example.com/"} {
		rec := record(h, "GET", target)
		fmt.Println(target)
		for _, name := range []string{"X-Content-Type-Options", "Content-Security-Policy",
			"Referrer-Policy", "Cache-Control", "Strict-Transport-Security"} {
			fmt.Printf("  %s: %q\n", name, rec.Header().Get(name))
		}
	}
	// Output:
	// 200 ok
	// http://example.com/
	//   X-Content-Type-Options: "nosniff"
	//   Content-Security-Policy: "default-src 'none'; frame-ancestors 'none'"
	//   Referrer-Policy: "no-referrer"
	//   Cache-Control: "no-store"
	//   Strict-Transport-Security: ""
	// https://example.com/
	//   X-Content-Type-Options: "nosniff"
	//   Content-Security-Policy: "default-src 'none'; frame-ancestors 'none'"
	//   Referrer-Policy: "no-referrer"
	//   Cache-Control: "no-store"
	//   Strict-Transport-Security: "max-age=63072000; includeSubDomains"
}

func ExamplePasswordStore() {
	s := NewPasswordStore(bcrypt.MinCost) // the cheapest cost keeps the example fast
	fmt.Println(s.Register("ann", "correct horse", "admin"))
	fmt.Println(s.Register("bob", "short", "user"))
	fmt.Println(errors.Is(s.Register("bob", strings.Repeat("x", 73), "user"), bcrypt.ErrPasswordTooLong))
	fmt.Println(s.Register("ann", "another password", "user"))

	fmt.Println(s.Check("ann", "correct horse"))
	fmt.Println(s.Check("ann", "wrong horse"))
	fmt.Println(s.Check("eve", "correct horse"))

	hash, _ := s.Hash("ann")
	cost, _ := bcrypt.Cost(hash)
	fmt.Println(strings.Contains(string(hash), "correct horse"), len(hash), cost, s.Count())
	// Output:
	// <nil>
	// register bob: password too weak: at least 8 characters
	// true
	// register ann: user already exists
	// admin <nil>
	//  invalid credentials
	//  invalid credentials
	// false 60 4 1
}

func ExampleParseToken() {
	secret := []byte("example-secret-of-32-bytes-long!")
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	token, _ := IssueToken(secret, "ann", "admin", time.Hour, now)

	c, err := ParseToken(secret, token, now.Add(30*time.Minute))
	fmt.Println(c.Subject, c.Role, c.Issuer, err)

	_, err = ParseToken(secret, token, now.Add(2*time.Hour))
	fmt.Println(errors.Is(err, jwt.ErrTokenExpired))

	_, err = ParseToken([]byte("another-secret-of-32-bytes-long!"), token, now)
	fmt.Println(errors.Is(err, jwt.ErrTokenSignatureInvalid))

	// Change the role in the payload, keep the signature.
	parts := strings.Split(token, ".")
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	forged := strings.Replace(string(payload), `"admin"`, `"root"`, 1)
	parts[1] = base64.RawURLEncoding.EncodeToString([]byte(forged))
	_, err = ParseToken(secret, strings.Join(parts, "."), now)
	fmt.Println(errors.Is(err, jwt.ErrTokenSignatureInvalid))

	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, c).SignedString(jwt.UnsafeAllowNoneSignatureType)
	_, err = ParseToken(secret, none, now)
	fmt.Println(err != nil)
	// Output:
	// ann admin go-training <nil>
	// true
	// true
	// true
	// true
}

func ExampleRequireRole() {
	secret := []byte("example-secret-of-32-bytes-long!")
	whoami := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := ClaimsFrom(r.Context())
		fmt.Fprintf(w, "hello %s", c.Subject)
	})
	mux := http.NewServeMux()
	mux.Handle("GET /me", RequireRole(secret, "")(whoami))
	mux.Handle("GET /admin", RequireRole(secret, "admin")(whoami))

	user, _ := IssueToken(secret, "bob", "user", time.Hour, time.Now())
	admin, _ := IssueToken(secret, "ann", "admin", time.Hour, time.Now())
	call(mux, "GET", "/me", "")
	call(mux, "GET", "/me", "", "Authorization", "Bearer "+user)
	call(mux, "GET", "/admin", "", "Authorization", "Bearer "+user)
	call(mux, "GET", "/admin", "", "Authorization", "Bearer "+admin)
	call(mux, "GET", "/admin", "", "Authorization", admin) // no "Bearer "
	// Output:
	// 401 {"error":"unauthorized"}
	// 200 hello bob
	// 403 {"error":"forbidden"}
	// 200 hello ann
	// 401 {"error":"unauthorized"}
}
