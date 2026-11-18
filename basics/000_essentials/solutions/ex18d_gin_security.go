package solutions

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// --- 18d.1 Trusted proxies ---

// NewIPEcho serves GET /ip with {"ip": c.ClientIP()}. Only the proxies in trusted (IP
// addresses or CIDR ranges) may name the client in X-Forwarded-For. With nil no proxy is
// trusted, so the IP is always the TCP peer - unlike Gin's default, which trusts everyone.
func NewIPEcho(trusted []string) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(trusted); err != nil {
		return nil, fmt.Errorf("trusted proxies: %w", err)
	}
	r.GET("/ip", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ip": c.ClientIP()})
	})
	return r, nil
}

// --- 18d.2 Rate limiting per client ---

// RateLimit allows every client - by c.ClientIP() - burst requests at once, refilled at limit
// requests per second. Over the limit it answers 429 {"error":"too many requests"} with a
// Retry-After header. It is only as good as ClientIP: see 18d.1.
func RateLimit(limit rate.Limit, burst int) gin.HandlerFunc {
	var (
		mu       sync.Mutex
		limiters = map[string]*rate.Limiter{} // real code evicts clients that went quiet
	)
	retryAfter := "1"
	if limit > 0 && limit < 1 {
		retryAfter = strconv.Itoa(int(math.Ceil(1 / float64(limit)))) // seconds per request
	}
	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		l, ok := limiters[ip]
		if !ok {
			l = rate.NewLimiter(limit, burst)
			limiters[ip] = l
		}
		mu.Unlock()
		if !l.Allow() {
			c.Header("Retry-After", retryAfter)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}

// --- 18d.3 A secure Gin API ---

// maxLoginBody is the largest body POST /login accepts.
const maxLoginBody = 1 << 10 // 1 KB

// NewSecureRouter serves an API on top of the PasswordStore (18c.2) and the tokens (18c.3):
//
//	POST /login            {"user", "password"} -> 200 {"token"}; 400 for a bad body,
//	                       401 for wrong credentials, 413 for a body over 1 KB
//	GET  /api/me           any valid token -> {"user", "role"}
//	GET  /api/admin/stats  the role admin -> {"users": the number of users}
//
// Tokens are valid for 15 minutes. No proxy is trusted.
func NewSecureRouter(store *PasswordStore, secret []byte) *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies(nil)

	r.POST("/login", limitBody(maxLoginBody), func(c *gin.Context) {
		var in struct {
			User     string `json:"user" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "body too large"})
				return
			}
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "user and password are required"})
			return
		}
		role, err := store.Check(in.User, in.Password)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		token, err := IssueToken(secret, in.User, role, 15*time.Minute, time.Now())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	api := r.Group("/api", requireTokenGin(secret))
	api.GET("/me", func(c *gin.Context) {
		claims := c.MustGet("claims").(*Claims)
		c.JSON(http.StatusOK, gin.H{"user": claims.Subject, "role": claims.Role})
	})
	admin := api.Group("/admin", requireRoleGin("admin"))
	admin.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": store.Count()})
	})
	return r
}

// limitBody caps the request body: Gin itself reads bodies of any size.
func limitBody(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		c.Next()
	}
}

// requireTokenGin puts the claims of a valid bearer token into the context, or answers 401.
func requireTokenGin(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		claims, err := ParseToken(secret, token, time.Now())
		if !ok || err != nil {
			c.Header("WWW-Authenticate", `Bearer realm="api"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}

// requireRoleGin runs after requireTokenGin and answers 403 for any other role.
func requireRoleGin(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, _ := c.Get("claims")
		if claims, ok := v.(*Claims); !ok || claims.Role != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}
