package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func getRedisClient() *redis.Client {
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisAddr := redisHost + ":" + redisPort

	return redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

var db *sql.DB

func initDB() {
	var err error

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "github_events")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("DB not reachable:", err)
	}

	log.Println("connected to PostgreSQL")
}

func appendLog(deliveryID, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s", timestamp, message)

	_, err := db.Exec(`
	UPDATE events 
	SET logs = CASE 
		WHEN logs IS NULL OR logs = '' THEN $1
		ELSE logs || CHR(10) || $1
		END,
		updated_at = CURRENT_TIMESTAMP
	WHERE delivery_id = $2
	`, logEntry, deliveryID)

	if err != nil {
		log.Printf("Failed to append log for %s: %v", deliveryID, err)
	}
}

func main() {
	initDB()

	r := gin.Default()

	r.Use(cors.Default())

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
			"is_retry":    false,
		}

		// Convert event → JSON string
		eventJSON, err := json.Marshal(event)
		if err != nil {
			log.Println("Error marshaling event:", err)
			return
		}

		// Initialize event in database with initial log
		_, err = db.Exec(`
		INSERT INTO events (delivery_id, event_type, repo, branch, message, status, retry_count, logs)
		VALUES ($1, $2, $3, $4, $5, 'pending', 0, $6)
		ON CONFLICT (delivery_id) DO NOTHING
		`, deliveryID, eventType, repoName, branchName, message, fmt.Sprintf("[%s] Event received from GitHub", time.Now().Format("2006-01-02 15:04:05")))

		if err != nil {
			log.Printf("Failed to initialize event in database: %v", err)
		}

		// Log event received
		appendLog(deliveryID, "Event received and validated")

		// push to redis queue -- name github_events_queue
		rdb := getRedisClient()
		err = rdb.LPush(ctx, "github_events_queue", eventJSON).Err()
		if err != nil {
			log.Println("Redis push error:", err)
			appendLog(deliveryID, "Failed to push event to Redis queue")
			return
		}

		appendLog(deliveryID, "Event pushed to Redis queue for processing")
		log.Println("Event pushed to Redis queue")

		c.JSON(200, gin.H{
			"message": "Webhook received",
		})
	})

	r.GET("/", func(c *gin.Context) {
		// Serve the frontend HTML file
		c.File("../frontend/index.html")
	})

	r.GET("/events", func(c *gin.Context) {
		rows, err := db.Query(`
		SELECT delivery_id, event_type, repo, branch, message, status, retry_count, created_at
		FROM events
		ORDER BY created_at DESC
		`)
		if err != nil {
			c.JSON(500, gin.H{"error": "DB error"})
			return
		}
		defer rows.Close()

		var events []map[string]interface{}

		for rows.Next() {
			var deliveryID, eventType, repo, branch, message, status string
			var retryCount int
			var createdAt interface{}

			err := rows.Scan(&deliveryID, &eventType, &repo, &branch, &message, &status, &retryCount, &createdAt)
			if err != nil {
				continue
			}

			events = append(events, map[string]interface{}{
				"delivery_id": deliveryID,
				"event_type":  eventType,
				"repo":        repo,
				"branch":      branch,
				"message":     message,
				"status":      status,
				"retry_count": retryCount,
				"created_at":  createdAt,
			})
		}

		c.JSON(200, events)
	})

	r.GET("/events/:id/logs", func(c *gin.Context) {
		deliveryID := c.Param("id")

		var logs string
		err := db.QueryRow("SELECT logs FROM events WHERE delivery_id = $1", deliveryID).Scan(&logs)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error"})
			}
			return
		}

		// Split logs into array for better frontend handling
		var logLines []string
		if logs != "" {
			logLines = strings.Split(logs, "\n")
		}

		c.JSON(http.StatusOK, gin.H{
			"delivery_id": deliveryID,
			"logs":        logLines,
			"raw_logs":    logs,
		})
	})
	r.Run(":8080")
}
