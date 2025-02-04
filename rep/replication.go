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
				replicatedUsers[user.ID] = user
				fmt.Println("Usuario replicado:", user)
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

		var updatedUsers []models.User
		if err := json.Unmarshal(body, &updatedUsers); err == nil {
			mutex.Lock()
			for _, user := range updatedUsers {
				replicatedUsers[user.ID] = user
				fmt.Println("Usuario actualizado/replicado:", user)
				lastID = user.ID
			}
			mutex.Unlock()
		}
		time.Sleep(5 * time.Second)
	}
}

func deleteReplicatedUser(id int) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(replicatedUsers, id)
	fmt.Println("Usuario eliminado en replicación:", id)
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


func deleteSync() {
	for {
		time.Sleep(10 * time.Second) 
		resp, err := http.Get("http://localhost:8080/users")
		if err != nil {
			fmt.Println("Error al obtener lista de usuarios:", err)
			continue
		}
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var users []models.User
		if err := json.Unmarshal(body, &users); err == nil {
			mutex.Lock()
			for id := range replicatedUsers {
				found := false
				for _, user := range users {
					if user.ID == id {
						found = true
						break
					}
				}
				if !found {
					delete(replicatedUsers, id)
					fmt.Println("Usuario eliminado en replicación:", id)
				}
			}
			mutex.Unlock()
		}
	}
}

func main() {
	go longPollingReplication()
	go shortPollingReplication()
	go deleteSync() 

	r := gin.Default()
	r.GET("/replicated-users", getReplicatedUsers)

	fmt.Println("Servidor de replicación corriendo en :8081")
	r.Run(":8081")
}
