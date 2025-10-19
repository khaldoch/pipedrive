package handler

import (
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Port string
	Host string

	// Pipedrive API configuration (for real integration)
	PipedriveAPIKey    string
	PipedriveBaseURL   string
	PipedriveCompanyID string

	// Retell AI configuration
	RetellAPIKey       string
	RetellAssistantID  string
	RetellBaseURL      string
	RetellFromNumber   string

	// Webhook security (optional)
	RetellWebhookSecret string
	CalWebhookSecret    string

	// Logging configuration
	LogLevel string
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	config := &Config{
		// Server defaults
		Port: getEnv("PORT", "8080"),
		Host: getEnv("HOST", "0.0.0.0"),

		// Pipedrive configuration
		PipedriveAPIKey:    getEnv("PIPEDRIVE_API_KEY", ""),
		PipedriveBaseURL:   getEnv("PIPEDRIVE_BASE_URL", "https://api.pipedrive.com/v1"),
		PipedriveCompanyID: getEnv("PIPEDRIVE_COMPANY_ID", ""),

		// Retell AI configuration
		RetellAPIKey:       getEnv("RETELL_API_KEY", ""),
		RetellAssistantID:  getEnv("RETELL_ASSISTANT_ID", ""),
		RetellBaseURL:      getEnv("RETELL_BASE_URL", "https://api.retellai.com"),
		RetellFromNumber:   getEnv("RETELL_FROM_NUMBER", "18005300627"),

		// Webhook secrets (optional for basic auth)
		RetellWebhookSecret: getEnv("RETELL_WEBHOOK_SECRET", ""),
		CalWebhookSecret:    getEnv("CAL_WEBHOOK_SECRET", ""),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	return config
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer with a fallback default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool gets an environment variable as boolean with a fallback default value
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.LogLevel == "production" || os.Getenv("GIN_MODE") == "release"
}

// HasPipedriveConfig returns true if Pipedrive API key is configured
func (c *Config) HasPipedriveConfig() bool {
	return c.PipedriveAPIKey != ""
}

// HasRetellConfig returns true if Retell AI API key and assistant ID are configured
func (c *Config) HasRetellConfig() bool {
	return c.RetellAPIKey != "" && c.RetellAssistantID != ""
}
