package config

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Version: 1,
		Settings: Settings{
			TLD:       "test",
			HTTPPort:  80,
			HTTPSPort: 443,
			AutoStart: true,
			LogLevel:  "info",
		},
		Projects: make(map[string]Project),
	}
}
