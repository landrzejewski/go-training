package examples

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"net/http"
	"strconv"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var database *sql.DB

func RestApi() {
	db, err := sql.Open("postgres", "postgres://admin:admin@localhost/users?sslmode=disable")
	if err != nil {
		fmt.Println(err)
	}

	db.Query("create table if not exists users (id serial primary key, name varchar(100), email varchar(50))")
	database = db

	router := gin.Default()
	router.GET("/users", getUsers)
	router.GET("/users/:id", getUserById)
	router.POST("/users", createUser)
	router.PUT("/users/:id", updateUser)
	router.DELETE("/users/:id", deleteUser)
	router.Run(":8080")
}

func getUsers(c *gin.Context) {
	rows, err := database.Query("select * from users")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var users = make([]User, 0)
	for rows.Next() {
		var user User = User{}
		rows.Scan(&user.ID, &user.Name, &user.Email)
		users = append(users, user)
	}
	c.JSON(http.StatusOK, users)
}

func getUserById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	rows, err := database.Query("select * from users where id = $1", id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rows.Next()
	var user = User{}
	rows.Scan(&user.ID, &user.Name, &user.Email)
	if user.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
	} else {
		c.JSON(http.StatusOK, user)
	}
}

func createUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := database.QueryRow("insert into users(name, email) values($1, $2) returning id", &user.Name, &user.Email).Scan(&user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func updateUser(c *gin.Context) {
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

	_, err = database.Exec("update users set name = $1, email = $2 where id = $3", updatedUser.Name, updatedUser.Email, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func deleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}
	_, err = database.Exec("delete from users where id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
}
