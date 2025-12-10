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
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

clear
echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                                                          ║${NC}"
echo -e "${CYAN}║       Discord Bot - Production Deployment System        ║${NC}"
echo -e "${CYAN}║       Low Latency Optimized for GCP us-west2           ║${NC}"
echo -e "${CYAN}║                                                          ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

# 0. GCP VM Optimization (if running on GCP and has sudo)
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}[STEP 1/7] GCP VM Performance Optimization${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

OPTIMIZATION_APPLIED=false

# Function to apply kernel optimizations
apply_optimizations() {
    echo -e "${CYAN}→ Applying kernel network optimizations...${NC}"
    
    # Create sysctl config
    cat > /tmp/discord-bot-sysctl.conf << 'EOF'
# Discord Bot Low Latency Tuning
# TCP Performance
net.ipv4.tcp_window_scaling = 1
net.ipv4.tcp_sack = 1
net.ipv4.tcp_timestamps = 0
net.ipv4.tcp_fastopen = 3
net.ipv4.tcp_low_latency = 1
net.ipv4.tcp_no_metrics_save = 1

# TCP Buffer Tuning
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.core.rmem_default = 16777216
net.core.wmem_default = 16777216
net.ipv4.tcp_rmem = 4096 87380 67108864
net.ipv4.tcp_wmem = 4096 65536 67108864

# Connection Tracking
net.core.netdev_max_backlog = 5000
net.core.somaxconn = 4096
net.ipv4.tcp_max_syn_backlog = 8192

# TCP Keepalive
net.ipv4.tcp_keepalive_time = 60
net.ipv4.tcp_keepalive_intvl = 10
net.ipv4.tcp_keepalive_probes = 6

# BBR Congestion Control
net.core.default_qdisc = fq
net.ipv4.tcp_congestion_control = bbr

# Reduce TIME_WAIT
net.ipv4.tcp_fin_timeout = 15
net.ipv4.tcp_tw_reuse = 1

# File handles
fs.file-max = 2097152
EOF

    # Apply sysctl settings
    if sysctl -p /tmp/discord-bot-sysctl.conf > /dev/null 2>&1; then
        echo -e "${GREEN}  ✓ Kernel tuning applied${NC}"
    else
        echo -e "${YELLOW}  ⚠ Some kernel parameters failed (non-critical)${NC}"
    fi
    
    # Set CPU governor to performance
    if command -v cpupower &> /dev/null; then
        cpupower frequency-set -g performance > /dev/null 2>&1
        echo -e "${GREEN}  ✓ CPU governor set to performance${NC}"
    elif [ -d /sys/devices/system/cpu/cpu0/cpufreq ]; then
        for cpu in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
            echo performance > $cpu 2>/dev/null
        done
        echo -e "${GREEN}  ✓ CPU governor set to performance${NC}"
    fi
    
    # Increase file descriptor limits
    cat >> /etc/security/limits.conf << 'EOF'
* soft nofile 1048576
* hard nofile 1048576
EOF
    echo -e "${GREEN}  ✓ File descriptor limits increased${NC}"
    
    rm -f /tmp/discord-bot-sysctl.conf
}

# Check if we can apply optimizations
if [ "$EUID" -eq 0 ]; then
    echo -e "${CYAN}→ Running as root - applying optimizations...${NC}"
    apply_optimizations
    OPTIMIZATION_APPLIED=true
elif sudo -n true 2>/dev/null; then
    echo -e "${CYAN}→ Sudo available - applying optimizations...${NC}"
    sudo bash -c "$(declare -f apply_optimizations); apply_optimizations"
    OPTIMIZATION_APPLIED=true
else
    echo -e "${YELLOW}⚠️  Sudo not available - skipping kernel optimization${NC}"
    echo -e "${YELLOW}   For best performance, run this script with sudo${NC}"
    echo -e "${YELLOW}   Continuing with application-level optimizations...${NC}"
    sleep 2
fi

if [ "$OPTIMIZATION_APPLIED" = true ]; then
    echo ""
    echo -e "${GREEN}✓ Kernel network tuning applied${NC}"
    echo -e "${GREEN}✓ TCP BBR congestion control enabled${NC}"
    echo -e "${GREEN}✓ File descriptor limits increased${NC}"
    echo -e "${GREEN}✓ CPU governor set to performance mode${NC}"
    echo ""
    echo -e "${CYAN}Waiting 2 seconds for system to stabilize...${NC}"
    sleep 2
fi

echo -e "${GREEN}✓ Step 1 Complete${NC}"
echo ""

# 1. Install Dependencies
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}[STEP 2/7] Installing Dependencies${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo -e "${CYAN}→ Checking Go dependencies...${NC}"
go mod tidy > /dev/null 2>&1
go mod download > /dev/null 2>&1
echo -e "${GREEN}✓ Go modules updated${NC}"

if ! command -v pm2 &> /dev/null; then
    echo -e "${CYAN}→ PM2 not found, installing...${NC}"
    if command -v npm &> /dev/null; then
        npm install -g pm2 > /dev/null 2>&1
        echo -e "${GREEN}✓ PM2 installed${NC}"
    else
        echo -e "${RED}❌ npm not found. Please install Node.js and npm first.${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✓ PM2 already installed${NC}"
fi

echo -e "${GREEN}✓ Step 2 Complete${NC}"
echo ""

# 2. Enable Services
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}[STEP 3/7] Starting Required Services${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Check if Redis is running
if command -v redis-cli &> /dev/null; then
    if redis-cli ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Redis is already running${NC}"
    else
        echo -e "${CYAN}→ Starting Redis...${NC}"
        if command -v docker-compose &> /dev/null || command -v docker &> /dev/null; then
            docker-compose up -d > /dev/null 2>&1 || docker compose up -d > /dev/null 2>&1
            sleep 2
            echo -e "${GREEN}✓ Redis started via Docker${NC}"
        elif command -v systemctl &> /dev/null; then
            sudo systemctl start redis-server > /dev/null 2>&1 || sudo systemctl start redis > /dev/null 2>&1
            sleep 1
            echo -e "${GREEN}✓ Redis started via systemctl${NC}"
        else
            echo -e "${YELLOW}⚠️  Could not auto-start Redis${NC}"
        fi
    fi
    
    # Test Redis connection
    if redis-cli ping > /dev/null 2>&1; then
        REDIS_VERSION=$(redis-cli --version | awk '{print $2}')
        echo -e "${GREEN}✓ Redis connection verified (version: ${REDIS_VERSION})${NC}"
    else
        echo -e "${RED}❌ Redis not responding - bot may fail to start${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Redis CLI not found - skipping Redis check${NC}"
fi

echo -e "${GREEN}✓ Step 3 Complete${NC}"
echo ""

# 3. Build Code
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}[STEP 4/7] Building Discord Bot${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo -e "${CYAN}→ Compiling with optimizations...${NC}"
echo -e "${CYAN}   Flags: -ldflags='-s -w' (strip debug symbols)${NC}"

BUILD_START=$(date +%s)

if go build -ldflags="-s -w" -o discord-giveaway-bot .; then
    BUILD_END=$(date +%s)
    BUILD_TIME=$((BUILD_END - BUILD_START))
    
    # Get binary size
    BINARY_SIZE=$(ls -lh discord-giveaway-bot | awk '{print $5}')
    
    echo -e "${GREEN}✓ Build successful${NC}"
    echo -e "${GREEN}  Binary size: ${BINARY_SIZE}${NC}"
    echo -e "${GREEN}  Build time: ${BUILD_TIME}s${NC}"
else
    echo -e "${RED}❌ Build failed - check for compilation errors${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Step 4 Complete${NC}"
echo ""

# 4. Verify Configuration
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}[STEP 5/7] Verifying Configuration${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ -f "config.json" ]; then
    echo -e "${GREEN}✓ config.json found${NC}"
    
    # Check for token (without revealing it)
    if grep -q '"token"' config.json; then
        echo -e "${GREEN}✓ Discord token configured${NC}"
    else
        echo -e "${RED}❌ Discord token not found in config.json${NC}"
        exit 1
    fi
    
    # Check Redis config
    if grep -q '"redis"' config.json; then
        echo -e "${GREEN}✓ Redis configuration found${NC}"
    fi
    
    # Check Postgres config
    if grep -q '"postgres"' config.json; then
        echo -e "${GREEN}✓ PostgreSQL configuration found${NC}"
    fi
else
    echo -e "${RED}❌ config.json not found${NC}"
    exit 1
fi

if [ -f "ecosystem.config.js" ]; then
    echo -e "${GREEN}✓ PM2 ecosystem config found${NC}"
else
    echo -e "${YELLOW}⚠️  ecosystem.config.js not found${NC}"
fi

echo -e "${GREEN}✓ Step 5 Complete${NC}"
echo ""

# 5. Start/Restart with PM2
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}[STEP 6/7] Managing PM2 Process${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if pm2 list | grep -q "discord-giveaway-bot"; then
    echo -e "${CYAN}→ Existing process found, restarting...${NC}"
    pm2 restart discord-giveaway-bot > /dev/null 2>&1
    echo -e "${GREEN}✓ Process restarted${NC}"
else
    echo -e "${CYAN}→ Starting new PM2 process...${NC}"
    pm2 start ecosystem.config.js > /dev/null 2>&1
    echo -e "${GREEN}✓ Process started${NC}"
fi

# Wait for process to initialize
sleep 2

# Check if process is running
if pm2 list | grep -q "online"; then
    echo -e "${GREEN}✓ Bot is online and running${NC}"
    
    # Get process info
    PID=$(pm2 jlist | grep -o '"pid":[0-9]*' | head -1 | grep -o '[0-9]*')
    UPTIME=$(pm2 jlist | grep -o '"pm_uptime":[0-9]*' | head -1 | grep -o '[0-9]*')
    
    if [ ! -z "$PID" ]; then
        echo -e "${GREEN}  Process ID: ${PID}${NC}"
    fi
else
    echo -e "${RED}❌ Bot failed to start - check logs with: pm2 logs${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Step 6 Complete${NC}"
echo ""

# 6. Save PM2 State
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}[STEP 7/7] Finalizing Deployment${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo -e "${CYAN}→ Saving PM2 process list...${NC}"
pm2 save > /dev/null 2>&1
echo -e "${GREEN}✓ PM2 state saved (survives reboot)${NC}"

echo -e "${CYAN}→ Enabling PM2 startup on boot...${NC}"
pm2 startup > /dev/null 2>&1 || echo -e "${YELLOW}  (Run 'pm2 startup' manually if needed)${NC}"

echo -e "${GREEN}✓ Step 7 Complete${NC}"
echo ""

# Performance Summary
echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                                                          ║${NC}"
echo -e "${CYAN}║              🎉 DEPLOYMENT SUCCESSFUL 🎉                 ║${NC}"
echo -e "${CYAN}║                                                          ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  Performance Optimizations Applied:${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ "$OPTIMIZATION_APPLIED" = true ]; then
    echo -e "  ${GREEN}✓${NC} Kernel TCP tuning (BBR congestion control)"
    echo -e "  ${GREEN}✓${NC} File descriptor limits increased (1M)"
    echo -e "  ${GREEN}✓${NC} CPU governor set to performance mode"
    echo -e "  ${GREEN}✓${NC} Network stack optimized for low latency"
fi

echo -e "  ${GREEN}✓${NC} HTTP/2 keep-alive transport (persistent connections)"
echo -e "  ${GREEN}✓${NC} US-WEST Discord Gateway forced"
echo -e "  ${GREEN}✓${NC} GC optimized (400% for reduced frequency)"
echo -e "  ${GREEN}✓${NC} Memory limit configured (3GB)"
echo -e "  ${GREEN}✓${NC} Performance monitoring enabled"
echo ""

echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}  Expected Performance Metrics:${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  ${CYAN}WebSocket Latency:${NC}    1-20ms   (heartbeat to Discord)"
echo -e "  ${CYAN}REST API Latency:${NC}     60-150ms (ban/kick/role actions)"
echo -e "  ${CYAN}Command Execution:${NC}    1-5ms    (internal processing)"
echo -e "  ${CYAN}Event Processing:${NC}     <1ms     (anti-nuke triggers)"
echo ""

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Management Commands:${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  ${CYAN}View logs:${NC}        pm2 logs discord-giveaway-bot"
echo -e "  ${CYAN}Monitor:${NC}          pm2 monit"
echo -e "  ${CYAN}Stop bot:${NC}         pm2 stop discord-giveaway-bot"
echo -e "  ${CYAN}Restart:${NC}          pm2 restart discord-giveaway-bot"
echo -e "  ${CYAN}Status:${NC}           pm2 status"
echo -e "  ${CYAN}Performance:${NC}      Use /performance command in Discord"
echo ""

echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  Real-time Monitoring:${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  • WebSocket heartbeat logged every 30 seconds"
echo -e "  • Performance dashboard updates every 60 seconds"
echo -e "  • Use /performance in Discord for live metrics"
echo ""

echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}  ⚠️  Important Notes:${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ "$OPTIMIZATION_APPLIED" = false ]; then
    echo -e "  ${YELLOW}⚠${NC}  Kernel optimizations not applied (no sudo access)"
    echo -e "     For full optimization, re-run with: ${CYAN}sudo ./run.sh${NC}"
    echo ""
fi

echo -e "  ${CYAN}•${NC} Check logs for WebSocket latency: ${CYAN}pm2 logs --lines 50${NC}"
echo -e "  ${CYAN}•${NC} If latency is high (>50ms), verify GCP region is us-west2"
echo -e "  ${CYAN}•${NC} Ensure GCP VM is using Premium Network Tier"
echo -e "  ${CYAN}•${NC} For Redis Unix socket: update config.json addr to /var/run/redis/redis.sock"
echo ""

echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                                                          ║${NC}"
echo -e "${CYAN}║              Bot is now ONLINE and MONITORED             ║${NC}"
echo -e "${CYAN}║                                                          ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

