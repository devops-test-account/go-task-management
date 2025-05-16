package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
)

type Notification struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationEvent struct {
	UserID    int    `json:"user_id"`
	TaskID    int    `json:"task_id"`
	EventType string `json:"event_type"`
	Message   string `json:"message"`
}

var (
	db     *sql.DB
	writer *kafka.Writer
	reader *kafka.Reader
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Database connection
	username := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("MYSQL_PORT")

	dsn := username + ":" + password + "@tcp(" + host + ":" + port + ")/notification_db"
	db, err = sql.Open("mysql", dsn)
	// db, err = sql.Open("mysql", "root:rootpassword@tcp(mysql:3306)/notification_db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Kafka Writer setup
	writer = &kafka.Writer{
		Addr:     kafka.TCP("kafka:9092"),
		Topic:    "task-notifications",
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	// Kafka Reader setup
	reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{"kafka:9092"},
		Topic:     "task-notifications",
		GroupID:   "notification-service",
		Partition: 0,
	})
	defer reader.Close()

	// Start Kafka message consumption
	go consumeNotifications()

	// Gin router setup
	r := gin.Default()
	r.POST("/api/notifications/send", sendNotification)
	r.GET("/api/notifications/user/:user_id", getUserNotifications)
	r.PUT("/api/notifications/:id/read", markNotificationAsRead)

	r.Run(":8084")
}

func sendNotification(c *gin.Context) {
	var event NotificationEvent
	if err := c.BindJSON(&event); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Publish to Kafka
	jsonData, err := json.Marshal(event)
	if err != nil {
		// Handle the error (e.g., log it, return it, etc.)
		log.Fatalf("Error marshaling event: %v", err)
	}

	err = writer.WriteMessages(context.Background(),
		kafka.Message{
			Value: jsonData,
		},
	)

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to send notification" + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Notification queued"})
}

func consumeNotifications() {
	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading message: %v", err)
			continue
		}

		var event NotificationEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// Create notification in database
		_, err = db.Exec(
			"INSERT INTO notifications (user_id, message, type, is_read, created_at) VALUES (?, ?, ?, ?, ?)",
			event.UserID, event.Message, event.EventType, false, time.Now(),
		)
		if err != nil {
			log.Printf("Error saving notification: %v", err)
		}
	}
}

func getUserNotifications(c *gin.Context) {
	userID := c.Param("user_id")

	rows, err := db.Query(
		"SELECT id, user_id, message, type, is_read, created_at FROM notifications WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to retrieve notifications"})
		return
	}
	defer rows.Close()

	var notifications []Notification
	var createdAtbytes []byte
	for rows.Next() {
		var notification Notification
		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Message,
			&notification.Type,
			&notification.IsRead,
			&createdAtbytes,
		)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to scan notification"})
			return
		}
		notification.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", string(createdAtbytes))
		notifications = append(notifications, notification)
	}

	c.JSON(200, notifications)
}

func markNotificationAsRead(c *gin.Context) {
	notificationID := c.Param("id")

	_, err := db.Exec(
		"UPDATE notifications SET is_read = true WHERE id = ?",
		notificationID,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to mark notification as read"})
		return
	}

	c.JSON(200, gin.H{"message": "Notification marked as read"})
}
