package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.POST("/webhook", func(c *gin.Context) {

		eventType := c.GetHeader("X-GitHub-Event")

		body, err := c.GetRawData()
		if err != nil {
			c.JSON(400, gin.H{"error": "Cannot read body"})
			return
		}

		log.Println("Event Type:", eventType)
		log.Println("Raw Body:", string(body))

		c.JSON(200, gin.H{
			"message": "Webhook received",
		})
	})

	r.Run(":8080")
}
