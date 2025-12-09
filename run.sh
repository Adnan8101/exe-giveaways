#!/bin/bash

# Production-ready run script
# 1. Enables services (Redis)
# 2. Builds the bot
# 3. Starts/Restarts with PM2 for 24/7 uptime

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}   Discord Bot - Production Start        ${NC}"
echo -e "${GREEN}=========================================${NC}"

# 0. Install Dependencies
echo -e "\n${YELLOW}[0/5] Installing dependencies...${NC}"
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

# 1. Enable Services
echo -e "\n${YELLOW}[1/5] Enabling services...${NC}"
if command -v docker-compose &> /dev/null; then
    docker-compose up -d
    echo -e "${GREEN}✓ Redis service started${NC}"
else
    echo "⚠️ Docker Compose not found, skipping service start"
fi

# 2. Build Code
echo -e "\n${YELLOW}[2/5] Building bot...${NC}"
# Build to root directory as expected by ecosystem.config.js
if go build -ldflags="-s -w" -o discord-giveaway-bot .; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo "❌ Build failed"
    exit 1
fi

# 3. Start/Restart with PM2
echo -e "\n${YELLOW}[3/5] Managing PM2 process...${NC}"
if pm2 list | grep -q "discord-giveaway-bot"; then
    echo "Restarting existing process..."
    pm2 restart discord-giveaway-bot
else
    echo "Starting new process..."
    pm2 start ecosystem.config.js
fi

# 4. Keep Online (Save)
echo -e "\n${YELLOW}[4/5] Saving PM2 state...${NC}"
pm2 save
echo -e "${GREEN}✓ PM2 list saved${NC}"

echo -e "\n${GREEN}=========================================${NC}"
echo -e "${GREEN}   Bot is ONLINE 🟢 (Managed by PM2)     ${NC}"
echo -e "${GREEN}=========================================${NC}"
echo "View logs:    pm2 logs discord-giveaway-bot"
echo "Stop bot:     pm2 stop discord-giveaway-bot"
echo "Restart:      pm2 restart discord-giveaway-bot"

