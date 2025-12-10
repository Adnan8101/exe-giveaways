#!/bin/bash
###############################################################################
# DEPLOY BOT TO SINGAPORE VM
# Run this script after VM setup is complete
###############################################################################

set -e

BOT_DIR="/opt/discord-bot"
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

echo "=========================================="
echo "🚀 Deploying Discord Bot to Singapore VM"
echo "=========================================="
echo ""

# Check if running on VM
if [ ! -d "$BOT_DIR" ]; then
    echo "❌ Error: $BOT_DIR not found"
    echo "   This script should run on the Singapore VM after setup"
    echo "   Run singapore-vm-setup.sh first"
    exit 1
fi

echo "📁 Deployment directory: $BOT_DIR"
echo ""

# 1. Stop existing bot
echo "1️⃣  Stopping existing bot..."
pm2 stop bot 2>/dev/null || echo "   No running bot to stop"
pm2 delete bot 2>/dev/null || echo "   No bot process to delete"
echo ""

# 2. Backup old files
echo "2️⃣  Backing up old deployment..."
if [ -f "$BOT_DIR/bot" ]; then
    BACKUP_DIR="$BOT_DIR/backups/backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    cp "$BOT_DIR/bot" "$BACKUP_DIR/" 2>/dev/null || true
    cp "$BOT_DIR/config.json" "$BACKUP_DIR/" 2>/dev/null || true
    echo "   ✅ Backed up to $BACKUP_DIR"
else
    echo "   No previous deployment found"
fi
echo ""

# 3. Copy source files
echo "3️⃣  Copying source files..."
rsync -av --exclude='build/' --exclude='bot' --exclude='.git/' \
    "$PROJECT_ROOT/" "$BOT_DIR/" > /dev/null
echo "   ✅ Source files copied"
echo ""

# 4. Build bot
echo "4️⃣  Building bot..."
cd "$BOT_DIR"
/usr/local/go/bin/go build -ldflags="-s -w" -o bot .
if [ $? -eq 0 ]; then
    echo "   ✅ Build successful"
    ls -lh bot
else
    echo "   ❌ Build failed"
    exit 1
fi
echo ""

# 5. Verify config
echo "5️⃣  Verifying configuration..."
if [ ! -f "$BOT_DIR/config.json" ]; then
    echo "   ❌ config.json not found!"
    echo "   Please create config.json with your bot token and database credentials"
    exit 1
fi

# Check if token exists in config
TOKEN=$(jq -r '.token' "$BOT_DIR/config.json" 2>/dev/null)
if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
    echo "   ❌ Bot token not found in config.json"
    exit 1
fi
echo "   ✅ Configuration valid"
echo ""

# 6. Test Redis connection
echo "6️⃣  Testing Redis connection..."
REDIS_ADDR=$(jq -r '.redis.addr' "$BOT_DIR/config.json" 2>/dev/null)
if redis-cli -h ${REDIS_ADDR%%:*} -p ${REDIS_ADDR##*:} ping > /dev/null 2>&1; then
    echo "   ✅ Redis connection OK"
else
    echo "   ⚠️  Redis connection failed - check config.json"
fi
echo ""

# 7. Create/update PM2 ecosystem config
echo "7️⃣  Updating PM2 configuration..."
cat > "$BOT_DIR/ecosystem.config.js" <<'EOF'
module.exports = {
  apps: [{
    name: 'bot',
    script: './bot',
    cwd: '/opt/discord-bot',
    instances: 1,
    autorestart: true,
    watch: false,
    max_memory_restart: '3G',
    env: {
      NODE_ENV: 'production'
    },
    error_file: '/opt/discord-bot/logs/error.log',
    out_file: '/opt/discord-bot/logs/output.log',
    log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
    merge_logs: true,
    min_uptime: '10s',
    max_restarts: 10,
    restart_delay: 4000
  }]
};
EOF
echo "   ✅ PM2 config updated"
echo ""

# 8. Start bot
echo "8️⃣  Starting bot with PM2..."
pm2 start ecosystem.config.js
pm2 save
echo "   ✅ Bot started"
echo ""

# 9. Show initial logs
echo "=========================================="
echo "📊 DEPLOYMENT COMPLETE"
echo "=========================================="
echo ""
echo "Bot status:"
pm2 status
echo ""
echo "Recent logs:"
sleep 3
pm2 logs bot --lines 20 --nostream
echo ""
echo "=========================================="
echo "📋 NEXT STEPS"
echo "=========================================="
echo ""
echo "• Monitor logs: pm2 logs bot"
echo "• Check latency: bash $BOT_DIR/scripts/check-latency.sh"
echo "• Restart bot: pm2 restart bot"
echo "• Stop bot: pm2 stop bot"
echo ""
echo "Expected WebSocket latency:"
echo "  • First 5 min: 30-50ms (gateway routing)"
echo "  • After 30 min: 20-30ms"
echo "  • After 1 hour: 15-22ms (optimal)"
echo ""
