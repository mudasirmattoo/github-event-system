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
		result, err := rdb.BRPop(ctx, 0*time.Second, "github_events_queue").Result()
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
		log.Println("----- Processing Event -----")
		log.Println("Event Type:", event["event_type"])
		log.Println("Repo:", event["repo"])
		log.Println("Branch:", event["branch"])
		log.Println("Message:", event["message"])
		log.Println("----------------------------")
	}
}
