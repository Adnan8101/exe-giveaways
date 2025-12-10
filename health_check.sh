#!/bin/bash

# ==============================================================================
# 🔍 BOT HEALTH CHECK - Pre-flight Diagnostics
# ==============================================================================
# Run this script to diagnose why the bot is crashing
# ==============================================================================

set +e  # Don't exit on error - we want to see all issues

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║       Bot Health Check - Diagnostics                    ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

ISSUES=0

# ------------------------------------------------------------------------------
# 1. Check config.json exists
# ------------------------------------------------------------------------------
echo -e "${CYAN}[CHECK 1] Config File...${NC}"
if [ ! -f "config.json" ]; then
    echo -e "${RED}❌ config.json not found!${NC}"
    ISSUES=$((ISSUES + 1))
else
    echo -e "${GREEN}✓ config.json exists${NC}"
    
    # Check if token field exists (without showing the actual token)
    if grep -q '"token"' config.json; then
        TOKEN_LENGTH=$(grep '"token"' config.json | cut -d'"' -f4 | wc -c)
        if [ "$TOKEN_LENGTH" -lt 50 ]; then
            echo -e "${RED}❌ Discord token appears to be empty or invalid (too short)${NC}"
            ISSUES=$((ISSUES + 1))
        else
            echo -e "${GREEN}✓ Discord token present (${TOKEN_LENGTH} chars)${NC}"
        fi
    else
        echo -e "${RED}❌ No token field in config.json${NC}"
        ISSUES=$((ISSUES + 1))
    fi
fi
echo ""

# ------------------------------------------------------------------------------
# 2. Check Redis
# ------------------------------------------------------------------------------
echo -e "${CYAN}[CHECK 2] Redis Connection...${NC}"
if ! command -v redis-cli &> /dev/null; then
    echo -e "${RED}❌ redis-cli not installed${NC}"
    ISSUES=$((ISSUES + 1))
else
    # Get Redis config from config.json
    REDIS_HOST=$(grep -A 5 '"redis"' config.json | grep '"host"' | cut -d'"' -f4)
    REDIS_PORT=$(grep -A 5 '"redis"' config.json | grep '"port"' | cut -d':' -f2 | tr -d ' ,')
    
    if [ -z "$REDIS_HOST" ]; then
        REDIS_HOST="localhost"
    fi
    if [ -z "$REDIS_PORT" ]; then
        REDIS_PORT="6379"
    fi
    
    echo -e "   Testing: ${REDIS_HOST}:${REDIS_PORT}"
    
    if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Redis is running and responding${NC}"
    else
        echo -e "${RED}❌ Cannot connect to Redis at ${REDIS_HOST}:${REDIS_PORT}${NC}"
        echo -e "${YELLOW}   Try: sudo systemctl start redis-server${NC}"
        ISSUES=$((ISSUES + 1))
    fi
fi
echo ""

# ------------------------------------------------------------------------------
# 3. Check PostgreSQL
# ------------------------------------------------------------------------------
echo -e "${CYAN}[CHECK 3] PostgreSQL Connection...${NC}"
if ! command -v psql &> /dev/null; then
    echo -e "${YELLOW}⚠️  psql not installed (cannot test directly)${NC}"
else
    # Get Postgres config from config.json
    PG_HOST=$(grep -A 10 '"postgres"' config.json | grep '"host"' | cut -d'"' -f4)
    PG_PORT=$(grep -A 10 '"postgres"' config.json | grep '"port"' | cut -d':' -f2 | tr -d ' ,')
    PG_USER=$(grep -A 10 '"postgres"' config.json | grep '"user"' | cut -d'"' -f4)
    PG_DB=$(grep -A 10 '"postgres"' config.json | grep '"database"' | cut -d'"' -f4)
    
    if [ -z "$PG_HOST" ]; then
        PG_HOST="localhost"
    fi
    if [ -z "$PG_PORT" ]; then
        PG_PORT="5432"
    fi
    
    echo -e "   Testing: ${PG_USER}@${PG_HOST}:${PG_PORT}/${PG_DB}"
    
    # Try to connect (will prompt for password if needed)
    if PGPASSWORD="" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DB" -c "SELECT 1;" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PostgreSQL is running and accessible${NC}"
    else
        echo -e "${RED}❌ Cannot connect to PostgreSQL${NC}"
        echo -e "${YELLOW}   Check if PostgreSQL is running: sudo systemctl start postgresql${NC}"
        echo -e "${YELLOW}   Check credentials in config.json${NC}"
        ISSUES=$((ISSUES + 1))
    fi
fi
echo ""

# ------------------------------------------------------------------------------
# 4. Check if bot binary exists
# ------------------------------------------------------------------------------
echo -e "${CYAN}[CHECK 4] Bot Binary...${NC}"
if [ ! -f "discord-giveaway-bot" ]; then
    echo -e "${RED}❌ Bot binary not found${NC}"
    echo -e "${YELLOW}   Run: go build -ldflags=\"-s -w\" -o discord-giveaway-bot .${NC}"
    ISSUES=$((ISSUES + 1))
else
    SIZE=$(ls -lh discord-giveaway-bot | awk '{print $5}')
    echo -e "${GREEN}✓ Bot binary exists (${SIZE})${NC}"
fi
echo ""

# ------------------------------------------------------------------------------
# 5. Check PM2
# ------------------------------------------------------------------------------
echo -e "${CYAN}[CHECK 5] PM2...${NC}"
if ! command -v pm2 &> /dev/null; then
    echo -e "${RED}❌ PM2 not installed${NC}"
    echo -e "${YELLOW}   Run: npm install -g pm2${NC}"
    ISSUES=$((ISSUES + 1))
else
    PM2_VERSION=$(pm2 --version)
    echo -e "${GREEN}✓ PM2 is installed (v${PM2_VERSION})${NC}"
fi
echo ""

# ------------------------------------------------------------------------------
# Summary
# ------------------------------------------------------------------------------
echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
if [ $ISSUES -eq 0 ]; then
    echo -e "${GREEN}║  ✅ ALL CHECKS PASSED                                    ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${GREEN}The bot should start successfully.${NC}"
    echo -e "If it still crashes, check PM2 logs:"
    echo -e "${CYAN}  pm2 logs discord-giveaway-bot${NC}"
else
    echo -e "${RED}║  ❌ FOUND ${ISSUES} ISSUE(S)                                     ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${RED}Fix the issues above before starting the bot.${NC}"
fi
echo ""
