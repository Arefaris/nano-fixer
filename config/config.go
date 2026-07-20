package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	APIKey      string
	APIBaseURL  string
	ModelName   string
	HotkeyMod   string
	HotkeyKey   string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Println("Error loading .env file:", err)
	}

	apiKey := os.Getenv("API_KEY")
	apiBaseURL := os.Getenv("API_BASE_URL")
	modelName := os.Getenv("MODEL_NAME")
	if modelName == "" {
		modelName = "gpt-4o-mini" // Default to a fast/cheap model if not specified, since gpt-5.4-nano is hypothetical
	}

	// We default to Shift+Alt+C
	hotkeyMod := os.Getenv("HOTKEY_MOD")
	if hotkeyMod == "" {
		hotkeyMod = "shift+alt"
	}
	hotkeyKey := os.Getenv("HOTKEY_KEY")
	if hotkeyKey == "" {
		hotkeyKey = "C"
	}

	return &Config{
		APIKey:      apiKey,
		APIBaseURL:  apiBaseURL,
		ModelName:   modelName,
		HotkeyMod:   hotkeyMod,
		HotkeyKey:   hotkeyKey,
	}
}
