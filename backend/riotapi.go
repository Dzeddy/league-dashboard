package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

		err = app.redisClient.Set(context.Background(), cacheKey, acc.PUUID, puuidCacheDuration).Err()
		if err != nil {
			log.Printf("Warning: Failed to cache PUUID: %v", err)
		}
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

		matchIDsJSON, _ := json.Marshal(matchIDs)
		err = app.redisClient.Set(context.Background(), cacheKey, matchIDsJSON, matchListCacheDuration).Err()
		if err != nil {
			log.Printf("Warning: Failed to cache match IDs: %v", err)
		}
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

		err = app.redisClient.Set(context.Background(), cacheKey, string(bodyBytes), matchDetailsCacheDuration).Err()
		if err != nil {
			log.Printf("Warning: Failed to cache match details for %s: %v", matchID, err)
		}
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
		perfJSON, _ := json.Marshal(cachedPerformance)
		if err := app.redisClient.Set(ctx, redisCacheKey, perfJSON, userPerformanceCacheDuration).Err(); err != nil {
			log.Printf("Warning: failed to update Redis cache for user performance %s: %v", puuid, err)
		}
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
	for _, matchID := range matchIDs {
		matchData, err := getMatchDetails(app, userRegion, matchID)
		if err != nil {
			log.Printf("Error getting match details for %s: %v. Skipping this match.", matchID, err)
			continue
		}
		if matchData == nil {
			continue
		}

		playerStats, err := extractPlayerMatchStats(matchData, puuid, app)
		if err != nil {
			log.Printf("Error extracting player stats for match %s, PUUID %s: %v. Skipping this match.", matchID, puuid, err)
			continue
		}
		if playerStats != nil {
			matches = append(matches, *playerStats)
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].GameCreation > matches[j].GameCreation
	})

	performance := UserPerformance{
		PUUID:     puuid,
		Region:    userRegion,
		RiotID:    gameName + "#" + tagLine,
		Matches:   matches,
		UpdatedAt: time.Now().Unix(),
	}

	opts := options.Update().SetUpsert(true)
	_, err = collection.UpdateOne(ctx, bson.M{"_id": puuid, "region": userRegion}, bson.M{"$set": performance}, opts)
	if err != nil {
		log.Printf("Warning: Failed to store user performance for %s in MongoDB: %v", puuid, err)
	}

	perfJSON, err := json.Marshal(performance)
	if err != nil {
		log.Printf("Warning: Failed to marshal performance data for Redis for %s: %v", puuid, err)
	} else {
		if err := app.redisClient.Set(ctx, redisCacheKey, perfJSON, userPerformanceCacheDuration).Err(); err != nil {
			log.Printf("Warning: failed to update Redis cache for user performance %s: %v", puuid, err)
		}
	}

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

	err = app.redisClient.Set(context.Background(), cacheKey, string(bodyBytes), staticDataCacheDuration).Err()
	if err != nil {
		log.Printf("Warning: Failed to cache champions for version %s: %v", version, err)
	}
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
	err = app.redisClient.Set(context.Background(), cacheKey, string(bodyBytes), staticDataCacheDuration).Err()
	if err != nil {
		log.Printf("Warning: Failed to cache items for version %s: %v", version, err)
	}
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
	err = app.redisClient.Set(context.Background(), cacheKey, string(bodyBytes), staticDataCacheDuration).Err()
	if err != nil {
		log.Printf("Warning: Failed to cache summoner spells for version %s: %v", version, err)
	}
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

	err = app.redisClient.Set(context.Background(), cacheKey, string(bodyBytes), staticDataCacheDuration).Err()
	if err != nil {
		log.Printf("Warning: Failed to cache runes for version %s: %v", version, err)
	}
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
