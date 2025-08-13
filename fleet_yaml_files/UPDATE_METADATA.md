# Fleet YAML Files Update Metadata

## Last Update
- **Timestamp**: 2025-08-13 01:20:56 UTC
- **Total Files Generated**: 398
- **Source**: Homebrew Casks API
- **Filter Criteria**: Non-deprecated casks with PKG file types
- **GitHub Run ID**: [16924938511](https://github.com/allenhouchins/cask2fleet/actions/runs/16924938511)

## Generation Details
- **Script**: cask2fleet (Go program)
- **Go Version**: go version go1.24.5 linux/amd64
- **Output Directory**: fleet_yaml_files
- **Triggered by**: push

## File Format
Each YAML file contains:
- `url`: Download URL for the PKG installer
- `automatic_install`: false
- `self_service`: false  
- `categories`: [] (empty array)

## Categories
Categories are currently limited to: Browsers, Communication, Developer tools, and Productivity.

## Configuration
This is a minimum version of each file. All configurable parameters can be seen at:
https://fleetdm.com/docs/rest-api/rest-api#parameters139

## Automation
This directory is automatically updated twice daily via GitHub Actions.
