# Redis Configuration

This application supports Redis password authentication and database selection through environment variables.

## Environment Variables

### Required
- `RIOT_API_KEY`: Your Riot API key

### Optional Redis Configuration
- `REDIS_ADDR`: Redis server address (default: `localhost:6379`)
- `REDIS_PASSWORD`: Redis password for authentication (default: empty - no auth)
- `REDIS_DB`: Redis database number to use (default: `0`)

## Setup

1. Copy `env.example` to `.env`:
   ```bash
   cp env.example .env
   ```

2. Edit `.env` with your actual values:
   ```env
   RIOT_API_KEY=your_actual_riot_api_key
   REDIS_PASSWORD=your_redis_password_if_needed
   REDIS_DB=0
   ```

## Redis Authentication

- If `REDIS_PASSWORD` is set, the application will use password authentication
- If `REDIS_PASSWORD` is empty or not set, the application will connect without authentication
- The application will log whether password authentication is enabled or disabled on startup

## Examples

### Redis without password (local development)
```env
REDIS_ADDR=localhost:6379
# REDIS_PASSWORD= (leave empty or comment out)
REDIS_DB=0
```

### Redis with password (production)
```env
REDIS_ADDR=your-redis-server:6379
REDIS_PASSWORD=your-secure-password
REDIS_DB=0
```

### Redis Cloud/Managed Services
```env
REDIS_ADDR=your-cloud-redis.example.com:6379
REDIS_PASSWORD=your-cloud-redis-password
REDIS_DB=0
``` 