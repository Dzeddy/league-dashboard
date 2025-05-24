# League Dashboard Frontend

A React-based dashboard for League of Legends player statistics and match analysis.

## 🚀 Quick Start

### Development
```bash
npm install
npm start
```

### Production Build
```bash
npm install
npm run build
```

### Serve Production Build Locally
```bash
npm run build
npm run serve
```

## 📦 Deployment

This project supports multiple deployment options:

### 1. GitHub Pages (Automatic - Recommended)
- **Automatic**: Push to `main` branch triggers deployment
- **Manual**: Run `npm run deploy`
- **URL**: https://dzeddy.github.io/league_dashboard

### 2. Using Serve (Node.js platforms)
```bash
npm run build
npm run deploy-serve
```

### 3. Docker Deployment
```bash
docker build -t league-dashboard-frontend .
docker run -p 80:80 league-dashboard-frontend
```

For detailed deployment instructions, see [DEPLOYMENT.md](./DEPLOYMENT.md).

## 🛠️ Available Scripts

- `npm start` - Start development server
- `npm run build` - Create production build
- `npm run test` - Run tests
- `npm run serve` - Serve production build locally
- `npm run build-and-serve` - Build and serve in one command
- `npm run deploy` - Deploy to GitHub Pages
- `npm run deploy-serve` - Start production server using serve

## 🔧 Configuration

### Environment Variables
- `REACT_APP_API_BASE_URL` - Backend API URL (default: http://localhost:8080/api)
- `PORT` - Port for serve command (default: 3000)

### GitHub Pages Setup
The project is pre-configured for GitHub Pages with:
- Correct `homepage` URL in `package.json`
- SPA routing support via `404.html` and routing scripts
- Automatic deployment via GitHub Actions

## 🏗️ Architecture

This is a React TypeScript application that:
- Fetches League of Legends data from a backend API
- Displays player statistics, match history, and performance analytics
- Supports responsive design for desktop and mobile
- Uses Axios for API calls and Day.js for date handling

## 📁 Project Structure

```
frontend/
├── public/           # Static assets and SPA routing files
├── src/             # React source code
├── build/           # Production build output
├── deploy-serve.js  # Custom deployment script
├── Dockerfile       # Container configuration
├── nginx.conf       # Nginx configuration for container
├── package.json     # Dependencies and scripts
└── DEPLOYMENT.md    # Detailed deployment guide
```

## 🔍 Features

- **Player Search**: Search for players by game name and tag
- **Match History**: View recent matches with detailed statistics
- **Performance Analytics**: Win rates, KDA, role performance
- **Champion Statistics**: Performance breakdown by champion
- **Responsive Design**: Works on desktop and mobile devices
- **SPA Routing**: Proper URL handling for GitHub Pages

## 🐛 Troubleshooting

### Common Issues

1. **Backend Not Available**: The frontend shows a demo message when the backend is unavailable
2. **Routing Issues**: Ensure SPA routing files are properly deployed
3. **API CORS**: Backend must allow requests from the frontend domain

### Build Issues
- Increase Node.js memory: `export NODE_OPTIONS="--max-old-space-size=4096"`
- Clear node_modules: `rm -rf node_modules && npm install`

## 📖 API Integration

The frontend expects a backend API with endpoints:
- `GET /api/static-data` - Game static data (champions, items, etc.)
- `GET /api/player/{region}/{gameName}/{tagLine}/matches` - Player match data
- `GET /api/player/{region}/{gameName}/{tagLine}/summary` - Player summary stats

## 🎨 Technologies Used

- **React 19** - UI framework
- **TypeScript** - Type safety
- **Axios** - HTTP client
- **Day.js** - Date manipulation
- **CSS3** - Styling and animations
- **React Testing Library** - Testing utilities

## 📄 License

This project is private and not licensed for public use. 