# Cask to Fleet Converter

This Go program converts Homebrew casks to Fleet-compatible YAML files, specifically targeting non-deprecated casks with PKG file types.

## Features

- Fetches Homebrew casks data from the official API
- Filters for non-deprecated casks with URLs and PKG file types
- Generates Fleet-compatible YAML files for each qualifying cask
- Creates a comprehensive summary of processed casks
- Handles package identifiers for proper uninstallation

## Requirements

- Go 1.21 or later
- Internet connection to fetch Homebrew casks data

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
2. Filter for qualifying casks (non-deprecated, has URL, PKG file type)
3. Generate individual YAML files for each cask
4. Create a summary document

## Output

The program creates a `fleet_yaml_files/` directory containing:
- Individual YAML files for each qualifying cask
- A `SUMMARY.md` file with details about all processed casks

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

The program identifies PKG files using multiple methods:
- File extension (.pkg)
- Filename containing "pkg"
- Common installer patterns

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
Found 1234 total casks
Processing 45 casks that meet criteria...
Generated: example-app.yaml
Generated: another-app.yaml
...
Generated 45 Fleet YAML files in fleet_yaml_files/

Summary generated: SUMMARY.md
Conversion completed successfully!
```

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

This repository includes automated updates to keep the Fleet YAML files current with the latest Homebrew casks.

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