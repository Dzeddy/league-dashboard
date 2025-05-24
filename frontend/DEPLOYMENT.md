# Frontend Deployment Guide

This guide explains how to deploy the League Dashboard frontend to various platforms and resolves common SPA routing issues.

## Important: SPA Routing Configuration

This React application uses client-side routing. Different deployment targets require different asset path configurations to avoid the "Uncaught SyntaxError: Unexpected token '<'" error that causes blank pages.

### The Problem
When accessing deep links (like `http://localhost:3000/league_dashboard`), the browser tries to load JavaScript and CSS files. If the asset paths are incorrect, the server returns `index.html` instead of the actual JS/CSS files, causing the browser to try parsing HTML as JavaScript.

### The Solution
We use different `homepage` configurations for different deployment targets:
- **Local Development**: `"homepage": "."` (relative paths)
- **GitHub Pages**: `"homepage": "https://dzeddy.github.io/league-dashboard"` (absolute paths)

## GitHub Pages Deployment (Recommended)

The project is configured for automatic deployment to GitHub Pages via GitHub Actions.

### Automatic Deployment

1. **Push to main branch**: The deployment happens automatically when you push to the `main` branch
2. **GitHub Actions**: The workflow will build and deploy the frontend to GitHub Pages
3. **Access**: Your app will be available at `https://dzeddy.github.io/league_dashboard`

### Manual Deployment to GitHub Pages

If you want to deploy manually:

```bash
cd frontend
npm install
npm run deploy
```

This will:
1. Set the homepage to the GitHub Pages URL
2. Build the app with the correct asset paths
3. Push the build to the `gh-pages` branch

## Local Development and Testing

### Option 1: Using `serve` (Recommended for Local Testing)

For local testing of the production build:

```bash
cd frontend
npm install
npm run build-and-serve
```

This will:
1. Set the homepage to relative paths (`.`)
2. Build the app
3. Start a production server on `http://localhost:3000`

You can now access `http://localhost:3000/league_dashboard` without routing errors.

### Option 2: Manual Build and Serve

```bash
cd frontend
npm install
npm run build:local
npm run serve
```

### Option 3: Development Server

For development with hot reload:

```bash
cd frontend
npm start
```

This runs on `http://localhost:3000` and doesn't require any special routing configuration.

## Alternative Deployment Options

### Option 1: Using the Deploy Script

For platforms that support Node.js applications:

```bash
cd frontend
npm install
npm run build:local
npm run deploy-serve
```

Or run directly:
```bash
cd frontend
npm install
npm run build:local
node deploy-serve.js
```

This script:
- Serves the built React app using `serve`
- Handles SPA routing correctly
- Supports environment variables for PORT configuration
- Includes proper error handling and graceful shutdown

### Option 2: Docker Deployment

You can also containerize the frontend. The included `nginx.conf` already handles SPA routing correctly:

```bash
cd frontend
npm run build:local  # or build:github depending on your deployment target
docker build -t league-dashboard-frontend .
docker run -p 80:80 league-dashboard-frontend
```

The Nginx configuration includes:
```nginx
location / {
    try_files $uri $uri/ /index.html;
}
```

This ensures that any unmatched routes serve `index.html`, allowing React Router to handle the routing.

## Environment Variables

The frontend supports the following environment variables:

- `REACT_APP_API_BASE_URL`: Backend API URL (default: `http://localhost:8080/api`)
- `PUBLIC_URL`: Base URL for the app (automatically set for GitHub Pages)
- `PORT`: Port for serving (when using serve, default: 3000)

## Troubleshooting

### GitHub Pages Issues

1. **404 on direct URLs**: The `404.html` file should handle SPA routing automatically
2. **Assets not loading**: Ensure the `homepage` field in `package.json` matches your GitHub Pages URL
3. **API calls failing**: Check CORS settings and ensure your backend allows requests from the GitHub Pages domain

### Build Issues

1. **Out of memory**: Increase Node.js memory limit:
   ```bash
   export NODE_OPTIONS="--max-old-space-size=4096"
   npm run build
   ```

2. **TypeScript errors**: Fix any TypeScript errors before building
3. **Missing dependencies**: Run `npm install` to ensure all dependencies are installed

### Serve Issues

1. **Port already in use**: Change the port:
   ```bash
   PORT=3001 npm run serve
   ```

2. **Routing not working**: Ensure you're using the `-s` flag with serve for SPA support

### SPA Routing Issues

#### "Uncaught SyntaxError: Unexpected token '<'" Error

**Symptoms**: Blank page when accessing deep links directly, console shows JavaScript parsing error

**Cause**: Browser receives HTML instead of JavaScript files due to incorrect asset paths

**Solutions**:

1. **For Local Development**:
   ```bash
   npm run build:local  # Sets homepage to relative paths
   npm run serve
   ```

2. **For GitHub Pages**:
   ```bash
   npm run build:github  # Sets homepage to GitHub Pages URL
   npm run deploy
   ```

3. **Check Current Homepage Setting**:
   ```bash
   npm pkg get homepage
   ```

4. **Manual Fix**:
   - For local: `npm pkg set homepage='.'`
   - For GitHub: `npm pkg set homepage='https://dzeddy.github.io/league-dashboard'`
   - Then rebuild: `npm run build`

#### Direct URL Access Returns 404

**For GitHub Pages**: Ensure `public/404.html` exists and contains SPA routing script

**For Nginx**: Ensure configuration includes `try_files $uri $uri/ /index.html;`

**For serve**: Always use the `-s` flag: `serve -s build`

## Available Scripts

- `npm start`: Development server with hot reload
- `npm run build`: Create production build with current homepage setting
- `npm run build:local`: Build for local development with relative paths
- `npm run build:github`: Build for GitHub Pages with absolute paths
- `npm run serve`: Serve production build locally
- `npm run build-and-serve`: Build for local use and serve in one command
- `npm run deploy`: Build for GitHub Pages and deploy
- `npm run deploy-serve`: Start production server using the deploy script
- `npm run set-homepage:local`: Set homepage to relative paths for local development
- `npm run set-homepage:github`: Set homepage to GitHub Pages URL
- `npm test`: Run tests

## Configuration Files

- `package.json`: Contains deployment scripts and homepage URL
- `public/404.html`: Handles SPA routing for GitHub Pages
- `public/index.html`: Contains SPA routing script for GitHub Pages
- `deploy-serve.js`: Custom deployment script using serve
- `.github/workflows/deploy.yml`: GitHub Actions deployment workflow 