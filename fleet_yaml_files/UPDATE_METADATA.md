# Fleet YAML Files Update Metadata

## Last Update
- **Timestamp**: 2025-08-11 21:38:44 UTC
- **Total Files Generated**: 457
- **Source**: Homebrew Casks API
- **Filter Criteria**: Non-deprecated casks with PKG file types
- **GitHub Run ID**: [16892800698](https://github.com/allenhouchins/cask2fleet/actions/runs/16892800698)

## Generation Details
- **Script**: cask2fleet (Go program)
- **Go Version**: go version go1.21.13 linux/amd64
- **Output Directory**: fleet_yaml_files
- **Triggered by**: workflow_dispatch

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
