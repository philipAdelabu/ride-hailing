#!/bin/bash

# Database Seed Script
# This script seeds the database with sample data for local development

set -e

# Default database connection parameters
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-ridehailing}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== Ride Hailing Database Seeding ===${NC}"
echo ""

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo -e "${RED}Error: psql is not installed${NC}"
    echo "Please install PostgreSQL client tools"
    exit 1
fi

# Confirm before seeding
echo -e "${YELLOW}This will clear existing data and seed the database with sample data.${NC}"
echo "Database: $DB_NAME"
echo "Host: $DB_HOST:$DB_PORT"
echo ""
read -p "Are you sure you want to continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
SEED_FILE="$SCRIPT_DIR/seed-database.sql"

# Check if seed file exists
if [ ! -f "$SEED_FILE" ]; then
    echo -e "${RED}Error: Seed file not found at $SEED_FILE${NC}"
    exit 1
fi

echo ""
echo -e "${YELLOW}Seeding database...${NC}"

# Run the seed file
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$SEED_FILE"

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ Database seeded successfully!${NC}"
    echo ""
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
    echo ""
    echo -e "${RED}✗ Database seeding failed${NC}"
    exit 1
fi
