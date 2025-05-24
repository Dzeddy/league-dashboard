# Frontend Deployment Guide

This guide explains how to deploy the League Dashboard frontend to various platforms.

## GitHub Pages Deployment (Recommended)

The project is already configured for automatic deployment to GitHub Pages via GitHub Actions.

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

This will build the app and push it to the `gh-pages` branch.

## Alternative Deployment Options

### Option 1: Using `serve` (Local/Development)

For local testing of the production build:

```bash
cd frontend
npm install
npm run build
npm run serve
```

This will start a production server on `http://localhost:3000`.

### Option 2: Using the Deploy Script

For platforms that support Node.js applications:

```bash
cd frontend
npm install
npm run build
npm run deploy-serve
```

Or run directly:
```bash
cd frontend
npm install
npm run build
node deploy-serve.js
```

This script:
- Serves the built React app using `serve`
- Handles SPA routing correctly
- Supports environment variables for PORT configuration
- Includes proper error handling and graceful shutdown

### Option 3: Docker Deployment

You can also containerize the frontend:

```dockerfile
# Dockerfile for frontend
FROM node:18-alpine as builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/build /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

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

## Available Scripts

- `npm start`: Development server
- `npm run build`: Create production build
- `npm run serve`: Serve production build locally
- `npm run build-and-serve`: Build and serve in one command
- `npm run deploy`: Deploy to GitHub Pages
- `npm run deploy-serve`: Start production server using the deploy script
- `npm test`: Run tests

## Configuration Files

- `package.json`: Contains deployment scripts and homepage URL
- `public/404.html`: Handles SPA routing for GitHub Pages
- `public/index.html`: Contains SPA routing script for GitHub Pages
- `deploy-serve.js`: Custom deployment script using serve
- `.github/workflows/deploy.yml`: GitHub Actions deployment workflow 