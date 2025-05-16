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

type TaskAssignment struct {
	ID         int       `json:"id"`
	TaskID     int       `json:"task_id"`
	AssignedTo int       `json:"assigned_to"`
	AssignedBy int       `json:"assigned_by"`
	AssignedAt time.Time `json:"assigned_at"`
	Status     string    `json:"status"`
}

type TaskWithAssignment struct {
	TaskID         int       `json:"task_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	TaskStatus     string    `json:"task_status"`
	AssignedTo     int       `json:"assigned_to"`
	AssignedToName string    `json:"assigned_to_name"`
	AssignmentDate time.Time `json:"assignment_date"`
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

	dsn := username + ":" + password + "@tcp(" + host + ":" + port + ")/task_assignment_db"
	db, err = sql.Open("mysql", dsn)
	// db, err = sql.Open("mysql", "root:rootpassword@tcp(mysql:3306)/task_assignment_db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := gin.Default()

	// Task assignment routes
	r.POST("/api/assignments/assign", assignTask)
	r.GET("/api/assignments/user/:user_id", getAssignedTasksForUser)
	r.PUT("/api/assignments/:id/status", updateAssignmentStatus)
	r.GET("/api/assignments", getAllAssignments)

	r.Run(":8083")
}

func assignTask(c *gin.Context) {
	var assignment TaskAssignment
	if err := c.BindJSON(&assignment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate task exists in task database
	var taskExists int
	err := db.QueryRow("SELECT COUNT(*) FROM task_db.tasks WHERE id = ?", assignment.TaskID).Scan(&taskExists)
	if err != nil || taskExists == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task does not exist"})
		return
	}

	// Validate user exists
	var userExists int
	err = db.QueryRow("SELECT COUNT(*) FROM user_db.users WHERE id = ?", assignment.AssignedTo).Scan(&userExists)
	if err != nil || userExists == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User does not exist"})
		return
	}

	// Insert assignment
	result, err := db.Exec(
		"INSERT INTO task_assignments (task_id, assigned_to, assigned_by, assigned_at, status) VALUES (?, ?, ?, ?, ?)",
		assignment.TaskID, assignment.AssignedTo, assignment.AssignedBy, time.Now(), "ASSIGNED",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Task assignment failed"})
		return
	}

	// Get the ID of the newly created assignment
	id, _ := result.LastInsertId()
	assignment.ID = int(id)
	assignment.AssignedAt = time.Now()
	assignment.Status = "ASSIGNED"

	c.JSON(http.StatusCreated, assignment)
}

func getAssignedTasksForUser(c *gin.Context) {
	userID := c.Param("user_id")

	query := `
		SELECT 
			t.id, 
			t.title, 
			t.description, 
			t.status, 
			ta.assigned_to, 
			u.username, 
			ta.assigned_at
		FROM task_assignment_db.task_assignments ta
		JOIN task_db.tasks t ON ta.task_id = t.id
		JOIN user_db.users u ON ta.assigned_to = u.id
		WHERE ta.assigned_to = ?
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve assigned tasks"})
		return
	}
	defer rows.Close()

	var assignedTasks []TaskWithAssignment
	var assignmentDateByte []byte
	for rows.Next() {
		var task TaskWithAssignment
		err := rows.Scan(
			&task.TaskID,
			&task.Title,
			&task.Description,
			&task.TaskStatus,
			&task.AssignedTo,
			&task.AssignedToName,
			&assignmentDateByte,
		)

		task.AssignmentDate, _ = time.Parse("2006-01-02 15:04:05", string(assignmentDateByte))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan task"})
			return
		}
		assignedTasks = append(assignedTasks, task)
	}

	c.JSON(http.StatusOK, assignedTasks)
}

func updateAssignmentStatus(c *gin.Context) {
	assignmentID := c.Param("id")

	var statusUpdate struct {
		Status string `json:"status"`
	}

	if err := c.BindJSON(&statusUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update assignment status
	_, err := db.Exec(
		"UPDATE task_assignments SET status = ? WHERE id = ?",
		statusUpdate.Status, assignmentID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assignment status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Assignment status updated successfully",
		"status":  statusUpdate.Status,
	})
}

func getAllAssignments(c *gin.Context) {
	query := `
		SELECT 
			ta.id, 
			ta.task_id, 
			ta.assigned_to, 
			ta.assigned_by, 
			ta.assigned_at, 
			ta.status
		FROM task_assignments ta
	`

	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve assignments"})
		return
	}
	defer rows.Close()

	var assignments []TaskAssignment
	for rows.Next() {
		var assignment TaskAssignment
		var assignedAtBytes []byte
		err := rows.Scan(
			&assignment.ID,
			&assignment.TaskID,
			&assignment.AssignedTo,
			&assignment.AssignedBy,
			&assignedAtBytes,
			&assignment.Status,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan assignment"})
			return
		}
		assignment.AssignedAt, _ = time.Parse("2006-01-02 15:04:05", string(assignedAtBytes))
		assignments = append(assignments, assignment)
	}

	c.JSON(http.StatusOK, assignments)
}
