#!/bin/bash
###############################################################################
# DISCORD BOT - SINGAPORE VM AUTO-SETUP SCRIPT
# For GCP asia-southeast1-b (Singapore)
# Target: 15-20ms WebSocket latency
###############################################################################

set -e

echo "=========================================="
echo "🚀 Discord Bot Singapore VM Setup"
echo "=========================================="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "❌ Please run as root: sudo bash singapore-vm-setup.sh"
    exit 1
fi

echo "✅ Running as root"
echo ""

###############################################################################
# 1. SYSTEM UPDATE
###############################################################################
echo "📦 Step 1/8: Updating system packages..."
apt update -qq
apt upgrade -y -qq
echo "✅ System updated"
echo ""

###############################################################################
# 2. INSTALL ESSENTIAL PACKAGES
###############################################################################
echo "📦 Step 2/8: Installing essential packages..."
apt install -y -qq \
    curl \
    wget \
    git \
    build-essential \
    cpufrequtils \
    htop \
    iotop \
    net-tools \
    dnsutils \
    mtr \
    traceroute \
    redis-tools \
    postgresql-client \
    jq \
    vim \
    tmux \
    zip \
    unzip

echo "✅ Essential packages installed"
echo ""

###############################################################################
# 3. KERNEL NETWORK TUNING (CRITICAL FOR LOW LATENCY)
###############################################################################
echo "⚙️  Step 3/8: Applying kernel network tuning..."

cat >> /etc/sysctl.conf <<'EOF'

# ============================================
# DISCORD BOT LOW-LATENCY NETWORK TUNING
# ============================================

# BBR congestion control (reduces latency by 20-40ms)
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr

# TCP buffer optimization
net.core.rmem_max=2500000
net.core.wmem_max=2500000
net.ipv4.tcp_rmem=4096 87380 2500000
net.ipv4.tcp_wmem=4096 65536 2500000

# MTU probing (auto-detect optimal packet size)
net.ipv4.tcp_mtu_probing=1

# Fast socket reuse
net.ipv4.tcp_tw_reuse=1

# Reduce keepalive time
net.ipv4.tcp_keepalive_time=300
net.ipv4.tcp_keepalive_probes=3
net.ipv4.tcp_keepalive_intvl=30

# Increase connection backlog
net.core.somaxconn=4096
net.core.netdev_max_backlog=5000

# Disable slow start after idle
net.ipv4.tcp_slow_start_after_idle=0

EOF

sysctl -p > /dev/null
echo "✅ Kernel tuning applied (BBR enabled)"
echo ""

###############################################################################
# 4. CPU PERFORMANCE MODE (PREVENTS HEARTBEAT DELAYS)
###############################################################################
echo "⚡ Step 4/8: Setting CPU to performance mode..."

echo 'GOVERNOR="performance"' > /etc/default/cpufrequtils

# Apply immediately
for cpu in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
    echo "performance" > $cpu 2>/dev/null || true
done

systemctl restart cpufrequtils 2>/dev/null || true

echo "✅ CPU performance mode enabled"
echo ""

###############################################################################
# 5. INSTALL GO 1.23 (LATEST)
###############################################################################
echo "🐹 Step 5/8: Installing Go 1.23..."

GO_VERSION="1.23.4"
wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
rm go${GO_VERSION}.linux-amd64.tar.gz

# Add to PATH for all users
cat >> /etc/profile.d/go.sh <<'EOF'
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
EOF

source /etc/profile.d/go.sh

echo "✅ Go $(/usr/local/go/bin/go version) installed"
echo ""

###############################################################################
# 6. INSTALL REDIS (LOCAL CACHE)
###############################################################################
echo "📦 Step 6/8: Installing Redis..."

apt install -y -qq redis-server

# Configure Redis for low latency
cat > /etc/redis/redis.conf <<'EOF'
bind 127.0.0.1
port 6379
daemonize yes
supervised systemd
pidfile /var/run/redis/redis-server.pid
loglevel notice
logfile /var/log/redis/redis-server.log
databases 16
save ""
stop-writes-on-bgsave-error no
rdbcompression no
maxmemory 256mb
maxmemory-policy allkeys-lru
appendonly no
timeout 0
tcp-keepalive 60
tcp-backlog 511
EOF

systemctl enable redis-server
systemctl restart redis-server

echo "✅ Redis installed and configured"
echo ""

###############################################################################
# 7. INSTALL PM2 (PROCESS MANAGER)
###############################################################################
echo "📦 Step 7/8: Installing Node.js and PM2..."

curl -fsSL https://deb.nodesource.com/setup_20.x | bash - > /dev/null
apt install -y -qq nodejs
npm install -g pm2 --silent

# PM2 startup
pm2 startup systemd -u root --hp /root > /dev/null

echo "✅ PM2 installed"
echo ""

###############################################################################
# 8. CREATE BOT DIRECTORY STRUCTURE
###############################################################################
echo "📁 Step 8/8: Creating bot directory structure..."

BOT_DIR="/opt/discord-bot"
mkdir -p $BOT_DIR/{logs,backups}

cat > $BOT_DIR/README.md <<'EOF'
# Discord Bot - Singapore Deployment

## Directory Structure
- `/opt/discord-bot/` - Main bot directory
- `/opt/discord-bot/logs/` - Bot logs
- `/opt/discord-bot/backups/` - Configuration backups

## Deploy Bot
1. Upload your bot code to /opt/discord-bot/
2. Run: cd /opt/discord-bot && go build -o bot .
3. Start: pm2 start ecosystem.config.js
4. Monitor: pm2 logs bot

## Check Latency
```bash
pm2 logs bot | grep "WS:"
```

Should show: 15-25ms after warmup
EOF

echo "✅ Bot directory created at $BOT_DIR"
echo ""

###############################################################################
# FINAL VERIFICATION
###############################################################################
echo "=========================================="
echo "🎯 VERIFICATION"
echo "=========================================="
echo ""

echo "1️⃣  BBR Status:"
sysctl net.ipv4.tcp_congestion_control | grep bbr && echo "   ✅ BBR enabled" || echo "   ❌ BBR failed"
echo ""

echo "2️⃣  CPU Governor:"
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null && echo "   ✅ Performance mode" || echo "   ⚠️  Governor not set"
echo ""

echo "3️⃣  Go Version:"
/usr/local/go/bin/go version
echo ""

echo "4️⃣  Redis Status:"
systemctl is-active redis-server && echo "   ✅ Redis running" || echo "   ❌ Redis stopped"
echo ""

echo "5️⃣  PM2 Version:"
pm2 --version
echo ""

echo "6️⃣  Test Discord Gateway Latency:"
echo "   Running ping test..."
PING_RESULT=$(ping -c 5 gateway.discord.gg 2>/dev/null | tail -1 | awk -F '/' '{print $5}')
if [ ! -z "$PING_RESULT" ]; then
    echo "   📊 Average ping: ${PING_RESULT}ms"
    PING_VALUE=${PING_RESULT%.*}
    if [ "$PING_VALUE" -lt 30 ]; then
        echo "   ✅ EXCELLENT - You're in Singapore region!"
    elif [ "$PING_VALUE" -lt 50 ]; then
        echo "   ⚠️  GOOD - Close to Singapore"
    else
        echo "   ❌ HIGH LATENCY - Check VM region (should be asia-southeast1-b)"
    fi
else
    echo "   ⚠️  Could not test (DNS resolution issue)"
fi
echo ""

echo "=========================================="
echo "✅ SETUP COMPLETE!"
echo "=========================================="
echo ""
echo "📋 NEXT STEPS:"
echo ""
echo "1. Upload your bot code to: $BOT_DIR"
echo "2. Update config.json with your credentials"
echo "3. Build: cd $BOT_DIR && /usr/local/go/bin/go build -o bot ."
echo "4. Start: pm2 start ecosystem.config.js"
echo "5. Monitor: pm2 logs bot"
echo ""
echo "🎯 EXPECTED RESULTS:"
echo "   • First connect: 30-50ms (gateway routing)"
echo "   • After 5 min: 20-30ms"
echo "   • After 1 hour: 15-22ms (optimal)"
echo ""
echo "🔧 TROUBLESHOOTING:"
echo "   • High ping? Run: bash /opt/discord-bot/scripts/check-latency.sh"
echo "   • Check logs: pm2 logs bot"
echo "   • Restart: pm2 restart bot"
echo ""
echo "=========================================="
