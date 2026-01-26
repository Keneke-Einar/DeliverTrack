#!/bin/bash

# Migration script to transition from old structure to new layered architecture
# This script helps identify what needs to be updated

echo "==================================="
echo "DeliverTrack Architecture Migration"
echo "==================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if new structure exists
echo "Checking new layered structure..."
echo ""

services=("delivery" "auth" "tracking" "notification" "analytics")
layers=("domain" "app" "ports" "adapters")

for service in "${services[@]}"; do
    echo "Service: $service"
    for layer in "${layers[@]}"; do
        dir="internal/$service/$layer"
        if [ -d "$dir" ]; then
            echo -e "  ${GREEN}✓${NC} $layer/ exists"
        else
            echo -e "  ${RED}✗${NC} $layer/ missing"
        fi
    done
    echo ""
done

echo "==================================="
echo "Old Files to Review/Migrate:"
echo "==================================="
echo ""

# Check for old files that may need migration
old_files=(
    "internal/delivery/delivery.go"
    "internal/auth/auth.go"
    "internal/auth/handlers.go"
    "internal/tracking/tracking.go"
    "internal/notification/notification.go"
    "internal/analytics/analytics.go"
)

for file in "${old_files[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${YELLOW}→${NC} $file (consider removing after migration)"
    fi
done

echo ""
echo "==================================="
echo "Entry Points to Update:"
echo "==================================="
echo ""

entry_points=(
    "cmd/delivery/main.go"
    "cmd/tracking/main.go"
    "cmd/notification/main.go"
    "cmd/analytics/main.go"
)

for file in "${entry_points[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${YELLOW}→${NC} $file (update to use new layered structure)"
    fi
done

echo ""
echo "==================================="
echo "Next Steps:"
echo "==================================="
echo ""
echo "1. Review the new layered architecture in docs/LAYERED_ARCHITECTURE.md"
echo "2. Update entry points (cmd/) to wire dependencies properly"
echo "3. Update import paths in your code to use new structure"
echo "4. Run 'go mod tidy' to clean up dependencies"
echo "5. Update tests to use new structure"
echo "6. Remove old files after confirming everything works"
echo ""
echo "Example new import paths:"
echo "  github.com/Keneke-Einar/delivertrack/internal/delivery/domain"
echo "  github.com/Keneke-Einar/delivertrack/internal/delivery/app"
echo "  github.com/Keneke-Einar/delivertrack/internal/delivery/adapters"
echo ""
echo "Reference implementation: cmd/delivery/main_new.go"
echo ""
