import React, { useState, useEffect, FormEvent, useCallback } from 'react';
import axios from 'axios';
import './App.css';
import { UserPerformance, StaticGameData, PlayerMatchStats, ChampionData, ItemData, SummonerSpellData, RuneInfo } from './types';
import dayjs from 'dayjs';
// @ts-ignore
import relativeTime from 'dayjs/plugin/relativeTime';
dayjs.extend(relativeTime);

const API_BASE_URL = 'http://localhost:8080/api';

interface AggregatedStats {
  winRate: number;
  wins: number;
  losses: number;
  matchesPlayed: number;
  kda: number;
  avgKills: number;
  avgDeaths: number;
  avgAssists: number;
  totalKills: number;
  totalDeaths: number;
  totalAssists: number;
}

function calculateAggregatedStats(matches: PlayerMatchStats[]): AggregatedStats {
  if (!matches || matches.length === 0) {
    return {
      winRate: 0, wins: 0, losses: 0, matchesPlayed: 0, kda: 0,
      avgKills: 0, avgDeaths: 0, avgAssists: 0,
      totalKills: 0, totalDeaths: 0, totalAssists: 0,
    };
  }

  const totalMatches = matches.length;
  const wins = matches.filter(m => m.win).length;
  const losses = totalMatches - wins;
  const winRate = totalMatches > 0 ? (wins / totalMatches) * 100 : 0;

  const totalKills = matches.reduce((sum, m) => sum + m.kills, 0);
  const totalDeaths = matches.reduce((sum, m) => sum + m.deaths, 0);
  const totalAssists = matches.reduce((sum, m) => sum + m.assists, 0);

  const avgKills = totalMatches > 0 ? totalKills / totalMatches : 0;
  const avgDeaths = totalMatches > 0 ? totalDeaths / totalMatches : 0;
  const avgAssists = totalMatches > 0 ? totalAssists / totalMatches : 0;

  // KDA: (Kills + Assists) / Deaths. If Deaths is 0, treat as (Kills + Assists) / 1.
  const kda = totalDeaths === 0 ? (totalKills + totalAssists) : (totalKills + totalAssists) / totalDeaths;

  return {
    winRate,
    wins,
    losses,
    matchesPlayed: totalMatches,
    kda,
    avgKills,
    avgDeaths,
    avgAssists,
    totalKills,
    totalDeaths,
    totalAssists,
  };
}

function App() {
  const [gameName, setGameName] = useState('');
  const [tagLine, setTagLine] = useState('');
  const [region, setRegion] = useState('na1'); // Default to NA1
  const [playerData, setPlayerData] = useState<UserPerformance | null>(null);
  const [staticData, setStaticData] = useState<StaticGameData | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isStaticDataLoading, setIsStaticDataLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loadedImages, setLoadedImages] = useState<Set<string>>(new Set());
  const [preloadedItems, setPreloadedItems] = useState<Set<number>>(new Set());
  const [visibleMatches, setVisibleMatches] = useState(25);
  const [selectedMatch, setSelectedMatch] = useState<PlayerMatchStats | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  // Derived stats
  const [lifetimeStats, setLifetimeStats] = useState<AggregatedStats | null>(null);
  const [currentSeasonStats, setCurrentSeasonStats] = useState<AggregatedStats | null>(null);
  // const [filteredSeasonalStats, setFilteredSeasonalStats] = useState<AggregatedStats | null>(null);

  const [fullMatchData, setFullMatchData] = useState<any | null>(null);
  const [isMatchLoading, setIsMatchLoading] = useState(false);

  // Preload common items when static data is loaded
  useEffect(() => {
    const fetchAndSetPopularItems = async () => {
      if (staticData) {
        try {
          // Assuming the endpoint returns an array of item IDs: number[]
          const response = await axios.get<number[]>(`${API_BASE_URL}/popular-items`);
          if (response.data && Array.isArray(response.data) && response.data.length > 0) {
            setPreloadedItems(new Set(response.data));
            console.log(`Successfully fetched and set ${response.data.length} popular items for preloading.`);
          } else {
            console.warn("Popular items response was empty, malformed, or contained no items. Falling back to default preload list.");
            const fallbackItems = new Set<number>([1001, 2003, 2010, 2031, 2033, 2055, 2065, 2138, 2139, 2140]);
            setPreloadedItems(fallbackItems);
          }
        } catch (err) {
          console.error("Error fetching popular items for preloading, falling back to default list:", err);
          const fallbackItemsOnError = new Set<number>([1001, 2003, 2010, 2031, 2033, 2055, 2065, 2138, 2139, 2140]);
          setPreloadedItems(fallbackItemsOnError);
        }
      }
    };

    fetchAndSetPopularItems();
  }, [staticData]);

  const preloadImage = useCallback((url: string, key: string) => {
    if (!url || loadedImages.has(key)) return;
    
    const img = new Image();
    img.onload = () => handleImageLoad(key);
    img.onerror = () => handleImageLoad(key); // Still mark as loaded to prevent infinite loading state
    img.src = url;
  }, [loadedImages]);

  const handleImageLoad = useCallback((imageKey: string) => {
    setLoadedImages(prev => new Set(prev).add(imageKey));
  }, []);

  const isImageLoaded = useCallback((imageKey: string) => {
    return loadedImages.has(imageKey);
  }, [loadedImages]);

  // Preload item images for a match
  const preloadMatchItems = useCallback((match: PlayerMatchStats) => {
    if (!staticData) return;
    
    match.items.forEach(itemId => {
      const itemImageKey = `item-${itemId}`;
      if (!isImageLoaded(itemImageKey)) {
        const itemUrl = `${dataDragonBase}/${getItemImageURL(itemId)}`;
        preloadImage(itemUrl, itemImageKey);
      }
    });
  }, [staticData, isImageLoaded, preloadImage]);

  // Fetch static data on component mount
  useEffect(() => {
    const fetchStaticData = async () => {
      try {
        setIsStaticDataLoading(true);
        const response = await axios.get<StaticGameData>(`${API_BASE_URL}/static-data`);
        setStaticData(response.data);
        console.log("Static data loaded:", response.data);
      } catch (err) {
        console.error("Error fetching static data:", err);
        setError("Could not load essential game data. Please try refreshing.");
      } finally {
        setIsStaticDataLoading(false);
      }
    };
    fetchStaticData();
  }, []);

  useEffect(() => {
    if (playerData?.matches) {
      // For now, all stats are based on the fetched matches.
      // Filtering by selectedPlaylist or selectedSeason would happen here.
      const allFetchedMatchesStats = calculateAggregatedStats(playerData.matches);
      setLifetimeStats(allFetchedMatchesStats);
      setCurrentSeasonStats(allFetchedMatchesStats); // Replace with actual current season logic if available
      // setFilteredSeasonalStats(allFetchedMatchesStats); // Replace with specific season filtering
    }
  }, [playerData]);

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!gameName || !tagLine || !region) {
      setError("Please enter Game Name, Tagline, and Region.");
      return;
    }
    if (!staticData) {
      setError("Game data is still loading. Please wait a moment and try again.");
      return;
    }
    
    setIsLoading(true);
    setError(null);
    setPlayerData(null);
    setLifetimeStats(null);
    setCurrentSeasonStats(null);

    try {
      const response = await axios.get<UserPerformance>(`${API_BASE_URL}/player/${region}/${gameName}/${tagLine}/matches?count=25`);
      setPlayerData(response.data);
      console.log("Player data:", response.data);
    } catch (err: any) {
      console.error("Error fetching player data:", err);
      if (err.response && err.response.data) {
        setError(`Error: ${err.response.data.error || err.response.data}`);
      } else if (err.request) {
        setError("Error: No response from server. Is the backend running?");
      } else {
        setError(`Error: ${err.message}`);
      }
    } finally {
      setIsLoading(false);
    }
  };

  const getChampionImageURL = (championName: string, championId?: number) => {
    if (!staticData) return 'placeholder.png';
    
    // Log for debugging champion name issues
    if (championName && (championName.includes("'") || championName.includes(" "))) {
      console.log(`Looking up champion: ${championName} (ID: ${championId})`);
    }
    
    // Prefer lookup by championId (key) if available - this is most reliable
    if (championId !== undefined && championId !== null) {
      const champ = staticData.champions[String(championId)];
      if (champ) {
        console.log(`Found champion by ID ${championId}: ${champ.id} (${champ.name})`);
        return `${staticData.latestVersion}/img/champion/${champ.image.full}`;
      } else {
        console.warn(`Champion ID ${championId} not found in static data. Available keys:`, Object.keys(staticData.champions).slice(0, 5));
      }
    }
    
    // Fallback: try to find by name (case-insensitive and handle special characters)
    if (championName) {
      const championKey = Object.keys(staticData.champions).find(key => {
        const champData = staticData.champions[key];
        // Try exact match first
        if (champData.name === championName) return true;
        // Try case-insensitive match
        if (champData.name.toLowerCase() === championName.toLowerCase()) return true;
        // Try ID match (for cases where championName might actually be the ID)
        if (champData.id === championName) return true;
        if (champData.id.toLowerCase() === championName.toLowerCase()) return true;
        // Try match without spaces/apostrophes for names like "Cho'Gath" -> "Chogath"
        const normalizedDataName = champData.name.toLowerCase().replace(/['\s]/g, '');
        const normalizedInputName = championName.toLowerCase().replace(/['\s]/g, '');
        if (normalizedDataName === normalizedInputName) return true;
        // Try ID without special chars
        const normalizedDataId = champData.id.toLowerCase().replace(/['\s]/g, '');
        if (normalizedDataId === normalizedInputName) return true;
        return false;
      });
      
      if (championKey) {
        const champ = staticData.champions[championKey];
        console.log(`Found champion by name "${championName}": ${champ.id} (${champ.name})`);
        return `${staticData.latestVersion}/img/champion/${champ.image.full}`;
      } else {
        console.warn(`Champion "${championName}" not found in static data`);
        // Log some available champions for debugging
        const sampleChamps = Object.values(staticData.champions).slice(0, 3);
        console.log(`Sample champions:`, sampleChamps.map(c => ({ id: c.id, name: c.name, key: c.key })));
      }
    }
    
    return 'placeholder.png';
  };

  const getItemImageURL = (itemId: number) => {
    if (!staticData || itemId === 0) return 'placeholder.png';
    const item = staticData.items[itemId.toString()];
    if (item) {
      return `${staticData.latestVersion}/img/item/${item.image.full}`;
    }
    return 'placeholder.png';
  };
  
  const getSummonerSpellImageURL = (spellId: number) => {
    if (!staticData || spellId === 0) return 'placeholder.png';
    // Summoner spell IDs in match data are numeric keys, map to string key for staticData
    const spell = staticData.summonerSpells[spellId.toString()]; 
    if (spell) {
        return `${staticData.latestVersion}/img/spell/${spell.image.full}`;
    }
    return 'placeholder.png';
  };

  const getRuneImageURL = (runeId: number) => {
    if (!staticData || runeId === 0) return 'placeholder.png';
    const rune = staticData.runes[runeId];
    if (rune) {
        // Icons in DDragon rune data are full paths already, e.g., "perk-images/Styles/Precision/PressTheAttack/PressTheAttack.png"
        return `img/${rune.icon}`;
    }
    return 'placeholder.png';
  };

  const formatGameDuration = (durationInSeconds: number): string => {
    const minutes = Math.floor(durationInSeconds / 60);
    const seconds = durationInSeconds % 60;
    return `${minutes}m ${seconds < 10 ? '0' : ''}${seconds}s`;
  };

  const dataDragonBase = "https://ddragon.leagueoflegends.com/cdn";

  const renderItemImage = (itemId: number, index: number, isTrinket: boolean = false) => {
    // If itemId is 0, render a placeholder immediately
    if (itemId === 0) {
      return (
        <div 
          key={`item-${index}-${itemId}`} 
          className={`item-container ${isTrinket ? 'trinket-container' : ''}`}
        >
          <div className="item-icon empty-item" />
        </div>
      );
    }

    const itemImageKey = `item-${itemId}`;
    const itemUrl = staticData ? `${dataDragonBase}/${getItemImageURL(itemId)}` : '';
    
    // Start preloading if not already loaded
    if (itemUrl && !isImageLoaded(itemImageKey)) {
      preloadImage(itemUrl, itemImageKey);
    }

    return (
      <div 
        key={`item-${index}-${itemId}`} 
        className={`item-container ${isTrinket ? 'trinket-container' : ''}`}
      >
        <img 
          src={itemUrl}
          alt={`Item ${itemId}`}
          className={`item-icon ${!isImageLoaded(itemImageKey) ? 'loading' : ''} ${isTrinket ? 'trinket-icon' : ''}`}
          onLoad={() => handleImageLoad(itemImageKey)}
          onError={(e) => {
            e.currentTarget.src = 'placeholder.png';
            handleImageLoad(itemImageKey);
          }}
        />
      </div>
    );
  };

  // Helper: Map teamPosition/lane to role icon URL
  const getRoleIconURL = (role: string) => {
    // Use Data Dragon or static mapping for role icons
    // Example mapping (replace URLs with actual Data Dragon or your own icons as needed)
    const roleMap: { [key: string]: string } = {
      TOP: 'https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/position-top.png',
      JUNGLE: 'https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/position-jungle.png',
      MIDDLE: 'https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/position-middle.png',
      MID: 'https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/position-middle.png',
      BOTTOM: 'https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/position-bottom.png',
      ADC: 'https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/position-bottom.png',
      SUPPORT: 'https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/position-utility.png',
      UTILITY: 'https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/position-utility.png',
      NONE: '',
    };
    return roleMap[role?.toUpperCase()] || '';
  };

  const renderMatchCard = (match: PlayerMatchStats) => {
    // Preload items for this match
    preloadMatchItems(match);

    const championImageKey = `champion-${match.championName}`;
    const championImageUrl = staticData ? `${dataDragonBase}/${getChampionImageURL(match.championName, match.championId)}` : '';
    
    // Determine role icon
    const role = match.teamPosition || 'NONE';
    const roleIconUrl = getRoleIconURL(role);

    // Summoner spell icons
    const spellIcons = match.summonerSpells.map((spellId, index) => {
      const spellImageKey = `spell-${spellId}`;
      return (
        <img
          key={`spell-${index}-${spellId}`}
          src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(spellId)}` : ''}
          alt={`Summoner Spell ${spellId}`}
          className={`spell-icon stacked ${!isImageLoaded(spellImageKey) ? 'loading' : ''}`}
          onLoad={() => handleImageLoad(spellImageKey)}
          onError={(e) => (e.currentTarget.src = 'placeholder.png')}
          style={{ display: 'block', marginBottom: index === 0 ? 2 : 0 }}
        />
      );
    });

    return (
      <div key={match.matchId} className={`match-card ${match.win ? 'win' : 'loss'} ${!isImageLoaded(championImageKey) ? 'loading' : ''}`}>
        <div className="match-header">
          <div className="champion-icon-container-with-spells">
            <div className="summoner-spells-vertical">
              {spellIcons}
            </div>
            <div className="champion-icon-container">
              <img
                src={championImageUrl}
                alt={match.championName}
                className={`champion-icon ${!isImageLoaded(championImageKey) ? 'loading' : ''}`}
                onLoad={() => handleImageLoad(championImageKey)}
                onError={(e) => (e.currentTarget.src = 'placeholder.png')}
              />
              {roleIconUrl && (
                <img src={roleIconUrl} alt={role} className="role-icon" />
              )}
            </div>
          </div>
          <div className="champion-details">
            <span className="champion-name">{match.championName}</span>
            <span className="game-mode">{match.gameMode.replace('_', ' ')} - {match.win ? "Victory" : "Defeat"}</span>
            <span className="game-duration">{formatGameDuration(match.gameDuration)}</span>
          </div>
          <div className="kda">
            <span>{match.kills} / {match.deaths} / {match.assists}</span>
            <span>{match.kda.toFixed(2)} KDA</span>
          </div>
        </div>
        <div className="match-body">
          <div className="runes-spells">
            {/* Summoner spells now shown next to champ icon, so skip here */}
            <div className="runes">
              {(() => {
                const runeImageKey = `rune-${match.primaryRune}`;
                return (
                  <img
                    src={staticData ? `${dataDragonBase}/${getRuneImageURL(match.primaryRune)}` : ''}
                    alt={`Primary Rune ${match.primaryRune}`}
                    className={`rune-icon ${!isImageLoaded(runeImageKey) ? 'loading' : ''}`}
                    onLoad={() => handleImageLoad(runeImageKey)}
                    onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                  />
                );
              })()}
            </div>
          </div>
          <div className="items">
            {match.items.slice(0, 6).map((itemId, index) => renderItemImage(itemId, index))}
            {renderItemImage(match.items[6], 6, true)}
          </div>
          <div className="match-stats-grid">
            <span>CS: {match.totalMinionsKilled}</span>
            <span>Gold: {match.goldEarned.toLocaleString()}</span>
            <span>Level: {match.champLevel}</span>
            <span>Vision: {match.visionScore}</span>
            <span>Dmg to Champs: {match.damageToChampions.toLocaleString()}</span>
            <span>Dmg Taken: {match.totalDamageTaken.toLocaleString()}</span>
          </div>
        </div>
      </div>
    );
  };

  const renderStatBox = (label: string, value: string | number, subLabel?: string) => (
    <div className="stat-box">
      <span className="stat-value">{value}</span>
      <span className="stat-label">{label}</span>
      {subLabel && <span className="stat-sublabel">{subLabel}</span>}
    </div>
  );
  
  // Helper: Get champion icon URL - now uses the improved getChampionImageURL function
  const getChampionIcon = (championName: string, championId?: number) => {
    const imageUrl = getChampionImageURL(championName, championId);
    return imageUrl === 'placeholder.png' ? 'placeholder.png' : `https://ddragon.leagueoflegends.com/cdn/${imageUrl}`;
  };

  // Helper: Format match time (e.g., '19m ago')
  const formatTimeAgo = (timestamp: number) => {
    return dayjs(timestamp).fromNow();
  };

  // Helper: Group matches by date (YYYY-MM-DD)
  const groupMatchesByDate = (matches: PlayerMatchStats[]) => {
    const groups: { [date: string]: PlayerMatchStats[] } = {};
    matches.forEach(match => {
      const date = dayjs(match.gameCreation).format('MMM D');
      if (!groups[date]) groups[date] = [];
      groups[date].push(match);
    });
    return groups;
  };

  // Helper: Calculate CS/min
  const getCsPerMin = (match: PlayerMatchStats) => {
    if (!match.gameDuration) return '0.00';
    return (match.totalMinionsKilled / (match.gameDuration / 60)).toFixed(2);
  };

  // Helper: Map game mode for display
  const getDisplayGameMode = (gameMode: string, queueId?: number) => {
    if (gameMode === 'CHERRY' || queueId === 1700) return 'Arena';
    // Add more mappings if needed
    return gameMode.replace('_', ' ').charAt(0).toUpperCase() + gameMode.replace('_', ' ').slice(1).toLowerCase();
  };

  // Helper: Get place for Arena (if available)
  const getArenaPlace = (match: PlayerMatchStats) => {
    // If you have place info in match, use it. Otherwise, return '-'.
    // Example: match.place or match.arenaPlace
    return (match as any).place || (match as any).arenaPlace || '-';
  };

  // Helper: Render items for a player
  const renderItemIcons = (items: number[]) => {
    if (!staticData) return null;
    return items.slice(0, 6).map((itemId, idx) => {
      if (itemId === 0) {
        return <div key={idx} className="item-icon empty-item" />;
      }
      const item = staticData.items[itemId.toString()];
      const url = item ? `https://ddragon.leagueoflegends.com/cdn/${staticData.latestVersion}/img/item/${item.image.full}` : 'placeholder.png';
      return <img key={idx} src={url} alt={`Item ${itemId}`} className="item-icon" onError={e => (e.currentTarget.src = 'placeholder.png')} />;
    });
  };

  // Fetch full match data on card click
  const handleMatchCardClick = async (match: PlayerMatchStats) => {
    setSelectedMatch(match);
    setIsModalOpen(true);
    setIsMatchLoading(true);
    setFullMatchData(null);
    try {
      const response = await axios.get(`${API_BASE_URL}/match/${region}/${match.matchId}`);
      setFullMatchData(response.data);
    } catch (err) {
      setFullMatchData(null);
    } finally {
      setIsMatchLoading(false);
    }
  };

  // Render: Recent Matches Section
  const renderRecentMatches = () => {
    if (!playerData || !playerData.matches || playerData.matches.length === 0 || !staticData) {
      if (playerData && (!playerData.matches || playerData.matches.length === 0)) {
        return <div className="recent-matches-section"><p>No recent matches found for this player.</p></div>;
      }
      return null;
    }
    const matchesToDisplay = playerData.matches.slice(0, visibleMatches);
    const grouped = groupMatchesByDate(matchesToDisplay);
    const dates = Object.keys(grouped);

    return (
      <div className="recent-matches-section">
        <h3>RECENT MATCHES</h3>
        {dates.length > 0 ? dates.map(date => {
          const matchesOnDate = grouped[date];
          const wins = matchesOnDate.filter(m => m.win).length;
          const losses = matchesOnDate.length - wins;
          return (
            <div key={date} className="match-date-group">
              <div className="match-date-header">
                <span>{date}</span>
                <span className="match-date-summary">{wins} <span className="win-text">W</span> - {losses} <span className="loss-text">L</span></span>
              </div>
              {matchesOnDate.map((match) => {
                const champIcon = getChampionIcon(match.championName, match.championId);
                const isArena = match.gameMode === 'CHERRY' || match.queueId === 1700;
                return (
                  <div
                    key={match.matchId}
                    className={`recent-match-card ${match.win ? 'win' : 'loss'}`}
                    onClick={() => handleMatchCardClick(match)}
                    style={{ cursor: 'pointer' }}
                  >
                    <div className="recent-match-left">
                      <div className="recent-champ-section">
                        <div className="recent-champ-with-spells">
                          <div className="recent-summoner-spells">
                            {match.summonerSpells.map((spellId, index) => (
                              <img
                                key={`spell-${index}-${spellId}`}
                                src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(spellId)}` : ''}
                                alt={`Summoner Spell ${spellId}`}
                                className="recent-summoner-spell"
                                onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                              />
                            ))}
                          </div>
                          <div className="recent-champ-info">
                            <img src={champIcon} alt={match.championName} className="recent-champ-icon" onError={e => (e.currentTarget.src = 'placeholder.png')} />
                            <span className="recent-champ-level">Lv.{match.champLevel}</span>
                          </div>
                        </div>
                        <div className="recent-match-meta">
                          <span className="recent-match-map">{match.championName}</span>
                          <span className="recent-match-time">{formatTimeAgo(match.gameCreation)}</span>
                          <span className="recent-match-mode">{getDisplayGameMode(match.gameMode, match.queueId)}</span>
                        </div>
                      </div>
                    </div>
                    <div className="recent-match-center">
                      <div className="recent-kda-section">
                        <span className="recent-match-kda">{match.kills}/{match.deaths}/{match.assists}</span>
                        <span className="recent-kda-ratio">{match.kda.toFixed(2)} KDA</span>
                      </div>
                      {!isArena && (
                        <span className="recent-match-cs">CS/min: {getCsPerMin(match)}</span>
                      )}
                      {isArena && (
                        <span className="recent-match-arena-place">Place: {getArenaPlace(match)}</span>
                      )}
                      <span className="recent-match-length">{formatGameDuration(match.gameDuration)}</span>
                    </div>
                    <div className="recent-match-right">
                      <div className="recent-items-preview">
                        {match.items.slice(0, 3).map((itemId, index) => (
                          itemId > 0 ? (
                            <img
                              key={index}
                              src={staticData ? `${dataDragonBase}/${getItemImageURL(itemId)}` : ''}
                              alt={`Item ${itemId}`}
                              className="recent-item-icon"
                              onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                            />
                          ) : (
                            <div key={index} className="recent-item-icon empty" />
                          )
                        ))}
                      </div>
                      <span className="recent-match-menu">⋮</span>
                    </div>
                  </div>
                );
              })}
            </div>
          );
        }) : (
          playerData && playerData.matches.length > 0 && <p>All matches shown or no matches for current view.</p>
        )}
        {playerData && playerData.matches.length > visibleMatches && (
          <button className="view-more-btn" onClick={() => setVisibleMatches(visibleMatches + 10)}>View More</button>
        )}
      </div>
    );
  };

  // Helper: Render modal content for a match
  const renderMatchModal = () => {
    if (!selectedMatch || !staticData) return null;
    const isArena = selectedMatch.gameMode === 'CHERRY' || selectedMatch.queueId === 1700;
    return (
      <div className="modal-overlay" onClick={() => setIsModalOpen(false)}>
        <div className="modal-content" onClick={e => e.stopPropagation()}>
          <button className="modal-close" onClick={() => setIsModalOpen(false)}>×</button>
          <h2>Match Details</h2>
          <div className="modal-match-meta">
            <span>{getDisplayGameMode(selectedMatch.gameMode, selectedMatch.queueId)}</span>
            <span>{formatGameDuration(selectedMatch.gameDuration)}</span>
          </div>
          {isMatchLoading ? (
            <div>Loading match details...</div>
          ) : isArena ? (
            fullMatchData && fullMatchData.info && fullMatchData.info.participants ? (
              <div className="arena-modal-content">
                {/* Group participants by teamId, sort teams by placement */}
                {(() => {
                  // Group by teamId
                  const teams: { [teamId: number]: any[] } = {};
                  fullMatchData.info.participants.forEach((p: any) => {
                    if (!teams[p.teamId]) teams[p.teamId] = [];
                    teams[p.teamId].push(p);
                  });
                  // Build array of teams with placement
                  const teamArr = Object.values(teams).map((members: any[]) => ({
                    placement: members[0].placement || members[0].place || 99,
                    teamId: members[0].teamId,
                    members,
                  }));
                  // Sort by placement (ascending)
                  teamArr.sort((a, b) => a.placement - b.placement);
                  return teamArr.map((team, idx) => (
                    <div key={team.teamId} className="arena-team-group">
                      <div className="arena-team-header">
                        <span className="arena-team-place">Place: {team.placement}</span>
                      </div>
                      <div className="arena-team-members">
                        {team.members.map((p: any, i: number) => (
                          <div key={i} className="arena-player-card enhanced">
                            <div className="arena-player-header-enhanced">
                              <div className="player-champ-info">
                                <img
                                  src={staticData ? `${dataDragonBase}/${getChampionImageURL(p.championName, p.championId)}` : ''}
                                  alt={p.championName}
                                  className="modal-champion-portrait"
                                  onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                                />
                                <div className="player-name-champ">
                                  <span className="arena-player-name">{p.riotIdGameName || p.summonerName || 'Unknown Player'}</span>
                                  <span className="arena-player-champ">{p.championName}</span>
                                  <span className="player-level">Level {p.champLevel}</span>
                                </div>
                              </div>
                              <div className="summoner-spells-modal">
                                <img
                                  src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(p.summoner1Id)}` : ''}
                                  alt="Summoner Spell 1"
                                  className="summoner-spell-modal"
                                  onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                                />
                                <img
                                  src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(p.summoner2Id)}` : ''}
                                  alt="Summoner Spell 2"
                                  className="summoner-spell-modal"
                                  onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                                />
                              </div>
                            </div>
                            <div className="arena-player-stats">
                              <div className="arena-player-kda">
                                <span className="kda-numbers">{p.kills}/{p.deaths}/{p.assists}</span>
                                <span className="kda-ratio">KDA: {p.deaths > 0 ? ((p.kills + p.assists) / p.deaths).toFixed(2) : (p.kills + p.assists)}</span>
                              </div>
                              <div className="arena-player-items">{renderItemIcons([p.item0, p.item1, p.item2, p.item3, p.item4, p.item5])}</div>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  ));
                })()}
              </div>
            ) : (
              <div>Full match data not available or participants missing.</div>
            )
          ) : (
            fullMatchData && fullMatchData.info && fullMatchData.info.participants ? (
              <div className="rift-modal-content">
                <div className="teams-container">
                  <div className="team team-blue">
                    <h4>Blue Team</h4>
                    {fullMatchData.info.participants.filter((p: any) => p.teamId === 100).map((p: any, idx: number) => (
                      <div key={idx} className="player-card-modal enhanced">
                        <div className="player-modal-header">
                          <div className="player-champ-info">
                            <img
                              src={staticData ? `${dataDragonBase}/${getChampionImageURL(p.championName, p.championId)}` : ''}
                              alt={p.championName}
                              className="modal-champion-portrait"
                              onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                            />
                            <div className="player-name-champ">
                              <span className="player-name-modal">{p.summonerName || p.riotIdGameName || '-'}</span>
                              <span className="player-champ-modal">{p.championName}</span>
                              <span className="player-level">Level {p.champLevel}</span>
                            </div>
                          </div>
                          <div className="summoner-spells-modal">
                            <img
                              src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(p.summoner1Id)}` : ''}
                              alt="Summoner Spell 1"
                              className="summoner-spell-modal"
                              onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                            />
                            <img
                              src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(p.summoner2Id)}` : ''}
                              alt="Summoner Spell 2"
                              className="summoner-spell-modal"
                              onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                            />
                          </div>
                        </div>
                        <div className="player-stats-grid">
                          <div className="kda-section">
                            <span className="kda-numbers">K/D/A: {p.kills}/{p.deaths}/{p.assists}</span>
                            <span className="kda-ratio">KDA: {p.deaths > 0 ? ((p.kills + p.assists) / p.deaths).toFixed(2) : (p.kills + p.assists)}</span>
                          </div>
                          <div className="additional-stats">
                            <span>Gold: {p.goldEarned}</span>
                            <span>CS/min: {((p.totalMinionsKilled + p.neutralMinionsKilled) / (selectedMatch.gameDuration / 60)).toFixed(2)}</span>
                          </div>
                        </div>
                        <div className="player-items-modal">{renderItemIcons([p.item0, p.item1, p.item2, p.item3, p.item4, p.item5])}</div>
                      </div>
                    ))}
                  </div>
                  <div className="team team-red">
                    <h4>Red Team</h4>
                    {fullMatchData.info.participants.filter((p: any) => p.teamId === 200).map((p: any, idx: number) => (
                      <div key={idx} className="player-card-modal enhanced">
                        <div className="player-modal-header">
                          <div className="player-champ-info">
                            <img
                              src={staticData ? `${dataDragonBase}/${getChampionImageURL(p.championName, p.championId)}` : ''}
                              alt={p.championName}
                              className="modal-champion-portrait"
                              onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                            />
                            <div className="player-name-champ">
                              <span className="player-name-modal">{p.summonerName || p.riotIdGameName || '-'}</span>
                              <span className="player-champ-modal">{p.championName}</span>
                              <span className="player-level">Level {p.champLevel}</span>
                            </div>
                          </div>
                          <div className="summoner-spells-modal">
                            <img
                              src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(p.summoner1Id)}` : ''}
                              alt="Summoner Spell 1"
                              className="summoner-spell-modal"
                              onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                            />
                            <img
                              src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(p.summoner2Id)}` : ''}
                              alt="Summoner Spell 2"
                              className="summoner-spell-modal"
                              onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                            />
                          </div>
                        </div>
                        <div className="player-stats-grid">
                          <div className="kda-section">
                            <span className="kda-numbers">K/D/A: {p.kills}/{p.deaths}/{p.assists}</span>
                            <span className="kda-ratio">KDA: {p.deaths > 0 ? ((p.kills + p.assists) / p.deaths).toFixed(2) : (p.kills + p.assists)}</span>
                          </div>
                          <div className="additional-stats">
                            <span>Gold: {p.goldEarned}</span>
                            <span>CS/min: {((p.totalMinionsKilled + p.neutralMinionsKilled) / (selectedMatch.gameDuration / 60)).toFixed(2)}</span>
                          </div>
                        </div>
                        <div className="player-items-modal">{renderItemIcons([p.item0, p.item1, p.item2, p.item3, p.item4, p.item5])}</div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            ) : (
              <div>Full match data not available.</div>
            )
          )}
        </div>
      </div>
    );
  };

  return (
    <div className="App">
      <main className="App-main-r6">
        <section className="search-section-r6">
          <form onSubmit={handleSubmit} className="search-form-r6">
            <div className="form-group-r6">
              <input
                type="text"
                value={gameName}
                onChange={(e) => setGameName(e.target.value)}
                placeholder="Game Name"
                required
                disabled={isStaticDataLoading}
              />
              <span className="tagline-separator-r6">#</span>
              <input
                type="text"
                value={tagLine}
                onChange={(e) => setTagLine(e.target.value)}
                placeholder="Tagline"
                required
                disabled={isStaticDataLoading}
              />
            </div>
            <select value={region} onChange={(e) => setRegion(e.target.value)} className="region-select-r6" disabled={isStaticDataLoading}>
              <option value="na1">NA</option>
              <option value="euw1">EUW</option>
              <option value="eun1">EUNE</option>
              <option value="kr">KR</option>
              {/* Add more regions */}
            </select>
            <button type="submit" disabled={isLoading || isStaticDataLoading} className="search-button-r6">
              {isStaticDataLoading ? 'Loading Data...' : isLoading ? 'Searching...' : 'Search'}
            </button>
          </form>
          {error && <p className="error-message-r6">{error}</p>}
        </section>

        {isLoading && <div className="loading-message-r6">Fetching player data...</div>}
        
        {playerData && !isLoading && (
          <div className="stats-overview-container">
            {/* 
              If you want to keep lifetime/seasonal stats, they would go here. 
              Example structure based on previous code:
            */}
            {/* 
            <div className="stats-layout-r6">
              <aside className="current-season-r6">
                <h3>CURRENT SEASON</h3>
                {currentSeasonStats ? (
                  <table> ... </table>
                ) : <p>No current season data.</p>}
              </aside>
              <section className="main-stats-r6">
                <div className="lifetime-overview-r6">...</div>
                <div className="seasonal-overview-r6">...</div>
              </section>
            </div>
            */}
          </div>
        )}

        {/* Recent Matches will always be attempted to render if playerData exists */}
        {/* It handles its own loading/no data states internally */}
        {renderRecentMatches()} 

        {!playerData && !isLoading && !isStaticDataLoading && !error && (
             <p className="initial-prompt-r6">Enter a Riot ID (Game Name#Tagline) and region to see player stats.</p>
        )}
      </main>
      <footer className="App-footer-r6">
        <p>League Performance Tracker - Data by Riot Games API & Data Dragon.</p>
      </footer>
      {isModalOpen && selectedMatch && renderMatchModal()}
    </div>
  );
}

export default App;
