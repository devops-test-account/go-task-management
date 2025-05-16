package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"
)

type ServiceConfig struct {
	Name    string
	BaseURL string
}

type APIGateway struct {
	services  map[string]*httputil.ReverseProxy
	jwtSecret []byte
}

func NewAPIGateway(jwtSecret string) *APIGateway {
	return &APIGateway{
		services: map[string]*httputil.ReverseProxy{
			"users":         createReverseProxy(os.Getenv("USER_URL")),
			"tasks":         createReverseProxy(os.Getenv("TASKS_URL")),
			"assignments":   createReverseProxy(os.Getenv("ASSIGNMENTS_URL")),
			"notifications": createReverseProxy(os.Getenv("NOTIFICATIONS_URL")),
			"dashboard":     createReverseProxy(os.Getenv("DASHBOARD_URL")),
		},
		jwtSecret: []byte(jwtSecret),
	}
}

func createReverseProxy(targetURL string) *httputil.ReverseProxy {
	url, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Invalid URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(url)

	// Error handling
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Bad Gateway: " + err.Error()))
	}

	return proxy
}

func (g *APIGateway) authenticateRequest(c *gin.Context) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
		c.Abort()
		return false
	}

	// Remove "Bearer " prefix
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse and validate JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return g.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return false
	}

	return true
}

func (g *APIGateway) proxyRequest(serviceName string, requireAuth bool) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Skip authentication for public routes
		if requireAuth {
			if !g.authenticateRequest(c) {
				return
			}
		}

		proxy, exists := g.services[serviceName]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Service not found"})
			return
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	gateway := NewAPIGateway(os.Getenv("JWT_SECRET_KEY"))

	r := gin.Default()

	// CORS Middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// User Service Routes
	userRoutes := r.Group("/api/users")
	{
		// Public routes (no authentication required)
		userRoutes.POST("/register", gateway.proxyRequest("users", false))
		userRoutes.POST("/login", gateway.proxyRequest("users", false))
		// Protected routes
		userRoutes.GET("/profile/:id", gateway.proxyRequest("users", true))
	}

	// Web Routes for User Service
	r.GET("/register", gateway.proxyRequest("users", false))
	r.GET("/login", gateway.proxyRequest("users", false))
	r.GET("/profile", gateway.proxyRequest("users", false))

	// Task Creation Routes
	taskRoutes := r.Group("/api/tasks")
	{
		taskRoutes.POST("", gateway.proxyRequest("tasks", true))
		taskRoutes.GET("", gateway.proxyRequest("tasks", true))
		taskRoutes.GET("/:id", gateway.proxyRequest("tasks", true))
		taskRoutes.PUT("/:id", gateway.proxyRequest("tasks", true))
		taskRoutes.DELETE("/:id", gateway.proxyRequest("tasks", true))
	}

	// Task Assignment Routes
	assignmentRoutes := r.Group("/api/assignments")
	{
		assignmentRoutes.POST("/assign", gateway.proxyRequest("assignments", true))
		assignmentRoutes.GET("/user/:user_id", gateway.proxyRequest("assignments", true))
		assignmentRoutes.PUT("/:id/status", gateway.proxyRequest("assignments", true))
		assignmentRoutes.GET("", gateway.proxyRequest("assignments", true))
	}

	// Notification Routes
	notificationRoutes := r.Group("/api/notifications")
	{
		notificationRoutes.POST("/send", gateway.proxyRequest("notifications", true))
		notificationRoutes.GET("/user/:user_id", gateway.proxyRequest("notifications", true))
		notificationRoutes.PUT("/:id/read", gateway.proxyRequest("notifications", true))
	}

	// Dashboard Routes
	dashboardRoutes := r.Group("/api/dashboard")
	{
		dashboardRoutes.GET("/:user_id", gateway.proxyRequest("dashboard", true))
		dashboardRoutes.GET("/:user_id/tasks", gateway.proxyRequest("dashboard", true))
	}

	// Web Routes for Dashboard Service
	r.GET("/tasks/:id", gateway.proxyRequest("dashboard", false))

	r.GET("/routes", func(c *gin.Context) {
		routes := []string{}
		for _, ri := range r.Routes() {
			routes = append(routes, ri.Method+" "+ri.Path)
		}
		c.JSON(http.StatusOK, routes)
	})

	// Start server
	server := &http.Server{
		Addr:           ":8080",
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Println("API Gateway running on :8080")
	log.Fatal(server.ListenAndServe())
}
