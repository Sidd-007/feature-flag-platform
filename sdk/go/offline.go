package featureflags

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// OfflineHandler manages offline flag configurations
type OfflineHandler struct {
	configPath   string
	environment  *Environment
	offline      bool
	logger       zerolog.Logger
	mutex        sync.RWMutex
	lastModified time.Time
}

// NewOfflineHandler creates a new offline handler
func NewOfflineHandler(configPath string, logger zerolog.Logger) (*OfflineHandler, error) {
	handler := &OfflineHandler{
		configPath: configPath,
		logger:     logger.With().Str("component", "offline").Logger(),
	}

	// Try to load initial configuration
	if err := handler.loadConfiguration(); err != nil {
		handler.logger.Warn().
			Err(err).
			Str("config_path", configPath).
			Msg("Failed to load offline configuration, starting with empty config")

		// Create empty environment
		handler.environment = &Environment{
			ID:        "offline",
			Name:      "Offline Environment",
			Flags:     make(map[string]*Flag),
			Segments:  make(map[string]*Segment),
			Version:   1,
			UpdatedAt: time.Now(),
		}
	}

	handler.logger.Info().
		Str("config_path", configPath).
		Int("flags_count", len(handler.environment.Flags)).
		Int("segments_count", len(handler.environment.Segments)).
		Msg("Offline handler initialized")

	return handler, nil
}

// IsOffline returns true if the handler is in offline mode
func (oh *OfflineHandler) IsOffline() bool {
	oh.mutex.RLock()
	defer oh.mutex.RUnlock()

	return oh.offline
}

// SetOffline sets the offline mode
func (oh *OfflineHandler) SetOffline(offline bool) {
	oh.mutex.Lock()
	defer oh.mutex.Unlock()

	oh.offline = offline

	oh.logger.Info().
		Bool("offline", offline).
		Msg("Offline mode changed")
}

// GetFlag retrieves a flag from offline configuration
func (oh *OfflineHandler) GetFlag(flagKey string) (*Flag, bool) {
	oh.mutex.RLock()
	defer oh.mutex.RUnlock()

	if oh.environment == nil {
		return nil, false
	}

	flag, exists := oh.environment.Flags[flagKey]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid concurrent modification
	flagCopy := *flag
	return &flagCopy, true
}

// GetAllFlags retrieves all flags from offline configuration
func (oh *OfflineHandler) GetAllFlags() map[string]*Flag {
	oh.mutex.RLock()
	defer oh.mutex.RUnlock()

	if oh.environment == nil {
		return make(map[string]*Flag)
	}

	// Return copies to avoid concurrent modification
	flags := make(map[string]*Flag)
	for key, flag := range oh.environment.Flags {
		flagCopy := *flag
		flags[key] = &flagCopy
	}

	return flags
}

// GetSegment retrieves a segment from offline configuration
func (oh *OfflineHandler) GetSegment(segmentID string) (*Segment, bool) {
	oh.mutex.RLock()
	defer oh.mutex.RUnlock()

	if oh.environment == nil {
		return nil, false
	}

	segment, exists := oh.environment.Segments[segmentID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid concurrent modification
	segmentCopy := *segment
	return &segmentCopy, true
}

// UpdateConfiguration updates the offline configuration with new data
func (oh *OfflineHandler) UpdateConfiguration(environment *Environment) error {
	oh.mutex.Lock()
	defer oh.mutex.Unlock()

	// Update environment
	oh.environment = environment

	// Save to file
	if err := oh.saveConfiguration(); err != nil {
		oh.logger.Error().
			Err(err).
			Msg("Failed to save offline configuration")
		return err
	}

	oh.logger.Info().
		Int("flags_count", len(environment.Flags)).
		Int("segments_count", len(environment.Segments)).
		Int64("version", environment.Version).
		Msg("Offline configuration updated")

	return nil
}

// UpdateFlag updates a specific flag in offline configuration
func (oh *OfflineHandler) UpdateFlag(flag *Flag) error {
	oh.mutex.Lock()
	defer oh.mutex.Unlock()

	if oh.environment == nil {
		oh.environment = &Environment{
			ID:        "offline",
			Name:      "Offline Environment",
			Flags:     make(map[string]*Flag),
			Segments:  make(map[string]*Segment),
			Version:   1,
			UpdatedAt: time.Now(),
		}
	}

	// Update flag
	flagCopy := *flag
	oh.environment.Flags[flag.Key] = &flagCopy
	oh.environment.Version++
	oh.environment.UpdatedAt = time.Now()

	// Save to file
	if err := oh.saveConfiguration(); err != nil {
		oh.logger.Error().
			Err(err).
			Str("flag_key", flag.Key).
			Msg("Failed to save offline configuration after flag update")
		return err
	}

	oh.logger.Debug().
		Str("flag_key", flag.Key).
		Bool("enabled", flag.Enabled).
		Msg("Flag updated in offline configuration")

	return nil
}

// RemoveFlag removes a flag from offline configuration
func (oh *OfflineHandler) RemoveFlag(flagKey string) error {
	oh.mutex.Lock()
	defer oh.mutex.Unlock()

	if oh.environment == nil {
		return nil
	}

	// Remove flag
	delete(oh.environment.Flags, flagKey)
	oh.environment.Version++
	oh.environment.UpdatedAt = time.Now()

	// Save to file
	if err := oh.saveConfiguration(); err != nil {
		oh.logger.Error().
			Err(err).
			Str("flag_key", flagKey).
			Msg("Failed to save offline configuration after flag removal")
		return err
	}

	oh.logger.Debug().
		Str("flag_key", flagKey).
		Msg("Flag removed from offline configuration")

	return nil
}

// RefreshFromFile reloads configuration from file if it has been modified
func (oh *OfflineHandler) RefreshFromFile() error {
	// Check if file has been modified
	fileInfo, err := os.Stat(oh.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, nothing to refresh
			return nil
		}
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	oh.mutex.RLock()
	lastModified := oh.lastModified
	oh.mutex.RUnlock()

	if !fileInfo.ModTime().After(lastModified) {
		// File hasn't been modified
		return nil
	}

	// File has been modified, reload
	if err := oh.loadConfiguration(); err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	oh.logger.Info().
		Time("file_modified", fileInfo.ModTime()).
		Msg("Offline configuration reloaded from file")

	return nil
}

// loadConfiguration loads configuration from file
func (oh *OfflineHandler) loadConfiguration() error {
	// Check if file exists
	if _, err := os.Stat(oh.configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist: %s", oh.configPath)
	}

	// Read file
	data, err := ioutil.ReadFile(oh.configPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse JSON
	var environment Environment
	if err := json.Unmarshal(data, &environment); err != nil {
		return fmt.Errorf("failed to parse configuration file: %w", err)
	}

	// Get file modification time
	fileInfo, err := os.Stat(oh.configPath)
	if err != nil {
		return fmt.Errorf("failed to stat configuration file: %w", err)
	}

	oh.environment = &environment
	oh.lastModified = fileInfo.ModTime()

	oh.logger.Debug().
		Int("flags_count", len(environment.Flags)).
		Int("segments_count", len(environment.Segments)).
		Int64("version", environment.Version).
		Time("last_modified", oh.lastModified).
		Msg("Configuration loaded from file")

	return nil
}

// saveConfiguration saves configuration to file
func (oh *OfflineHandler) saveConfiguration() error {
	if oh.environment == nil {
		return fmt.Errorf("no environment to save")
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(oh.environment, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write to temporary file first
	tempPath := oh.configPath + ".tmp"
	if err := ioutil.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, oh.configPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	// Update last modified time
	if fileInfo, err := os.Stat(oh.configPath); err == nil {
		oh.lastModified = fileInfo.ModTime()
	}

	oh.logger.Debug().
		Str("config_path", oh.configPath).
		Int("flags_count", len(oh.environment.Flags)).
		Msg("Configuration saved to file")

	return nil
}

// GetEnvironment returns a copy of the current environment
func (oh *OfflineHandler) GetEnvironment() *Environment {
	oh.mutex.RLock()
	defer oh.mutex.RUnlock()

	if oh.environment == nil {
		return nil
	}

	// Return a deep copy
	envCopy := *oh.environment

	// Copy flags
	envCopy.Flags = make(map[string]*Flag)
	for key, flag := range oh.environment.Flags {
		flagCopy := *flag
		envCopy.Flags[key] = &flagCopy
	}

	// Copy segments
	envCopy.Segments = make(map[string]*Segment)
	for key, segment := range oh.environment.Segments {
		segmentCopy := *segment
		envCopy.Segments[key] = &segmentCopy
	}

	return &envCopy
}

// GetVersion returns the current configuration version
func (oh *OfflineHandler) GetVersion() int64 {
	oh.mutex.RLock()
	defer oh.mutex.RUnlock()

	if oh.environment == nil {
		return 0
	}

	return oh.environment.Version
}

// IsConfigurationAvailable returns true if offline configuration is available
func (oh *OfflineHandler) IsConfigurationAvailable() bool {
	oh.mutex.RLock()
	defer oh.mutex.RUnlock()

	return oh.environment != nil && len(oh.environment.Flags) > 0
}

// GetConfigurationPath returns the configuration file path
func (oh *OfflineHandler) GetConfigurationPath() string {
	return oh.configPath
}

// SetConfigurationPath updates the configuration file path
func (oh *OfflineHandler) SetConfigurationPath(path string) error {
	oh.mutex.Lock()
	defer oh.mutex.Unlock()

	oh.configPath = path

	// Try to load from new path
	if err := oh.loadConfiguration(); err != nil {
		oh.logger.Warn().
			Err(err).
			Str("new_path", path).
			Msg("Failed to load configuration from new path")
	}

	oh.logger.Info().
		Str("new_path", path).
		Msg("Configuration path updated")

	return nil
}

// ExportConfiguration exports the current configuration to a file
func (oh *OfflineHandler) ExportConfiguration(outputPath string) error {
	oh.mutex.RLock()
	defer oh.mutex.RUnlock()

	if oh.environment == nil {
		return fmt.Errorf("no configuration to export")
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(oh.environment, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	oh.logger.Info().
		Str("output_path", outputPath).
		Int("flags_count", len(oh.environment.Flags)).
		Int("segments_count", len(oh.environment.Segments)).
		Msg("Configuration exported")

	return nil
}

// ImportConfiguration imports configuration from a file
func (oh *OfflineHandler) ImportConfiguration(inputPath string) error {
	// Read file
	data, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse JSON
	var environment Environment
	if err := json.Unmarshal(data, &environment); err != nil {
		return fmt.Errorf("failed to parse configuration file: %w", err)
	}

	oh.mutex.Lock()
	defer oh.mutex.Unlock()

	// Update environment
	oh.environment = &environment

	// Save to current path
	if err := oh.saveConfiguration(); err != nil {
		return fmt.Errorf("failed to save imported configuration: %w", err)
	}

	oh.logger.Info().
		Str("input_path", inputPath).
		Int("flags_count", len(environment.Flags)).
		Int("segments_count", len(environment.Segments)).
		Int64("version", environment.Version).
		Msg("Configuration imported")

	return nil
}

// ValidateConfiguration validates the current configuration
func (oh *OfflineHandler) ValidateConfiguration() []string {
	oh.mutex.RLock()
	defer oh.mutex.RUnlock()

	var errors []string

	if oh.environment == nil {
		errors = append(errors, "no environment configuration")
		return errors
	}

	// Validate flags
	for key, flag := range oh.environment.Flags {
		if !flag.IsValid() {
			errors = append(errors, fmt.Sprintf("invalid flag: %s", key))
		}

		// Check variation references
		for _, rule := range flag.Rules {
			if rule.Serve.VariationID != "" {
				found := false
				for _, variation := range flag.Variations {
					if variation.ID == rule.Serve.VariationID {
						found = true
						break
					}
				}
				if !found {
					errors = append(errors, fmt.Sprintf("flag %s: rule references invalid variation %s", key, rule.Serve.VariationID))
				}
			}
		}

		// Check prerequisite references
		for _, prereq := range flag.Prerequisites {
			if _, exists := oh.environment.Flags[prereq.FlagKey]; !exists {
				errors = append(errors, fmt.Sprintf("flag %s: prerequisite references invalid flag %s", key, prereq.FlagKey))
			}
		}
	}

	return errors
}

// GetStats returns offline handler statistics
func (oh *OfflineHandler) GetStats() map[string]interface{} {
	oh.mutex.RLock()
	defer oh.mutex.RUnlock()

	stats := map[string]interface{}{
		"offline":          oh.offline,
		"config_path":      oh.configPath,
		"config_available": oh.environment != nil,
		"last_modified":    oh.lastModified,
	}

	if oh.environment != nil {
		stats["flags_count"] = len(oh.environment.Flags)
		stats["segments_count"] = len(oh.environment.Segments)
		stats["version"] = oh.environment.Version
		stats["updated_at"] = oh.environment.UpdatedAt

		// Count enabled/disabled flags
		enabledFlags := 0
		for _, flag := range oh.environment.Flags {
			if flag.Enabled {
				enabledFlags++
			}
		}
		stats["enabled_flags"] = enabledFlags
		stats["disabled_flags"] = len(oh.environment.Flags) - enabledFlags
	}

	return stats
}

// Close closes the offline handler
func (oh *OfflineHandler) Close() {
	oh.logger.Info().Msg("Offline handler closed")
}
