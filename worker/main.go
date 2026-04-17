package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"database/sql"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

var rdb = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

const workerCount = 5

var db *sql.DB

func initDB() {
	var err error

	connStr := "user=postgres dbname=github_events sslmode=disable"

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
	for i := 0; i < workerCount; i++ {
		go worker(jobs, i)
	}

	// Fetch from Redis
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

	// Check if already processed (only for non-retry events)
	isRetry, _ := event["is_retry"].(bool)

	if !isRetry {
		exists, _ := rdb.SIsMember(ctx, "processed_events", deliveryID).Result()
		if exists {
			log.Println("Duplicate event detected, skipping:", deliveryID)
			return
		}
	}

	log.Println("Processing event...")

	repo, ok := event["repo"].(string)
	if !ok {
		log.Println("Invalid repo in event")
		return
	}

	// Simulate failure for github-event-system repo
	if repo == "github-event-system" {
		log.Println("simulated failure for repo:", repo)

		retryVal, ok := event["retry_count"].(float64)
		if !ok {
			retryVal = 0
		}
		retryCount := int(retryVal)

		if retryCount < 3 {
			event["retry_count"] = retryCount + 1
			event["is_retry"] = true
			updatedJSON, err := json.Marshal(event)
			if err != nil {
				log.Println("Error marshaling retry event:", err)
				return
			}

			// Add delay before retry (exponential backoff)
			delay := time.Duration(retryCount+1) * time.Second
			time.Sleep(delay)

			err = rdb.LPush(ctx, "retry_queue", updatedJSON).Err()
			if err != nil {
				log.Println("Error pushing to retry queue:", err)
				return
			}

			log.Println("Retry attempt:", retryCount+1, "for delivery:", deliveryID)
			saveEvent(event, "retry", retryCount+1)

		} else {
			// Max retries reached, move to dead letter queue
			eventJSON, err := json.Marshal(event)
			if err != nil {
				log.Println("Error marshaling failed event:", err)
				return
			}

			err = rdb.LPush(ctx, "dead_letter_queue", eventJSON).Err()
			if err != nil {
				log.Println("Error pushing to DLQ:", err)
				return
			}

			log.Println("Max retries reached, moved to DLQ for delivery:", deliveryID)
			saveEvent(event, "failed", retryCount)
		}

		return
	}

	// Success case
	log.Println("SUCCESS processed event:", deliveryID, "message:", event["message"])

	retryVal, ok := event["retry_count"].(float64)
	if !ok {
		retryVal = 0
	}
	retryCount := int(retryVal)

	saveEvent(event, "success", retryCount)

	// Mark as processed only on success
	err := rdb.SAdd(ctx, "processed_events", deliveryID).Err()
	if err != nil {
		log.Println("Error marking as processed:", err)
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
	retry_count = EXCLUDED.retry_count
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
