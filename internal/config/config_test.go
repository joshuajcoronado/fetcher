package config

import (
	"os"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	// Set up environment variables
	envVars := map[string]string{
		"ETHERSCAN_API_KEY":     "test_etherscan_key",
		"ALPHAVANTAGE_API_KEY":  "test_alphavantage_key",
		"RENTCAST_API_KEY":      "test_rentcast_key",
		"GUIDELINE_EMAIL":       "test@example.com",
		"GUIDELINE_PASSWORD":    "test_password",
		"ETHERSCAN_BASE_URL":    "https://test.etherscan.io",
		"ALPHAVANTAGE_BASE_URL": "https://test.alphavantage.co",
		"RENTCAST_BASE_URL":     "https://test.rentcast.io",
		"GUIDELINE_BASE_URL":    "https://test.guideline.com",
	}

	// Set environment variables
	for key, value := range envVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	// Load configuration
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	// Verify all fields are set correctly
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"EtherscanAPIKey", cfg.EtherscanAPIKey, "test_etherscan_key"},
		{"AlphavantageAPIKey", cfg.AlphavantageAPIKey, "test_alphavantage_key"},
		{"RentcastAPIKey", cfg.RentcastAPIKey, "test_rentcast_key"},
		{"GuidelineEmail", cfg.GuidelineEmail, "test@example.com"},
		{"GuidelinePassword", cfg.GuidelinePassword, "test_password"},
		{"EtherscanBaseURL", cfg.EtherscanBaseURL, "https://test.etherscan.io"},
		{"AlphavantageBaseURL", cfg.AlphavantageBaseURL, "https://test.alphavantage.co"},
		{"RentcastBaseURL", cfg.RentcastBaseURL, "https://test.rentcast.io"},
		{"GuidelineBaseURL", cfg.GuidelineBaseURL, "https://test.guideline.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	// Set only required environment variables
	requiredVars := map[string]string{
		"ETHERSCAN_API_KEY":    "test_etherscan_key",
		"ALPHAVANTAGE_API_KEY": "test_alphavantage_key",
		"RENTCAST_API_KEY":     "test_rentcast_key",
		"GUIDELINE_EMAIL":      "test@example.com",
		"GUIDELINE_PASSWORD":   "test_password",
	}

	// Set environment variables
	for key, value := range requiredVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	// Ensure base URL env vars are unset
	baseURLVars := []string{
		"ETHERSCAN_BASE_URL",
		"ALPHAVANTAGE_BASE_URL",
		"RENTCAST_BASE_URL",
		"GUIDELINE_BASE_URL",
	}
	for _, key := range baseURLVars {
		os.Unsetenv(key)
	}

	// Load configuration
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	// Verify default base URLs are used
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"EtherscanBaseURL", cfg.EtherscanBaseURL, "https://api.etherscan.io/v2/api"},
		{"AlphavantageBaseURL", cfg.AlphavantageBaseURL, "https://www.alphavantage.co/query"},
		{"RentcastBaseURL", cfg.RentcastBaseURL, "https://api.rentcast.io/v1"},
		{"GuidelineBaseURL", cfg.GuidelineBaseURL, "https://my.guideline.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	// Clear all environment variables
	envVars := []string{
		"ETHERSCAN_API_KEY",
		"ALPHAVANTAGE_API_KEY",
		"RENTCAST_API_KEY",
		"GUIDELINE_EMAIL",
		"GUIDELINE_PASSWORD",
	}

	for _, key := range envVars {
		os.Unsetenv(key)
	}

	// Test cases for missing required fields
	tests := []struct {
		name        string
		setupEnv    map[string]string
		wantErrText string
	}{
		{
			name: "missing all required",
			setupEnv: map[string]string{},
			wantErrText: "missing required configuration",
		},
		{
			name: "missing ETHERSCAN_API_KEY",
			setupEnv: map[string]string{
				"ALPHAVANTAGE_API_KEY": "test",
				"RENTCAST_API_KEY":     "test",
				"GUIDELINE_EMAIL":      "test@example.com",
				"GUIDELINE_PASSWORD":   "test",
			},
			wantErrText: "ETHERSCAN_API_KEY",
		},
		{
			name: "missing ALPHAVANTAGE_API_KEY",
			setupEnv: map[string]string{
				"ETHERSCAN_API_KEY":  "test",
				"RENTCAST_API_KEY":   "test",
				"GUIDELINE_EMAIL":    "test@example.com",
				"GUIDELINE_PASSWORD": "test",
			},
			wantErrText: "ALPHAVANTAGE_API_KEY",
		},
		{
			name: "missing RENTCAST_API_KEY",
			setupEnv: map[string]string{
				"ETHERSCAN_API_KEY":    "test",
				"ALPHAVANTAGE_API_KEY": "test",
				"GUIDELINE_EMAIL":      "test@example.com",
				"GUIDELINE_PASSWORD":   "test",
			},
			wantErrText: "RENTCAST_API_KEY",
		},
		{
			name: "missing GUIDELINE_EMAIL",
			setupEnv: map[string]string{
				"ETHERSCAN_API_KEY":    "test",
				"ALPHAVANTAGE_API_KEY": "test",
				"RENTCAST_API_KEY":     "test",
				"GUIDELINE_PASSWORD":   "test",
			},
			wantErrText: "GUIDELINE_EMAIL",
		},
		{
			name: "missing GUIDELINE_PASSWORD",
			setupEnv: map[string]string{
				"ETHERSCAN_API_KEY":    "test",
				"ALPHAVANTAGE_API_KEY": "test",
				"RENTCAST_API_KEY":     "test",
				"GUIDELINE_EMAIL":      "test@example.com",
			},
			wantErrText: "GUIDELINE_PASSWORD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars
			for _, key := range envVars {
				os.Unsetenv(key)
			}

			// Set up test-specific environment
			for key, value := range tt.setupEnv {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Attempt to load configuration
			_, err := Load()
			if err == nil {
				t.Fatal("Load() expected error, got nil")
			}

			// Verify error message contains expected text
			if err.Error() == "" || (tt.wantErrText != "" && !contains(err.Error(), tt.wantErrText)) {
				t.Errorf("Load() error = %q, want error containing %q", err.Error(), tt.wantErrText)
			}
		})
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}