package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

var rdb = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

func main() {
	log.Println("Worker started... Waiting for events")

	for {
		// BRPOP = Blocking pop from queue
		// Waits until a message is available
		result, err := rdb.BRPop(ctx, 0*time.Second, "github_events_queue", "retry_queue").Result()
		if err != nil {
			log.Println("Redis error:", err)
			continue
		}

		// result[1] contains actual data
		eventJSON := result[1]

		log.Println("Received event from queue")

		//  JSON → Go map
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
			log.Println("Error parsing event:", err)
			continue
		}

		// Process event (for now just log)
		log.Println("Processing event...")

		// simulating failure

		if event["repo"] == "github-event-system" {
			log.Println("simulated failure")

			retryCount := int(event["retry_count"].(float64))

			if retryCount < 3 {
				// Increment retry count
				event["retry_count"] = retryCount + 1

				// Convert back to JSON
				updatedJSON, _ := json.Marshal(event)

				// Push back to retry queue
				err := rdb.LPush(ctx, "retry_queue", updatedJSON).Err()
				if err != nil {
					log.Println("Retry push error:", err)
				}

				log.Println("Event pushed to retry queue (attempt)", retryCount+1)

			} else {
				// Move to Dead Letter Queue
				eventJSON, _ := json.Marshal(event)

				err := rdb.LPush(ctx, "dead_letter_queue", eventJSON).Err()
				if err != nil {
					log.Println("DLQ push error:", err)
				}

				log.Println("Event moved to Dead letter queue")

			}

			continue
		}

		log.Println("----- Processing Event -----")
		log.Println("Event Type:", event["event_type"])
		log.Println("Repo:", event["repo"])
		log.Println("Branch:", event["branch"])
		log.Println("Message:", event["message"])
		log.Println("----------------------------")
	}
}
