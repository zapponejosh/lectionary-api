package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any existing env vars that might interfere
	clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with defaults failed: %v", err)
	}

	// Check defaults are applied
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.Env != EnvDevelopment {
		t.Errorf("Env = %q, want %q", cfg.Env, EnvDevelopment)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "text")
	}
}

func TestLoad_FromEnv(t *testing.T) {
	clearEnv()

	// Set custom values
	os.Setenv("PORT", "3000")
	os.Setenv("ENV", "production")
	os.Setenv("DATABASE_PATH", "/data/test.db")
	os.Setenv("API_KEY", "secret-key-123")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "json")
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want 3000", cfg.Port)
	}
	if cfg.Env != EnvProduction {
		t.Errorf("Env = %q, want %q", cfg.Env, EnvProduction)
	}
	if cfg.DatabasePath != "/data/test.db" {
		t.Errorf("DatabasePath = %q, want %q", cfg.DatabasePath, "/data/test.db")
	}
	if cfg.APIKey != "secret-key-123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "secret-key-123")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "json")
	}
}

func TestConfig_Validate(t *testing.T) {
	// Table-driven tests for validation
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid development config",
			config: Config{
				Port:         8080,
				Env:          EnvDevelopment,
				DatabasePath: "./data/test.db",
				APIKey:       "", // OK in development
				LogLevel:     "info",
				LogFormat:    "text",
			},
			wantErr: false,
		},
		{
			name: "valid production config",
			config: Config{
				Port:         8080,
				Env:          EnvProduction,
				DatabasePath: "/data/lectionary.db",
				APIKey:       "required-in-prod",
				LogLevel:     "info",
				LogFormat:    "json",
			},
			wantErr: false,
		},
		{
			name: "production requires API key",
			config: Config{
				Port:         8080,
				Env:          EnvProduction,
				DatabasePath: "/data/lectionary.db",
				APIKey:       "", // Missing!
				LogLevel:     "info",
				LogFormat:    "json",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too low",
			config: Config{
				Port:         0,
				Env:          EnvDevelopment,
				DatabasePath: "./data/test.db",
				LogLevel:     "info",
				LogFormat:    "text",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			config: Config{
				Port:         70000,
				Env:          EnvDevelopment,
				DatabasePath: "./data/test.db",
				LogLevel:     "info",
				LogFormat:    "text",
			},
			wantErr: true,
		},
		{
			name: "invalid environment",
			config: Config{
				Port:         8080,
				Env:          "invalid",
				DatabasePath: "./data/test.db",
				LogLevel:     "info",
				LogFormat:    "text",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: Config{
				Port:         8080,
				Env:          EnvDevelopment,
				DatabasePath: "./data/test.db",
				LogLevel:     "verbose", // Not valid
				LogFormat:    "text",
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			config: Config{
				Port:         8080,
				Env:          EnvDevelopment,
				DatabasePath: "./data/test.db",
				LogLevel:     "info",
				LogFormat:    "xml", // Not valid
			},
			wantErr: true,
		},
		{
			name: "empty database path",
			config: Config{
				Port:         8080,
				Env:          EnvDevelopment,
				DatabasePath: "",
				LogLevel:     "info",
				LogFormat:    "text",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	cfg := &Config{Env: EnvDevelopment}
	if !cfg.IsDevelopment() {
		t.Error("IsDevelopment() = false, want true")
	}

	cfg.Env = EnvProduction
	if cfg.IsDevelopment() {
		t.Error("IsDevelopment() = true, want false")
	}
}

func TestConfig_IsProduction(t *testing.T) {
	cfg := &Config{Env: EnvProduction}
	if !cfg.IsProduction() {
		t.Error("IsProduction() = false, want true")
	}

	cfg.Env = EnvDevelopment
	if cfg.IsProduction() {
		t.Error("IsProduction() = true, want false")
	}
}

// clearEnv removes all config-related environment variables
func clearEnv() {
	vars := []string{
		"PORT", "ENV", "DATABASE_PATH", "API_KEY",
		"LOG_LEVEL", "LOG_FORMAT",
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}
}
