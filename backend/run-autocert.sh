#!/bin/bash

# League Dashboard Backend - Let's Encrypt Autocert Runner
# This script helps you run the backend with Let's Encrypt automatic certificate management

set -e

echo "🔐 League Dashboard Backend - Let's Encrypt Autocert Setup"
echo "=========================================================="

# Check if running as root (required for ports 80/443)
if [ "$EUID" -ne 0 ]; then
    echo "⚠️  Warning: Not running as root. You may need sudo for ports 80/443."
    echo "   If you get permission errors, run: sudo ./run-autocert.sh"
    echo ""
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "📝 No .env file found. Creating from template..."
    cp env.example .env
    echo "✅ Created .env file from template"
    echo "⚠️  Please edit .env and set your actual domains and email:"
    echo "   - AUTOCERT_DOMAINS=yourdomain.com,www.yourdomain.com"
    echo "   - AUTOCERT_EMAIL=admin@yourdomain.com"
    echo "   - RIOT_API_KEY=your_actual_api_key"
    echo ""
    read -p "Press Enter to continue after editing .env file..."
fi

# Load environment variables
if [ -f ".env" ]; then
    echo "📋 Loading environment variables from .env..."
    export $(cat .env | grep -v '^#' | xargs)
fi

# Validate required environment variables
echo "🔍 Validating configuration..."

if [ -z "$RIOT_API_KEY" ] || [ "$RIOT_API_KEY" = "your_riot_api_key_here" ]; then
    echo "❌ RIOT_API_KEY not set or using default value"
    echo "   Please set your actual Riot API key in .env"
    exit 1
fi

if [ -z "$AUTOCERT_DOMAINS" ] || [ "$AUTOCERT_DOMAINS" = "yourdomain.com,www.yourdomain.com" ]; then
    echo "❌ AUTOCERT_DOMAINS not set or using default value"
    echo "   Please set your actual domain(s) in .env"
    exit 1
fi

if [ -z "$AUTOCERT_EMAIL" ] || [ "$AUTOCERT_EMAIL" = "your-email@example.com" ]; then
    echo "❌ AUTOCERT_EMAIL not set or using default value"
    echo "   Please set your actual email in .env"
    exit 1
fi

# Check if binary exists
if [ ! -f "league_backend" ]; then
    echo "🔨 Building application..."
    go build -o league_backend main.go riotapi.go models.go handlers.go
    echo "✅ Build complete"
fi

# Create certs directory if it doesn't exist
if [ ! -d "certs" ]; then
    echo "📁 Creating certs directory..."
    mkdir -p certs
    chmod 700 certs
fi

# Display configuration
echo ""
echo "🚀 Starting with configuration:"
echo "   Domains: $AUTOCERT_DOMAINS"
echo "   Email: $AUTOCERT_EMAIL"
echo "   SSL: ${USE_SSL:-true}"
echo "   MongoDB: ${MONGO_URI:-mongodb://localhost:27017}"
echo "   Redis: ${REDIS_ADDR:-localhost:6379}"
echo ""

# Check if ports are available
echo "🔍 Checking port availability..."

if lsof -Pi :80 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "⚠️  Port 80 is already in use. Let's Encrypt challenges may fail."
    echo "   Current process using port 80:"
    lsof -Pi :80 -sTCP:LISTEN
    echo ""
fi

if lsof -Pi :443 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "⚠️  Port 443 is already in use. HTTPS server may fail to start."
    echo "   Current process using port 443:"
    lsof -Pi :443 -sTCP:LISTEN
    echo ""
fi

echo "🎯 Starting League Dashboard Backend with Let's Encrypt..."
echo "   HTTP (challenges): http://localhost:80"
echo "   HTTPS (main app): https://localhost:443"
echo ""
echo "📝 Logs will show certificate acquisition progress."
echo "   First run may take a moment to obtain certificates."
echo ""

# Start the application
exec ./league_backend 