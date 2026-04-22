package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

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

func getWorkerCount() int {
	if count := os.Getenv("WORKER_COUNT"); count != "" {
		if c, err := fmt.Sscanf(count, "%d", new(int)); err == nil && c == 1 {
			var wc int
			fmt.Sscanf(count, "%d", &wc)
			return wc
		}
	}
	return 5
}

const defaultWorkerCount = 5

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

	log.Println("connected to postgres")
}

func main() {
	initDB()

	log.Println("Worker started... Waiting for events")

	// Channel to hold jobs
	jobs := make(chan string, 100)

	// Start workers
	workerCount := getWorkerCount()
	for i := 0; i < workerCount; i++ {
		go worker(jobs, i)
	}

	// Fetch from Redis
	rdb := getRedisClient()
	for {
		result, err := rdb.BRPop(ctx, 0*time.Second, "github_events_queue", "retry_queue").Result()
		if err != nil {
			log.Println("Redis error:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		eventJSON := result[1]

		// Send job to workers
		jobs <- eventJSON
	}

	// for {
	// 	// BRPOP = Blocking pop from queue
	// 	// Waits until a message is available
	// 	result, err := rdb.BRPop(ctx, 0*time.Second, "github_events_queue", "retry_queue").Result()
	// 	if err != nil {
	// 		log.Println("Redis error:", err)
	// 		continue
	// 	}

	// 	// result[1] contains actual data
	// 	eventJSON := result[1]

	// 	log.Println("Received event from queue")

	// 	//  JSON → Go map
	// 	var event map[string]interface{}
	// 	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
	// 		log.Println("Error parsing event:", err)
	// 		continue
	// 	}

	// 	// Process event
	// 	log.Println("Processing event...")

	// 	// simulating failure

	// 	repo, ok := event["repo"].(string)
	// 	log.Printf("Repo value: '%s'\n", repo)
	// 	if !ok {
	// 		log.Println("Invalid repo type")
	// 		continue
	// 	}
	// 	// log.Printf("Repo value: %v, Type: %T\n", event["repo"], event["repo"])

	// 	if repo == "github-event-system" {
	// 		log.Println("simulated failure")

	// 		retryVal, ok := event["retry_count"].(float64)
	// 		if !ok {
	// 			retryVal = 0
	// 		}
	// 		retryCount := int(retryVal)

	// 		if retryCount < 3 {
	// 			// Increment retry count
	// 			event["retry_count"] = retryCount + 1

	// 			// Convert back to JSON
	// 			updatedJSON, _ := json.Marshal(event)

	// 			// Push back to retry queue
	// 			err := rdb.LPush(ctx, "retry_queue", updatedJSON).Err()
	// 			if err != nil {
	// 				log.Println("Retry push error:", err)
	// 			}

	// 			log.Println("Event pushed to retry queue (attempt)", retryCount+1)

	// 		} else {
	// 			// Move to Dead Letter Queue
	// 			eventJSON, _ := json.Marshal(event)

	// 			err := rdb.LPush(ctx, "dead_letter_queue", eventJSON).Err()
	// 			if err != nil {
	// 				log.Println("DLQ push error:", err)
	// 			}

	// 			log.Println("Event moved to Dead letter queue")

	// 		}

	// 		continue
	// 	}

	// 	log.Println("----- Processing Event -----")
	// 	log.Println("Event Type:", event["event_type"])
	// 	log.Println("Repo:", event["repo"])
	// 	log.Println("Branch:", event["branch"])
	// 	log.Println("Message:", event["message"])
	// 	log.Println("----------------------------")
	// }
}

func worker(jobs <-chan string, id int) {
	log.Printf("Worker %d started\n", id)

	for eventJSON := range jobs {
		log.Printf("Worker %d processing event\n", id)
		processEvent(eventJSON)
	}
}

func processEvent(eventJSON string) {

	var event map[string]interface{}

	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		log.Println("Error parsing event:", err)
		return
	}

	deliveryID, ok := event["delivery_id"].(string)
	if !ok {
		log.Println("Invalid delivery ID")
		return
	}

	rdb := getRedisClient()

	// Log event pickup
	appendLog(deliveryID, "Worker picked up event for processing")

	// Check if already processed (only for non-retry events)
	isRetry, _ := event["is_retry"].(bool)
	retryVal, _ := event["retry_count"].(float64)
	retryCount := int(retryVal)

	message, _ := event["message"].(string)

	if !isRetry {
		rdb := getRedisClient()
		exists, _ := rdb.SIsMember(ctx, "processed_events", deliveryID).Result()
		if exists {
			appendLog(deliveryID, "Duplicate event detected, skipping processing")
			log.Println("Duplicate event detected, skipping:", deliveryID)
			return
		}
		appendLog(deliveryID, "Starting initial event processing")
	} else {
		appendLog(deliveryID, fmt.Sprintf("Starting retry processing (attempt %d)", int(event["retry_count"].(float64))))
	}

	log.Println("Processing event...")

	// repo, ok := event["repo"].(string)
	if !ok {
		log.Println("Invalid repo in event")
		return
	}

	// Simulate failure for github-event-system repo

	rand.Seed(time.Now().UnixNano())

	shouldFail := false

	// Condition 1: message contains "fail"
	if strings.Contains(strings.ToLower(message), "fail") {
		shouldFail = true
	}

	// Condition 2: random failure (30%)
	if rand.Intn(100) < 30 {
		shouldFail = true
	}

	if shouldFail {
		appendLog(deliveryID, "Simulated failure occurred")

		if retryCount < 3 {
			// Increment retry
			retryCount++
			event["retry_count"] = retryCount
			event["is_retry"] = true

			updatedJSON, err := json.Marshal(event)
			if err != nil {
				log.Println("Error marshaling retry event:", err)
				return
			}

			// Exponential backoff
			delay := time.Duration(retryCount) * time.Second
			appendLog(deliveryID, fmt.Sprintf("Retrying after %v delay", delay))
			time.Sleep(delay)

			err = rdb.LPush(ctx, "retry_queue", updatedJSON).Err()
			if err != nil {
				log.Println("Retry push error:", err)
				appendLog(deliveryID, "Failed to push to retry queue")
				return
			}

			appendLog(deliveryID, fmt.Sprintf("Retry attempt %d queued", retryCount))
			saveEvent(event, "retry", retryCount)

		} else {
			// --- DEAD LETTER QUEUE ---
			eventJSON, _ := json.Marshal(event)

			err := rdb.LPush(ctx, "dead_letter_queue", eventJSON).Err()
			if err != nil {
				log.Println("DLQ push error:", err)
				appendLog(deliveryID, "Failed to push to DLQ")
				return
			}

			appendLog(deliveryID, "Max retries reached → moved to DLQ")
			saveEvent(event, "failed", retryCount)
		}

		return
	}

	// Success case
	appendLog(deliveryID, fmt.Sprintf("Successfully processed event: %s", event["message"]))
	log.Println("SUCCESS processed event:", deliveryID, "message:", event["message"])

	saveEvent(event, "success", retryCount)

	// Mark as processed only on success
	appendLog(deliveryID, "Marking event as processed")
	err := rdb.SAdd(ctx, "processed_events", deliveryID).Err()
	if err != nil {
		appendLog(deliveryID, "Failed to mark as processed in Redis")
		log.Println("Error marking as processed:", err)
	} else {
		appendLog(deliveryID, "Event processing completed successfully")
	}

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

func saveEvent(event map[string]interface{}, status string, retryCount int) {
	deliveryID := event["delivery_id"].(string)

	_, err := db.Exec(`
	INSERT INTO events (delivery_id, event_type, repo, branch, message, status, retry_count)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (delivery_id)
	DO UPDATE SET 
	status = EXCLUDED.status,
	retry_count = EXCLUDED.retry_count,
	updated_at = CURRENT_TIMESTAMP
	`,
		deliveryID,
		event["event_type"],
		event["repo"],
		event["branch"],
		event["message"],
		status,
		retryCount,
	)

	if err != nil {
		log.Println("DB insert error:", err)
	}
}
