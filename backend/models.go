package main

import (
	"net/http"

	redis "github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
)

// AccountDTO represents the Riot Account-v1 DTO
type AccountDTO struct {
	PUUID    string `json:"puuid"`
	GameName string `json:"gameName"`
	TagLine  string `json:"tagLine"`
}

// MatchDto represents the Riot Match-v5 DTO (simplified)
type MatchDto struct {
	Metadata MatchMetadataDto `json:"metadata"`
	Info     MatchInfoDto     `json:"info"`
}

// MatchMetadataDto represents parts of the metadata for a match
type MatchMetadataDto struct {
	MatchID      string   `json:"matchId"`
	Participants []string `json:"participants"` // List of PUUIDs
}

// MatchInfoDto represents parts of the info for a match
type MatchInfoDto struct {
	GameCreation     int64            `json:"gameCreation"`
	GameDuration     int64            `json:"gameDuration"` // Duration in seconds
	GameEndTimestamp int64            `json:"gameEndTimestamp"`
	GameID           int64            `json:"gameId"`
	GameMode         string           `json:"gameMode"`
	GameType         string           `json:"gameType"`
	GameVersion      string           `json:"gameVersion"`
	MapID            int              `json:"mapId"`
	Participants     []ParticipantDto `json:"participants"`
	QueueID          int              `json:"queueId"`
	// Teams              []TeamDto            `json:"teams"` // We might need this later for win/loss
	EndOfGameResult string `json:"endOfGameResult,omitempty"`
}

// ParticipantDto represents a participant in a match (simplified)
type ParticipantDto struct {
	PUUID          string `json:"puuid"`
	SummonerName   string `json:"summonerName,omitempty"` // Sometimes empty
	RiotIDGameName string `json:"riotIdGameName,omitempty"`
	RiotIDTagline  string `json:"riotIdTagline,omitempty"`
	ChampionID     int    `json:"championId"`
	ChampionName   string `json:"championName"`
	TeamID         int    `json:"teamId"`
	Win            bool   `json:"win"`

	Kills                int `json:"kills"`
	Deaths               int `json:"deaths"`
	Assists              int `json:"assists"`
	TotalMinionsKilled   int `json:"totalMinionsKilled"`
	NeutralMinionsKilled int `json:"neutralMinionsKilled"`
	VisionScore          int `json:"visionScore"`
	GoldEarned           int `json:"goldEarned"`
	// Add other relevant stats here...
	Challenges                  *ParticipantChallengesDto `json:"challenges,omitempty"` // Use pointer for optional fields
	Perks                       *PerksDto                 `json:"perks,omitempty"`
	TeamPosition                string                    `json:"teamPosition"`
	Lane                        string                    `json:"lane"`
	Item0                       int                       `json:"item0"`
	Item1                       int                       `json:"item1"`
	Item2                       int                       `json:"item2"`
	Item3                       int                       `json:"item3"`
	Item4                       int                       `json:"item4"`
	Item5                       int                       `json:"item5"`
	Item6                       int                       `json:"item6"` // Trinket
	Summoner1Casts              int                       `json:"summoner1Casts"`
	Summoner1Id                 int                       `json:"summoner1Id"`
	Summoner2Casts              int                       `json:"summoner2Casts"`
	Summoner2Id                 int                       `json:"summoner2Id"`
	ChampLevel                  int                       `json:"champLevel"`
	DamageDealtToTurrets        int                       `json:"damageDealtToTurrets"`
	DamageDealtToObjectives     int                       `json:"damageDealtToObjectives"`
	TotalDamageDealtToChampions int                       `json:"totalDamageDealtToChampions"`
	TotalDamageTaken            int                       `json:"totalDamageTaken"`
	TimePlayed                  int                       `json:"timePlayed"`
}

// ParticipantChallengesDto holds specific challenge data if needed
type ParticipantChallengesDto struct {
	KDA               float64 `json:"kda,omitempty"`
	KillParticipation float64 `json:"killParticipation,omitempty"`
	// Add other challenges as needed
}

// PerksDto represents perk information
type PerksDto struct {
	StatPerks StatPerksDto `json:"statPerks"`
	Styles    []StyleDto   `json:"styles"`
}

// StatPerksDto represents stat perk selections
type StatPerksDto struct {
	Defense int `json:"defense"`
	Flex    int `json:"flex"`
	Offense int `json:"offense"`
}

// StyleDto represents a perk style (e.g., primary or subStyle)
type StyleDto struct {
	Description string         `json:"description"`
	Selections  []SelectionDto `json:"selections"`
	Style       int            `json:"style"` // Rune tree ID
}

// SelectionDto represents a selected perk
type SelectionDto struct {
	Perk int `json:"perk"` // Perk ID
	Var1 int `json:"var1"`
	Var2 int `json:"var2"`
	Var3 int `json:"var3"`
}

// PlayerMatchStats is a custom struct to store aggregated stats for a player in a match
type PlayerMatchStats struct {
	MatchID            string        `json:"matchId" bson:"matchId"`
	GameMode           string        `json:"gameMode" bson:"gameMode"`
	GameCreation       int64         `json:"gameCreation" bson:"gameCreation"`
	GameDuration       int64         `json:"gameDuration" bson:"gameDuration"`
	ChampionName       string        `json:"championName" bson:"championName"`
	ChampionID         int           `json:"championId" bson:"championId"`
	Win                bool          `json:"win" bson:"win"`
	Kills              int           `json:"kills" bson:"kills"`
	Deaths             int           `json:"deaths" bson:"deaths"`
	Assists            int           `json:"assists" bson:"assists"`
	KDA                float64       `json:"kda" bson:"kda"`
	KillParticipation  float64       `json:"killParticipation,omitempty" bson:"killParticipation,omitempty"`
	TotalMinionsKilled int           `json:"totalMinionsKilled" bson:"totalMinionsKilled"`
	VisionScore        int           `json:"visionScore" bson:"visionScore"`
	GoldEarned         int           `json:"goldEarned" bson:"goldEarned"`
	TeamPosition       string        `json:"teamPosition" bson:"teamPosition"`
	Items              []int         `json:"items" bson:"items"`
	SummonerSpells     []int         `json:"summonerSpells" bson:"summonerSpells"`
	PrimaryRune        int           `json:"primaryRune" bson:"primaryRune"`
	SecondaryStyle     int           `json:"secondaryStyle" bson:"secondaryStyle"`
	ChampLevel         int           `json:"champLevel" bson:"champLevel"`
	DamageToTurrets    int           `json:"damageToTurrets" bson:"damageToTurrets"`
	DamageToObjectives int           `json:"damageToObjectives" bson:"damageToObjectives"`
	DamageToChampions  int           `json:"damageToChampions" bson:"damageToChampions"`
	TotalDamageTaken   int           `json:"totalDamageTaken" bson:"totalDamageTaken"`
	TeamID             int           `json:"teamId" bson:"teamId"` // 100 for blue, 200 for red
	QueueID            int           `json:"queueId" bson:"queueId"`
	FullMatchData      *MatchInfoDto `json:"-" bson:"-"` // To hold the original match data if needed for more processing, but not sent to frontend directly for this summary
}

// UserPerformance stores a collection of match stats for a user
type UserPerformance struct {
	PUUID     string             `json:"puuid" bson:"_id"` // Use PUUID as MongoDB document ID
	Region    string             `json:"region" bson:"region"`
	RiotID    string             `json:"riotId" bson:"riotId"` // GameName#TagLine
	Matches   []PlayerMatchStats `json:"matches" bson:"matches"`
	UpdatedAt int64              `json:"updatedAt" bson:"updatedAt"`
}

// ChampionData holds basic champion information
type ChampionData struct {
	Version string             `json:"version"`
	ID      string             `json:"id"`   // e.g., "Aatrox"
	Key     string             `json:"key"`  // e.g., "266"
	Name    string             `json:"name"` // e.g., "Aatrox"
	Title   string             `json:"title"`
	Image   ChampionImageDTO   `json:"image"`
	Partype string             `json:"partype"` // Resource type (Mana, Energy, etc.)
	Stats   map[string]float64 `json:"stats"`
}

// ChampionImageDTO for Data Dragon
type ChampionImageDTO struct {
	Full   string `json:"full"`   // Aatrox.png
	Sprite string `json:"sprite"` // champion0.png
	Group  string `json:"group"`  // champion
	X      int    `json:"x"`
	Y      int    `json:"y"`
	W      int    `json:"w"`
	H      int    `json:"h"`
}

// DataDragonChampions holds the full champion data structure from Data Dragon
type DataDragonChampions struct {
	Type    string                  `json:"type"`
	Format  string                  `json:"format"`
	Version string                  `json:"version"`
	Data    map[string]ChampionData `json:"data"` // Keyed by champion ID (e.g., "Aatrox")
}

// ItemData holds basic item information from Data Dragon
type ItemData struct {
	Name        string             `json:"name"`
	Description string             `json:"description"` // This can be HTML
	Plaintext   string             `json:"plaintext"`
	Image       ItemImageDTO       `json:"image"`
	Gold        ItemGoldDTO        `json:"gold"`
	Tags        []string           `json:"tags"`
	Maps        map[string]bool    `json:"maps"`
	Stats       map[string]float64 `json:"stats"`
	Depth       int                `json:"depth,omitempty"` // For component items
}

// ItemImageDTO for Data Dragon items
type ItemImageDTO struct {
	Full   string `json:"full"` // e.g., "1001.png"
	Sprite string `json:"sprite"`
	Group  string `json:"group"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	W      int    `json:"w"`
	H      int    `json:"h"`
}

// ItemGoldDTO for Data Dragon items
type ItemGoldDTO struct {
	Base        int  `json:"base"`
	Purchasable bool `json:"purchasable"`
	Total       int  `json:"total"`
	Sell        int  `json:"sell"`
}

// DataDragonItems holds the full item data structure from Data Dragon
type DataDragonItems struct {
	Type    string              `json:"type"`
	Version string              `json:"version"`
	Data    map[string]ItemData `json:"data"` // Keyed by item ID (e.g., "1001")
}

// RunePathData from Data Dragon
type RunePathData struct {
	ID    int        `json:"id"`
	Key   string     `json:"key"`  // e.g., "Precision"
	Icon  string     `json:"icon"` // path to icon
	Name  string     `json:"name"`
	Slots []RuneSlot `json:"slots"`
}

// RuneSlot from Data Dragon
type RuneSlot struct {
	Runes []RuneInfo `json:"runes"`
}

// RuneInfo from Data Dragon
type RuneInfo struct {
	ID        int    `json:"id"`
	Key       string `json:"key"` // e.g., "PressTheAttack"
	Icon      string `json:"icon"`
	Name      string `json:"name"`
	ShortDesc string `json:"shortDesc"` // Contains HTML
	LongDesc  string `json:"longDesc"`  // Contains HTML
}

// SummonerSpellData from Data Dragon
type SummonerSpellData struct {
	ID            string                `json:"id"`   // e.g., "SummonerFlash"
	Name          string                `json:"name"` // e.g., "Flash"
	Description   string                `json:"description"`
	Tooltip       string                `json:"tooltip"`
	MaxRank       int                   `json:"maxrank"`
	Cooldown      []float64             `json:"cooldown"`
	CooldownBurn  string                `json:"cooldownBurn"`
	Cost          []int                 `json:"cost"`
	CostBurn      string                `json:"costBurn"`
	Key           string                `json:"key"` // e.g., "4" for Flash (this is the ID used in match data)
	SummonerLevel int                   `json:"summonerLevel"`
	Modes         []string              `json:"modes"` // Game modes it's available in
	CostType      string                `json:"costType"`
	MaxAmmo       string                `json:"maxammo"`
	Range         []int                 `json:"range"`
	RangeBurn     string                `json:"rangeBurn"`
	Image         SummonerSpellImageDTO `json:"image"`
	Resource      string                `json:"resource,omitempty"`
}

// SummonerSpellImageDTO for Data Dragon
type SummonerSpellImageDTO struct {
	Full   string `json:"full"` // e.g., "SummonerFlash.png"
	Sprite string `json:"sprite"`
	Group  string `json:"group"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	W      int    `json:"w"`
	H      int    `json:"h"`
}

// DataDragonSummonerSpells holds the full summoner spell data structure from Data Dragon
type DataDragonSummonerSpells struct {
	Type    string                       `json:"type"`
	Version string                       `json:"version"`
	Data    map[string]SummonerSpellData `json:"data"` // Keyed by spell ID (e.g., "SummonerFlash")
}

// DataDragonVersions lists available Data Dragon versions
type DataDragonVersions []string

// StaticData holds all loaded static data (champions, items, etc.)
type StaticData struct {
	Champions      map[string]ChampionData      // Keyed by Champion Key (string version of ID)
	Items          map[string]ItemData          // Keyed by Item ID (string)
	Runes          map[int]RuneInfo             // Keyed by Rune ID (int)
	SummonerSpells map[string]SummonerSpellData // Keyed by Summoner Spell Key (string version of ID)
	LatestVersion  string
}

// GlobalAppData holds clients and other global resources
type GlobalAppData struct {
	httpClient    *http.Client
	redisClient   *redis.Client
	mongoClient   *mongo.Client
	mongoDatabase string
	riotAPIKey    string
	staticData    *StaticData
}

type RecentGamesSummary struct {
	PUUID         string                   `json:"puuid" bson:"_id"`
	Region        string                   `json:"region" bson:"region"`
	RiotID        string                   `json:"riotId" bson:"riotId"`
	TotalMatches  int                      `json:"totalMatches" bson:"totalMatches"`
	OverallStats  OverallStats             `json:"overallStats" bson:"overallStats"`
	RoleStats     map[string]RoleStats     `json:"roleStats" bson:"roleStats"`
	ChampionStats map[string]ChampionStats `json:"championStats" bson:"championStats"`
	RecentMatches []PlayerMatchStats       `json:"recentMatches" bson:"recentMatches"`
	LastUpdated   int64                    `json:"lastUpdated" bson:"lastUpdated"`
}

type OverallStats struct {
	Wins                 int     `json:"wins" bson:"wins"`
	Losses               int     `json:"losses" bson:"losses"`
	WinRate              float64 `json:"winRate" bson:"winRate"`
	TotalKills           int     `json:"totalKills" bson:"totalKills"`
	TotalDeaths          int     `json:"totalDeaths" bson:"totalDeaths"`
	TotalAssists         int     `json:"totalAssists" bson:"totalAssists"`
	AvgKills             float64 `json:"avgKills" bson:"avgKills"`
	AvgDeaths            float64 `json:"avgDeaths" bson:"avgDeaths"`
	AvgAssists           float64 `json:"avgAssists" bson:"avgAssists"`
	OverallKDA           float64 `json:"overallKDA" bson:"overallKDA"`
	AvgGameDuration      float64 `json:"avgGameDuration" bson:"avgGameDuration"`
	TotalGameTime        int64   `json:"totalGameTime" bson:"totalGameTime"`
	AvgVisionScore       float64 `json:"avgVisionScore" bson:"avgVisionScore"`
	AvgCSPerMin          float64 `json:"avgCSPerMin" bson:"avgCSPerMin"`
	AvgGoldPerMin        float64 `json:"avgGoldPerMin" bson:"avgGoldPerMin"`
	AvgDamageToChampions float64 `json:"avgDamageToChampions" bson:"avgDamageToChampions"`
	AvgKillParticipation float64 `json:"avgKillParticipation" bson:"avgKillParticipation"`
}

type RoleStats struct {
	Role                 string  `json:"role" bson:"role"`
	GamesPlayed          int     `json:"gamesPlayed" bson:"gamesPlayed"`
	Wins                 int     `json:"wins" bson:"wins"`
	Losses               int     `json:"losses" bson:"losses"`
	WinRate              float64 `json:"winRate" bson:"winRate"`
	TotalKills           int     `json:"totalKills" bson:"totalKills"`
	TotalDeaths          int     `json:"totalDeaths" bson:"totalDeaths"`
	TotalAssists         int     `json:"totalAssists" bson:"totalAssists"`
	AvgKills             float64 `json:"avgKills" bson:"avgKills"`
	AvgDeaths            float64 `json:"avgDeaths" bson:"avgDeaths"`
	AvgAssists           float64 `json:"avgAssists" bson:"avgAssists"`
	RoleKDA              float64 `json:"roleKDA" bson:"roleKDA"`
	AvgVisionScore       float64 `json:"avgVisionScore" bson:"avgVisionScore"`
	AvgCSPerMin          float64 `json:"avgCSPerMin" bson:"avgCSPerMin"`
	AvgGoldPerMin        float64 `json:"avgGoldPerMin" bson:"avgGoldPerMin"`
	AvgDamageToChampions float64 `json:"avgDamageToChampions" bson:"avgDamageToChampions"`
	AvgKillParticipation float64 `json:"avgKillParticipation" bson:"avgKillParticipation"`
}

type ChampionStats struct {
	ChampionName         string  `json:"championName" bson:"championName"`
	ChampionID           int     `json:"championId" bson:"championId"`
	GamesPlayed          int     `json:"gamesPlayed" bson:"gamesPlayed"`
	Wins                 int     `json:"wins" bson:"wins"`
	Losses               int     `json:"losses" bson:"losses"`
	WinRate              float64 `json:"winRate" bson:"winRate"`
	TotalKills           int     `json:"totalKills" bson:"totalKills"`
	TotalDeaths          int     `json:"totalDeaths" bson:"totalDeaths"`
	TotalAssists         int     `json:"totalAssists" bson:"totalAssists"`
	AvgKills             float64 `json:"avgKills" bson:"avgKills"`
	AvgDeaths            float64 `json:"avgDeaths" bson:"avgDeaths"`
	AvgAssists           float64 `json:"avgAssists" bson:"avgAssists"`
	ChampionKDA          float64 `json:"championKDA" bson:"championKDA"`
	BestKDA              float64 `json:"bestKDA" bson:"bestKDA"`
	WorstKDA             float64 `json:"worstKDA" bson:"worstKDA"`
	AvgVisionScore       float64 `json:"avgVisionScore" bson:"avgVisionScore"`
	AvgCSPerMin          float64 `json:"avgCSPerMin" bson:"avgCSPerMin"`
	AvgGoldPerMin        float64 `json:"avgGoldPerMin" bson:"avgGoldPerMin"`
	AvgDamageToChampions float64 `json:"avgDamageToChampions" bson:"avgDamageToChampions"`
	AvgKillParticipation float64 `json:"avgKillParticipation" bson:"avgKillParticipation"`
	LastPlayed           int64   `json:"lastPlayed" bson:"lastPlayed"`
}
