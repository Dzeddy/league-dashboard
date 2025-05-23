package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
)

func getPlayerPerformanceHandler(app *GlobalAppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		region := vars["region"]
		gameName := vars["gameName"]
		tagLine := vars["tagLine"]

		countStr := r.URL.Query().Get("count")
		queueIDStr := r.URL.Query().Get("queueId")

		count := defaultMatchCount
		if countStr != "" {
			c, err := strconv.Atoi(countStr)
			if err == nil && c > 0 && c <= 100 { // 100 max
				count = c
			} else if err != nil {
				http.Error(w, "Invalid 'count' parameter", http.StatusBadRequest)
				return
			}
		}

		queueID := defaultQueueID
		if queueIDStr != "" {
			q, err := strconv.Atoi(queueIDStr)
			if err == nil {
				queueID = q
			} else {
				http.Error(w, "Invalid 'queueId' parameter", http.StatusBadRequest)
				return
			}
		}

		log.Printf("Handler: Received request for %s#%s in region %s, count: %d, queueId: %d", gameName, tagLine, region, count, queueID)

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

		performanceData, err := fetchAndStoreUserPerformance(app, region, gameName, tagLine, count, queueID)
		if err != nil {
			log.Printf("Error fetching user performance for %s#%s: %v", gameName, tagLine, err)
			http.Error(w, fmt.Sprintf("Error fetching player data: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(performanceData); err != nil {
			log.Printf("Error encoding response for %s#%s: %v", gameName, tagLine, err)
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
		vars := mux.Vars(r)
		region := vars["region"]
		matchId := vars["matchId"]

		if app.riotAPIKey == "" {
			log.Println("Error: RIOT_API_KEY is not set.")
			http.Error(w, "Server configuration error: Riot API Key not set.", http.StatusInternalServerError)
			return
		}

		match, err := getMatchDetails(app, region, matchId)
		if err != nil {
			log.Printf("Error fetching match details for %s: %v", matchId, err)
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

		err = app.redisClient.Set(ctx, popularItemsCacheKey, newCachedItemsJSON, popularItemsCacheTTL).Err()
		if err != nil {
			log.Printf("Error setting popular items in Redis cache: %v", err)
		} else {
			log.Printf("Successfully cached %d popular items.", len(itemIDs))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(itemIDs)
	}
}

func getRecentGamesSummaryHandler(app *GlobalAppData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		region := vars["region"]
		gameName := vars["gameName"]
		tagLine := vars["tagLine"]

		countStr := r.URL.Query().Get("count")
		queueIDStr := r.URL.Query().Get("queueId")

		count := defaultMatchCount
		if countStr != "" {
			c, err := strconv.Atoi(countStr)
			if err == nil && c > 0 && c <= 100 { // 100 max
				count = c
			} else if err != nil {
				http.Error(w, "Invalid 'count' parameter", http.StatusBadRequest)
				return
			}
		}

		queueID := defaultQueueID
		if queueIDStr != "" {
			q, err := strconv.Atoi(queueIDStr)
			if err == nil {
				queueID = q
			} else {
				http.Error(w, "Invalid 'queueId' parameter", http.StatusBadRequest)
				return
			}
		}

		log.Printf("Handler: Received recent games summary request for %s#%s in region %s, count: %d, queueId: %d", gameName, tagLine, region, count, queueID)

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

		summaryData, err := fetchRecentGamesSummary(app, region, gameName, tagLine, count, queueID)
		if err != nil {
			log.Printf("Error fetching recent games summary for %s#%s: %v", gameName, tagLine, err)
			http.Error(w, fmt.Sprintf("Error fetching recent games summary: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summaryData); err != nil {
			log.Printf("Error encoding response for %s#%s: %v", gameName, tagLine, err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}
