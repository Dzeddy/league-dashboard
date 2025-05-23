# Frontend Environment Configuration

The frontend uses environment variables to configure the API connection. These variables must be prefixed with `REACT_APP_` to be accessible in the React application.

## Setup

### For Local Development

1. Create a `.env.local` file in the `frontend/` directory:
```bash
cd frontend
touch .env.local
```

2. Add the following configuration to `.env.local`:
```
REACT_APP_API_BASE_URL=http://localhost:8080/api
```

### For Production Deployment

Set the environment variable to point to your production API:
```
REACT_APP_API_BASE_URL=https://your-domain.com/api
```

## Available Environment Variables

- `REACT_APP_API_BASE_URL`: The base URL for the backend API
  - Default: `http://localhost:8080/api`
  - Example: `https://api.yourleaguedashboard.com/api`

## Docker Deployment

For Docker deployments, you can pass the environment variable when running the container:

```bash
docker run -e REACT_APP_API_BASE_URL=https://your-api-domain.com/api your-frontend-image
```

Or use Docker Compose environment files:

```yaml
services:
  frontend:
    image: your-frontend-image
    environment:
      - REACT_APP_API_BASE_URL=https://your-api-domain.com/api
```

## Notes

- Environment variables are embedded into the build at build time, not runtime
- The application will fallback to `http://localhost:8080/api` if no environment variable is set
- Remember to rebuild the React application after changing environment variables 