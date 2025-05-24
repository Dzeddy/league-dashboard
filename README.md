# League Dashboard

A comprehensive League of Legends performance tracking dashboard that provides detailed match statistics, player analytics, and performance insights.

## Features

- **Player Performance Tracking**: Track detailed statistics for any League of Legends player
- **Match History**: View comprehensive match history with detailed statistics
- **Performance Analytics**: Calculate win rates, KDA, and other key performance metrics
- **Visual Champions & Items**: Rich visual interface showing champion portraits and item builds
- **Real-time Data**: Fetches live data from the Riot Games API
- **Multiple Regions**: Support for all League of Legends regions
- **Responsive Design**: Modern, mobile-friendly user interface

## Architecture

This is a full-stack application consisting of:

- **Frontend**: React TypeScript application with modern UI/UX
- **Backend**: Go-based REST API server
- **Database**: MongoDB for data persistence
- **Cache**: Redis for performance optimization
- **External API**: Riot Games API integration

## Quick Start

### Prerequisites

Before running this application, make sure you have the following installed:

- [Node.js](https://nodejs.org/) (v16 or higher)
- [Go](https://golang.org/) (v1.21 or higher)
- [MongoDB](https://www.mongodb.com/)
- [Redis](https://redis.io/)
- [Riot API Key](https://developer.riotgames.com/) (required for data fetching)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/league_dashboard.git
   cd league_dashboard
   ```

2. **Set up environment variables**
   ```bash
   cp .env.example .env
   ```
   Edit `.env` and add your configuration:
   ```env
   RIOT_API_KEY=your_riot_api_key_here
   MONGO_URI=mongodb://localhost:27017
   MONGO_DATABASE=leagueperformancetracker
   REDIS_ADDR=localhost:6379
   ```

3. **Install dependencies**
   ```bash
   # Install root dependencies
   npm install

   # Install frontend dependencies
   cd frontend
   npm install
   cd ..

   # Install backend dependencies
   cd backend
   go mod download
   cd ..
   ```

4. **Start the services**

   **Option A: Start all services individually**
   ```bash
   # Terminal 1: Start MongoDB (if not running as service)
   mongod

   # Terminal 2: Start Redis (if not running as service)
   redis-server

   # Terminal 3: Start the backend
   cd backend
   go run .

   # Terminal 4: Start the frontend
   cd frontend
   npm start
   ```

   **Option B: Use the provided scripts**
   ```bash
   # Start backend and frontend concurrently
   npm run dev
   ```

5. **Access the application**
   - Frontend: http://localhost:3000
   - Backend API: https://localhost:8443 (HTTPS with auto-generated SSL certificates)
   
   **Note**: The backend now uses HTTPS by default. Your browser may show a security warning for the self-signed certificate - this is normal for development. Click "Advanced" and "Proceed to localhost" to continue.

## Usage

1. **Search for a Player**: Enter a player's Game Name and Tag Line (e.g., "Faker", "T1")
2. **Select Region**: Choose the appropriate region (e.g., kr, na1, euw1)
3. **View Performance**: Explore detailed match history, statistics, and performance metrics
4. **Analyze Matches**: Click on individual matches for in-depth analysis

## Development

### Project Structure

```
league_dashboard/
├── frontend/                 # React TypeScript frontend
│   ├── public/              # Static assets
│   ├── src/                 # Source code
│   │   ├── App.tsx         # Main application component
│   │   ├── types.ts        # TypeScript type definitions
│   │   └── ...
│   └── package.json        # Frontend dependencies
├── backend/                 # Go backend server
│   ├── main.go             # Application entry point
│   ├── handlers.go         # HTTP request handlers
│   ├── riotapi.go          # Riot API integration
│   ├── models.go           # Data models
│   └── go.mod              # Backend dependencies
├── database/               # Database related files
├── .env                    # Environment variables (create from .env.example)
├── .gitignore             # Git ignore rules
└── README.md              # This file
```

### Available Scripts

In the root directory:
- `npm run dev` - Start both frontend and backend in development mode
- `npm run build` - Build the frontend for production

In the frontend directory:
- `npm start` - Start the React development server
- `npm run build` - Build the React app for production
- `npm test` - Run the test suite
- `npm run eject` - Eject from Create React App (one-way operation)

In the backend directory:
- `go run .` - Start the Go server in development mode (HTTPS by default)
- `./run-ssl.sh` - Start the server with SSL enabled (recommended)
- `go build` - Build the Go application
- `go test ./...` - Run all tests

### API Endpoints

- `GET /api/health` - Health check endpoint
- `GET /api/player/{region}/{gameName}/{tagLine}/matches` - Get player performance data
- `GET /api/static-data` - Get League of Legends static game data
- `GET /api/match/{region}/{matchId}` - Get detailed match information
- `GET /api/popular-items` - Get popular items for preloading

## Database Schema

The application uses MongoDB with the following main collections:
- **players** - Player information and statistics
- **matches** - Match data and results
- **static_data** - Cached League of Legends static data (champions, items, etc.)

## Configuration

### Environment Variables

#### Backend Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `RIOT_API_KEY` | Your Riot Games API key | *Required* |
| `MONGO_URI` | MongoDB connection string | `mongodb://localhost:27017` |
| `MONGO_DATABASE` | MongoDB database name | `leagueperformancetracker` |
| `REDIS_ADDR` | Redis server address | `localhost:6379` |
| `PORT` | Server port | `8443` (HTTPS) / `8080` (HTTP) |
| `USE_SSL` | Enable HTTPS/SSL | `true` |
| `SSL_CERT_FILE` | SSL certificate file path | `server.crt` (auto-generated) |
| `SSL_KEY_FILE` | SSL private key file path | `server.key` (auto-generated) |

**SSL Configuration:**
The backend now supports HTTPS by default. SSL certificates are automatically generated for development. See `backend/SSL_SETUP.md` for detailed SSL configuration instructions.

#### Frontend Configuration

The frontend requires environment variables prefixed with `REACT_APP_` to be accessible in the browser.

| Variable | Description | Default |
|----------|-------------|---------|
| `REACT_APP_API_BASE_URL` | Backend API base URL | `https://localhost:8443/api` |

**Setting up frontend environment variables:**

1. Create a `.env.local` file in the `frontend/` directory:
   ```bash
   cd frontend
   echo "REACT_APP_API_BASE_URL=https://localhost:8443/api" > .env.local
   ```

2. For production deployment, set the environment variable to your production API URL:
   ```bash
   REACT_APP_API_BASE_URL=https://your-api-domain.com/api
   ```

**Important Notes:**
- Frontend environment variables are embedded at build time, not runtime
- You must rebuild the React application after changing environment variables
- See `frontend/README.env.md` for detailed configuration instructions

### Supported Regions

- `na1` - North America
- `euw1` - Europe West
- `eun1` - Europe Nordic & East
- `kr` - Korea
- `jp1` - Japan
- `br1` - Brazil
- `la1` - Latin America North
- `la2` - Latin America South
- `oc1` - Oceania
- `ru` - Russia
- `tr1` - Turkey

## Deployment

### Production Build

1. **Build the frontend**
   ```bash
   cd frontend
   npm run build
   ```

2. **Build the backend**
   ```bash
   cd backend
   go build -o league-dashboard-server
   ```

3. **Deploy with environment variables**
   Ensure all production environment variables are properly set.

### Docker Support

*Docker configuration coming soon...*

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go conventions for backend code
- Use TypeScript for all frontend code
- Write tests for new features
- Update documentation as needed
- Follow the existing code style

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Riot Games](https://developer.riotgames.com/) for providing the League of Legends API
- [Data Dragon](https://developer.riotgames.com/docs/lol#data-dragon) for static assets
- React and Go communities for excellent documentation and tools

## Support

If you encounter any issues or have questions:

1. Check the [Issues](https://github.com/yourusername/league_dashboard/issues) page
2. Create a new issue with detailed information
3. Join our community discussions

## Changelog

### Version 1.0.0
- Initial release
- Player performance tracking
- Match history visualization
- Riot API integration
- MongoDB and Redis support

---

**Note**: This application is not affiliated with Riot Games. League of Legends is a trademark of Riot Games, Inc. 