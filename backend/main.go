package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var app GlobalAppData

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
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

func main() {
	log.Println("Starting League Performance Tracker backend...")

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Configuration
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

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
		log.Println("REDIS_ADDR not set, defaulting to localhost:6379")
	}

	// Initialize HTTP client
	app.httpClient = &http.Client{Timeout: defaultTimeout}

	// Initialize Redis Client
	app.redisClient = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	ctxRedis, cancelRedis := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelRedis()
	if err := app.redisClient.Ping(ctxRedis).Err(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis.")

	// Initialize MongoDB Client
	ctxMongo, cancelMongo := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelMongo()
	clientOptions := options.Client().ApplyURI(mongoURI)
	var errMongo error
	app.mongoClient, errMongo = mongo.Connect(ctxMongo, clientOptions)
	if errMongo != nil {
		log.Fatalf("Could not connect to MongoDB: %v", errMongo)
	}
	// Ping MongoDB
	if err := app.mongoClient.Ping(ctxMongo, readpref.Primary()); err != nil {
		log.Fatalf("Could not ping MongoDB: %v", err)
	}
	log.Println("Successfully connected to MongoDB.")

	// Populate static data (champions, items, etc.) on startup.
	// This is now a blocking call to ensure data is ready before server starts.
	log.Println("Initiating population of static data...")
	if err := populateStaticData(&app); err != nil {
		log.Fatalf("CRITICAL: Failed to populate static data on startup: %v. Application cannot start correctly.", err)
	} else {
		log.Println("Static data population complete. All static data is preloaded and cached in memory.")
	}

	// Setup Router
	r := mux.NewRouter()

	// Apply CORS middleware to all routes
	r.Use(corsMiddleware)

	apiRouter := r.PathPrefix("/api").Subrouter()
	// Apply CORS middleware to API routes as well to ensure it's not lost in subrouter
	apiRouter.Use(corsMiddleware)

	// API Endpoints
	apiRouter.HandleFunc("/health", healthCheckHandler).Methods("GET", "OPTIONS")
	apiRouter.HandleFunc("/player/{region}/{gameName}/{tagLine}/matches", getPlayerPerformanceHandler(&app)).Methods("GET", "OPTIONS")
	apiRouter.HandleFunc("/static-data", getStaticDataHandler(&app)).Methods("GET", "OPTIONS")
	apiRouter.HandleFunc("/match/{region}/{matchId}", getMatchDetailsHandler(&app)).Methods("GET", "OPTIONS")

	// Endpoint for popular items, for frontend preloading
	apiRouter.HandleFunc("/popular-items", getPopularItemsHandler(&app)).Methods("GET", "OPTIONS")

	// Serve frontend (React build) - This needs to be configured after `npm run build` in frontend
	// For development, React's dev server (npm start) will handle serving the frontend.
	// For production, you would serve the static files from `frontend/build`.
	// Example (you might need to adjust the path based on your final setup):
	// r.PathPrefix("/").Handler(http.FileServer(http.Dir("../frontend/build/")))

	log.Println("Backend server starting on :8080")
	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on :8080: %v\n", err)
	}
	log.Println("Backend server stopped.")
}
