import React, { useState, useEffect, FormEvent } from 'react';
import axios from 'axios';
import './App.css';
import { UserPerformance, StaticGameData, PlayerMatchStats, RecentGamesSummary, PlayerDashboardData } from './types';
import dayjs from 'dayjs';
// @ts-ignore
import relativeTime from 'dayjs/plugin/relativeTime';
dayjs.extend(relativeTime);

const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080/api';

// Type definitions for sorting functionality
type SortColumn = 'champion' | 'games' | 'winrate' | 'kda' | 'cs' | 'damage' | 'lastPlayed';
type SortDirection = 'asc' | 'desc';

// interface AggregatedStats {
//   winRate: number;
//   wins: number;
//   losses: number;
//   matchesPlayed: number;
//   kda: number;
//   avgKills: number;
//   avgDeaths: number;
//   avgAssists: number;
//   totalKills: number;
//   totalDeaths: number;
//   totalAssists: number;
// }

function App() {
  const [gameName, setGameName] = useState('');
  const [tagLine, setTagLine] = useState('');
  const [region, setRegion] = useState('na1'); // Default to NA1
  const [playerData, setPlayerData] = useState<UserPerformance | null>(null);
  const [recentGamesSummary, setRecentGamesSummary] = useState<RecentGamesSummary | null>(null);
  const [staticData, setStaticData] = useState<StaticGameData | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isStaticDataLoading, setIsStaticDataLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // const [loadedImages, setLoadedImages] = useState<Set<string>>(new Set());
  const [visibleMatches, setVisibleMatches] = useState(25);
  const [selectedMatch, setSelectedMatch] = useState<PlayerMatchStats | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<'overview' | 'roles' | 'champions' | 'matches'>('overview');

  const [fullMatchData, setFullMatchData] = useState<any | null>(null);
  const [isMatchLoading, setIsMatchLoading] = useState(false);

  // Sorting state for champion performance table
  const [sortColumn, setSortColumn] = useState<SortColumn>('games');
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc');

  // Derived stats (these are calculated but will be used later for filtering/display)
  // const [lifetimeStats, setLifetimeStats] = useState<AggregatedStats | null>(null);
  // const [currentSeasonStats, setCurrentSeasonStats] = useState<AggregatedStats | null>(null);
  // const [filteredSeasonalStats, setFilteredSeasonalStats] = useState<AggregatedStats | null>(null);

  // Preload common items when static data is loaded
  useEffect(() => {
    const fetchAndSetPopularItems = async () => {
      if (staticData) {
        try {
          // Assuming the endpoint returns an array of item IDs: number[]
          const response = await axios.get<number[]>(`${API_BASE_URL}/popular-items`);
          if (response.data && Array.isArray(response.data) && response.data.length > 0) {
            console.log(`Successfully fetched and set ${response.data.length} popular items for preloading.`);
          } else {
            console.warn("Popular items response was empty, malformed, or contained no items. Falling back to default preload list.");
          }
        } catch (err) {
          console.error("Error fetching popular items for preloading, falling back to default list:", err);
        }
      }
    };

    fetchAndSetPopularItems();
  }, [staticData]);

  // const preloadImage = useCallback((url: string, key: string) => {
  //   if (!url || loadedImages.has(key)) return;
  //   
  //   const img = new Image();
  //   img.onload = () => setLoadedImages(prev => new Set(prev).add(key));
  //   img.onerror = () => setLoadedImages(prev => new Set(prev).add(key)); // Still mark as loaded to prevent infinite loading state
  //   img.src = url;
  // }, [loadedImages]);

  // Preload item images for a match (unused but may be needed later)
  // const preloadMatchItems = useCallback((match: PlayerMatchStats) => {
  //   if (!staticData) return;
  //   
  //   match.items.forEach(itemId => {
  //     const itemImageKey = `item-${itemId}`;
  //     if (!isImageLoaded(itemImageKey)) {
  //       const itemUrl = `${dataDragonBase}/${getItemImageURL(itemId)}`;
  //       preloadImage(itemUrl, itemImageKey);
  //     }
  //   });
  // }, [staticData, isImageLoaded, preloadImage]); // getItemImageURL is a stable function defined in the component

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
        // Don't set error for static data loading failure - let the UI show anyway
        // The search form will show an appropriate message when user tries to search
        console.log("Backend unavailable - frontend will run in limited mode");
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
      // const allFetchedMatchesStats = calculateAggregatedStats(playerData.matches);
      // setLifetimeStats(allFetchedMatchesStats);
      // setCurrentSeasonStats(allFetchedMatchesStats); // Replace with actual current season logic if available
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
      setError("Backend is currently unavailable. This is a demo showing the frontend interface. The backend server needs to be deployed to fetch live League of Legends data.");
      return;
    }
    
    setIsLoading(true);
    setError(null);
    setPlayerData(null);
    setRecentGamesSummary(null);
    // setLifetimeStats(null);
    // setCurrentSeasonStats(null);

    try {
      // Use the new consolidated dashboard endpoint
      const dashboardResponse = await axios.get<PlayerDashboardData>(`${API_BASE_URL}/player/${region}/${gameName}/${tagLine}/dashboard?count=25`);
      
      // Extract the data from the consolidated response
      setPlayerData({
        puuid: dashboardResponse.data.summary.puuid,
        region: dashboardResponse.data.summary.region,
        riotId: dashboardResponse.data.summary.riotId,
        matches: dashboardResponse.data.matches,
        updatedAt: dashboardResponse.data.summary.lastUpdated
      });
      setRecentGamesSummary(dashboardResponse.data.summary);
      console.log("Dashboard data:", dashboardResponse.data);
    } catch (err: any) {
      console.error("Error fetching player data:", err);
      if (err.response && err.response.data) {
        setError(`Error: ${err.response.data.error || err.response.data}`);
      } else if (err.request) {
        setError("Backend server is not responding. This frontend is deployed but the backend API needs to be running to fetch League of Legends data.");
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

  const formatGameDuration = (durationInSeconds: number): string => {
    const minutes = Math.floor(durationInSeconds / 60);
    const seconds = durationInSeconds % 60;
    return `${minutes}m ${seconds < 10 ? '0' : ''}${seconds}s`;
  };

  const dataDragonBase = "https://ddragon.leagueoflegends.com/cdn";

  // Helper: Get display name for roles (including game modes)
  const getRoleDisplayName = (role: string) => {
    const displayNames: { [key: string]: string } = {
      TOP: 'Top Lane',
      JUNGLE: 'Jungle',
      MID: 'Mid Lane',
      MIDDLE: 'Mid Lane',
      BOT: 'Bot Lane',
      BOTTOM: 'Bot Lane',
      ADC: 'Bot Lane',
      SUPPORT: 'Support',
      UTILITY: 'Support',
      ARAM: 'ARAM',
      ARENA: 'Arena',
    };
    return displayNames[role?.toUpperCase()] || role;
  };

  // Helper: Map teamPosition/lane to role icon URL
  const getRoleIconURL = (role: string) => {
    const roleMap: { [key: string]: string } = {
      TOP: `${process.env.PUBLIC_URL}/roles/64px-Top_icon.webp`,
      JUNGLE: `${process.env.PUBLIC_URL}/roles/64px-Jungle_icon.webp`,
      MID: `${process.env.PUBLIC_URL}/roles/64px-Middle_icon.webp`,
      MIDDLE: `${process.env.PUBLIC_URL}/roles/64px-Middle_icon.webp`,
      BOT: `${process.env.PUBLIC_URL}/roles/64px-Bottom_icon.webp`,
      BOTTOM: `${process.env.PUBLIC_URL}/roles/64px-Bottom_icon.webp`,
      ADC: `${process.env.PUBLIC_URL}/roles/64px-Bottom_icon.webp`,
      SUPPORT: `${process.env.PUBLIC_URL}/roles/64px-Support_icon.webp`,
      UTILITY: `${process.env.PUBLIC_URL}/roles/64px-Support_icon.webp`,
      // Special game modes
      ARENA: `${process.env.PUBLIC_URL}/roles/64px-Jungle_icon.webp`,
      CHERRY: `${process.env.PUBLIC_URL}/roles/64px-Jungle_icon.webp`,
      ARAM: `${process.env.PUBLIC_URL}/roles/64px-Middle_icon.webp`,
      NONE: '',
    };
    return roleMap[role?.toUpperCase()] || '';
  };

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
    // Calculate approximate placement based on various metrics in Arena modes.
    // You may need to adjust this based on actual arena results if available in match data.
    return '?'; // Or compute based on match.kills, match.deaths, etc.
  };

  // Helper: Render items for a player
  const renderItemIcons = (items: number[]) => {
    return items.map((itemId, idx) => {
      if (itemId === 0) {
        return (
          <div
            key={idx}
            className="item-icon empty"
            title="No item"
          />
        );
      }
      
      return (
        <img
          key={idx}
          src={staticData ? `${dataDragonBase}/${getItemImageURL(itemId)}` : 'placeholder.png'}
          alt={`Item ${itemId}`}
          className="item-icon"
          onError={(e) => (e.currentTarget.src = 'placeholder.png')}
          title={staticData && staticData.items[itemId.toString()] ? staticData.items[itemId.toString()].name : `Item ${itemId}`}
        />
      );
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
        return (
          <div className="match-history">
            <div className="stats-header">
              <h2>Match History</h2>
              <span className="sub-text">Your recent League of Legends matches</span>
            </div>
            <div className="no-matches-message">
              <p>No recent matches found for this player.</p>
            </div>
          </div>
        );
      }
      return null;
    }
    
    const matchesToDisplay = playerData.matches.slice(0, visibleMatches);
    const grouped = groupMatchesByDate(matchesToDisplay);
    const dates = Object.keys(grouped);

    return (
      <div className="match-history">
        <div className="stats-header">
          <h2>Match History</h2>
          <span className="sub-text">Your recent League of Legends matches</span>
        </div>
        
        {dates.length > 0 ? dates.map(date => {
          const matchesOnDate = grouped[date];
          const wins = matchesOnDate.filter(m => m.win).length;
          const losses = matchesOnDate.length - wins;
          
          return (
            <div key={date} className="match-date-section">
              <div className="match-date-header">
                <h3 className="match-date-title">{date}</h3>
                <div className="match-date-summary">
                  <span className="wins">{wins}W</span>
                  <span className="losses">{losses}L</span>
                </div>
              </div>
              
              <div className="matches-grid">
                {matchesOnDate.map((match) => {
                  const champIcon = getChampionIcon(match.championName, match.championId);
                  const isArena = match.gameMode === 'CHERRY' || match.queueId === 1700;
                  
                  return (
                    <div
                      key={match.matchId}
                      className={`match-card ${match.win ? 'win' : 'loss'}`}
                      onClick={() => handleMatchCardClick(match)}
                    >
                      <div className="match-result-indicator">
                        <span className="result-text">{match.win ? 'WIN' : 'LOSS'}</span>
                      </div>
                      
                      <div className="match-champion-section">
                        <div className="champion-info">
                          <img 
                            src={champIcon} 
                            alt={match.championName} 
                            className="champion-portrait"
                            onError={e => (e.currentTarget.src = 'placeholder.png')} 
                          />
                          <div className="champion-details">
                            <span className="champion-name">{match.championName}</span>
                            <span className="champion-level">Level {match.champLevel}</span>
                          </div>
                        </div>
                        
                        <div className="summoner-spells">
                          {match.summonerSpells.map((spellId, index) => (
                            <img
                              key={`spell-${index}-${spellId}`}
                              src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(spellId)}` : ''}
                              alt={`Summoner Spell ${spellId}`}
                              className="summoner-spell"
                              onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                            />
                          ))}
                        </div>
                      </div>
                      
                      <div className="match-stats-section">
                        <div className="kda-stats">
                          <div className="kda-main">
                            <span className="kda-numbers">{match.kills}/{match.deaths}/{match.assists}</span>
                            <span className="kda-ratio">{match.kda.toFixed(2)} KDA</span>
                          </div>
                        </div>
                        
                        <div className="performance-stats">
                          {!isArena ? (
                            <div className="stat-item">
                              <span className="stat-value">{getCsPerMin(match)}</span>
                              <span className="stat-label">CS/min</span>
                            </div>
                          ) : (
                            <div className="stat-item">
                              <span className="stat-value">{getArenaPlace(match)}</span>
                              <span className="stat-label">Place</span>
                            </div>
                          )}
                          
                          <div className="stat-item">
                            <span className="stat-value">{formatGameDuration(match.gameDuration)}</span>
                            <span className="stat-label">Duration</span>
                          </div>
                        </div>
                      </div>
                      
                      <div className="match-items-section">
                        <div className="item-build">
                          {match.items.slice(0, 6).map((itemId, index) => (
                            itemId > 0 ? (
                              <img
                                key={index}
                                src={staticData ? `${dataDragonBase}/${getItemImageURL(itemId)}` : ''}
                                alt={`Item ${itemId}`}
                                className="item-icon"
                                onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                              />
                            ) : (
                              <div key={index} className="item-icon empty" />
                            )
                          ))}
                        </div>
                      </div>
                      
                      <div className="match-meta-section">
                        <div className="game-mode">{getDisplayGameMode(match.gameMode, match.queueId)}</div>
                        <div className="time-ago">{formatTimeAgo(match.gameCreation)}</div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          );
        }) : (
          <div className="no-matches-message">
            <p>All matches shown or no matches for current view.</p>
          </div>
        )}
        
        {playerData && playerData.matches.length > visibleMatches && (
          <div className="load-more-section">
            <button className="load-more-btn" onClick={() => setVisibleMatches(visibleMatches + 10)}>
              Load More Matches
            </button>
          </div>
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
          
          <div className="modal-header">
            <h2>Match Details</h2>
            <div className="modal-match-meta">
              <span className="game-mode-badge">{getDisplayGameMode(selectedMatch.gameMode, selectedMatch.queueId)}</span>
              <span className="game-duration-badge">{formatGameDuration(selectedMatch.gameDuration)}</span>
              <span className="time-ago-badge">{formatTimeAgo(selectedMatch.gameCreation)}</span>
            </div>
          </div>
          
          {isMatchLoading ? (
            <div className="modal-loading">
              <div className="loading-spinner"></div>
              <span>Loading match details...</span>
            </div>
          ) : isArena ? (
            fullMatchData && fullMatchData.info && fullMatchData.info.participants ? (
              <div className="arena-modal-content">
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
                    <div key={team.teamId} className="arena-team-section">
                      <div className="arena-team-header">
                        <h3 className="arena-team-place">#{team.placement}</h3>
                        <span className="arena-team-label">Team {team.teamId}</span>
                      </div>
                      
                      <div className="arena-players-grid">
                        {team.members.map((p: any, i: number) => (
                          <div key={i} className="arena-player-card">
                            <div className="player-header">
                              <div className="player-champion-info">
                                <img
                                  src={staticData ? `${dataDragonBase}/${getChampionImageURL(p.championName, p.championId)}` : ''}
                                  alt={p.championName}
                                  className="player-champion-portrait"
                                  onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                                />
                                <div className="player-details">
                                  <span className="player-name">{p.riotIdGameName || p.summonerName || 'Unknown Player'}</span>
                                  <span className="champion-name-small">{p.championName}</span>
                                  <span className="champion-level-small">Level {p.champLevel}</span>
                                </div>
                              </div>
                              
                              <div className="player-summoner-spells">
                                <img
                                  src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(p.summoner1Id)}` : ''}
                                  alt="Summoner Spell 1"
                                  className="summoner-spell-small"
                                  onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                                />
                                <img
                                  src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(p.summoner2Id)}` : ''}
                                  alt="Summoner Spell 2"
                                  className="summoner-spell-small"
                                  onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                                />
                              </div>
                            </div>
                            
                            <div className="player-stats">
                              <div className="player-kda">
                                <span className="kda-numbers-small">{p.kills}/{p.deaths}/{p.assists}</span>
                                <span className="kda-ratio-small">{p.deaths > 0 ? ((p.kills + p.assists) / p.deaths).toFixed(2) : (p.kills + p.assists)} KDA</span>
                              </div>
                              
                              <div className="player-items">
                                {renderItemIcons([p.item0, p.item1, p.item2, p.item3, p.item4, p.item5])}
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  ));
                })()}
              </div>
            ) : (
              <div className="modal-no-data">
                <p>Full match data not available or participants missing.</p>
              </div>
            )
          ) : (
            fullMatchData && fullMatchData.info && fullMatchData.info.participants ? (
              <div className="rift-modal-content">
                <div className="teams-container">
                  <div className="team-section team-blue">
                    <div className="team-header">
                      <h3>Blue Team</h3>
                      <span className="team-result">
                        {(() => {
                          const teamResult = fullMatchData.info.teams?.find((t: any) => t.teamId === 100)?.win;
                          if (teamResult !== undefined) {
                            return teamResult ? 'Victory' : 'Defeat';
                          }
                          return 'Team 100';
                        })()}
                      </span>
                    </div>
                    
                    <div className="team-players">
                      {fullMatchData.info.participants.filter((p: any) => p.teamId === 100).map((p: any, idx: number) => (
                        <div key={idx} className="rift-player-card">
                          <div className="player-header">
                            <div className="player-champion-info">
                              <img
                                src={staticData ? `${dataDragonBase}/${getChampionImageURL(p.championName, p.championId)}` : ''}
                                alt={p.championName}
                                className="player-champion-portrait"
                                onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                              />
                              <div className="player-details">
                                <span className="player-name">{p.summonerName || p.riotIdGameName || 'Unknown'}</span>
                                <span className="champion-name-small">{p.championName}</span>
                                <span className="champion-level-small">Level {p.champLevel}</span>
                              </div>
                            </div>
                            
                            <div className="player-summoner-spells">
                              <img
                                src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(p.summoner1Id)}` : ''}
                                alt="Summoner Spell 1"
                                className="summoner-spell-small"
                                onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                              />
                              <img
                                src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(p.summoner2Id)}` : ''}
                                alt="Summoner Spell 2"
                                className="summoner-spell-small"
                                onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                              />
                            </div>
                          </div>
                          
                          <div className="player-stats">
                            <div className="player-kda">
                              <span className="kda-numbers-small">{p.kills}/{p.deaths}/{p.assists}</span>
                              <span className="kda-ratio-small">{p.deaths > 0 ? ((p.kills + p.assists) / p.deaths).toFixed(2) : (p.kills + p.assists)} KDA</span>
                            </div>
                            
                            <div className="player-performance">
                              <div className="stat-item-small">
                                <span className="stat-value-small">{p.goldEarned?.toLocaleString() || 'N/A'}</span>
                                <span className="stat-label-small">Gold</span>
                              </div>
                              <div className="stat-item-small">
                                <span className="stat-value-small">
                                  {((p.totalMinionsKilled || 0) + (p.neutralMinionsKilled || 0)) > 0 && selectedMatch.gameDuration > 0 
                                    ? (((p.totalMinionsKilled || 0) + (p.neutralMinionsKilled || 0)) / (selectedMatch.gameDuration / 60)).toFixed(1)
                                    : '0.0'
                                  }
                                </span>
                                <span className="stat-label-small">CS/min</span>
                              </div>
                            </div>
                            
                            <div className="player-items">
                              {renderItemIcons([p.item0 || 0, p.item1 || 0, p.item2 || 0, p.item3 || 0, p.item4 || 0, p.item5 || 0])}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                  
                  <div className="team-section team-red">
                    <div className="team-header">
                      <h3>Red Team</h3>
                      <span className="team-result">
                        {(() => {
                          const teamResult = fullMatchData.info.teams?.find((t: any) => t.teamId === 200)?.win;
                          if (teamResult !== undefined) {
                            return teamResult ? 'Victory' : 'Defeat';
                          }
                          return 'Team 200';
                        })()}
                      </span>
                    </div>
                    
                    <div className="team-players">
                      {fullMatchData.info.participants.filter((p: any) => p.teamId === 200).map((p: any, idx: number) => (
                        <div key={idx} className="rift-player-card">
                          <div className="player-header">
                            <div className="player-champion-info">
                              <img
                                src={staticData ? `${dataDragonBase}/${getChampionImageURL(p.championName, p.championId)}` : ''}
                                alt={p.championName}
                                className="player-champion-portrait"
                                onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                              />
                              <div className="player-details">
                                <span className="player-name">{p.summonerName || p.riotIdGameName || 'Unknown'}</span>
                                <span className="champion-name-small">{p.championName}</span>
                                <span className="champion-level-small">Level {p.champLevel}</span>
                              </div>
                            </div>
                            
                            <div className="player-summoner-spells">
                              <img
                                src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(p.summoner1Id)}` : ''}
                                alt="Summoner Spell 1"
                                className="summoner-spell-small"
                                onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                              />
                              <img
                                src={staticData ? `${dataDragonBase}/${getSummonerSpellImageURL(p.summoner2Id)}` : ''}
                                alt="Summoner Spell 2"
                                className="summoner-spell-small"
                                onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                              />
                            </div>
                          </div>
                          
                          <div className="player-stats">
                            <div className="player-kda">
                              <span className="kda-numbers-small">{p.kills}/{p.deaths}/{p.assists}</span>
                              <span className="kda-ratio-small">{p.deaths > 0 ? ((p.kills + p.assists) / p.deaths).toFixed(2) : (p.kills + p.assists)} KDA</span>
                            </div>
                            
                            <div className="player-performance">
                              <div className="stat-item-small">
                                <span className="stat-value-small">{p.goldEarned?.toLocaleString() || 'N/A'}</span>
                                <span className="stat-label-small">Gold</span>
                              </div>
                              <div className="stat-item-small">
                                <span className="stat-value-small">
                                  {((p.totalMinionsKilled || 0) + (p.neutralMinionsKilled || 0)) > 0 && selectedMatch.gameDuration > 0 
                                    ? (((p.totalMinionsKilled || 0) + (p.neutralMinionsKilled || 0)) / (selectedMatch.gameDuration / 60)).toFixed(1)
                                    : '0.0'
                                  }
                                </span>
                                <span className="stat-label-small">CS/min</span>
                              </div>
                            </div>
                            
                            <div className="player-items">
                              {renderItemIcons([p.item0 || 0, p.item1 || 0, p.item2 || 0, p.item3 || 0, p.item4 || 0, p.item5 || 0])}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            ) : (
              <div className="modal-no-data">
                <p>Full match data not available.</p>
              </div>
            )
          )}
        </div>
      </div>
    );
  };

  // Enhanced Analytics Rendering Functions
  const renderOverviewStats = () => {
    if (!recentGamesSummary) return null;

    const { overallStats, totalMatches } = recentGamesSummary;

    return (
      <div className="overview-stats">
        <div className="stats-header">
          <h2>Recent Games Overview</h2>
          <span className="total-matches">{totalMatches} matches analyzed</span>
        </div>
        
        <div className="stats-grid">
          <div className="stat-card winrate">
            <div className="stat-value">{overallStats.winRate.toFixed(1)}%</div>
            <div className="stat-label">Win Rate</div>
            <div className="stat-sub">{overallStats.wins}W - {overallStats.losses}L</div>
          </div>

          <div className="stat-card kda">
            <div className="stat-value">{overallStats.overallKDA.toFixed(2)}</div>
            <div className="stat-label">Average KDA</div>
            <div className="stat-sub">{overallStats.avgKills.toFixed(1)} / {overallStats.avgDeaths.toFixed(1)} / {overallStats.avgAssists.toFixed(1)}</div>
          </div>

          <div className="stat-card cs">
            <div className="stat-value">{overallStats.avgCSPerMin.toFixed(1)}</div>
            <div className="stat-label">CS per Minute</div>
            <div className="stat-sub">Farm Performance</div>
          </div>

          <div className="stat-card vision">
            <div className="stat-value">{overallStats.avgVisionScore.toFixed(0)}</div>
            <div className="stat-label">Avg Vision Score</div>
            <div className="stat-sub">Ward Contribution</div>
          </div>

          <div className="stat-card damage">
            <div className="stat-value">{Math.round(overallStats.avgDamageToChampions / 1000)}k</div>
            <div className="stat-label">Avg Damage</div>
            <div className="stat-sub">Per Game</div>
          </div>

          <div className="stat-card gold">
            <div className="stat-value">{Math.round(overallStats.avgGoldPerMin)}</div>
            <div className="stat-label">Gold per Minute</div>
            <div className="stat-sub">Resource Generation</div>
          </div>

          <div className="stat-card participation">
            <div className="stat-value">{(overallStats.avgKillParticipation * 100).toFixed(1)}%</div>
            <div className="stat-label">Kill Participation</div>
            <div className="stat-sub">Team Fight Impact</div>
          </div>

          <div className="stat-card duration">
            <div className="stat-value">{Math.round(overallStats.avgGameDuration / 60)}m</div>
            <div className="stat-label">Avg Game Length</div>
            <div className="stat-sub">Match Duration</div>
          </div>
        </div>
      </div>
    );
  };

  const renderRoleStats = () => {
    if (!recentGamesSummary) return null;

    const { roleStats } = recentGamesSummary;
    const roles = Object.keys(roleStats).sort((a, b) => roleStats[b].gamesPlayed - roleStats[a].gamesPlayed);

    return (
      <div className="role-stats">
        <div className="stats-header">
          <h2>Performance by Role</h2>
          <span className="sub-text">Statistics grouped by position played</span>
        </div>

        <div className="role-cards">
          {roles.map(role => {
            const stats = roleStats[role];
            return (
              <div key={role} className="role-card">
                <div className="role-header">
                  <div className="role-icon">
                    {(() => {
                      const roleIconUrl = getRoleIconURL(role);
                      return roleIconUrl ? (
                        <img 
                          src={roleIconUrl} 
                          alt={role} 
                          className="role-image"
                          onError={(e) => {
                            e.currentTarget.style.display = 'none';
                            // Could add a fallback icon here if needed
                          }}
                        />
                      ) : null;
                    })()}
                  </div>
                  <div className="role-info">
                    <h3>{getRoleDisplayName(role)}</h3>
                    <span className="games-played">{stats.gamesPlayed} games</span>
                  </div>
                  <div className="role-winrate">
                    <span className="winrate-value">{stats.winRate.toFixed(1)}%</span>
                    <span className="winrate-record">{stats.wins}W - {stats.losses}L</span>
                  </div>
                </div>

                <div className="role-stats-grid">
                  <div className="role-stat">
                    <span className="role-stat-value">{stats.roleKDA.toFixed(2)}</span>
                    <span className="role-stat-label">KDA</span>
                  </div>
                  {role.toUpperCase() === 'ARAM' ? (
                    <>
                      <div className="role-stat">
                        <span className="role-stat-value">{stats.avgKills.toFixed(1)}</span>
                        <span className="role-stat-label">Avg Kills</span>
                      </div>
                      <div className="role-stat">
                        <span className="role-stat-value">{Math.round(stats.avgDamageToChampions / 1000)}k</span>
                        <span className="role-stat-label">Damage</span>
                      </div>
                      <div className="role-stat">
                        <span className="role-stat-value">{(stats.avgKillParticipation * 100).toFixed(0)}%</span>
                        <span className="role-stat-label">KP</span>
                      </div>
                    </>
                  ) : role.toUpperCase() === 'ARENA' ? (
                    <>
                      <div className="role-stat">
                        <span className="role-stat-value">{stats.avgKills.toFixed(1)}</span>
                        <span className="role-stat-label">Avg Kills</span>
                      </div>
                      <div className="role-stat">
                        <span className="role-stat-value">{Math.round(stats.avgDamageToChampions / 1000)}k</span>
                        <span className="role-stat-label">Damage</span>
                      </div>
                      <div className="role-stat">
                        <span className="role-stat-value">{stats.avgDeaths.toFixed(1)}</span>
                        <span className="role-stat-label">Avg Deaths</span>
                      </div>
                    </>
                  ) : (
                    <>
                      <div className="role-stat">
                        <span className="role-stat-value">{stats.avgCSPerMin.toFixed(1)}</span>
                        <span className="role-stat-label">CS/min</span>
                      </div>
                      <div className="role-stat">
                        <span className="role-stat-value">{Math.round(stats.avgDamageToChampions / 1000)}k</span>
                        <span className="role-stat-label">Damage</span>
                      </div>
                      <div className="role-stat">
                        <span className="role-stat-value">{(stats.avgKillParticipation * 100).toFixed(0)}%</span>
                        <span className="role-stat-label">KP</span>
                      </div>
                    </>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    );
  };

  // Sort handler for champion performance table
  const handleSort = (column: SortColumn) => {
    if (sortColumn === column) {
      // If clicking the same column, toggle direction
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      // If clicking a different column, set new column and default to descending
      setSortColumn(column);
      setSortDirection('desc');
    }
  };

  const renderChampionStats = () => {
    if (!recentGamesSummary) return null;

    const { championStats } = recentGamesSummary;
    const dataDragonBase = "https://ddragon.leagueoflegends.com/cdn";
    
    // Get all champions and apply sorting
    const champions = Object.keys(championStats).sort((a, b) => {
      const statsA = championStats[a];
      const statsB = championStats[b];
      
      let comparison = 0;
      
      switch (sortColumn) {
        case 'champion':
          comparison = a.localeCompare(b);
          break;
        case 'games':
          comparison = statsA.gamesPlayed - statsB.gamesPlayed;
          break;
        case 'winrate':
          comparison = statsA.winRate - statsB.winRate;
          break;
        case 'kda':
          comparison = statsA.championKDA - statsB.championKDA;
          break;
        case 'cs':
          comparison = statsA.avgCSPerMin - statsB.avgCSPerMin;
          break;
        case 'damage':
          comparison = statsA.avgDamageToChampions - statsB.avgDamageToChampions;
          break;
        case 'lastPlayed':
          comparison = statsA.lastPlayed - statsB.lastPlayed;
          break;
        default:
          comparison = statsB.gamesPlayed - statsA.gamesPlayed; // Default sort by games
      }
      
      return sortDirection === 'asc' ? comparison : -comparison;
    }).slice(0, 10); // Show top 10

    const getSortIcon = (column: SortColumn) => {
      if (sortColumn !== column) return '';
      return sortDirection === 'asc' ? ' ↑' : ' ↓';
    };

    return (
      <div className="champion-stats">
        <div className="stats-header">
          <h2>Champion Performance</h2>
          <span className="sub-text">Click column headers to sort • Showing top 10 champions</span>
        </div>

        <div className="champion-table">
          <div className="champion-table-header">
            <div 
              className={`champ-col sortable ${sortColumn === 'champion' ? 'active' : ''}`}
              onClick={() => handleSort('champion')}
            >
              Champion{getSortIcon('champion')}
            </div>
            <div 
              className={`games-col sortable ${sortColumn === 'games' ? 'active' : ''}`}
              onClick={() => handleSort('games')}
            >
              Games{getSortIcon('games')}
            </div>
            <div 
              className={`winrate-col sortable ${sortColumn === 'winrate' ? 'active' : ''}`}
              onClick={() => handleSort('winrate')}
            >
              Win Rate{getSortIcon('winrate')}
            </div>
            <div 
              className={`kda-col sortable ${sortColumn === 'kda' ? 'active' : ''}`}
              onClick={() => handleSort('kda')}
            >
              KDA{getSortIcon('kda')}
            </div>
            <div 
              className={`cs-col sortable ${sortColumn === 'cs' ? 'active' : ''}`}
              onClick={() => handleSort('cs')}
            >
              CS/min{getSortIcon('cs')}
            </div>
            <div 
              className={`damage-col sortable ${sortColumn === 'damage' ? 'active' : ''}`}
              onClick={() => handleSort('damage')}
            >
              Damage{getSortIcon('damage')}
            </div>
            <div 
              className={`last-played-col sortable ${sortColumn === 'lastPlayed' ? 'active' : ''}`}
              onClick={() => handleSort('lastPlayed')}
            >
              Last Played{getSortIcon('lastPlayed')}
            </div>
          </div>

          {champions.map(championName => {
            const stats = championStats[championName];
            return (
              <div key={championName} className="champion-row">
                <div className="champ-col">
                  <div className="champion-info">
                    <img
                      src={staticData ? `${dataDragonBase}/${getChampionImageURL(championName, stats.championId)}` : ''}
                      alt={championName}
                      className="champion-avatar"
                      onError={(e) => (e.currentTarget.src = 'placeholder.png')}
                    />
                    <span className="champion-name">{championName}</span>
                  </div>
                </div>
                <div className="games-col">{stats.gamesPlayed}</div>
                <div className="winrate-col">
                  <span className={`winrate ${stats.winRate >= 60 ? 'high' : stats.winRate >= 50 ? 'medium' : 'low'}`}>
                    {stats.winRate.toFixed(1)}%
                  </span>
                  <span className="record">({stats.wins}W {stats.losses}L)</span>
                </div>
                <div className="kda-col">
                  <span className="kda-value">{stats.championKDA.toFixed(2)}</span>
                  <div className="kda-breakdown">
                    {stats.avgKills.toFixed(1)} / {stats.avgDeaths.toFixed(1)} / {stats.avgAssists.toFixed(1)}
                  </div>
                </div>
                <div className="cs-col">{stats.avgCSPerMin.toFixed(1)}</div>
                <div className="damage-col">{Math.round(stats.avgDamageToChampions / 1000)}k</div>
                <div className="last-played-col">{formatTimeAgo(stats.lastPlayed)}</div>
              </div>
            );
          })}
        </div>
      </div>
    );
  };

  const renderEnhancedDashboard = () => {
    if (!recentGamesSummary) return null;

    return (
      <div className="enhanced-dashboard">
        <div className="dashboard-tabs">
          <button
            className={`tab ${activeTab === 'overview' ? 'active' : ''}`}
            onClick={() => setActiveTab('overview')}
          >
            Overview
          </button>
          <button
            className={`tab ${activeTab === 'roles' ? 'active' : ''}`}
            onClick={() => setActiveTab('roles')}
          >
            By Role
          </button>
          <button
            className={`tab ${activeTab === 'champions' ? 'active' : ''}`}
            onClick={() => setActiveTab('champions')}
          >
            Champions
          </button>
          <button
            className={`tab ${activeTab === 'matches' ? 'active' : ''}`}
            onClick={() => setActiveTab('matches')}
          >
            Match History
          </button>
        </div>

        <div className="dashboard-content">
          {activeTab === 'overview' && renderOverviewStats()}
          {activeTab === 'roles' && renderRoleStats()}
          {activeTab === 'champions' && renderChampionStats()}
          {activeTab === 'matches' && renderRecentMatches()}
        </div>
      </div>
    );
  };

  return (
    <div className="App">
      <main className="App-main-r6">
        <section className="search-section">
          <div className="search-card">
            <div className="search-header">
              <h2>Player Lookup</h2>
              <span className="sub-text">Search for any League of Legends player (e.x. Derp518 #2877)</span>
            </div>
            
            <form onSubmit={handleSubmit} className="search-form">
              <div className="search-inputs">
                <div className="player-id-group">
                  <label className="input-label">Riot ID</label>
                  <div className="riot-id-input">
                    <input
                      type="text"
                      value={gameName}
                      onChange={(e) => setGameName(e.target.value)}
                      placeholder="Game Name"
                      className="game-name-input"
                      required
                      disabled={isStaticDataLoading}
                    />
                    <span className="tagline-separator">#</span>
                    <input
                      type="text"
                      value={tagLine}
                      onChange={(e) => setTagLine(e.target.value)}
                      placeholder="Tag"
                      className="tagline-input"
                      required
                      disabled={isStaticDataLoading}
                    />
                  </div>
                </div>
                
                <div className="region-group">
                  <label className="input-label">Region</label>
                  <select 
                    value={region} 
                    onChange={(e) => setRegion(e.target.value)} 
                    className="region-select" 
                    disabled={isStaticDataLoading}
                  >
                    <option value="na1">NA</option>
                    <option value="euw1">EUW</option>
                    <option value="eun1">EUNE</option>
                    <option value="kr">KR</option>
                    {/* Add more regions */}
                  </select>
                </div>
              </div>
              
              <button type="submit" disabled={isLoading || isStaticDataLoading} className="search-button">
                {isStaticDataLoading ? 'Loading Data...' : isLoading ? 'Searching...' : 'Search Player'}
              </button>
            </form>
            
            {error && <div className="error-message">{error}</div>}
          </div>
        </section>

        {isLoading && <div className="loading-message-r6">Fetching player data...</div>}
        
        {/* Enhanced Dashboard with comprehensive analytics */}
        {recentGamesSummary && !isLoading && renderEnhancedDashboard()}

        {!recentGamesSummary && !isLoading && !isStaticDataLoading && !error && (
             <p className="initial-prompt-r6">Enter a Riot ID (Game Name#Tagline) and region to see comprehensive match analytics.</p>
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
