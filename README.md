# Cask to Fleet Converter

This Go program converts Homebrew casks and Installomator entries to Fleet-compatible YAML files, specifically targeting non-deprecated entries with PKG file types.

## Features

- Fetches Homebrew casks data from the official API
- Fetches Installomator script data from GitHub
- Filters for non-deprecated entries with URLs and PKG file types
- **Deduplicates entries** with Installomator taking priority over Homebrew casks
- **Alphabetically sorts** all entries regardless of source
- Generates Fleet-compatible YAML files for each qualifying entry
- Creates a comprehensive summary of processed entries
- Handles package identifiers for proper uninstallation
- **Precise PKG detection** to avoid false positives (ZIP, DMG, TAR files)
- **Go 1.24+ optimized** with improved performance and memory efficiency

## Requirements

- Go 1.24 or later
- Internet connection to fetch Homebrew casks data and Installomator script

## Recent Improvements

### Go 1.24 Upgrade (Latest)
- **Updated to Go 1.24** for latest features and performance improvements
- **Enhanced slice operations** using the `slices` package for better efficiency
- **Improved memory allocation** with pre-allocated slices
- **Updated GitHub Actions** to use Go 1.24

### PKG Detection Improvements (Latest)
- **Fixed false positives** that were incorrectly including ZIP, DMG, and TAR files
- **Precise detection logic** that only includes actual PKG installer files
- **Cleaned up 139 incorrectly generated files** from previous runs
- **Added safety filters** to prevent future false positives

### Example of Fixed Issues
- ❌ **Before**: `uninstallpkg_1.2.2.zip` was incorrectly identified as PKG
- ✅ **After**: Only legitimate PKG files like `zoom.pkg`, `microsoft-office.pkg` are included

## Installation

1. Clone or download this repository
2. Ensure you have Go installed on your system

## Usage

### Build and Run

```bash
# Build the program
go build -o cask2fleet main.go

# Run the program
./cask2fleet
```

### Run Directly with Go

```bash
go run main.go
```

The program will:
1. Fetch all Homebrew casks from the API
2. Fetch all Installomator entries from the script
3. Filter for qualifying entries (non-deprecated, has URL, PKG file type)
4. Deduplicate entries with Installomator taking priority
5. Sort all entries alphabetically
6. Generate individual YAML files for each entry
7. Create a summary document

## Output

The program creates a `fleet_yaml_files/` directory containing:
- Individual YAML files for each qualifying entry
- A `SUMMARY.md` file with details about all processed entries (including source information)
- An `UPDATE_METADATA.md` file with generation details and statistics

## Fleet YAML Structure

Each generated YAML file follows the Fleet software configuration format:

```yaml
apiVersion: v1
kind: Software
metadata:
  name: app-name-software
  labels:
    source: homebrew-cask
    type: pkg-installer
spec:
  name: App Name
  version: 1.0.0
  description: App description
  homepage: https://example.com
  source:
    type: url
    url: https://example.com/app.pkg
  install:
    type: pkg
    source: https://example.com/app.pkg
  uninstall:
    type: pkgutil
    identifiers: ["com.example.app"]
```

## Filtering Criteria

The program includes casks that meet ALL of the following criteria:
- **Not deprecated**: `deprecated: false`
- **Has URL**: Contains a valid download URL
- **PKG file type**: URL points to a PKG installer file

## Package Detection

The program uses precise PKG file detection to avoid false positives:

### Primary Detection Methods
- **File extension**: URLs ending with `.pkg`
- **Installer patterns**: URLs matching patterns like `installer.*\.pkg$`, `pkg.*installer$`

### Safety Filters
- **Excludes ZIP files**: URLs containing `.zip` are automatically excluded
- **Excludes DMG files**: URLs containing `.dmg` are automatically excluded  
- **Excludes TAR files**: URLs containing `.tar` are automatically excluded
- **Requires installer keywords**: For URLs containing "pkg" but not ending in `.pkg`, must also contain "installer", "setup", or "package"

### Examples
✅ **Valid PKG files**:
- `https://example.com/app.pkg`
- `https://example.com/installer.pkg`
- `https://example.com/app-setup.pkg`

❌ **Excluded files**:
- `https://example.com/uninstallpkg_1.2.2.zip` (ZIP file)
- `https://example.com/font-noto-sans-osage.zip` (ZIP file)
- `https://example.com/app.dmg` (DMG file)

## Error Handling

The program includes comprehensive error handling for:
- Network issues when fetching data
- Invalid cask data
- File I/O errors
- YAML generation issues

## Customization

You can modify the program to:
- Change filtering criteria
- Adjust YAML structure
- Modify output directory
- Add additional metadata fields

## Example Output

```
Fetching Homebrew casks...
Found 7562 total Homebrew casks
Fetching Installomator data...
Found 92 total Installomator entries
Processing 317 Homebrew casks and 92 Installomator entries that meet criteria...
Generated 398 unique entries after deduplication
Generated: 1password8.yml (from Installomator)
Generated: adoptopenjdk.yml
Generated: airmedia.yml
...
Generated 398 Fleet YAML files in fleet_yaml_files/

Summary generated: SUMMARY.md
Conversion completed successfully!
```

**Note**: The number of generated files varies based on the current Homebrew casks and Installomator entries available. The program now uses precise PKG detection, so only legitimate PKG installer files are included. Installomator entries take priority over Homebrew casks for duplicates.

## Building for Distribution

To build for different platforms:

```bash
# Build for macOS (current platform)
GOOS=darwin GOARCH=amd64 go build -o cask2fleet-macos-amd64 main.go

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o cask2fleet-linux-amd64 main.go

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o cask2fleet-windows-amd64.exe main.go
```

## 🤖 Automation

This repository includes automated updates to keep the Fleet YAML files current with the latest Homebrew casks and Installomator entries.

### GitHub Actions Workflow

The repository includes a GitHub Actions workflow (`.github/workflows/update-fleet-yaml.yml`) that:

- **Runs twice daily** at 6:00 AM and 6:00 PM UTC
- **Automatically commits** updated YAML files to the repository
- **Can be triggered manually** via the GitHub Actions tab
- **Runs on code changes** to the Go program

### Local Update Script

You can also run updates locally using the provided shell script:

```bash
# Make the script executable (first time only)
chmod +x update_fleet_yaml.sh

# Run the update
./update_fleet_yaml.sh
```

The script will:
- Build the Go program
- Generate updated YAML files
- Create metadata about the update
- Show a summary of generated files

### Update Metadata

Each update creates an `UPDATE_METADATA.md` file in the `fleet_yaml_files/` directory containing:
- Timestamp of the update
- Number of files generated
- Generation details
- Link to the GitHub Actions run

## Troubleshooting

- **Network errors**: Ensure you have internet access and the Homebrew API is reachable
- **Permission errors**: Make sure you have write permissions in the current directory
- **Compilation errors**: Verify you have Go 1.21+ installed

## Performance

The Go implementation provides:
- Fast execution compared to interpreted languages
- Efficient memory usage
- Single binary distribution
- Cross-platform compatibility

## License

This program is provided as-is for educational and development purposes. 