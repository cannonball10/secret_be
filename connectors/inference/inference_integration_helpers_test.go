//go:build integration

package inference

import (
	"encoding/json"
	"os"
	"path/filepath"

	schemas "github.com/cannonball10/foundation/schemas/inference"
	"github.com/joho/godotenv"
)

// loadEnvFromProjectRoot attempts to load .env from the project root.
// It searches upward from the current directory to find go.mod, then loads .env from that directory.
func loadEnvFromProjectRoot() error {
	// Start from the current working directory
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Walk up the directory tree to find go.mod (project root)
	dir := wd
	for {
		// Check if go.mod exists in current directory
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Found project root, try to load .env from here
			envPath := filepath.Join(dir, ".env")
			if err := godotenv.Load(envPath); err == nil {
				return nil
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root, stop searching
			break
		}
		dir = parent
	}

	// Fallback: try loading from current directory
	return godotenv.Load()
}

func weatherToolDefinition() schemas.ToolDefinition {
	return schemas.ToolDefinition{
		Name:        "get_weather",
		Description: "Get the current weather for a city",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"city": {
					"type": "string",
					"description": "The city name"
				}
			},
			"required": ["city"]
		}`),
	}
}
