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

// generateSummary creates a summary document of processed casks
func (cp *CaskProcessor) generateSummary(includedCasks []*Cask) error {
	summaryFile := filepath.Join(cp.outputDir, "SUMMARY.md")

	summaryContent := fmt.Sprintf(`# Fleet YAML Files Generated from Homebrew Casks

Generated on: %s

## Summary

Total casks processed: %d

## Generated Files

`, time.Now().UTC().Format("2006-01-02 15:04:05 UTC"), len(includedCasks))

	for _, cask := range includedCasks {
		name := cask.Token
		if len(cask.Name) > 0 {
			name = cask.Name[0]
		}

		summaryContent += fmt.Sprintf(`### %s (%s)

- **Version**: %s
- **Description**: %s
- **File**: `+"`%s.yml`"+`
- **URL**: %s

`, name, cask.Token, cask.Version, cask.Desc, cask.Token, cask.URL)
	}

	err := os.WriteFile(summaryFile, []byte(summaryContent), 0644)
	if err != nil {
		return fmt.Errorf("error writing summary: %w", err)
	}

	fmt.Printf("Summary generated: %s\n", filepath.Base(summaryFile))
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

	fmt.Printf("Found %d total casks\n", len(casks))

	// Filter and process casks using slices package for better performance
	includedCasks := make([]*Cask, 0, len(casks)/4) // Pre-allocate with reasonable capacity
	for _, cask := range casks {
		if cp.shouldIncludeCask(cask) {
			includedCasks = append(includedCasks, cask)
		}
	}

	fmt.Printf("Processing %d casks that meet criteria...\n", len(includedCasks))

	// Generate YAML files
	for _, cask := range includedCasks {
		fleetConfig := cp.generateFleetYAML(cask)
		err := cp.saveYAMLFile(cask, fleetConfig)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", cask.Token, err)
		}
	}

	fmt.Printf("\nGenerated %d Fleet YAML files in %s/\n", len(includedCasks), cp.outputDir)

	// Generate summary
	err = cp.generateSummary(includedCasks)
	if err != nil {
		fmt.Printf("Warning: Could not generate summary: %v\n", err)
	}

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
