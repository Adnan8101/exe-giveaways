#!/bin/bash
###############################################################################
# DISCORD GATEWAY LATENCY DIAGNOSTIC TOOL
# Use this to troubleshoot high WebSocket latency
###############################################################################

echo "=========================================="
echo "🔍 Discord Gateway Latency Diagnostics"
echo "=========================================="
echo ""

# 1. Check public IP and location
echo "1️⃣  Public IP & Location:"
PUBLIC_IP=$(curl -s ifconfig.me)
echo "   IP: $PUBLIC_IP"
LOCATION=$(curl -s "http://ip-api.com/json/$PUBLIC_IP" | jq -r '.city, .country, .regionName' | tr '\n' ', ')
echo "   Location: $LOCATION"
echo ""

# 2. Test Discord gateway ping
echo "2️⃣  Discord Gateway Ping Test:"
echo "   Testing gateway.discord.gg..."
PING_OUTPUT=$(ping -c 10 gateway.discord.gg 2>/dev/null)
PING_AVG=$(echo "$PING_OUTPUT" | tail -1 | awk -F '/' '{print $5}')

if [ ! -z "$PING_AVG" ]; then
    echo "   Average: ${PING_AVG}ms"
    PING_VALUE=${PING_AVG%.*}
    
    if [ "$PING_VALUE" -lt 25 ]; then
        echo "   ✅ EXCELLENT - Singapore gateway routing"
    elif [ "$PING_VALUE" -lt 40 ]; then
        echo "   ✅ GOOD - Close to Singapore"
    elif [ "$PING_VALUE" -lt 80 ]; then
        echo "   ⚠️  MODERATE - Not optimal"
    else
        echo "   ❌ HIGH - Wrong region or routing issue"
    fi
else
    echo "   ❌ Ping failed"
fi
echo ""

# 3. Traceroute to Discord
echo "3️⃣  Route to Discord Gateway:"
echo "   First 8 hops:"
traceroute -m 8 -q 1 gateway.discord.gg 2>/dev/null | head -9
echo ""

# 4. Check BBR status
echo "4️⃣  TCP Congestion Control:"
BBR=$(sysctl net.ipv4.tcp_congestion_control | awk '{print $3}')
echo "   Algorithm: $BBR"
if [ "$BBR" == "bbr" ]; then
    echo "   ✅ BBR enabled (optimal)"
else
    echo "   ❌ BBR not enabled - run: echo 'net.ipv4.tcp_congestion_control=bbr' | sudo tee -a /etc/sysctl.conf && sudo sysctl -p"
fi
echo ""

# 5. Check CPU governor
echo "5️⃣  CPU Performance Mode:"
if [ -f /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor ]; then
    GOVERNOR=$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor)
    echo "   Governor: $GOVERNOR"
    if [ "$GOVERNOR" == "performance" ]; then
        echo "   ✅ Performance mode enabled"
    else
        echo "   ⚠️  Not in performance mode - may cause heartbeat delays"
    fi
else
    echo "   ⚠️  Cannot determine governor"
fi
echo ""

# 6. Check if bot is running
echo "6️⃣  Bot Status:"
if command -v pm2 &> /dev/null; then
    BOT_STATUS=$(pm2 jlist 2>/dev/null | jq -r '.[] | select(.name=="bot") | .pm2_env.status')
    if [ "$BOT_STATUS" == "online" ]; then
        echo "   ✅ Bot is running (PM2)"
        echo ""
        echo "   Recent WS latency from logs:"
        pm2 logs bot --lines 50 --nostream 2>/dev/null | grep -i "ws latency" | tail -3 || echo "   No latency data in logs yet"
    else
        echo "   ⚠️  Bot not running"
    fi
else
    echo "   ⚠️  PM2 not installed"
fi
echo ""

# 7. Recommendations
echo "=========================================="
echo "💡 RECOMMENDATIONS"
echo "=========================================="
echo ""

if [ ! -z "$PING_VALUE" ] && [ "$PING_VALUE" -gt 50 ]; then
    echo "⚠️  HIGH LATENCY DETECTED ($PING_AVG ms)"
    echo ""
    echo "Possible causes:"
    echo "1. VM not in Singapore region (asia-southeast1-b)"
    echo "2. Network routing issue"
    echo "3. Discord using different gateway cluster"
    echo ""
    echo "Actions to fix:"
    echo "• Verify VM region: gcloud compute instances list"
    echo "• Create new VM in asia-southeast1-b"
    echo "• Stop bot, delete any session files, restart"
    echo "• Wait 5-10 minutes for gateway routing to optimize"
else
    echo "✅ Latency is acceptable"
    echo ""
    echo "For further optimization:"
    echo "• Let bot run for 1 hour (gateway routing improves)"
    echo "• Monitor: pm2 logs bot | grep 'WS Latency'"
    echo "• Expected after warmup: 15-22ms"
fi
echo ""

echo "=========================================="
echo "📊 QUICK REFERENCE"
echo "=========================================="
echo ""
echo "Expected latency by region:"
echo "  • Singapore (asia-southeast1): 12-25ms"
echo "  • Hong Kong (asia-east2): 30-45ms"
echo "  • Taiwan (asia-east1): 35-50ms"
echo "  • Tokyo (asia-northeast1): 60-80ms"
echo "  • Mumbai (asia-south1): 70-90ms"
echo "  • US-West: 150-180ms"
echo "  • US-East: 200-250ms"
echo ""
