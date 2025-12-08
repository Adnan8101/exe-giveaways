# Discord Bot Production Deployment

**Production-ready deployment script for 24/7 uptime**

## Quick Start

```bash
# On your server instance
cd ~/exe-giveaways
./run.sh
```

The script will automatically:
- ✅ Install Go 1.21 if missing
- ✅ Install Redis if missing  
- ✅ Install PostgreSQL if missing
- ✅ Build the bot
- ✅ Create systemd service
- ✅ Set up health monitoring
- ✅ Start everything

## Features

### 1. Auto-Installation
Detects and installs missing dependencies automatically

### 2. Systemd Service
Bot runs as a system service that:
- Starts on boot
- Restarts on crash (10s delay)
- Logs to `/var/log/discord-bot.log`
- Memory limited to 1GB

### 3. Health Monitoring
Runs every minute via cron to check:
- Redis connectivity
- PostgreSQL connectivity  
- Bot process status
- Auto-restarts failed services

### 4. 24/7 Operation
Services persist through:
- SSH disconnects
- Server reboots
- Process crashes

## Management Commands

```bash
# View real-time logs
sudo journalctl -u discord-bot -f

# View health check logs
tail -f /var/log/discord-bot-health.log

# Restart bot
sudo systemctl restart discord-bot

# Stop bot
sudo systemctl stop discord-bot

# Check status
sudo systemctl status discord-bot

# View Redis status
sudo systemctl status redis-server

# View PostgreSQL status
sudo systemctl status postgresql
```

## Service Architecture

```
┌─────────────────────────────────────┐
│         Systemd Manager             │
│  (Manages all services)             │
└────────┬────────────────────────────┘
         │
    ┌────┴────┬─────────┬──────────┐
    │         │         │          │
┌───▼───┐ ┌──▼──┐  ┌───▼────┐ ┌───▼────────┐
│ Redis │ │ DB  │  │  Bot   │ │   Cron     │
│       │ │     │  │        │ │ (Health)   │
└───────┘ └─────┘  └────────┘ └────────────┘
```

## Health Check Logic

Every minute, the health checker:
1. Pings Redis → restart if down
2. Pings PostgreSQL → restart if down
3. Checks bot process → restart if down
4. Verifies bot binary → rebuild if missing

## Files Created

- `/etc/systemd/system/discord-bot.service` - Systemd service file
- `health_check.sh` - Health monitoring script
- `/var/log/discord-bot.log` - Bot output log
- `/var/log/discord-bot-error.log` - Bot error log
- `/var/log/discord-bot-health.log` - Health check log

## First Time Setup

After running `./run.sh`, verify everything is running:

```bash
# Check all services
sudo systemctl status discord-bot redis-server postgresql

# Follow live logs
sudo journalctl -u discord-bot -f
```

## Troubleshooting

### Bot won't start
```bash
# Check logs
sudo journalctl -u discord-bot -n 50

# Check config
cat config.json

# Rebuild manually
go build -ldflags="-s -w" -o bot .
sudo systemctl restart discord-bot
```

### Redis connection failed
```bash
# Check Redis
redis-cli ping

# Restart Redis
sudo systemctl restart redis-server
```

### Database connection failed
```bash
# Check PostgreSQL
pg_isready

# Restart PostgreSQL
sudo systemctl restart postgresql
```

## Production Checklist

- ✅ Firewall configured (if needed)
- ✅ Config.json has correct credentials
- ✅ Database schema initialized
- ✅ Bot token is valid
- ✅ Services auto-start on boot
- ✅ Health monitoring active
- ✅ Logs rotating (systemd handles this)

## Performance Optimizations Active

The deployed bot includes all optimizations from the performance upgrade:
- Multi-layer caching (L1/L2)
- Prepared statements with auto-recovery
- Lock-free concurrent maps
- Object pooling
- 100-connection database pool
- Event handler consolidation

**Expected performance**: Sub-50ms command response times under load.
