# Fleet YAML Files Update Metadata

## Last Update
- **Timestamp**: 2025-08-13 01:28:38 UTC
- **Total Files Generated**: 398
- **Sources**: 
  - Homebrew Casks API
  - Installomator Script
- **Filter Criteria**: Non-deprecated entries with PKG file types
- **Deduplication**: Installomator entries take priority over Homebrew casks
- **GitHub Run ID**: [16925042464](https://github.com/allenhouchins/cask2fleet/actions/runs/16925042464)

## Generation Details
- **Script**: cask2fleet (Go program)
- **Go Version**: go version go1.24.5 linux/amd64
- **Output Directory**: fleet_yaml_files
- **Triggered by**: workflow_dispatch
- **Processing**: Combined and deduplicated from multiple sources

## Sources

### Homebrew Casks
- **API Endpoint**: https://formulae.brew.sh/api/cask.json
- **Filter**: Non-deprecated casks with PKG file URLs

### Installomator
- **Source**: https://raw.githubusercontent.com/Installomator/Installomator/main/Installomator.sh
- **Filter**: Entries with PKG file URLs
- **Priority**: Takes precedence over Homebrew casks for duplicates

## Deduplication Strategy
- Installomator entries are processed first and take priority
- Homebrew casks are added only if they don't conflict with existing entries
- Conflicts are resolved by URL and identifier matching
- Final output is sorted alphabetically by identifier

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
