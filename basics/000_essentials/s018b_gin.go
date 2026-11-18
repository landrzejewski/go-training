package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ============================================================================================
// 018b - Gin, the popular alternative to net/http routing
// ============================================================================================
//
// What you build: the users API from 018 once more, this time with Gin
// (github.com/gin-gonic/gin), the most widely used Go web framework.
//
//	1. a first router    routes, path and query parameters, JSON answers
//	2. the users API     the same answers as in 018, with "binding" instead of hand-written checks
//	3. a binding trap    why binding:"required" rejects a legitimate 0
//	4. middleware        Gin's middleware, and route groups that share it
//
// The same jobs side by side:
//
//	                  net/http (Go 1.22+)                  Gin
//	route             mux.HandleFunc("GET /u/{id}", f)     r.GET("/u/:id", f)
//	handler           func(w http.ResponseWriter,          func(c *gin.Context)
//	                       r *http.Request)
//	path parameter    r.PathValue("id")                    c.Param("id")
//	query parameter   r.URL.Query().Get("q")               c.Query("q"), c.DefaultQuery("q", "x")
//	read + validate   json.Decoder + your own checks       c.ShouldBindJSON(&req) + binding tags
//	write JSON        your writeJSON helper                c.JSON(status, value)
//	middleware        func(http.Handler) http.Handler      func(c *gin.Context) with c.Next()
//	route groups      none (wrap a second mux)             r.Group("/admin", middleware...)
//	wrong method      405 + Allow                          404, unless HandleMethodNotAllowed
//
// When to pick which: net/http is enough for a small service and adds no dependency. Gin pays
// off when groups, binding and its middleware (CORS, sessions, rate limits ...) save real code.

// ginBasics runs the steps of section 018b.
func ginBasics() {
	fmt.Println("--- 018b Gin ---")
	ginFirstRouter() // 1. a first router
	ginUsersAPI()    // 2. the users API
	ginBindingTrap() // 3. a binding trap
	ginMiddleware()  // 4. middleware

	// Three more rules:
	//   - pass c.Request.Context() to the database, as r.Context() in net/http;
	//   - a *gin.Context is reused after the handler returns: give c.Copy() to a goroutine;
	//   - r.Run(":8080") is http.ListenAndServe, so it has no timeouts. Use newHTTPServer(r)
	//     from 018 instead, which also gives you the graceful Shutdown.

	// More: basics/018_http_services (net/http), examples/rest.go (a Gin CRUD API on PostgreSQL)
}

// --------------------------------------------------------------------------------------------
// Step 1 - a first router
// --------------------------------------------------------------------------------------------

// ginFirstRouter is step 1: one route with a path parameter and two query parameters.
func ginFirstRouter() {
	step("1. a first router: parameters in, JSON out")

	// gin.New() is an engine with no middleware at all. gin.Default() adds gin.Logger() and
	// gin.Recovery() - the same two jobs as logRequests and recoverPanics in 018. (These demos
	// use gin.New, because Gin's Logger prints the time.) Gin starts in a "debug mode" that
	// prints every route; services.go switches the whole program to gin.ReleaseMode.
	r := gin.New()
	r.GET("/hello/:name", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{ // gin.H is just map[string]any
			"hello": c.Param("name"),              // the :name part of the path
			"lang":  c.DefaultQuery("lang", "en"), // ?lang=..., or "en" when it is missing
			"page":  c.Query("page"),              // ?page=..., or "" when it is missing
		})
	})
	// Register every route BEFORE the first request: a router must not change while it serves.

	// *gin.Engine implements http.Handler, so httptest, middleware and http.Server all work
	// with it exactly as in 018.
	show("GET /hello/gopher?page=2 -> %s", callAPI(r, "GET", "/hello/gopher?page=2", ""))
}

// --------------------------------------------------------------------------------------------
// Step 2 - the users API
// --------------------------------------------------------------------------------------------

// ginUsersAPI is step 2: usersMux from 018, rebuilt with Gin. The answers are the same; only
// the validation messages differ, because they come from the binding tags.
func ginUsersAPI() {
	step("2. the users API on Gin: the same answers as in 018")
	api := usersGin(NewMemoryUserRepo())
	for _, c := range []struct{ method, path, body string }{
		{"POST", "/users", `{"name":"Ada","email":"ada@example.com"}`},
		{"POST", "/users", `{"name":"Ada","email":"ada@example.com"}`}, // the same email again
		{"POST", "/users", `{"name":"","email":"not-an-email"}`},       // two mistakes at once
		{"GET", "/users/1", ""},
		{"GET", "/users/42", ""},
		{"DELETE", "/users", ""}, // Gin's 405 body is plain text: "405 method not allowed"
	} {
		show("%-6s %-10s -> %s", c.method, c.path, callAPI(api, c.method, c.path, c.body))
	}
}

// createUserRequest is the body of POST /users. Gin checks it with the binding tags (they come
// from the go-playground/validator package): required = not the zero value, email = a valid
// address. A separate request type keeps these rules out of the domain type User.
type createUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

// usersGin is usersMux from 018 on Gin: :id instead of {id}, binding instead of validateUser,
// and the same statusFor (018, step 3) for the errors.
func usersGin(repo UserRepository) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.HandleMethodNotAllowed = true // answer a wrong method with 405, as ServeMux does

	r.GET("/users", func(c *gin.Context) {
		users, err := repo.List(c.Request.Context())
		if err != nil {
			ginError(c, err)
			return
		}
		c.JSON(http.StatusOK, users)
	})
	r.POST("/users", func(c *gin.Context) {
		var req createUserRequest
		// ShouldBindJSON returns the error and lets YOU answer. c.BindJSON would answer 400
		// on its own, in a format you do not control.
		if err := c.ShouldBindJSON(&req); err != nil {
			ginError(c, &InputError{Msg: bindingMessage(err)})
			return
		}
		u := User{Name: req.Name, Email: req.Email}
		if err := repo.Create(c.Request.Context(), &u); err != nil {
			ginError(c, err)
			return
		}
		c.JSON(http.StatusCreated, u)
	})
	r.GET("/users/:id", func(c *gin.Context) {
		id, err := parseID(c.Param("id"))
		if err != nil {
			ginError(c, err)
			return
		}
		u, err := repo.ByID(c.Request.Context(), id)
		if err != nil {
			ginError(c, err)
			return
		}
		c.JSON(http.StatusOK, u)
	})
	r.DELETE("/users/:id", func(c *gin.Context) {
		id, err := parseID(c.Param("id"))
		if err != nil {
			ginError(c, err)
			return
		}
		if err := repo.Delete(c.Request.Context(), id); err != nil {
			ginError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
	return r
}

// ginError answers with the same error JSON as writeError in 018. AbortWithStatusJSON also
// stops any handlers still waiting in the chain.
func ginError(c *gin.Context, err error) {
	status, msg := statusFor(err)
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}

// bindingMessage turns a binding error into a message for the client. The validator's own
// text names Go struct fields ("Key: 'createUserRequest.Name' Error:Field validation ..."),
// so it is better not to send it as it is.
func bindingMessage(err error) string {
	verrs, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return "invalid JSON: " + err.Error() // not a validation problem: the JSON itself is broken
	}
	parts := make([]string, len(verrs))
	for i, fe := range verrs {
		parts[i] = strings.ToLower(fe.Field()) + " fails '" + fe.Tag() + "'"
	}
	return strings.Join(parts, ", ")
}

// --------------------------------------------------------------------------------------------
// Step 3 - a binding trap
// --------------------------------------------------------------------------------------------

// ginBindingTrap is step 3. binding:"required" means "not the ZERO VALUE" (001), so a
// legitimate 0 is rejected. When 0 is a valid value, use a pointer: then only a missing field
// is an error.
func ginBindingTrap() {
	step(`3. a binding trap: binding:"required" rejects a legitimate 0`)
	type order struct {
		Quantity int  `json:"quantity" binding:"required"` // 0 fails, like a missing field
		Discount *int `json:"discount" binding:"required"` // a pointer: 0 is fine, missing is not
	}
	r := gin.New()
	r.POST("/orders", func(c *gin.Context) {
		var o order
		if err := c.ShouldBindJSON(&o); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": bindingMessage(err)})
			return
		}
		c.JSON(http.StatusCreated, o)
	})
	show("POST /orders quantity=0 -> %s", callAPI(r, "POST", "/orders", `{"quantity":0,"discount":0}`))
	show("POST /orders quantity=2 -> %s", callAPI(r, "POST", "/orders", `{"quantity":2,"discount":0}`))
}

// --------------------------------------------------------------------------------------------
// Step 4 - middleware
// --------------------------------------------------------------------------------------------

// ginMiddleware is step 4. Middleware in Gin is a gin.HandlerFunc: code before c.Next() runs on
// the way in, code after it on the way out, and c.Abort...() stops the chain, so the handler
// never runs. A GROUP gives routes a shared path prefix and shared middleware - something
// net/http has no equivalent for.
func ginMiddleware() {
	step("4. middleware: a group of routes behind a token check")
	r := gin.New()
	admin := r.Group("/admin", requireToken("s3cret")) // every route in the group needs the token
	admin.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user": c.GetString("user")}) // stored by the middleware
	})
	for _, token := range []string{"", "Bearer s3cret"} {
		req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
		if token != "" {
			req.Header.Set("Authorization", token)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		show("GET /admin/stats, Authorization=%-15q -> %d %s", token, rec.Code, rec.Body.String())
	}
}

// requireToken is Gin middleware that lets through only requests with the right bearer token.
// (018c does this properly, with signed tokens; here the point is the shape of a middleware.)
func requireToken(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") != "Bearer "+token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return // Abort stops the chain; return ends this function
		}
		c.Set("user", "admin") // a value for the handlers after this one
		c.Next()               // run the rest of the chain; code after this runs on the way out
	}
}
