# League Dashboard

A comprehensive League of Legends performance tracking application that provides detailed player statistics, match analysis, and performance insights. The application consists of a React TypeScript frontend and a Go backend that integrates with the Riot Games API.

## Architecture Overview

### Backend (Go)
- **Framework**: Go with Gorilla Mux for routing
- **Database**: MongoDB for persistent data storage
- **Cache**: Redis for API response caching and performance optimization
- **External APIs**: Riot Games API for player and match data, Data Dragon for static game data
- **Security**: SSL/TLS support with automatic certificate generation
- **Deployment**: Docker support with configurable SSL and environment detection

### Frontend (React TypeScript)
- **Framework**: React 19 with TypeScript
- **HTTP Client**: Axios for API communication
- **Styling**: CSS3 with responsive design
- **Date Handling**: Day.js for date manipulation
- **Testing**: React Testing Library and Jest
- **Deployment**: GitHub Pages, Docker, or static hosting platforms

## Features

### Player Analytics
- **Player Search**: Search by Riot ID (GameName#TagLine) across all regions
- **Match History**: Detailed match statistics with filtering by queue type
- **Performance Metrics**: KDA, win rates, kill participation, CS/min, gold/min
- **Role Analysis**: Performance breakdown by position (Top, Jungle, Mid, ADC, Support)
- **Champion Statistics**: Per-champion performance with win rates and averages

### Data Visualization
- **Recent Games Summary**: Aggregated statistics across recent matches
- **Champion Performance**: Detailed breakdown by champion played
- **Role Performance**: Statistics segmented by team position
- **Item Analysis**: Popular items tracking and usage statistics
- **Trend Analysis**: Performance trends over time

### Technical Features
- **Caching Strategy**: Multi-layer caching with Redis for optimal performance
- **Rate Limiting**: Riot API rate limit compliance
- **Error Handling**: Comprehensive error handling and user feedback
- **Responsive Design**: Mobile and desktop optimized interface
- **Real-time Data**: Live data fetching with cache invalidation

## Quick Start

### Prerequisites
- **Backend**: Go 1.24+, MongoDB, Redis, Riot API Key
- **Frontend**: Node.js 16+, npm

### Backend Setup

1. **Clone and navigate to backend directory**
```bash
cd backend
```

2. **Install dependencies**
```bash
go mod download
```

3. **Environment Configuration**
Create a `.env` file with the following variables:
```env
RIOT_API_KEY=your_riot_api_key_here
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=leagueperformancetracker
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
PORT=8080
USE_SSL=false
```

4. **Start required services**
```bash
# MongoDB (using Docker)
docker run -d -p 27017:27017 --name mongodb mongo:latest

# Redis (using Docker)
docker run -d -p 6379:6379 --name redis redis:latest
```

5. **Run the backend**
```bash
go run .
```

### Frontend Setup

1. **Navigate to frontend directory**
```bash
cd frontend
```

2. **Install dependencies**
```bash
npm install
```

3. **Configure environment**
Create a `.env` file:
```env
REACT_APP_API_BASE_URL=http://localhost:8080/api
```

4. **Start development server**
```bash
npm start
```

The application will be available at `http://localhost:3000`

## API Documentation

### Endpoints

#### Player Performance
```
GET /api/player/{region}/{gameName}/{tagLine}/matches
```
- **Parameters**: 
  - `count` (optional): Number of matches (1-100, default: 25)
  - `queueId` (optional): Queue type filter (default: all queues)
- **Response**: Detailed match history with player statistics

#### Player Summary
```
GET /api/player/{region}/{gameName}/{tagLine}/summary
```
- **Response**: Aggregated player statistics and performance summary

#### Static Game Data
```
GET /api/static-data
```
- **Response**: Champions, items, runes, and summoner spells data

#### Match Details
```
GET /api/match/{region}/{matchId}
```
- **Response**: Detailed match information for specific match ID

#### Popular Items
```
GET /api/popular-items
```
- **Response**: Most frequently used items across all tracked matches

#### Health Check
```
GET /api/health
```
- **Response**: Service health status

### Supported Regions
- **Americas**: na1, br1, la1, la2
- **Asia**: kr, jp1
- **Europe**: euw1, eun1, tr1, ru
- **Oceania**: oc1
- **Southeast Asia**: ph2, sg2, th2, tw2, vn2

## Configuration

### Backend Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `RIOT_API_KEY` | Riot Games API key | - | Yes |
| `MONGO_URI` | MongoDB connection string | `mongodb://localhost:27017` | No |
| `MONGO_DATABASE` | MongoDB database name | `leagueperformancetracker` | No |
| `REDIS_ADDR` | Redis server address | `localhost:6379` | No |
| `REDIS_PASSWORD` | Redis password | - | No |
| `REDIS_DB` | Redis database number | `0` | No |
| `PORT` | Server port | `8080` (HTTP) / `8443` (HTTPS) | No |
| `USE_SSL` | Enable SSL/TLS | `true` | No |
| `SSL_CERT_FILE` | SSL certificate file path | `server.crt` | No |
| `SSL_KEY_FILE` | SSL private key file path | `server.key` | No |

### Frontend Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `REACT_APP_API_BASE_URL` | Backend API URL | `http://localhost:8080/api` |
| `PORT` | Development server port | `3000` |

## Deployment

### Backend Deployment

#### Docker Deployment
```bash
# Build image
docker build -t league-dashboard-backend .

# Run container
docker run -p 8080:8080 \
  -e RIOT_API_KEY=your_key \
  -e MONGO_URI=mongodb://mongo:27017 \
  -e REDIS_ADDR=redis:6379 \
  league-dashboard-backend
```

#### Production Deployment
1. Set `USE_SSL=true` for HTTPS
2. Configure proper SSL certificates or use auto-generated ones
3. Set up MongoDB and Redis instances
4. Configure environment variables for production
5. Use a process manager like systemd or supervisor

### Frontend Deployment

#### GitHub Pages
```bash
npm run deploy
```

#### Docker Deployment
```bash
# Build image
docker build -t league-dashboard-frontend .

# Run container
docker run -p 80:80 league-dashboard-frontend
```

#### Static Hosting
```bash
# Build for production
npm run build

# Deploy build folder to any static hosting service
```

## Data Models

### Core Data Structures

#### PlayerMatchStats
Represents a player's performance in a single match:
- Match metadata (ID, duration, game mode)
- Champion information
- Performance metrics (KDA, CS, gold, damage)
- Items and runes used
- Team position and result

#### RecentGamesSummary
Aggregated statistics across multiple matches:
- Overall performance metrics
- Role-specific statistics
- Champion-specific performance
- Recent match history

#### StaticGameData
Game reference data from Riot's Data Dragon:
- Champion information and abilities
- Item details and statistics
- Rune and mastery data
- Summoner spell information

## Caching Strategy

### Redis Cache Layers
- **PUUID Cache**: 24 hours (player account data)
- **Match List Cache**: 1 hour (recent match IDs)
- **Match Details Cache**: 7 days (individual match data)
- **Static Data Cache**: 24 hours (champions, items, runes)
- **User Performance Cache**: 30 minutes (aggregated player stats)

### Cache Keys Format
```
puuid:{region}:{gamename}:{tagline}
matchids:{region}:{puuid}:{count}:q{queueid}:{starttime}
matchdetails:{region}:{matchid}
static_data:{datatype}:{version}
user_performance:{region}:{puuid}
```

## Development

### Backend Development
```bash
# Run with hot reload (using air)
go install github.com/cosmtrek/air@latest
air

# Run tests
go test ./...

# Build binary
go build -o league_backend
```

### Frontend Development
```bash
# Start development server
npm start

# Run tests
npm test

# Build for production
npm run build

# Serve production build locally
npm run serve
```

### Code Structure

#### Backend (`/backend`)
```
backend/
├── main.go           # Server setup and configuration
├── handlers.go       # HTTP request handlers
├── riotapi.go        # Riot API integration
├── models.go         # Data structures and types
├── go.mod           # Go module dependencies
└── Dockerfile       # Container configuration
```

#### Frontend (`/frontend`)
```
frontend/
├── src/
│   ├── App.tsx      # Main application component
│   ├── types.ts     # TypeScript type definitions
│   └── index.tsx    # Application entry point
├── public/          # Static assets
├── package.json     # Node.js dependencies
└── Dockerfile       # Container configuration
```

## Performance Considerations

### Backend Optimizations
- **Concurrent API Calls**: Parallel processing of multiple match requests
- **Database Indexing**: Optimized MongoDB indexes for common queries
- **Connection Pooling**: Efficient database and Redis connection management
- **Rate Limiting**: Riot API rate limit compliance and queuing

### Frontend Optimizations
- **Code Splitting**: Lazy loading of components
- **Memoization**: React.memo for expensive components
- **Efficient Rendering**: Optimized re-render cycles
- **Asset Optimization**: Compressed images and minified code

## Troubleshooting

### Common Issues

#### Backend Issues
1. **Riot API Rate Limits**: Implement exponential backoff and request queuing
2. **Database Connection**: Verify MongoDB and Redis connectivity
3. **SSL Certificate**: Check certificate validity and file permissions
4. **Memory Usage**: Monitor Go garbage collection and optimize data structures

#### Frontend Issues
1. **CORS Errors**: Ensure backend CORS configuration includes frontend domain
2. **API Timeouts**: Implement proper error handling and retry logic
3. **Build Failures**: Clear node_modules and reinstall dependencies
4. **Routing Issues**: Verify SPA routing configuration for deployment

### Monitoring and Logging
- **Backend Logs**: Structured logging with different levels (INFO, WARN, ERROR)
- **Performance Metrics**: Monitor API response times and cache hit rates
- **Error Tracking**: Comprehensive error logging and alerting
- **Health Checks**: Regular service health monitoring

## Contributing

### Development Workflow
1. Fork the repository
2. Create a feature branch
3. Make changes with appropriate tests
4. Ensure all tests pass
5. Submit a pull request

### Code Standards
- **Go**: Follow Go formatting standards (gofmt)
- **TypeScript**: Use ESLint and Prettier for code formatting
- **Testing**: Maintain test coverage above 80%
- **Documentation**: Update documentation for API changes

## License

This project is private and not licensed for public use.

## Support

For issues and questions:
1. Check the troubleshooting section
2. Review existing GitHub issues
3. Create a new issue with detailed information
4. Include logs and error messages when reporting bugs 