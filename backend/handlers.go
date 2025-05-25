package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
)

func getPlayerPerformanceHandler(app *GlobalAppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		region := chi.URLParam(r, "region")
		gameName := chi.URLParam(r, "gameName")
		tagLine := chi.URLParam(r, "tagLine")

		countStr := r.URL.Query().Get("count")
		queueIDStr := r.URL.Query().Get("queueId")

		// Validate and sanitize input parameters
		validatedGameName, validatedTagLine, validatedRegion, err := ValidateAndSanitizeInput(gameName, tagLine, region)
		if err != nil {
			log.Printf("Input validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid input: %v", err), http.StatusBadRequest)
			return
		}

		// Additional NoSQL injection prevention
		if err := PreventNoSQLInjection(validatedGameName); err != nil {
			log.Printf("Potential NoSQL injection attempt in gameName: %s", validatedGameName)
			http.Error(w, "Invalid input detected", http.StatusBadRequest)
			return
		}
		if err := PreventNoSQLInjection(validatedTagLine); err != nil {
			log.Printf("Potential NoSQL injection attempt in tagLine: %s", validatedTagLine)
			http.Error(w, "Invalid input detected", http.StatusBadRequest)
			return
		}

		// Validate count parameter
		count, err := ValidateCount(countStr, defaultMatchCount, 100)
		if err != nil {
			log.Printf("Count validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid count parameter: %v", err), http.StatusBadRequest)
			return
		}

		// Validate queueID parameter
		queueID, err := ValidateQueueID(queueIDStr, defaultQueueID)
		if err != nil {
			log.Printf("QueueID validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid queueId parameter: %v", err), http.StatusBadRequest)
			return
		}

		log.Printf("Handler: Received player performance request for %s#%s in region %s, count: %d, queueId: %d", validatedGameName, validatedTagLine, validatedRegion, count, queueID)

		if app.riotAPIKey == "" {
			log.Println("Error: RIOT_API_KEY is not set.")
			http.Error(w, "Server configuration error: Riot API Key not set.", http.StatusInternalServerError)
			return
		}

		if app.staticData == nil {
			log.Println("Static data not yet loaded, attempting to load now.")
			err := populateStaticData(app)
			if err != nil {
				log.Printf("Error populating static data on demand: %v", err)
				http.Error(w, "Error loading required game data. Please try again shortly.", http.StatusInternalServerError)
				return
			}
		}

		performance, err := fetchAndStoreUserPerformance(app, validatedRegion, validatedGameName, validatedTagLine, count, queueID, 0)
		if err != nil {
			log.Printf("Error fetching user performance for %s#%s: %v", validatedGameName, validatedTagLine, err)
			http.Error(w, fmt.Sprintf("Error fetching user performance: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(performance); err != nil {
			log.Printf("Error encoding response for %s#%s: %v", validatedGameName, validatedTagLine, err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func getStaticDataHandler(app *GlobalAppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if app.staticData == nil {
			log.Println("Static data requested but not loaded yet.")
			err := populateStaticData(app)
			if err != nil {
				log.Printf("Error populating static data on demand for /static-data: %v", err)
				http.Error(w, "Static data is not available at the moment, please try again later.", http.StatusServiceUnavailable)
				return
			}
		}

		response := struct {
			Champions      map[string]ChampionData      `json:"champions"`
			Items          map[string]ItemData          `json:"items"`
			Runes          map[int]RuneInfo             `json:"runes"`
			SummonerSpells map[string]SummonerSpellData `json:"summonerSpells"`
			LatestVersion  string                       `json:"latestVersion"`
		}{
			Champions:      app.staticData.Champions,
			Items:          app.staticData.Items,
			Runes:          app.staticData.Runes,
			SummonerSpells: app.staticData.SummonerSpells,
			LatestVersion:  app.staticData.LatestVersion,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding static data response: %v", err)
			http.Error(w, "Failed to encode static data response", http.StatusInternalServerError)
		}
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{"status": "ok", "timestamp": time.Now().String()}
	json.NewEncoder(w).Encode(response)
	log.Println("Health check performed.")
}

func getMatchDetailsHandler(app *GlobalAppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		region := chi.URLParam(r, "region")
		matchId := chi.URLParam(r, "matchId")

		// Validate and sanitize input parameters
		validatedRegion, validatedMatchId, err := ValidateMatchInput(region, matchId)
		if err != nil {
			log.Printf("Input validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid input: %v", err), http.StatusBadRequest)
			return
		}

		// Additional NoSQL injection prevention
		if err := PreventNoSQLInjection(validatedMatchId); err != nil {
			log.Printf("Potential NoSQL injection attempt in matchId: %s", validatedMatchId)
			http.Error(w, "Invalid input detected", http.StatusBadRequest)
			return
		}

		log.Printf("Handler: Received match details request for match %s in region %s", validatedMatchId, validatedRegion)

		if app.riotAPIKey == "" {
			log.Println("Error: RIOT_API_KEY is not set.")
			http.Error(w, "Server configuration error: Riot API Key not set.", http.StatusInternalServerError)
			return
		}

		match, err := getMatchDetails(app, validatedRegion, validatedMatchId)
		if err != nil {
			log.Printf("Error fetching match details for %s: %v", validatedMatchId, err)
			http.Error(w, "Error fetching match details", http.StatusInternalServerError)
			return
		}
		if match == nil {
			http.Error(w, "Match not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(match)
	}
}

const popularItemsCacheKey = "popular_items_v1"
const popularItemsCacheTTL = 24 * time.Hour
const topNPopularItems = 50

type popularItemDBResult struct {
	ItemID int `bson:"_id"`
	Count  int `bson:"count"`
}

func fetchTopPopularItemIDsFromDB(app *GlobalAppData, count int) ([]int, error) {
	log.Printf("Database: Fetching top %d popular item IDs from MongoDB.", count)

	collection := app.mongoClient.Database(app.mongoDatabase).Collection("userperformances")

	pipeline := []bson.M{
		{"$unwind": "$matches"},
		{"$unwind": "$matches.items"},
		{"$match": bson.M{
			"matches.items": bson.M{"$ne": 0},
		}},
		{"$group": bson.M{
			"_id":   "$matches.items",
			"count": bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"count": -1}},
		{"$limit": count},
		{"$project": bson.M{
			"_id": 1,
		}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("MongoDB aggregation error: %v", err)
		return nil, fmt.Errorf("failed to execute popular items aggregation: %w", err)
	}
	defer cursor.Close(ctx)

	var results []popularItemDBResult
	if err = cursor.All(ctx, &results); err != nil {
		log.Printf("Error decoding aggregation results: %v", err)
		return nil, fmt.Errorf("failed to decode popular items results: %w", err)
	}

	itemIDs := make([]int, 0, len(results))
	for _, result := range results {
		itemIDs = append(itemIDs, result.ItemID)
	}

	if len(itemIDs) == 0 {
		log.Println("No popular items found from DB after aggregation.")
		return []int{}, nil
	}

	log.Printf("Successfully fetched %d popular item IDs from DB.", len(itemIDs))
	return itemIDs, nil
}

func getPopularItemsHandler(app *GlobalAppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		cachedItemsJSON, err := app.redisClient.Get(ctx, popularItemsCacheKey).Result()
		if err == nil {
			var popularItemIDs []int
			if err := json.Unmarshal([]byte(cachedItemsJSON), &popularItemIDs); err == nil {
				log.Println("Cache hit for popular items.")
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(popularItemIDs)
				return
			}
			log.Printf("Error unmarshalling popular items from Redis: %v. Proceeding to fetch from DB.", err)
		} else if err != redis.Nil {
			log.Printf("Error fetching popular items from Redis: %v. Proceeding to fetch from DB.", err)
		} else {
			log.Println("Cache miss for popular items. Fetching from DB.")
		}

		itemIDs, dbErr := fetchTopPopularItemIDsFromDB(app, topNPopularItems)
		if dbErr != nil {
			log.Printf("Error fetching popular items from DB: %v", dbErr)
			http.Error(w, "Failed to fetch popular items.", http.StatusInternalServerError)
			return
		}

		if len(itemIDs) == 0 {
			log.Println("No popular items found from DB (or placeholder returned empty).")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]int{}) // Return empty list
			return
		}

		newCachedItemsJSON, err := json.Marshal(itemIDs)
		if err != nil {
			log.Printf("Error marshalling popular items for Redis cache: %v", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(itemIDs)
			return
		}

		// Move Redis caching off the critical path - run asynchronously
		go func(cacheData []byte) {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = app.redisClient.Set(cacheCtx, popularItemsCacheKey, cacheData, popularItemsCacheTTL).Err()
		}(newCachedItemsJSON)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(itemIDs)
	}
}

func getRecentGamesSummaryHandler(app *GlobalAppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		region := chi.URLParam(r, "region")
		gameName := chi.URLParam(r, "gameName")
		tagLine := chi.URLParam(r, "tagLine")

		countStr := r.URL.Query().Get("count")
		queueIDStr := r.URL.Query().Get("queueId")

		// Validate and sanitize input parameters
		validatedGameName, validatedTagLine, validatedRegion, err := ValidateAndSanitizeInput(gameName, tagLine, region)
		if err != nil {
			log.Printf("Input validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid input: %v", err), http.StatusBadRequest)
			return
		}

		// Additional NoSQL injection prevention
		if err := PreventNoSQLInjection(validatedGameName); err != nil {
			log.Printf("Potential NoSQL injection attempt in gameName: %s", validatedGameName)
			http.Error(w, "Invalid input detected", http.StatusBadRequest)
			return
		}
		if err := PreventNoSQLInjection(validatedTagLine); err != nil {
			log.Printf("Potential NoSQL injection attempt in tagLine: %s", validatedTagLine)
			http.Error(w, "Invalid input detected", http.StatusBadRequest)
			return
		}

		// Validate count parameter
		count, err := ValidateCount(countStr, defaultMatchCount, 100)
		if err != nil {
			log.Printf("Count validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid count parameter: %v", err), http.StatusBadRequest)
			return
		}

		// Validate queueID parameter
		queueID, err := ValidateQueueID(queueIDStr, defaultQueueID)
		if err != nil {
			log.Printf("QueueID validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid queueId parameter: %v", err), http.StatusBadRequest)
			return
		}

		log.Printf("Handler: Received recent games summary request for %s#%s in region %s, count: %d, queueId: %d", validatedGameName, validatedTagLine, validatedRegion, count, queueID)

		if app.riotAPIKey == "" {
			log.Println("Error: RIOT_API_KEY is not set.")
			http.Error(w, "Server configuration error: Riot API Key not set.", http.StatusInternalServerError)
			return
		}

		if app.staticData == nil {
			log.Println("Static data not yet loaded, attempting to load now.")
			err := populateStaticData(app)
			if err != nil {
				log.Printf("Error populating static data on demand: %v", err)
				http.Error(w, "Error loading required game data. Please try again shortly.", http.StatusInternalServerError)
				return
			}
		}

		summaryData, err := fetchRecentGamesSummary(app, validatedRegion, validatedGameName, validatedTagLine, count, queueID)
		if err != nil {
			log.Printf("Error fetching recent games summary for %s#%s: %v", validatedGameName, validatedTagLine, err)
			http.Error(w, fmt.Sprintf("Error fetching recent games summary: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summaryData); err != nil {
			log.Printf("Error encoding response for %s#%s: %v", validatedGameName, validatedTagLine, err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func getPlayerDashboardHandler(app *GlobalAppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		region := chi.URLParam(r, "region")
		gameName := chi.URLParam(r, "gameName")
		tagLine := chi.URLParam(r, "tagLine")

		countStr := r.URL.Query().Get("count")
		queueIDStr := r.URL.Query().Get("queueId")
		offsetStr := r.URL.Query().Get("offset")

		// Input validation
		validatedGameName, validatedTagLine, validatedRegion, err := ValidateAndSanitizeInput(gameName, tagLine, region)
		if err != nil {
			log.Printf("Input validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid input: %v", err), http.StatusBadRequest)
			return
		}

		// NoSQL injection prevention
		if err := PreventNoSQLInjection(validatedGameName); err != nil {
			log.Printf("Potential NoSQL injection attempt in gameName: %s", validatedGameName)
			http.Error(w, "Invalid input detected", http.StatusBadRequest)
			return
		}
		if err := PreventNoSQLInjection(validatedTagLine); err != nil {
			log.Printf("Potential NoSQL injection attempt in tagLine: %s", validatedTagLine)
			http.Error(w, "Invalid input detected", http.StatusBadRequest)
			return
		}

		// Parameter validation
		count, err := ValidateCount(countStr, defaultMatchCount, 100)
		if err != nil {
			log.Printf("Count validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid count parameter: %v", err), http.StatusBadRequest)
			return
		}

		queueID, err := ValidateQueueID(queueIDStr, defaultQueueID)
		if err != nil {
			log.Printf("QueueID validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid queueId parameter: %v", err), http.StatusBadRequest)
			return
		}

		offset, err := ValidateOffset(offsetStr, 0)
		if err != nil {
			log.Printf("Offset validation error: %v", err)
			http.Error(w, fmt.Sprintf("Invalid offset parameter: %v", err), http.StatusBadRequest)
			return
		}

		log.Printf("Handler: Received player dashboard request for %s#%s in region %s, count: %d, queueId: %d, offset: %d", validatedGameName, validatedTagLine, validatedRegion, count, queueID, offset)

		if app.riotAPIKey == "" {
			log.Println("Error: RIOT_API_KEY is not set.")
			http.Error(w, "Server configuration error: Riot API Key not set.", http.StatusInternalServerError)
			return
		}

		if app.staticData == nil {
			log.Println("Static data not yet loaded, attempting to load now.")
			err := populateStaticData(app)
			if err != nil {
				log.Printf("Error populating static data on demand: %v", err)
				http.Error(w, "Error loading required game data. Please try again shortly.", http.StatusInternalServerError)
				return
			}
		}

		// Fetch user performance
		userPerformance, err := fetchAndStoreUserPerformance(app, validatedRegion, validatedGameName, validatedTagLine, count, queueID, offset)
		if err != nil {
			log.Printf("Error fetching user performance for %s#%s: %v", validatedGameName, validatedTagLine, err)
			http.Error(w, fmt.Sprintf("Error fetching user performance: %v", err), http.StatusInternalServerError)
			return
		}

		// Calculate incremental stats for the returned matches
		incrementalStats := calculateIncrementalStats(userPerformance.Matches)

		// Prepare pagination info
		hasMore := len(userPerformance.Matches) == count
		pagination := PaginationInfo{
			Offset:  offset,
			Limit:   count,
			Total:   -1, // We don't know the total
			HasMore: hasMore,
		}

		// Prepare response
		var dashboardData PaginatedDashboardResponse

		if offset == 0 {
			// First page: include full summary
			summary := calculateRecentGamesSummary(userPerformance.Matches, userPerformance.PUUID, userPerformance.Region, userPerformance.RiotID)
			dashboardData = PaginatedDashboardResponse{
				Summary:          summary,
				Matches:          userPerformance.Matches,
				Pagination:       pagination,
				IncrementalStats: incrementalStats,
			}
		} else {
			// Subsequent pages: no summary, just matches and incremental stats
			dashboardData = PaginatedDashboardResponse{
				Summary:          nil,
				Matches:          userPerformance.Matches,
				Pagination:       pagination,
				IncrementalStats: incrementalStats,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(dashboardData); err != nil {
			log.Printf("Error encoding dashboard response for %s#%s: %v", validatedGameName, validatedTagLine, err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}
