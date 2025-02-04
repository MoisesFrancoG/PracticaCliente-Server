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

func getUser(c *gin.Context) {
	usersMutex.Lock()
	defer usersMutex.Unlock()
	c.JSON(200, users)
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

    go longPollHandler()
    r.GET("/longpoll", longPoll)

    r.GET("/check-changes", checkChanges)

    r.Run(":8080")
}