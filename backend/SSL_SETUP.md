# SSL Setup Guide

This guide explains how to configure SSL/HTTPS for the League Dashboard backend.

## Quick Start

The backend supports multiple SSL/HTTPS options:

1. **Let's Encrypt Autocert** (Recommended for Production) - Automatic free SSL certificates
2. **Auto-Generated Self-Signed** (Development) - Automatically generated certificates
3. **Custom Certificates** (Production) - Bring your own certificates
4. **HTTP Only** (Development) - Disable SSL entirely

## SSL Options

### Option 1: Let's Encrypt Autocert (🌟 Recommended for Production)

**NEW**: Automatic SSL certificate management with Let's Encrypt. Free, trusted certificates that auto-renew.

```bash
# Set environment variables
export AUTOCERT_DOMAINS="yourdomain.com,www.yourdomain.com"
export AUTOCERT_EMAIL="admin@yourdomain.com"
export USE_SSL=true

# Run with autocert
./run-autocert.sh
```

**Features:**
- ✅ Free SSL certificates from Let's Encrypt
- ✅ Automatic renewal (no manual intervention)
- ✅ Trusted by all browsers (no warnings)
- ✅ Multiple domain support
- ✅ HTTP-01 challenge validation

**Requirements:**
- Public domain name pointing to your server
- Ports 80 and 443 accessible from internet
- Root/sudo access for privileged ports

📖 **See [README_LETSENCRYPT.md](README_LETSENCRYPT.md) for detailed setup instructions.**

### Option 2: Auto-Generated Certificates (Development)

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

### Option 3: Custom Certificates (Production)

For production with your own certificates:

```bash
# Set environment variables
export SSL_CERT_FILE=/path/to/your/certificate.crt
export SSL_KEY_FILE=/path/to/your/private.key
export USE_SSL=true
export PORT=443

# Start the server
go run .
```

### Option 4: Disable SSL (Development Only)

To run without SSL (HTTP only):

```bash
export USE_SSL=false
export PORT=8080
go run .
```

⚠️ **Warning**: Only disable SSL for local development. Production should always use HTTPS.

## Environment Variables

Configure SSL behavior with these environment variables:

```bash
# SSL Mode
USE_SSL=true                    # Enable/disable SSL (default: true)
PORT=443                        # Server port (443 for HTTPS, 8080 for HTTP)

# Let's Encrypt Autocert (Recommended for Production)
AUTOCERT_DOMAINS=example.com,www.example.com  # Comma-separated domains
AUTOCERT_EMAIL=admin@example.com               # Contact email

# Manual Certificates (Alternative for Production)
SSL_CERT_FILE=server.crt        # Path to certificate file
SSL_KEY_FILE=server.key         # Path to private key file
```

## Production Deployment Comparison

| Method | Pros | Cons | Best For |
|--------|------|------|----------|
| **Let's Encrypt Autocert** | Free, automatic, trusted, auto-renewal | Requires public domain, ports 80/443 | Production servers |
| **Custom Certificates** | Full control, works offline | Manual renewal, costs money | Enterprise/internal |
| **Self-Signed** | Quick setup, no external deps | Browser warnings, not trusted | Development only |

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

// For production with Let's Encrypt
const API_BASE_URL = 'https://yourdomain.com/api';
```

## Troubleshooting

### Let's Encrypt Issues

1. **Domain not accessible**: Verify DNS points to correct IP
2. **Challenge failed**: Ensure port 80 is open and accessible
3. **Rate limits**: Let's Encrypt has rate limits (50 certs/week per domain)

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

If ports are already in use:

```bash
# Check what's using the ports
lsof -i :80
lsof -i :443

# Use different ports (development only)
export PORT=8444
go run .
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

# Test Let's Encrypt certificate
curl https://yourdomain.com/api/health
```

## Migration Guide

### From Self-Signed to Let's Encrypt

1. Set up your domain DNS to point to your server
2. Configure environment variables:
   ```bash
   export AUTOCERT_DOMAINS="yourdomain.com"
   export AUTOCERT_EMAIL="admin@yourdomain.com"
   export USE_SSL=true
   ```
3. Remove or comment out `SSL_CERT_FILE` and `SSL_KEY_FILE`
4. Restart with `./run-autocert.sh`

### From Custom Certificates to Let's Encrypt

1. Configure autocert environment variables
2. Remove custom certificate configuration
3. Restart the application
4. Let's Encrypt will take over certificate management 