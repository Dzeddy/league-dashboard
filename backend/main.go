package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var app GlobalAppData

func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:3000":    true,
		"https://dzeddy.github.io": true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isAWSEnvironment detects if the application is running in an AWS environment
func isAWSEnvironment() bool {
	// Check for common AWS environment variables
	awsEnvVars := []string{
		"AWS_EXECUTION_ENV",
		"AWS_LAMBDA_FUNCTION_NAME",
		"ECS_CONTAINER_METADATA_URI",
		"AWS_REGION",
		"AWS_DEFAULT_REGION",
	}

	for _, envVar := range awsEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// loadEnvironmentConfig loads environment variables from .env file or AWS environment
func loadEnvironmentConfig() {
	isAWS := isAWSEnvironment()

	if isAWS {
		log.Println("AWS environment detected. Using environment variables from AWS.")
	} else {
		// Try to load .env file for local development
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found or error loading .env file. Using system environment variables.")
			log.Printf("Continuing with system environment variables. Error was: %v", err)
		} else {
			log.Println("Successfully loaded .env file for local development.")
		}
	}
}

// parseRedisConfig parses Redis configuration from environment variables
// Supports both simple host:port format and Redis URLs with credentials
func parseRedisConfig() (addr, password string, db int, err error) {
	// Default values
	addr = "localhost:6379"
	password = ""
	db = 0

	// Try REDIS_URL first (common in cloud deployments)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		log.Println("Using REDIS_URL for Redis configuration")

		// Parse the Redis URL
		u, parseErr := url.Parse(redisURL)
		if parseErr != nil {
			err = parseErr
			return
		}

		// Extract host and port
		addr = u.Host

		// Extract password from URL
		if u.User != nil {
			password = u.User.Username()
			if pwd, set := u.User.Password(); set {
				password = pwd
			}
		}

		// Extract database number from path
		if u.Path != "" && u.Path != "/" {
			dbStr := strings.TrimPrefix(u.Path, "/")
			if dbNum, parseErr := strconv.Atoi(dbStr); parseErr == nil {
				db = dbNum
			}
		}

		log.Printf("Parsed Redis URL - Host: %s, DB: %d, Auth: %t", addr, db, password != "")
		return
	}

	// Fall back to individual environment variables
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr != "" {
		// Remove quotes if present
		addr = strings.Trim(redisAddr, `"'`)
		log.Printf("Using REDIS_ADDR: %s", addr)
	} else {
		log.Println("REDIS_ADDR not set, defaulting to localhost:6379")
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")
	if redisPassword != "" {
		password = redisPassword
	}

	redisDB := os.Getenv("REDIS_DB")
	if redisDB != "" {
		if dbNum, parseErr := strconv.Atoi(redisDB); parseErr == nil {
			db = dbNum
		}
	}

	return
}

func main() {
	log.Println("Starting League Performance Tracker backend...")

	// Load environment configuration
	loadEnvironmentConfig()

	app.riotAPIKey = os.Getenv("RIOT_API_KEY")
	if app.riotAPIKey == "" {
		log.Fatal("CRITICAL: RIOT_API_KEY environment variable not set.")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
		log.Println("MONGO_URI not set, defaulting to mongodb://localhost:27017")
	}

	app.mongoDatabase = os.Getenv("MONGO_DATABASE")
	if app.mongoDatabase == "" {
		app.mongoDatabase = "leagueperformancetracker"
		log.Println("MONGO_DATABASE not set, defaulting to leagueperformancetracker")
	}

	// Parse Redis configuration
	redisAddr, redisPassword, redisDBNum, err := parseRedisConfig()
	if err != nil {
		log.Fatalf("Error parsing Redis configuration: %v", err)
	}

	if redisPassword != "" {
		log.Println("Redis password authentication enabled")
	} else {
		log.Println("Redis password authentication disabled (no password set)")
	}

	// Add environment configuration summary
	log.Printf("Configuration summary - MongoDB: %s, Redis: %s, Database: %s",
		mongoURI, redisAddr, app.mongoDatabase)

	app.httpClient = &http.Client{Timeout: defaultTimeout}

	app.redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDBNum,
	})
	ctxRedis, cancelRedis := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelRedis()
	if err := app.redisClient.Ping(ctxRedis).Err(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis.")

	ctxMongo, cancelMongo := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelMongo()
	clientOptions := options.Client().ApplyURI(mongoURI)
	var errMongo error
	app.mongoClient, errMongo = mongo.Connect(ctxMongo, clientOptions)
	if errMongo != nil {
		log.Fatalf("Could not connect to MongoDB: %v", errMongo)
	}
	if err := app.mongoClient.Ping(ctxMongo, readpref.Primary()); err != nil {
		log.Fatalf("Could not ping MongoDB: %v", err)
	}
	log.Println("Successfully connected to MongoDB.")

	log.Println("Initiating population of static data...")
	if err := populateStaticData(&app); err != nil {
		log.Fatalf("CRITICAL: Failed to populate static data on startup: %v. Application cannot start correctly.", err)
	} else {
		log.Println("Static data population complete. All static data is preloaded and cached in memory.")
	}

	r := mux.NewRouter()

	r.Use(corsMiddleware)

	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(corsMiddleware)

	apiRouter.HandleFunc("/health", healthCheckHandler).Methods("GET", "OPTIONS")
	apiRouter.HandleFunc("/player/{region}/{gameName}/{tagLine}/matches", getPlayerPerformanceHandler(&app)).Methods("GET", "OPTIONS")
	apiRouter.HandleFunc("/player/{region}/{gameName}/{tagLine}/summary", getRecentGamesSummaryHandler(&app)).Methods("GET", "OPTIONS")
	apiRouter.HandleFunc("/static-data", getStaticDataHandler(&app)).Methods("GET", "OPTIONS")
	apiRouter.HandleFunc("/match/{region}/{matchId}", getMatchDetailsHandler(&app)).Methods("GET", "OPTIONS")

	apiRouter.HandleFunc("/popular-items", getPopularItemsHandler(&app)).Methods("GET", "OPTIONS")

	log.Println("Backend server starting on :8080")
	srv := &http.Server{
		Handler:      handlers.CompressHandler(r),
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on :8080: %v\n", err)
	}
	log.Println("Backend server stopped.")
}
