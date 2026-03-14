#!/bin/bash
# 0Chain Nginx Path-Based Reverse Proxy Setup
#
# Creates a single nginx server block that routes all 0Chain services
# via path-based URLs (e.g., test.zus.network/miner01/).
#
# Usage:
#   ./deploy_nginx.sh <domain>                    # HTTP only
#   ./deploy_nginx.sh <domain> --ssl              # With Let's Encrypt SSL
#   ./deploy_nginx.sh <domain> --ssl --email admin@zus.network
#
# Prerequisites:
#   - nginx installed
#   - Domain DNS pointing to this server's IP
#   - All 0Chain services running with ports mapped to 0.0.0.0

set -e

DOMAIN="${1:-}"
ENABLE_SSL=false
EMAIL="admin@zus.network"

# Parse args
shift || true
while [[ $# -gt 0 ]]; do
    case "$1" in
        --ssl)     ENABLE_SSL=true; shift ;;
        --email)   EMAIL="$2"; shift 2 ;;
        *)         echo "Unknown arg: $1"; exit 1 ;;
    esac
done

if [ -z "$DOMAIN" ]; then
    echo "Usage: $0 <domain> [--ssl] [--email admin@example.com]"
    echo ""
    echo "Examples:"
    echo "  $0 test.zus.network"
    echo "  $0 test.zus.network --ssl --email admin@zus.network"
    exit 1
fi

echo "Setting up nginx path-based routing for ${DOMAIN}"

# Install nginx if needed
if ! command -v nginx &> /dev/null; then
    echo "Installing nginx..."
    if command -v apt-get &> /dev/null; then
        apt-get update -qq && apt-get install -y -qq nginx
    elif command -v yum &> /dev/null; then
        yum install -y nginx
    else
        echo "ERROR: Cannot install nginx"
        exit 1
    fi
fi

# Create directories
mkdir -p /etc/nginx/sites-available /etc/nginx/sites-enabled

CONF_FILE="/etc/nginx/sites-available/${DOMAIN}"

# Port mapping reference:
#   Miners:      7071-7074 (miner-1 through miner-4)
#   Sharders:    7171-7172 (sharder-1 through sharder-2)
#   Blobbers:    5051-5062 (blobber-1 through blobber-12)
#   Validators:  5041-5042 (validator-1,2), 5063-5072 (validator-3 through validator-12)
#   0dns:        9091
#   0box:        9081
#   zauth:       8080
#   zvault:      8090
#   ES:          9200
#   pgAdmin:     5050

cat > "$CONF_FILE" << 'NGINXEOF'
server {
    listen 80;
    server_name DOMAIN_PLACEHOLDER;

    # --- Miners ---
    location /miner01/ { proxy_pass http://localhost:7071/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }
    location /miner02/ { proxy_pass http://localhost:7072/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }
    location /miner03/ { proxy_pass http://localhost:7073/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }
    location /miner04/ { proxy_pass http://localhost:7074/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }

    # --- Sharders ---
    location /sharder01/ { proxy_pass http://localhost:7171/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }
    location /sharder02/ { proxy_pass http://localhost:7172/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }

    # --- Blobbers (ports 5051-5062) ---
    location /blobber01/ { proxy_pass http://localhost:5051/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /blobber02/ { proxy_pass http://localhost:5052/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /blobber03/ { proxy_pass http://localhost:5053/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /blobber04/ { proxy_pass http://localhost:5054/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /blobber05/ { proxy_pass http://localhost:5055/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /blobber06/ { proxy_pass http://localhost:5056/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /blobber07/ { proxy_pass http://localhost:5057/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /blobber08/ { proxy_pass http://localhost:5058/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /blobber09/ { proxy_pass http://localhost:5059/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /blobber10/ { proxy_pass http://localhost:5060/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /blobber11/ { proxy_pass http://localhost:50611/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /blobber12/ { proxy_pass http://localhost:50612/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }

    # --- Validators ---
    location /validator01/ { proxy_pass http://localhost:5041/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /validator02/ { proxy_pass http://localhost:5042/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /validator03/ { proxy_pass http://localhost:5063/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /validator04/ { proxy_pass http://localhost:5064/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /validator05/ { proxy_pass http://localhost:5065/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /validator06/ { proxy_pass http://localhost:5066/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /validator07/ { proxy_pass http://localhost:5067/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /validator08/ { proxy_pass http://localhost:5068/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /validator09/ { proxy_pass http://localhost:5069/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /validator10/ { proxy_pass http://localhost:5070/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /validator11/ { proxy_pass http://localhost:50711/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /validator12/ { proxy_pass http://localhost:50712/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }

    # --- Services ---
    location /0dns/    { proxy_pass http://localhost:9091/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }
    location /0box/    { proxy_pass http://localhost:9081/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }
    location /zauth/   { proxy_pass http://localhost:8080/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }
    location /zvault/  { proxy_pass http://localhost:8090/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }
    location /elasticsearch/ { proxy_pass http://localhost:9200/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
    location /pgadmin/ { proxy_pass http://localhost:5050/; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; proxy_set_header X-Script-Name /pgadmin; }

    # --- Landing page ---
    location / {
        default_type text/plain;
        return 200 "0Chain Test Network\n\nChain:\n  /miner01..04/_diagnostics\n  /sharder01..02/_diagnostics\n\nStorage:\n  /blobber01..12/\n  /validator01..12/\n\nServices:\n  /0dns/network\n  /0box/\n  /zauth/\n  /zvault/\n  /elasticsearch/\n  /pgadmin/\n";
    }
}
NGINXEOF

# Replace domain placeholder
sed -i "s/DOMAIN_PLACEHOLDER/${DOMAIN}/" "$CONF_FILE"

# Enable site
ln -sf "$CONF_FILE" "/etc/nginx/sites-enabled/${DOMAIN}"

# Remove default site
rm -f /etc/nginx/sites-enabled/default

# Test config
echo "Testing nginx configuration..."
nginx -t

# Reload nginx
systemctl reload nginx 2>/dev/null || nginx -s reload 2>/dev/null || true
echo "Nginx reloaded with path-based routing"

# SSL setup
if [ "$ENABLE_SSL" = "true" ]; then
    if ! command -v certbot &> /dev/null; then
        echo "Installing certbot..."
        apt-get install -y -qq certbot python3-certbot-nginx 2>/dev/null || \
            yum install -y certbot python3-certbot-nginx 2>/dev/null
    fi

    echo "Requesting SSL certificate for ${DOMAIN}..."
    certbot --nginx --non-interactive --agree-tos --email "$EMAIL" \
        --redirect -d "${DOMAIN}" 2>&1 || {
        echo "WARNING: Certbot failed. Services still available via HTTP."
        echo "Retry: certbot --nginx -d ${DOMAIN}"
    }
fi

echo ""
echo "Done! Service URLs:"
echo "  Chain:      http://${DOMAIN}/miner01..04/  http://${DOMAIN}/sharder01..02/"
echo "  Blobbers:   http://${DOMAIN}/blobber01..12/"
echo "  Validators: http://${DOMAIN}/validator01..12/"
echo "  0dns:       http://${DOMAIN}/0dns/"
echo "  0box:       http://${DOMAIN}/0box/"
echo "  zauth:      http://${DOMAIN}/zauth/"
echo "  zvault:     http://${DOMAIN}/zvault/"
echo "  ES:         http://${DOMAIN}/elasticsearch/"
echo "  pgAdmin:    http://${DOMAIN}/pgadmin/"
