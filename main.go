package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Cask represents a Homebrew cask structure
type Cask struct {
	Token      string                 `json:"token"`
	Name       []string               `json:"name"`
	Desc       string                 `json:"desc"`
	Homepage   string                 `json:"homepage"`
	URL        string                 `json:"url"`
	Version    string                 `json:"version"`
	Sha256     string                 `json:"sha256"`
	Deprecated bool                   `json:"deprecated"`
	DependsOn  map[string]interface{} `json:"depends_on"`
	Artifacts  []interface{}          `json:"artifacts"`
	Variations map[string]interface{} `json:"variations"`
}

// WinGetPackage represents a WinGet package structure
type WinGetPackage struct {
	PackageIdentifier string          `json:"PackageIdentifier"`
	PackageName       string          `json:"PackageName"`
	Publisher         string          `json:"Publisher"`
	Description       string          `json:"Description"`
	PackageURL        string          `json:"PackageURL"`
	License           string          `json:"License"`
	LicenseURL        string          `json:"LicenseURL"`
	Tags              []string        `json:"Tags"`
	Versions          []WinGetVersion `json:"Versions"`
}

// WinGetVersion represents a version of a WinGet package
type WinGetVersion struct {
	PackageVersion string            `json:"PackageVersion"`
	DefaultLocale  WinGetLocale      `json:"DefaultLocale"`
	Installers     []WinGetInstaller `json:"Installers"`
}

// WinGetLocale represents locale information for a WinGet package
type WinGetLocale struct {
	PackageLocale    string `json:"PackageLocale"`
	Publisher        string `json:"Publisher"`
	PackageName      string `json:"PackageName"`
	ShortDescription string `json:"ShortDescription"`
	Description      string `json:"Description"`
}

// WinGetInstaller represents an installer for a WinGet package
type WinGetInstaller struct {
	InstallerIdentifier string `json:"InstallerIdentifier"`
	InstallerURL        string `json:"InstallerUrl"`
	InstallerSha256     string `json:"InstallerSha256"`
	Architecture        string `json:"Architecture"`
	InstallerType       string `json:"InstallerType"`
	Scope               string `json:"Scope"`
}

// InstallomatorEntry represents an Installomator package entry
type InstallomatorEntry struct {
	Label       string
	Name        string
	Type        string
	PackageID   string
	DownloadURL string
	Source      string // "installomator" to identify source
}

// CombinedEntry represents a unified entry from either Homebrew or Installomator
type CombinedEntry struct {
	Identifier  string
	Name        string
	URL         string
	Source      string // "homebrew" or "installomator"
	Description string
	Version     string
	PackageID   string
}

// FleetSoftware represents the Fleet software configuration structure
type FleetSoftware struct {
	URL              string   `yaml:"url"`
	AutomaticInstall bool     `yaml:"automatic_install"`
	SelfService      bool     `yaml:"self_service"`
	Categories       []string `yaml:"categories"`
	InstallScript    string   `yaml:"install_script,omitempty"`
	UninstallScript  string   `yaml:"uninstall_script,omitempty"`
}

// CacheEntry represents a cached manifest entry
type CacheEntry struct {
	URL      string    `json:"url"`
	Hash     string    `json:"hash"`
	LastSeen time.Time `json:"last_seen"`
}

// PackageProcessor handles the conversion of packages to Fleet YAML
type PackageProcessor struct {
	outputDir     string
	cacheFile     string
	cache         map[string]CacheEntry
	cacheMutex    sync.RWMutex
	packagesMutex sync.Mutex
}

// addPackageThreadSafe adds a package to the slice in a thread-safe manner
func (pp *PackageProcessor) addPackageThreadSafe(packages *[]*WinGetPackage, pkg *WinGetPackage) {
	pp.packagesMutex.Lock()
	defer pp.packagesMutex.Unlock()
	*packages = append(*packages, pkg)
}

// NewPackageProcessor creates a new processor instance
func NewPackageProcessor(outputDir string) *PackageProcessor {
	// Initialize cache
	cacheFile := filepath.Join(outputDir, ".winget_cache.json")
	processor := &PackageProcessor{
		outputDir: outputDir,
		cacheFile: cacheFile,
		cache:     make(map[string]CacheEntry),
	}

	// Load existing cache
	processor.loadCache()

	return processor
}

// loadCache loads the cache from disk
func (pp *PackageProcessor) loadCache() {
	pp.cacheMutex.Lock()
	defer pp.cacheMutex.Unlock()

	data, err := os.ReadFile(pp.cacheFile)
	if err != nil {
		// Cache file doesn't exist, start with empty cache
		return
	}

	err = json.Unmarshal(data, &pp.cache)
	if err != nil {
		fmt.Printf("Warning: Could not load cache: %v\n", err)
		pp.cache = make(map[string]CacheEntry)
	}
}

// saveCache saves the cache to disk
func (pp *PackageProcessor) saveCache() {
	pp.cacheMutex.RLock()
	defer pp.cacheMutex.RUnlock()

	data, err := json.MarshalIndent(pp.cache, "", "  ")
	if err != nil {
		fmt.Printf("Warning: Could not marshal cache: %v\n", err)
		return
	}

	err = os.WriteFile(pp.cacheFile, data, 0644)
	if err != nil {
		fmt.Printf("Warning: Could not save cache: %v\n", err)
	}
}

// getCacheKey generates a cache key for a URL
func (pp *PackageProcessor) getCacheKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])
}

// isCached checks if a URL is cached and unchanged
func (pp *PackageProcessor) isCached(url string, content []byte) bool {
	pp.cacheMutex.RLock()
	defer pp.cacheMutex.RUnlock()

	key := pp.getCacheKey(url)
	entry, exists := pp.cache[key]
	if !exists {
		return false
	}

	// Check if content hash matches
	contentHash := sha256.Sum256(content)
	contentHashStr := hex.EncodeToString(contentHash[:])

	return entry.Hash == contentHashStr
}

// updateCache updates the cache with new content
func (pp *PackageProcessor) updateCache(url string, content []byte) {
	pp.cacheMutex.Lock()
	defer pp.cacheMutex.Unlock()

	key := pp.getCacheKey(url)
	contentHash := sha256.Sum256(content)
	contentHashStr := hex.EncodeToString(contentHash[:])

	pp.cache[key] = CacheEntry{
		URL:      url,
		Hash:     contentHashStr,
		LastSeen: time.Now(),
	}
}

// isPKGFile checks if the URL points to a PKG file
func (pp *PackageProcessor) isPKGFile(url string) bool {
	if url == "" {
		return false
	}

	url = strings.ToLower(url)

	// Check for .pkg extension (most reliable)
	if strings.HasSuffix(url, ".pkg") {
		return true
	}

	// Check for installer patterns that typically indicate PKG files
	pkgPatterns := []string{
		`installer.*\.pkg$`,
		`\.pkg$`,
		`pkg.*installer$`,
		`installer.*pkg$`,
	}

	for _, pattern := range pkgPatterns {
		matched, _ := regexp.MatchString(pattern, url)
		if matched {
			return true
		}
	}

	// Additional check: must end with .pkg or be a clear PKG installer
	// Avoid false positives like ZIP files containing "pkg" in the name
	if strings.Contains(url, "pkg") && !strings.Contains(url, ".zip") && !strings.Contains(url, ".dmg") && !strings.Contains(url, ".tar") {
		// Only include if it's likely a PKG installer, not a utility or other file type
		if strings.Contains(url, "installer") || strings.Contains(url, "setup") || strings.Contains(url, "package") {
			return true
		}
	}

	return false
}

// isMSIFile checks if the URL points to an MSI file
func (pp *PackageProcessor) isMSIFile(url string) bool {
	if url == "" {
		return false
	}

	url = strings.ToLower(url)

	// Check for .msi extension (most reliable)
	if strings.HasSuffix(url, ".msi") {
		return true
	}

	// Check for installer patterns that typically indicate MSI files
	msiPatterns := []string{
		`installer.*\.msi$`,
		`\.msi$`,
		`msi.*installer$`,
		`installer.*msi$`,
	}

	for _, pattern := range msiPatterns {
		matched, _ := regexp.MatchString(pattern, url)
		if matched {
			return true
		}
	}

	// Additional check: must end with .msi or be a clear MSI installer
	// Avoid false positives like ZIP files containing "msi" in the name
	if strings.Contains(url, "msi") && !strings.Contains(url, ".zip") && !strings.Contains(url, ".exe") && !strings.Contains(url, ".msix") {
		// Only include if it's likely an MSI installer, not a utility or other file type
		if strings.Contains(url, "installer") || strings.Contains(url, "setup") || strings.Contains(url, "package") {
			return true
		}
	}

	return false
}

// isEXEFile checks if the URL points to an EXE file
func (pp *PackageProcessor) isEXEFile(url string) bool {
	if url == "" {
		return false
	}

	url = strings.ToLower(url)

	// Check for .exe extension (most reliable)
	if strings.HasSuffix(url, ".exe") {
		return true
	}

	// Check for installer patterns that typically indicate EXE files
	exePatterns := []string{
		`installer.*\.exe$`,
		`\.exe$`,
		`exe.*installer$`,
		`installer.*exe$`,
	}

	for _, pattern := range exePatterns {
		matched, _ := regexp.MatchString(pattern, url)
		if matched {
			return true
		}
	}

	// Additional check: must end with .exe or be a clear EXE installer
	// Avoid false positives like ZIP files containing "exe" in the name
	if strings.Contains(url, "exe") && !strings.Contains(url, ".zip") && !strings.Contains(url, ".msi") && !strings.Contains(url, ".msix") {
		// Only include if it's likely an EXE installer, not a utility or other file type
		if strings.Contains(url, "installer") || strings.Contains(url, "setup") || strings.Contains(url, "package") {
			return true
		}
	}

	return false
}

// isWindowsInstaller checks if the URL points to a Windows installer (MSI or EXE)
func (pp *PackageProcessor) isWindowsInstaller(url string) bool {
	return pp.isMSIFile(url) || pp.isEXEFile(url)
}

// saveYAMLFile saves a Fleet YAML file to the appropriate directory
func (pp *PackageProcessor) saveYAMLFile(filename string, fleetConfig *FleetSoftware, source string) error {
	// Determine the appropriate directory based on the source
	var targetDir string
	switch source {
	case "Homebrew", "Installomator":
		targetDir = filepath.Join(pp.outputDir, "macOS")
	case "WinGet":
		targetDir = filepath.Join(pp.outputDir, "Windows")
	default:
		targetDir = pp.outputDir
	}

	// Create the target directory if it doesn't exist
	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating directory %s: %w", targetDir, err)
	}

	// Convert filename to lowercase and replace underscores with hyphens
	cleanFilename := strings.ToLower(filename)
	cleanFilename = strings.ReplaceAll(cleanFilename, "_", "-")

	// Create the full file path
	filePath := filepath.Join(targetDir, cleanFilename)

	// Marshal the configuration to YAML
	yamlData, err := yaml.Marshal(fleetConfig)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %w", err)
	}

	// Add comment at the bottom
	comment := "\n# Categories are currently limited to Browsers, Communication, Developer tools, and Productivity.\n# This is a minimum version of this file. All configurable parameters can be seen at https://fleetdm.com/docs/rest-api/rest-api#parameters139\n"

	// Add special comment for Windows EXE files
	if source == "WinGet" && pp.isEXEFile(fleetConfig.URL) {
		comment += "# Note: Any exe requires an install script and uninstall script to be defined\n"
	}

	finalData := append(yamlData, []byte(comment)...)

	// Write the file
	err = os.WriteFile(filePath, finalData, 0644)
	if err != nil {
		return fmt.Errorf("error writing file %s: %w", filePath, err)
	}

	fmt.Printf("Generated %s: %s\n", source, cleanFilename)
	return nil
}

// fetchWinGetPackages fetches all WinGet packages from the local repository
func (pp *PackageProcessor) fetchWinGetPackages() ([]*WinGetPackage, error) {
	fmt.Println("Fetching WinGet manifests from local repository...")

	// Check if the repository exists locally
	repoPath := "winget-pkgs"
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		// Repository doesn't exist, clone it
		fmt.Println("Cloning WinGet repository...")
		err = pp.cloneWinGetRepository(repoPath)
		if err != nil {
			return nil, fmt.Errorf("error cloning WinGet repository: %w", err)
		}
	} else {
		// Repository exists, pull latest changes
		fmt.Println("Updating WinGet repository...")
		err := pp.updateWinGetRepository(repoPath)
		if err != nil {
			fmt.Printf("Warning: Could not update repository: %v\n", err)
			// Continue with existing version
		}
	}

	var packages []*WinGetPackage

	// Traverse the manifests directory locally
	manifestsPath := filepath.Join(repoPath, "manifests")
	err := pp.traverseWinGetManifestsLocal(manifestsPath, &packages)
	if err != nil {
		return nil, fmt.Errorf("error traversing WinGet manifests: %w", err)
	}

	return packages, nil
}

// cloneWinGetRepository clones the WinGet repository
func (pp *PackageProcessor) cloneWinGetRepository(path string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", "https://github.com/microsoft/winget-pkgs.git", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// updateWinGetRepository pulls the latest changes from the repository
func (pp *PackageProcessor) updateWinGetRepository(path string) error {
	cmd := exec.Command("git", "-C", path, "pull", "origin", "master")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// traverseWinGetManifestsLocal recursively traverses the WinGet manifests directory
func (pp *PackageProcessor) traverseWinGetManifestsLocal(path string, packages *[]*WinGetPackage) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", path, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subPath := filepath.Join(path, entry.Name())
			err := pp.traverseWinGetManifestsLocal(subPath, packages)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(entry.Name(), ".yaml") &&
			!strings.Contains(entry.Name(), ".installer.") &&
			!strings.Contains(entry.Name(), ".locale.") {
			// Found a main package file, process it
			manifestPath := filepath.Join(path, entry.Name())
			err := pp.processWinGetPackage(manifestPath, packages)
			if err != nil {
				fmt.Printf("Warning: Error processing package %s: %v\n", manifestPath, err)
				continue
			}
		}
	}
	return nil
}

// processWinGetPackage processes a WinGet package by reading the main file and its installer file
func (pp *PackageProcessor) processWinGetPackage(manifestPath string, packages *[]*WinGetPackage) error {
	// Read the main package file
	yamlData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("error reading manifest file %s: %w", manifestPath, err)
	}

	// Parse the main package file
	var packageInfo struct {
		PackageIdentifier string `yaml:"PackageIdentifier"`
		PackageVersion    string `yaml:"PackageVersion"`
		DefaultLocale     string `yaml:"DefaultLocale"`
	}

	err = yaml.Unmarshal(yamlData, &packageInfo)
	if err != nil {
		return fmt.Errorf("error unmarshaling package info: %w", err)
	}

	// Look for the corresponding installer file
	dir := filepath.Dir(manifestPath)
	baseName := strings.TrimSuffix(filepath.Base(manifestPath), ".yaml")
	installerPath := filepath.Join(dir, baseName+".installer.yaml")

	// Check if installer file exists
	if _, err := os.Stat(installerPath); os.IsNotExist(err) {
		// No installer file found, skip this package
		return nil
	}

	// Read and parse the installer file
	installerData, err := os.ReadFile(installerPath)
	if err != nil {
		return fmt.Errorf("error reading installer file %s: %w", installerPath, err)
	}

	// Check cache for this installer file
	if pp.isCached(installerPath, installerData) {
		fmt.Printf("Cache hit for installer %s\n", installerPath)
		return nil
	}

	// Update cache
	pp.updateCache(installerPath, installerData)

	// Parse the installer file
	var installerInfo struct {
		PackageIdentifier string `yaml:"PackageIdentifier"`
		PackageVersion    string `yaml:"PackageVersion"`
		InstallerLocale   string `yaml:"InstallerLocale"`
		Installers        []struct {
			Architecture    string `yaml:"Architecture"`
			InstallerUrl    string `yaml:"InstallerUrl"`
			InstallerSha256 string `yaml:"InstallerSha256"`
			InstallerType   string `yaml:"InstallerType"`
			Scope           string `yaml:"Scope"`
		} `yaml:"Installers"`
	}

	err = yaml.Unmarshal(installerData, &installerInfo)
	if err != nil {
		return fmt.Errorf("error unmarshaling installer info: %w", err)
	}

	// Create WinGet package structure
	winGetPackage := &WinGetPackage{
		PackageIdentifier: installerInfo.PackageIdentifier,
		PackageName:       installerInfo.PackageIdentifier, // Use identifier as name for now
		Publisher:         "",                              // Would need to parse locale file for this
		Description:       "",                              // Would need to parse locale file for this
		Versions: []WinGetVersion{
			{
				PackageVersion: installerInfo.PackageVersion,
				DefaultLocale: WinGetLocale{
					PackageLocale:    installerInfo.InstallerLocale,
					Publisher:        "",
					PackageName:      installerInfo.PackageIdentifier,
					ShortDescription: "",
					Description:      "",
				},
				Installers: []WinGetInstaller{},
			},
		},
	}

	// Add installers
	for _, installer := range installerInfo.Installers {
		winGetInstaller := WinGetInstaller{
			InstallerIdentifier: fmt.Sprintf("%s-%s-%s", installerInfo.PackageIdentifier, installer.Architecture, installer.InstallerType),
			InstallerURL:        installer.InstallerUrl,
			InstallerSha256:     installer.InstallerSha256,
			Architecture:        installer.Architecture,
			InstallerType:       installer.InstallerType,
			Scope:               installer.Scope,
		}
		winGetPackage.Versions[0].Installers = append(winGetPackage.Versions[0].Installers, winGetInstaller)
	}

	// Check if this package has any Windows installers (MSI or EXE) for x64
	if pp.hasWindowsInstaller(winGetPackage) {
		pp.addPackageThreadSafe(packages, winGetPackage)
	}

	return nil
}

// hasWindowsInstaller checks if a WinGet package has any Windows installers (MSI or EXE) for x64
func (pp *PackageProcessor) hasWindowsInstaller(pkg *WinGetPackage) bool {
	if len(pkg.Versions) == 0 {
		return false
	}

	// Check the latest version
	latestVersion := pkg.Versions[0]

	for _, installer := range latestVersion.Installers {
		if installer.Architecture == "x64" && pp.isWindowsInstaller(installer.InstallerURL) {
			return true
		}
	}

	return false
}

// shouldIncludeWinGetPackage determines if a WinGet package should be included based on criteria
func (pp *PackageProcessor) shouldIncludeWinGetPackage(pkg *WinGetPackage) bool {
	// Must have versions
	if len(pkg.Versions) == 0 {
		return false
	}

	// Get the latest version
	latestVersion := pkg.Versions[0]

	// Must have installers
	if len(latestVersion.Installers) == 0 {
		return false
	}

	// Check if any installer is a Windows installer (MSI or EXE) for x64 architecture
	for _, installer := range latestVersion.Installers {
		if installer.Architecture == "x64" && pp.isWindowsInstaller(installer.InstallerURL) {
			return true
		}
	}

	return false
}

// generateFleetYAMLFromWinGet generates Fleet YAML configuration from WinGet package
func (pp *PackageProcessor) generateFleetYAMLFromWinGet(pkg *WinGetPackage) *FleetSoftware {
	// Find the first x64 Windows installer (MSI or EXE)
	var installerURL string
	var isEXE bool

	for _, version := range pkg.Versions {
		for _, installer := range version.Installers {
			if installer.Architecture == "x64" && pp.isWindowsInstaller(installer.InstallerURL) {
				installerURL = installer.InstallerURL
				isEXE = pp.isEXEFile(installer.InstallerURL)
				break
			}
		}
		if installerURL != "" {
			break
		}
	}

	// Create base configuration
	config := &FleetSoftware{
		URL:              installerURL,
		AutomaticInstall: false,
		SelfService:      false,
		Categories:       []string{},
	}

	// Add install_script and uninstall_script only for EXE files
	if isEXE {
		config.InstallScript = "# TODO: Add install script for this EXE"
		config.UninstallScript = "# TODO: Add uninstall script for this EXE"
	}

	return config
}

// saveYAMLFileFromWinGet saves the Fleet configuration to a YAML file for WinGet packages
func (pp *PackageProcessor) saveYAMLFileFromWinGet(pkg *WinGetPackage, fleetConfig *FleetSoftware) error {
	// Create safe filename
	safeIdentifier := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(pkg.PackageIdentifier, "_")
	filename := fmt.Sprintf("%s.yml", safeIdentifier)

	return pp.saveYAMLFile(filename, fleetConfig, "WinGet")
}

// processWinGetPackages processes all WinGet packages and generates YAML files
func (pp *PackageProcessor) processWinGetPackages() error {
	fmt.Println("Fetching WinGet packages...")
	packages, err := pp.fetchWinGetPackages()
	if err != nil {
		return fmt.Errorf("failed to fetch WinGet packages: %w", err)
	}

	if len(packages) == 0 {
		fmt.Println("No WinGet packages found or WinGet support not yet fully implemented")
		return nil
	}

	fmt.Printf("Found %d total WinGet packages\n", len(packages))

	// Filter and process packages
	var includedPackages []*WinGetPackage
	for _, pkg := range packages {
		if pp.shouldIncludeWinGetPackage(pkg) {
			includedPackages = append(includedPackages, pkg)
		}
	}

	fmt.Printf("Processing %d WinGet packages that meet criteria...\n", len(includedPackages))

	// Generate YAML files
	for _, pkg := range includedPackages {
		fleetConfig := pp.generateFleetYAMLFromWinGet(pkg)
		err := pp.saveYAMLFileFromWinGet(pkg, fleetConfig)
		if err != nil {
			fmt.Printf("Error processing WinGet package %s: %v\n", pkg.PackageIdentifier, err)
		}
	}

	fmt.Printf("\nGenerated %d WinGet Fleet YAML files in %s/\n", len(includedPackages), pp.outputDir)
	return nil
}

// fetchCasks fetches all Homebrew casks from the API
func (pp *PackageProcessor) fetchCasks() ([]*Cask, error) {
	resp, err := http.Get("https://formulae.brew.sh/api/cask.json")
	if err != nil {
		return nil, fmt.Errorf("error fetching casks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var casks []*Cask
	err = json.Unmarshal(body, &casks)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return casks, nil
}

// fetchInstallomatorData fetches the Installomator script and extracts package entries
func (pp *PackageProcessor) fetchInstallomatorData() ([]*InstallomatorEntry, error) {
	resp, err := http.Get("https://raw.githubusercontent.com/Installomator/Installomator/main/Installomator.sh")
	if err != nil {
		return nil, fmt.Errorf("error fetching Installomator script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Installomator API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading Installomator response body: %w", err)
	}

	return pp.parseInstallomatorScript(string(body))
}

// parseInstallomatorScript parses the Installomator shell script to extract package entries
func (pp *PackageProcessor) parseInstallomatorScript(scriptContent string) ([]*InstallomatorEntry, error) {
	var entries []*InstallomatorEntry

	// Split the script into lines
	lines := strings.Split(scriptContent, "\n")

	var currentEntry *InstallomatorEntry
	var inEntry bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for start of a new entry (label pattern: word followed by )
		if match := regexp.MustCompile(`^([a-zA-Z0-9_-]+)\)$`).FindStringSubmatch(line); match != nil {
			// Save previous entry if it exists and has a download URL
			if currentEntry != nil && currentEntry.DownloadURL != "" && pp.isPKGFile(currentEntry.DownloadURL) {
				entries = append(entries, currentEntry)
			}

			// Start new entry
			currentEntry = &InstallomatorEntry{
				Label:  match[1],
				Source: "installomator",
			}
			inEntry = true
			continue
		}

		// If we're in an entry, parse the fields
		if inEntry && currentEntry != nil {
			// Check for end of entry (;;)
			if strings.TrimSpace(line) == ";;" {
				inEntry = false
				continue
			}

			// Parse name field
			if strings.HasPrefix(line, "name=") {
				currentEntry.Name = strings.Trim(strings.TrimPrefix(line, "name="), `"`)
				continue
			}

			// Parse type field
			if strings.HasPrefix(line, "type=") {
				currentEntry.Type = strings.Trim(strings.TrimPrefix(line, "type="), `"`)
				continue
			}

			// Parse packageID field
			if strings.HasPrefix(line, "packageID=") {
				currentEntry.PackageID = strings.Trim(strings.TrimPrefix(line, "packageID="), `"`)
				continue
			}

			// Parse downloadURL field
			if strings.HasPrefix(line, "downloadURL=") {
				currentEntry.DownloadURL = strings.Trim(strings.TrimPrefix(line, "downloadURL="), `"`)
				continue
			}
		}
	}

	// Don't forget the last entry
	if currentEntry != nil && currentEntry.DownloadURL != "" && pp.isPKGFile(currentEntry.DownloadURL) {
		entries = append(entries, currentEntry)
	}

	return entries, nil
}

// shouldIncludeCask determines if a cask should be included based on criteria
func (pp *PackageProcessor) shouldIncludeCask(cask *Cask) bool {
	// Must not be deprecated
	if cask.Deprecated {
		return false
	}

	// Must have a URL
	if cask.URL == "" {
		return false
	}

	// Must be a PKG file
	if !pp.isPKGFile(cask.URL) {
		return false
	}

	return true
}

// shouldIncludeInstallomatorEntry determines if an Installomator entry should be included
func (pp *PackageProcessor) shouldIncludeInstallomatorEntry(entry *InstallomatorEntry) bool {
	// Must have a download URL
	if entry.DownloadURL == "" {
		return false
	}

	// Must be a PKG file
	if !pp.isPKGFile(entry.DownloadURL) {
		return false
	}

	// Must have a label (identifier)
	if entry.Label == "" {
		return false
	}

	return true
}

// generateFleetYAML generates Fleet-compatible YAML structure for a cask
func (pp *PackageProcessor) generateFleetYAML(cask *Cask) *FleetSoftware {
	// Create Fleet software configuration
	fleetConfig := &FleetSoftware{
		URL:              cask.URL,
		AutomaticInstall: false,
		SelfService:      false,
		Categories:       []string{},
	}

	return fleetConfig
}

// generateFleetYAMLFromInstallomator generates Fleet-compatible YAML structure for an Installomator entry
func (pp *PackageProcessor) generateFleetYAMLFromInstallomator(entry *InstallomatorEntry) *FleetSoftware {
	// Create Fleet software configuration
	fleetConfig := &FleetSoftware{
		URL:              entry.DownloadURL,
		AutomaticInstall: false,
		SelfService:      false,
		Categories:       []string{},
	}

	return fleetConfig
}

// saveYAMLFileFromInstallomator saves the Fleet configuration to a YAML file for Installomator entries
func (pp *PackageProcessor) saveYAMLFileFromInstallomator(entry *InstallomatorEntry, fleetConfig *FleetSoftware) error {
	// Create safe filename
	safeLabel := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(entry.Label, "_")
	filename := fmt.Sprintf("%s.yml", safeLabel)

	return pp.saveYAMLFile(filename, fleetConfig, "Installomator")
}

// combineAndDeduplicate combines Homebrew casks and Installomator entries, removing duplicates
// Installomator entries take priority over Homebrew casks for duplicates
func (pp *PackageProcessor) combineAndDeduplicate(casks []*Cask, installomatorEntries []*InstallomatorEntry) []*CombinedEntry {
	// Create a map to track seen URLs and identifiers
	seenURLs := make(map[string]bool)
	seenIdentifiers := make(map[string]bool)
	var combinedEntries []*CombinedEntry

	// First, add all Installomator entries (they have priority)
	for _, entry := range installomatorEntries {
		url := strings.ToLower(entry.DownloadURL)
		identifier := strings.ToLower(entry.Label)

		if !seenURLs[url] && !seenIdentifiers[identifier] {
			combinedEntries = append(combinedEntries, &CombinedEntry{
				Identifier:  entry.Label,
				Name:        entry.Name,
				URL:         entry.DownloadURL,
				Source:      "installomator",
				Description: entry.Name, // Use name as description
				Version:     "",         // Installomator doesn't always have version
				PackageID:   entry.PackageID,
			})
			seenURLs[url] = true
			seenIdentifiers[identifier] = true
		}
	}

	// Then add Homebrew casks that don't conflict
	for _, cask := range casks {
		url := strings.ToLower(cask.URL)
		identifier := strings.ToLower(cask.Token)

		if !seenURLs[url] && !seenIdentifiers[identifier] {
			name := cask.Token
			if len(cask.Name) > 0 {
				name = cask.Name[0]
			}

			combinedEntries = append(combinedEntries, &CombinedEntry{
				Identifier:  cask.Token,
				Name:        name,
				URL:         cask.URL,
				Source:      "homebrew",
				Description: cask.Desc,
				Version:     cask.Version,
				PackageID:   "", // Homebrew casks don't have packageID
			})
			seenURLs[url] = true
			seenIdentifiers[identifier] = true
		}
	}

	return combinedEntries
}

// processCasks is the main method to process all casks and generate YAML files
func (pp *PackageProcessor) processCasks() error {
	fmt.Println("Fetching Homebrew casks...")
	casks, err := pp.fetchCasks()
	if err != nil {
		return fmt.Errorf("failed to fetch casks: %w", err)
	}

	if len(casks) == 0 {
		return fmt.Errorf("no casks found")
	}

	fmt.Printf("Found %d total Homebrew casks\n", len(casks))

	fmt.Println("Fetching Installomator data...")
	installomatorEntries, err := pp.fetchInstallomatorData()
	if err != nil {
		return fmt.Errorf("failed to fetch Installomator data: %w", err)
	}

	fmt.Printf("Found %d total Installomator entries\n", len(installomatorEntries))

	// Filter Homebrew casks
	includedCasks := make([]*Cask, 0, len(casks)/4)
	for _, cask := range casks {
		if pp.shouldIncludeCask(cask) {
			includedCasks = append(includedCasks, cask)
		}
	}

	// Filter Installomator entries
	includedInstallomatorEntries := make([]*InstallomatorEntry, 0, len(installomatorEntries)/4)
	for _, entry := range installomatorEntries {
		if pp.shouldIncludeInstallomatorEntry(entry) {
			includedInstallomatorEntries = append(includedInstallomatorEntries, entry)
		}
	}

	fmt.Printf("Processing %d Homebrew casks and %d Installomator entries that meet criteria...\n",
		len(includedCasks), len(includedInstallomatorEntries))

	// Combine and deduplicate entries
	combinedEntries := pp.combineAndDeduplicate(includedCasks, includedInstallomatorEntries)

	// Sort alphabetically by identifier
	// Note: We'll use a simple string comparison since we removed slices package
	for i := 0; i < len(combinedEntries)-1; i++ {
		for j := i + 1; j < len(combinedEntries); j++ {
			if strings.ToLower(combinedEntries[i].Identifier) > strings.ToLower(combinedEntries[j].Identifier) {
				combinedEntries[i], combinedEntries[j] = combinedEntries[j], combinedEntries[i]
			}
		}
	}

	fmt.Printf("Generated %d unique entries after deduplication\n", len(combinedEntries))

	// Generate YAML files
	for _, entry := range combinedEntries {
		if entry.Source == "homebrew" {
			// Find the original cask
			for _, cask := range includedCasks {
				if cask.Token == entry.Identifier {
					fleetConfig := pp.generateFleetYAML(cask)
					safeToken := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(cask.Token, "_")
					filename := fmt.Sprintf("%s.yml", safeToken)
					err := pp.saveYAMLFile(filename, fleetConfig, "Homebrew")
					if err != nil {
						fmt.Printf("Error processing %s: %v\n", cask.Token, err)
					}
					break
				}
			}
		} else if entry.Source == "installomator" {
			// Find the original Installomator entry
			for _, installomatorEntry := range includedInstallomatorEntries {
				if installomatorEntry.Label == entry.Identifier {
					fleetConfig := pp.generateFleetYAMLFromInstallomator(installomatorEntry)
					err := pp.saveYAMLFileFromInstallomator(installomatorEntry, fleetConfig)
					if err != nil {
						fmt.Printf("Error processing %s: %v\n", installomatorEntry.Label, err)
					}
					break
				}
			}
		}
	}

	fmt.Printf("\nGenerated %d Fleet YAML files in %s/macOS/ and %s/Windows/\n", len(combinedEntries), pp.outputDir, pp.outputDir)

	return nil
}

// shouldIncludeWinGetPackage determines if a WinGet package should be included based on criteria
func main() {
	// Create output directory
	outputDir := "fleet_yaml_files"
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Create processor and run
	processor := NewPackageProcessor(outputDir)

	// Process Homebrew casks (PKG files)
	err = processor.processCasks()
	if err != nil {
		fmt.Printf("Error processing Homebrew casks: %v\n", err)
		os.Exit(1)
	}

	// Process WinGet packages (MSI files)
	err = processor.processWinGetPackages()
	if err != nil {
		fmt.Printf("Error processing WinGet packages: %v\n", err)
		// Don't exit here as Homebrew processing succeeded
	}

	// Save cache at the end
	processor.saveCache()

	fmt.Println("Conversion completed successfully!")
}
