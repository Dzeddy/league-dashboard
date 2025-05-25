package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/sync/errgroup"
)

const (
	puuidCacheDuration           = 24 * time.Hour
	matchListCacheDuration       = 1 * time.Hour
	matchDetailsCacheDuration    = 7 * 24 * time.Hour
	userPerformanceCacheDuration = 30 * time.Minute
	staticDataCacheDuration      = 24 * time.Hour
	defaultTimeout               = 10 * time.Second
	defaultMatchCount            = 25
	defaultQueueID               = 0
	defaultConcurrencyLimit      = 20 // Tunable concurrency limit for match fetching
	dataDragonBaseURL            = "https://ddragon.leagueoflegends.com"
)

func getPUUID(app *GlobalAppData, region, gameName, tagLine string) (string, error) {
	apiRegion := getAPIRegion(region)
	cacheKey := fmt.Sprintf("puuid:%s:%s:%s", apiRegion, strings.ToLower(gameName), strings.ToLower(tagLine))

	val, err := app.redisClient.Get(context.Background(), cacheKey).Result()
	if err == redis.Nil {
		url := fmt.Sprintf("https://%s.api.riotgames.com/riot/account/v1/accounts/by-riot-id/%s/%s", apiRegion, gameName, tagLine)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("X-Riot-Token", app.riotAPIKey)

		resp, err := app.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to make PUUID request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("PUUID request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var acc AccountDTO
		if err := json.NewDecoder(resp.Body).Decode(&acc); err != nil {
			return "", fmt.Errorf("failed to decode PUUID response: %w", err)
		}

		if acc.PUUID == "" {
			return "", fmt.Errorf("PUUID not found for %s#%s in region %s", gameName, tagLine, region)
		}

		// Move Redis caching off the critical path - run asynchronously
		go func(key, value string) {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = app.redisClient.Set(cacheCtx, key, value, puuidCacheDuration).Err()
		}(cacheKey, acc.PUUID)

		return acc.PUUID, nil
	} else if err != nil {
		return "", fmt.Errorf("failed to get PUUID from cache: %w", err)
	}
	return val, nil
}

func getMatchIDs(app *GlobalAppData, region, puuid string, count int, queueID int, startTime int64) ([]string, error) {
	apiRegion := getAPIRegion(region)
	cacheKey := fmt.Sprintf("matchids:%s:%s:%d:q%d:%d", apiRegion, puuid, count, queueID, startTime)

	val, err := app.redisClient.Get(context.Background(), cacheKey).Result()
	if err == redis.Nil {
		url := fmt.Sprintf("https://%s.api.riotgames.com/lol/match/v5/matches/by-puuid/%s/ids?count=%d", apiRegion, puuid, count)
		if queueID != 0 {
			url += fmt.Sprintf("&queue=%d", queueID)
		}
		if startTime > 0 {
			url += fmt.Sprintf("&startTime=%d", startTime)
		}

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("X-Riot-Token", app.riotAPIKey)

		resp, err := app.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make match IDs request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("match IDs request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var matchIDs []string
		if err := json.NewDecoder(resp.Body).Decode(&matchIDs); err != nil {
			return nil, fmt.Errorf("failed to decode match IDs response: %w", err)
		}

		// Move Redis caching off the critical path - run asynchronously
		go func(key string, data []string) {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if dataJSON, err := json.Marshal(data); err == nil {
				_ = app.redisClient.Set(cacheCtx, key, dataJSON, matchListCacheDuration).Err()
			}
		}(cacheKey, matchIDs)

		return matchIDs, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get match IDs from cache: %w", err)
	}

	var matchIDs []string
	if err := json.Unmarshal([]byte(val), &matchIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached match IDs: %w", err)
	}
	return matchIDs, nil
}

func getMatchDetails(app *GlobalAppData, region, matchID string) (*MatchDto, error) {
	apiRegion := getAPIRegion(region)
	cacheKey := fmt.Sprintf("matchdetails:%s:%s", apiRegion, matchID)

	val, err := app.redisClient.Get(context.Background(), cacheKey).Result()
	if err == redis.Nil {
		url := fmt.Sprintf("https://%s.api.riotgames.com/lol/match/v5/matches/%s", apiRegion, matchID)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("X-Riot-Token", app.riotAPIKey)

		resp, err := app.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make match details request for %s: %w", matchID, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			if resp.StatusCode == http.StatusNotFound {
				log.Printf("Match %s not found in region %s, skipping.", matchID, apiRegion)
				return nil, nil
			}
			return nil, fmt.Errorf("match details request for %s failed with status %d: %s", matchID, resp.StatusCode, string(bodyBytes))
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read match details response body for %s: %w", matchID, err)
		}

		var match MatchDto
		if err := json.Unmarshal(bodyBytes, &match); err != nil {
			return nil, fmt.Errorf("failed to decode match details response for %s: %w", matchID, err)
		}

		// Move Redis caching off the critical path - run asynchronously
		go func(key string, data []byte) {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = app.redisClient.Set(cacheCtx, key, string(data), matchDetailsCacheDuration).Err()
		}(cacheKey, bodyBytes)

		return &match, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get match details for %s from cache: %w", matchID, err)
	}

	var match MatchDto
	if err := json.Unmarshal([]byte(val), &match); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached match details for %s: %w", matchID, err)
	}
	return &match, nil
}

func extractPlayerMatchStats(matchData *MatchDto, playerPUUID string, app *GlobalAppData) (*PlayerMatchStats, error) {
	if matchData == nil || matchData.Info.Participants == nil {
		return nil, fmt.Errorf("matchData or participants list is nil for match %s", matchData.Metadata.MatchID)
	}

	var playerParticipant *ParticipantDto
	for i, p := range matchData.Info.Participants {
		if p.PUUID == playerPUUID {
			playerParticipant = &matchData.Info.Participants[i]
			break
		}
	}

	if playerParticipant == nil {
		return nil, fmt.Errorf("player PUUID %s not found in match %s participants", playerPUUID, matchData.Metadata.MatchID)
	}

	kda := 0.0
	if playerParticipant.Deaths > 0 {
		kda = float64(playerParticipant.Kills+playerParticipant.Assists) / float64(playerParticipant.Deaths)
	} else {
		kda = float64(playerParticipant.Kills + playerParticipant.Assists)
	}

	killParticipation := 0.0
	if playerParticipant.Challenges != nil {
		killParticipation = playerParticipant.Challenges.KillParticipation
	}

	championName := playerParticipant.ChampionName
	if championName == "" && app.staticData != nil && app.staticData.Champions != nil {
		if champData, ok := app.staticData.Champions[strconv.Itoa(playerParticipant.ChampionID)]; ok {
			championName = champData.Name
		}
	}

	stats := &PlayerMatchStats{
		MatchID:      matchData.Metadata.MatchID,
		GameMode:     matchData.Info.GameMode,
		GameCreation: matchData.Info.GameCreation,
		GameDuration: matchData.Info.GameDuration,
		ChampionName: championName, ChampionID: playerParticipant.ChampionID,
		Win:               playerParticipant.Win,
		Kills:             playerParticipant.Kills,
		Deaths:            playerParticipant.Deaths,
		Assists:           playerParticipant.Assists,
		KDA:               kda,
		KillParticipation: killParticipation, TotalMinionsKilled: playerParticipant.TotalMinionsKilled + playerParticipant.NeutralMinionsKilled,
		VisionScore:        playerParticipant.VisionScore,
		GoldEarned:         playerParticipant.GoldEarned,
		TeamPosition:       playerParticipant.TeamPosition,
		Items:              []int{playerParticipant.Item0, playerParticipant.Item1, playerParticipant.Item2, playerParticipant.Item3, playerParticipant.Item4, playerParticipant.Item5, playerParticipant.Item6},
		SummonerSpells:     []int{playerParticipant.Summoner1Id, playerParticipant.Summoner2Id},
		ChampLevel:         playerParticipant.ChampLevel,
		DamageToTurrets:    playerParticipant.DamageDealtToTurrets,
		DamageToObjectives: playerParticipant.DamageDealtToObjectives,
		DamageToChampions:  playerParticipant.TotalDamageDealtToChampions,
		TotalDamageTaken:   playerParticipant.TotalDamageTaken,
		TeamID:             playerParticipant.TeamID,
		QueueID:            matchData.Info.QueueID,
	}

	if playerParticipant.Perks != nil && len(playerParticipant.Perks.Styles) > 0 {
		for _, style := range playerParticipant.Perks.Styles {
			if style.Description == "primaryStyle" && len(style.Selections) > 0 {
				stats.PrimaryRune = style.Selections[0].Perk
			}
			if style.Description == "subStyle" {
				stats.SecondaryStyle = style.Style
			}
		}
	}

	return stats, nil
}

// getConcurrencyLimit returns the concurrency limit for match fetching,
// checking environment variable first, then falling back to default
func getConcurrencyLimit() int {
	if limitStr := os.Getenv("MATCH_FETCH_CONCURRENCY"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			return limit
		}
	}
	return defaultConcurrencyLimit
}

// fetchMatchesConcurrently fetches match details concurrently using errgroup with tunable concurrency
func fetchMatchesConcurrently(app *GlobalAppData, region string, ids []string, puuid string) []PlayerMatchStats {
	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(getConcurrencyLimit()) // tune until you hit Riot's global rate-limit

	results := make([]PlayerMatchStats, len(ids))
	for i, id := range ids {
		i, id := i, id // capture loop variables
		g.Go(func() error {
			match, err := getMatchDetails(app, region, id)
			if err != nil || match == nil {
				return err // auto-propagate error for cancellation
			}
			stats, err := extractPlayerMatchStats(match, puuid, app)
			if err != nil {
				return err
			}
			if stats != nil {
				results[i] = *stats
			}
			return nil
		})
	}
	_ = g.Wait() // ignore err: partial data is ok

	// Filter out empty results (from failed fetches)
	var matches []PlayerMatchStats
	for _, result := range results {
		if result.MatchID != "" { // Check if the result is valid
			matches = append(matches, result)
		}
	}

	// Sort matches by game creation time
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].GameCreation > matches[j].GameCreation
	})

	return matches
}

func fetchAndStoreUserPerformance(app *GlobalAppData, userRegion, gameName, tagLine string, count, queueID int) (*UserPerformance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout*time.Duration(count+5))
	defer cancel()

	puuid, err := getPUUID(app, userRegion, gameName, tagLine)
	if err != nil {
		return nil, fmt.Errorf("error getting PUUID: %w", err)
	}

	collection := app.mongoClient.Database(app.mongoDatabase).Collection("userperformances")
	var cachedPerformance UserPerformance
	cacheKeyDB := fmt.Sprintf("%s_%s", userRegion, puuid)
	redisCacheKey := fmt.Sprintf("userperformance:%s", cacheKeyDB)
	val, err := app.redisClient.Get(ctx, redisCacheKey).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(val), &cachedPerformance); err == nil {
			log.Printf("User performance for %s loaded from Redis cache.", puuid)
			if len(cachedPerformance.Matches) >= count {
				if len(cachedPerformance.Matches) > count {
					trimmed := *&cachedPerformance
					trimmed.Matches = cachedPerformance.Matches[:count]
					return &trimmed, nil
				}
				return &cachedPerformance, nil
			}
			log.Printf("Redis cache hit, but not enough matches (%d < %d), will fetch fresh data.", len(cachedPerformance.Matches), count)
		}
		log.Printf("Error unmarshalling user performance from Redis, will fetch: %v", err)
	}

	err = collection.FindOne(ctx, bson.M{"_id": puuid, "region": userRegion}).Decode(&cachedPerformance)
	if err == nil && len(cachedPerformance.Matches) >= count && time.Now().Unix()-cachedPerformance.UpdatedAt < int64(userPerformanceCacheDuration/time.Second)/2 {
		log.Printf("User performance for %s loaded from MongoDB.", puuid)

		// Move Redis caching off the critical path - run asynchronously
		go func(key string, data UserPerformance) {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if perfJSON, err := json.Marshal(data); err == nil {
				_ = app.redisClient.Set(cacheCtx, key, perfJSON, userPerformanceCacheDuration).Err()
			}
		}(redisCacheKey, cachedPerformance)

		if len(cachedPerformance.Matches) > count {
			trimmed := *&cachedPerformance
			trimmed.Matches = cachedPerformance.Matches[:count]
			return &trimmed, nil
		}
		return &cachedPerformance, nil
	}
	if err != nil && err != mongo.ErrNoDocuments {
		log.Printf("Error fetching user performance from MongoDB for %s: %v. Will fetch from API.", puuid, err)
	}

	log.Printf("Fetching fresh match data for %s#%s (%s)", gameName, tagLine, puuid)

	var seasonStartTime int64 = 0

	matchIDs, err := getMatchIDs(app, userRegion, puuid, count, queueID, seasonStartTime)
	if err != nil {
		return nil, fmt.Errorf("error getting match IDs: %w", err)
	}

	if len(matchIDs) == 0 {
		log.Printf("No match IDs found for %s in region %s with queue %d", puuid, userRegion, queueID)
		if cachedPerformance.PUUID != "" {
			cachedPerformance.UpdatedAt = time.Now().Unix()
			return &cachedPerformance, nil
		}
		return &UserPerformance{PUUID: puuid, Region: userRegion, RiotID: gameName + "#" + tagLine, Matches: []PlayerMatchStats{}, UpdatedAt: time.Now().Unix()}, nil
	}

	var matches []PlayerMatchStats
	matches = fetchMatchesConcurrently(app, userRegion, matchIDs, puuid)

	performance := UserPerformance{
		PUUID:     puuid,
		Region:    userRegion,
		RiotID:    gameName + "#" + tagLine,
		Matches:   matches,
		UpdatedAt: time.Now().Unix(),
	}

	// Move persistence off the critical path - run asynchronously
	go func(data UserPerformance, redisKey string, collection *mongo.Collection) {
		persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Async MongoDB write
		opts := options.Update().SetUpsert(true)
		filter := bson.M{"_id": data.PUUID, "region": data.Region}
		update := bson.M{"$set": data}
		_, _ = collection.UpdateOne(persistCtx, filter, update, opts)

		// Async Redis write
		if dataJSON, err := json.Marshal(data); err == nil {
			_ = app.redisClient.Set(persistCtx, redisKey, dataJSON, userPerformanceCacheDuration).Err()
		}
	}(performance, redisCacheKey, collection)

	return &performance, nil
}

func loadDataDragonVersions(app *GlobalAppData) ([]string, error) {
	url := fmt.Sprintf("%s/api/versions.json", dataDragonBaseURL)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := app.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ddragon versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ddragon versions request failed with status %d", resp.StatusCode)
	}

	var versions DataDragonVersions
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, fmt.Errorf("failed to decode ddragon versions: %w", err)
	}
	return versions, nil
}

func loadChampions(app *GlobalAppData, version string) (map[string]ChampionData, error) {
	cacheKey := fmt.Sprintf("ddragon:champions:%s", version)
	val, err := app.redisClient.Get(context.Background(), cacheKey).Result()
	if err == nil {
		var champions DataDragonChampions
		if json.Unmarshal([]byte(val), &champions) == nil {
			log.Printf("Champions loaded from cache for version %s", version)
			return champions.Data, nil
		}
	}

	url := fmt.Sprintf("%s/cdn/%s/data/en_US/champion.json", dataDragonBaseURL, version)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := app.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch champions for version %s: %w", version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("champions request for version %s failed with status %d", version, resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var champions DataDragonChampions
	if err := json.Unmarshal(bodyBytes, &champions); err != nil {
		return nil, fmt.Errorf("failed to decode champions for version %s: %w", version, err)
	}

	// Move Redis caching off the critical path - run asynchronously
	go func(key string, data []byte) {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = app.redisClient.Set(cacheCtx, key, string(data), staticDataCacheDuration).Err()
	}(cacheKey, bodyBytes)

	log.Printf("Champions loaded from API for version %s", version)
	return champions.Data, nil
}

func loadItems(app *GlobalAppData, version string) (map[string]ItemData, error) {
	cacheKey := fmt.Sprintf("ddragon:items:%s", version)
	val, err := app.redisClient.Get(context.Background(), cacheKey).Result()
	if err == nil {
		var items DataDragonItems
		if json.Unmarshal([]byte(val), &items) == nil {
			log.Printf("Items loaded from cache for version %s", version)
			return items.Data, nil
		}
	}

	url := fmt.Sprintf("%s/cdn/%s/data/en_US/item.json", dataDragonBaseURL, version)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := app.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch items for version %s: %w", version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("items request for version %s failed with status %d", version, resp.StatusCode)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	var items DataDragonItems
	if err := json.Unmarshal(bodyBytes, &items); err != nil {
		return nil, fmt.Errorf("failed to decode items for version %s: %w", version, err)
	}

	// Move Redis caching off the critical path - run asynchronously
	go func(key string, data []byte) {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = app.redisClient.Set(cacheCtx, key, string(data), staticDataCacheDuration).Err()
	}(cacheKey, bodyBytes)

	log.Printf("Items loaded from API for version %s", version)
	return items.Data, nil
}

func loadSummonerSpells(app *GlobalAppData, version string) (map[string]SummonerSpellData, error) {
	cacheKey := fmt.Sprintf("ddragon:summonerspells:%s", version)
	val, err := app.redisClient.Get(context.Background(), cacheKey).Result()
	if err == nil {
		var spells DataDragonSummonerSpells
		if json.Unmarshal([]byte(val), &spells) == nil {
			log.Printf("Summoner spells loaded from cache for version %s", version)
			return spells.Data, nil
		}
	}

	url := fmt.Sprintf("%s/cdn/%s/data/en_US/summoner.json", dataDragonBaseURL, version)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := app.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch summoner spells for version %s: %w", version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("summoner spells request for version %s failed with status %d", version, resp.StatusCode)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	var spells DataDragonSummonerSpells
	if err := json.Unmarshal(bodyBytes, &spells); err != nil {
		return nil, fmt.Errorf("failed to decode summoner spells for version %s: %w", version, err)
	}

	// Move Redis caching off the critical path - run asynchronously
	go func(key string, data []byte) {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = app.redisClient.Set(cacheCtx, key, string(data), staticDataCacheDuration).Err()
	}(cacheKey, bodyBytes)

	log.Printf("Summoner spells loaded from API for version %s", version)
	return spells.Data, nil
}

func loadRunes(app *GlobalAppData, version string) (map[int]RuneInfo, error) {
	cacheKey := fmt.Sprintf("ddragon:runesreforged:%s", version)
	val, err := app.redisClient.Get(context.Background(), cacheKey).Result()
	if err == nil {
		var runePaths []RunePathData
		if json.Unmarshal([]byte(val), &runePaths) == nil {
			log.Printf("Runes loaded from cache for version %s", version)
			return flattenRuneData(runePaths), nil
		}
	}

	url := fmt.Sprintf("%s/cdn/%s/data/en_US/runesReforged.json", dataDragonBaseURL, version)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := app.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch runes for version %s: %w", version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("runes request for version %s failed with status %d", version, resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var runePaths []RunePathData
	if err := json.Unmarshal(bodyBytes, &runePaths); err != nil {
		return nil, fmt.Errorf("failed to decode runes for version %s: %w", version, err)
	}

	// Move Redis caching off the critical path - run asynchronously
	go func(key string, data []byte) {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = app.redisClient.Set(cacheCtx, key, string(data), staticDataCacheDuration).Err()
	}(cacheKey, bodyBytes)

	log.Printf("Runes loaded from API for version %s", version)
	return flattenRuneData(runePaths), nil
}

func flattenRuneData(runePaths []RunePathData) map[int]RuneInfo {
	flatRunes := make(map[int]RuneInfo)
	for _, path := range runePaths {
		for _, slot := range path.Slots {
			for _, runeInfo := range slot.Runes {
				flatRunes[runeInfo.ID] = runeInfo
			}
		}
	}
	return flatRunes
}

func populateStaticData(app *GlobalAppData) error {
	log.Println("Populating static data from Data Dragon...")
	versions, err := loadDataDragonVersions(app)
	if err != nil || len(versions) == 0 {
		return fmt.Errorf("could not load Data Dragon versions: %w", err)
	}
	latestVersion := versions[0]
	log.Printf("Latest Data Dragon version: %s", latestVersion)

	champions, err := loadChampions(app, latestVersion)
	if err != nil {
		return fmt.Errorf("error loading champions: %w", err)
	}

	championKeyToDataMap := make(map[string]ChampionData)
	for _, champ := range champions {
		championKeyToDataMap[champ.Key] = champ
	}

	items, err := loadItems(app, latestVersion)
	if err != nil {
		return fmt.Errorf("error loading items: %w", err)
	}

	summonerSpells, err := loadSummonerSpells(app, latestVersion)
	if err != nil {
		return fmt.Errorf("error loading summoner spells: %w", err)
	}
	summonerSpellsByKey := make(map[string]SummonerSpellData)
	for _, spell := range summonerSpells {
		summonerSpellsByKey[spell.Key] = spell
	}

	runes, err := loadRunes(app, latestVersion)
	if err != nil {
		return fmt.Errorf("error loading runes: %w", err)
	}

	app.staticData = &StaticData{
		Champions:      championKeyToDataMap,
		Items:          items,
		Runes:          runes,
		SummonerSpells: summonerSpellsByKey,
		LatestVersion:  latestVersion,
	}
	log.Println("Static data populated successfully.")
	return nil
}

func getAPIRegion(region string) string {
	lowerRegion := strings.ToLower(region)
	switch {
	case strings.HasPrefix(lowerRegion, "na"), strings.HasPrefix(lowerRegion, "br"), strings.HasPrefix(lowerRegion, "lan"), strings.HasPrefix(lowerRegion, "las"), strings.HasPrefix(lowerRegion, "oce"):
		return "americas"
	case strings.HasPrefix(lowerRegion, "kr"), strings.HasPrefix(lowerRegion, "jp"):
		return "asia"
	case strings.HasPrefix(lowerRegion, "eun"), strings.HasPrefix(lowerRegion, "euw"), strings.HasPrefix(lowerRegion, "tr"), strings.HasPrefix(lowerRegion, "ru"):
		return "europe"
	case strings.HasPrefix(lowerRegion, "sg"), strings.HasPrefix(lowerRegion, "ph"), strings.HasPrefix(lowerRegion, "th"), strings.HasPrefix(lowerRegion, "vn"), strings.HasPrefix(lowerRegion, "tw"):
		return "sea"
	default:
		log.Printf("Warning: Unknown region prefix for '%s', defaulting to 'americas' for API routing. Please check region mapping.", region)
		return "americas"
	}
}

// Add this helper function to check if a game mode is classic
func isClassicMode(gameMode string) bool {
	return strings.ToUpper(gameMode) == "CLASSIC"
}

func calculateRecentGamesSummary(matches []PlayerMatchStats, puuid, region, riotID string) *RecentGamesSummary {
	if len(matches) == 0 {
		return &RecentGamesSummary{
			PUUID:         puuid,
			Region:        region,
			RiotID:        riotID,
			TotalMatches:  0,
			OverallStats:  OverallStats{},
			RoleStats:     make(map[string]RoleStats),
			ChampionStats: make(map[string]ChampionStats),
			RecentMatches: []PlayerMatchStats{},
			LastUpdated:   time.Now().Unix(),
		}
	}

	// Calculate overall stats
	overallStats := calculateOverallStats(matches)

	// Calculate role-based stats
	roleStats := calculateRoleStats(matches)

	// Calculate champion-based stats
	championStats := calculateChampionStats(matches)

	return &RecentGamesSummary{
		PUUID:         puuid,
		Region:        region,
		RiotID:        riotID,
		TotalMatches:  len(matches),
		OverallStats:  overallStats,
		RoleStats:     roleStats,
		ChampionStats: championStats,
		RecentMatches: matches,
		LastUpdated:   time.Now().Unix(),
	}
}

// calculateOverallStats computes aggregate statistics across all matches
func calculateOverallStats(matches []PlayerMatchStats) OverallStats {
	if len(matches) == 0 {
		return OverallStats{}
	}

	var wins, totalKills, totalDeaths, totalAssists int
	var totalGameTime, totalVisionScore, totalDamage int64
	var totalKillParticipation float64

	// Separate tracking for classic mode stats
	var classicGameTime, classicCS, classicGold int64
	var classicGameCount int

	for _, match := range matches {
		if match.Win {
			wins++
		}
		totalKills += match.Kills
		totalDeaths += match.Deaths
		totalAssists += match.Assists
		totalGameTime += match.GameDuration
		totalVisionScore += int64(match.VisionScore)
		totalDamage += int64(match.DamageToChampions)
		totalKillParticipation += match.KillParticipation

		// Only count CS and Gold for classic mode
		if isClassicMode(match.GameMode) {
			classicGameTime += match.GameDuration
			classicCS += int64(match.TotalMinionsKilled)
			classicGold += int64(match.GoldEarned)
			classicGameCount++
		}
	}

	losses := len(matches) - wins
	winRate := float64(wins) / float64(len(matches)) * 100

	// Calculate KDA
	var overallKDA float64
	if totalDeaths > 0 {
		overallKDA = float64(totalKills+totalAssists) / float64(totalDeaths)
	} else {
		overallKDA = float64(totalKills + totalAssists)
	}

	// Calculate CS/min and Gold/min only for classic games
	var avgCSPerMin, avgGoldPerMin float64
	if classicGameTime > 0 {
		avgCSPerMin = (float64(classicCS) / float64(classicGameTime)) * 60
		avgGoldPerMin = (float64(classicGold) / float64(classicGameTime)) * 60
	}

	return OverallStats{
		Wins:                 wins,
		Losses:               losses,
		WinRate:              winRate,
		TotalKills:           totalKills,
		TotalDeaths:          totalDeaths,
		TotalAssists:         totalAssists,
		AvgKills:             float64(totalKills) / float64(len(matches)),
		AvgDeaths:            float64(totalDeaths) / float64(len(matches)),
		AvgAssists:           float64(totalAssists) / float64(len(matches)),
		OverallKDA:           overallKDA,
		AvgGameDuration:      float64(totalGameTime) / float64(len(matches)),
		TotalGameTime:        totalGameTime,
		AvgVisionScore:       float64(totalVisionScore) / float64(len(matches)),
		AvgCSPerMin:          avgCSPerMin,
		AvgGoldPerMin:        avgGoldPerMin,
		AvgDamageToChampions: float64(totalDamage) / float64(len(matches)),
		AvgKillParticipation: totalKillParticipation / float64(len(matches)),
	}
}

// calculateRoleStats computes statistics grouped by role/position
func calculateRoleStats(matches []PlayerMatchStats) map[string]RoleStats {
	roleMap := make(map[string][]PlayerMatchStats)

	// Group matches by role
	for _, match := range matches {
		role := normalizeRole(match.TeamPosition, match.GameMode)
		roleMap[role] = append(roleMap[role], match)
	}

	roleStats := make(map[string]RoleStats)
	for role, roleMatches := range roleMap {
		if len(roleMatches) == 0 {
			continue
		}

		var wins, totalKills, totalDeaths, totalAssists int
		var totalVisionScore, totalDamage int64
		var totalKillParticipation float64
		var totalGameTime int64

		// Separate tracking for classic mode stats
		var classicGameTime, classicCS, classicGold int64
		var classicGameCount int

		for _, match := range roleMatches {
			if match.Win {
				wins++
			}
			totalKills += match.Kills
			totalDeaths += match.Deaths
			totalAssists += match.Assists
			totalVisionScore += int64(match.VisionScore)
			totalDamage += int64(match.DamageToChampions)
			totalKillParticipation += match.KillParticipation
			totalGameTime += match.GameDuration

			// Only count CS and Gold for classic mode
			if isClassicMode(match.GameMode) {
				classicGameTime += match.GameDuration
				classicCS += int64(match.TotalMinionsKilled)
				classicGold += int64(match.GoldEarned)
				classicGameCount++
			}
		}

		losses := len(roleMatches) - wins
		winRate := float64(wins) / float64(len(roleMatches)) * 100

		var roleKDA float64
		if totalDeaths > 0 {
			roleKDA = float64(totalKills+totalAssists) / float64(totalDeaths)
		} else {
			roleKDA = float64(totalKills + totalAssists)
		}

		// Calculate CS/min and Gold/min only for classic games
		var avgCSPerMin, avgGoldPerMin float64
		if classicGameTime > 0 {
			avgCSPerMin = (float64(classicCS) / float64(classicGameTime)) * 60
			avgGoldPerMin = (float64(classicGold) / float64(classicGameTime)) * 60
		}

		roleStats[role] = RoleStats{
			Role:                 role,
			GamesPlayed:          len(roleMatches),
			Wins:                 wins,
			Losses:               losses,
			WinRate:              winRate,
			TotalKills:           totalKills,
			TotalDeaths:          totalDeaths,
			TotalAssists:         totalAssists,
			AvgKills:             float64(totalKills) / float64(len(roleMatches)),
			AvgDeaths:            float64(totalDeaths) / float64(len(roleMatches)),
			AvgAssists:           float64(totalAssists) / float64(len(roleMatches)),
			RoleKDA:              roleKDA,
			AvgVisionScore:       float64(totalVisionScore) / float64(len(roleMatches)),
			AvgCSPerMin:          avgCSPerMin,
			AvgGoldPerMin:        avgGoldPerMin,
			AvgDamageToChampions: float64(totalDamage) / float64(len(roleMatches)),
			AvgKillParticipation: totalKillParticipation / float64(len(roleMatches)),
		}
	}

	return roleStats
}

// calculateChampionStats computes statistics grouped by champion
func calculateChampionStats(matches []PlayerMatchStats) map[string]ChampionStats {
	championMap := make(map[string][]PlayerMatchStats)

	// Group matches by champion
	for _, match := range matches {
		championMap[match.ChampionName] = append(championMap[match.ChampionName], match)
	}

	championStats := make(map[string]ChampionStats)
	for championName, championMatches := range championMap {
		if len(championMatches) == 0 {
			continue
		}

		var wins, totalKills, totalDeaths, totalAssists int
		var totalVisionScore, totalDamage int64
		var totalKillParticipation float64
		var totalGameTime int64
		var bestKDA, worstKDA float64
		var lastPlayed int64
		var championID int

		// Separate tracking for classic mode stats
		var classicGameTime, classicCS, classicGold int64
		var classicGameCount int

		bestKDA = -1      // Initialize to impossible value
		worstKDA = 999999 // Initialize to very high value

		for i, match := range championMatches {
			if i == 0 {
				championID = match.ChampionID
				lastPlayed = match.GameCreation
			}

			if match.GameCreation > lastPlayed {
				lastPlayed = match.GameCreation
			}

			if match.Win {
				wins++
			}
			totalKills += match.Kills
			totalDeaths += match.Deaths
			totalAssists += match.Assists
			totalVisionScore += int64(match.VisionScore)
			totalDamage += int64(match.DamageToChampions)
			totalKillParticipation += match.KillParticipation
			totalGameTime += match.GameDuration

			// Only count CS and Gold for classic mode
			if isClassicMode(match.GameMode) {
				classicGameTime += match.GameDuration
				classicCS += int64(match.TotalMinionsKilled)
				classicGold += int64(match.GoldEarned)
				classicGameCount++
			}

			// Track best and worst KDA
			if match.KDA > bestKDA {
				bestKDA = match.KDA
			}
			if match.KDA < worstKDA {
				worstKDA = match.KDA
			}
		}

		losses := len(championMatches) - wins
		winRate := float64(wins) / float64(len(championMatches)) * 100

		var championKDA float64
		if totalDeaths > 0 {
			championKDA = float64(totalKills+totalAssists) / float64(totalDeaths)
		} else {
			championKDA = float64(totalKills + totalAssists)
		}

		// Calculate CS/min and Gold/min only for classic games
		var avgCSPerMin, avgGoldPerMin float64
		if classicGameTime > 0 {
			avgCSPerMin = (float64(classicCS) / float64(classicGameTime)) * 60
			avgGoldPerMin = (float64(classicGold) / float64(classicGameTime)) * 60
		}

		championStats[championName] = ChampionStats{
			ChampionName:         championName,
			ChampionID:           championID,
			GamesPlayed:          len(championMatches),
			Wins:                 wins,
			Losses:               losses,
			WinRate:              winRate,
			TotalKills:           totalKills,
			TotalDeaths:          totalDeaths,
			TotalAssists:         totalAssists,
			AvgKills:             float64(totalKills) / float64(len(championMatches)),
			AvgDeaths:            float64(totalDeaths) / float64(len(championMatches)),
			AvgAssists:           float64(totalAssists) / float64(len(championMatches)),
			ChampionKDA:          championKDA,
			BestKDA:              bestKDA,
			WorstKDA:             worstKDA,
			AvgVisionScore:       float64(totalVisionScore) / float64(len(championMatches)),
			AvgCSPerMin:          avgCSPerMin,
			AvgGoldPerMin:        avgGoldPerMin,
			AvgDamageToChampions: float64(totalDamage) / float64(len(championMatches)),
			AvgKillParticipation: totalKillParticipation / float64(len(championMatches)),
			LastPlayed:           lastPlayed,
		}
	}

	return championStats
}

// normalizeRole standardizes role names for consistent grouping, including game modes
func normalizeRole(teamPosition string, gameMode string) string {
	// For certain game modes, treat the game mode as the role
	switch strings.ToUpper(gameMode) {
	case "ARAM":
		return "ARAM"
	case "CHERRY":
		return "Arena"
	}

	// For other game modes, use traditional role mapping
	switch strings.ToUpper(teamPosition) {
	case "TOP":
		return "Top"
	case "JUNGLE":
		return "Jungle"
	case "MIDDLE", "MID":
		return "Mid"
	case "BOTTOM", "BOT":
		return "Bot"
	case "UTILITY", "SUPPORT":
		return "Support"
	default:
		if teamPosition == "" {
			return "Unknown"
		}
		return teamPosition
	}
}

// fetchRecentGamesSummary gets comprehensive match summary with caching
func fetchRecentGamesSummary(app *GlobalAppData, userRegion, gameName, tagLine string, count, queueID int) (*RecentGamesSummary, error) {
	// First fetch the regular user performance data
	userPerformance, err := fetchAndStoreUserPerformance(app, userRegion, gameName, tagLine, count, queueID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user performance: %w", err)
	}

	// Calculate comprehensive summary
	summary := calculateRecentGamesSummary(userPerformance.Matches, userPerformance.PUUID, userPerformance.Region, userPerformance.RiotID)

	return summary, nil
}
