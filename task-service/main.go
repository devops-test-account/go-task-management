package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type Task struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedBy   int       `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

var db *sql.DB

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

	dsn := username + ":" + password + "@tcp(" + host + ":" + port + ")/task_db"
	db, err = sql.Open("mysql", dsn)
	// db, err = sql.Open("mysql", "root:rootpassword@tcp(mysql:3306)/task_db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := gin.Default()

	// Task creation routes
	r.POST("/api/tasks", createTask)
	r.GET("/api/tasks", getAllTasks)
	r.GET("/api/tasks/:id", getTaskByID)
	r.PUT("/api/tasks/:id", updateTask)
	r.DELETE("/api/tasks/:id", deleteTask)

	r.Run(":8082")
}

func createTask(c *gin.Context) {
	var task Task
	if err := c.BindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default status if not provided
	if task.Status == "" {
		task.Status = "TODO"
	}

	// Insert task into database
	result, err := db.Exec(
		"INSERT INTO tasks (title, description, status, created_by, created_at) VALUES (?, ?, ?, ?, ?)",
		task.Title, task.Description, task.Status, task.CreatedBy, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Task creation failed: "})
		return
	}

	// Get the ID of the newly created task
	id, _ := result.LastInsertId()
	task.ID = int(id)
	task.CreatedAt = time.Now()

	c.JSON(http.StatusCreated, task)
}

func getAllTasks(c *gin.Context) {
	rows, err := db.Query("SELECT id, title, description, status, created_by, created_at FROM tasks")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var createdAtBytes []byte
		err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.CreatedBy, &createdAtBytes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan task"})
			return
		}

		task.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", string(createdAtBytes))

		tasks = append(tasks, task)
	}

	c.JSON(http.StatusOK, tasks)
}

func getTaskByID(c *gin.Context) {
	id := c.Param("id")

	var task Task
	var createdAtBytes []byte
	err := db.QueryRow("SELECT id, title, description, status, created_by, created_at FROM tasks WHERE id = ?", id).
		Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.CreatedBy, &createdAtBytes)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found" + err.Error()})
		return
	}
	task.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", string(createdAtBytes))

	c.JSON(http.StatusOK, task)
}

func updateTask(c *gin.Context) {
	id := c.Param("id")
	var task Task
	if err := c.BindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default status if not provided
	if task.Status == "" {
		task.Status = "TODO"
	}

	_, err := db.Exec(
		"UPDATE tasks SET title = ?, description = ?, status = ? WHERE id = ?",
		task.Title, task.Description, task.Status, id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Task update failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task updated successfully"})
}

func deleteTask(c *gin.Context) {
	id := c.Param("id")

	_, err := db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Task deletion failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}
