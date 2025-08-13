# Generate Fleet YAML

A Go application that automatically generates Fleet YAML configuration files from multiple package sources including Homebrew Casks, Installomator scripts, and WinGet manifests.

## Features

- **Multi-Source Support**: Generates Fleet YAML files from Homebrew Casks, Installomator scripts, and WinGet manifests
- **Cross-Platform**: Supports both macOS (PKG) and Windows (MSI/EXE) installers
- **Smart Filtering**: Only includes legitimate installer packages (PKG, MSI, EXE)
- **Organized Output**: Automatically organizes files by platform (macOS/Windows)
- **Consistent Structure**: All files follow the same professional YAML structure
- **EXE Script Support**: Automatically adds required install/uninstall script parameters for Windows EXE files
- **Production Ready**: Includes GitHub Actions workflow for automated updates

## Requirements

- **Go 1.24+** - Latest Go version with new features
- **Git** - Required for WinGet repository cloning (only during GitHub Actions execution)

## Installation

```bash
git clone https://github.com/yourusername/generate_fleet_yaml.git
cd generate_fleet_yaml
go build -o generate_fleet_yaml
```

## Usage

### Local Execution

```bash
./generate_fleet_yaml
```

This will:
1. Fetch Homebrew casks and filter for PKG files
2. Process Installomator scripts for additional PKG files
3. Clone and process WinGet manifests for MSI/EXE files
4. Generate organized YAML files in `fleet_yaml_files/`

### GitHub Actions

The project includes a GitHub Actions workflow (`.github/workflows/update-fleet-yaml.yml`) that:
- Runs automatically on schedule
- Builds and executes the application
- Commits updated YAML files
- Provides detailed metadata about the update

## Sources

### Homebrew Casks (macOS)
- **API**: `https://formulae.brew.sh/api/cask.json`
- **Filter**: Only includes URLs ending in `.pkg`
- **Output**: `fleet_yaml_files/macOS/`

### Installomator Scripts (macOS)
- **Source**: Installomator script parsing
- **Filter**: Only includes URLs ending in `.pkg`
- **Output**: `fleet_yaml_files/macOS/`

### WinGet Repository (Windows)
- **Source**: Local clone of [WinGet manifests repository](https://github.com/microsoft/winget-pkgs)
- **Filter**: Only includes x64 MSI and EXE installers
- **Output**: `fleet_yaml_files/Windows/`
- **Architecture**: x64 only
- **Installers**: MSI and EXE files

## WinGet Support (Production Ready - Local Repository)

The WinGet integration uses a local repository approach for maximum efficiency:

- **Local Repository**: Clones the WinGet manifests repository locally during execution
- **Full Repository Traversal**: Processes all manifests in the repository
- **Intelligent Caching**: Uses SHA256 hashes to avoid re-processing unchanged files
- **Efficient File System Traversal**: Direct file system access for maximum performance
- **x64 Architecture Filtering**: Only processes x64 Windows installers
- **Thread-Safe Operations**: Safe concurrent processing with proper synchronization

### Advanced WinGet Features

- **Local Repository Management**: Automatically clones and updates the WinGet repository
- **Git Integration**: Uses Git commands for repository management
- **Manifest Structure Support**: Handles the complex WinGet manifest structure (main package + installer files)
- **Performance Optimized**: Local file system access eliminates API rate limits

### Git Requirements

- **Git Installation**: Git must be available in the execution environment
- **GitHub Actions**: Git is pre-installed in GitHub Actions runners
- **Local Development**: Ensure Git is installed on your system

## Output

### Directory Structure

```
fleet_yaml_files/
├── macOS/          # macOS PKG files
│   ├── 1password8.yml
│   ├── zoom.yml
│   └── ...
└── Windows/        # Windows MSI/EXE files
    ├── zoom-zoom.yml
    ├── microsoft-teams.yml
    └── ...
```

### File Naming Convention
All generated files use lowercase names with hyphens instead of underscores:
- **Example**: `Yandex_Music.yml` → `yandex-music.yml`
- **Example**: `Zoom_Zoom.yml` → `zoom-zoom.yml`
- **Example**: `1password8.yml` → `1password8.yml` (already lowercase)

### File Structure
All YAML files follow the same consistent structure:

**macOS PKG files:**
```yaml
url: https://example.com/installer.pkg
automatic_install: false
self_service: false
categories: []

# Categories are currently limited to Browsers, Communication, Developer tools, and Productivity.
# This is a minimum version of this file. All configurable parameters can be seen at https://fleetdm.com/docs/rest-api/rest-api#parameters139
```

**Windows MSI files:**
```yaml
url: https://example.com/installer.msi
automatic_install: false
self_service: false
categories: []

# Categories are currently limited to Browsers, Communication, Developer tools, and Productivity.
# This is a minimum version of this file. All configurable parameters can be seen at https://fleetdm.com/docs/rest-api/rest-api#parameters139
```

**Windows EXE files:**
```yaml
url: https://example.com/installer.exe
automatic_install: false
self_service: false
categories: []
install_script: '# TODO: Add install script for this EXE'
uninstall_script: '# TODO: Add uninstall script for this EXE'

# Categories are currently limited to Browsers, Communication, Developer tools, and Productivity.
# This is a minimum version of this file. All configurable parameters can be seen at https://fleetdm.com/docs/rest-api/rest-api#parameters139
# Note: Any exe requires an install script and uninstall script to be defined
```

## Fleet YAML Structure

The generated YAML files follow the Fleet software configuration format:

- **url**: Direct download link to the installer
- **automatic_install**: Set to `false` by default
- **self_service**: Set to `false` by default
- **categories**: Empty array for manual categorization
- **install_script**: Required for EXE files (with TODO placeholder)
- **uninstall_script**: Required for EXE files (with TODO placeholder)

## Filtering Logic

### PKG Detection (macOS)
- **Includes**: URLs ending in `.pkg`
- **Excludes**: URLs containing `.zip`, `.dmg`, `.tar`, or `.mpkg`
- **Regex**: `\.pkg$`

### MSI Detection (Windows)
- **Includes**: URLs ending in `.msi`
- **Regex**: `\.msi$`

### EXE Detection (Windows)
- **Includes**: URLs ending in `.exe`
- **Regex**: `\.exe$`

### Windows Installer Detection
- **Includes**: Both MSI and EXE files
- **Architecture**: x64 only
- **Regex**: `\.(msi|exe)$`

## Performance Features

- **Go 1.24 Features**: Uses `slices.Compact` for efficient deduplication
- **Pre-allocated Slices**: Optimized memory usage with capacity hints
- **Local Repository**: Eliminates API rate limits for WinGet processing
- **Intelligent Caching**: Avoids re-processing unchanged files
- **Efficient File System Access**: Direct file operations for maximum speed

## GitHub Actions Workflow

The included workflow (`.github/workflows/update-fleet-yaml.yml`) provides:

- **Automated Execution**: Runs on schedule
- **Cache Management**: Efficient dependency caching
- **File Counting**: Detailed statistics about generated files
- **Metadata Generation**: Creates `UPDATE_METADATA.md` with update details
- **Git Operations**: Automatic commits with descriptive messages

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details. 