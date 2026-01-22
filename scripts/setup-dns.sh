#!/bin/bash

# Space CLI DNS Resolver Setup Script
# This script sets up macOS DNS resolver for *.space.local domains

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo ""
echo "========================================="
echo "  Space CLI DNS Resolver Setup"
echo "========================================="
echo ""
echo "This script will configure macOS to resolve *.space.local domains"
echo "to the Space CLI embedded DNS server running on localhost:5353."
echo ""
echo -e "${YELLOW}Note: This requires sudo access to modify /etc/resolver/${NC}"
echo ""

# Check if running on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo -e "${RED}Error: This script only supports macOS${NC}"
    exit 1
fi

# Create resolver directory
echo "Creating /etc/resolver directory..."
sudo mkdir -p /etc/resolver

# Create resolver configuration
echo "Creating DNS resolver configuration for space.local..."
echo "nameserver 127.0.0.1
port 5353" | sudo tee /etc/resolver/space.local > /dev/null

# Verify the file was created
if [[ -f /etc/resolver/space.local ]]; then
    echo -e "${GREEN}✓ DNS resolver configured successfully!${NC}"
    echo ""
    echo "Configuration:"
    cat /etc/resolver/space.local | sed 's/^/  /'
    echo ""
    echo -e "${GREEN}Setup complete!${NC}"
    echo ""
    echo "Now you can run 'space up' and access your services at:"
    echo "  • http://postgres.space.local:5432"
    echo "  • http://app.space.local:3000"
    echo "  • etc."
    echo ""
else
    echo -e "${RED}Error: Failed to create resolver configuration${NC}"
    exit 1
fi
