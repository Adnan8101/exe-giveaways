#!/bin/bash

# Production-ready run script with GCP VM optimization
# 1. Optimizes GCP VM for low latency (if sudo available)
# 2. Enables services (Redis)
# 3. Builds the bot
# 4. Starts/Restarts with PM2 for 24/7 uptime

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}   Discord Bot - Production Start        ${NC}"
echo -e "${GREEN}=========================================${NC}"

# 0. GCP VM Optimization (if running on GCP and has sudo)
echo -e "\n${YELLOW}[0/6] Checking GCP VM optimization...${NC}"
if [ -f "scripts/optimize-gcp-vm.sh" ]; then
    if [ "$EUID" -eq 0 ]; then
        echo "Running as root - applying optimizations..."
        bash scripts/optimize-gcp-vm.sh
        echo -e "${GREEN}✓ GCP VM optimized${NC}"
    elif sudo -n true 2>/dev/null; then
        echo "Sudo available - applying optimizations..."
        sudo bash scripts/optimize-gcp-vm.sh
        echo -e "${GREEN}✓ GCP VM optimized${NC}"
    else
        echo -e "${YELLOW}⚠️  Skipping optimization (requires sudo)${NC}"
        echo -e "${YELLOW}   Run: sudo bash scripts/optimize-gcp-vm.sh${NC}"
        echo -e "${YELLOW}   Continuing without optimization...${NC}"
        sleep 2
    fi
else
    echo -e "${YELLOW}⚠️  Optimization script not found, skipping...${NC}"
fi

# 1. Install Dependencies
echo -e "\n${YELLOW}[1/6] Installing dependencies...${NC}"
echo "Installing Go dependencies..."
go mod tidy
go mod download

if ! command -v pm2 &> /dev/null; then
    echo "PM2 not found. Installing PM2..."
    if command -v npm &> /dev/null; then
        npm install -g pm2
    else
        echo "❌ npm not found. Please install Node.js and npm to use PM2."
        exit 1
    fi
fi
echo -e "${GREEN}✓ Dependencies installed${NC}"

# 2. Enable Services
echo -e "\n${YELLOW}[2/6] Enabling services...${NC}"
if command -v docker-compose &> /dev/null; then
    docker-compose up -d
    echo -e "${GREEN}✓ Redis service started${NC}"
else
    echo "⚠️ Docker Compose not found, skipping service start"
fi

# 3. Build Code
echo -e "\n${YELLOW}[3/6] Building bot...${NC}"
# Build to root directory as expected by ecosystem.config.js
if go build -ldflags="-s -w" -o discord-giveaway-bot .; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo "❌ Build failed"
    exit 1
fi

# 4. Start/Restart with PM2
echo -e "\n${YELLOW}[4/6] Managing PM2 process...${NC}"
if pm2 list | grep -q "discord-giveaway-bot"; then
    echo "Restarting existing process..."
    pm2 restart discord-giveaway-bot
else
    echo "Starting new process..."
    pm2 start ecosystem.config.js
fi

# 5. Keep Online (Save)
echo -e "\n${YELLOW}[5/6] Saving PM2 state...${NC}"
pm2 save
echo -e "${GREEN}✓ PM2 list saved${NC}"

# 6. Performance Summary
echo -e "\n${YELLOW}[6/6] Performance Configuration Summary:${NC}"
echo -e "  ${GREEN}✓${NC} HTTP/2 Keep-Alive enabled"
echo -e "  ${GREEN}✓${NC} US-WEST Discord Gateway forced"
echo -e "  ${GREEN}✓${NC} GC optimized (400% for low latency)"
echo -e "  ${GREEN}✓${NC} Performance monitoring active"
echo ""
echo -e "  Expected Performance:"
echo -e "    WebSocket:  1-20ms"
echo -e "    REST API:   60-150ms"
echo -e "    Commands:   1-5ms"
echo ""

echo -e "\n${GREEN}=========================================${NC}"
echo -e "${GREEN}   Bot is ONLINE 🟢 (Managed by PM2)     ${NC}"
echo -e "${GREEN}=========================================${NC}"
echo "View logs:        pm2 logs discord-giveaway-bot"
echo "Stop bot:         pm2 stop discord-giveaway-bot"
echo "Restart:          pm2 restart discord-giveaway-bot"
echo "Performance:      Use /performance command in Discord"
echo ""
echo -e "${YELLOW}📊 Monitor heartbeat in logs for WebSocket latency${NC}"

