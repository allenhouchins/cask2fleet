#!/bin/bash

# Update Fleet YAML Generator
# This script builds and runs the generate_fleet_yaml application

echo "🚀 Building generate_fleet_yaml..."

# Build the application
go build -o generate_fleet_yaml main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo "🔄 Running generate_fleet_yaml..."
    
    # Run the application
    ./generate_fleet_yaml
    
    if [ $? -eq 0 ]; then
        echo "✅ generate_fleet_yaml completed successfully!"
        
        # Show summary
        echo "📊 Generated files:"
        echo "   - macOS files: $(ls fleet_yaml_files/macOS/*.yml 2>/dev/null | wc -l)"
        echo "   - Windows files: $(ls fleet_yaml_files/Windows/*.yml 2>/dev/null | wc -l)"
        echo "   - Total files: $(find fleet_yaml_files -name "*.yml" 2>/dev/null | wc -l)"
    else
        echo "❌ generate_fleet_yaml failed!"
        exit 1
    fi
else
    echo "❌ Build failed!"
    exit 1
fi 