#!/bin/bash

# Run All Services Script
# This script starts all microservices in the background

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Services to run
SERVICES=(
    "auth"
    "rides"
    "geo"
    "payments"
    "notifications"
    "realtime"
    "fraud"
    "analytics"
    "admin"
    "promos"
    "scheduler"
    "ml-eta"
    "mobile"
)

# PID file directory
PID_DIR="./.pids"
mkdir -p "$PID_DIR"

# Log directory
LOG_DIR="./logs"
mkdir -p "$LOG_DIR"

echo -e "${YELLOW}=== Starting All Ride Hailing Services ===${NC}"
echo ""

# Function to start a service
start_service() {
    local service=$1
    local pid_file="$PID_DIR/$service.pid"
    local log_file="$LOG_DIR/$service.log"

    # Check if service directory exists
    if [ ! -d "cmd/$service" ]; then
        echo -e "${YELLOW}⊘ Skipping $service (directory not found)${NC}"
        return
    fi

    # Check if already running
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            echo -e "${YELLOW}⊘ $service is already running (PID: $pid)${NC}"
            return
        fi
        rm -f "$pid_file"
    fi

    echo -e "${BLUE}Starting $service...${NC}"
    nohup go run ./cmd/$service > "$log_file" 2>&1 &
    local pid=$!
    echo $pid > "$pid_file"

    # Wait a bit and check if it's still running
    sleep 1
    if ps -p "$pid" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ $service started (PID: $pid)${NC}"
    else
        echo -e "${RED}✗ $service failed to start (check $log_file)${NC}"
        rm -f "$pid_file"
    fi
}

# Start all services
for service in "${SERVICES[@]}"; do
    start_service "$service"
done

echo ""
echo -e "${GREEN}=== All Services Started ===${NC}"
echo ""
echo "Service logs are in: $LOG_DIR"
echo "To view logs: tail -f $LOG_DIR/<service>.log"
echo ""
echo "To stop all services: ./scripts/stop-all-services.sh"
echo ""
echo "To check service status: ps aux | grep 'go run ./cmd'"
