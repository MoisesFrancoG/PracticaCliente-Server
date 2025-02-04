package main

import (
	"fmt"
	"practica/models"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	users = []models.User{}
	usersMutex sync.Mutex
	suscribers = make(chan chan models.User)
)

func createUser(c *gin.Context) {
	var newUser models.User

	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	usersMutex.Lock()
	users = append(users, newUser)
	usersMutex.Unlock()

	c.JSON(201, newUser)
}

func updateUser(c *gin.Context) {
	var updatedUser models.User

	if err := c.ShouldBindJSON(&updatedUser); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	usersMutex.Lock()
	defer usersMutex.Unlock()

	for i, user := range users {
		if user.ID == updatedUser.ID {
			users[i] = updatedUser
			c.JSON(200, updatedUser)
			return
		}
	}

	c.JSON(404, gin.H{"error": "Usuario no encontrado"})
}

func getUser(c *gin.Context) {
	usersMutex.Lock()
	defer usersMutex.Unlock()
	c.JSON(200, users)
}

func deleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "ID inválido"})
		return
	}

	usersMutex.Lock()
	defer usersMutex.Unlock()

	for i, user := range users {
		if user.ID == id {
			users = append(users[:i], users[i+1:]...)
			c.JSON(200, gin.H{"message": "Usuario eliminado"})
			return
		}
	}

	c.JSON(404, gin.H{"error": "Usuario no encontrado"})
}

func longPoll(c *gin.Context) {
	ch := make(chan models.User)
	suscribers <- ch

	for user := range ch {
		c.SSEvent("user", user)
		time.Sleep(1 * time.Second)
	}
}

func longPollHandler() {
    for clientChan := range suscribers {
        usersMutex.Lock()
        for _, user := range users {
            clientChan <- user
        }
        close(clientChan)
        usersMutex.Unlock()
    }
}

func checkChanges(c *gin.Context) {
	lastIDstr := c.Query("lastID")

	lastID, err := strconv.Atoi(lastIDstr)
	if err != nil {
		lastID = 0
	}
	fmt.Print(lastID)

	usersMutex.Lock()
	defer usersMutex.Unlock()

	var newUsers []models.User
	for _, user := range users {
		if user.ID > lastID {
			newUsers = append(newUsers, user)
		}
	}
	c.JSON(200, newUsers)

}

func main() {
    r := gin.Default()

    r.POST("/users", createUser)
	r.GET("/users", getUser)
	r.PUT("/users", updateUser)
	r.DELETE("/users/:id", deleteUser)

	go longPollHandler()
	r.GET("/longpoll", longPoll)
	r.GET("/check-changes", checkChanges)

	r.Run(":8080")
}