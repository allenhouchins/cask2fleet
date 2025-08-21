# Fleet YAML Files Update Metadata

## Last Update
- **Timestamp**: 2025-08-21 06:05:28 UTC
- **GitHub Actions Run**: https://github.com/allenhouchins/fleet-software-yaml-collection/actions/runs/17118563708
- **Total Files Generated**: 6439
- **macOS Files**: 397
- **Windows Files**: 6042

## Sources
- **Homebrew Casks API** (macOS PKG files)
- **Installomator Script** (macOS PKG files)
- **WinGet Repository** (Windows MSI/EXE files)

## Filter Criteria
- **macOS**: Only PKG installer files (excludes ZIP, DMG, TAR, MPKG)
- **Windows**: Only x64 MSI and EXE installer files
- **Architecture**: x64 only for Windows installers

## Output Directory
fleet_yaml_files (organized by platform)

## File Structure
All files follow consistent Fleet YAML structure with appropriate parameters:
- Standard fields: url, automatic_install, self_service, categories
- EXE files include: install_script, uninstall_script (with TODO placeholders)
- Helpful comments with Fleet API documentation links

## Automation
This directory is automatically updated via GitHub Actions workflow.
