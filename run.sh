#!/bin/bash

# ==============================================================================
# 🚀 DISCORD BOT - UNIFIED RUN SCRIPT
# ==============================================================================
# This script handles setup, optimization, building, and deployment.
# It is designed to be the ONLY script you need to run.
# ==============================================================================

set -e

# Configuration
BOT_NAME="discord-giveaway-bot"
PM2_CONFIG="ecosystem.config.js"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║       Discord Bot - Unified Run Script                  ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

# ------------------------------------------------------------------------------
# 1. OS DETECTION & PRE-CHECKS
# ------------------------------------------------------------------------------
OS="$(uname -s)"
echo -e "${BLUE}[STEP 1] Checking Environment (${OS})...${NC}"

IS_LINUX=false
IS_ROOT=false

if [ "$OS" = "Linux" ]; then
    IS_LINUX=true
    if [ "$EUID" -eq 0 ]; then
        IS_ROOT=true
    else
        echo -e "${YELLOW}⚠️  Not running as root. System optimizations will be skipped.${NC}"
        echo -e "${YELLOW}   Run with 'sudo bash run.sh' for full performance tuning.${NC}"
    fi
elif [ "$OS" = "Darwin" ]; then
    echo -e "${YELLOW}⚠️  Running on macOS. Linux-specific optimizations will be skipped.${NC}"
else
    echo -e "${RED}❌ Unsupported OS: $OS${NC}"
    exit 1
fi

# ------------------------------------------------------------------------------
# 2. SYSTEM OPTIMIZATION (Linux Only)
# ------------------------------------------------------------------------------
if [ "$IS_LINUX" = true ] && [ "$IS_ROOT" = true ]; then
    echo -e "${BLUE}[STEP 2] Applying System Optimizations...${NC}"
    
    # Kernel Tuning
    cat > /etc/sysctl.d/99-discord-bot.conf <<EOF
# Network Performance
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr
net.ipv4.tcp_window_scaling=1
net.ipv4.tcp_sack=1
net.core.rmem_max=16777216
net.core.wmem_max=16777216
net.ipv4.tcp_rmem=4096 87380 16777216
net.ipv4.tcp_wmem=4096 65536 16777216
net.core.netdev_max_backlog=5000
net.core.somaxconn=4096
fs.file-max=100000
EOF
    sysctl -p /etc/sysctl.d/99-discord-bot.conf > /dev/null 2>&1 || true
    echo -e "${GREEN}✓ Kernel network tuning applied${NC}"

    # CPU Governor
    if [ -d /sys/devices/system/cpu/cpu0/cpufreq ]; then
        for cpu in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
            echo performance > "$cpu" 2>/dev/null || true
        done
        echo -e "${GREEN}✓ CPU governor set to performance${NC}"
    fi
else
    echo -e "${BLUE}[STEP 2] Skipping System Optimizations (Not Linux Root)${NC}"
fi

# ------------------------------------------------------------------------------
# 3. DEPENDENCY CHECK & INSTALL
# ------------------------------------------------------------------------------
echo -e "${BLUE}[STEP 3] Checking Dependencies...${NC}"

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}⚠️  Go not found.${NC}"
    if [ "$IS_LINUX" = true ] && [ "$IS_ROOT" = true ]; then
        echo -e "${CYAN}→ Installing Go...${NC}"
        wget -q https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
        rm -rf /usr/local/go && tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
        rm go1.23.4.linux-amd64.tar.gz
        export PATH=$PATH:/usr/local/go/bin
        echo "export PATH=\$PATH:/usr/local/go/bin" >> /etc/profile
        echo -e "${GREEN}✓ Go installed${NC}"
    else
        echo -e "${RED}❌ Please install Go manually.${NC}"
        exit 1
    fi
else
    GO_CURRENT=$(go version | awk '{print $3}' | sed 's/go//')
    echo -e "${GREEN}✓ Go is installed ($GO_CURRENT)${NC}"
    
    # Check Go version
    if [[ "$GO_CURRENT" < "1.21" ]]; then
         echo -e "${YELLOW}⚠️  Go version might be too old. Recommended: 1.23+${NC}"
    fi
fi

# Check Redis
if ! command -v redis-cli &> /dev/null; then
    echo -e "${YELLOW}⚠️  Redis not found.${NC}"
    if [ "$IS_LINUX" = true ] && [ "$IS_ROOT" = true ]; then
        echo -e "${CYAN}→ Installing Redis...${NC}"
        apt update -qq && apt install -y -qq redis-server
        systemctl enable redis-server
        systemctl start redis-server
        echo -e "${GREEN}✓ Redis installed and started${NC}"
    else
        echo -e "${YELLOW}⚠️  Please ensure Redis is installed and running.${NC}"
    fi
else
    echo -e "${GREEN}✓ Redis is installed${NC}"
fi

# Check PM2
if ! command -v pm2 &> /dev/null; then
    echo -e "${YELLOW}⚠️  PM2 not found.${NC}"
    if [ "$IS_LINUX" = true ] && [ "$IS_ROOT" = true ]; then
        echo -e "${CYAN}→ Installing Node.js & PM2...${NC}"
        curl -fsSL https://deb.nodesource.com/setup_20.x | bash - > /dev/null
        apt install -y -qq nodejs
        npm install -g pm2
        echo -e "${GREEN}✓ PM2 installed${NC}"
    else
        echo -e "${RED}❌ Please install PM2 (npm install -g pm2).${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✓ PM2 is installed${NC}"
fi

# ------------------------------------------------------------------------------
# 4. BUILD
# ------------------------------------------------------------------------------
echo -e "${BLUE}[STEP 4] Building Bot...${NC}"

echo -e "${CYAN}→ Tidy modules...${NC}"
go mod tidy

echo -e "${CYAN}→ Compiling...${NC}"
if go build -ldflags="-s -w" -o "$BOT_NAME" .; then
    SIZE=$(ls -lh "$BOT_NAME" | awk '{print $5}')
    echo -e "${GREEN}✓ Build successful (Size: $SIZE)${NC}"
else
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

# ------------------------------------------------------------------------------
# 5. DEPLOY & RUN
# ------------------------------------------------------------------------------
echo -e "${BLUE}[STEP 5] Deploying with PM2...${NC}"

# Ensure ecosystem.config.js exists
if [ ! -f "$PM2_CONFIG" ]; then
    echo -e "${CYAN}→ Creating $PM2_CONFIG...${NC}"
    cat > "$PM2_CONFIG" <<EOF
module.exports = {
  apps: [{
    name: '$BOT_NAME',
    script: './$BOT_NAME',
    instances: 1,
    autorestart: true,
    watch: false,
    max_memory_restart: '2G',
    env: {
      NODE_ENV: 'production'
    }
  }]
};
EOF
fi

if pm2 list | grep -q "$BOT_NAME"; then
    echo -e "${CYAN}→ Restarting existing process (Clean Start)...${NC}"
    pm2 delete "$BOT_NAME"
    pm2 start "$PM2_CONFIG"
else
    echo -e "${CYAN}→ Starting new process...${NC}"
    pm2 start "$PM2_CONFIG"
fi

pm2 save > /dev/null 2>&1

# ------------------------------------------------------------------------------
# 6. STATUS & LATENCY
# ------------------------------------------------------------------------------
echo -e "${BLUE}[STEP 6] Status Check...${NC}"
sleep 2
pm2 status "$BOT_NAME"

echo -e "${CYAN}→ Checking Discord Gateway Latency...${NC}"
PING=$(ping -c 3 gateway.discord.gg 2>/dev/null | tail -1 | awk -F '/' '{print $5}')
if [ ! -z "$PING" ]; then
    echo -e "   Gateway Ping: ${GREEN}${PING}ms${NC}"
else
    echo -e "   Gateway Ping: ${YELLOW}Failed${NC}"
fi

echo -e "${GREEN}🎉 DONE! Bot is running.${NC}"
echo -e "   View logs: ${CYAN}pm2 logs $BOT_NAME${NC}"

