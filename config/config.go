package config

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/billgraziano/dpapi"
	"golang.org/x/sys/windows/registry"
)

const ConfigFileName = "config.json"

type Config struct {
	APIKey         string `json:"APIKey"`
	APIBaseURL     string `json:"APIBaseURL"`
	ModelName      string `json:"ModelName"`
	HotkeyMod      string `json:"HotkeyMod"`
	HotkeyKey      string `json:"HotkeyKey"`
	TargetLanguage string `json:"TargetLanguage"`
	Autostart      bool   `json:"Autostart"`
	UseLocalAI     bool   `json:"UseLocalAI"`
}

// GetConfigPath returns the path to the config file in the executable's directory
func GetConfigPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exePath), ConfigFileName), nil
}

func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ModelName:      "gpt-4o-mini",
		HotkeyMod:      "shift+alt",
		HotkeyKey:      "C",
		TargetLanguage: "Auto",
		Autostart:      false,
	}

	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Save default config if not exists
			err = Save(cfg)
			if err != nil {
				return nil, err
			}
			return cfg, nil
		}
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(cfg)
	if err != nil {
		return nil, err
	}

	// DPAPI Decryption
	if cfg.APIKey != "" {
		decrypted, err := dpapi.Decrypt(cfg.APIKey)
		if err == nil {
			cfg.APIKey = decrypted
		} else {
			// Failed to decrypt, assume it's an old plaintext key
			log.Println("Found plaintext API Key (or decryption failed). Encrypting it on disk...")
			err = Save(cfg)
			if err != nil {
				log.Println("Warning: Failed to encrypt existing plaintext API key:", err)
			}
		}
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Update Windows Autostart Registry based on the setting
	err = SetWindowsAutostart(cfg.Autostart)
	if err != nil {
		log.Println("Error updating autostart registry:", err)
	}

	// Create a copy to avoid mutating the running config
	cfgToSave := *cfg
	if cfgToSave.APIKey != "" {
		encrypted, err := dpapi.Encrypt(cfgToSave.APIKey)
		if err == nil {
			cfgToSave.APIKey = encrypted
		} else {
			log.Println("Warning: Failed to encrypt API Key with DPAPI:", err)
		}
	}

	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(&cfgToSave)
}

func SetWindowsAutostart(enabled bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if enabled {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		// Wrap executable path in quotes to handle paths with spaces correctly
		return k.SetStringValue("NanoFixer", `"`+exePath+`"`)
	} else {
		err := k.DeleteValue("NanoFixer")
		if err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
	}
	return nil
}
