package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"practica/models"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
    replicatedUsers = make(map[int]models.User) 
    mutex           sync.Mutex
)

func longPollingReplication() {
    for {
        resp, err := http.Get("http://localhost:8080/longpoll")
        if err != nil {
            fmt.Println("Error en long polling:", err)
            time.Sleep(5 * time.Second)
            continue
        }

        decoder := json.NewDecoder(resp.Body)
        for decoder.More() {
            var user models.User
            if err := decoder.Decode(&user); err == nil {
                mutex.Lock()
                if _, exists := replicatedUsers[user.ID]; !exists {
                    replicatedUsers[user.ID] = user
                    fmt.Println("Usuario replicado:", user)
                }
                mutex.Unlock()
            }
        }
        resp.Body.Close()
        time.Sleep(5 * time.Second)
    }
}

func shortPollingReplication() {
    lastID := 0
    for {
        resp, err := http.Get(fmt.Sprintf("http://localhost:8080/check-changes?last_id=%d", lastID))
        if err != nil {
            fmt.Println("Error en short polling:", err)
            time.Sleep(5 * time.Second)
            continue
        }
        body, _ := ioutil.ReadAll(resp.Body)
        resp.Body.Close()

        var newUsers []models.User
        if err := json.Unmarshal(body, &newUsers); err == nil {
            mutex.Lock()
            for _, user := range newUsers {
                if _, exists := replicatedUsers[user.ID]; !exists {
                    replicatedUsers[user.ID] = user
                    fmt.Println("Nuevo usuario replicado:", user)
                    lastID = user.ID
                }
            }
            mutex.Unlock()
        }
        time.Sleep(5 * time.Second)
    }
}

func getReplicatedUsers(c *gin.Context) {
    mutex.Lock()
    defer mutex.Unlock()

    var usersList []models.User
    for _, user := range replicatedUsers {
        usersList = append(usersList, user)
    }
    c.JSON(http.StatusOK, usersList)
}

func main() {
    go longPollingReplication()
    go shortPollingReplication()

    r := gin.Default()
    r.GET("/replicated-users", getReplicatedUsers)

    fmt.Println("Servidor de replicación corriendo en :8081")
    r.Run(":8081")
}
