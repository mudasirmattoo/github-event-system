package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

var rdb = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

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
		branchName := strings.TrimPrefix(branch.(string), "refs/heads/")

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

		// log.Println("----- New Event -----")
		// log.Println("Delivery ID:", deliveryID)
		// log.Println("Event Type:", eventType)
		// log.Println("Repo:", repoName)
		// log.Println("Full Repo:", fullRepo)
		// log.Println("Branch:", branch)
		// log.Println("Branch Name:", branchName)
		// log.Println("Pushed by:", pusher)
		// log.Println("Commit ID:", commitID)
		// log.Println("Message:", message)
		// log.Println("Timestamp:", timestamp)
		// log.Println("Compare URL:", compareURL)

		event := map[string]interface{}{
			"event_type":  eventType,
			"delivery_id": deliveryID,
			"repo":        repoName,
			"full_repo":   fullRepo,
			"branch":      branch,
			"branch name": branchName,
			"pusher":      pusher,
			"commit_id":   commitID,
			"message":     message,
			"timestamp":   timestamp,
			"compare_url": compareURL,
			"retry_count": 0,
		}

		// Convert event → JSON string
		eventJSON, err := json.Marshal(event)
		if err != nil {
			log.Println("Error marshaling event:", err)
			return
		}

		// push to redis queue -- name github_events_queue
		err = rdb.LPush(ctx, "github_events_queue", eventJSON).Err()
		if err != nil {
			log.Println("Redis push error:", err)
			return
		}

		log.Println("Event pushed to Redis queue")

		c.JSON(200, gin.H{
			"message": "Webhook received",
		})
	})

	r.Run(":8080")
}
