#!/bin/bash

# ==============================================================================
# 🚀 DISCORD BOT - ULTRA-FAST BUILD & RUN SCRIPT
# ==============================================================================
# Optimized for MAXIMUM build speed and <3µs detection + <100ms execution
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
echo -e "${CYAN}║   🚀 Discord Bot - ULTRA-FAST Build & Run               ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

# ------------------------------------------------------------------------------
# 1. OS DETECTION & PRE-CHECKS
# ------------------------------------------------------------------------------
OS="$(uname -s)"
echo -e "${BLUE}[STEP 1] Checking Environment (${OS})...${NC}"

IS_LINUX=false

if [ "$OS" = "Linux" ]; then
    IS_LINUX=true
elif [ "$OS" = "Darwin" ]; then
    echo -e "${YELLOW}⚠️  Running on macOS. Some optimizations may differ.${NC}"
else
    echo -e "${RED}❌ Unsupported OS: $OS${NC}"
    exit 1
fi

# ------------------------------------------------------------------------------
# 2. SYSTEM OPTIMIZATION (Optional - No sudo required)
# ------------------------------------------------------------------------------
echo -e "${BLUE}[STEP 2] Checking System Optimizations...${NC}"

# Check if BBR is enabled (informational only)
if [ "$IS_LINUX" = true ]; then
    CC=$(sysctl -n net.ipv4.tcp_congestion_control 2>/dev/null || echo "unknown")
    if [ "$CC" = "bbr" ]; then
        echo -e "${GREEN}✓ TCP BBR already enabled${NC}"
    else
        echo -e "${YELLOW}⚠️  TCP BBR not enabled (current: $CC)${NC}"
        echo -e "${YELLOW}   For best performance, run: sudo sysctl -w net.ipv4.tcp_congestion_control=bbr${NC}"
    fi
fi

# ------------------------------------------------------------------------------
# 3. DEPENDENCY CHECK (No Auto-Install)
# ------------------------------------------------------------------------------
echo -e "${BLUE}[STEP 3] Checking Dependencies...${NC}"

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go not found. Please install Go 1.21+${NC}"
    exit 1
else
    GO_CURRENT=$(go version | awk '{print $3}' | sed 's/go//')
    echo -e "${GREEN}✓ Go is installed ($GO_CURRENT)${NC}"
fi

# Check Redis (informational only)
if ! command -v redis-cli &> /dev/null; then
    echo -e "${YELLOW}⚠️  Redis not found. Please ensure Redis is installed and running.${NC}"
else
    if redis-cli ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Redis is running${NC}"
    else
        echo -e "${YELLOW}⚠️  Redis installed but not running${NC}"
    fi
fi

# Check PM2
if ! command -v pm2 &> /dev/null; then
    echo -e "${YELLOW}⚠️  PM2 not found. Install with: npm install -g pm2${NC}"
    echo -e "${YELLOW}   Continuing without PM2...${NC}"
    USE_PM2=false
else
    echo -e "${GREEN}✓ PM2 is installed${NC}"
    USE_PM2=true
fi

# ------------------------------------------------------------------------------
# 4. ULTRA-FAST BUILD
# ------------------------------------------------------------------------------
echo -e "${BLUE}[STEP 4] Building Bot (ULTRA-OPTIMIZED)...${NC}"

echo -e "${CYAN}→ Tidy modules...${NC}"
go mod tidy

echo -e "${CYAN}→ Compiling with maximum optimizations...${NC}"
echo -e "${YELLOW}   • Maximum inlining (-l=4)${NC}"
echo -e "${YELLOW}   • Stripped binary (-s -w)${NC}"
echo -e "${YELLOW}   • Zero allocations + lock-free hot paths${NC}"
echo -e "${YELLOW}   • Target: <3µs detection + <100ms ban execution${NC}"
echo ""

# Use parallel compilation
export GOMAXPROCS=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)

if CGO_ENABLED=0 go build -gcflags="-l=4" -ldflags="-s -w" -o "$BOT_NAME" .; then
    SIZE=$(ls -lh "$BOT_NAME" | awk '{print $5}')
    echo -e "${GREEN}✓ Build successful (Size: $SIZE)${NC}"
    echo -e "${GREEN}✓ Optimizations: Max inlining, stripped debug symbols${NC}"
else
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

# ------------------------------------------------------------------------------
# 5. DEPLOY & RUN
# ------------------------------------------------------------------------------
echo -e "${BLUE}[STEP 5] Starting Bot...${NC}"

# Create logs directory
mkdir -p logs

if [ "$USE_PM2" = true ]; then
    # PM2 deployment
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
    max_memory_restart: '3G',
    kill_timeout: 5000,
    error_file: './logs/error.log',
    out_file: './logs/out.log',
    log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
    merge_logs: true,
    env: {
      NODE_ENV: 'production',
      GOMAXPROCS: '$GOMAXPROCS'
    }
  }]
};
EOF
    fi

    if pm2 list | grep -q "$BOT_NAME"; then
        echo -e "${CYAN}→ Restarting existing process...${NC}"
        pm2 delete "$BOT_NAME"
        sleep 1
    fi

    echo -e "${CYAN}→ Starting with PM2...${NC}"
    pm2 start "$PM2_CONFIG"
    pm2 save > /dev/null 2>&1

    # Health check
    echo -e "${BLUE}[STEP 6] Verifying Bot Health...${NC}"
    echo -e "${CYAN}→ Waiting 3 seconds...${NC}"
    sleep 3

    if pm2 list | grep -q "$BOT_NAME.*online"; then
        echo -e "${GREEN}✓ Bot is running${NC}"
        pm2 status "$BOT_NAME"
        
        echo ""
        echo -e "${GREEN}🎉 DONE! Bot is running with ULTRA optimizations.${NC}"
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "   ${BLUE}Performance Targets:${NC}"
        echo -e "   • Detection:    ${GREEN}<3µs${NC}"
        echo -e "   • Execution:    ${GREEN}<100ms${NC}"
        echo -e ""
        echo -e "   ${BLUE}Useful Commands:${NC}"
        echo -e "   • View logs:    ${CYAN}pm2 logs $BOT_NAME${NC}"
        echo -e "   • Status:       ${CYAN}pm2 status${NC}"
        echo -e "   • Restart:      ${CYAN}pm2 restart $BOT_NAME${NC}"
        echo -e "   • Stop:         ${CYAN}pm2 stop $BOT_NAME${NC}"
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    else
        echo -e "${RED}❌ Bot failed to start!${NC}"
        echo -e "${CYAN}Recent error logs:${NC}"
        tail -n 20 logs/error.log 2>/dev/null || echo "No error logs found"
        exit 1
    fi
else
    # Direct execution (no PM2)
    echo -e "${CYAN}→ Starting bot directly (no PM2)...${NC}"
    echo -e "${YELLOW}   Press Ctrl+C to stop${NC}"
    echo ""
    ./"$BOT_NAME"
fi

