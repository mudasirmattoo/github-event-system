package main

import (
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.POST("/webhook", func(c *gin.Context) {

		eventType := c.GetHeader("X-GitHub-Event")

		deliveryID := c.GetHeader("X-Github-Delivery")

		body, err := c.GetRawData()
		if err != nil {
			c.JSON(400, gin.H{"error": "Cannot read body"})
			return
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Println("Error parsing JSON:", err)
			return
		}

		// Repository info
		repoData := payload["repository"].(map[string]interface{})
		repoName := repoData["name"]
		fullRepo := repoData["full_name"]

		// Branch (ref = refs/heads/main)
		branch := payload["ref"]

		// User info
		pusherData := payload["pusher"].(map[string]interface{})
		pusher := pusherData["name"]

		// Commit info (take first commit for now)
		commits := payload["commits"].([]interface{})

		var commitID, message, timestamp interface{}

		if len(commits) > 0 {
			firstCommit := commits[0].(map[string]interface{})
			commitID = firstCommit["id"]
			message = firstCommit["message"]
			timestamp = firstCommit["timestamp"]
		}

		compareURL := payload["compare"]

		log.Println("----- New Event -----")
		log.Println("Delivery ID:", deliveryID)
		log.Println("Event Type:", eventType)
		log.Println("Repo:", repoName)
		log.Println("Full Repo:", fullRepo)
		log.Println("Branch:", branch)
		log.Println("Pushed by:", pusher)
		log.Println("Commit ID:", commitID)
		log.Println("Message:", message)
		log.Println("Timestamp:", timestamp)
		log.Println("Compare URL:", compareURL)

		c.JSON(200, gin.H{
			"message": "Webhook received",
		})
	})

	r.Run(":8080")
}
