#!/bin/bash

# Update Fleet YAML files from Homebrew casks
# This script fetches the latest Homebrew casks and generates Fleet-compatible YAML files

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
OUTPUT_DIR="fleet_yaml_files"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TIMESTAMP=$(date -u +"%Y-%m-%d %H:%M:%S UTC")

echo -e "${BLUE}🚀 Starting Fleet YAML update process...${NC}"
echo -e "${BLUE}📅 Timestamp: ${TIMESTAMP}${NC}"
echo -e "${BLUE}📁 Output directory: ${OUTPUT_DIR}${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed. Please install Go 1.21 or later.${NC}"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${BLUE}🔧 Go version: ${GO_VERSION}${NC}"

# Navigate to script directory
cd "$SCRIPT_DIR"

# Clean previous output
if [ -d "$OUTPUT_DIR" ]; then
    echo -e "${YELLOW}🧹 Cleaning previous output directory...${NC}"
    rm -rf "$OUTPUT_DIR"
fi

# Build the Go program
echo -e "${BLUE}🔨 Building cask2fleet program...${NC}"
if ! go build -o cask2fleet main.go; then
    echo -e "${RED}❌ Failed to build cask2fleet program${NC}"
    exit 1
fi

# Run the program
echo -e "${BLUE}🔄 Running cask2fleet to generate YAML files...${NC}"
if ! ./cask2fleet; then
    echo -e "${RED}❌ Failed to run cask2fleet program${NC}"
    exit 1
fi

# Check if files were generated
if [ ! -d "$OUTPUT_DIR" ] || [ -z "$(ls -A "$OUTPUT_DIR" 2>/dev/null)" ]; then
    echo -e "${RED}❌ No YAML files were generated${NC}"
    exit 1
fi

# Count generated files
FILE_COUNT=$(find "$OUTPUT_DIR" -name "*.yml" | wc -l)
echo -e "${GREEN}✅ Successfully generated ${FILE_COUNT} YAML files${NC}"

# Create a summary file with metadata
SUMMARY_FILE="$OUTPUT_DIR/UPDATE_METADATA.md"
cat > "$SUMMARY_FILE" << EOF
# Fleet YAML Files Update Metadata

## Last Update
- **Timestamp**: ${TIMESTAMP}
- **Total Files Generated**: ${FILE_COUNT}
- **Source**: Homebrew Casks API
- **Filter Criteria**: Non-deprecated casks with PKG file types

## Generation Details
- **Script**: cask2fleet (Go program)
- **Go Version**: ${GO_VERSION}
- **Output Directory**: ${OUTPUT_DIR}

## File Format
Each YAML file contains:
- \`url\`: Download URL for the PKG installer
- \`automatic_install\`: false
- \`self_service\`: false  
- \`categories\`: [] (empty array)

## Categories
Categories are currently limited to: Browsers, Communication, Developer tools, and Productivity.

## Configuration
This is a minimum version of each file. All configurable parameters can be seen at:
https://fleetdm.com/docs/rest-api/rest-api#parameters139

## Automation
This directory is automatically updated twice daily via GitHub Actions.
EOF

echo -e "${GREEN}📝 Created update metadata: ${SUMMARY_FILE}${NC}"

# Show some example files
echo -e "${BLUE}📋 Example generated files:${NC}"
ls -la "$OUTPUT_DIR"/*.yml | head -5

echo -e "${GREEN}🎉 Fleet YAML update completed successfully!${NC}"
echo -e "${BLUE}📊 Summary:${NC}"
echo -e "   • Generated ${FILE_COUNT} YAML files"
echo -e "   • Output directory: ${OUTPUT_DIR}"
echo -e "   • Timestamp: ${TIMESTAMP}" 