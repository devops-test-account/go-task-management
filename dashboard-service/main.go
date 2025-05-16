package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type TaskDashboard struct {
	UserID        int           `json:"user_id"`
	Username      string        `json:"username"`
	TotalTasks    int           `json:"total_tasks"`
	TaskBreakdown TaskBreakdown `json:"task_breakdown"`
}

type TaskBreakdown struct {
	Todo       int `json:"todo"`
	InProgress int `json:"in_progress"`
	Completed  int `json:"completed"`
}

type TaskDetail struct {
	TaskID       int    `json:"task_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	AssignedDate string `json:"assigned_date"`
	DueDate      string `json:"due_date"`
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

	dsn := username + ":" + password + "@tcp(" + host + ":" + port + ")/dashboard_db"
	db, err = sql.Open("mysql", dsn)
	// db, err = sql.Open("mysql", "root:rootpassword@tcp(mysql:3306)/dashboard_db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := gin.Default()

	// Static files and templates
	r.Static("/static", "./static")
	r.SetFuncMap(template.FuncMap{
		"formatDate": formatDate,
	})
	r.LoadHTMLGlob("templates/*")

	// Web routes
	r.GET("/dashboard/:user_id", dashboardHandler)
	r.GET("/tasks/:user_id", tasksHandler)

	// API routes
	r.GET("/api/dashboard/:user_id", getUserDashboard)
	r.GET("/api/dashboard/:user_id/tasks", getUserTasks)

	r.Run(":8085")
}

func getUserDashboard(c *gin.Context) {
	userID := c.Param("user_id")

	// Get user details
	var username string
	err := db.QueryRow("SELECT username FROM user_db.users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get task breakdown
	var breakdown TaskBreakdown
	query := `
		SELECT 
			SUM(CASE WHEN t.status = 'TODO' THEN 1 ELSE 0 END) as todo,
			SUM(CASE WHEN t.status = 'IN_PROGRESS' THEN 1 ELSE 0 END) as in_progress,
			SUM(CASE WHEN t.status = 'COMPLETED' THEN 1 ELSE 0 END) as completed
		FROM task_assignment_db.task_assignments ta
		JOIN task_db.tasks t ON ta.task_id = t.id
		WHERE ta.assigned_to = ?
	`
	err = db.QueryRow(query, userID).Scan(
		&breakdown.Todo,
		&breakdown.InProgress,
		&breakdown.Completed,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task breakdown"})
		return
	}

	// Calculate total tasks
	totalTasks := breakdown.Todo + breakdown.InProgress + breakdown.Completed
	fmt.Printf("Total Tasks are: %d", totalTasks)
	dashboard := TaskDashboard{
		UserID:        parseInt(userID),
		Username:      username,
		TotalTasks:    totalTasks,
		TaskBreakdown: breakdown,
	}

	c.JSON(http.StatusOK, dashboard)
}

func getUserTasks(c *gin.Context) {
	userID := c.Param("user_id")
	status := c.DefaultQuery("status", "")

	query := `
		SELECT 
			t.id, 
			t.title, 
			t.description, 
			t.status, 
			ta.assigned_at
		FROM task_assignment_db.task_assignments ta
		JOIN task_db.tasks t ON ta.task_id = t.id
		WHERE ta.assigned_to = ?
	`
	args := []interface{}{userID}

	if status != "" {
		query += " AND t.status = ?"
		args = append(args, status)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks" + err.Error()})
		return
	}
	defer rows.Close()

	var tasks []TaskDetail
	for rows.Next() {
		var task TaskDetail
		err := rows.Scan(
			&task.TaskID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.AssignedDate,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan task"})
			return
		}
		tasks = append(tasks, task)
	}

	c.JSON(http.StatusOK, tasks)
}

// Utility function to parse string to int
func parseInt(s string) int {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	if err != nil {
		return 0
	}
	return i
}

func dashboardHandler(c *gin.Context) {
	userID := c.Param("user_id")

	// Fetch dashboard data
	var dashboard TaskDashboard
	query := `
		SELECT 
			u.username, 
			SUM(CASE WHEN t.status = 'TODO' THEN 1 ELSE 0 END) as todo,
			SUM(CASE WHEN t.status = 'IN_PROGRESS' THEN 1 ELSE 0 END) as in_progress,
			SUM(CASE WHEN t.status = 'COMPLETED' THEN 1 ELSE 0 END) as completed
		FROM user_db.users u
		LEFT JOIN task_assignment_db.task_assignments ta ON u.id = ta.assigned_to
		LEFT JOIN task_db.tasks t ON ta.task_id = t.id
		WHERE u.id = ?
		GROUP BY u.id, u.username
	`

	var breakdown TaskBreakdown
	err := db.QueryRow(query, userID).Scan(
		&dashboard.Username,
		&breakdown.Todo,
		&breakdown.InProgress,
		&breakdown.Completed,
	)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to retrieve dashboard data",
		})
		return
	}

	dashboard.UserID = parseInt(userID)
	dashboard.TotalTasks = breakdown.Todo + breakdown.InProgress + breakdown.Completed
	dashboard.TaskBreakdown = breakdown

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title":     "User Dashboard",
		"dashboard": dashboard,
	})
}

func tasksHandler(c *gin.Context) {
	userID := c.Param("user_id")
	status := c.DefaultQuery("status", "")

	// Fetch tasks
	query := `
		SELECT 
			t.id, 
			t.title, 
			t.description, 
			t.status, 
			ta.assigned_at
		FROM task_assignment_db.task_assignments ta
		JOIN task_db.tasks t ON ta.task_id = t.id
		WHERE ta.assigned_to = ?
	`
	args := []interface{}{userID}

	if status != "" {
		query += " AND t.status = ?"
		args = append(args, status)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to retrieve tasks",
		})
		return
	}
	defer rows.Close()

	var tasks []TaskDetail
	for rows.Next() {
		var task TaskDetail
		err := rows.Scan(
			&task.TaskID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.AssignedDate,
		)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"error": "Failed to process tasks",
			})
			return
		}
		tasks = append(tasks, task)
	}

	c.HTML(http.StatusOK, "tasks.html", gin.H{
		"title":  "User Tasks",
		"tasks":  tasks,
		"userID": userID,
	})
}

// Utility function to format date
func formatDate(dateStr string) string {
	// Implement date formatting logic
	return dateStr
}
