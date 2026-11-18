package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ============================================================================================
// 018d - HTTP security with Gin
// ============================================================================================
//
// What you build: the protection of 018c in Gin. Everything in 018c still applies - Gin is a
// router on top of net/http - so this section is about what is DIFFERENT in Gin.
//
//	1. trusted proxies   Gin's dangerous default: whom to believe about the client's address
//	2. a secure API      018c's login and roles as Gin middleware on route groups
//	3. Basic auth        gin.BasicAuth for an internal endpoint
//	4. cookies and XSS   the trap in c.SetCookie; c.JSON escapes HTML, c.PureJSON does not
//	5. net/http + Gin    018c's middleware wraps a Gin engine unchanged; production settings

// ginSecurity runs the steps of section 018d.
func ginSecurity() {
	fmt.Println("--- 018d HTTP security with Gin ---")
	ginTrustedProxiesDemo() // 1. trusted proxies
	ginSecureAPIDemo()      // 2. a secure API
	ginBasicAuthDemo()      // 3. Basic auth
	ginCookiesAndXSSDemo()  // 4. cookies and XSS
	ginWithNetHTTPDemo()    // 5. net/http + Gin

	// More: github.com/gin-contrib (cors, secure, sessions) and Gin's documentation on
	// trusted proxies.
}

// --------------------------------------------------------------------------------------------
// Step 1 - trusted proxies
// --------------------------------------------------------------------------------------------

// ginTrustedProxiesDemo is step 1. Behind a PROXY (a load balancer, nginx), the TCP connection
// comes from the proxy, which writes the client's real address into the X-Forwarded-For
// header. But any client can write that header too. By default Gin trusts EVERY proxy, so
// c.ClientIP() returns whatever the client claims - and a rate limit, an IP allow-list or an
// audit log built on it is fooled by one header. Tell Gin which proxies are yours, or that
// there are none.
func ginTrustedProxiesDemo() {
	step("1. trusted proxies: which X-Forwarded-For header Gin believes")
	for _, c := range []struct {
		label   string
		trusted []string
		setup   bool   // false: Gin's default
		from    string // the TCP connection's address
	}{
		{"Gin's default (trusts everyone)", nil, false, "203.0.113.7:4000"},
		{"SetTrustedProxies(nil)", nil, true, "203.0.113.7:4000"},
		{"trusted 10.0.0.0/8, via the proxy", []string{"10.0.0.0/8"}, true, "10.0.0.5:4000"},
		{"trusted 10.0.0.0/8, direct", []string{"10.0.0.0/8"}, true, "203.0.113.7:4000"},
	} {
		r := gin.New()
		if c.setup {
			if err := r.SetTrustedProxies(c.trusted); err != nil {
				show("SetTrustedProxies: %v", err)
				continue
			}
		}
		r.GET("/ip", func(ctx *gin.Context) { ctx.String(http.StatusOK, ctx.ClientIP()) })
		req := httptest.NewRequest(http.MethodGet, "/ip", nil)
		req.RemoteAddr = c.from
		req.Header.Set("X-Forwarded-For", "1.2.3.4") // what the client CLAIMS to be
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		show("%-35s from %-16s -> ClientIP %s", c.label, c.from, rec.Body.String())
	}
	// Behind Cloudflare or Google App Engine: r.TrustedPlatform = gin.PlatformCloudflare.
}

// --------------------------------------------------------------------------------------------
// Step 2 - a secure API
// --------------------------------------------------------------------------------------------

// ginSecureAPIDemo is step 2: secureMux from 018c, rebuilt in Gin with the same AuthService and
// the same answers. The middleware sits on route GROUPS instead of on single routes:
//
//	POST   /login            public                    -> 200 {"token": "eyJ..."}
//	GET    /me               group "/": any valid token -> 200 {"role": ..., "user": ...}
//	DELETE /admin/users/:id  group "/admin": role admin -> 204, or 403 for anyone else
func ginSecureAPIDemo() {
	step("2. a secure API: 018c's login and roles (401 = not logged in, 403 = not allowed) on Gin groups")
	auth := newDemoAuth()
	r := secureGin(auth)
	show("POST /login, wrong password -> %s", callAPI(r, "POST", "/login", `{"email":"ada@example.com","password":"guess"}`))
	show("POST /login, not an email   -> %s", callAPI(r, "POST", "/login", `{"email":"ada","password":"guess"}`))
	show("POST /login, 2 KB body      -> %s", callAPI(r, "POST", "/login", `{"email":"`+strings.Repeat("a", 2048)+`"}`))

	// The tokens are made directly, exactly as Login makes them (018c, step 4).
	adminToken, _ := auth.IssueToken("ada@example.com", "admin")
	userToken, _ := auth.IssueToken("bob@example.com", "user")
	for _, c := range []struct{ who, method, path, token string }{
		{"no token", "GET", "/me", ""},
		{"Bob", "DELETE", "/admin/users/7", userToken},
		{"Ada (admin)", "DELETE", "/admin/users/7", adminToken},
	} {
		show("%-11s %-6s %-14s -> %s", c.who, c.method, c.path,
			callAPI(r, c.method, c.path, "", "Authorization", "Bearer "+c.token))
	}
}

// secureGin is the API of this step. Everything in the group "/" needs a valid token, and the
// group "/admin" inside it also needs the role "admin".
func secureGin(auth *AuthService) *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies(nil) // nothing runs in front of this server: ClientIP is the TCP peer
	r.HandleMethodNotAllowed = true
	r.Use(ginSecurityHeaders(), gin.Recovery())

	r.POST("/login", ginMaxBody(1<<10), func(c *gin.Context) { // a login body is tiny: 1 KB
		var in loginRequest
		if err := c.ShouldBindJSON(&in); err != nil {
			if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "body too large"})
				return
			}
			ginError(c, &InputError{Msg: bindingMessage(err)})
			return
		}
		token, err := auth.Login(in.Email, in.Password)
		if err != nil {
			ginError(c, err) // ErrInvalidCredentials -> 401, through statusFor (018)
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	authed := r.Group("/", ginRequireAuth(auth))
	authed.GET("/me", func(c *gin.Context) {
		claims := c.MustGet("claims").(*Claims)
		c.JSON(http.StatusOK, gin.H{"user": claims.Subject, "role": claims.Role})
	})
	admin := authed.Group("/admin", ginRequireRole("admin"))
	admin.DELETE("/users/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

// ginSecurityHeaders is securityHeaders from 018c as Gin middleware.
func ginSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		setSecurityHeaders(c.Writer.Header(), c.Request.TLS != nil)
		c.Next()
	}
}

// ginRequireAuth is requireAuth from 018c for Gin: the claims go into the gin.Context.
func ginRequireAuth(auth *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := bearerClaims(auth, c.GetHeader("Authorization"))
		if err != nil {
			c.Header("WWW-Authenticate", `Bearer realm="api"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}

// ginRequireRole is requireRole from 018c for Gin; it runs after ginRequireAuth.
func ginRequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, _ := c.Get("claims")
		if claims, ok := v.(*Claims); !ok || claims.Role != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

// ginMaxBody caps the request body. Gin has NO limit of its own: ShouldBindJSON reads whatever
// arrives. (MaxMultipartMemory only decides how much of an upload is kept in memory.)
func ginMaxBody(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		c.Next()
	}
}

// --------------------------------------------------------------------------------------------
// Step 3 - Basic auth
// --------------------------------------------------------------------------------------------

// ginBasicAuthDemo is step 3. gin.BasicAuth guards an internal endpoint (metrics, an admin
// tool) with a user name and a password. Use it over HTTPS only: Basic auth sends them
// base64-ENCODED, which is not encryption. Load the accounts from the configuration, never
// from the source code.
func ginBasicAuthDemo() {
	step("3. Basic auth: a user name and a password for an internal endpoint")
	r := gin.New()
	r.GET("/metrics", gin.BasicAuth(gin.Accounts{"ops": "metrics-pass"}), func(c *gin.Context) {
		c.String(http.StatusOK, "requests_total 42 (seen by %s)", c.MustGet(gin.AuthUserKey))
	})
	basic := "Basic " + base64.StdEncoding.EncodeToString([]byte("ops:metrics-pass"))
	show("GET /metrics, no password -> %s", callAPI(r, "GET", "/metrics", ""))
	show("GET /metrics, Basic auth  -> %s", callAPI(r, "GET", "/metrics", "", "Authorization", basic))
}

// --------------------------------------------------------------------------------------------
// Step 4 - cookies and XSS
// --------------------------------------------------------------------------------------------

// ginCookiesAndXSSDemo is step 4.
func ginCookiesAndXSSDemo() {
	step("4. cookies and XSS: the c.SetCookie trap, and escaped JSON")
	r := gin.New()

	// Cookies in Gin. c.SetCookie writes the same Set-Cookie header as http.SetCookie in 018c
	// (step 5 explains every attribute), with two differences: the attributes are positional
	// parameters, and Gin URL-escapes the value - so read it back with c.Cookie, which
	// unescapes it, not with c.Request.Cookie.
	//
	//	c.SetCookie(name,      value, maxAge, path, domain, secure, httpOnly)
	//	c.SetCookie("session", id,    3600,   "/",  "",     true,   true)
	//	//          Name       Value  Max-Age Path  (none)  Secure  HttpOnly
	//
	// The trap: nothing at the call site says which of the last two booleans is which. Gin's own
	// documentation shows c.SetCookie("gin_cookie", "test", 3600, "/", "localhost", false, true),
	// and copying it compiles - and silently drops Secure. SameSite is not a parameter at all:
	// without c.SetSameSite first, the cookie has no SameSite attribute. http.SetCookie(c.Writer,
	// &http.Cookie{...}) names every field, and it works in Gin too, because c.Writer is an
	// http.ResponseWriter.
	//
	// Reading and deleting:
	//
	//	id, err := c.Cookie("session")                      // err is http.ErrNoCookie when missing
	//	c.SetCookie("session", "", -1, "/", "", true, true) // maxAge -1: the browser drops it
	r.POST("/login-cookie", func(c *gin.Context) {
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("session", rand.Text(), 3600, "/", "", true, true)
		c.Status(http.StatusNoContent)
	})

	// XSS (cross-site scripting) means getting a <script> into a page someone else views.
	// c.JSON escapes <, > and & as \u003c..., so a JSON answer that a browser somehow shows as
	// HTML cannot run a script. c.PureJSON does not escape: use it only when you must. Gin's
	// HTML templates (r.LoadHTMLGlob) are html/template, which escapes by itself.
	r.GET("/echo", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"q": c.Query("q")}) })
	r.GET("/echo-pure", func(c *gin.Context) { c.PureJSON(http.StatusOK, gin.H{"q": c.Query("q")}) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/login-cookie", nil))
	_, attributes, _ := strings.Cut(rec.Header().Get("Set-Cookie"), ";")
	show("c.SetCookie: session=<26 random characters>;%s", attributes)
	show("  -> the same attributes as http.SetCookie in 018c, step 5")

	q := "?q=" + url.QueryEscape("<script>alert(1)</script>")
	show("c.JSON     -> %s", callAPI(r, "GET", "/echo"+q, ""))
	show("c.PureJSON -> %s", callAPI(r, "GET", "/echo-pure"+q, ""))
}

// --------------------------------------------------------------------------------------------
// Step 5 - net/http + Gin
// --------------------------------------------------------------------------------------------

// ginWithNetHTTPDemo is step 5. *gin.Engine is an http.Handler, so every net/http middleware
// from 018c - CrossOriginProtection, rateLimit, cors - wraps a whole Gin engine unchanged:
//
//	srv := newHTTPServer(http.NewCrossOriginProtection().Handler(r))
//
// The other way round, gin.WrapH(h) puts any net/http handler on a Gin route.
func ginWithNetHTTPDemo() {
	step("5. net/http + Gin: 018c's middleware around a Gin engine")
	r := gin.New()
	r.POST("/transfer", func(c *gin.Context) { c.String(http.StatusOK, "transferred") })
	h := rateLimit(rate.Every(time.Minute), 10)(http.NewCrossOriginProtection().Handler(r))
	show("cross-site POST  -> %s", callAPI(h, "POST", "/transfer", "", "Sec-Fetch-Site", "cross-site"))
	show("same-origin POST -> %s", callAPI(h, "POST", "/transfer", "", "Sec-Fetch-Site", "same-origin"))

	// TLS in Gin. r.RunTLS(":443", "cert.pem", "key.pem") exists, but like r.Run it is a server
	// without timeouts. Use the server from 018 instead, exactly as in 018c, step 7:
	//
	//	srv := newHTTPServer(r)
	//	srv.Addr = ":443"
	//	err := srv.ListenAndServeTLS("cert.pem", "key.pem")
	//
	// Behind a proxy that ends TLS (018c, step 7), the request reaches Gin as plain HTTP, so
	// c.Request.TLS is nil and ginSecurityHeaders never sends HSTS. The simple fix: behind such a
	// proxy, always send HSTS (or let the proxy send it) - browsers ignore it over plain HTTP
	// anyway, so sending it too often does no harm. The proxy also adds an
	// "X-Forwarded-Proto: https" header, but Gin ignores it, and SetTrustedProxies (step 1) only
	// affects ClientIP. Any client can send that header too: before you base a decision on it,
	// check yourself that c.RemoteIP() is your proxy.

	// Production settings, in one place:
	//   - gin.SetMode(gin.ReleaseMode), and gin.New() with your own logger and gin.Recovery();
	//   - r.SetTrustedProxies with your own proxies only (step 1);
	//   - security headers, body limits and rate limits (018c, steps 1 and 6);
	//   - CORS with gin-contrib/cors and an explicit list of origins;
	//   - the http.Server with timeouts from 018, behind TLS.
}
