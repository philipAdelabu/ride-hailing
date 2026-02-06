#!/bin/bash

# Stop All Services Script
# This script stops all running microservices

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# PID file directory
PID_DIR="./.pids"

echo -e "${YELLOW}=== Stopping All Ride Hailing Services ===${NC}"
echo ""

# Stop all services
stopped=0
failed=0

# Use nullglob to handle case when no .pid files exist
shopt -s nullglob

# Stop services using PID files (if they exist)
if [ -d "$PID_DIR" ]; then
    for pid_file in "$PID_DIR"/*.pid; do
        service_name=$(basename "$pid_file" .pid)
        pid=$(cat "$pid_file")

        if ps -p "$pid" > /dev/null 2>&1; then
            echo -e "${YELLOW}Stopping $service_name (PID: $pid)...${NC}"
            kill "$pid" 2>/dev/null || kill -9 "$pid" 2>/dev/null

            # Wait for process to stop
            sleep 1

            if ! ps -p "$pid" > /dev/null 2>&1; then
                echo -e "${GREEN}✓ $service_name stopped${NC}"
                ((stopped++))
            else
                echo -e "${RED}✗ Failed to stop $service_name${NC}"
                ((failed++))
            fi
        else
            echo -e "${YELLOW}⊘ $service_name was not running${NC}"
        fi

        rm -f "$pid_file"
    done

    # Clean up PID directory if empty
    if [ -z "$(ls -A "$PID_DIR" 2>/dev/null)" ]; then
        rmdir "$PID_DIR" 2>/dev/null || true
    fi
fi

# Also kill any "go run ./cmd/*" processes (including orphaned ones)
echo ""
echo -e "${YELLOW}Checking for running service processes...${NC}"

# Find all go run processes for our services
while IFS= read -r line; do
    if [ -n "$line" ]; then
        pid=$(echo "$line" | awk '{print $2}')
        service=$(echo "$line" | grep -oE "cmd/[^/]+" | cut -d'/' -f2 || echo "unknown")
        echo -e "${YELLOW}  Stopping $service (PID: $pid)...${NC}"
        kill "$pid" 2>/dev/null || kill -9 "$pid" 2>/dev/null || true
        ((stopped++))
    fi
done < <(ps aux | grep "go run ./cmd/" | grep -v grep)

# Also kill the actual compiled binaries spawned by "go run"
binary_pids=$(lsof -ti:8081,8082,8083,8084,8085,8086,8087,8088,8089,8090,8091,8092,8093 2>/dev/null || true)
if [ -n "$binary_pids" ]; then
    echo -e "${YELLOW}Killing processes on service ports...${NC}"
    for pid in $binary_pids; do
        echo -e "${YELLOW}  Killing PID $pid...${NC}"
        kill "$pid" 2>/dev/null || kill -9 "$pid" 2>/dev/null || true
    done
    ((stopped++))
fi

# Final check
remaining=$(ps aux | grep "go run ./cmd/" | grep -v grep | wc -l | tr -d ' ')
if [ "$remaining" -eq 0 ]; then
    echo -e "${GREEN}✓ All service processes stopped${NC}"
else
    echo -e "${YELLOW}⚠ Some processes may still be running${NC}"
fi

echo ""
if [ $stopped -gt 0 ]; then
    echo -e "${GREEN}Stopped $stopped service(s)${NC}"
fi
if [ $failed -gt 0 ]; then
    echo -e "${RED}Failed to stop $failed service(s)${NC}"
fi
echo ""
