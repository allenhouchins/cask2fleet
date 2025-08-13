package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
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

// InstallomatorEntry represents an Installomator package entry
type InstallomatorEntry struct {
	Label       string
	Name        string
	Type        string
	PackageID   string
	DownloadURL string
	Source      string // "installomator" to identify source
}

// FleetSoftware represents the Fleet software configuration structure
type FleetSoftware struct {
	URL              string   `yaml:"url"`
	AutomaticInstall bool     `yaml:"automatic_install"`
	SelfService      bool     `yaml:"self_service"`
	Categories       []string `yaml:"categories"`
}

// CaskProcessor handles the conversion of casks to Fleet YAML
type CaskProcessor struct {
	outputDir string
}

// NewCaskProcessor creates a new processor instance
func NewCaskProcessor(outputDir string) *CaskProcessor {
	return &CaskProcessor{
		outputDir: outputDir,
	}
}

// isPKGFile checks if the URL points to a PKG file
func (cp *CaskProcessor) isPKGFile(url string) bool {
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

// shouldIncludeCask determines if a cask should be included based on criteria
func (cp *CaskProcessor) shouldIncludeCask(cask *Cask) bool {
	// Must not be deprecated
	if cask.Deprecated {
		return false
	}

	// Must have a URL
	if cask.URL == "" {
		return false
	}

	// Must be a PKG file
	if !cp.isPKGFile(cask.URL) {
		return false
	}

	return true
}

// extractPKGIdentifiers extracts package identifiers for uninstallation
func (cp *CaskProcessor) extractPKGIdentifiers(cask *Cask) []string {
	var identifiers []string

	// Look for pkgutil identifiers in artifacts
	if cask.Artifacts != nil {
		for _, artifact := range cask.Artifacts {
			if artifactMap, ok := artifact.(map[string]interface{}); ok {
				if pkgutil, exists := artifactMap["pkgutil"]; exists {
					switch v := pkgutil.(type) {
					case string:
						identifiers = append(identifiers, v)
					case []interface{}:
						for _, id := range v {
							if idStr, ok := id.(string); ok {
								identifiers = append(identifiers, idStr)
							}
						}
					}
				}
			}
		}
	}

	// If no identifiers found, try to generate from token
	if len(identifiers) == 0 && cask.Token != "" {
		// Common pattern: com.company.appname
		if strings.Contains(cask.Token, "-") {
			parts := strings.Split(cask.Token, "-")
			if len(parts) >= 2 {
				company := parts[0]
				app := strings.Join(parts[1:], "-")
				identifiers = append(identifiers, fmt.Sprintf("com.%s.%s", company, app))
			}
		}
	}

	// Remove duplicates using slices package (Go 1.24+)
	return slices.Compact(identifiers)
}

// generateFleetYAML generates Fleet-compatible YAML structure for a cask
func (cp *CaskProcessor) generateFleetYAML(cask *Cask) *FleetSoftware {
	// Create Fleet software configuration
	fleetConfig := &FleetSoftware{
		URL:              cask.URL,
		AutomaticInstall: false,
		SelfService:      false,
		Categories:       []string{},
	}

	return fleetConfig
}

// generateUninstallScript generates an uninstall script for the cask
func (cp *CaskProcessor) generateUninstallScript(cask *Cask) string {
	identifiers := cp.extractPKGIdentifiers(cask)
	if len(identifiers) > 0 {
		// Create uninstall script using pkgutil
		script := "#!/bin/bash\n"
		for _, identifier := range identifiers {
			script += fmt.Sprintf("pkgutil --forget '%s'\n", identifier)
		}
		return script
	}

	// Fallback uninstall script
	return "#!/bin/bash\necho 'No specific uninstall script available for this package'"
}

// saveYAMLFile saves the Fleet configuration to a YAML file
func (cp *CaskProcessor) saveYAMLFile(cask *Cask, fleetConfig *FleetSoftware) error {
	// Create safe filename
	safeToken := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(cask.Token, "_")
	filename := fmt.Sprintf("%s.yml", safeToken)
	filepath := filepath.Join(cp.outputDir, filename)

	// Marshal to YAML
	yamlData, err := yaml.Marshal(fleetConfig)
	if err != nil {
		return fmt.Errorf("error marshaling YAML for %s: %w", filename, err)
	}

	// Add comment at the bottom
	comment := "\n# Categories are currently limited to Browsers, Communication, Developer tools, and Productivity.\n# This is a minimum version of this file. All configurable parameters can be seen at https://fleetdm.com/docs/rest-api/rest-api#parameters139\n"
	finalData := append(yamlData, []byte(comment)...)

	// Write to file
	err = os.WriteFile(filepath, finalData, 0644)
	if err != nil {
		return fmt.Errorf("error writing %s: %w", filename, err)
	}

	fmt.Printf("Generated: %s\n", filename)
	return nil
}



// fetchCasks fetches all Homebrew casks from the API
func (cp *CaskProcessor) fetchCasks() ([]*Cask, error) {
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
func (cp *CaskProcessor) fetchInstallomatorData() ([]*InstallomatorEntry, error) {
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

	return cp.parseInstallomatorScript(string(body))
}

// parseInstallomatorScript parses the Installomator shell script to extract package entries
func (cp *CaskProcessor) parseInstallomatorScript(scriptContent string) ([]*InstallomatorEntry, error) {
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
			if currentEntry != nil && currentEntry.DownloadURL != "" && cp.isPKGFile(currentEntry.DownloadURL) {
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
	if currentEntry != nil && currentEntry.DownloadURL != "" && cp.isPKGFile(currentEntry.DownloadURL) {
		entries = append(entries, currentEntry)
	}
	
	return entries, nil
}

// shouldIncludeInstallomatorEntry determines if an Installomator entry should be included
func (cp *CaskProcessor) shouldIncludeInstallomatorEntry(entry *InstallomatorEntry) bool {
	// Must have a download URL
	if entry.DownloadURL == "" {
		return false
	}
	
	// Must be a PKG file
	if !cp.isPKGFile(entry.DownloadURL) {
		return false
	}
	
	// Must have a label (identifier)
	if entry.Label == "" {
		return false
	}
	
	return true
}

// generateFleetYAMLFromInstallomator generates Fleet-compatible YAML structure for an Installomator entry
func (cp *CaskProcessor) generateFleetYAMLFromInstallomator(entry *InstallomatorEntry) *FleetSoftware {
	// Create Fleet software configuration
	fleetConfig := &FleetSoftware{
		URL:              entry.DownloadURL,
		AutomaticInstall: false,
		SelfService:      false,
		Categories:       []string{},
	}

	return fleetConfig
}

// generateUninstallScriptFromInstallomator generates an uninstall script for the Installomator entry
func (cp *CaskProcessor) generateUninstallScriptFromInstallomator(entry *InstallomatorEntry) string {
	if entry.PackageID != "" {
		// Create uninstall script using pkgutil
		script := "#!/bin/bash\n"
		script += fmt.Sprintf("pkgutil --forget '%s'\n", entry.PackageID)
		return script
	}

	// Fallback uninstall script
	return "#!/bin/bash\necho 'No specific uninstall script available for this package'"
}

// saveYAMLFileFromInstallomator saves the Fleet configuration to a YAML file for Installomator entries
func (cp *CaskProcessor) saveYAMLFileFromInstallomator(entry *InstallomatorEntry, fleetConfig *FleetSoftware) error {
	// Create safe filename
	safeLabel := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(entry.Label, "_")
	filename := fmt.Sprintf("%s.yml", safeLabel)
	filepath := filepath.Join(cp.outputDir, filename)

	// Marshal to YAML
	yamlData, err := yaml.Marshal(fleetConfig)
	if err != nil {
		return fmt.Errorf("error marshaling YAML for %s: %w", filename, err)
	}

	// Add comment at the bottom
	comment := "\n# Categories are currently limited to Browsers, Communication, Developer tools, and Productivity.\n# This is a minimum version of this file. All configurable parameters can be seen at https://fleetdm.com/docs/rest-api/rest-api#parameters139\n"
	finalData := append(yamlData, []byte(comment)...)

	// Write to file
	err = os.WriteFile(filepath, finalData, 0644)
	if err != nil {
		return fmt.Errorf("error writing %s: %w", filename, err)
	}

	fmt.Printf("Generated: %s (from Installomator)\n", filename)
	return nil
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

// processCasks is the main method to process all casks and generate YAML files
func (cp *CaskProcessor) processCasks() error {
	fmt.Println("Fetching Homebrew casks...")
	casks, err := cp.fetchCasks()
	if err != nil {
		return fmt.Errorf("failed to fetch casks: %w", err)
	}

	if len(casks) == 0 {
		return fmt.Errorf("no casks found")
	}

	fmt.Printf("Found %d total Homebrew casks\n", len(casks))

	fmt.Println("Fetching Installomator data...")
	installomatorEntries, err := cp.fetchInstallomatorData()
	if err != nil {
		return fmt.Errorf("failed to fetch Installomator data: %w", err)
	}

	fmt.Printf("Found %d total Installomator entries\n", len(installomatorEntries))

	// Filter Homebrew casks
	includedCasks := make([]*Cask, 0, len(casks)/4)
	for _, cask := range casks {
		if cp.shouldIncludeCask(cask) {
			includedCasks = append(includedCasks, cask)
		}
	}

	// Filter Installomator entries
	includedInstallomatorEntries := make([]*InstallomatorEntry, 0, len(installomatorEntries)/4)
	for _, entry := range installomatorEntries {
		if cp.shouldIncludeInstallomatorEntry(entry) {
			includedInstallomatorEntries = append(includedInstallomatorEntries, entry)
		}
	}

	fmt.Printf("Processing %d Homebrew casks and %d Installomator entries that meet criteria...\n", 
		len(includedCasks), len(includedInstallomatorEntries))

	// Combine and deduplicate entries
	combinedEntries := cp.combineAndDeduplicate(includedCasks, includedInstallomatorEntries)

	// Sort alphabetically by identifier
	slices.SortFunc(combinedEntries, func(a, b *CombinedEntry) int {
		return strings.Compare(strings.ToLower(a.Identifier), strings.ToLower(b.Identifier))
	})

	fmt.Printf("Generated %d unique entries after deduplication\n", len(combinedEntries))

	// Generate YAML files
	for _, entry := range combinedEntries {
		if entry.Source == "homebrew" {
			// Find the original cask
			for _, cask := range includedCasks {
				if cask.Token == entry.Identifier {
					fleetConfig := cp.generateFleetYAML(cask)
					err := cp.saveYAMLFile(cask, fleetConfig)
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
					fleetConfig := cp.generateFleetYAMLFromInstallomator(installomatorEntry)
					err := cp.saveYAMLFileFromInstallomator(installomatorEntry, fleetConfig)
					if err != nil {
						fmt.Printf("Error processing %s: %v\n", installomatorEntry.Label, err)
					}
					break
				}
			}
		}
	}

	fmt.Printf("\nGenerated %d Fleet YAML files in %s/\n", len(combinedEntries), cp.outputDir)

	// Generate summary
	err = cp.generateCombinedSummary(combinedEntries)
	if err != nil {
		fmt.Printf("Warning: Could not generate summary: %v\n", err)
	}

	return nil
}

// combineAndDeduplicate combines Homebrew casks and Installomator entries, removing duplicates
// Installomator entries take priority over Homebrew casks for duplicates
func (cp *CaskProcessor) combineAndDeduplicate(casks []*Cask, installomatorEntries []*InstallomatorEntry) []*CombinedEntry {
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

// generateCombinedSummary creates a summary document of processed entries from both sources
func (cp *CaskProcessor) generateCombinedSummary(entries []*CombinedEntry) error {
	summaryFile := filepath.Join(cp.outputDir, "SUMMARY.md")

	summaryContent := fmt.Sprintf(`# Fleet YAML Files Generated from Homebrew Casks and Installomator

Generated on: %s

## Summary

Total entries processed: %d

## Generated Files

`, time.Now().UTC().Format("2006-01-02 15:04:05 UTC"), len(entries))

	for _, entry := range entries {
		// Create safe filename for the summary
		safeIdentifier := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(entry.Identifier, "_")
		
		summaryContent += fmt.Sprintf(`### %s (%s)

- **Source**: %s
- **Name**: %s
- **Description**: %s
- **Version**: %s
- **File**: `+"`%s.yml`"+`
- **URL**: %s

`, entry.Name, entry.Identifier, entry.Source, entry.Name, entry.Description, entry.Version, safeIdentifier, entry.URL)
	}

	err := os.WriteFile(summaryFile, []byte(summaryContent), 0644)
	if err != nil {
		return fmt.Errorf("error writing summary: %w", err)
	}

	fmt.Printf("Summary generated: %s\n", filepath.Base(summaryFile))
	return nil
}

func main() {
	// Create output directory
	outputDir := "fleet_yaml_files"
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Create processor and run
	processor := NewCaskProcessor(outputDir)
	err = processor.processCasks()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Conversion completed successfully!")
}
