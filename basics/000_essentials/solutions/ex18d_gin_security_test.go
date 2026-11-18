package solutions

// The examples use call and recordBody from ex18_http_test.go and, for 18d.3, the code of
// 18c.2 and 18c.3 - copy those files too.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

// from sends GET path as if from the TCP peer remote, with a client-controlled
// X-Forwarded-For header, and prints "status body".
func from(h http.Handler, path, remote, forwardedFor string) {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = remote
	req.Header.Set("X-Forwarded-For", forwardedFor)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	fmt.Println(strings.TrimSpace(fmt.Sprint(rec.Code, " ", rec.Body.String(), " ", rec.Header().Get("Retry-After"))))
}

func ExampleNewIPEcho() {
	noProxy, _ := NewIPEcho(nil)
	from(noProxy, "/ip", "203.0.113.7:4000", "1.2.3.4") // the header is ignored

	behindProxy, _ := NewIPEcho([]string{"10.0.0.0/8"})
	from(behindProxy, "/ip", "10.0.0.5:4000", "1.2.3.4")    // from our proxy: believed
	from(behindProxy, "/ip", "203.0.113.7:4000", "1.2.3.4") // straight from the internet: ignored

	_, err := NewIPEcho([]string{"not-an-ip"})
	fmt.Println(err)
	// Output:
	// 200 {"ip":"203.0.113.7"}
	// 200 {"ip":"1.2.3.4"}
	// 200 {"ip":"203.0.113.7"}
	// trusted proxies: invalid IP address: not-an-ip
}

func ExampleRateLimit() {
	ok := func(c *gin.Context) { c.String(http.StatusOK, "ok") }

	safe := gin.New()
	_ = safe.SetTrustedProxies(nil)
	safe.Use(RateLimit(rate.Every(time.Minute), 2))
	safe.GET("/", ok)
	// The attacker changes X-Forwarded-For every time - and it does not help.
	for _, fake := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		from(safe, "/", "203.0.113.7:4000", fake)
	}
	from(safe, "/", "198.51.100.9:4000", "") // another client has its own limit

	// Gin's default trusts every proxy, so each fake address gets a fresh limit.
	unsafe := gin.New()
	unsafe.Use(RateLimit(rate.Every(time.Minute), 2))
	unsafe.GET("/", ok)
	for _, fake := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		from(unsafe, "/", "203.0.113.7:4000", fake)
	}
	// Output:
	// 200 ok
	// 200 ok
	// 429 {"error":"too many requests"} 60
	// 200 ok
	// 200 ok
	// 200 ok
	// 200 ok
}

func ExampleNewSecureRouter() {
	store := NewPasswordStore(bcrypt.MinCost)
	_ = store.Register("ann", "ann-password", "admin")
	_ = store.Register("bob", "bob-password", "user")
	r := NewSecureRouter(store, []byte("example-secret-of-32-bytes-long!"))

	call(r, "POST", "/login", `{"user":"bob","password":"guess"}`)
	call(r, "POST", "/login", `{"user":"bob"}`)
	call(r, "POST", "/login", `{"user":"`+strings.Repeat("b", 2000)+`","password":"x"}`)

	login := func(user, password string) string {
		rec := recordBody(r, "POST", "/login", `{"user":"`+user+`","password":"`+password+`"}`)
		var out struct{ Token string }
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return out.Token
	}
	bob, ann := login("bob", "bob-password"), login("ann", "ann-password")

	call(r, "GET", "/api/me", "")
	call(r, "GET", "/api/me", "", "Authorization", "Bearer "+bob)
	call(r, "GET", "/api/admin/stats", "", "Authorization", "Bearer "+bob)
	call(r, "GET", "/api/admin/stats", "", "Authorization", "Bearer "+ann)
	// Output:
	// 401 {"error":"invalid credentials"}
	// 400 {"error":"user and password are required"}
	// 413 {"error":"body too large"}
	// 401 {"error":"unauthorized"}
	// 200 {"role":"user","user":"bob"}
	// 403 {"error":"forbidden"}
	// 200 {"users":2}
}
