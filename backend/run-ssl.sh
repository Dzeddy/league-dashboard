#!/bin/bash

# SSL-enabled backend runner script
# This script sets up SSL configuration and starts the League Dashboard backend

echo "🚀 Starting League Dashboard Backend with SSL"
echo "=============================================="

# Check if .env file exists
if [ -f ".env" ]; then
    echo "✅ Found .env file - loading environment variables"
    source .env
else
    echo "⚠️  No .env file found - using default/system environment variables"
fi

# Set SSL defaults if not already set
export USE_SSL=${USE_SSL:-true}
export PORT=${PORT:-8443}

# Print configuration
echo ""
echo "Configuration:"
echo "  SSL Enabled: $USE_SSL"
echo "  Port: $PORT"
echo "  Certificate: ${SSL_CERT_FILE:-server.crt} (auto-generated if missing)"
echo "  Private Key: ${SSL_KEY_FILE:-server.key} (auto-generated if missing)"
echo ""

# Build and run
echo "📦 Building backend..."
if go build -o league_backend_ssl .; then
    echo "✅ Build successful"
    echo ""
    echo "🔒 Starting HTTPS server..."
    echo "🌐 Frontend should connect to: https://localhost:$PORT/api"
    echo "🔧 Health check: https://localhost:$PORT/api/health"
    echo ""
    echo "Press Ctrl+C to stop the server"
    echo "=================================="
    ./league_backend_ssl
else
    echo "❌ Build failed"
    exit 1
fi 