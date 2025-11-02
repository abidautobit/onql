package config

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/joho/godotenv"
)

var (
	once sync.Once
)

// Load ensures .env file is loaded exactly once (thread-safe)
// This is called automatically on first access to Env()
func Load() {
	once.Do(func() {
		// Walk up the directory tree to find config/.env
		// This works regardless of nesting depth
		envPath := findEnvFile()

		if envPath != "" {
			if err := godotenv.Load(envPath); err != nil {
				log.Printf("Warning: Found .env at %s but failed to load: %v", envPath, err)
			}
		} else {
			log.Printf("Warning: Could not find .env file, using environment variables")
		}
	})
}

// findEnvFile walks up the directory tree looking for config/.env
// Returns the full path to the .env file, or empty string if not found
func findEnvFile() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Walk up the directory tree
	for {
		envPath := filepath.Join(cwd, "config", ".env")

		// Check if config/.env exists at this level
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}

		// Move up one directory
		parent := filepath.Dir(cwd)

		// Stop if we've reached the root or can't go further up
		if parent == cwd || parent == "." || parent == "/" {
			break
		}

		cwd = parent
	}

	return ""
}

// Env returns the value of an environment variable
// It ensures config is loaded before accessing environment variables
func Env(key string) string {
	Load() // Lazy load on first access
	return os.Getenv(key)
}
