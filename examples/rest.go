package examples

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // registers the "postgres" driver
)

// User is the resource served by RestAPI.
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// defaultDatabaseURL matches the database started by docker-compose.yml. It is
// used only when the DATABASE_URL environment variable is unset.
const defaultDatabaseURL = "postgres://admin:admin@localhost/users?sslmode=disable"

// userHandler groups the HTTP handlers around the connection pool they use.
// Handing the *sql.DB over as a field, rather than through a package-level
// variable, keeps the handlers testable with any database.
type userHandler struct {
	db *sql.DB
}

// RestAPI serves a CRUD API for users on :8080, backed by PostgreSQL. The
// connection string comes from DATABASE_URL, falling back to the docker-compose
// database.
func RestAPI() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = defaultDatabaseURL
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		// Without stopping here the next line would dereference a nil *sql.DB.
		log.Fatalf("cannot open the database connection: %v", err)
	}
	defer db.Close()

	// sql.Open only validates the DSN - the first real connection is made lazily.
	// Ping fails fast when the database is down instead of failing on the first
	// request.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("cannot reach the database: %v", err)
	}

	// Run DDL/DML with Exec, not Query. Query returns a *sql.Rows that must be
	// closed - abandoning it kept a connection checked out of the pool until the
	// GC finalizer ran.
	if _, err := db.ExecContext(ctx, "create table if not exists users (id serial primary key, name varchar(100), email varchar(50))"); err != nil {
		log.Fatalf("cannot create the table: %v", err)
	}

	h := &userHandler{db: db}
	router := gin.Default()
	router.GET("/users", h.getUsers)
	router.GET("/users/:id", h.getUserByID)
	router.POST("/users", h.createUser)
	router.PUT("/users/:id", h.updateUser)
	router.DELETE("/users/:id", h.deleteUser)
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func (h *userHandler) getUsers(c *gin.Context) {
	// Columns listed explicitly: "select *" ties Scan to the physical column order,
	// so adding a column silently breaks the read. The request context cancels
	// the query when the client goes away.
	rows, err := h.db.QueryContext(c.Request.Context(), "select id, name, email from users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Without Close the connection returns to the pool only once the result set is
	// fully drained.
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		users = append(users, user)
	}
	// rows.Err() tells a normal end of the result set apart from a transport error
	// that happened midway.
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *userHandler) getUserByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	// QueryRow instead of Query: the previous version called rows.Next() once and
	// returned without Close, so every GET /users/:id leaked a connection. A missing
	// row is detected via sql.ErrNoRows, not via the "user.ID == 0" heuristic
	// (which would wrongly report 404 for a legitimate record with id 0).
	var user User
	err = h.db.QueryRowContext(c.Request.Context(), "select id, name, email from users where id = $1", id).
		Scan(&user.ID, &user.Name, &user.Email)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *userHandler) createUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.db.QueryRowContext(c.Request.Context(),
		"insert into users(name, email) values($1, $2) returning id", user.Name, user.Email).Scan(&user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *userHandler) updateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	var updatedUser User
	if err := c.ShouldBindJSON(&updatedUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.db.ExecContext(c.Request.Context(),
		"update users set name = $1, email = $2 where id = $3", updatedUser.Name, updatedUser.Email, id)
	if err != nil {
		// gin.H{"error": err} serialised the error interface, which has no exported
		// fields - the client received {"error":{}}. err.Error() is what is needed.
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Without checking RowsAffected, updating a non-existent id returned 200, so a
	// 404 was never produced. The error from RowsAffected is reported rather than
	// discarded: `err == nil && affected == 0` would silently treat a driver
	// failure as a successful update.
	affected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	updatedUser.ID = id
	c.JSON(http.StatusOK, updatedUser)
}

func (h *userHandler) deleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}
	result, err := h.db.ExecContext(c.Request.Context(), "delete from users where id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	// Without this the handler returned without writing a response, so gin sent
	// 200 OK with an empty body instead of 204 No Content.
	c.Status(http.StatusNoContent)
}
