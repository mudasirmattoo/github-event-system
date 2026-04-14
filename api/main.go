package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.POST("/webhook", func(c *gin.Context) {
		type WebhookPayload struct {
			Event string `json:"event"`
			Repo  string `json:"repo"`
		}

		var payload WebhookPayload

		if err := c.BindJSON(&payload); err != nil {
			c.JSON(400, gin.H{"error": "Invalid JSON"})
			return
		}

		log.Println("Received event:", payload)

		c.JSON(200, gin.H{
			"message": "Webhook received",
		})
	})

	r.Run(":8080")
}
