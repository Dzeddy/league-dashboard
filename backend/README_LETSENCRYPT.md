# Let's Encrypt Automatic Certificate Management

This backend now supports automatic SSL certificate management using Let's Encrypt through Go's `autocert` package. This provides free, automatically renewing SSL certificates for your production deployment.

## Features

- **Automatic Certificate Acquisition**: Certificates are automatically obtained from Let's Encrypt
- **Automatic Renewal**: Certificates are automatically renewed before expiration
- **HTTP-01 Challenge Support**: Uses HTTP-01 challenge for domain validation
- **Multiple Domain Support**: Can handle multiple domains and subdomains
- **Secure Storage**: Certificates are cached locally in the `certs` directory

## Configuration

### Environment Variables

Set the following environment variables for Let's Encrypt autocert:

```bash
# Required: Comma-separated list of domains
AUTOCERT_DOMAINS=yourdomain.com,www.yourdomain.com,api.yourdomain.com

# Recommended: Contact email for Let's Encrypt notifications
AUTOCERT_EMAIL=admin@yourdomain.com

# SSL must be enabled
USE_SSL=true

# Port will default to 443 for HTTPS
PORT=443
```

### Example .env Configuration

```bash
# Let's Encrypt Configuration
AUTOCERT_DOMAINS=example.com,www.example.com
AUTOCERT_EMAIL=admin@example.com
USE_SSL=true
PORT=443

# Other required configuration...
RIOT_API_KEY=your_riot_api_key_here
MONGO_URI=mongodb://localhost:27017
# ... etc
```

## How It Works

1. **HTTP Server (Port 80)**: A background HTTP server runs on port 80 to handle Let's Encrypt HTTP-01 challenges
2. **HTTPS Server (Port 443)**: The main application runs on port 443 with automatic certificate management
3. **Certificate Cache**: Certificates are stored in the `./certs` directory and automatically renewed
4. **Domain Validation**: Let's Encrypt validates domain ownership through HTTP-01 challenges

## Prerequisites

### Domain Requirements

- **Public Domain**: You must own a public domain name
- **DNS Configuration**: Domain(s) must point to your server's public IP address
- **Port Access**: Ports 80 and 443 must be accessible from the internet

### Server Requirements

- **Public IP**: Server must have a public IP address
- **Firewall**: Ports 80 and 443 must be open
- **Root/Sudo Access**: May be required to bind to ports 80 and 443

## Deployment Steps

### 1. Configure DNS

Point your domain(s) to your server's public IP:

```
A    example.com        -> YOUR_SERVER_IP
A    www.example.com    -> YOUR_SERVER_IP
```

### 2. Set Environment Variables

```bash
export AUTOCERT_DOMAINS="example.com,www.example.com"
export AUTOCERT_EMAIL="admin@example.com"
export USE_SSL="true"
```

### 3. Run the Application

```bash
# Make sure ports 80 and 443 are available
sudo ./league_backend
```

### 4. Verify Certificate

The application will automatically:
- Request certificates from Let's Encrypt
- Handle domain validation
- Start serving HTTPS traffic
- Log certificate acquisition progress

## Certificate Management

### Automatic Renewal

Certificates are automatically renewed when they're within 30 days of expiration. No manual intervention required.

### Certificate Storage

Certificates are stored in the `./certs` directory:

```
certs/
├── example.com
├── example.com+1
└── acme_account+key
```

### Backup Considerations

Consider backing up the `./certs` directory to avoid re-requesting certificates after server restarts.

## Troubleshooting

### Common Issues

1. **Domain Not Accessible**
   - Verify DNS points to correct IP
   - Check firewall settings
   - Ensure ports 80/443 are open

2. **Certificate Request Failed**
   - Check domain ownership
   - Verify HTTP-01 challenge accessibility
   - Review Let's Encrypt rate limits

3. **Permission Denied (Ports 80/443)**
   - Run with sudo/root privileges
   - Use port forwarding if needed
   - Consider using higher ports with reverse proxy

### Debug Logs

The application logs certificate acquisition progress:

```
Starting HTTP server for Let's Encrypt challenges on :80
Starting HTTPS server with Let's Encrypt on :443
Configured domains: [example.com www.example.com]
Contact email: admin@example.com
```

### Rate Limits

Let's Encrypt has rate limits:
- 50 certificates per registered domain per week
- 5 duplicate certificates per week
- 300 new orders per account per 3 hours

## Development vs Production

### Development (USE_SSL=false)

```bash
USE_SSL=false
PORT=8080
```

Uses HTTP only, no certificates required.

### Production (USE_SSL=true)

```bash
USE_SSL=true
AUTOCERT_DOMAINS=yourdomain.com
AUTOCERT_EMAIL=admin@yourdomain.com
```

Uses HTTPS with automatic Let's Encrypt certificates.

## Security Considerations

- Certificates are automatically renewed
- TLS 1.2+ is enforced
- Strong cipher suites are configured
- HTTPS redirects are handled automatically
- Contact email helps with security notifications

## Migration from Manual Certificates

If migrating from manual SSL certificates:

1. Set `USE_SSL=true`
2. Configure `AUTOCERT_DOMAINS` and `AUTOCERT_EMAIL`
3. Remove or comment out `SSL_CERT_FILE` and `SSL_KEY_FILE`
4. Restart the application

The autocert system will take over certificate management automatically. 