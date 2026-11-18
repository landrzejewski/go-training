package solutions

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// --- 18b.1 Greetings with Gin ---

// NewGreetRouter is NewGreetMux (18.1) on Gin. HandleMethodNotAllowed makes a wrong method a
// 405, as with ServeMux - Gin's default is 404.
func NewGreetRouter() *gin.Engine {
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.GET("/hello/:name", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"greeting": greetingFor(c.Query("lang"), c.Param("name"))})
	})
	return r
}

// --- 18b.2 Binding and validation ---

// SignupRequest is the body of POST /signup. Age uses gte=18 rather than required: required
// rejects the zero value, and the rule here is about the value, not about its presence.
type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Age      int    `json:"age" binding:"gte=18"`
}

// NewSignupRouter serves POST /signup: 201 {"email": ...} for a valid body, 400 {"invalid":
// [...]} with the sorted names of the failing fields, or 400 {"error":"invalid JSON"}.
func NewSignupRouter() *gin.Engine {
	r := gin.New()
	r.POST("/signup", func(c *gin.Context) {
		var req SignupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			verrs, ok := errors.AsType[validator.ValidationErrors](err)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
				return
			}
			fields := make([]string, len(verrs))
			for i, fe := range verrs {
				fields[i] = strings.ToLower(fe.Field())
			}
			slices.Sort(fields)
			c.JSON(http.StatusBadRequest, gin.H{"invalid": fields})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"email": req.Email}) // never echo the password
	})
	return r
}

// --- 18b.3 Groups and middleware ---

// RequireAPIKeyGin is RequireAPIKey (18.3) as Gin middleware.
func RequireAPIKeyGin(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-API-Key") != key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

// NewTodoRouter serves the TodoStore from 18.2:
//
//	GET  /api/todos        public: the list
//	GET  /api/todos/:id    public: one todo, 400 / 404 as in 18.2
//	POST /api/admin/todos  needs the X-API-Key header: 201 and the new todo
func NewTodoRouter(store *TodoStore, apiKey string) *gin.Engine {
	r := gin.New()
	api := r.Group("/api")
	api.GET("/todos", func(c *gin.Context) {
		c.JSON(http.StatusOK, store.All())
	})
	api.GET("/todos/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		t, err := store.Get(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, t)
	})

	admin := api.Group("/admin", RequireAPIKeyGin(apiKey)) // the middleware covers the whole group
	admin.POST("/todos", func(c *gin.Context) {
		var in struct {
			Title string `json:"title" binding:"required"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
			return
		}
		c.JSON(http.StatusCreated, store.Add(strings.TrimSpace(in.Title)))
	})
	return r
}
