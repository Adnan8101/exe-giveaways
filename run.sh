#!/bin/bash

# Production Deployment Script for Discord Bot
# Installs dependencies, sets up services, and keeps everything running 24/7

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "========================================"
echo "Discord Bot Production Deployment Setup"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

# Check if running as root for installations
check_sudo() {
    if [ "$EUID" -ne 0 ]; then
        print_warning "Some operations require sudo privileges"
        SUDO="sudo"
    else
        SUDO=""
    fi
}

# Install Go if not present
install_go() {
    if command -v go &> /dev/null; then
        print_status "Go is already installed ($(go version))"
        return 0
    fi

    print_warning "Go not found. Installing Go 1.21..."
    
    # Download and install Go
    cd /tmp
    wget -q https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
    $SUDO rm -rf /usr/local/go
    $SUDO tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
    
    # Add to PATH if not already there
    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
    fi
    
    export PATH=$PATH:/usr/local/go/bin
    export PATH=$PATH:$HOME/go/bin
    
    cd "$SCRIPT_DIR"
    print_status "Go installed successfully"
}

# Install Redis if not present
install_redis() {
    if command -v redis-server &> /dev/null; then
        print_status "Redis is already installed"
        return 0
    fi

    print_warning "Redis not found. Installing Redis..."
    
    $SUDO apt-get update -qq
    $SUDO apt-get install -y redis-server redis-tools
    
    # Configure Redis for production
    $SUDO sed -i 's/^supervised no/supervised systemd/' /etc/redis/redis.conf
    $SUDO sed -i 's/^# maxmemory <bytes>/maxmemory 256mb/' /etc/redis/redis.conf
    $SUDO sed -i 's/^# maxmemory-policy noeviction/maxmemory-policy allkeys-lru/' /etc/redis/redis.conf
    
    $SUDO systemctl enable redis-server
    $SUDO systemctl restart redis-server
    
    print_status "Redis installed and configured"
}

# Install PostgreSQL if not present
install_postgresql() {
    if command -v psql &> /dev/null; then
        print_status "PostgreSQL is already installed"
        return 0
    fi

    print_warning "PostgreSQL not found. Installing PostgreSQL..."
    
    $SUDO apt-get update -qq
    $SUDO apt-get install -y postgresql postgresql-contrib
    
    $SUDO systemctl enable postgresql
    $SUDO systemctl start postgresql
    
    print_status "PostgreSQL installed"
}

# Health check function
check_service_health() {
    local service_name=$1
    local check_command=$2
    
    if eval "$check_command" &> /dev/null; then
        print_status "$service_name is running"
        return 0
    else
        print_error "$service_name is not responding"
        return 1
    fi
}

# Create systemd service file for the bot
create_bot_service() {
    print_status "Creating systemd service for bot..."
    
    local service_file="/etc/systemd/system/discord-bot.service"
    
    $SUDO tee "$service_file" > /dev/null <<EOF
[Unit]
Description=Discord Giveaway Bot
After=network.target postgresql.service redis-server.service
Requires=redis-server.service

[Service]
Type=simple
User=$USER
WorkingDirectory=$SCRIPT_DIR
ExecStart=$SCRIPT_DIR/bot
Restart=always
RestartSec=10
StandardOutput=append:/var/log/discord-bot.log
StandardError=append:/var/log/discord-bot-error.log

# Resource limits
LimitNOFILE=65536
MemoryMax=1G

# Environment
Environment="PATH=/usr/local/go/bin:/usr/bin:/bin"

[Install]
WantedBy=multi-user.target
EOF

    $SUDO systemctl daemon-reload
    print_status "Bot service created"
}

# Create health check script
create_health_check() {
    print_status "Creating health check monitor..."
    
    cat > "$SCRIPT_DIR/health_check.sh" <<'EOF'
#!/bin/bash

# Health check script - runs every minute via cron

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/discord-bot-health.log"

log_message() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
}

# Check Redis
if ! redis-cli ping &> /dev/null; then
    log_message "ERROR: Redis not responding, restarting..."
    sudo systemctl restart redis-server
fi

# Check PostgreSQL
if ! pg_isready &> /dev/null; then
    log_message "ERROR: PostgreSQL not responding, restarting..."
    sudo systemctl restart postgresql
fi

# Check Bot
if ! systemctl is-active --quiet discord-bot; then
    log_message "ERROR: Bot not running, restarting..."
    sudo systemctl restart discord-bot
fi

# Check if bot binary exists
if [ ! -f "$SCRIPT_DIR/bot" ]; then
    log_message "ERROR: Bot binary missing, rebuilding..."
    cd "$SCRIPT_DIR"
    go build -ldflags="-s -w" -o bot .
    sudo systemctl restart discord-bot
fi
EOF

    chmod +x "$SCRIPT_DIR/health_check.sh"
    
    # Add to crontab if not already there
    (crontab -l 2>/dev/null | grep -v health_check.sh; echo "* * * * * $SCRIPT_DIR/health_check.sh") | crontab -
    
    print_status "Health check monitor installed (runs every minute)"
}

# Main installation flow
main() {
    echo ""
    print_status "Starting production deployment..."
    echo ""
    
    # Check sudo availability
    check_sudo
    
    # Install dependencies
    print_status "Phase 1: Installing dependencies..."
    install_go
    install_redis
    install_postgresql
    echo ""
    
    # Build the bot
    print_status "Phase 2: Building bot..."
    if [ -f "go.mod" ]; then
        go mod download
        go build -ldflags="-s -w" -o bot .
        print_status "Bot built successfully"
    else
        print_error "go.mod not found!"
        exit 1
    fi
    echo ""
    
    # Check service health
    print_status "Phase 3: Checking service health..."
    check_service_health "Redis" "redis-cli ping"
    check_service_health "PostgreSQL" "pg_isready"
    echo ""
    
    # Setup systemd service
    print_status "Phase 4: Setting up systemd service..."
    create_bot_service
    create_health_check
    
    # Start the bot service
    $SUDO systemctl enable discord-bot
    $SUDO systemctl restart discord-bot
    
    echo ""
    print_status "Phase 5: Service status..."
    $SUDO systemctl status redis-server --no-pager -l || true
    echo ""
    $SUDO systemctl status postgresql --no-pager -l || true
    echo ""
    $SUDO systemctl status discord-bot --no-pager -l || true
    
    echo ""
    echo "========================================"
    print_status "Deployment complete! Bot is now running 24/7"
    echo "========================================"
    echo ""
    echo "Useful commands:"
    echo "  View bot logs:        sudo journalctl -u discord-bot -f"
    echo "  View health log:      tail -f /var/log/discord-bot-health.log"
    echo "  Restart bot:          sudo systemctl restart discord-bot"
    echo "  Stop bot:             sudo systemctl stop discord-bot"
    echo "  Check status:         sudo systemctl status discord-bot"
    echo ""
    echo "Services will auto-restart on failure and survive reboots!"
    echo ""
}

# Run main function
main
