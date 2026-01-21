#!/usr/bin/env bash
# Version management helper script for space-cli
# Usage:
#   ./scripts/version.sh         - Show current version
#   ./scripts/version.sh get     - Show current version
#   ./scripts/version.sh patch   - Bump patch version (0.0.x)
#   ./scripts/version.sh minor   - Bump minor version (0.x.0)
#   ./scripts/version.sh major   - Bump major version (x.0.0)
#   ./scripts/version.sh set X.Y.Z - Set specific version

set -e

VERSION_FILE="VERSION"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Ensure VERSION file exists
if [ ! -f "$VERSION_FILE" ]; then
    echo "0.1.0" > "$VERSION_FILE"
    echo -e "${GREEN}✓${NC} Created VERSION file with initial version 0.1.0"
fi

# Read current version
CURRENT_VERSION=$(cat "$VERSION_FILE")

# Parse semantic version
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# Remove any leading zeros to avoid octal interpretation
MAJOR=$((10#$MAJOR))
MINOR=$((10#$MINOR))
PATCH=$((10#$PATCH))

# Function to display version
show_version() {
    echo -e "${BLUE}Current version:${NC} $CURRENT_VERSION"
}

# Function to validate version format
validate_version() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo -e "${RED}✗${NC} Invalid version format: $version"
        echo -e "  Expected format: ${YELLOW}MAJOR.MINOR.PATCH${NC} (e.g., 1.2.3)"
        exit 1
    fi
}

# Function to write new version
write_version() {
    local new_version=$1
    local bump_type=$2

    echo "$new_version" > "$VERSION_FILE"
    echo -e "${GREEN}✓${NC} Bumped $bump_type version: ${YELLOW}$CURRENT_VERSION${NC} → ${GREEN}$new_version${NC}"
    echo -e "  Run ${BLUE}make install${NC} to rebuild with new version"
}

# Parse command
COMMAND=${1:-get}

case "$COMMAND" in
    get|show)
        show_version
        ;;

    patch)
        PATCH=$((PATCH + 1))
        NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
        write_version "$NEW_VERSION" "patch"
        ;;

    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
        write_version "$NEW_VERSION" "minor"
        ;;

    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
        write_version "$NEW_VERSION" "major"
        ;;

    set)
        if [ -z "$2" ]; then
            echo -e "${RED}✗${NC} Missing version argument"
            echo -e "  Usage: ${BLUE}$0 set X.Y.Z${NC}"
            exit 1
        fi
        NEW_VERSION=$2
        validate_version "$NEW_VERSION"
        echo "$NEW_VERSION" > "$VERSION_FILE"
        echo -e "${GREEN}✓${NC} Set version: ${YELLOW}$CURRENT_VERSION${NC} → ${GREEN}$NEW_VERSION${NC}"
        echo -e "  Run ${BLUE}make install${NC} to rebuild with new version"
        ;;

    help|--help|-h)
        echo "Version management helper for space-cli"
        echo ""
        echo "Usage:"
        echo "  $0 [command]"
        echo ""
        echo "Commands:"
        echo "  get, show    Show current version (default)"
        echo "  patch        Bump patch version (0.0.x)"
        echo "  minor        Bump minor version (0.x.0)"
        echo "  major        Bump major version (x.0.0)"
        echo "  set X.Y.Z    Set specific version"
        echo "  help         Show this help message"
        echo ""
        echo "Examples:"
        echo "  $0              # Show current version"
        echo "  $0 patch        # Bump patch: 1.2.3 → 1.2.4"
        echo "  $0 minor        # Bump minor: 1.2.3 → 1.3.0"
        echo "  $0 major        # Bump major: 1.2.3 → 2.0.0"
        echo "  $0 set 2.0.0    # Set to 2.0.0"
        echo ""
        echo "Note: The pre-commit hook will automatically bump versions on commit:"
        echo "  [major] or [breaking] - bumps major version"
        echo "  [minor] or [feature]  - bumps minor version"
        echo "  [patch] or default    - bumps patch version"
        ;;

    *)
        echo -e "${RED}✗${NC} Unknown command: $COMMAND"
        echo -e "  Run ${BLUE}$0 help${NC} for usage information"
        exit 1
        ;;
esac
