#!/bin/bash

# Database Seed Script for Docker
# This script seeds the database running in Docker with sample data

set -e

# Default parameters
CONTAINER_NAME="${CONTAINER_NAME:-ridehailing-postgres-dev}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-ridehailing}"
SEED_TYPE="${SEED_TYPE:-light}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== Ride Hailing Database Seeding (Docker) ===${NC}"
echo ""

# Check if Docker container is running
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo -e "${RED}Error: Docker container '${CONTAINER_NAME}' is not running${NC}"
    echo "Please start the container first with: make dev-infra"
    exit 1
fi

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Determine which seed file to use
case "$SEED_TYPE" in
    light)
        SEED_FILE="$SCRIPT_DIR/seed-database.sql"
        SEED_DESC="Light seed (11 users, 9 rides)"
        ;;
    medium)
        SEED_FILE="$SCRIPT_DIR/seed-medium.sql"
        SEED_DESC="Medium seed (52 users, 200 rides)"
        ;;
    heavy)
        SEED_FILE="$SCRIPT_DIR/seed-heavy.sql"
        SEED_DESC="Heavy seed (710 users, 5000 rides)"
        ;;
    *)
        echo -e "${RED}Error: Invalid SEED_TYPE. Use 'light', 'medium', or 'heavy'${NC}"
        exit 1
        ;;
esac

# Check if seed file exists
if [ ! -f "$SEED_FILE" ]; then
    echo -e "${RED}Error: Seed file not found at $SEED_FILE${NC}"
    exit 1
fi

# Confirm before seeding
echo -e "${YELLOW}This will clear existing data and seed the database with sample data.${NC}"
echo "Seed type: $SEED_DESC"
echo "Container: $CONTAINER_NAME"
echo "Database: $DB_NAME"
echo ""
read -p "Are you sure you want to continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

echo ""
echo -e "${YELLOW}Seeding database...${NC}"

# Run the seed file via Docker
if docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" < "$SEED_FILE"; then
    echo ""
    echo -e "${GREEN}✓ Database seeded successfully!${NC}"
    echo ""

    if [ "$SEED_TYPE" = "light" ]; then
        echo "Sample users created:"
        echo "  Riders:"
        echo "    - alice@example.com (password: password123)"
        echo "    - bob@example.com (password: password123)"
        echo "    - carol@example.com (password: password123)"
        echo "  Drivers:"
        echo "    - driver1@example.com (password: password123)"
        echo "    - driver2@example.com (password: password123)"
        echo "  Admin:"
        echo "    - admin@example.com (password: password123)"
    else
        echo "Database seeded with $SEED_DESC"
        echo "Admin user: admin@example.com (password: password123)"
    fi
else
    echo ""
    echo -e "${RED}✗ Database seeding failed${NC}"
    exit 1
fi
