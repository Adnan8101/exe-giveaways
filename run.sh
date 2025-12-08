#!/bin/bash

# Kill existing bot instances
echo "Stopping existing bot instances..."
pkill -f discord-giveaway-bot || true

# Start Redis if not running
if ! pgrep -x "redis-server" > /dev/null; then
    echo "Starting Redis..."
    if command -v brew &> /dev/null; then
        brew services start redis
    else
        redis-server --daemonize yes
    fi
    # Wait for Redis to start
    sleep 2
else
    echo "Redis is already running."
fi

# Start PostgreSQL if not running
if ! pgrep -x "postgres" > /dev/null; then
    echo "Starting PostgreSQL..."
    if command -v brew &> /dev/null; then
        brew services start postgresql
    else
        pg_ctl -D /usr/local/var/postgres start || echo "Please start PostgreSQL manually"
    fi
    sleep 2
else
    echo "PostgreSQL is already running."
fi

# Build the bot
echo "Building bot..."
go build -ldflags="-s -w" -o discord-giveaway-bot main.go

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

# Run the bot
echo "Starting bot with Goo Power..."
export GOGC=200
export GOMEMLIMIT=512MiB
./discord-giveaway-bot
