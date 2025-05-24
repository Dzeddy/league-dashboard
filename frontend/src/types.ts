export interface AccountDTO {
    puuid: string;
    gameName: string;
    tagLine: string;
}

export interface MatchMetadataDto {
    matchId: string;
    participants: string[];
}

export interface ParticipantChallengesDto {
    kda?: number;
    killParticipation?: number;
}

export interface StatPerksDto {
    defense: number;
    flex: number;
    offense: number;
}

export interface SelectionDto {
    perk: number;
    var1: number;
    var2: number;
    var3: number;
}

export interface StyleDto {
    description: string;
    selections: SelectionDto[];
    style: number;
}

export interface PerksDto {
    statPerks: StatPerksDto;
    styles: StyleDto[];
}

export interface ParticipantDto {
    puuid: string;
    summonerName?: string;
    riotIdGameName?: string;
    riotIdTagline?: string;
    championId: number;
    championName: string;
    teamId: number;
    win: boolean;
    kills: number;
    deaths: number;
    assists: number;
    totalMinionsKilled: number;
    neutralMinionsKilled: number;
    visionScore: number;
    goldEarned: number;
    challenges?: ParticipantChallengesDto;
    perks?: PerksDto;
    teamPosition: string;
    lane: string;
    item0: number;
    item1: number;
    item2: number;
    item3: number;
    item4: number;
    item5: number;
    item6: number; // Trinket
    summoner1Casts: number;
    summoner1Id: number;
    summoner2Casts: number;
    summoner2Id: number;
    champLevel: number;
    damageDealtToTurrets: number;
    damageDealtToObjectives: number;
    totalDamageDealtToChampions: number;
    totalDamageTaken: number;
    timePlayed: number;
}

export interface MatchInfoDto {
    gameCreation: number;
    gameDuration: number; // Duration in seconds
    gameEndTimestamp: number;
    gameId: number;
    gameMode: string;
    gameType: string;
    gameVersion: string;
    mapId: number;
    participants: ParticipantDto[];
    queueId: number;
    endOfGameResult?: string;
}

export interface MatchDto {
    metadata: MatchMetadataDto;
    info: MatchInfoDto;
}

export interface PlayerMatchStats {
    matchId: string;
    gameMode: string;
    gameCreation: number;
    gameDuration: number;
    championName: string;
    championId: number;
    win: boolean;
    kills: number;
    deaths: number;
    assists: number;
    kda: number;
    killParticipation?: number;
    totalMinionsKilled: number;
    visionScore: number;
    goldEarned: number;
    teamPosition: string;
    items: number[];
    summonerSpells: number[];
    primaryRune: number;
    secondaryStyle: number;
    champLevel: number;
    damageToTurrets: number;
    damageToObjectives: number;
    damageToChampions: number;
    totalDamageTaken: number;
    teamId: number; // 100 for blue, 200 for red
    queueId: number;
    // fullMatchData is not typically sent to frontend for this summary
}

export interface UserPerformance {
    puuid: string;
    region: string;
    riotId: string; // GameName#TagLine
    matches: PlayerMatchStats[];
    updatedAt: number;
}

// --- Static Data Dragon Types ---
export interface ChampionImageDTO {
    full: string;   // Aatrox.png
    sprite: string; // champion0.png
    group: string;  // champion
    x: number;
    y: number;
    w: number;
    h: number;
}

export interface ChampionData {
    version: string;
    id: string;      // e.g., "Aatrox"
    key: string;     // e.g., "266" (Champion ID)
    name: string;    // e.g., "Aatrox"
    title: string;
    image: ChampionImageDTO;
    partype: string; // Resource type (Mana, Energy, etc.)
    stats: Record<string, number>; // Champion base stats
}

export interface ItemImageDTO {
    full: string;  // e.g., "1001.png"
    sprite: string;
    group: string;
    x: number;
    y: number;
    w: number;
    h: number;
}

export interface ItemGoldDTO {
    base: number;
    purchasable: boolean;
    total: number;
    sell: number;
}

export interface ItemData {
    name: string;
    description: string; // This can be HTML, handle with care or use plaintext
    plaintext: string;
    image: ItemImageDTO;
    gold: ItemGoldDTO;
    tags: string[];
    maps: Record<string, boolean>;
    stats: Record<string, number>;
    depth?: number;
}

export interface RuneInfo {
    id: number;
    key: string; // e.g., "PressTheAttack"
    icon: string; // path to icon
    name: string;
    shortDesc: string; // Contains HTML
    longDesc: string;  // Contains HTML
}

export interface SummonerSpellImageDTO {
    full: string;  // e.g., "SummonerFlash.png"
    sprite: string;
    group: string;
    x: number;
    y: number;
    w: number;
    h: number;
}

export interface SummonerSpellData {
    id: string;   // e.g., "SummonerFlash"
    name: string; // e.g., "Flash"
    description: string;
    tooltip: string;
    key: string; // e.g., "4" for Flash (this is the ID used in match data)
    image: SummonerSpellImageDTO;
}

export interface StaticGameData {
    champions: Record<string, ChampionData>;      // Keyed by Champion Key (string version of ID)
    items: Record<string, ItemData>;              // Keyed by Item ID (string)
    runes: Record<number, RuneInfo>;              // Keyed by Rune ID (int)
    summonerSpells: Record<string, SummonerSpellData>; // Keyed by Summoner Spell Key (string version of ID)
    latestVersion: string;
}

// Enhanced Recent Games Summary Types
export interface RecentGamesSummary {
    puuid: string;
    region: string;
    riotId: string;
    totalMatches: number;
    overallStats: OverallStats;
    roleStats: Record<string, RoleStats>;
    championStats: Record<string, ChampionStats>;
    recentMatches: PlayerMatchStats[];
    lastUpdated: number;
}

export interface OverallStats {
    wins: number;
    losses: number;
    winRate: number;
    totalKills: number;
    totalDeaths: number;
    totalAssists: number;
    avgKills: number;
    avgDeaths: number;
    avgAssists: number;
    overallKDA: number;
    avgGameDuration: number;
    totalGameTime: number;
    avgVisionScore: number;
    avgCSPerMin: number;
    avgGoldPerMin: number;
    avgDamageToChampions: number;
    avgKillParticipation: number;
}

export interface RoleStats {
    role: string;
    gamesPlayed: number;
    wins: number;
    losses: number;
    winRate: number;
    totalKills: number;
    totalDeaths: number;
    totalAssists: number;
    avgKills: number;
    avgDeaths: number;
    avgAssists: number;
    roleKDA: number;
    avgVisionScore: number;
    avgCSPerMin: number;
    avgGoldPerMin: number;
    avgDamageToChampions: number;
    avgKillParticipation: number;
}

export interface ChampionStats {
    championName: string;
    championId: number;
    gamesPlayed: number;
    wins: number;
    losses: number;
    winRate: number;
    totalKills: number;
    totalDeaths: number;
    totalAssists: number;
    avgKills: number;
    avgDeaths: number;
    avgAssists: number;
    championKDA: number;
    bestKDA: number;
    worstKDA: number;
    avgVisionScore: number;
    avgCSPerMin: number;
    avgGoldPerMin: number;
    avgDamageToChampions: number;
    avgKillParticipation: number;
    lastPlayed: number;
} 