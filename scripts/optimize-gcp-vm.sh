#!/bin/bash
# Production-Grade GCP VM Optimization Script
# For n2-highcpu-4 on us-west2 (Los Angeles)
# Optimized for Discord bot with extreme low latency requirements

set -e

echo "=================================================="
echo "🚀 GCP VM OPTIMIZATION FOR LOW LATENCY DISCORD BOT"
echo "=================================================="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "⚠️  Please run as root (sudo)"
    exit 1
fi

echo "📋 System Information:"
echo "   • OS: $(lsb_release -d | cut -f2)"
echo "   • Kernel: $(uname -r)"
echo "   • CPU Cores: $(nproc)"
echo "   • Memory: $(free -h | awk '/^Mem:/ {print $2}')"
echo ""

# ==========================================
# 1. KERNEL NETWORK TUNING
# ==========================================
echo "⚙️  Step 1: Applying kernel network optimizations..."

cat >> /etc/sysctl.d/99-discord-bot-optimization.conf << 'EOF'
# ===================================
# Discord Bot Low Latency Tuning
# Target: <20ms WS latency, <150ms REST
# ===================================

# TCP Performance
net.ipv4.tcp_window_scaling = 1
net.ipv4.tcp_sack = 1
net.ipv4.tcp_timestamps = 0
net.ipv4.tcp_fastopen = 3
net.ipv4.tcp_low_latency = 1
net.ipv4.tcp_no_metrics_save = 1

# Disable Nagle's algorithm (reduce latency)
net.ipv4.tcp_nodelay = 1

# TCP Buffer Tuning for High Throughput
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

# TCP Keepalive (for persistent connections)
net.ipv4.tcp_keepalive_time = 60
net.ipv4.tcp_keepalive_intvl = 10
net.ipv4.tcp_keepalive_probes = 6

# Congestion Control (BBR for better throughput)
net.core.default_qdisc = fq
net.ipv4.tcp_congestion_control = bbr

# Reduce TIME_WAIT sockets
net.ipv4.tcp_fin_timeout = 15
net.ipv4.tcp_tw_reuse = 1

# File handles
fs.file-max = 2097152

EOF

sysctl -p /etc/sysctl.d/99-discord-bot-optimization.conf
echo "✓ Kernel tuning applied"
echo ""

# ==========================================
# 2. REDIS UNIX SOCKET SETUP
# ==========================================
echo "⚙️  Step 2: Configuring Redis for Unix socket..."

if command -v redis-cli &> /dev/null; then
    # Backup Redis config
    if [ -f /etc/redis/redis.conf ]; then
        cp /etc/redis/redis.conf /etc/redis/redis.conf.backup
        
        # Configure Unix socket
        cat >> /etc/redis/redis.conf << 'EOF'

# Unix Socket for local connections (microsecond latency)
unixsocket /var/run/redis/redis.sock
unixsocketperm 770

# Performance optimizations
tcp-backlog 511
timeout 0
tcp-keepalive 300

# Disable RDB persistence for speed (data in memory only)
save ""

# AOF persistence (optional - comment out for max speed)
# appendonly yes
# appendfsync everysec

# Memory optimizations
maxmemory-policy allkeys-lru
maxmemory 2gb

EOF

        # Create socket directory
        mkdir -p /var/run/redis
        chown redis:redis /var/run/redis
        chmod 755 /var/run/redis
        
        # Add your user to redis group
        if [ -n "$SUDO_USER" ]; then
            usermod -a -G redis $SUDO_USER
            echo "✓ Added $SUDO_USER to redis group"
        fi
        
        systemctl restart redis
        echo "✓ Redis configured for Unix socket: /var/run/redis/redis.sock"
    else
        echo "⚠️  Redis config not found at /etc/redis/redis.conf"
    fi
else
    echo "ℹ️  Redis not installed - skipping Redis optimization"
fi
echo ""

# ==========================================
# 3. GCP NETWORK OPTIMIZATION
# ==========================================
echo "⚙️  Step 3: GCP Network Interface optimization..."

# Enable gVNIC if available
if lsmod | grep -q gve; then
    echo "✓ gVNIC driver detected"
    
    # Optimize gVNIC settings
    for iface in $(ip -o link show | awk -F': ' '{print $2}' | grep -v lo); do
        if ethtool -i $iface 2>/dev/null | grep -q gve; then
            # Disable unnecessary offloads for lower latency
            ethtool -K $iface tso off gso off 2>/dev/null || true
            # Increase ring buffer
            ethtool -G $iface rx 1024 tx 1024 2>/dev/null || true
            echo "✓ Optimized interface: $iface"
        fi
    done
else
    echo "ℹ️  gVNIC not detected - using standard virtio network"
fi
echo ""

# ==========================================
# 4. CPU GOVERNOR OPTIMIZATION
# ==========================================
echo "⚙️  Step 4: Setting CPU governor to performance mode..."

if command -v cpufreq-set &> /dev/null; then
    for cpu in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
        echo performance > $cpu
    done
    echo "✓ CPU governor set to performance"
else
    # Install cpufrequtils
    apt-get update -qq
    apt-get install -y cpufrequtils
    echo 'GOVERNOR="performance"' > /etc/default/cpufrequtils
    systemctl restart cpufrequtils
    echo "✓ CPU governor set to performance"
fi
echo ""

# ==========================================
# 5. DISABLE UNNECESSARY SERVICES
# ==========================================
echo "⚙️  Step 5: Disabling unnecessary services..."

services_to_disable=(
    "bluetooth.service"
    "cups.service"
    "avahi-daemon.service"
)

for service in "${services_to_disable[@]}"; do
    if systemctl is-enabled $service 2>/dev/null | grep -q enabled; then
        systemctl disable $service 2>/dev/null || true
        systemctl stop $service 2>/dev/null || true
        echo "✓ Disabled: $service"
    fi
done
echo ""

# ==========================================
# 6. ULIMIT CONFIGURATION
# ==========================================
echo "⚙️  Step 6: Setting ulimits for high performance..."

cat >> /etc/security/limits.conf << 'EOF'
# Discord Bot Performance Limits
*    soft    nofile    1048576
*    hard    nofile    1048576
*    soft    nproc     unlimited
*    hard    nproc     unlimited
root soft    nofile    1048576
root hard    nofile    1048576
EOF

echo "✓ Ulimits configured"
echo ""

# ==========================================
# 7. SYSTEMD SERVICE OPTIMIZATION
# ==========================================
echo "⚙️  Step 7: Creating optimized systemd service..."

cat > /etc/systemd/system/discord-bot.service << 'EOF'
[Unit]
Description=Discord Bot - Low Latency Production
After=network.target redis.service postgresql.service
Wants=redis.service postgresql.service

[Service]
Type=simple
User=YOUR_USER
WorkingDirectory=/path/to/discord-bot
ExecStart=/path/to/discord-bot/bot

# Performance optimizations
Nice=-10
CPUSchedulingPolicy=fifo
CPUSchedulingPriority=50
IOSchedulingClass=realtime
IOSchedulingPriority=0

# Resource limits
LimitNOFILE=1048576
LimitNPROC=unlimited

# Restart policy
Restart=always
RestartSec=5s

# Environment
Environment="GOMAXPROCS=4"
Environment="GODEBUG=gctrace=0"

[Install]
WantedBy=multi-user.target
EOF

echo "✓ Systemd service template created at /etc/systemd/system/discord-bot.service"
echo "  ⚠️  Remember to update User, WorkingDirectory, and ExecStart paths"
echo ""

# ==========================================
# FINAL CHECKS
# ==========================================
echo "=================================================="
echo "✅ OPTIMIZATION COMPLETE"
echo "=================================================="
echo ""
echo "📊 Current Settings:"
echo "   • TCP Congestion Control: $(sysctl -n net.ipv4.tcp_congestion_control)"
echo "   • CPU Governor: $(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null || echo 'N/A')"
echo "   • Max File Descriptors: $(ulimit -n)"
echo ""

if [ -f /var/run/redis/redis.sock ]; then
    echo "   • Redis Socket: /var/run/redis/redis.sock ✓"
else
    echo "   • Redis Socket: Not configured"
fi

echo ""
echo "🔧 NEXT STEPS:"
echo "=================================================="
echo "1. Update config.json to use Redis Unix socket:"
echo '   "addr": "/var/run/redis/redis.sock"'
echo ""
echo "2. Verify GCP VM is using:"
echo "   • Region: us-west2 (Los Angeles)"
echo "   • Network Tier: Premium"
echo "   • Network Driver: gVNIC"
echo ""
echo "3. Set Discord Gateway to US-WEST (already in code)"
echo ""
echo "4. Reboot for all changes to take effect:"
echo "   sudo reboot"
echo ""
echo "5. After reboot, verify WebSocket latency:"
echo "   Should see: <20ms heartbeat latency"
echo ""
echo "=================================================="
echo "🎯 Expected Performance:"
echo "   • WebSocket: 1-20ms"
echo "   • REST API: 60-150ms"
echo "   • Command execution: 1-5ms"
echo "=================================================="
