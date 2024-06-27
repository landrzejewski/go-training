package main

import (
	"database/sql"
	_ "github.com/lib/pq"

	"fmt"
	"net/http"
	"github.com/gin-gonic/gin"
)

var lastId = 1

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var database *sql.DB

func main() {
	db, err := sql.Open("postgres", "postgres://admin:admin@localhost/users?sslmode=disable")
	if err != nil {
		fmt.Println(err)
	}

	db.Query("create table if not exists users (id serial primary key, name varchar(100), email varchar(50))")
	database = db

	router := gin.Default()
	router.GET("/users", getUsers)
	// router.GET("/users/:id", getUserById)
	// router.POST("/users", createUser)
	// router.PUT("/users/:id", updateUser)
	// router.DELETE("/users/:id", deleteUser)
	router.Run(":8080")
}

func getUsers(c *gin.Context) {
	rows, err := database.Query("select * from users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}
	var users []User = make([]User, 0)
	for rows.Next() {
		var user User = User{}
		rows.Scan(&user.ID, &user.Name, &user.Email)
		users = append(users, user)
	}
	c.JSON(http.StatusOK, users)
}

// func getUserById(c *gin.Context) {
// 	 id, err := strconv.Atoi(c.Param("id"))
// 	 if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
// 		return
// 	 }
// 	 for _, user := range users {
// 		if user.ID == id {
// 			c.JSON(http.StatusOK, user)
// 			return
// 		}
// 	 }
// 	 c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
// }

// func createUser(c *gin.Context) {
// 	var user User
// 	if err := c.ShouldBindJSON(&user); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	lastId++
// 	user.ID = lastId
// 	users = append(users, user)
// 	c.JSON(http.StatusCreated, user)
// }

// func updateUser(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 	   c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
// 	   return
// 	}

// 	var updatedUser User
// 	if err := c.ShouldBindJSON(&updatedUser); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	for index, user := range users {
// 		if user.ID == id {
// 			updatedUser.ID = user.ID
// 			users[index] = updatedUser
// 			c.JSON(http.StatusOK, updatedUser)
// 			return
// 		}
// 	 }
// 	 c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
// }

// func deleteUser(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 	   c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
// 	   return
// 	}

// 	for index, user := range users {
// 		if user.ID == id {
// 			users = append(users[:index], users[index + 1:]...)
// 			c.JSON(http.StatusOK, gin.H{})
// 			return
// 		}
// 	 }
// 	 c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
// }