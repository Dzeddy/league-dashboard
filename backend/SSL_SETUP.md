# SSL Setup Guide

This guide explains how to configure SSL/HTTPS for the League Dashboard backend.

## Quick Start

The backend now supports HTTPS by default. When you start the server, it will:

1. **Automatically generate** self-signed certificates if none are found
2. **Start on port 8443** (HTTPS) by default
3. **Use secure TLS configuration** with modern cipher suites

## Environment Variables

Configure SSL behavior with these environment variables:

```bash
# Enable/disable SSL (default: true)
USE_SSL=true

# Server port (default: 8443 for HTTPS, 8080 for HTTP)
PORT=8443

# SSL certificate files (optional - auto-generated if not provided)
SSL_CERT_FILE=server.crt
SSL_KEY_FILE=server.key
```

## Certificate Options

### Option 1: Auto-Generated Certificates (Recommended for Development)

The server will automatically generate self-signed certificates when started:

```bash
# Simply run the server - certificates will be generated automatically
go run .
```

Generated certificates are valid for:
- `localhost`
- `*.localhost`
- `127.0.0.1`
- `::1` (IPv6 loopback)

### Option 2: Custom Certificates (Production)

For production, use proper SSL certificates:

```bash
# Set environment variables
export SSL_CERT_FILE=/path/to/your/certificate.crt
export SSL_KEY_FILE=/path/to/your/private.key
export USE_SSL=true
export PORT=443

# Start the server
go run .
```

### Option 3: Disable SSL (Development Only)

To run without SSL (HTTP only):

```bash
export USE_SSL=false
export PORT=8080
go run .
```

⚠️ **Warning**: Only disable SSL for local development. Production should always use HTTPS.

## CORS Configuration

The server automatically includes HTTPS origins in CORS:

- `https://localhost:3000` (local development)
- `https://dzeddy.github.io` (GitHub Pages)
- `https://league-dashboard-eosin.vercel.app` (Vercel deployment)

## Frontend Configuration

Update your frontend to use the HTTPS endpoint:

```javascript
// For local development
const API_BASE_URL = 'https://localhost:8443/api';

// For production
const API_BASE_URL = 'https://your-domain.com/api';
```

## Troubleshooting

### Self-Signed Certificate Warnings

Browsers will show security warnings for self-signed certificates. For development:

1. **Accept the certificate** in your browser
2. Or add the certificate to your system's trusted certificates
3. Or use a tool like [mkcert](https://github.com/FiloSottile/mkcert) for locally-trusted certificates

### Certificate Generation Errors

If certificate generation fails:

1. Check file permissions in the backend directory
2. Ensure the directory is writable
3. Check for existing certificate files that might be corrupted

### Port Already in Use

If port 8443 is already in use:

```bash
export PORT=8444  # Use a different port
go run .
```

## Production Deployment

For production deployment:

1. **Use proper SSL certificates** from a CA (Let's Encrypt, etc.)
2. **Set appropriate environment variables**
3. **Configure firewall rules** for your chosen port
4. **Use a reverse proxy** (nginx, Apache) if needed
5. **Update DNS records** to point to your server

Example nginx configuration:
```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;
    
    ssl_certificate /path/to/certificate.crt;
    ssl_certificate_key /path/to/private.key;
    
    location /api/ {
        proxy_pass https://localhost:8443/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Security Features

The SSL configuration includes:

- **TLS 1.2+ only** (no older protocols)
- **Strong cipher suites** prioritizing ECDHE and AEAD
- **Server cipher preference** for security
- **Modern elliptic curves** (P-521, P-384, P-256)

## Testing SSL

Test your SSL configuration:

```bash
# Test SSL connection
curl -k https://localhost:8443/api/health

# Check certificate details
openssl s_client -connect localhost:8443 -servername localhost
``` 