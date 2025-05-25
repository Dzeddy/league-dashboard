package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/handlers"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Optimized HTTP transport for Riot API requests with connection reuse and TLS optimization
var riotTransport = &http.Transport{
	// Keep a handful of connections to each Riot edge-node alive
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 100,
	IdleConnTimeout:     90 * time.Second,
	// TLS handshakes are expensive; enable session resumption & HTTP/2
	TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
	ForceAttemptHTTP2: true,
}

// Optimized HTTP client for Riot API requests
var riotHTTP = &http.Client{
	Transport: riotTransport,
	Timeout:   10 * time.Second, // Using the same timeout as defaultTimeout
}

var app GlobalAppData

func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:3000":                     true,
		"https://localhost:3000":                    true,
		"https://dzeddy.github.io":                  true,
		"https://league-dashboard-eosin.vercel.app": true,
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

// generateSelfSignedCert generates a self-signed certificate for development
func generateSelfSignedCert(certFile, keyFile string) error {
	// Generate a new private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"League Dashboard"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:    []string{"localhost", "*.localhost"},
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %v", err)
	}

	// Write certificate to file
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to open cert.pem for writing: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("failed to write certificate: %v", err)
	}

	// Write private key to file
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return fmt.Errorf("failed to open key.pem for writing: %v", err)
	}
	defer keyOut.Close()

	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %v", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privKeyBytes}); err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}

	log.Printf("Generated self-signed certificate: %s and %s", certFile, keyFile)
	return nil
}

// ensureSSLCerts ensures SSL certificates exist, generating self-signed ones if needed
func ensureSSLCerts() (string, string, error) {
	certFile := os.Getenv("SSL_CERT_FILE")
	keyFile := os.Getenv("SSL_KEY_FILE")

	// Use default paths if not specified
	if certFile == "" {
		certFile = "server.crt"
	}
	if keyFile == "" {
		keyFile = "server.key"
	}

	// Make paths absolute
	certFile, _ = filepath.Abs(certFile)
	keyFile, _ = filepath.Abs(keyFile)

	// Check if both certificate files exist
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		log.Println("SSL certificate not found, generating self-signed certificate...")
		if err := generateSelfSignedCert(certFile, keyFile); err != nil {
			return "", "", err
		}
	} else if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		log.Println("SSL private key not found, generating self-signed certificate...")
		if err := generateSelfSignedCert(certFile, keyFile); err != nil {
			return "", "", err
		}
	} else {
		log.Printf("Using existing SSL certificate: %s", certFile)
	}

	return certFile, keyFile, nil
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

// createIndexes creates MongoDB indexes for optimal query performance
func createIndexes(client *mongo.Client, database string) error {
	collection := client.Database(database).Collection("userperformances")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "_id", Value: 1},
				{Key: "region", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "updatedAt", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "matches.matchId", Value: 1},
			},
		},
	}

	_, err := collection.Indexes().CreateMany(context.Background(), indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %v", err)
	}

	log.Println("Successfully created MongoDB indexes for userperformances collection")
	return nil
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

	// Use the optimized HTTP client with connection reuse and TLS optimization
	app.httpClient = riotHTTP

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

	// Create MongoDB indexes for optimal query performance
	if err := createIndexes(app.mongoClient, app.mongoDatabase); err != nil {
		log.Printf("Warning: Failed to create MongoDB indexes: %v", err)
	}

	log.Println("Initiating population of static data...")
	if err := populateStaticData(&app); err != nil {
		log.Fatalf("CRITICAL: Failed to populate static data on startup: %v. Application cannot start correctly.", err)
	} else {
		log.Println("Static data population complete. All static data is preloaded and cached in memory.")
	}

	r := chi.NewRouter()

	r.Use(corsMiddleware)

	r.Route("/api", func(api chi.Router) {
		api.Options("/*", func(w http.ResponseWriter, r *http.Request) {
			// CORS preflight response is handled by corsMiddleware
			w.WriteHeader(http.StatusOK)
		})

		api.Get("/health", healthCheckHandler)

		// New consolidated dashboard endpoint that combines matches and summary
		api.Get("/player/{region}/{gameName}/{tagLine}/dashboard", getPlayerDashboardHandler(&app))

		// Legacy endpoints (kept for backward compatibility during transition)
		api.Get("/player/{region}/{gameName}/{tagLine}/matches", getPlayerPerformanceHandler(&app))
		api.Get("/player/{region}/{gameName}/{tagLine}/summary", getRecentGamesSummaryHandler(&app))

		api.Get("/static-data", getStaticDataHandler(&app))
		api.Get("/match/{region}/{matchId}", getMatchDetailsHandler(&app))

		api.Get("/popular-items", getPopularItemsHandler(&app))
	})

	// Check if SSL should be enabled (default: true)
	useSSL := os.Getenv("USE_SSL")
	if useSSL == "" {
		useSSL = "true" // Default to SSL enabled
	}

	// Get port configuration
	port := os.Getenv("PORT")
	if port == "" {
		if useSSL == "true" {
			port = "8443" // Default HTTPS port
		} else {
			port = "8080" // Default HTTP port
		}
	}

	srv := &http.Server{
		Handler:      handlers.CompressHandler(r),
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if useSSL == "true" {
		// Set up SSL certificates
		certFile, keyFile, err := ensureSSLCerts()
		if err != nil {
			log.Fatalf("Failed to set up SSL certificates: %v", err)
		}

		// Configure TLS
		tlsConfig := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		}
		srv.TLSConfig = tlsConfig

		log.Printf("Starting HTTPS server on :%s", port)
		log.Printf("Using SSL certificate: %s", certFile)

		if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start HTTPS server on :%s: %v\n", port, err)
		}
	} else {
		log.Printf("Starting HTTP server on :%s", port)
		log.Println("WARNING: SSL is disabled. This should only be used for development!")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start HTTP server on :%s: %v\n", port, err)
		}
	}

	log.Println("Backend server stopped.")
}
