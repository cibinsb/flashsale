package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/julienschmidt/httprouter"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- Configuration ---
var (
	redis_password = os.Getenv("REDIS_PASSWORD")
	redis_host     = os.Getenv("REDIS_HOST")
	redis_port     = os.Getenv("REDIS_PORT")
	RedisAddr      = fmt.Sprintf("redis://default:%s@%s:%s", redis_password, redis_host, redis_port)
	postgres_host  = os.Getenv("POSTGRES_HOST")
	postgres_port  = os.Getenv("POSTGRES_PORT")
	postgres_user  = os.Getenv("POSTGRES_USER")
	postgres_pass  = os.Getenv("POSTGRES_PASSWORD")
	postgres_db    = os.Getenv("POSTGRES_DB")

	PostgresDSN = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require", postgres_host, postgres_port, postgres_user, postgres_pass, postgres_db)
)

const (
	RedisCacheTTL = 5 * time.Minute // Time-To-Live for hot data
	Port          = ":8080"
)

// --- Models ---

// Order maps to the 'orders' table. gorm.Model is omitted for speed.
type Order struct {
	ID         string    `gorm:"primaryKey" json:"id"`
	UserID     string    `json:"user_id"`
	ItemSKU    string    `json:"item_sku"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"total_price"`
	CreatedAt  time.Time `json:"created_at"`
}

// --- Client Setup ---

var (
	ctx = context.Background()
	Rdb *redis.Client
	DB  *gorm.DB
)

func checkEnvVars() {
	requiredVars := []string{"REDIS_PASSWORD", "REDIS_HOST", "REDIS_PORT", "POSTGRES_HOST",
		"POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"}
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			log.Fatalf("Environment variable %s is not set", v)
		}
	}
	log.Println("✅ All required environment variables are set.")
}
func initClients() {
	// 1. Initialize Redis Client
	opt, _ := redis.ParseURL(RedisAddr)

	Rdb := redis.NewClient(opt)

	// Rdb = redis.NewClient(&redis.Options{
	// 	Addr:     RedisAddr,
	// 	PoolSize: 100, // Large pool size for 100K QPS I/O concurrency
	// })

	// Check Redis connection
	_, err := Rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	log.Println("✅ Successfully connected to Redis.")

	// 2. Initialize PostgreSQL Client
	// Set up database connection pool for GORM
	db, err := gorm.Open(postgres.Open(PostgresDSN), &gorm.Config{
		// Disable default transaction for single read queries to reduce overhead
		SkipDefaultTransaction: true,
	})
	if err != nil {
		log.Fatalf("Could not connect to Postgres: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Could not get underlying SQL DB: %v", err)
	}

	// Configure connection pool for high concurrency
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50) // Adjust based on DB capacity
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	log.Println("✅ Successfully connected to PostgreSQL.")
}

// --- Handlers ---

// GetOrderHandler fetches an order by ID, implementing the Redis-first logic.
func GetOrderHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	orderID := ps.ByName("id")
	if orderID == "" {
		http.Error(w, "Order ID is required", http.StatusBadRequest)
		return
	}

	start := time.Now()

	// 1. Check Redis (HIGH-SPEED PATH)
	orderJSON, err := Rdb.Get(ctx, orderID).Result()

	if err == nil {
		// Cache Hit: Deserialize and return (Target: <5ms)
		w.Header().Set("X-Cache-Status", "HIT")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s\n", orderJSON)

		log.Printf("HIT | ID: %s | Latency: %s", orderID, time.Since(start))
		return
	}

	if err != redis.Nil {
		// Log Redis error but attempt to continue to DB
		log.Printf("Redis Error for ID %s: %v. Falling back to DB.", orderID, err)
	}

	// 2. Fallback to PostgreSQL (SLOWER PATH)
	var order Order
	// Use Raw SQL for minimal GORM overhead on a primary key query
	result := DB.Raw("SELECT * FROM orders WHERE id = $1", orderID).Scan(&order)

	if result.Error == gorm.ErrRecordNotFound {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}
	if result.Error != nil {
		// Critical error connecting or querying DB
		log.Printf("Postgres Error for ID %s: %v", orderID, result.Error)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 3. Cache Miss: Serialize, return, and populate Redis

	// Serialize to JSON
	jsonBytes, err := json.Marshal(order)
	if err != nil {
		log.Printf("JSON serialization error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ASYNCHRONOUS CACHE WRITE (Fire-and-forget to avoid blocking the user response)
	// Use a goroutine to update the cache
	// Better practice: Use a context with a timeout for the fire-and-forget task
	go func() {
		// Set a short timeout (e.g., 50ms) for the cache operation
		timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel() // MUST call cancel to release context resources

		Rdb.Set(timeoutCtx, orderID, jsonBytes, RedisCacheTTL)
		// Log error if it failed, but do not block the main response path
	}()

	// Respond to the client (Target: <50ms)
	w.Header().Set("X-Cache-Status", "MISS")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", jsonBytes)

	log.Printf("MISS | ID: %s | Latency: %s", orderID, time.Since(start))
}

// --- Main Function ---

func main() {
	// Check required environment variables
	checkEnvVars()

	// Initialize connections and pools
	initClients()

	// Initialize the fast router
	router := httprouter.New()
	router.GET("/order/:id", GetOrderHandler)

	log.Printf("🚀 Starting Flash Sale Query Service on port %s", Port)

	// Start the server
	if err := http.ListenAndServe(Port, router); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
